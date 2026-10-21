<template>
  <div class="images-page">
    <header class="images-heading">
      <h1>本地镜像</h1>
      <el-button text circle aria-label="刷新镜像列表" title="刷新镜像列表" :disabled="busy || loading" @click="loadImages">
        <PortainerIcon name="refresh" :class="{ spinning: loading }" />
      </el-button>
    </header>

    <section class="images-card pull-card" aria-labelledby="pull-title">
      <h2 id="pull-title"><span class="section-icon"><PortainerIcon name="download" /></span>拉取镜像</h2>
      <form class="pull-form" @submit.prevent="pullImage">
        <label for="pull-image-name">镜像</label>
        <el-input id="pull-image-name" v-model="pullName" placeholder="例如 nginx:latest 或 registry.example.com/app:1.0" :disabled="busy" autocomplete="off" />
        <el-button native-type="submit" class="pull-button" :loading="pulling" :disabled="busy || loading || !pullName.trim()">
          <PortainerIcon v-if="!pulling" name="download" />{{ pulling ? '拉取中…' : '拉取镜像' }}
        </el-button>
        <p class="pull-help" :class="{ 'transfer-progress': pulling }" role="status">
          {{ pulling ? pullProgress : '支持 Docker Hub 或完整仓库地址；未填写标签时使用 latest。' }}
        </p>
      </form>
      <el-alert v-if="pullError" class="pull-error" title="拉取失败" :description="pullError" type="error" show-icon @close="pullError = ''" />
    </section>

    <section class="images-card image-list" aria-labelledby="images-title">
      <header class="images-toolbar">
        <h2 id="images-title"><span class="section-icon"><PortainerIcon name="list" /></span>镜像列表</h2>
        <div class="images-actions">
          <el-input v-model="search" class="image-search" placeholder="搜索 ID 或标签…" aria-label="搜索镜像 ID 或标签" clearable :disabled="busy">
            <template #prefix><PortainerIcon name="search" /></template>
          </el-input>
          <div class="remove-actions">
            <el-button class="remove-button" :disabled="!selectedImages.length || busy || loading" @click="openDelete('normal')"><PortainerIcon name="trash" />删除</el-button>
            <el-dropdown trigger="click" :disabled="!selectedImages.length || busy || loading" @command="openDelete">
              <el-button class="remove-button remove-options" aria-label="更多删除选项" :disabled="!selectedImages.length || busy || loading"><PortainerIcon name="chevron-down" /></el-button>
              <template #dropdown><el-dropdown-menu><el-dropdown-item command="force">强制删除所选镜像</el-dropdown-item></el-dropdown-menu></template>
            </el-dropdown>
          </div>
          <div class="archive-actions">
            <el-button :disabled="busy || loading" @click="openImport"><PortainerIcon name="upload" />导入</el-button>
            <el-button :disabled="!selectedImages.length || busy || loading" :loading="exporting" @click="exportImages">
              <PortainerIcon v-if="!exporting" name="download" />{{ exporting ? '导出中…' : '导出' }}
            </el-button>
          </div>
        </div>
      </header>

      <div v-if="listError || exportError" class="list-error" role="alert">
        <span>{{ listError || exportError }}</span>
        <el-button v-if="listError" link :disabled="busy || loading" @click="loadImages">重试</el-button>
        <el-button v-else link @click="exportError = ''">关闭</el-button>
      </div>
      <div v-if="usageError && !listError" class="usage-warning" role="status">
        <span>暂时无法读取容器使用状态，未使用标记已隐藏。</span>
        <el-button link :disabled="busy || loading" @click="loadImages">重试</el-button>
      </div>

      <div ref="tableScroll" v-loading="loading" class="image-table-scroll" aria-label="镜像列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="image-table" aria-label="本地镜像列表">
          <colgroup><col class="selection-column" /><col class="id-column" /><col /><col class="size-column" /><col class="date-column" /></colgroup>
          <thead>
            <tr>
              <th scope="col" class="selection-cell">
                <el-checkbox :model-value="allPageSelected" :indeterminate="somePageSelected && !allPageSelected" :disabled="!pageImages.length || busy || loading" aria-label="选择当前页全部镜像" @change="togglePageSelection" />
              </th>
              <th v-for="column in columns" :key="column.key" scope="col" :aria-sort="sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none'">
                <button class="column-sort" :disabled="busy" @click="changeSort(column.key)">
                  {{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" />
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="image in pageImages" :key="image.Id" :class="{ selected: selectedIds.has(image.Id) }">
              <td class="selection-cell">
                <el-checkbox :model-value="selectedIds.has(image.Id)" :disabled="busy || loading" :aria-label="`选择镜像 ${shortId(image.Id)}`" @change="toggleSelection(image.Id, Boolean($event))" />
              </td>
              <td><div class="image-identity">
                <router-link class="image-id" :title="image.Id" :to="{ name: 'ImageDetail', params: { id: image.Id }, query: listQuery }">{{ shortId(image.Id) }}</router-link>
                <span v-if="imageUseCount(image.Id, containers) === 0" class="image-unused" title="没有容器引用此镜像（包含已停止的容器）">未使用</span>
              </div></td>
              <td><div class="image-tags"><span v-for="tag in imageTags(image)" :key="tag" class="image-tag">{{ tag }}</span><span v-if="!imageTags(image).length" class="image-tag untagged">未标记</span></div></td>
              <td class="image-size">{{ formatImageSize(image.Size) }}</td>
              <td class="image-created">{{ formatImageDate(image.Created) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-if="!pageImages.length && !loading" class="image-empty" role="status">
          <PortainerIcon :name="search.trim() ? 'search' : 'list'" />
          <p>{{ listError ? '暂时无法加载镜像列表' : search.trim() ? '没有找到匹配的镜像' : '暂无本地镜像' }}</p>
          <el-button v-if="search.trim()" link @click="search = ''">清除搜索</el-button>
          <span v-else-if="!listError">拉取或导入镜像后，将显示在这里</span>
        </div>
      </div>

      <footer class="images-pagination">
        <div class="image-count" role="status">
          <span>共 {{ filteredImages.length }} 个镜像</span>
          <template v-if="selectedImages.length"><span class="selected-count">已选 {{ selectedImages.length }} 项</span><el-button link :disabled="busy" @click="selectedIds = new Set()">清除选择</el-button></template>
        </div>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredImages.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" />
      </footer>
    </section>

    <el-dialog v-model="importVisible" title="导入镜像" width="560px" class="local-images-dialog" append-to-body :close-on-click-modal="!importing" :close-on-press-escape="!importing" :show-close="!importing">
      <p class="dialog-description">选择通过镜像导出命令生成的归档文件，导入到本机 Docker。</p>
      <label class="image-upload-zone" :class="{ 'is-disabled': importing }" @dragover.prevent @drop.prevent="dropFile">
        <PortainerIcon name="upload" />
        <strong>{{ importFile ? importFile.name : '点击选择或拖拽镜像文件' }}</strong>
        <span>{{ importFile ? `${formatImageSize(importFile.size)} · 点击可重新选择` : '支持 .tar、.tar.gz、.tgz、.tar.bz2、.tar.xz、.tar.zst' }}</span>
        <input type="file" accept=".tar,.gz,.tgz,.bz2,.xz,.zst" aria-label="选择镜像归档文件" :disabled="importing" @change="chooseFile" />
      </label>
      <p v-if="importing" class="import-progress" role="status">{{ importProgress }}</p>
      <el-alert v-if="importError" title="导入失败" :description="importError" type="error" :closable="false" show-icon />
      <template #footer>
        <el-button :disabled="importing" @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="importing" :disabled="!importFile" @click="importImage">{{ importing ? '导入中…' : '开始导入' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="deleteVisible" :title="forceDelete ? '强制删除镜像' : '删除镜像'" width="560px" class="local-images-dialog" append-to-body :close-on-click-modal="!deleting" :close-on-press-escape="!deleting" :show-close="!deleting">
      <p class="dialog-description">{{ forceDelete ? '强制删除会尝试移除所选镜像及其所有标签，此操作无法撤销。' : `确定删除以下 ${deleteTargets.length} 个镜像？此操作无法撤销。` }}</p>
      <p v-if="forceDelete" class="dialog-description">被保留版本引用的镜像仍无法删除。</p>
      <ul class="delete-targets"><li v-for="image in deleteTargets" :key="image.Id"><strong>{{ imageTags(image)[0] || '未标记镜像' }}</strong><code>{{ shortId(image.Id) }}</code></li></ul>
      <div v-if="deleteFailures.length" class="delete-failures" role="alert">
        <strong>{{ deleteFailures.length }} 个镜像删除失败，可重试或取消</strong>
        <p v-for="failure in deleteFailures" :key="failure.id">{{ shortId(failure.id) }}：{{ failure.message }}</p>
      </div>
      <p v-if="deleting" class="import-progress" role="status">{{ deleteProgress }}</p>
      <template #footer>
        <el-button :disabled="deleting" @click="deleteVisible = false">取消</el-button>
        <el-button type="danger" :loading="deleting" @click="removeImages">{{ deleteFailures.length ? '重试删除' : forceDelete ? '确认强制删除' : '确认删除' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { localImagesApi } from '@/api/localImages'
import { filterImages, formatImageDate, formatImageSize, imageArchiveError, imageExportName, imageListQuery, imageTags, imageUseCount, readImageListState, sortImages, type ImageContainer, type ImageSortKey, type LocalImage } from '@/utils/localImages'

const initialState = readImageListState(useRoute().query)
const images = ref<LocalImage[]>([])
const containers = ref<ImageContainer[] | null>(null)
const usageError = ref('')
const loading = ref(false)
const listError = ref('')
const search = ref(initialState.search)
const selectedIds = ref(new Set<string>())
const page = ref(initialState.page)
const pageSize = ref(initialState.size)
const sortKey = ref<ImageSortKey>(initialState.sort)
const descending = ref(initialState.descending)
const listQuery = computed(() => imageListQuery({ search: search.value, page: page.value, size: pageSize.value, sort: sortKey.value, descending: descending.value }))
const tableScroll = ref<HTMLElement>()
const lifetime = new AbortController()
const columns: { key: ImageSortKey; label: string }[] = [
  { key: 'Id', label: 'ID' }, { key: 'RepoTags', label: '标签' },
  { key: 'Size', label: '大小' }, { key: 'Created', label: '创建时间' },
]
const pulling = ref(false)
const pullName = ref('')
const pullProgress = ref('')
const pullError = ref('')
const importVisible = ref(false)
const importFile = ref<File>()
const importing = ref(false)
const importProgress = ref('')
const importError = ref('')
const exporting = ref(false)
const exportError = ref('')
const deleteVisible = ref(false)
const deleting = ref(false)
const forceDelete = ref(false)
const deleteTargets = ref<LocalImage[]>([])
const deleteFailures = ref<{ id: string; message: string }[]>([])
const deleteProgress = ref('')
const busy = computed(() => pulling.value || importing.value || exporting.value || deleting.value)
const filteredImages = computed(() => sortImages(filterImages(images.value, search.value), sortKey.value, descending.value))
const pageImages = computed(() => filteredImages.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const selectedImages = computed(() => images.value.filter(image => selectedIds.value.has(image.Id)))
const allPageSelected = computed(() => pageImages.value.length > 0 && pageImages.value.every(image => selectedIds.value.has(image.Id)))
const somePageSelected = computed(() => pageImages.value.some(image => selectedIds.value.has(image.Id)))
const shortId = (id: string) => id.length > 36 ? `${id.slice(0, 31)}…` : id
const errorMessage = (error: unknown) => error instanceof Error ? error.message : '操作失败，请重试'

watch(search, () => { page.value = 1; selectedIds.value = new Set() })
watch(pageSize, () => { page.value = 1 })
watch(() => filteredImages.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch([page, pageSize, sortKey, descending, search], () => { tableScroll.value?.scrollTo({ top: 0 }) })

function toggleSelection(id: string, checked: boolean) {
  const next = new Set(selectedIds.value)
  if (checked) next.add(id)
  else next.delete(id)
  selectedIds.value = next
}
function togglePageSelection(checked: string | number | boolean) {
  const next = new Set(selectedIds.value)
  for (const image of pageImages.value) {
    if (checked) next.add(image.Id)
    else next.delete(image.Id)
  }
  selectedIds.value = next
}
function changeSort(key: ImageSortKey) {
  descending.value = sortKey.value === key ? !descending.value : key === 'Created' || key === 'Size'
  sortKey.value = key
  page.value = 1
}
async function loadImages() {
  if (loading.value || lifetime.signal.aborted) return
  loading.value = true
  listError.value = ''
  usageError.value = ''
  containers.value = null
  try {
    const [imageResult, containerResult] = await Promise.allSettled([
      localImagesApi.list(lifetime.signal), localImagesApi.containers(lifetime.signal),
    ])
    if (lifetime.signal.aborted) return
    if (imageResult.status === 'fulfilled') {
      images.value = imageResult.value
      const existingIds = new Set(images.value.map(image => image.Id))
      selectedIds.value = new Set([...selectedIds.value].filter(id => existingIds.has(id)))
    } else listError.value = `镜像列表加载失败：${errorMessage(imageResult.reason)}`
    if (containerResult.status === 'fulfilled') containers.value = containerResult.value
    else usageError.value = errorMessage(containerResult.reason)
  } finally { loading.value = false }
}
async function pullImage() {
  if (busy.value || loading.value || !pullName.value.trim()) return
  pulling.value = true
  pullError.value = ''
  pullProgress.value = '正在连接镜像仓库…'
  try {
    await localImagesApi.pull(pullName.value, message => { pullProgress.value = message }, lifetime.signal)
    ElMessage.success('镜像拉取成功')
    pullName.value = ''
  } catch (error) {
    if (!lifetime.signal.aborted) pullError.value = errorMessage(error)
  } finally {
    pulling.value = false
    await loadImages()
  }
}
function openImport() {
  importFile.value = undefined
  importError.value = ''
  importProgress.value = ''
  importVisible.value = true
}
function setImportFile(files: File[]) {
  if (importing.value) return
  importError.value = ''
  if (files.length !== 1) { importFile.value = undefined; importError.value = '请一次选择一个镜像归档文件'; return }
  const file = files[0]
  importError.value = imageArchiveError(file)
  importFile.value = importError.value ? undefined : file
}
function chooseFile(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files?.length) setImportFile(Array.from(input.files))
  input.value = ''
}
function dropFile(event: DragEvent) {
  setImportFile(Array.from(event.dataTransfer?.files || []))
}
async function importImage() {
  if (!importFile.value || busy.value) return
  importing.value = true
  importError.value = ''
  importProgress.value = '正在上传并导入镜像，请稍候…'
  try {
    await localImagesApi.import(importFile.value, message => { importProgress.value = message }, lifetime.signal)
    ElMessage.success('镜像导入成功')
    importVisible.value = false
  } catch (error) {
    if (!lifetime.signal.aborted) importError.value = errorMessage(error)
  } finally {
    importing.value = false
    await loadImages()
  }
}
function openDelete(mode: string) {
  if (busy.value || loading.value || !selectedImages.value.length) return
  forceDelete.value = mode === 'force'
  deleteTargets.value = [...selectedImages.value]
  deleteFailures.value = []
  deleteVisible.value = true
}
async function removeImages() {
  if (busy.value) return
  deleting.value = true
  deleteFailures.value = []
  const targets = [...deleteTargets.value]
  const failed: LocalImage[] = []
  let removed = 0
  try {
    for (const [index, image] of targets.entries()) {
      if (lifetime.signal.aborted) break
      deleteProgress.value = `正在删除 ${index + 1} / ${targets.length} 个镜像…`
      try {
        await localImagesApi.remove(image.Id, forceDelete.value, lifetime.signal)
        toggleSelection(image.Id, false)
        removed++
      } catch (error) {
        if (lifetime.signal.aborted) break
        failed.push(image)
        deleteFailures.value.push({ id: image.Id, message: errorMessage(error) })
      }
    }
    if (lifetime.signal.aborted) return
    if (removed) ElMessage.success(`已删除 ${removed} 个镜像`)
    if (!failed.length) deleteVisible.value = false
    else deleteTargets.value = failed
  } finally {
    deleting.value = false
    await loadImages()
  }
}
async function exportImages() {
  if (busy.value || loading.value || !selectedImages.value.length) return
  exporting.value = true
  exportError.value = ''
  const targets = [...selectedImages.value]
  try {
    const archive = await localImagesApi.export(targets, lifetime.signal)
    const url = URL.createObjectURL(archive)
    const link = document.createElement('a')
    link.href = url
    link.download = imageExportName(targets)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
    ElMessage.success(`已生成 ${targets.length} 个镜像的归档文件`)
  } catch (error) {
    if (!lifetime.signal.aborted) exportError.value = `镜像导出失败：${errorMessage(error)}`
  } finally { exporting.value = false }
}

onMounted(loadImages)
onBeforeUnmount(() => lifetime.abort())
</script>

<style scoped>
.images-page {
  --el-color-primary: #33a7d5;
  --el-fill-color-blank: #202020;
  --el-border-color: #414141;
  --el-text-color-regular: #d6d6d6;
  --el-text-color-placeholder: #999;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  min-height: 0;
  gap: 16px;
  color: #ebebeb;
  font-size: 13px;
}
.images-heading { display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; }
.images-heading h1 { margin: 0; font-size: 18px; line-height: 28px; font-weight: 600; }
.images-heading .el-button { color: #ccc; height: 28px; }
.images-card { background: #202020; border: 1px solid #303030; border-radius: 10px; overflow: hidden; }
h2 { display: flex; align-items: center; gap: 10px; margin: 0; font-size: 15px; line-height: 24px; font-weight: 600; white-space: nowrap; }
.section-icon { display: grid; place-items: center; width: 32px; height: 32px; flex-shrink: 0; border-radius: 50%; background: #303844; color: #c6d4e5; }
.section-icon .portainer-icon { width: 18px; height: 18px; }
.pull-card { padding: 16px 20px; flex-shrink: 0; }
.pull-card h2 { margin-bottom: 14px; }
.pull-form { display: grid; grid-template-columns: 68px minmax(0, 1fr) auto; align-items: center; gap: 8px 14px; }
.pull-form label { font-weight: 500; }
.pull-form .el-input { min-width: 0; }
.pull-help { grid-column: 2 / -1; color: #aaa; font-size: 12px; line-height: 18px; overflow-wrap: anywhere; }
.transfer-progress { color: #a3d7eb; max-height: 36px; overflow: hidden; }
.pull-error { margin-top: 12px; }
.pull-error :deep(.el-alert__description) { overflow-wrap: anywhere; max-height: 54px; overflow-y: auto; }
.images-page :deep(.el-input__wrapper) { background: #171717; border-color: #3b3b3b; border-radius: 4px; }
.images-page :deep(.el-input__wrapper:hover) { border-color: #666; }
.images-page :deep(.el-input__wrapper.is-focus) { border-color: #33a7d5; }
.images-page :deep(.el-input__inner) { font-size: 13px; }
.images-page :deep(.el-button) { font-size: 12px; }
.images-page :deep(.el-button > span) { gap: 6px; }
.pull-button.el-button { color: #222; background: #f2f2f2; border-color: #f2f2f2; font-weight: 500; }
.pull-button.el-button:not(.is-disabled):hover { color: #111; background: #d8d8d8; border-color: #d8d8d8; }
.pull-button.el-button.is-disabled { color: #777; background: #bdbdbd; border-color: #bdbdbd; }
.image-list { display: flex; flex-direction: column; flex: 1; min-height: 0; }
.images-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px 20px; padding: 16px 20px; flex-shrink: 0; }
.images-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.image-search { width: 230px; }
.images-actions :deep(.el-button) { margin: 0; height: 30px; padding: 0 10px; color: #d6d6d6; background: transparent; border-color: #555; }
.images-actions :deep(.el-button:not(.is-disabled):hover) { background: #323232; border-color: #888; color: #fff; }
.images-actions :deep(.el-button.is-disabled) { opacity: .4; }
.images-actions .remove-button { background: #482427; border-color: #6c3439; color: #f1a7ac; }
.images-actions .remove-button:not(.is-disabled):hover { background: #673035; border-color: #9a4d54; color: #ffd6d9; }
.remove-actions { display: flex; }
.remove-actions > .remove-button { border-radius: 5px 0 0 5px; }
.remove-actions .remove-options { margin-left: -1px; padding: 0 5px; border-radius: 0 5px 5px 0; }
.archive-actions { display: flex; }
.archive-actions :deep(.el-button:first-child) { border-radius: 5px 0 0 5px; }
.archive-actions :deep(.el-button:last-child) { border-radius: 0 5px 5px 0; margin-left: -1px; }
.image-table-scroll { flex: 1; min-height: 0; overflow: auto; position: relative; outline-offset: -2px; }
.image-table-scroll:focus-visible { outline: 2px solid #33a7d5; }
.image-table { width: 100%; min-width: 720px; border-spacing: 0; table-layout: fixed; text-align: left; }
.selection-column { width: 50px; }
.id-column { width: 28%; }
.size-column { width: 110px; }
.date-column { width: 186px; }
.image-table thead { position: sticky; top: 0; z-index: 2; background: #303030; }
.image-table th { height: 42px; font-size: 12px; font-weight: 500; padding: 0 12px; }
.column-sort { display: inline-flex; align-items: center; gap: 5px; padding: 8px 0; border: 0; background: transparent; color: #d5d5d5; font: inherit; cursor: pointer; }
.column-sort .portainer-icon { width: 13px; height: 13px; color: #777; }
.column-sort .sort-active { color: #ddd; }
.image-table th[aria-sort="ascending"] .sort-active :deep(path:first-child),
.image-table th[aria-sort="descending"] .sort-active :deep(path:last-child) { opacity: .3; }
.image-table td { height: 48px; padding: 10px 12px; border-bottom: 1px solid #303030; line-height: 20px; vertical-align: middle; }
.image-table tr:hover td { background: #282828; }
.image-table tr.selected td { background: #24343c; }
.image-table .selection-cell { padding-left: 20px; padding-right: 8px; }
.selection-cell :deep(.el-checkbox) { height: 20px; vertical-align: middle; }
.selection-cell :deep(.el-checkbox__inner) { width: 15px; height: 15px; border-radius: 4px; background: #eee; border-color: #bbb; }
.selection-cell :deep(.is-checked .el-checkbox__inner), .selection-cell :deep(.is-indeterminate .el-checkbox__inner) { background: #1685b3; border-color: #1685b3; }
.image-identity { display: flex; align-items: center; gap: 8px; min-width: 0; }
.image-id { display: block; flex: 0 1 auto; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #35acd7; text-decoration: none; font-variant-numeric: tabular-nums; }
.image-id:hover { color: #76cdec; text-decoration: underline; }
.image-unused { flex-shrink: 0; border-radius: 10px; padding: 0 7px; background: #923a1e; color: #ffe1ce; font-size: 11px; font-weight: 500; line-height: 19px; }
.image-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.image-tag { display: inline-block; max-width: 100%; padding: 1px 7px; border-radius: 12px; background: #075474; color: #d4ecf6; font-size: 11px; line-height: 17px; overflow-wrap: anywhere; }
.image-tag.untagged { background: #3a4049; color: #d0d3d8; }
.image-size, .image-created { white-space: nowrap; font-variant-numeric: tabular-nums; font-size: 12px; }
.image-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; min-height: 150px; height: calc(100% - 42px); color: #999; }
.image-empty > .portainer-icon { width: 30px; height: 30px; }
.image-empty > span { font-size: 12px; }
.images-pagination { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px 16px; padding: 12px 20px; min-height: 58px; border-top: 1px solid #303030; flex-shrink: 0; }
.image-count { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; color: #aaa; font-size: 12px; }
.selected-count { color: #91cde4; }
.images-pagination :deep(.el-pagination) { --el-pagination-bg-color: transparent; --el-pagination-button-disabled-bg-color: transparent; --el-pagination-hover-color: #42b6df; flex-wrap: wrap; font-size: 12px; }
.images-pagination :deep(.el-pagination__sizes) { margin-right: 8px; }
.images-pagination :deep(.el-select) { width: 110px; }
.images-pagination :deep(.el-select__wrapper) { background: #171717; box-shadow: 0 0 0 1px #444 inset; min-height: 30px; }
.list-error { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 20px; background: #3c2528; color: #ffc3c8; overflow-wrap: anywhere; flex-shrink: 0; }
.list-error > span { max-height: 54px; overflow-y: auto; }
.usage-warning { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-shrink: 0; padding: 8px 20px; background: #3d3323; color: #e9cc9d; font-size: 12px; }
.spinning { animation: image-spin 1s linear infinite; }
@keyframes image-spin { to { transform: rotate(360deg); } }
@media (max-width: 1100px) {
  .images-toolbar { align-items: flex-start; gap: 10px; padding: 12px 16px; }
  .images-actions { flex: 1; justify-content: flex-end; }
  .image-search { width: 210px; }
  .images-pagination { padding: 10px 16px; }
}
@media (max-width: 760px) {
  .images-page { gap: 12px; }
  .pull-card { padding: 12px 16px; }
  .pull-form { grid-template-columns: minmax(0, 1fr) auto; gap: 8px; }
  .pull-form label { display: none; }
  .pull-help { grid-column: 1 / -1; }
  .images-toolbar h2 { width: 100%; }
  .images-actions { justify-content: flex-start; width: 100%; }
  .image-search { flex: 1; min-width: 160px; }
  .images-pagination { gap: 4px; }
  .image-count { width: 100%; }
  .images-pagination :deep(.el-pagination) { margin-left: auto; }
}
@media (max-height: 700px) {
  .images-page { gap: 10px; }
  .pull-card { padding-top: 10px; padding-bottom: 10px; }
  .pull-card h2 { margin-bottom: 8px; }
  .images-toolbar { padding-top: 10px; padding-bottom: 10px; }
}
</style>

<style>
.local-images-dialog { --el-dialog-bg-color: #242424; --el-border-color: #444; border-radius: 10px; }
.local-images-dialog .el-dialog__body { padding-top: 20px; padding-bottom: 20px; }
.local-images-dialog .el-dialog__footer { padding-top: 16px; }
.local-images-dialog .dialog-description { margin-bottom: 16px; color: #bcbcbc; font-size: 13px; line-height: 1.7; }
.image-upload-zone { position: relative; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; min-height: 180px; padding: 24px; margin-bottom: 16px; border: 1px dashed #666; border-radius: 8px; background: #1b1b1b; text-align: center; cursor: pointer; }
.image-upload-zone:hover, .image-upload-zone:focus-within { border-color: #55b2d3; }
.image-upload-zone > .portainer-icon { width: 30px; height: 30px; color: #a9cbd8; }
.image-upload-zone strong { max-width: 100%; overflow-wrap: anywhere; font-size: 14px; }
.image-upload-zone span { font-size: 12px; line-height: 1.6; color: #999; }
.image-upload-zone input { position: absolute; inset: 0; opacity: 0; width: 100%; height: 100%; cursor: pointer; }
.image-upload-zone.is-disabled { opacity: .6; cursor: wait; }
.import-progress { color: #9bd7ee; margin: 12px 0; font-size: 13px; line-height: 1.6; overflow-wrap: anywhere; }
.delete-targets { list-style: none; padding: 0; margin: 0 0 16px; max-height: 220px; overflow: auto; }
.delete-targets li { display: flex; flex-direction: column; gap: 4px; padding: 10px 12px; border-bottom: 1px solid #3b3b3b; background: #1b1b1b; }
.delete-targets strong { font-size: 13px; font-weight: 500; overflow-wrap: anywhere; }
.delete-targets code { color: #999; font-size: 12px; }
.delete-failures { max-height: 180px; overflow: auto; padding: 12px; background: #40272b; border-radius: 6px; color: #ffc0c7; font-size: 12px; line-height: 1.7; }
.delete-failures p { margin-top: 8px; overflow-wrap: anywhere; }
</style>
