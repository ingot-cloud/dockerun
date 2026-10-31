<template>
  <section class="container-console" aria-label="容器控制台">
    <div class="resource-card console-settings">
      <form v-if="mode === 'exec'" class="console-toolbar console-toolbar--exec" @submit.prevent="connect">
        <label class="console-field console-command" for="console-command">
          <span>命令</span>
          <el-input v-if="custom" id="console-command" v-model="customCommand" placeholder="例如 /bin/sh -c 'echo hello'" :disabled="connected || connecting" aria-label="自定义执行命令" />
          <el-select v-else id="console-command" v-model="command" :disabled="connected || connecting" aria-label="执行命令"><el-option v-for="shell in ['/bin/ash', '/bin/bash', '/bin/dash', '/bin/sh']" :key="shell" :label="shell" :value="shell" /></el-select>
        </label>
        <label class="console-custom" for="console-custom"><el-switch id="console-custom" v-model="custom" size="small" :disabled="connected || connecting" aria-label="使用自定义命令" /><span>自定义命令</span></label>
        <label class="console-field console-user" for="console-user"><span>用户</span><el-input id="console-user" v-model="user" placeholder="root" :disabled="connected || connecting" aria-label="执行用户" /></label>
        <span class="console-state" :class="{ connected }" role="status">{{ connecting ? '正在连接…' : connected ? '已连接' : '未连接' }}</span>
        <div class="console-actions"><el-button v-if="!connected" native-type="submit" type="primary" :loading="connecting" :disabled="details.State?.Status !== 'running'">连接</el-button><el-button v-else native-type="button" @click="disconnect">断开连接</el-button></div>
      </form>
      <div v-else class="console-toolbar console-toolbar--attach">
        <span class="console-state" :class="{ connected }" role="status">{{ connecting ? '正在连接…' : connected ? '已连接' : '未连接' }}</span>
        <el-button v-if="connected" @click="disconnect">分离</el-button><el-button v-else type="primary" :loading="connecting" :disabled="details.State?.Status !== 'running'" @click="connect">附加到容器</el-button>
      </div>
      <p v-if="details.State?.Status !== 'running'" class="console-warning" role="status">容器当前未运行，无法连接控制台。</p>
      <template v-if="mode === 'attach'"><p v-if="!details.Config?.OpenStdin" class="console-warning" role="status">此容器未启用交互输入，控制台可能无法接收输入。</p><p v-if="!details.Config?.Tty" class="console-warning" role="status">此容器未启用终端，控制台可能无法正常显示。</p></template>
    </div>
    <div v-if="error" class="console-error" role="alert">{{ error }}</div>
    <div class="console-terminal-panel" :aria-busy="connecting">
      <div v-show="terminalVisible" ref="terminalElement" class="container-terminal" role="region" aria-label="容器交互终端"></div>
      <div v-if="!terminalVisible" class="console-empty"><PortainerIcon :name="mode === 'exec' ? 'terminal' : 'attach'" /><span>{{ connecting ? '正在连接终端…' : details.State?.Status !== 'running' ? '容器启动后可连接终端' : '连接后可在此操作容器终端' }}</span></div>
    </div>
  </section>
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
function resize() { if (!terminal || !fit || !terminalElement.value?.clientWidth || !terminalElement.value.clientHeight) return; fit.fit(); send({ type: 'resize', cols: terminal.cols, rows: terminal.rows }) }
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
    terminal = new Terminal({ cursorBlink: true, fontSize: 13, fontFamily: 'Menlo, Monaco, Consolas, monospace', scrollback: 5000, theme: { background: '#171717', foreground: '#ddd', cursor: '#ddd', selectionBackground: '#285475' } })
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
.container-console { display: flex; flex-direction: column; flex: 1; min-width: 0; min-height: 0; gap: 12px; overflow: hidden; }
.container-resource .console-settings { flex-shrink: 0; padding: 12px 16px; }
.console-toolbar { display: flex; align-items: center; flex-wrap: wrap; min-width: 0; gap: 8px 12px; }
.console-field { display: flex; align-items: center; min-width: 0; gap: 8px; }
.console-field > span { flex-shrink: 0; color: #aaa; font-size: 12px; }
.console-command { flex: 1 1 240px; }.console-command :deep(.el-select), .console-command :deep(.el-input) { flex: 1; min-width: 0; }
.console-custom { display: inline-flex; align-items: center; flex-shrink: 0; gap: 6px; color: #bbb; font-size: 12px; white-space: nowrap; cursor: pointer; }
.console-user { flex: 0 1 160px; }.console-user :deep(.el-input) { min-width: 0; }
.console-state { flex-shrink: 0; color: #999; font-size: 12px; line-height: 24px; white-space: nowrap; }.console-state.connected { color: #9acb8c; }
.console-actions { display: flex; align-items: center; }.console-toolbar .el-button { margin: 0; }
.console-toolbar--attach { justify-content: space-between; }
.console-warning { margin: 8px 0 0; color: #e6c48e; line-height: 20px; font-size: 12px; overflow-wrap: anywhere; }
.console-error { flex-shrink: 0; padding: 8px 12px; border: 1px solid #693a40; border-radius: 6px; background: #3c2528; color: #ffc3c8; font-size: 12px; line-height: 20px; overflow-wrap: anywhere; }
.console-terminal-panel { flex: 1; min-height: 0; min-width: 0; padding: 10px; background: #171717; border: 1px solid #383838; border-radius: 6px; overflow: hidden; }
.container-terminal { height: 100%; width: 100%; min-height: 0; min-width: 0; overflow: hidden; }
.container-terminal :deep(.xterm) { height: 100%; }
.container-terminal :deep(.xterm-viewport) { overscroll-behavior: contain; }
.console-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; height: 100%; min-height: 0; color: #888; font-size: 12px; text-align: center; }
.console-empty .portainer-icon { width: 24px; height: 24px; }
@media (max-width: 760px) {
  .container-resource .console-settings { padding: 10px 12px; }
  .console-toolbar--exec { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px 12px; }
  .console-command { grid-column: 1 / -1; }.console-user { min-width: 0; }
  .console-actions { justify-content: flex-end; }
  .console-terminal-panel { padding: 8px; }
}
</style>
