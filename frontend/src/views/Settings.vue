<template>
  <div class="registry-resource maintenance-page">
    <header class="registry-heading">
      <h1>仓库与维护</h1>
      <div class="registry-heading-actions"><router-link to="/builtin" class="maintenance-browse"><PortainerIcon name="box" />浏览内置仓库</router-link><el-button text circle aria-label="刷新仓库与维护" title="刷新仓库与维护" :disabled="refreshing || busy" @click="refresh(true)"><PortainerIcon name="refresh" :class="{ spinning: refreshing }" /></el-button></div>
    </header>
    <div v-if="errors.status" class="registry-error" role="alert"><span>仓库状态加载失败：{{ errors.status }}</span><el-button link :disabled="refreshing || busy" @click="refresh(true)">重试</el-button></div>
    <section v-loading="initialLoading" class="registry-card maintenance-status" aria-labelledby="maintenance-status-title" :aria-busy="initialLoading">
      <header class="maintenance-status-heading"><h2 id="maintenance-status-title"><span class="registry-section-icon"><PortainerIcon name="registry" /></span>内置仓库<span v-if="status" class="maintenance-state" :class="{ offline: !status.enabled, readonly: status.read_only }"><i />{{ !status.enabled ? '未启用' : status.operation === 'backup' ? '备份保护中' : status.operation === 'registry_gc' ? '镜像层清理中' : status.read_only ? '只读维护中' : '运行中' }}</span></h2><span class="maintenance-engine">{{ status?.engine || '—' }}</span></header>
      <dl class="maintenance-status-grid">
        <div class="maintenance-endpoint"><dt>访问地址</dt><dd><code :title="status?.public_url">{{ status?.public_url || '—' }}</code><el-button v-if="status?.public_url" link aria-label="复制仓库访问地址" title="复制仓库访问地址" @click="copyAddress"><el-icon><CopyDocument /></el-icon></el-button></dd></div>
        <div><dt>仓库存储占用</dt><dd :title="status?.storage_error">{{ maintenanceStorage(status?.storage_bytes) }}<span v-if="status?.storage_error" class="maintenance-storage-error">读取失败</span></dd></div>
        <div><dt>传输协议</dt><dd>{{ status ? maintenanceTransport(status) : '—' }}</dd></div>
      </dl>
      <div v-if="status?.storage_error" class="registry-warning" role="status">{{ status.storage_error }}</div>
      <div class="maintenance-readonly-row"><div><strong>只读维护</strong><p>{{ status?.operation ? '后台维护任务正在保护仓库，完成前无法切换写入状态。' : backupPending || collectionPending ? '已有维护任务排队或执行中，完成后可切换只读维护。' : manualReadOnly ? '推送和删除已暂停；清理完成后，请手动关闭只读维护。' : '开启后暂停推送和删除，等待在途请求结束，再执行镜像层清理。' }}</p></div><el-switch :model-value="manualReadOnly" :disabled="!canToggleReadOnly" :loading="action === 'readonly'" aria-label="只读维护" @change="setReadOnly(Boolean($event))" /></div>
    </section>
    <div v-if="status && !status.enabled" class="registry-warning" role="status">内置仓库尚未启用。请在安装配置中启用后重启服务；仍可创建完整备份。</div>
    <div class="maintenance-action-grid">
      <section class="registry-card maintenance-operation" aria-labelledby="maintenance-gc-title"><div class="maintenance-operation-title"><span class="registry-section-icon"><PortainerIcon name="trash" /></span><h2 id="maintenance-gc-title">清理未引用镜像层</h2></div><p>保留已保存的镜像清单及其引用的内容。先开启只读维护，任务将在后台执行。</p><div class="maintenance-operation-footer"><span>{{ collectionPending ? '已有清理任务等待执行或执行中' : !manualReadOnly ? '需要只读维护窗口' : '完成后保持只读维护' }}</span><el-button :loading="action === 'registry_gc'" :disabled="!canCollect || busy || refreshing" @click="submitTask('registry_gc')"><PortainerIcon name="trash" />{{ collectionPending ? '已有清理任务' : '开始清理' }}</el-button></div></section>
      <section class="registry-card maintenance-operation" aria-labelledby="maintenance-backup-title"><div class="maintenance-operation-title"><span class="registry-section-icon"><PortainerIcon name="download" /></span><h2 id="maintenance-backup-title">完整备份</h2><el-button link @click="scopeVisible = true"><PortainerIcon name="info" />备份范围</el-button></div><p>打包元数据、密钥、仓库内容及发布材料。备份期间暂停写入，完成后恢复原维护状态。</p><div class="maintenance-operation-footer"><span>{{ backupPending ? '已有备份任务等待执行或执行中' : '后台执行，离开页面后仍会继续' }}</span><el-button class="registry-primary" :loading="action === 'backup'" :disabled="backupPending || busy || refreshing" @click="submitTask('backup')"><PortainerIcon name="plus" />{{ backupPending ? '已有备份任务' : '创建完整备份' }}</el-button></div></section>
    </div>
    <div v-if="actionError" class="registry-error" role="alert"><span>{{ actionError }}</span><el-button link :disabled="busy" @click="actionError = ''">关闭</el-button></div>
    <div v-if="submissionMessage" class="maintenance-success" role="status"><span>{{ submissionMessage }}</span><el-button link @click="submissionMessage = ''">关闭</el-button></div>
    <div v-if="errors.backups || errors.collections || errors.audit" class="registry-warning maintenance-partial-error" role="status"><span>{{ !essentialLoaded && needsMaintenancePolling(backups, collections, status?.operation) ? '部分任务状态读取失败，自动更新已暂停。请刷新后重试。' : '部分任务或审计记录未能加载，其他内容仍可查看；切换相应标签可查看详情。' }}</span><el-button link :disabled="refreshing || busy" @click="refresh(true)">重试</el-button></div>
    <MaintenanceHistory ref="history" :backups="backups" :collections="collections" :events="events" :loading="refreshing" :errors="errors" :busy="busy" :polling="polling" :downloading="downloading" :can-collect="canCollect" :last-updated="lastUpdated" @refresh="refresh(true)" @download="downloadBackup" @retry="submitTask" />
    <el-dialog v-model="scopeVisible" title="完整备份范围" width="600px" class="maintenance-scope-dialog" append-to-body>
      <div class="registry-resource maintenance-scope"><p>完整备份包含恢复 Dockerun 所需的元数据和文件：</p><ul><li>数据库一致性快照（SQLite 或 MySQL）</li><li>数据目录中的密钥、内置仓库内容</li><li>发布材料、已上传离线材料和任务材料</li></ul><p>业务数据卷和外部 Registry 内容需要分别备份。通过环境变量提供的密钥应随安装配置保存。</p><p>恢复需停止服务，并使用命令行恢复到新的空数据目录；MySQL 还需单独导入备份中的数据库快照。</p></div>
      <template #footer><el-button @click="scopeVisible = false">关闭</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import PortainerIcon from '@/components/PortainerIcon.vue'
import MaintenanceHistory from '@/components/maintenance/MaintenanceHistory.vue'
import { maintenanceApi, type AuditEvent, type MaintenanceJob, type MaintenanceKind, type RegistryMaintenanceStatus } from '@/api/maintenance'
import { isActiveMaintenanceJob, maintenanceError, maintenanceStorage, maintenanceTransport, needsMaintenancePolling } from '@/utils/maintenance'
import '@/styles/registries.css'
import '@/styles/maintenance.css'

const status = ref<RegistryMaintenanceStatus>(), backups = ref<MaintenanceJob[]>([]), collections = ref<MaintenanceJob[]>([]), events = ref<AuditEvent[]>([])
const errors = reactive({ status: '', backups: '', collections: '', audit: '' })
const refreshing = ref(false), initialLoading = ref(true), confirming = ref(false), action = ref<MaintenanceKind | 'readonly' | ''>('')
const actionError = ref(''), submissionMessage = ref(''), scopeVisible = ref(false), downloading = ref(0), lastUpdated = ref('')
const history = ref<InstanceType<typeof MaintenanceHistory>>()
const busy = computed(() => Boolean(action.value || confirming.value))
const manualReadOnly = computed(() => status.value?.manual_read_only ?? status.value?.read_only ?? false)
const backupPending = computed(() => backups.value.some(isActiveMaintenanceJob))
const collectionPending = computed(() => collections.value.some(isActiveMaintenanceJob))
const essentialLoaded = computed(() => !errors.status && !errors.backups && !errors.collections)
const polling = computed(() => essentialLoaded.value && needsMaintenancePolling(backups.value, collections.value, status.value?.operation))
const canToggleReadOnly = computed(() => Boolean(status.value?.enabled && essentialLoaded.value && !status.value.operation && !backupPending.value && !collectionPending.value && !busy.value && !refreshing.value))
const canCollect = computed(() => Boolean(status.value?.enabled && manualReadOnly.value && essentialLoaded.value && !status.value.operation && !collectionPending.value))
const controller = new AbortController()
const { copy } = useClipboard({ legacy: true })
let timer: ReturnType<typeof setTimeout> | undefined
let disposed = false
let refreshAgain = false
function stopPolling() { clearTimeout(timer); timer = undefined }
function schedulePolling() { stopPolling(); if (!disposed && !busy.value && polling.value) timer = setTimeout(() => { void refresh(false) }, 3000) }
async function fetchGroup(key: keyof typeof errors) {
  try {
    if (key === 'status') { const result = await maintenanceApi.status(controller.signal); if (!disposed) status.value = result }
    if (key === 'backups') { const result = await maintenanceApi.backups(controller.signal); if (!disposed) backups.value = result || [] }
    if (key === 'collections') { const result = await maintenanceApi.collections(controller.signal); if (!disposed) collections.value = result || [] }
    if (key === 'audit') { const result = await maintenanceApi.audit(controller.signal); if (!disposed) events.value = result || [] }
    if (!disposed) errors[key] = ''
  } catch (error) { if (!disposed) errors[key] = maintenanceError(error) }
}
async function refresh(full = true) {
  if (disposed) return
  if (refreshing.value) { refreshAgain = true; return }
  stopPolling(); refreshing.value = true
  const before = [...backups.value, ...collections.value].map(job => `${job.id}:${job.status}`).join(',')
  try {
    const groups: (keyof typeof errors)[] = full ? ['status', 'backups', 'collections', 'audit'] : ['status', 'backups', 'collections']
    await Promise.all(groups.map(fetchGroup))
    if (disposed) return
    const after = [...backups.value, ...collections.value].map(job => `${job.id}:${job.status}`).join(',')
    if (!full && before !== after) await fetchGroup('audit')
    if (essentialLoaded.value) lastUpdated.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  } finally {
    if (!disposed) { refreshing.value = false; initialLoading.value = false; if (refreshAgain) { refreshAgain = false; void refresh(true) } else schedulePolling() }
  }
}
async function copyAddress() { if (!status.value?.public_url) return; try { await copy(status.value.public_url); ElMessage.success('仓库地址已复制') } catch { ElMessage.error('复制失败，请重试') } }
async function setReadOnly(value: boolean) {
  if (!canToggleReadOnly.value) return
  confirming.value = true; stopPolling()
  try { await ElMessageBox.confirm(value ? '开启只读维护后，将暂停推送和删除，并等待在途请求结束。浏览与拉取仍可使用。' : '结束只读维护后，仓库将恢复接受推送和删除。确定继续？', value ? '开启只读维护' : '结束只读维护', { type: 'warning', confirmButtonText: value ? '开启只读维护' : '恢复写入', cancelButtonText: '取消', closeOnClickModal: false }) }
  catch { return }
  finally { confirming.value = false; schedulePolling() }
  if (disposed) return
  action.value = 'readonly'; actionError.value = ''; submissionMessage.value = ''; stopPolling()
  try { const result = await maintenanceApi.setReadOnly(value); if (!disposed) { if (status.value) Object.assign(status.value, result); ElMessage.success(value ? '只读维护已开启' : '已结束只读维护') } }
  catch (error) { if (!disposed) actionError.value = `切换只读维护失败：${maintenanceError(error)}` }
  finally { if (!disposed) { action.value = ''; await refresh(true) } }
}
async function submitTask(kind: MaintenanceKind) {
  if (busy.value || refreshing.value || disposed || (kind === 'backup' && backupPending.value)) return
  if (kind === 'registry_gc') {
    if (!canCollect.value) { actionError.value = '请先开启只读维护，并等待当前清理任务完成'; return }
    confirming.value = true; stopPolling()
    try { await ElMessageBox.confirm('将创建后台任务，清理未被镜像清单引用的镜像层。已保存的清单及其引用会保留；完成后仍保持只读维护。', '清理未引用镜像层', { type: 'warning', confirmButtonText: '创建清理任务', cancelButtonText: '取消', closeOnClickModal: false }) }
    catch { return }
    finally { confirming.value = false; schedulePolling() }
  }
  if (disposed) return
  action.value = kind; actionError.value = ''; submissionMessage.value = ''; stopPolling()
  try {
    const result = kind === 'backup' ? await maintenanceApi.backup() : await maintenanceApi.collect()
    if (disposed) return
    history.value?.show(kind)
    submissionMessage.value = `任务 #${result.job_id}：${result.message}`
    ElMessage.success(result.reused ? '已定位到现有任务' : '后台任务已创建')
  } catch (error) { if (!disposed) actionError.value = `任务创建失败：${maintenanceError(error)}` }
  finally { if (!disposed) { action.value = ''; await refresh(true) } }
}
async function downloadBackup(job: MaintenanceJob) {
  if (downloading.value || job.status !== 'ready' || busy.value || disposed) return
  downloading.value = job.id; actionError.value = ''
  try {
    const blob = await maintenanceApi.download(job.id, controller.signal)
    if (disposed) return
    const url = URL.createObjectURL(blob), anchor = document.createElement('a')
    anchor.href = url; anchor.download = job.file || `dockerun-backup-${job.id}.tar.gz`
    document.body.appendChild(anchor); anchor.click(); anchor.remove()
    setTimeout(() => URL.revokeObjectURL(url), 60000)
    ElMessage.success('备份文件下载已开始')
  } catch (error) { if (!disposed) actionError.value = `备份下载失败：${maintenanceError(error)}` }
  finally { if (!disposed) downloading.value = 0 }
}
onMounted(() => { void refresh(true) })
onBeforeUnmount(() => { disposed = true; stopPolling(); controller.abort() })
</script>
