<template>
  <WorkspaceEditor :title="details ? `容器详情 · ${displayName}` : '容器详情'" :loading="loading" :busy="busy" @refresh="load">
    <template v-if="details && summary" #actions>
      <span class="workspace-editor-status" role="status" aria-label="容器状态"><span class="container-detail-state" :class="details.State?.Status">{{ containerStateLabel(details.State?.Status || '') }}</span><span v-if="details.State?.Health">{{ healthLabels[details.State.Health.Status] || details.State.Health.Status }}</span></span>
      <ContainerActionGroup class="container-detail-actions" :containers="[summary]" :busy="busy || loading" :loading="activeAction" @action="runAction" />
    </template>
    <template v-if="error" #feedback><div class="management-error" role="alert"><span>{{ error }}</span><el-button link :disabled="loading || busy" @click="load">重试</el-button></div></template>
    <div class="container-resource workspace-page workspace-editor-page" v-loading="loading" :aria-busy="loading">
    <template v-if="details && summary">
      <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="box" /></span>容器状态</h2></div>
        <table class="resource-table" aria-label="容器状态"><tbody>
          <tr><th>容器 ID</th><td><code>{{ details.Id }}</code><el-button link @click="copyContainerText(details.Id)">复制</el-button></td></tr>
          <tr><th>名称</th><td><form v-if="editingName" class="resource-inline" @submit.prevent="rename"><el-input v-model="newName" aria-label="容器名称" class="form-control" :disabled="busy" /><el-button native-type="submit" :loading="busy">保存</el-button><el-button :disabled="busy" @click="editingName = false">取消</el-button></form><span v-else>{{ displayName }}<el-button link :disabled="busy" @click="newName = displayName; editingName = true">编辑</el-button></span></td></tr>
          <tr><th>状态</th><td><span class="resource-state" :class="details.State?.Status">{{ containerStateLabel(details.State?.Status || '') }}</span><span v-if="details.State?.Health"> · {{ healthLabels[details.State.Health.Status] || details.State.Health.Status }}</span><span v-if="details.State?.Status === 'exited'"> · 退出码 {{ details.State.ExitCode }}</span></td></tr>
          <tr><th>创建时间</th><td>{{ formatImageDate(details.Created) }}</td></tr><tr><th>启动时间</th><td>{{ validDate(details.State?.StartedAt) }}</td></tr><tr v-if="details.State?.Status === 'exited'"><th>停止时间</th><td>{{ validDate(details.State?.FinishedAt) }}</td></tr>
          <tr v-if="details.State?.Error"><th>错误信息</th><td>{{ details.State.Error }}</td></tr>
          <tr><th>快捷操作</th><td><ContainerQuickActions :container="summary" :query="listQuery" labels /></td></tr>
        </tbody></table>
      </section>
      <section v-if="details.State?.Health" class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="chart" /></span>健康检查</h2></div><p class="resource-help">状态：{{ healthLabels[details.State.Health.Status] || details.State.Health.Status }} · 连续失败 {{ details.State.Health.FailingStreak }} 次</p><table class="resource-table" aria-label="健康检查记录"><thead><tr><th>检查时间</th><th>退出码</th><th>输出</th></tr></thead><tbody><tr v-for="(log, index) in details.State.Health.Log" :key="index"><td>{{ formatImageDate(log.Start) }}</td><td>{{ log.ExitCode }}</td><td><code>{{ log.Output }}</code></td></tr></tbody></table></section>
      <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="box" /></span>创建镜像</h2></div><p class="resource-help commit-help">将容器的文件系统和配置保存为本地镜像。挂载的数据卷内容不包含在镜像中。</p><form class="resource-form" @submit.prevent="commit"><label for="commit-image">镜像</label><el-input id="commit-image" v-model="commitImage" placeholder="例如 my-app:backup 或 registry.example.com/app:1.0" :disabled="busy" /><span></span><div class="resource-inline"><el-button native-type="submit" :disabled="busy || !commitImage.trim()" :loading="committing">创建镜像</el-button><span class="resource-help">未填写标签时使用 latest。</span></div></form></section>
      <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="info" /></span>容器详情</h2></div><table class="resource-table" aria-label="容器配置"><tbody>
        <tr><th>镜像</th><td><router-link :to="{ name: 'ImageDetail', params: { id: details.Image || details.Config?.Image || '' } }">{{ details.Config?.Image }}</router-link><div v-if="details.Image" class="resource-muted"><code>{{ details.Image }}</code></div></td></tr>
        <tr><th>端口配置</th><td><div v-for="port in portBindings" :key="port.key"><code>{{ port.host }} → {{ port.port }}</code></div><span v-if="!portBindings.length" class="resource-muted">未发布端口</span></td></tr>
        <tr><th>启动命令</th><td><code>{{ details.Config?.Cmd?.join(' ') || '—' }}</code></td></tr><tr><th>入口命令</th><td><code>{{ details.Config?.Entrypoint?.join(' ') || '—' }}</code></td></tr>
        <tr><th>工作目录</th><td>{{ details.Config?.WorkingDir || '—' }}</td></tr><tr><th>运行用户</th><td>{{ details.Config?.User || '默认用户' }}</td></tr>
        <tr><th>环境变量</th><td><table v-if="environment.length" class="key-value-table" aria-label="容器环境变量"><tbody><tr v-for="(entry, index) in environment" :key="index"><th>{{ entry.name }}</th><td>{{ entry.value }}</td></tr></tbody></table><span v-else class="resource-muted">未设置环境变量</span></td></tr>
        <tr><th>标签</th><td><table v-if="labels.length" class="key-value-table" aria-label="容器标签"><tbody><tr v-for="[key, value] in labels" :key="key"><th>{{ key }}</th><td>{{ value }}</td></tr></tbody></table><span v-else class="resource-muted">未设置标签</span></td></tr>
        <tr><th>重启策略</th><td><form class="resource-inline" @submit.prevent="updateRestart"><el-select v-model="restartPolicy" aria-label="重启策略" class="form-control" :disabled="busy"><el-option v-for="(label, value) in restartPolicies" :key="value" :label="label" :value="value" /></el-select><el-input-number v-if="restartPolicy === 'on-failure'" v-model="restartRetries" :min="0" :max="100000" aria-label="最大重试次数" :disabled="busy" /><span v-if="restartPolicy === 'on-failure'" class="resource-help">最大重试次数，0 表示不限</span><el-button native-type="submit" :disabled="busy || !restartChanged">更新</el-button></form></td></tr>
      </tbody></table></section>
      <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="list" /></span>数据卷与挂载</h2></div><div class="resource-table-scroll"><table class="resource-table" aria-label="容器挂载"><thead><tr><th>宿主机路径或数据卷</th><th>容器内路径</th><th>访问模式</th></tr></thead><tbody><tr v-for="(mount, index) in details.Mounts" :key="index"><td>{{ mount.Source }}</td><td>{{ mount.Destination }}</td><td>{{ mount.RW ? '可读写' : '只读' }}</td></tr><tr v-if="!details.Mounts?.length"><td colspan="3" class="resource-muted">暂无挂载</td></tr></tbody></table></div></section>
      <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="link" /></span>已连接网络</h2><el-input v-model="networkSearch" class="form-control" placeholder="搜索网络…" aria-label="搜索已连接网络" clearable /></div><div class="resource-inline network-connect"><label for="join-network">加入网络</label><el-select id="join-network" v-model="joinNetwork" class="form-control" placeholder="选择网络" :disabled="busy"><el-option v-for="network in availableNetworks" :key="network.Id" :label="network.Name" :value="network.Id" /></el-select><el-button :disabled="busy || !joinNetwork" @click="connectNetwork">加入网络</el-button></div><div class="resource-table-scroll"><table class="resource-table" aria-label="容器网络"><thead><tr><th>网络</th><th>IP 地址</th><th>网关</th><th>MAC 地址</th><th>操作</th></tr></thead><tbody><tr v-for="[name, network] in connectedNetworks" :key="name"><td>{{ name }}</td><td>{{ network.IPAddress || network.GlobalIPv6Address || '—' }}</td><td>{{ network.Gateway || '—' }}</td><td>{{ network.MacAddress || '—' }}</td><td><el-button link :disabled="busy" @click="disconnectNetwork(network.NetworkID || name, name)">离开网络</el-button></td></tr><tr v-if="!connectedNetworks.length"><td colspan="5" class="resource-muted">暂无匹配网络</td></tr></tbody></table></div></section>
    </template>
    </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PortainerIcon from '@/components/PortainerIcon.vue'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import ContainerActionGroup from '@/components/containers/ContainerActionGroup.vue'
import ContainerQuickActions from '@/components/containers/ContainerQuickActions.vue'
import { dockerApi } from '@/api/docker'
import { containerActions, containerListQuery, containerStateLabel, readContainerListState, type Container, type ContainerAction, type ContainerDetails } from '@/utils/containers'
import { containerError, copyContainerText } from '@/utils/containerFeedback'
import { formatImageDate, imageEnvironment } from '@/utils/localImages'
import '@/styles/containers.css'
const route = useRoute(), router = useRouter()
const id = computed(() => String(route.params.id))
const listQuery = computed(() => containerListQuery(readContainerListState(route.query)))
const details = ref<ContainerDetails>(), loading = ref(false), busy = ref(false), error = ref(''), activeAction = ref('')
const displayName = computed(() => details.value?.Name?.replace(/^\//, '') || id.value.slice(0, 12))
const summary = computed<Container | null>(() => details.value ? { Id: details.value.Id, Names: [details.value.Name || ''], State: details.value.State?.Status || '', Image: details.value.Config?.Image || '', Created: 0 } : null)
const environment = computed(() => imageEnvironment(details.value?.Config?.Env))
const labels = computed(() => Object.entries(details.value?.Config?.Labels || {}).sort(([a], [b]) => a.localeCompare(b)))
const portBindings = computed(() => Object.entries(details.value?.HostConfig?.PortBindings || {}).flatMap(([port, bindings]) => (bindings || []).map((binding, index) => ({ key: `${port}-${index}`, host: `${binding.HostIp || '0.0.0.0'}:${binding.HostPort}`, port }))))
const editingName = ref(false), newName = ref(''), commitImage = ref(''), committing = ref(false)
const restartPolicy = ref('no'), restartRetries = ref(0)
const restartPolicies: Record<string, string> = { no: '不自动重启', 'on-failure': '失败时重启', always: '始终重启', 'unless-stopped': '除手动停止外始终重启' }
const healthLabels: Record<string, string> = { healthy: '健康', unhealthy: '不健康', starting: '检查中' }
const restartChanged = computed(() => restartPolicy.value !== (details.value?.HostConfig?.RestartPolicy?.Name || 'no') || (restartPolicy.value === 'on-failure' && restartRetries.value !== (details.value?.HostConfig?.RestartPolicy?.MaximumRetryCount || 0)))
const networks = ref<{ Id: string; Name: string }[]>([]), joinNetwork = ref(''), networkSearch = ref('')
const connectedNetworks = computed(() => Object.entries(details.value?.NetworkSettings?.Networks || {}).filter(([name]) => name.toLowerCase().includes(networkSearch.value.toLowerCase())))
const availableNetworks = computed(() => networks.value.filter(network => !Object.keys(details.value?.NetworkSettings?.Networks || {}).includes(network.Name)))
const validDate = (value?: string) => value && !value.startsWith('0001-') ? formatImageDate(value) : '—'
let controller: AbortController | undefined
async function load() {
  controller?.abort(); const current = new AbortController(); controller = current
  loading.value = true; error.value = ''
  try { const result = await dockerApi.containers.inspect(id.value, current.signal); if (current.signal.aborted) return; details.value = result; restartPolicy.value = result.HostConfig?.RestartPolicy?.Name || 'no'; restartRetries.value = result.HostConfig?.RestartPolicy?.MaximumRetryCount || 0 }
  catch (err) { if (!current.signal.aborted) { details.value = undefined; error.value = `容器详情加载失败：${containerError(err)}` } }
  finally { if (!current.signal.aborted) loading.value = false }
  try { networks.value = await dockerApi.networks.list() as unknown as typeof networks.value || [] } catch { /* 网络列表失败不影响详情，已连接网络仍可查看。 */ }
}
async function mutate(task: () => Promise<unknown>, message: string) {
  if (busy.value) return
  busy.value = true; error.value = ''
  try { await task(); ElMessage.success(message); await load() } catch (err) { error.value = containerError(err) } finally { busy.value = false }
}
async function runAction(action: ContainerAction) {
  if (busy.value) return
  if (action === 'remove' || action === 'kill') {
    try { await ElMessageBox.confirm(action === 'remove' ? `确定删除容器 ${displayName.value}？此操作无法撤销，挂载的数据卷会保留。` : `确定强制停止容器 ${displayName.value}？未保存的数据可能丢失。`, action === 'remove' ? '删除容器' : '强制停止容器', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' }) } catch { return }
  }
  activeAction.value = action
  if (action === 'remove') {
    busy.value = true
    try { await dockerApi.containers.remove(id.value, false); ElMessage.success('容器已删除'); await router.push({ name: 'Containers', query: listQuery.value }) } catch (err) { error.value = containerError(err) } finally { busy.value = false }
  } else await mutate(() => dockerApi.containers[action](id.value), `容器已${containerActions.find(item => item.key === action)!.label}`)
  activeAction.value = ''
}
async function rename() { if (!newName.value.trim()) return; await mutate(() => dockerApi.containers.rename(id.value, newName.value.trim()), '容器名称已更新'); if (!error.value) editingName.value = false }
async function updateRestart() { await mutate(() => dockerApi.containers.updateRestart(id.value, restartPolicy.value, restartPolicy.value === 'on-failure' ? restartRetries.value : 0), '重启策略已更新') }
async function commit() { if (!commitImage.value.trim()) return; committing.value = true; await mutate(() => dockerApi.containers.commit(id.value, commitImage.value.trim()), '镜像已保存到本地镜像'); committing.value = false }
async function connectNetwork() { await mutate(() => dockerApi.networks.connect(joinNetwork.value, id.value), '已加入网络'); if (!error.value) joinNetwork.value = '' }
async function disconnectNetwork(networkId: string, name: string) { try { await ElMessageBox.confirm(`确定让容器离开网络 ${name}？`, '离开网络', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning' }) } catch { return }; await mutate(() => dockerApi.networks.disconnect(networkId, id.value), '已离开网络') }
watch(id, () => { details.value = undefined; editingName.value = false; void load() }, { immediate: true })
onBeforeUnmount(() => controller?.abort())
</script>
<style scoped>
.container-detail-state { display: inline-flex; align-items: center; border-radius: 10px; padding: 0 8px; line-height: 20px; background: #414141; color: #ddd; font-size: 11px; }
.container-detail-state.running { background: #145f3c; color: #d2f5e4; }
.container-detail-state.paused { background: #665023; color: #f9e1b5; }
.commit-help { margin-bottom: 18px; }.network-connect { margin-bottom: 16px; }
@media (max-width: 760px) {
  .container-detail-actions { flex-wrap: wrap; gap: 6px; max-width: 100%; min-width: 0; white-space: normal; }
  .container-detail-actions :deep(.action-wrapper + .action-wrapper) { margin-left: 0; }
  .container-detail-actions :deep(.el-button) { border-radius: 4px; }
}
</style>
