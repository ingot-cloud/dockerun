<template>
  <div class="registry-resource registry-images-page">
    <WorkspaceHeading :title="`仓库镜像 · ${registryName || '镜像仓库'}`" :busy="deleting || confirming" :refreshable="false" />
    <section class="registry-card registry-list" aria-labelledby="registry-images-title">
      <header class="registries-toolbar">
        <h2 id="registry-images-title"><span class="registry-section-icon"><PortainerIcon name="box" /></span>镜像列表</h2>
        <div class="registries-actions"><el-input v-model="searchKeyword" class="registry-search" placeholder="搜索镜像名称…" aria-label="搜索仓库镜像" clearable :disabled="deleting || confirming"><template #prefix><PortainerIcon name="search" /></template></el-input><el-button :disabled="loading || deleting || confirming" @click="fetchImages"><PortainerIcon name="refresh" :class="{ spinning: loading }" />刷新</el-button></div>
      </header>
      <div v-if="listError" class="registry-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading || deleting || confirming" @click="fetchImages">重试</el-button></div>
      <div v-if="deleteError" class="registry-error" role="alert"><span>{{ deleteError }}</span><el-button link @click="deleteError = ''">关闭</el-button></div>
      <div v-if="partialTagFailures" class="registry-warning" role="status">本页 {{ partialTagFailures }} 个镜像的标签暂时无法读取。展开对应镜像可单独重试，其他镜像仍可正常浏览。</div>
      <div ref="tableScroll" v-loading="loading" class="registry-table-scroll" tabindex="0" aria-label="仓库镜像列表滚动区域" :aria-busy="loading">
        <table class="registry-table registry-image-table" aria-label="仓库镜像列表">
          <colgroup><col class="registry-expand-col" /><col /><col class="registry-tag-count-col" /><col class="registry-image-actions-col" /></colgroup>
          <thead><tr><th scope="col"><span class="sr-only">展开标签</span></th><th scope="col">镜像引用</th><th scope="col">标签</th><th scope="col">快捷操作</th></tr></thead>
          <tbody><template v-for="(image, index) in images" :key="image.name">
            <tr><td><button class="registry-expand-button" :aria-expanded="expanded.has(image.name)" :aria-controls="`registry-tags-${index}`" :aria-label="`${expanded.has(image.name) ? '收起' : '展开'} ${image.name} 的标签`" @click="toggleExpanded(image.name)"><PortainerIcon name="chevron-down" /></button></td><td><div class="registry-image-name"><PortainerIcon name="box" /><code>{{ image.name }}</code></div></td><td><el-button v-if="image.tags_error" link class="registry-tag-error" :title="image.tags_error" @click="toggleExpanded(image.name)">读取失败</el-button><span v-else class="registry-badge">{{ image.tags.length }}</span></td><td><el-button link aria-label="复制镜像名称" title="复制镜像名称" @click="copyImageName(image.name)"><el-icon><CopyDocument /></el-icon></el-button></td></tr>
            <tr v-if="expanded.has(image.name)" :id="`registry-tags-${index}`" class="registry-image-tags"><td colspan="4"><h3>镜像标签 <span v-if="!image.tags_error" class="registry-muted">（{{ image.tags.length }} 个标签）</span></h3><div v-if="image.tags_error" class="registry-error registry-tag-error-detail" role="alert"><span>标签读取失败：{{ image.tags_error }}</span><el-button link :loading="retryingTags.has(image.name)" :disabled="loading || deleting || confirming || retryingTags.has(image.name)" @click="retryTags(image)">重试标签</el-button></div><div v-else class="registry-tags"><el-tag v-for="tag in image.tags" :key="tag" size="small" :closable="!deleting && !confirming" @close="deleteImage(image.name, tag)">{{ tag }}</el-tag><span v-if="!image.tags.length" class="registry-muted">暂无标签</span></div></td></tr>
          </template></tbody>
        </table>
        <div v-if="!images.length && !loading" class="registry-empty" role="status"><PortainerIcon :name="searchKeyword.trim() ? 'search' : 'box'" /><p>{{ listError ? '暂时无法加载镜像列表' : searchKeyword.trim() ? '没有找到匹配的镜像' : '暂无镜像' }}</p><el-button v-if="searchKeyword.trim()" link @click="searchKeyword = ''">清除搜索</el-button><span v-else-if="!listError">此仓库中的镜像将显示在这里</span></div>
      </div>
      <footer class="registries-pagination"><div class="registry-count" role="status">共 {{ totalImages }} 个镜像</div><el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="totalImages" :pager-count="5" :disabled="loading || deleting || confirming" layout="sizes, prev, pager, next" /></footer>
    </section>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import { registryApi, type RegistryImage } from '@/api/registry'
import PortainerIcon from '@/components/PortainerIcon.vue'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import { registryError } from '@/utils/registries'
import '@/styles/registries.css'

const route = useRoute()
const registryId = computed(() => Number(route.params.id))
const loading = ref(false), deleting = ref(false), confirming = ref(false)
const images = ref<RegistryImage[]>([]), expanded = ref(new Set<string>())
const retryingTags = ref(new Set<string>())
const partialTagFailures = computed(() => images.value.filter(image => image.tags_error).length)
const searchKeyword = ref(''), registryName = ref(''), listError = ref(''), deleteError = ref('')
const currentPage = ref(1), pageSize = ref(10), totalImages = ref(0)
const tableScroll = ref<HTMLElement>()
const { copy } = useClipboard({ legacy: true })
let searchTimer: ReturnType<typeof setTimeout> | undefined
let controller: AbortController | undefined
let requestVersion = 0
let disposed = false
const tagControllers = new Map<string, AbortController>()

function cancelTagRequests() {
  for (const request of tagControllers.values()) request.abort()
  tagControllers.clear(); retryingTags.value = new Set()
}

async function fetchImages() {
  clearTimeout(searchTimer)
  cancelTagRequests()
  controller?.abort()
  const current = new AbortController(); controller = current
  const version = ++requestVersion
  const id = registryId.value
  loading.value = true; listError.value = ''
  try {
    const result = await registryApi.images.list(id, currentPage.value, pageSize.value, searchKeyword.value.trim(), current.signal)
    if (disposed || current.signal.aborted || version !== requestVersion) return
    totalImages.value = result.total || 0
    registryName.value = result.registry_name || registryName.value
    const maxPage = Math.max(1, Math.ceil(totalImages.value / pageSize.value))
    if (currentPage.value > maxPage) { currentPage.value = maxPage; return }
    images.value = (result.items || []).map(image => ({ ...image, tags: image.tags || [] }))
    const visible = new Set(images.value.map(image => image.name))
    expanded.value = new Set([...expanded.value].filter(name => visible.has(name)))
    tableScroll.value?.scrollTo({ top: 0 })
  } catch (error) {
    if (!disposed && !current.signal.aborted && version === requestVersion) { listError.value = `镜像列表加载失败：${registryError(error)}` }
  } finally { if (version === requestVersion) loading.value = false }
}
function toggleExpanded(name: string) { const next = new Set(expanded.value); if (next.has(name)) next.delete(name); else next.add(name); expanded.value = next }
async function retryTags(image: RegistryImage) {
  if (loading.value || deleting.value || confirming.value || retryingTags.value.has(image.name)) return
  const version = requestVersion, id = registryId.value
  const current = new AbortController()
  tagControllers.set(image.name, current); retryingTags.value.add(image.name)
  try {
    const result = await registryApi.images.getTags(id, image.name, current.signal)
    if (disposed || current.signal.aborted || version !== requestVersion || id !== registryId.value) return
    image.tags = result.tags || []; delete image.tags_error
  } catch (error) {
    if (!disposed && !current.signal.aborted && version === requestVersion && id === registryId.value) image.tags_error = registryError(error)
  } finally {
    if (tagControllers.get(image.name) === current) { tagControllers.delete(image.name); retryingTags.value.delete(image.name) }
  }
}
async function copyImageName(name: string) { try { await copy(name); ElMessage.success('镜像名称已复制') } catch { ElMessage.error('复制失败，请重试') } }
async function deleteImage(name: string, tag: string) {
  if (deleting.value || confirming.value || loading.value) return
  const id = registryId.value
  confirming.value = true
  try { await ElMessageBox.confirm(`确定删除镜像「${name}:${tag}」？仓库会删除该标签对应的镜像清单，指向同一清单的其他标签也可能受影响。`, '删除镜像', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }) }
  catch { return }
  finally { confirming.value = false }
  if (disposed || id !== registryId.value) return
  deleting.value = true; deleteError.value = ''
  try { await registryApi.images.delete(id, name, tag); if (!disposed && id === registryId.value) { ElMessage.success('镜像已删除'); await fetchImages() } }
  catch (error) { if (!disposed && id === registryId.value) deleteError.value = `删除失败：${registryError(error)}` }
  finally { deleting.value = false }
}
watch(searchKeyword, () => {
  cancelTagRequests()
  clearTimeout(searchTimer); controller?.abort(); requestVersion++; loading.value = false
  searchTimer = setTimeout(() => { if (currentPage.value !== 1) currentPage.value = 1; else void fetchImages() }, 350)
})
watch(pageSize, () => { if (currentPage.value !== 1) currentPage.value = 1; else void fetchImages() })
watch(currentPage, () => { void fetchImages() })
watch(registryId, async id => {
  cancelTagRequests()
  clearTimeout(searchTimer); controller?.abort(); requestVersion++
  images.value = []; totalImages.value = 0; registryName.value = ''; expanded.value = new Set(); deleteError.value = ''
  if (currentPage.value !== 1) currentPage.value = 1; else void fetchImages()
  try { const registry = await registryApi.get(id); if (!disposed && id === registryId.value) registryName.value = registry.name } catch { /* The list provides a retryable error. */ }
}, { immediate: true })
onBeforeUnmount(() => { disposed = true; requestVersion++; clearTimeout(searchTimer); controller?.abort(); cancelTagRequests() })
</script>
<style scoped>
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
</style>
