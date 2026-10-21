<template>
  <section class="resource-card log-settings"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="logs" /></span>日志查看设置</h2><el-button :loading="loading" @click="load">刷新</el-button></div><div class="resource-form">
    <label for="log-auto">自动刷新日志</label><el-switch id="log-auto" v-model="autoRefresh" aria-label="自动刷新日志" />
    <label for="log-wrap">自动换行</label><el-switch id="log-wrap" v-model="wrap" aria-label="日志自动换行" />
    <label for="log-time">显示时间戳</label><el-switch id="log-time" v-model="timestamps" aria-label="显示时间戳" />
    <label for="log-period">获取范围</label><el-select id="log-period" v-model="period" class="form-control" aria-label="日志获取范围"><el-option v-for="option in periods" :key="option.seconds" :label="option.label" :value="option.seconds" /></el-select>
    <label for="log-search">搜索</label><el-input id="log-search" v-model="search" placeholder="筛选日志…" aria-label="筛选日志" clearable />
    <label for="log-lines">行数</label><el-input-number id="log-lines" v-model="tail" :min="1" :max="100000" :controls="false" aria-label="日志行数" />
    <label>操作</label><div class="resource-inline"><el-button @click="download">下载日志</el-button><el-button :disabled="!filteredLines.length" @click="copy(false)">复制</el-button><el-button :disabled="!selectedIds.size" @click="copy(true)">复制所选行</el-button><el-button :disabled="!selectedIds.size" @click="selectedIds = new Set()">取消选择</el-button></div>
  </div></section>
  <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
  <div ref="logViewport" class="log-output" :class="{ wrap }" role="region" aria-label="容器日志内容" tabindex="0" :aria-busy="loading">
    <button v-for="(line, index) in filteredLines" :key="line.key" class="log-line" :class="{ selected: selectedIds.has(line.key) }" :aria-pressed="selectedIds.has(line.key)" @click="selectLine(index, $event)"><span class="log-line-number">{{ line.number }}</span><span>{{ line.text || ' ' }}</span></button>
    <p v-if="!filteredLines.length" class="log-empty">{{ loading ? '正在获取日志…' : search ? '没有匹配的日志' : '暂无日志' }}</p>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dockerApi } from '@/api/docker'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { containerError, copyContainerText, downloadContainerText } from '@/utils/containerFeedback'
import { formatImageDate } from '@/utils/localImages'
const props = defineProps<{ id: string; name: string }>()
const autoRefresh = ref(true), wrap = ref(true), timestamps = ref(false), period = ref(0), tail = ref(100), search = ref('')
const rawLogs = ref(''), loading = ref(false), error = ref(''), selectedIds = ref(new Set<string>()), logViewport = ref<HTMLElement>()
const periods = [{ seconds: 0, label: '全部日志' }, { seconds: 86400, label: '最近一天' }, { seconds: 14400, label: '最近 4 小时' }, { seconds: 3600, label: '最近一小时' }, { seconds: 600, label: '最近 10 分钟' }]
const lines = computed(() => {
  const occurrences = new Map<string, number>()
  return rawLogs.value.replace(/\r\n/g, '\n').replace(/\n$/, '').split('\n').filter((value, index, values) => value || values.length > 1 || index > 0).map((raw, index) => {
    const count = occurrences.get(raw) || 0; occurrences.set(raw, count + 1)
    const text = raw.replace(/^(\d{4}-\d{2}-\d{2}T\S+)(\s)/, (prefix, timestamp: string, separator: string) => {
      if (!timestamps.value) return ''
      const localTime = formatImageDate(timestamp.replace(/(\.\d{3})\d+(?=Z$|[+-]\d{2}:\d{2}$)/, '$1'))
      return localTime === '—' ? prefix : `${localTime}${separator}`
    })
    return { key: `${raw}\0${count}`, text, number: index + 1 }
  })
})
const filteredLines = computed(() => lines.value.filter(line => line.text.toLowerCase().includes(search.value.toLowerCase())))
let timer: ReturnType<typeof setTimeout> | undefined, controller: AbortController | undefined, disposed = false, lastIndex = -1
function schedule() { clearTimeout(timer); if (autoRefresh.value && !disposed) timer = setTimeout(load, 3000) }
function scrollToBottom() { const viewport = logViewport.value; viewport?.scrollTo({ top: viewport.scrollHeight }) }
async function load() {
  if (loading.value || disposed) return
  clearTimeout(timer); controller = new AbortController(); loading.value = true; error.value = ''
  const viewport = logViewport.value, nearBottom = !viewport || viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight < 60
  try {
    const result = await dockerApi.containers.logs(props.id, String(tail.value || 100), { since: period.value ? Math.floor(Date.now() / 1000) - period.value : undefined, timestamps: true, signal: controller.signal })
    if (disposed) return
    rawLogs.value = result
    const existing = new Set(lines.value.map(line => line.key)); selectedIds.value = new Set([...selectedIds.value].filter(id => existing.has(id)))
    if (autoRefresh.value || (nearBottom && !selectedIds.value.size)) { await nextTick(); if (!disposed) scrollToBottom() }
  } catch (err) { if (!disposed) error.value = `日志加载失败：${containerError(err)}` }
  finally { loading.value = false; if (!error.value) schedule() }
}
function selectLine(index: number, event: MouseEvent) {
  const selected = new Set(selectedIds.value)
  const start = event.shiftKey && lastIndex >= 0 ? Math.min(index, lastIndex) : index
  const end = event.shiftKey && lastIndex >= 0 ? Math.max(index, lastIndex) : index
  const adding = !selected.has(filteredLines.value[index].key)
  for (let current = start; current <= end; current++) { if (adding) selected.add(filteredLines.value[current].key); else selected.delete(filteredLines.value[current].key) }
  selectedIds.value = selected; lastIndex = index
}
const text = (selected: boolean) => filteredLines.value.filter(line => !selected || selectedIds.value.has(line.key)).map(line => line.text).join('\n')
const copy = (selected: boolean) => copyContainerText(text(selected))
const download = () => downloadContainerText(`${props.name}-日志.log`, text(false))
watch(autoRefresh, () => { if (autoRefresh.value) void load(); else clearTimeout(timer) })
watch([autoRefresh, filteredLines, wrap], () => { if (autoRefresh.value) scrollToBottom() }, { flush: 'post' })
watch([period, tail], () => { if (!loading.value) void load() })
onMounted(load)
onBeforeUnmount(() => { disposed = true; clearTimeout(timer); controller?.abort() })
</script>
<style scoped>
.log-output { min-height: 340px; max-height: 65vh; overflow: auto; background: #151515; border: 1px solid #383838; border-radius: 6px; padding: 12px 0; font: 12px/1.7 Menlo, Monaco, Consolas, monospace; }
.log-line { display: flex; width: 100%; min-width: max-content; border: 0; background: transparent; color: #d5d5d5; padding: 0 16px 0 0; margin: 0; text-align: left; font: inherit; cursor: text; white-space: pre; }
.log-output.wrap .log-line { min-width: 0; }.log-output.wrap .log-line > span:last-child { white-space: pre-wrap; overflow-wrap: anywhere; min-width: 0; }
.log-line:hover { background: #252c33; }.log-line.selected { background: #1e465e; }.log-line-number { display: inline-block; flex-shrink: 0; min-width: 54px; padding: 0 12px; color: #6f7b86; text-align: right; user-select: none; }
.log-empty { color: #999; padding: 24px; text-align: center; }
</style>
