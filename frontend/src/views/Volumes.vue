<template>
  <div class="volume-resource volumes-page">
    <header class="volume-heading"><h1>卷</h1><el-button text circle aria-label="刷新卷列表" title="刷新卷列表" :disabled="busy || loading" @click="loadVolumes"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button></header>
    <section class="volume-card volume-list" aria-labelledby="volumes-title">
      <header class="volumes-toolbar">
        <h2 id="volumes-title"><span class="volume-section-icon"><PortainerIcon name="volume" /></span>卷列表</h2>
        <div class="volumes-actions">
          <el-input v-model="search" class="volume-search" placeholder="搜索名称、应用栈、路径或标签…" aria-label="搜索卷" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input>
          <el-button class="volume-remove" :disabled="!selectedVolumes.length || busy || loading || !!listError" @click="deleteVisible = true"><PortainerIcon name="trash" />删除</el-button>
          <el-button class="volume-primary" :disabled="busy || loading" @click="router.push({ name: 'VolumeCreate', query: listQuery })"><PortainerIcon name="plus" />创建卷</el-button>
        </div>
      </header>
      <form class="volume-filters" @submit.prevent>
        <el-select v-model="driverFilter" placeholder="全部驱动" aria-label="筛选卷驱动" :disabled="busy"><el-option label="全部驱动" value="" /><el-option v-for="driver in drivers" :key="driver" :label="driver" :value="driver" /></el-select>
        <el-select v-model="usageFilter" placeholder="全部使用状态" aria-label="筛选卷使用状态" :disabled="busy"><el-option label="全部使用状态" value="" /><el-option label="使用中" value="used" /><el-option label="未使用" value="unused" /><el-option label="状态未知" value="unknown" /></el-select>
        <el-button v-if="hasFilters" link :disabled="busy" @click="resetFilters">重置筛选</el-button>
        <span class="volume-summary"><template v-if="!usageError">未使用 <strong>{{ unusedCount }}</strong><span>/</span></template>全部 <strong>{{ volumes.length }}</strong></span>
      </form>
      <div v-if="listError" class="volume-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading || busy" @click="loadVolumes">重试</el-button></div>
      <div v-if="usageError" class="volume-warning" role="status"><span>无法读取容器引用状态，未使用标记已隐藏。{{ usageError }}</span><el-button link :disabled="loading || busy" @click="loadVolumes">重试</el-button></div>
      <div v-if="warnings.length" class="volume-warning" role="status"><span>{{ warnings.join('；') }}</span></div>
      <div ref="tableScroll" v-loading="loading" class="volume-table-scroll" aria-label="卷列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="volume-table" aria-label="卷列表">
          <colgroup><col class="volume-selection-col" /><col class="volume-name-col" /><col class="volume-stack-col" /><col class="volume-driver-col" /><col /><col class="volume-date-col" /><col class="volume-scope-col" /></colgroup>
          <thead><tr><th scope="col" class="volume-selection"><el-checkbox :model-value="allPageSelected" :indeterminate="somePageSelected && !allPageSelected" :disabled="!selectablePage.length || busy || loading || !!listError" aria-label="选择当前页全部可删除卷" @change="togglePageSelection(Boolean($event))" /></th><th v-for="column in volumeColumns" :key="column.key" scope="col" :aria-sort="sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none'"><button class="volume-sort" :disabled="busy" @click="changeSort(column.key)">{{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" /></button></th></tr></thead>
          <tbody><tr v-for="volume in pageVolumes" :key="volume.Name" :class="{ selected: selectedNames.has(volume.Name) }">
            <td class="volume-selection"><el-checkbox :model-value="selectedNames.has(volume.Name)" :disabled="!canRemove(volume) || busy || loading || !!listError" :aria-label="`选择卷 ${volume.Name}`" :title="canRemove(volume) ? `选择 ${volume.Name}` : '此卷仍被容器引用，包含已停止的容器'" @change="toggleSelection(volume.Name, Boolean($event))" /></td>
            <td><div class="volume-identity"><router-link :to="{ name: 'VolumeDetail', params: { name: volume.Name }, query: listQuery }" :title="volume.Name">{{ volume.Name }}</router-link><span v-if="volumeUseCount(volume, containers) === 0" class="volume-unused" title="没有容器引用此卷（包含已停止的容器）">未使用</span></div></td>
            <td><span class="volume-truncate volume-muted" :title="volumeStack(volume)">{{ volumeStack(volume) || '—' }}</span></td><td>{{ volume.Driver }}</td>
            <td><span class="volume-truncate volume-mountpoint" :title="volume.Mountpoint">{{ volume.Mountpoint || '—' }}</span></td><td class="volume-date">{{ formatVolumeDate(volume.CreatedAt) }}</td><td>{{ volume.Scope || '—' }}</td>
          </tr></tbody>
        </table>
        <div v-if="!pageVolumes.length && !loading" class="volume-empty" role="status"><PortainerIcon :name="hasFilters ? 'search' : 'volume'" /><p>{{ listError ? '暂时无法加载卷列表' : hasFilters ? '没有找到匹配的卷' : '暂无卷' }}</p><el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button><span v-else-if="!listError">创建卷后，可在这里查看和管理持久化数据</span></div>
      </div>
      <footer class="volumes-pagination"><div class="volume-count" role="status"><span>共 {{ filteredVolumes.length }} 个卷</span><template v-if="selectedVolumes.length"><span class="volume-selected-count">已选 {{ selectedVolumes.length }} 项</span><el-button link :disabled="busy" @click="selectedNames = new Set()">清除选择</el-button></template></div><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredVolumes.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" /></footer>
    </section>
    <VolumeDeleteDialog v-model="deleteVisible" :volumes="selectedVolumes" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { dockerApi } from '@/api/docker'
import PortainerIcon from '@/components/PortainerIcon.vue'
import VolumeDeleteDialog from '@/components/volumes/VolumeDeleteDialog.vue'
import { volumeError } from '@/utils/volumeFeedback'
import { filterVolumes, formatVolumeDate, readVolumeListState, sortVolumes, volumeColumns, volumeListQuery, volumeStack, volumeUseCount, type DockerVolume, type VolumeContainer, type VolumeSortKey } from '@/utils/volumes'
import '@/styles/volumes.css'
const router = useRouter(), initial = readVolumeListState(useRoute().query)
const volumes = ref<DockerVolume[]>([]), containers = ref<VolumeContainer[] | null>(null)
const loading = ref(false), listError = ref(''), usageError = ref(''), warnings = ref<string[]>([]), deleteVisible = ref(false)
const search = ref(initial.search), driverFilter = ref(initial.driver), usageFilter = ref(initial.usage), page = ref(initial.page), pageSize = ref(initial.size), sortKey = ref<VolumeSortKey>(initial.sort), descending = ref(initial.descending)
const selectedNames = ref(new Set<string>()), tableScroll = ref<HTMLElement>()
const busy = computed(() => deleteVisible.value), hasFilters = computed(() => Boolean(search.value || driverFilter.value || usageFilter.value))
const drivers = computed(() => [...new Set([...volumes.value.map(volume => volume.Driver), ...(driverFilter.value ? [driverFilter.value] : [])])].sort())
const listQuery = computed(() => volumeListQuery({ search: search.value, driver: driverFilter.value, usage: usageFilter.value, page: page.value, size: pageSize.value, sort: sortKey.value, descending: descending.value }))
const filteredVolumes = computed(() => sortVolumes(filterVolumes(volumes.value, containers.value, search.value, driverFilter.value, usageFilter.value), sortKey.value, descending.value))
const pageVolumes = computed(() => filteredVolumes.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const canRemove = (volume: DockerVolume) => { const count = volumeUseCount(volume, containers.value); return count === null || count === 0 }
const selectablePage = computed(() => pageVolumes.value.filter(canRemove))
const selectedVolumes = computed(() => volumes.value.filter(volume => selectedNames.value.has(volume.Name) && canRemove(volume)))
const allPageSelected = computed(() => selectablePage.value.length > 0 && selectablePage.value.every(volume => selectedNames.value.has(volume.Name)))
const somePageSelected = computed(() => selectablePage.value.some(volume => selectedNames.value.has(volume.Name)))
const unusedCount = computed(() => volumes.value.filter(volume => volumeUseCount(volume, containers.value) === 0).length)
const controller = new AbortController()
let disposed = false
watch([search, driverFilter, usageFilter], () => { page.value = 1; selectedNames.value = new Set() })
watch(pageSize, () => { page.value = 1 })
watch(() => filteredVolumes.value.length, total => { if (!loading.value) page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch([page, pageSize, sortKey, descending, search, driverFilter, usageFilter], () => tableScroll.value?.scrollTo({ top: 0 }))
function resetFilters() { search.value = ''; driverFilter.value = ''; usageFilter.value = '' }
function toggleSelection(name: string, checked: boolean) {
  const next = new Set(selectedNames.value)
  if (checked && volumes.value.some(volume => volume.Name === name && canRemove(volume))) next.add(name)
  else next.delete(name)
  selectedNames.value = next
}
function togglePageSelection(checked: boolean) { for (const volume of selectablePage.value) toggleSelection(volume.Name, checked) }
function changeSort(key: VolumeSortKey) { descending.value = sortKey.value === key ? !descending.value : key === 'CreatedAt'; sortKey.value = key; page.value = 1 }
function onRemoved(names: string[]) { const removed = new Set(names); volumes.value = volumes.value.filter(volume => !removed.has(volume.Name)); selectedNames.value = new Set([...selectedNames.value].filter(name => !removed.has(name))); void loadVolumes() }
async function loadVolumes() {
  if (loading.value || disposed) return
  loading.value = true; listError.value = ''; usageError.value = ''; warnings.value = []
  try {
    const [volumeResult, containerResult] = await Promise.allSettled([dockerApi.volumes.list(controller.signal), dockerApi.containers.list(true, controller.signal)])
    if (disposed) return
    if (volumeResult.status === 'fulfilled') { volumes.value = volumeResult.value.Volumes || []; warnings.value = volumeResult.value.Warnings || [] }
    else listError.value = `卷列表加载失败：${volumeError(volumeResult.reason)}`
    if (containerResult.status === 'fulfilled') { containers.value = (containerResult.value as unknown as VolumeContainer[] | null) || []; if (containers.value.some(container => container.Mounts === undefined)) usageError.value = '部分容器未返回挂载信息' }
    else { containers.value = null; usageError.value = volumeError(containerResult.reason) }
    const existing = new Set(volumes.value.filter(canRemove).map(volume => volume.Name))
    selectedNames.value = new Set([...selectedNames.value].filter(name => existing.has(name)))
    page.value = Math.min(page.value, Math.max(1, Math.ceil(filteredVolumes.value.length / pageSize.value)))
  } finally { loading.value = false }
}
onMounted(loadVolumes)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
