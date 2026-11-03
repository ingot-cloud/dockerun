<template>
  <div class="containers-page">
    <header class="containers-heading">
      <h1>容器</h1>
      <el-button text circle aria-label="刷新容器列表" title="刷新容器列表" :disabled="busy || loading" @click="loadContainers"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button>
    </header>
    <section class="containers-card" aria-labelledby="containers-title">
      <header class="containers-toolbar">
        <h2 id="containers-title"><span class="section-icon"><PortainerIcon name="box" /></span>容器列表</h2>
        <el-input v-model="search" class="container-search" placeholder="搜索名称、镜像、IP 或端口…" aria-label="搜索容器" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input>
        <ContainerActionGroup :containers="selectedContainers" :busy="busy || loading" :loading="activeAction" @action="runAction" />
      </header>
      <form class="container-filters" @submit.prevent>
        <el-select v-model="stateFilter" class="state-filter" placeholder="全部状态" aria-label="筛选容器状态" :disabled="busy"><el-option label="全部状态" value="" /><el-option v-for="(label, state) in containerStates" :key="state" :label="label" :value="state" /></el-select>
        <el-button v-if="search || stateFilter" link :disabled="busy" @click="resetFilters">重置筛选</el-button>
        <span class="container-summary">运行中 <strong>{{ runningCount }}</strong><span class="summary-divider">/</span>全部 <strong>{{ containers.length }}</strong></span>
      </form>
      <div v-if="listError" class="list-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading || busy" @click="loadContainers">重试</el-button></div>
      <div v-if="actionFailures.length" class="action-failures" role="alert"><div><strong>{{ actionFailures.length }} 个容器操作失败</strong><el-button link @click="actionFailures = []">关闭</el-button></div><ul><li v-for="failure in actionFailures" :key="failure.id">{{ failure.name }}：{{ failure.message }}</li></ul></div>
      <div ref="tableScroll" v-loading="loading" class="container-table-scroll" aria-label="容器列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="container-table" aria-label="容器列表">
          <colgroup><col class="selection-column" /><col class="name-column" /><col class="state-column" /><col class="quick-column" /><col class="stack-column" /><col class="image-column" /><col class="date-column" /><col class="ip-column" /><col class="ports-column" /></colgroup>
          <thead><tr>
            <th scope="col" class="selection-cell"><el-checkbox :model-value="allPageSelected" :indeterminate="somePageSelected && !allPageSelected" :disabled="!pageContainers.length || busy || loading" aria-label="选择当前页全部容器" @change="togglePageSelection" /></th>
            <th v-for="column in columns" :key="column.label" scope="col" :aria-sort="column.key ? (sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none') : undefined"><button v-if="column.key" class="column-sort" :disabled="busy" @click="changeSort(column.key)">{{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" /></button><span v-else>{{ column.label }}</span></th>
          </tr></thead>
          <tbody><tr v-for="container in pageContainers" :key="container.Id" :class="{ selected: selectedIds.has(container.Id) }">
            <td class="selection-cell"><el-checkbox :model-value="selectedIds.has(container.Id)" :disabled="busy || loading" :aria-label="`选择容器 ${containerName(container)}`" @change="toggleSelection(container.Id, Boolean($event))" /></td>
            <td><router-link class="container-name" :title="containerName(container)" :to="{ name: 'ContainerDetail', params: { id: container.Id }, query: listQuery }">{{ containerName(container) }}</router-link><span class="container-id" :title="container.Id">{{ container.Id.slice(0, 12) }}</span></td>
            <td><span class="container-state" :class="`state-${container.State}`">{{ containerStateLabel(container.State) }}</span></td>
            <td><ContainerQuickActions :container="container" :query="listQuery" /></td>
            <td><el-tooltip :content="containerStack(container)" :disabled="!containerStack(container)" placement="top" :show-after="150" popper-style="max-width: 360px; overflow-wrap: anywhere;"><span class="stack-name" :class="{ muted: !containerStack(container) }">{{ containerStack(container) || '—' }}</span></el-tooltip></td>
            <td><router-link v-if="container.ImageID" class="container-image" :title="containerImage(container)" :to="{ name: 'ImageDetail', params: { id: container.ImageID } }">{{ containerImage(container) }}</router-link><span v-else class="container-image" :title="containerImage(container)">{{ containerImage(container) }}</span><span v-if="container.ImageID" class="container-id" :title="container.ImageID">{{container.ImageID.replace(/^sha256:/,'').slice(0,12)}}</span></td>
            <td class="container-date">{{ formatImageDate(container.Created) }}</td>
            <td><CompactItems :items="containerIPs(container)" label="网络地址" /></td>
            <td><ContainerPorts :container="container" /></td>
          </tr></tbody>
        </table>
        <div v-if="!pageContainers.length && !loading" class="container-empty" role="status"><PortainerIcon :name="search || stateFilter ? 'search' : 'box'" /><p>{{ listError ? '暂时无法加载容器列表' : search || stateFilter ? '没有找到匹配的容器' : '暂无容器' }}</p><el-button v-if="search || stateFilter" link @click="resetFilters">重置筛选</el-button><span v-else-if="!listError">部署应用后，可在这里查看和管理容器</span></div>
      </div>
      <footer class="containers-pagination">
        <div class="container-count" role="status"><span>共 {{ filteredContainers.length }} 个容器</span><template v-if="selectedContainers.length"><span class="selected-count">已选 {{ selectedContainers.length }} 项</span><el-button link :disabled="busy" @click="selectedIds = new Set()">清除选择</el-button></template></div>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredContainers.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" />
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ContainerActionGroup from '@/components/containers/ContainerActionGroup.vue'
import ContainerQuickActions from '@/components/containers/ContainerQuickActions.vue'
import ContainerPorts from '@/components/containers/ContainerPorts.vue'
import CompactItems from '@/components/containers/CompactItems.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { isAxiosError } from 'axios'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { dockerApi } from '@/api/docker'
import { formatImageDate } from '@/utils/localImages'
import { canContainerAction, containerActions, containerImage, containerIPs, containerListQuery, containerName, containerStack, containerStateLabel, containerStates, filterContainers, readContainerListState, sortContainers, type Container, type ContainerAction, type ContainerSortKey } from '@/utils/containers'

const initialState = readContainerListState(useRoute().query)
const containers = ref<Container[]>([])
const loading = ref(false)
const listError = ref('')
const search = ref(initialState.search)
const stateFilter = ref(initialState.state)
const page = ref(initialState.page)
const pageSize = ref(initialState.size)
const sortKey = ref<ContainerSortKey>(initialState.sort)
const descending = ref(initialState.descending)
const listQuery = computed(() => containerListQuery({ search: search.value, state: stateFilter.value, page: page.value, size: pageSize.value, sort: sortKey.value, descending: descending.value }))
const selectedIds = ref(new Set<string>())
const tableScroll = ref<HTMLElement>()
const activeAction = ref<ContainerAction | ''>('')
const confirming = ref(false)
const busy = computed(() => Boolean(activeAction.value) || confirming.value)
const actionFailures = ref<{ id: string; name: string; message: string }[]>([])
let disposed = false
const actions = containerActions
const columns: { key?: ContainerSortKey; label: string }[] = [
  { key: 'name', label: '名称' }, { key: 'State', label: '状态' }, { label: '快捷操作' }, { key: 'stack', label: '应用栈' },
  { key: 'Image', label: '镜像' }, { key: 'Created', label: '创建时间' }, { label: 'IP 地址' }, { label: '端口映射' },
]
const filteredContainers = computed(() => sortContainers(filterContainers(containers.value, search.value, stateFilter.value), sortKey.value, descending.value))
const pageContainers = computed(() => filteredContainers.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const selectedContainers = computed(() => containers.value.filter(container => selectedIds.value.has(container.Id)))
const allPageSelected = computed(() => pageContainers.value.length > 0 && pageContainers.value.every(container => selectedIds.value.has(container.Id)))
const somePageSelected = computed(() => pageContainers.value.some(container => selectedIds.value.has(container.Id)))
const runningCount = computed(() => containers.value.filter(container => container.State === 'running').length)
const actionLabel = (action: ContainerAction) => actions.find(item => item.key === action)!.label
const actionTargets = (action: ContainerAction) => selectedContainers.value.filter(container => canContainerAction(container, action))
const errorMessage = (error: unknown) => {
  if (isAxiosError(error)) {
    let data = error.response?.data
    if (typeof data === 'string') {
      try { data = JSON.parse(data) } catch { data = undefined }
    }
    if (typeof data?.error === 'string') return data.error
    if (error.code === 'ECONNABORTED') return '请求超时，请重试'
    return error.response ? `请求失败（${error.response.status}），请重试` : '无法连接服务，请检查网络后重试'
  }
  return error instanceof Error ? error.message : '操作失败，请重试'
}
watch([search, stateFilter], () => { page.value = 1; selectedIds.value = new Set() })
watch(pageSize, () => { page.value = 1 })
watch(() => filteredContainers.value.length, total => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch([page, pageSize, sortKey, descending, search, stateFilter], () => { tableScroll.value?.scrollTo({ top: 0 }) })
function resetFilters() { search.value = ''; stateFilter.value = '' }
function toggleSelection(id: string, checked: boolean) {
  const next = new Set(selectedIds.value)
  if (checked) next.add(id)
  else next.delete(id)
  selectedIds.value = next
}
function togglePageSelection(checked: string | number | boolean) {
  const next = new Set(selectedIds.value)
  for (const container of pageContainers.value) { if (checked) next.add(container.Id); else next.delete(container.Id) }
  selectedIds.value = next
}
function changeSort(key: ContainerSortKey) {
  descending.value = sortKey.value === key ? !descending.value : key === 'Created'
  sortKey.value = key
  page.value = 1
}

async function loadContainers() {
  if (loading.value || disposed) return
  loading.value = true
  listError.value = ''
  try {
    const result = await dockerApi.containers.list(true) as unknown as Container[] | null
    if (disposed) return
    containers.value = result || []
    const existing = new Set(containers.value.map(container => container.Id))
    selectedIds.value = new Set([...selectedIds.value].filter(id => existing.has(id)))
  } catch (error) { if (!disposed) listError.value = `容器列表加载失败：${errorMessage(error)}` }
  finally { loading.value = false }
}
async function runAction(action: ContainerAction, container?: Container) {
  if (busy.value || loading.value) return
  const targets = container ? (canContainerAction(container, action) ? [container] : []) : actionTargets(action)
  if (!targets.length) return
  const label = actionLabel(action)
  if (action === 'remove' || action === 'kill') {
    confirming.value = true
    try {
      await ElMessageBox.confirm(`${action === 'remove' ? `确定删除以下 ${targets.length} 个容器？此操作无法撤销，挂载的数据卷会保留。` : `确定强制停止以下 ${targets.length} 个容器？未保存的数据可能丢失。`}\n${targets.map(containerName).join('、')}`, action === 'remove' ? '删除容器' : '强制停止容器', { confirmButtonText: '确认', cancelButtonText: '取消', type: 'warning', customClass: 'container-delete-confirm' })
    } catch { return }
    finally { confirming.value = false }
  }
  if (disposed) return
  activeAction.value = action
  actionFailures.value = []
  let succeeded = 0
  const skipped = container ? 0 : selectedContainers.value.length - targets.length
  try {
    for (const target of targets) {
      if (disposed) break
      try {
        if (action === 'remove') await dockerApi.containers.remove(target.Id, false)
        else await dockerApi.containers[action](target.Id)
        succeeded++
        toggleSelection(target.Id, false)
      } catch (error) { actionFailures.value.push({ id: target.Id, name: containerName(target), message: errorMessage(error) }) }
    }
    if (!disposed && succeeded) ElMessage.success(`已${label} ${succeeded} 个容器${skipped ? `，跳过 ${skipped} 个状态不适用的容器` : ''}`)
    await loadContainers()
  } finally { activeAction.value = '' }
}

onMounted(loadContainers)
onBeforeUnmount(() => { disposed = true })
</script>

<style scoped>
.containers-page { --el-color-primary: #33a7d5; --el-fill-color-blank: #202020; --el-border-color: #414141; --el-text-color-regular: #d6d6d6; --el-text-color-placeholder: #999; display: flex; flex-direction: column; flex: 1; min-width: 0; min-height: 0; gap: 16px; color: #ebebeb; font-size: 13px; }
.containers-heading { display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; }
.containers-heading h1 { margin: 0; font-size: 18px; line-height: 28px; font-weight: 600; }
.containers-heading .el-button { color: #ccc; height: 28px; }
.containers-card { display: flex; flex-direction: column; flex: 1; min-height: 0; background: #202020; border: 1px solid #303030; border-radius: 10px; overflow: hidden; }
.containers-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: nowrap; gap: 16px; padding: 18px 20px; overflow-x: auto; flex-shrink: 0; }
h2 { display: flex; align-items: center; gap: 10px; margin: 0; font-size: 15px; line-height: 24px; font-weight: 600; white-space: nowrap; }
.section-icon { display: grid; place-items: center; width: 32px; height: 32px; flex-shrink: 0; border-radius: 50%; background: #303844; color: #c6d4e5; }
.section-icon .portainer-icon { width: 18px; height: 18px; }
.containers-page :deep(.el-button) { font-size: 12px; }.containers-page :deep(.el-button > span) { gap: 6px; }
.container-filters { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; padding: 0 20px 16px; flex-shrink: 0; }.container-search { width: 240px; min-width: 180px; max-width: 280px; flex-shrink: 0; margin-left: auto; }.state-filter { width: 140px; }
.containers-page :deep(.el-input__wrapper), .container-filters :deep(.el-select__wrapper) { background: #171717; border: 1px solid #3b3b3b; border-radius: 4px; box-shadow: none; min-height: 32px; }
.containers-page :deep(.el-input__wrapper:hover), .container-filters :deep(.el-select__wrapper:hover) { border-color: #666; }.containers-page :deep(.el-input__wrapper.is-focus), .container-filters :deep(.el-select__wrapper.is-focused) { border-color: #33a7d5; }
.containers-page :deep(.el-input__inner), .container-filters :deep(.el-select__placeholder) { font-size: 12px; }
.container-summary { margin-left: auto; color: #999; white-space: nowrap; font-size: 12px; }.container-summary strong { font-weight: 500; color: #d6d6d6; margin-left: 4px; }.summary-divider { margin: 0 10px; color: #666; }
.container-table-scroll { flex: 1; min-height: 0; overflow: auto; position: relative; outline-offset: -2px; }.container-table-scroll:focus-visible { outline: 2px solid #33a7d5; }
.container-table { width: 100%; min-width: 1240px; border-spacing: 0; table-layout: fixed; text-align: left; }
.selection-column { width: 48px; }.name-column { width: 180px; }.state-column { width: 88px; }.quick-column { width: 128px; }.stack-column { width: 110px; }.image-column { width: auto; }.date-column { width: 165px; }.ip-column { width: 135px; }.ports-column { width: 200px; }
.container-table thead { position: sticky; top: 0; z-index: 2; background: #303030; }.container-table th { height: 42px; font-size: 12px; font-weight: 500; padding: 0 12px; color: #d5d5d5; }
.column-sort { display: inline-flex; align-items: center; gap: 5px; padding: 8px 0; border: 0; background: transparent; color: inherit; font: inherit; cursor: pointer; }.column-sort .portainer-icon { width: 13px; height: 13px; color: #777; }.column-sort .sort-active { color: #ddd; }
.container-table th[aria-sort="ascending"] .sort-active :deep(path:first-child), .container-table th[aria-sort="descending"] .sort-active :deep(path:last-child) { opacity: .3; }
.container-table td { height: 60px; padding: 8px 12px; border-bottom: 1px solid #303030; line-height: 20px; vertical-align: middle; }.container-table tr:hover td { background: #282828; }.container-table tr.selected td { background: #24343c; }
.container-table .selection-cell { padding-left: 20px; padding-right: 8px; }.selection-cell :deep(.el-checkbox) { height: 20px; vertical-align: middle; }.selection-cell :deep(.el-checkbox__inner) { width: 15px; height: 15px; border-radius: 4px; background: #eee; border-color: #bbb; }.selection-cell :deep(.is-checked .el-checkbox__inner), .selection-cell :deep(.is-indeterminate .el-checkbox__inner) { background: #1685b3; border-color: #1685b3; }
.container-name, .container-image { display: block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #35acd7; text-decoration: none; }.container-name { padding: 0; border: 0; background: transparent; font: inherit; text-align: left; cursor: pointer; }.container-name:hover, a.container-image:hover { color: #76cdec; text-decoration: underline; }.container-id { display: block; color: #858585; font-size: 11px; font-variant-numeric: tabular-nums; }
.container-state { display: inline-flex; align-items: center; gap: 5px; border-radius: 10px; padding: 0 7px; background: #383838; color: #ccc; font-size: 11px; white-space: nowrap; }.container-state::before { content: ''; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }.state-running { color: #a5dbb8; background: #204632; }.state-paused, .state-restarting { color: #f1d69d; background: #554224; }.state-dead { color: #ffb1b8; background: #5a2a31; }
.stack-name { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #bbb; }.container-date, .container-ips, .container-ports { font-size: 12px; font-variant-numeric: tabular-nums; }.container-date { white-space: nowrap; }.container-ips, .container-ports { display: flex; flex-direction: column; gap: 2px; overflow-wrap: anywhere; }.container-ports > span { display: inline-flex; align-items: center; gap: 4px; flex-wrap: wrap; }.container-ports .portainer-icon { width: 12px; height: 12px; }.container-ports small { color: #8d8d8d; font-size: 10px; }.published { color: #46b1d7; }.muted { color: #888; }
.container-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; min-height: 150px; height: calc(100% - 42px); color: #999; }.container-empty > .portainer-icon { width: 30px; height: 30px; }.container-empty > span { font-size: 12px; }
.containers-pagination { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px 16px; padding: 12px 20px; min-height: 58px; border-top: 1px solid #303030; flex-shrink: 0; }.container-count { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; color: #aaa; font-size: 12px; }.selected-count { color: #91cde4; }
.containers-pagination :deep(.el-pagination) { --el-pagination-bg-color: transparent; --el-pagination-button-disabled-bg-color: transparent; --el-pagination-hover-color: #42b6df; flex-wrap: wrap; font-size: 12px; }.containers-pagination :deep(.el-pagination__sizes) { margin-right: 8px; }.containers-pagination :deep(.el-select) { width: 110px; }.containers-pagination :deep(.el-select__wrapper) { background: #171717; box-shadow: 0 0 0 1px #444 inset; min-height: 30px; }
.list-error, .action-failures { flex-shrink: 0; padding: 10px 20px; background: #3c2528; color: #ffc3c8; overflow-wrap: anywhere; }.list-error, .action-failures > div { display: flex; align-items: center; justify-content: space-between; gap: 12px; }.list-error > span { max-height: 54px; overflow-y: auto; }.action-failures ul { max-height: 70px; overflow: auto; list-style: none; padding: 0; font-size: 12px; line-height: 22px; }
.spinning { animation: container-spin 1s linear infinite; }@keyframes container-spin { to { transform: rotate(360deg); } }
@media (max-width: 1100px) { .containers-toolbar { padding: 12px 16px; }.container-filters { padding: 0 16px 12px; }.containers-pagination { padding: 10px 16px; } }
@media (max-width: 760px) { .containers-page { gap: 12px; }.container-search { width: 210px; }.container-summary { margin-left: auto; }.containers-toolbar { gap: 10px; } }
.container-table tbody tr { height: 60px; }
.container-name { text-decoration: none; }
</style>
