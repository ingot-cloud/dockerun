<template>
  <div class="image-detail-page">
    <header class="detail-heading">
      <h1>镜像详情</h1>
      <div class="detail-heading-actions">
        <router-link :to="listLocation">返回镜像列表</router-link>
        <el-button text circle aria-label="刷新镜像详情" title="刷新镜像详情" :disabled="loading || busy" @click="loadImage"><PortainerIcon name="refresh" /></el-button>
      </div>
    </header>

    <el-alert v-if="detailError" title="镜像详情加载失败" :description="detailError" type="error" :closable="false" show-icon>
      <el-button link @click="loadImage">重新加载</el-button>
    </el-alert>
    <div v-if="loading && !image" class="detail-card detail-loading" role="status">正在加载镜像详情…</div>

    <template v-if="image">
      <section class="detail-card" aria-labelledby="detail-tags-title">
        <h2 id="detail-tags-title"><span class="detail-icon"><PortainerIcon name="tag" /></span>镜像标签</h2>
        <div class="detail-card-body tag-list">
          <span v-for="tag in tags" :key="tag" class="detail-tag">{{ tag }}</span>
          <span v-if="!tags.length" class="detail-tag tag-empty">未标记</span>
        </div>
      </section>

      <section class="detail-card" aria-labelledby="tag-image-title">
        <h2 id="tag-image-title"><span class="detail-icon"><PortainerIcon name="tag" /></span>为镜像添加标签</h2>
        <form class="tag-image-form" @submit.prevent="tagImage">
          <label for="image-new-tag">新标签</label>
          <el-input id="image-new-tag" v-model="newTag" :disabled="busy || loading" placeholder="例如 myapp:1.0 或 registry.example.com/myapp:1.0" autocomplete="off" />
          <el-button native-type="submit" :loading="tagging" :disabled="busy || loading || !newTag.trim()">添加标签</el-button>
          <p>为此镜像添加新的本地引用，未填写版本时使用 latest。</p>
        </form>
        <el-alert v-if="tagError" class="tag-error" title="添加标签失败" :description="tagError" type="error" :closable="false" show-icon />
      </section>

      <section class="detail-card" aria-labelledby="image-info-title">
        <h2 id="image-info-title"><span class="detail-icon"><PortainerIcon name="list" /></span>镜像信息</h2>
        <el-alert v-if="actionError" title="操作失败" :description="actionError" type="error" show-icon @close="actionError = ''" />
        <table class="metadata-table" aria-label="镜像基本信息">
          <tbody>
            <tr><th scope="row">ID</th><td><div class="id-and-actions"><code class="full-image-id">{{ image.Id }}</code><div class="image-detail-actions">
              <el-button class="detail-remove" :disabled="busy || loading" @click="openDelete"><PortainerIcon name="trash" />删除镜像</el-button>
              <el-button :disabled="busy || loading" :loading="exporting" @click="exportImage"><PortainerIcon v-if="!exporting" name="download" />{{ exporting ? '导出中…' : '导出镜像' }}</el-button>
            </div></div></td></tr>
            <tr><th scope="row">大小</th><td>{{ formatImageSize(image.Size) }}</td></tr>
            <tr><th scope="row">创建时间</th><td>{{ formatImageDate(image.Created) }}</td></tr>
            <tr><th scope="row">构建平台</th><td>{{ platform }}<span v-if="image.DockerVersion" class="detail-muted"> · Docker {{ image.DockerVersion }}</span></td></tr>
            <tr><th scope="row">使用状态</th><td>
              <span v-if="useCount === 0" class="usage-badge unused">未使用</span>
              <span v-else-if="useCount !== null" class="usage-badge in-use">使用中</span>
              <span v-else class="detail-muted">{{ usageLoading ? '正在读取容器使用状态…' : usageError ? '暂时无法读取容器使用状态' : '暂时无法确认容器使用状态' }}</span>
              <span v-if="useCount !== null" class="detail-muted usage-description">{{ useCount ? `${useCount} 个容器引用（包含已停止的容器）` : '没有容器引用此镜像' }}</span>
            </td></tr>
            <tr v-if="image.Author"><th scope="row">作者</th><td>{{ image.Author }}</td></tr>
            <tr v-if="image.RepoDigests?.length"><th scope="row">镜像摘要</th><td><div v-for="digest in image.RepoDigests" :key="digest" class="digest-value">{{ digest }}</div></td></tr>
            <tr><th scope="row">标签（Labels）</th><td>
              <table v-if="labels.length" class="key-value-table" aria-label="镜像元数据标签"><tbody><tr v-for="[key, value] in labels" :key="key"><th scope="row">{{ key }}</th><td>{{ value }}</td></tr></tbody></table>
              <span v-else class="detail-muted">暂无元数据标签</span>
            </td></tr>
          </tbody>
        </table>
      </section>

      <section class="detail-card" aria-labelledby="dockerfile-title">
        <h2 id="dockerfile-title"><span class="detail-icon"><PortainerIcon name="list" /></span>Dockerfile 配置</h2>
        <table class="metadata-table" aria-label="Dockerfile 配置"><tbody>
          <tr><th scope="row">启动命令（CMD）</th><td><code>{{ config.Cmd?.join(' ') || '—' }}</code></td></tr>
          <tr><th scope="row">入口命令（ENTRYPOINT）</th><td><code>{{ config.Entrypoint?.join(' ') || '—' }}</code></td></tr>
          <tr><th scope="row">声明端口（EXPOSE）</th><td><div v-if="ports.length" class="exposed-ports"><code v-for="port in ports" :key="port">{{ port }}</code></div><span v-else class="detail-muted">未声明端口</span></td></tr>
          <tr><th scope="row">环境变量（ENV）</th><td>
            <table v-if="environment.length" class="key-value-table" aria-label="镜像环境变量"><tbody><tr v-for="(variable, index) in environment" :key="index"><th scope="row">{{ variable.name }}</th><td>{{ variable.value }}</td></tr></tbody></table>
            <span v-else class="detail-muted">未设置环境变量</span>
          </td></tr>
          <tr v-if="config.User"><th scope="row">运行用户（USER）</th><td><code>{{ config.User }}</code></td></tr>
          <tr v-if="config.WorkingDir"><th scope="row">工作目录（WORKDIR）</th><td><code>{{ config.WorkingDir }}</code></td></tr>
          <tr v-if="volumes.length"><th scope="row">数据卷（VOLUME）</th><td><div v-for="volume in volumes" :key="volume"><code>{{ volume }}</code></div></td></tr>
          <tr v-if="config.Healthcheck?.Test?.length"><th scope="row">健康检查（HEALTHCHECK）</th><td><code>{{ config.Healthcheck.Test.join(' ') }}</code></td></tr>
        </tbody></table>
      </section>

      <section class="detail-card" aria-labelledby="image-layers-title">
        <h2 id="image-layers-title"><span class="detail-icon"><PortainerIcon name="list" /></span>镜像层<span v-if="!historyLoading && !historyError" class="layer-count">共 {{ layers.length }} 条构建记录</span></h2>
        <div v-if="historyLoading" class="detail-loading" role="status">正在读取镜像层历史…</div>
        <div v-else-if="historyError" class="history-error" role="alert"><p>镜像层历史加载失败：{{ historyError }}</p><el-button link @click="loadImage">重试</el-button></div>
        <div v-else-if="!layers.length" class="detail-loading">暂无镜像层历史</div>
        <div v-else class="layers-table-scroll">
          <table class="layers-table" aria-label="镜像层历史">
            <thead><tr>
              <th scope="col" :aria-sort="historySort === 'index' ? (historyDescending ? 'descending' : 'ascending') : 'none'"><button @click="sortLayers('index')">序号<PortainerIcon name="sort" /></button></th>
              <th scope="col" :aria-sort="historySort === 'size' ? (historyDescending ? 'descending' : 'ascending') : 'none'"><button @click="sortLayers('size')">大小<PortainerIcon name="sort" /></button></th>
              <th scope="col">构建指令</th>
            </tr></thead>
            <tbody><tr v-for="layer in layers" :key="layer.index">
              <td>{{ layer.index }}</td><td>{{ formatImageSize(layer.Size) }}</td>
              <td><code class="layer-command" :class="{ collapsed: (layer.command.length > 130 || layer.command.includes('\n')) && !expandedLayers.has(layer.index) }">{{ layer.command }}</code>
                <button v-if="layer.command.length > 130 || layer.command.includes('\n')" class="expand-command" :aria-expanded="expandedLayers.has(layer.index)" :aria-label="`${expandedLayers.has(layer.index) ? '收起' : '展开'}第 ${layer.index} 条构建指令`" @click="toggleLayer(layer.index)">{{ expandedLayers.has(layer.index) ? '收起' : '展开' }}</button>
              </td>
            </tr></tbody>
          </table>
        </div>
      </section>
    </template>

    <el-dialog v-model="deleteVisible" title="删除镜像" width="540px" class="image-detail-dialog" append-to-body :show-close="!deleting" :close-on-click-modal="!deleting" :close-on-press-escape="!deleting">
      <p>确定删除此镜像及其标签？此操作无法撤销。</p>
      <code class="delete-image-id">{{ image?.Id }}</code>
      <el-checkbox v-model="forceDelete" :disabled="deleting">强制删除镜像</el-checkbox>
      <p class="detail-dialog-help">被保留版本引用的镜像仍无法删除。</p>
      <el-alert v-if="deleteError" title="删除失败" :description="deleteError" type="error" :closable="false" show-icon />
      <template #footer><el-button :disabled="deleting" @click="deleteVisible = false">取消</el-button><el-button type="danger" :loading="deleting" @click="deleteImage">{{ forceDelete ? '确认强制删除' : '确认删除' }}</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { localImagesApi } from '@/api/localImages'
import { formatImageDate, formatImageSize, imageEnvironment, imageExportName, imageLayerCommand, imageListQuery, imageUseCount, readImageListState, type ImageContainer, type ImageHistoryLayer, type LocalImage, type LocalImageDetails } from '@/utils/localImages'

const route = useRoute()
const router = useRouter()
const image = ref<LocalImageDetails | null>(null)
const containers = ref<ImageContainer[] | null>(null)
const history = ref<ImageHistoryLayer[]>([])
const loading = ref(false)
const historyLoading = ref(false)
const detailError = ref('')
const historyError = ref('')
const usageError = ref('')
const usageLoading = ref(false)
const actionError = ref('')
const newTag = ref('')
const tagging = ref(false)
const tagError = ref('')
const exporting = ref(false)
const deleting = ref(false)
const deleteVisible = ref(false)
const forceDelete = ref(false)
const deleteError = ref('')
const busy = computed(() => tagging.value || exporting.value || deleting.value)
const imageId = computed(() => String(route.params.id || ''))
const listLocation = computed(() => ({ name: 'Images', query: imageListQuery(readImageListState(route.query)) }))
const config = computed(() => image.value?.Config || {})
const tags = computed(() => (image.value?.RepoTags || []).filter(tag => tag !== '<none>:<none>'))
const labels = computed(() => Object.entries(config.value.Labels || {}).sort(([a], [b]) => a.localeCompare(b)))
const ports = computed(() => Object.keys(config.value.ExposedPorts || {}).sort())
const volumes = computed(() => Object.keys(config.value.Volumes || {}).sort())
const environment = computed(() => imageEnvironment(config.value.Env))
const platform = computed(() => [image.value?.Os, image.value?.Architecture, image.value?.Variant].filter(Boolean).join(' / ') || '—')
const useCount = computed(() => imageUseCount(imageId.value, containers.value))
const historySort = ref<'index' | 'size'>('index')
const historyDescending = ref(false)
const expandedLayers = ref(new Set<number>())
const layers = computed(() => [...history.value].reverse()
  .map((layer, index) => ({ ...layer, index, command: imageLayerCommand(layer.CreatedBy) }))
  .sort((a, b) => ((historySort.value === 'index' ? a.index - b.index : a.Size - b.Size) * (historyDescending.value ? -1 : 1)) || a.index - b.index))
let lifetime = new AbortController()
let readController: AbortController | undefined
const errorText = (error: unknown) => error instanceof Error ? error.message : '操作失败，请重试'

async function loadImage() {
  readController?.abort()
  const controller = new AbortController()
  readController = controller
  const signal = controller.signal
  loading.value = true
  historyLoading.value = true
  detailError.value = ''
  historyError.value = ''
  usageError.value = ''
  usageLoading.value = true
  containers.value = null
  await Promise.allSettled([
    localImagesApi.inspect(imageId.value, signal).then(value => { if (!signal.aborted) image.value = value })
      .catch(error => { if (!signal.aborted) { detailError.value = errorText(error); image.value = null } })
      .finally(() => { if (!signal.aborted) loading.value = false }),
    localImagesApi.history(imageId.value, signal).then(value => { if (!signal.aborted) history.value = value })
      .catch(error => { if (!signal.aborted) { historyError.value = errorText(error); history.value = [] } })
      .finally(() => { if (!signal.aborted) historyLoading.value = false }),
    localImagesApi.containers(signal).then(value => { if (!signal.aborted) containers.value = value })
      .catch(error => { if (!signal.aborted) usageError.value = errorText(error) })
      .finally(() => { if (!signal.aborted) usageLoading.value = false }),
  ])
}
function sortLayers(key: 'index' | 'size') {
  historyDescending.value = historySort.value === key ? !historyDescending.value : key === 'size'
  historySort.value = key
}
function toggleLayer(index: number) {
  const next = new Set(expandedLayers.value)
  if (next.has(index)) next.delete(index)
  else next.add(index)
  expandedLayers.value = next
}
async function tagImage() {
  if (busy.value || loading.value || !image.value || !newTag.value.trim()) return
  tagError.value = ''
  let target = newTag.value.trim()
  if (/\s|@/.test(target)) { tagError.value = '请输入镜像仓库与标签，例如 myapp:1.0'; return }
  if (!target.split('/').pop()?.includes(':')) target += ':latest'
  tagging.value = true
  const signal = lifetime.signal
  const source = image.value.Id
  try {
    const existing = (await localImagesApi.list(signal)).find(item => item.RepoTags?.includes(target))
    if (existing && existing.Id !== source) {
      await ElMessageBox.confirm(`标签 ${target} 已指向其他镜像，是否将它改为指向当前镜像？`, '替换镜像标签', { confirmButtonText: '确认替换', cancelButtonText: '取消', type: 'warning' })
    }
    if (signal.aborted) return
    await localImagesApi.tag(source, target, signal)
    if (signal.aborted) return
    ElMessage.success('镜像标签添加成功')
    newTag.value = ''
    await loadImage()
  } catch (error) {
    if (!signal.aborted && error !== 'cancel' && error !== 'close') tagError.value = errorText(error)
  } finally { if (signal === lifetime.signal) tagging.value = false }
}
function openDelete() {
  forceDelete.value = false
  deleteError.value = ''
  deleteVisible.value = true
}
async function deleteImage() {
  if (busy.value || !image.value) return
  deleting.value = true
  deleteError.value = ''
  const signal = lifetime.signal
  const source = image.value.Id
  try {
    await localImagesApi.remove(source, forceDelete.value, signal)
    if (signal.aborted) return
    ElMessage.success('镜像删除成功')
    deleteVisible.value = false
    await router.replace(listLocation.value)
  } catch (error) {
    if (!signal.aborted) deleteError.value = errorText(error)
  } finally { if (signal === lifetime.signal) deleting.value = false }
}
async function exportImage() {
  if (busy.value || !image.value) return
  exporting.value = true
  actionError.value = ''
  const signal = lifetime.signal
  const target: LocalImage = { ...image.value, Created: Date.parse(image.value.Created) / 1000 }
  try {
    const archive = await localImagesApi.export([target], signal)
    if (signal.aborted) return
    const url = URL.createObjectURL(archive)
    const link = document.createElement('a')
    link.href = url
    link.download = imageExportName([target])
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    ElMessage.success('镜像归档已生成')
  } catch (error) {
    if (!signal.aborted) actionError.value = errorText(error)
  } finally { if (signal === lifetime.signal) exporting.value = false }
}

watch(imageId, () => {
  lifetime.abort()
  lifetime = new AbortController()
  tagging.value = false
  exporting.value = false
  deleting.value = false
  deleteVisible.value = false
  image.value = null
  newTag.value = ''
  tagError.value = ''
  actionError.value = ''
  expandedLayers.value = new Set()
  void loadImage()
}, { immediate: true })
onBeforeUnmount(() => { readController?.abort(); lifetime.abort() })
</script>

<style scoped>
.image-detail-page { --el-color-primary: #33a7d5; width: 100%; min-width: 0; color: #e7e7e7; font-size: 13px; }
.detail-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.detail-heading h1 { font-size: 18px; font-weight: 600; line-height: 28px; margin: 0; }
.detail-heading-actions { display: flex; align-items: center; gap: 16px; }
.detail-heading-actions a { color: #35acd7; text-decoration: none; font-size: 12px; }
.detail-heading-actions a:hover { text-decoration: underline; }
.detail-heading-actions .el-button { color: #ccc; height: 28px; }
.detail-card { background: #202020; border: 1px solid #303030; border-radius: 10px; overflow: hidden; margin-bottom: 16px; }
.detail-card h2 { display: flex; align-items: center; gap: 10px; padding: 16px 20px; font-size: 15px; line-height: 24px; font-weight: 600; margin: 0; }
.detail-icon { display: grid; place-items: center; width: 32px; height: 32px; border-radius: 50%; background: #303844; color: #c6d4e5; flex-shrink: 0; }
.detail-icon .portainer-icon { width: 18px; height: 18px; }
.detail-card-body { padding: 4px 20px 20px; }
.tag-list { display: flex; flex-wrap: wrap; gap: 8px; }
.detail-tag { padding: 4px 10px; background: #075474; color: #d4ecf6; border-radius: 5px; overflow-wrap: anywhere; max-width: 100%; font-size: 12px; }
.tag-empty { background: #3a4049; color: #ccc; }
.tag-image-form { display: grid; grid-template-columns: 108px minmax(0, 1fr) auto; align-items: center; gap: 8px 14px; padding: 4px 20px 20px; }
.tag-image-form label { font-weight: 500; }
.tag-image-form p { grid-column: 2 / -1; font-size: 12px; color: #aaa; line-height: 1.7; }
.tag-image-form :deep(.el-input__wrapper) { background: #171717; border-color: #3b3b3b; border-radius: 4px; }
.tag-image-form :deep(.el-input__wrapper.is-focus) { border-color: #33a7d5; }
.tag-image-form :deep(.el-input__inner) { font-size: 13px; }
.image-detail-page :deep(.el-button) { font-size: 12px; }
.image-detail-page :deep(.el-button > span) { gap: 6px; }
.tag-image-form .el-button, .image-detail-actions .el-button { color: #222; background: #f2f2f2; border-color: #f2f2f2; margin: 0; }
.tag-image-form .el-button.is-disabled { opacity: .45; }
.tag-error { margin: 0 20px 20px; width: auto; }
.metadata-table { width: 100%; border-collapse: collapse; table-layout: fixed; }
.metadata-table > tbody > tr > th { width: 142px; padding: 13px 20px; text-align: left; font-weight: 500; vertical-align: top; }
.metadata-table > tbody > tr > td { padding: 13px 20px 13px 0; }
.metadata-table > tbody > tr > th, .metadata-table > tbody > tr > td { border-top: 1px solid #303030; line-height: 1.7; overflow-wrap: anywhere; }
.metadata-table code { font-size: 12px; white-space: pre-wrap; overflow-wrap: anywhere; color: #dedede; }
.id-and-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.full-image-id { user-select: all; }
.image-detail-actions { display: flex; flex-wrap: wrap; gap: 6px; }
.image-detail-actions .el-button { height: 28px; padding: 0 9px; }
.image-detail-actions .detail-remove { color: #fff; background: #a9333c; border-color: #a9333c; }
.key-value-table { border-collapse: collapse; width: 100%; table-layout: fixed; font-size: 12px; }
.key-value-table th, .key-value-table td { border-bottom: 1px solid #343434; padding: 9px 12px; text-align: left; font-weight: 400; vertical-align: top; overflow-wrap: anywhere; }
.key-value-table tr:last-child th, .key-value-table tr:last-child td { border-bottom: 0; }
.key-value-table th { width: 32%; color: #c9c9c9; }
.key-value-table tr:nth-child(odd) { background: #262626; }
.key-value-table tr:nth-child(even) { background: #222; }
.detail-muted { color: #999; font-size: 12px; }
.usage-description { margin-left: 10px; }
.usage-badge { display: inline-block; border-radius: 10px; padding: 1px 8px; font-size: 11px; font-weight: 500; line-height: 18px; }
.unused { background: #923a1e; color: #ffe1ce; }
.in-use { background: #234637; color: #b8e3ce; }
.digest-value { font-size: 12px; color: #bfcad0; }
.exposed-ports { display: flex; flex-wrap: wrap; gap: 12px; }
.layer-count { margin-left: auto; font-size: 12px; font-weight: 400; color: #999; }
.layers-table-scroll { overflow-x: auto; }
.layers-table { width: 100%; border-collapse: collapse; table-layout: fixed; font-size: 12px; }
.layers-table th { text-align: left; background: #303030; font-weight: 500; padding: 10px 20px; }
.layers-table th:first-child { width: 96px; }
.layers-table th:nth-child(2) { width: 120px; }
.layers-table th button { background: none; border: 0; padding: 0; display: inline-flex; align-items: center; gap: 5px; white-space: nowrap; color: inherit; font: inherit; cursor: pointer; }
.layers-table th .portainer-icon { width: 13px; height: 13px; color: #aaa; }
.layers-table td { border-bottom: 1px solid #303030; padding: 11px 20px; vertical-align: top; line-height: 1.7; }
.layers-table td:nth-child(2) { white-space: nowrap; }
.layers-table tr:last-child td { border-bottom: 0; }
.layer-command { display: block; white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.7; font-size: 12px; color: #d6d6d6; }
.layer-command.collapsed { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.expand-command { border: 0; padding: 2px 0; background: none; color: #35acd7; font-size: 12px; cursor: pointer; }
.detail-loading { padding: 40px 20px; color: #999; text-align: center; }
.history-error { padding: 8px 20px 20px; color: #f3b3b9; font-size: 12px; overflow-wrap: anywhere; }
@media (max-width: 760px) {
  .detail-card h2 { padding: 12px 16px; }
  .tag-image-form { padding: 4px 16px 16px; grid-template-columns: minmax(0, 1fr) auto; }
  .tag-image-form label { grid-column: 1 / -1; }
  .tag-image-form p { grid-column: 1 / -1; }
  .metadata-table > tbody > tr > th { width: 112px; padding-left: 16px; padding-right: 12px; }
  .metadata-table > tbody > tr > td { padding-right: 16px; }
  .key-value-table th { width: 40%; }
  .key-value-table th, .key-value-table td { padding: 7px; }
  .layers-table th, .layers-table td { padding-left: 12px; padding-right: 12px; }
  .layers-table th:first-child { width: 80px; }
  .layers-table th:nth-child(2) { width: 84px; }
  .layer-count { font-size: 11px; }
}
</style>

<style>
.image-detail-dialog { border-radius: 10px; }
.image-detail-dialog .el-dialog__body { padding-top: 20px; padding-bottom: 20px; }
.image-detail-dialog .el-dialog__footer { padding-top: 16px; }
.image-detail-dialog .delete-image-id { display: block; padding: 12px; margin: 16px 0; background: #191919; font-size: 12px; overflow-wrap: anywhere; }
.image-detail-dialog .detail-dialog-help { color: #999; font-size: 12px; margin: 8px 0 16px; }
</style>
