<template>
  <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="chart" /></span>统计说明</h2><el-button :loading="loading" @click="load">刷新</el-button></div><p class="resource-help">实时显示容器 {{ name }} 的资源使用情况，以及容器内正在运行的进程。</p><div class="resource-inline refresh-rate"><label for="stats-rate">刷新间隔</label><el-select id="stats-rate" v-model="rate" class="form-control" aria-label="统计刷新间隔"><el-option v-for="seconds in [1, 3, 5, 10, 30, 60]" :key="seconds" :label="`${seconds} 秒`" :value="seconds" /></el-select></div></section>
  <el-alert v-if="!running" title="容器当前未运行，无法获取实时统计。" type="warning" :closable="false" show-icon />
  <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
  <div class="statistics-charts"><ResourceChart title="内存使用" :names="['缓存', '内存']" :points="memoryPoints" unit="bytes" /><ResourceChart title="处理器使用率" :names="['处理器']" :points="cpuPoints" unit="percent" /><ResourceChart title="网络使用（汇总）" :names="networkNames" :points="networkPoints" unit="rate" /><ResourceChart title="磁盘读写（汇总）" :names="['读取', '写入']" :points="ioPoints" unit="rate" /></div>
  <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="list" /></span>进程</h2><el-input v-model="search" class="form-control" placeholder="搜索进程…" aria-label="搜索容器进程" clearable /></div><el-alert v-if="processError" :title="processError" type="error" :closable="false" /><div class="resource-table-scroll"><table class="resource-table process-table" aria-label="容器进程"><thead><tr><th v-for="(title, index) in processes.Titles" :key="index"><button class="process-sort" @click="sort(index)">{{ title }}<PortainerIcon name="sort" /></button></th></tr></thead><tbody><tr v-for="(process, row) in pageProcesses" :key="row"><td v-for="(value, index) in process" :key="index">{{ value }}</td></tr><tr v-if="!pageProcesses.length"><td :colspan="Math.max(1, processes.Titles.length)" class="resource-muted">{{ loading ? '正在获取进程…' : '暂无匹配进程' }}</td></tr></tbody></table></div><div class="process-pagination"><span>共 {{ filteredProcesses.length }} 个进程</span><el-pagination v-model:current-page="page" v-model:page-size="size" :page-sizes="[10, 25, 50, 100]" :total="filteredProcesses.length" layout="sizes, prev, pager, next" /></div></section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dockerApi } from '@/api/docker'
import PortainerIcon from '@/components/PortainerIcon.vue'
import ResourceChart from './ResourceChart.vue'
import { containerMetrics, type ContainerStats } from '@/utils/containers'
import { containerError } from '@/utils/containerFeedback'
const props = defineProps<{ id: string; name: string; running: boolean }>()
const rate = ref(5), loading = ref(false), error = ref(''), processError = ref('')
const samples = ref<{ time: number; stats: ContainerStats; metrics: ReturnType<typeof containerMetrics> }[]>([])
const processes = ref<{ Titles: string[]; Processes: string[][] }>({ Titles: [], Processes: [] })
const search = ref(''), page = ref(1), size = ref(10), sortIndex = ref(1), descending = ref(false)
const filteredProcesses = computed(() => processes.value.Processes.filter(process => process.some(value => value.toLowerCase().includes(search.value.toLowerCase()))).sort((a, b) => String(a[sortIndex.value] || '').localeCompare(String(b[sortIndex.value] || ''), 'zh-CN', { numeric: true }) * (descending.value ? -1 : 1)))
const pageProcesses = computed(() => filteredProcesses.value.slice((page.value - 1) * size.value, page.value * size.value))
const memoryPoints = computed(() => samples.value.map(sample => ({ time: sample.time, values: [sample.metrics.cache, sample.metrics.memory] })))
const cpuPoints = computed(() => samples.value.map(sample => ({ time: sample.time, values: [sample.metrics.cpu || 0] })))
const networkInterfaces = computed(() => [...new Set(samples.value.flatMap(sample => Object.keys(sample.stats.networks || {})))])
const networkNames = computed(() => networkInterfaces.value.flatMap(name => [`${name} 接收`, `${name} 发送`]))
function delta(index: number, value: (sample: typeof samples.value[number]) => number) { if (index < 1) return 0; const previous = samples.value[index - 1], current = samples.value[index]; const seconds = (current.time - previous.time) / 1000; return seconds > 0 ? Math.max(0, value(current) - value(previous)) / seconds : 0 }
const networkPoints = computed(() => samples.value.map((sample, index) => ({ time: sample.time, values: networkInterfaces.value.flatMap(name => [delta(index, entry => entry.stats.networks?.[name]?.rx_bytes || 0), delta(index, entry => entry.stats.networks?.[name]?.tx_bytes || 0)]) })))
const ioPoints = computed(() => samples.value.map((sample, index) => ({ time: sample.time, values: [delta(index, entry => entry.metrics.read), delta(index, entry => entry.metrics.written)] })))
let timer: ReturnType<typeof setTimeout> | undefined, controller: AbortController | undefined, disposed = false
async function load() {
  if (loading.value || disposed || !props.running) return
  clearTimeout(timer); controller = new AbortController(); loading.value = true; error.value = ''; processError.value = ''
  const results = await Promise.allSettled([dockerApi.containers.stats(props.id, controller.signal), dockerApi.containers.top(props.id, controller.signal)])
  if (disposed) return
  if (results[0].status === 'fulfilled') { const stats = results[0].value; samples.value = [...samples.value, { time: Date.parse(stats.read || '') || Date.now(), stats, metrics: containerMetrics(stats) }].slice(-60) } else error.value = `统计加载失败：${containerError(results[0].reason)}`
  if (results[1].status === 'fulfilled') processes.value = { Titles: results[1].value.Titles || [], Processes: results[1].value.Processes || [] }; else processError.value = `进程加载失败：${containerError(results[1].reason)}`
  loading.value = false
  if (!error.value) timer = setTimeout(load, rate.value * 1000)
}
function sort(index: number) { descending.value = sortIndex.value === index ? !descending.value : false; sortIndex.value = index; page.value = 1 }
watch([search, size], () => { page.value = 1 })
watch(() => filteredProcesses.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / size.value))) })
watch(rate, () => { clearTimeout(timer); if (!loading.value) void load() })
onMounted(load)
onBeforeUnmount(() => { disposed = true; clearTimeout(timer); controller?.abort() })
</script>
<style scoped>
.refresh-rate { margin-top: 16px; }.statistics-charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }.process-table { table-layout: auto; min-width: 780px; }.process-table td { white-space: nowrap; }.process-table td:last-child { min-width: 300px; max-width: 700px; white-space: pre-wrap; word-break: break-word; }.process-sort { display: inline-flex; align-items: center; gap: 5px; border: 0; background: transparent; font: inherit; color: #ccc; cursor: pointer; }.process-sort .portainer-icon { width: 12px; height: 12px; }.process-pagination { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-top: 14px; font-size: 12px; color: #999; }
@media(max-width:1000px) { .statistics-charts { grid-template-columns: 1fr; } }
</style>
