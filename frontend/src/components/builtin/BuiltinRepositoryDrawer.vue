<template>
  <el-drawer :model-value="modelValue" title="镜像标签与版本" size="820px" class="builtin-drawer" append-to-body :before-close="beforeClose" :close-on-click-modal="!busy" :close-on-press-escape="!busy" :show-close="!busy" @update:model-value="emit('update:modelValue', $event)">
    <div class="registry-resource builtin-repository">
      <div class="builtin-repository-heading"><div><span class="registry-muted">镜像名称</span><h2>{{ repository }}</h2></div><el-button text circle aria-label="刷新镜像标签" title="刷新标签" :disabled="tagsLoading || busy" @click="loadTags(true)"><PortainerIcon name="refresh" :class="{ spinning: tagsLoading }" /></el-button></div>
      <div v-if="readOnly" class="registry-warning" role="status">仓库正在只读维护，暂时无法删除镜像版本。</div>
      <div class="builtin-repository-body">
        <aside class="builtin-tag-sidebar" aria-label="镜像标签">
          <div class="builtin-tag-search"><el-input v-model="tagSearch" placeholder="搜索标签…" aria-label="搜索镜像标签" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input><span class="registry-muted">{{ filteredTags.length }} 个标签</span></div>
          <div v-if="tagsError" class="registry-error builtin-side-error" role="alert"><span>{{ tagsError }}</span><el-button link :disabled="tagsLoading || busy" @click="loadTags(true)">重试</el-button></div>
          <div v-loading="tagsLoading" class="builtin-tag-list" :aria-busy="tagsLoading">
            <button v-for="tag in filteredTags" :key="tag" :class="{ active: selectedTag === tag }" :aria-pressed="selectedTag === tag" :disabled="busy || tagsLoading" :title="tag" @click="loadManifest(tag)"><PortainerIcon name="tag" /><span>{{ tag }}</span></button>
            <p v-if="!filteredTags.length && !tagsLoading && !tagsError" class="builtin-tag-empty">{{ tagSearch ? '没有匹配的标签' : '暂无标签' }}</p>
          </div>
        </aside>
        <section v-loading="manifestLoading" class="builtin-manifest" aria-label="镜像版本详情" :aria-busy="manifestLoading">
          <div v-if="selectedTag" class="builtin-version-heading"><PortainerIcon name="tag" /><strong :title="selectedTag">{{ selectedTag }}</strong></div>
          <div v-if="manifestError" class="registry-error" role="alert"><span>版本读取失败：{{ manifestError }}</span><el-button link :disabled="manifestLoading || busy" @click="loadManifest(selectedTag)">重试</el-button></div>
          <template v-if="detail">
            <dl class="builtin-manifest-details">
              <div><dt>镜像摘要（Digest）</dt><dd><code>{{ detail.digest || '—' }}</code><el-button v-if="detail.digest" link aria-label="复制镜像摘要" @click="copyText(detail.digest, '镜像摘要')"><el-icon><CopyDocument /></el-icon></el-button></dd></div>
              <div><dt>清单类型</dt><dd><span class="registry-badge">{{ detail.manifest.manifests ? '多平台索引' : '镜像清单' }}</span></dd></div>
              <div><dt>内容大小</dt><dd>{{ manifestSize(detail) }}<span v-if="!detail.manifest.manifests" class="builtin-detail-note">镜像配置与压缩层的总大小</span></dd></div>
              <div><dt>关联发布版本</dt><dd>{{ detail.release_ids?.length ? detail.release_ids.map(id => `#${id}`).join('、') : '暂无关联' }}</dd></div>
            </dl>
            <section class="builtin-command"><header><h3>{{ selectedTag.startsWith('sha256:') ? '拉取此版本' : '拉取此标签' }}</h3><el-button link aria-label="复制标签拉取命令" @click="copyText(pullCommand, '拉取命令')"><el-icon><CopyDocument /></el-icon>复制</el-button></header><pre>{{ pullCommand }}</pre></section>
            <section v-if="detail.digest && !selectedTag.startsWith('sha256:')" class="builtin-command"><header><h3>按摘要拉取</h3><el-button link aria-label="复制摘要拉取命令" @click="copyText(digestCommand, '拉取命令')"><el-icon><CopyDocument /></el-icon>复制</el-button></header><pre>{{ digestCommand }}</pre></section>
            <section v-if="detail.manifest.manifests" class="builtin-platforms"><h3>镜像平台</h3><div v-for="item in detail.manifest.manifests" :key="item.digest"><span>{{ [item.platform?.os, item.platform?.architecture, item.platform?.variant].filter(Boolean).join('/') || '未指定平台' }}</span><code :title="item.digest">{{ item.digest }}</code></div></section>
            <details class="builtin-raw-manifest"><summary>查看原始镜像清单</summary><pre>{{ JSON.stringify(detail.manifest, null, 2) }}</pre></details>
          </template>
          <div v-else-if="!manifestLoading && !manifestError" class="registry-empty"><PortainerIcon name="tag" /><p>选择标签查看镜像版本</p></div>
        </section>
      </div>
      <div v-if="deleteError" class="registry-error" role="alert"><span>删除失败：{{ deleteError }}</span><el-button link :disabled="busy" @click="deleteError = ''">关闭</el-button></div>
      <footer class="builtin-repository-footer"><p>当前及保留的发布版本受保护。删除清单会影响指向同一摘要 的所有标签。</p><el-button v-if="admin" class="registry-remove" :loading="deleting" :disabled="!detail?.digest || tagsLoading || manifestLoading || busy || readOnly" @click="removeManifest"><PortainerIcon name="trash" />删除镜像版本</el-button><span v-else class="registry-muted">仅管理员可删除版本</span></footer>
    </div>
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import { builtinApi, type BuiltinManifest } from '@/api/builtin'
import { builtinError, imageReference, manifestSize } from '@/utils/builtin'
import PortainerIcon from '@/components/PortainerIcon.vue'
const props = defineProps<{ modelValue: boolean; repository: string; initialReference: string; host: string; admin: boolean; readOnly: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; removed: [] }>()
const tags = ref<string[]>([]), tagSearch = ref(''), selectedTag = ref(''), detail = ref<BuiltinManifest>()
const tagsLoading = ref(false), manifestLoading = ref(false), tagsError = ref(''), manifestError = ref(''), deleteError = ref('')
const confirming = ref(false), deleting = ref(false), busy = computed(() => confirming.value || deleting.value)
const filteredTags = computed(() => tags.value.filter(tag => tag.toLowerCase().includes(tagSearch.value.trim().toLowerCase())))
const pullCommand = computed(() => `docker pull ${imageReference(props.host, props.repository, selectedTag.value)}`)
const digestCommand = computed(() => `docker pull ${imageReference(props.host, props.repository, detail.value?.digest || '')}`)
const { copy } = useClipboard({ legacy: true })
let tagsController: AbortController | undefined, manifestController: AbortController | undefined
let disposed = false
function cancelRequests() { tagsController?.abort(); manifestController?.abort() }
function beforeClose(done: () => void) { if (!busy.value) done() }
async function copyText(value: string, label: string) { try { await copy(value); ElMessage.success(`${label}已复制`) } catch { ElMessage.error('复制失败，请重试') } }
async function loadTags(preserve = false) {
  tagsController?.abort(); manifestController?.abort()
  const current = new AbortController(); tagsController = current
  const previous = preserve ? selectedTag.value : props.initialReference
  const name = props.repository
  tagsLoading.value = true; manifestLoading.value = false; tagsError.value = ''; manifestError.value = ''; detail.value = undefined; selectedTag.value = ''
  try {
    const result = await builtinApi.tags(name, current.signal)
    if (disposed || current.signal.aborted) return
    tags.value = (result.tags || []).slice().sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
    const target = previous && (tags.value.includes(previous) || previous.startsWith('sha256:')) ? previous : tags.value[0]
    if (target) void loadManifest(target)
  } catch (error) { if (!disposed && !current.signal.aborted) { tags.value = []; tagsError.value = builtinError(error) } }
  finally { if (!disposed && !current.signal.aborted) tagsLoading.value = false }
}
async function loadManifest(reference: string) {
  if (!reference || busy.value) return
  manifestController?.abort()
  const current = new AbortController(); manifestController = current
  selectedTag.value = reference; detail.value = undefined; manifestError.value = ''; deleteError.value = ''; manifestLoading.value = true
  try { const result = await builtinApi.manifest(props.repository, reference, current.signal); if (!disposed && !current.signal.aborted) detail.value = result }
  catch (error) { if (!disposed && !current.signal.aborted) manifestError.value = builtinError(error) }
  finally { if (!disposed && !current.signal.aborted) manifestLoading.value = false }
}
async function removeManifest() {
  if (!props.admin || props.readOnly || !detail.value?.digest || busy.value || manifestLoading.value || tagsLoading.value) return
  const name = props.repository, digest = detail.value.digest
  confirming.value = true
  try { await ElMessageBox.confirm(`确定删除「${name}${selectedTag.value.startsWith('sha256:') ? '@' : ':'}${selectedTag.value}」对应的镜像清单？指向 ${digest} 的其他标签也会受影响。当前及保留的发布版本受保护。`, '删除镜像版本', { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消', closeOnClickModal: false }) }
  catch { return }
  finally { confirming.value = false }
  if (disposed || !props.modelValue || name !== props.repository) return
  deleting.value = true; deleteError.value = ''
  try {
    await builtinApi.remove(name, digest)
    if (disposed) return
    ElMessage.success('镜像版本已删除'); detail.value = undefined; selectedTag.value = ''
    emit('removed'); deleting.value = false; await loadTags()
  } catch (error) { if (!disposed) deleteError.value = builtinError(error) }
  finally { if (!disposed) deleting.value = false }
}
watch(() => [props.modelValue, props.repository] as const, ([open]) => {
  cancelRequests()
  tags.value = []; tagSearch.value = ''; selectedTag.value = ''; detail.value = undefined; tagsError.value = ''; manifestError.value = ''; deleteError.value = ''; tagsLoading.value = false; manifestLoading.value = false
  if (open && props.repository) void loadTags()
}, { immediate: true })
onBeforeUnmount(() => { disposed = true; cancelRequests() })
</script>
