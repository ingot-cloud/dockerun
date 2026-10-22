<template>
  <el-dialog :model-value="modelValue" title="客户端使用" width="660px" class="builtin-usage-dialog" append-to-body @update:model-value="emit('update:modelValue', $event)">
    <div class="registry-resource builtin-usage">
      <p class="builtin-help">使用 Dockerun 账号登录 <code>{{ host }}</code>，当前权限：<strong>{{ permission }}</strong>。</p>
      <div v-if="connection.http" class="registry-warning builtin-connection-notice" role="status">当前入口使用 HTTP。{{ connection.loopback ? '在 Docker Desktop 中访问宿主机开发服务时，请先按下方说明配置地址及 HTTP 访问。' : '请确认 Docker daemon 已允许此地址使用 HTTP；正式部署建议配置 HTTPS。' }}</div>
      <details v-if="connection.http || connection.loopback" class="builtin-connection-help" :open="connection.http && connection.loopback">
        <summary>Docker 客户端连接配置</summary>
        <div class="builtin-connection-steps">
          <template v-if="connection.loopback">
            <p class="builtin-help">如果 Dockerun 运行在宿主机、Docker daemon 运行在 Docker Desktop 虚拟机，请使用 <code>{{ connection.desktopHost }}</code> 访问宿主机服务。daemon 中的 <code>localhost</code> 可能无法连接宿主机。</p>
            <p class="builtin-help">同时将 Dockerun 的 <code>PUBLIC_URL</code> 改为 <code>{{ connection.desktopUrl }}</code> 并重启后端。仓库返回的认证令牌地址也使用此配置。</p>
            <section v-if="connection.http" class="builtin-command"><header><h3>本地开发后端启动示例（项目根目录）</h3><el-button link aria-label="复制 Docker Desktop 后端启动命令" @click="copyCommand(desktopBackendCommand)"><el-icon><CopyDocument /></el-icon>复制</el-button></header><pre>{{ desktopBackendCommand }}</pre></section>
          </template>
          <template v-if="connection.http">
            <p class="builtin-help">Docker Desktop：进入 Settings → Docker Engine，将下面的地址合并到已有的 <code>insecure-registries</code> 列表，保留其他设置，再点击 Apply &amp; Restart。该配置允许此地址明文传输，仅用于本地调试。</p>
            <section class="builtin-command"><header><h3>Docker Engine 配置片段</h3><el-button link aria-label="复制 HTTP 仓库配置片段" @click="copyCommand(connection.daemonConfig)"><el-icon><CopyDocument /></el-icon>复制</el-button></header><pre>{{ connection.daemonConfig }}</pre></section>
            <p class="builtin-help">Linux Docker Engine：将实际仓库地址合并到 <code>/etc/docker/daemon.json</code> 中的同一配置项，并重启 Docker 服务。容器部署时使用 daemon 可达的发布地址。</p>
          </template>
          <p v-if="connection.loopback" class="builtin-help">配置完成并刷新本页后，使用新的仓库地址登录。只在 <code>docker login</code> 参数中添加 <code>http://</code> 不能替代 daemon 的 HTTP 配置。</p>
          <p v-if="!connection.http" class="builtin-help">HTTPS 证书应覆盖实际仓库域名，并由 Docker daemon 信任。</p>
        </div>
      </details>
      <div v-if="readOnly" class="registry-warning" role="status">仓库正在只读维护，暂时无法推送镜像。</div>
      <section v-for="command in commands" :key="command.label" class="builtin-command"><header><h3>{{ command.label }}</h3><el-button link :aria-label="`复制${command.label}命令`" @click="copyCommand(command.text)"><el-icon><CopyDocument /></el-icon>复制</el-button></header><pre>{{ command.text }}</pre></section>
      <p class="builtin-help">将示例中的 <code>team/app:v1</code> 替换为实际镜像名称与标签。密码请在终端提示时输入。</p>
      <p v-if="canPush" class="builtin-help">普通用户推送后会生成镜像版本。自动部署需要管理员为指定环境授权。</p>
    </div>
    <template #footer><el-button @click="emit('update:modelValue', false)">关闭</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useClipboard } from '@vueuse/core'
import { registryConnectionHelp, registryLoginCommand } from '@/utils/builtin'
const props = defineProps<{ modelValue: boolean; host: string; publicUrl: string; username: string; canPush: boolean; readOnly: boolean; permission: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { copy } = useClipboard({ legacy: true })
const connection = computed(() => registryConnectionHelp(props.publicUrl))
const desktopBackendCommand = computed(() => `make dev-backend PUBLIC_URL=${connection.value.desktopUrl}`)
const commands = computed(() => [
  { label: '登录仓库', text: registryLoginCommand(props.host, props.username) },
  { label: '拉取镜像', text: `docker pull ${props.host}/team/app:v1` },
  ...(props.canPush ? [{ label: '标记并推送镜像', text: `docker tag app:v1 ${props.host}/team/app:v1\ndocker push ${props.host}/team/app:v1` }] : []),
])
async function copyCommand(value: string) { try { await copy(value); ElMessage.success('命令已复制') } catch { ElMessage.error('复制失败，请重试') } }
</script>
