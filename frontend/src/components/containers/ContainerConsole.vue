<template>
  <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon :name="mode === 'exec' ? 'terminal' : 'attach'" /></span>{{ mode === 'exec' ? '执行' : '附加' }}</h2><span class="console-state">{{ connecting ? '正在连接…' : connected ? '已连接' : '未连接' }}</span></div>
    <el-alert v-if="details.State?.Status !== 'running'" title="容器当前未运行，无法连接控制台。" type="warning" :closable="false" show-icon />
    <template v-if="mode === 'exec'"><form class="resource-form" @submit.prevent="connect"><label for="console-command">命令</label><el-input v-if="custom" id="console-command" v-model="customCommand" placeholder="例如 /bin/sh -c 'echo hello'" :disabled="connected || connecting" aria-label="自定义执行命令" /><el-select v-else id="console-command" v-model="command" class="form-control" :disabled="connected || connecting" aria-label="执行命令"><el-option v-for="shell in ['/bin/ash', '/bin/bash', '/bin/dash', '/bin/sh']" :key="shell" :label="shell" :value="shell" /></el-select><label for="console-custom">使用自定义命令</label><el-switch id="console-custom" v-model="custom" :disabled="connected || connecting" aria-label="使用自定义命令" /><label for="console-user">用户</label><el-input id="console-user" v-model="user" placeholder="root" :disabled="connected || connecting" aria-label="执行用户" /><span></span><div><el-button v-if="!connected" native-type="submit" :loading="connecting" :disabled="details.State?.Status !== 'running'">连接</el-button><el-button v-else @click="disconnect">断开连接</el-button></div></form></template>
    <template v-else><p v-if="!details.Config?.OpenStdin" class="console-warning">此容器未启用交互输入，控制台可能无法接收输入。</p><p v-if="!details.Config?.Tty" class="console-warning">此容器未启用终端，控制台可能无法正常显示。</p><el-button v-if="connected" @click="disconnect">分离</el-button><el-button v-else :loading="connecting" :disabled="details.State?.Status !== 'running'" @click="connect">附加到容器</el-button></template>
  </section>
  <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
  <div v-show="terminalVisible" ref="terminalElement" class="container-terminal" role="region" aria-label="容器交互终端"></div>
</template>
<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { dockerApi } from '@/api/docker'
import { parseContainerCommand, type ContainerDetails } from '@/utils/containers'
import { containerError } from '@/utils/containerFeedback'
Terminal.strings.promptLabel='终端输入'
Terminal.strings.tooMuchOutput='输出内容过多，已暂停朗读。请逐行查看终端内容。'
const props = defineProps<{ id: string; details: ContainerDetails; mode: 'exec' | 'attach' }>()
const command = ref('/bin/bash'), customCommand = ref(''), custom = ref(false), user = ref('root')
const connecting = ref(false), connected = ref(false), terminalVisible = ref(false), error = ref(''), terminalElement = ref<HTMLElement>()
let terminal: Terminal | undefined, fit: FitAddon | undefined, socket: WebSocket | undefined, observer: ResizeObserver | undefined, disposed = false, sequence = 0
function send(value: unknown) { if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify(value)) }
function resize() { if (!terminal || !fit) return; fit.fit(); send({ type: 'resize', cols: terminal.cols, rows: terminal.rows }) }
async function connect() {
  if (connecting.value || connected.value || disposed) return
  const request = ++sequence
  connecting.value = true; error.value = ''
  try {
    const args = props.mode === 'exec' ? parseContainerCommand(custom.value ? customCommand.value : command.value) : []
    const result = await dockerApi.containers.console(props.id, { mode: props.mode, command: args, user: user.value.trim() || 'root' })
    if (disposed || request !== sequence) return
    terminalVisible.value = true; await nextTick()
    terminal?.dispose(); observer?.disconnect()
    terminal = new Terminal({ cursorBlink: true, fontSize: 13, fontFamily: 'Menlo, Monaco, Consolas, monospace', scrollback: 5000, theme: { background: '#151515', foreground: '#ddd', cursor: '#ddd', selectionBackground: '#285475' } })
    fit = new FitAddon(); terminal.loadAddon(fit); terminal.open(terminalElement.value!); fit.fit()
    terminal.textarea?.setAttribute('aria-label', '终端输入')
    const url = new URL(`/api/containers/${encodeURIComponent(props.id)}/console/ws`, window.location.href); url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    socket = new WebSocket(url, ['dockerun-console', result.ticket]); socket.binaryType = 'arraybuffer'
    terminal.onData(data => send({ type: 'input', data }))
    terminal.onResize(({ cols, rows }) => send({ type: 'resize', cols, rows }))
    observer = new ResizeObserver(resize); observer.observe(terminalElement.value!)
    socket.onmessage = event => {
      if (disposed || request !== sequence) return
      if (event.data instanceof ArrayBuffer) terminal?.write(new Uint8Array(event.data))
      else {
        const message = JSON.parse(event.data)
        if (message.type === 'ready') { connected.value = true; connecting.value = false; resize(); terminal?.focus() }
        if (message.type === 'error') { error.value = message.message || '控制台连接失败'; connecting.value = false }
        if (message.type === 'exit') { terminal?.writeln('\r\n[控制台连接已结束]'); connected.value = false }
      }
    }
    socket.onerror = () => { if (request === sequence && !disposed) { error.value = '控制台连接失败，请检查容器状态后重试'; connecting.value = false } }
    socket.onclose = () => { if (request === sequence && !disposed) { if (connected.value) terminal?.writeln('\r\n[控制台连接已断开]'); connected.value = false; connecting.value = false } }
  } catch (err) { if (!disposed && request === sequence) { error.value = containerError(err); connecting.value = false } }
}
function disconnect() { sequence++; socket?.close(); socket = undefined; connected.value = false; connecting.value = false; terminal?.writeln('\r\n[已断开连接]'); observer?.disconnect() }
onMounted(() => { if (props.mode === 'attach' && props.details.State?.Status === 'running') void connect() })
onBeforeUnmount(() => { disposed = true; disconnect(); terminal?.dispose() })
</script>
<style scoped>
.console-state { color: #999; font-size: 12px; }.console-warning { color: #dbbc84; line-height: 1.8; margin-bottom: 12px; font-size: 12px; }.container-terminal { height: 480px; min-height: 280px; padding: 14px; background: #151515; border: 1px solid #383838; border-radius: 6px; overflow: hidden; }
</style>
