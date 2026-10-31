<template>
  <section class="container-logs" aria-label="容器日志查看器">
    <div class="resource-card log-settings">
      <div class="log-filters">
        <label class="log-field log-period" for="log-period"><span>范围</span><el-select id="log-period" v-model="period" aria-label="日志获取范围"><el-option v-for="option in periods" :key="option.seconds" :label="option.label" :value="option.seconds" /></el-select></label>
        <label class="log-field log-tail" for="log-lines"><span>行数</span><el-input-number id="log-lines" v-model="tail" :min="1" :max="100000" :controls="false" aria-label="日志行数" /></label>
        <el-input id="log-search" v-model="search" class="log-search" placeholder="筛选日志…" aria-label="筛选日志" clearable><template #prefix><PortainerIcon name="search" /></template></el-input>
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
      <div class="log-toolbar">
        <div class="log-options">
          <label for="log-auto"><el-switch id="log-auto" v-model="autoRefresh" size="small" aria-label="自动刷新日志" /><span>自动刷新</span></label>
          <label for="log-wrap"><el-switch id="log-wrap" v-model="wrap" size="small" aria-label="日志自动换行" /><span>自动换行</span></label>
          <label for="log-time"><el-switch id="log-time" v-model="timestamps" size="small" aria-label="显示时间戳" /><span>时间戳</span></label>
        </div>
        <div class="log-actions">
          <span v-if="selectedIds.size" class="log-selection">已选 {{ selectedIds.size }} 行</span>
          <el-button :disabled="!filteredLines.length" @click="download">下载日志</el-button>
          <el-button :disabled="!filteredLines.length" @click="copy(false)">复制</el-button>
          <template v-if="selectedIds.size"><el-button @click="copy(true)">复制所选行</el-button><el-button @click="clearSelection">取消选择</el-button></template>
        </div>
      </div>
    </div>
    <div v-if="error" class="log-error" role="alert"><span>{{ error }}</span><el-button link :disabled="loading" @click="load">重试</el-button></div>
    <div ref="logViewport" class="log-output" :class="{ wrap }" role="region" aria-label="容器日志内容" tabindex="0" :aria-busy="loading" @pointerdown="beginTextSelection">
      <div v-for="(line, index) in filteredLines" :key="line.key" class="log-line" :class="[{ selected: selectedIds.has(line.key) }, line.level && `log-level-${line.level}`]">
        <button type="button" class="log-line-number" :aria-label="`选择第 ${line.number} 行日志`" :aria-pressed="selectedIds.has(line.key)" title="点击行号选择日志，按住 Shift 可连续选择" @click="selectLine(index, $event)">{{ line.number }}</button>
        <span class="log-line-text"><span v-for="(segment, segmentIndex) in line.segments" :key="segmentIndex" :style="segment.style">{{ segment.text }}</span><template v-if="!line.text">{{ ' ' }}</template></span>
      </div>
      <p v-if="!filteredLines.length" class="log-empty">{{ loading ? '正在获取日志…' : error ? '暂时无法加载日志，请重试' : search ? '没有匹配的日志' : '暂无日志' }}</p>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dockerApi } from '@/api/docker'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { containerError, copyContainerText, downloadContainerText } from '@/utils/containerFeedback'
import { formatImageDate } from '@/utils/localImages'
import { captureLogScrollAnchor, isLogViewportAtBottom, parseContainerLogs, restoreLogScrollAnchor, sliceLogSegments } from '@/utils/containerLogs'
const props = defineProps<{ id: string; name: string }>()
const autoRefresh = ref(true), wrap = ref(true), timestamps = ref(false), period = ref(0), tail = ref(100), search = ref('')
const rawLogs = ref(''), loading = ref(false), error = ref(''), selectedIds = ref(new Set<string>()), logViewport = ref<HTMLElement>()
const periods = [{ seconds: 0, label: '全部日志' }, { seconds: 86400, label: '最近一天' }, { seconds: 14400, label: '最近 4 小时' }, { seconds: 3600, label: '最近一小时' }, { seconds: 600, label: '最近 10 分钟' }]
const parsedLines = computed(() => parseContainerLogs(rawLogs.value))
const lines = computed(() => parsedLines.value.map(line => {
  const timestamp = line.text.match(/^(\d{4}-\d{2}-\d{2}T\S+)(\s)/)
  if (!timestamp) return line
  const segments = sliceLogSegments(line.segments, timestamp[0].length)
  if (timestamps.value) {
    const localTime = formatImageDate(timestamp[1].replace(/(\.\d{3})\d+(?=Z$|[+-]\d{2}:\d{2}$)/, '$1'))
    segments.unshift({ text: localTime === '—' ? timestamp[0] : `${localTime}${timestamp[2]}`, style: { color: '#89929e' } })
  }
  return { ...line, text: segments.map(segment => segment.text).join(''), segments }
}))
const filteredLines = computed(() => lines.value.filter(line => line.text.toLowerCase().includes(search.value.toLowerCase())))
let timer: ReturnType<typeof setTimeout> | undefined, controller: AbortController | undefined, disposed = false, lastSelectedKey = ''
let selectingText = false, pendingLogs: string | undefined
function schedule() { clearTimeout(timer); if (autoRefresh.value && !disposed) timer = setTimeout(load, 3000) }
function scrollToBottom() { const viewport = logViewport.value; viewport?.scrollTo({ top: viewport.scrollHeight }) }
function selectedLogText() {
  const viewport = logViewport.value, selection = window.getSelection()
  return viewport && selection && !selection.isCollapsed && viewport.contains(selection.anchorNode) && viewport.contains(selection.focusNode) ? selection.toString() : ''
}
function applyLogs(result: string) {
  // 选中文字期间继续获取最新日志，等取消选区后再更新正文，避免选区被刷新清除。
  if (selectingText || selectedLogText()) { pendingLogs = result; return }
  pendingLogs = undefined
  rawLogs.value = result
  const existing = new Set(lines.value.map(line => line.key))
  selectedIds.value = new Set([...selectedIds.value].filter(id => existing.has(id)))
  if (!existing.has(lastSelectedKey)) lastSelectedKey = ''
}
function applyPendingLogs() { if (!disposed && !selectingText && !selectedLogText() && pendingLogs !== undefined) applyLogs(pendingLogs) }
function beginTextSelection(event: PointerEvent) { if (event.button === 0 && event.target instanceof Element && event.target.closest('.log-line-text')) selectingText = true }
function endTextSelection() { selectingText = false; applyPendingLogs() }
async function load() {
  if (loading.value || disposed) return
  clearTimeout(timer); controller = new AbortController(); loading.value = true; error.value = ''
  try {
    const result = await dockerApi.containers.logs(props.id, String(tail.value || 100), { since: period.value ? Math.floor(Date.now() / 1000) - period.value : undefined, timestamps: true, signal: controller.signal })
    if (disposed) return
    applyLogs(result)
  } catch (err) { if (!disposed) error.value = `日志加载失败：${containerError(err)}` }
  finally { loading.value = false; if (!error.value) schedule() }
}
function clearSelection() { selectedIds.value = new Set(); lastSelectedKey = '' }
function selectLine(index: number, event: MouseEvent) {
  const selected = new Set(selectedIds.value)
  const lastIndex = filteredLines.value.findIndex(line => line.key === lastSelectedKey)
  const start = event.shiftKey && lastIndex >= 0 ? Math.min(index, lastIndex) : index
  const end = event.shiftKey && lastIndex >= 0 ? Math.max(index, lastIndex) : index
  const adding = !selected.has(filteredLines.value[index].key)
  for (let current = start; current <= end; current++) { if (adding) selected.add(filteredLines.value[current].key); else selected.delete(filteredLines.value[current].key) }
  selectedIds.value = selected; lastSelectedKey = filteredLines.value[index].key
}
const text = (selected: boolean) => filteredLines.value.filter(line => !selected || selectedIds.value.has(line.key)).map(line => line.text).join('\n')
const copy = (selected: boolean) => copyContainerText(selected ? text(true) : selectedLogText() || text(false))
const download = () => downloadContainerText(`${props.name}-日志.log`, text(false))
watch(autoRefresh, () => { if (autoRefresh.value) void load(); else clearTimeout(timer) })
watch([filteredLines, wrap, () => selectedIds.value.size, error], async (_, [previousLines]) => {
  const viewport = logViewport.value
  if (!viewport) return
  // 在更新正文前读取当前位置，包含等待请求期间发生的上翻操作。
  const follow = isLogViewportAtBottom(viewport) && !selectingText && !selectedLogText()
  const anchor = captureLogScrollAnchor(viewport, previousLines)
  await nextTick()
  if (disposed) return
  if (follow && !selectingText && !selectedLogText()) scrollToBottom()
  else restoreLogScrollAnchor(viewport, filteredLines.value, anchor)
})
watch([period, tail], () => { if (!loading.value) void load() })
watch(search, clearSelection)
onMounted(() => {
  document.addEventListener('selectionchange', applyPendingLogs)
  document.addEventListener('pointerup', endTextSelection)
  document.addEventListener('pointercancel', endTextSelection)
  void load()
})
onBeforeUnmount(() => {
  disposed = true; clearTimeout(timer); controller?.abort()
  document.removeEventListener('selectionchange', applyPendingLogs)
  document.removeEventListener('pointerup', endTextSelection)
  document.removeEventListener('pointercancel', endTextSelection)
})
</script>
<style scoped>
.container-logs { display: flex; flex-direction: column; flex: 1; min-height: 0; min-width: 0; gap: 12px; overflow: hidden; }
.container-resource .log-settings { flex-shrink: 0; padding: 12px 16px; }
.log-filters, .log-toolbar, .log-options, .log-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; min-width: 0; }
.log-filters { margin-bottom: 8px; }
.log-field { display: flex; align-items: center; gap: 8px; min-width: 0; font-size: 12px; }
.log-field > span { flex-shrink: 0; color: #aaa; }
.log-period { flex: 0 1 200px; }.log-period :deep(.el-select) { flex: 1; min-width: 0; }
.log-tail :deep(.el-input-number) { width: 84px; }
.log-search { flex: 1 1 180px; min-width: 0; }
.log-filters > .el-button, .log-actions > .el-button { margin: 0; }
.log-toolbar { justify-content: space-between; gap: 8px 16px; }
.log-options { gap: 8px 16px; }
.log-options label { display: inline-flex; align-items: center; gap: 6px; white-space: nowrap; font-size: 12px; color: #bbb; cursor: pointer; }
.log-selection { color: #999; font-size: 12px; white-space: nowrap; }
.log-error { display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; gap: 12px; padding: 8px 12px; border: 1px solid #693a40; border-radius: 6px; background: #3c2528; color: #ffc3c8; overflow-wrap: anywhere; }
.log-error > span { min-width: 0; }.log-error > .el-button { flex-shrink: 0; }
.log-output { flex: 1; min-height: 0; min-width: 0; overflow: auto; overflow-anchor: none; overscroll-behavior: contain; background: #171717; border: 1px solid #383838; border-radius: 6px; padding: 8px 0; font: 12px/1.7 Menlo, Monaco, Consolas, monospace; }
.log-line { display: flex; width: 100%; min-width: max-content; border: 0; background: transparent; color: #d5d5d5; padding: 0 16px 0 0; margin: 0; text-align: left; font: inherit; cursor: text; white-space: pre; }
.log-output.wrap .log-line { min-width: 0; }.log-output.wrap .log-line-text { white-space: pre-wrap; overflow-wrap: anywhere; min-width: 0; }
.log-line.log-level-error { color: #f1a7ac; }.log-line.log-level-warning { color: #e6c48e; }.log-line.log-level-info { color: #9acb8c; }.log-line.log-level-debug { color: #89929e; }
.log-line-text { user-select: text; -webkit-user-select: text; }.log-line-text::selection, .log-line-text :deep(::selection) { background: #315b76; color: #fff; }
.log-line:hover { background: #252c33; }.log-line.selected { background: #1e465e; }.log-line-number { display: inline-block; align-self: flex-start; flex-shrink: 0; min-width: 54px; padding: 0 12px; border: 0; background: transparent; color: #6f7b86; text-align: right; font: inherit; cursor: pointer; user-select: none; }
.log-line-number:hover, .log-line-number[aria-pressed="true"] { color: #85d3ee; }
.log-line-number:focus-visible, .log-output:focus-visible { outline: 1px solid #35acd7; outline-offset: -1px; }
.log-empty { color: #999; padding: 24px; text-align: center; }
@media (max-width: 760px) {
  .container-resource .log-settings { padding: 10px 12px; }
  .log-period { flex: 1 1 110px; }.log-tail :deep(.el-input-number) { width: 60px; }
  .log-search { order: 1; flex-basis: 100%; }
  .log-options { gap: 8px 12px; }.log-actions { gap: 6px; }
  .log-line { padding-right: 10px; }.log-line-number { min-width: 40px; padding: 0 8px; }
}
</style>
