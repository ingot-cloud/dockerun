<template>
  <div class="network-resource networks-page">
    <header class="network-heading">
      <h1>网络</h1>
      <el-button text circle aria-label="刷新网络列表" title="刷新网络列表" :disabled="busy || loading" @click="loadNetworks"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button>
    </header>
    <section class="network-card network-list" aria-labelledby="networks-title">
      <header class="networks-toolbar">
        <h2 id="networks-title"><span class="network-section-icon"><PortainerIcon name="network" /></span>网络列表</h2>
        <div class="networks-actions">
          <el-input v-model="search" class="network-search" placeholder="搜索名称、ID、应用栈或子网…" aria-label="搜索网络" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input>
          <el-button class="network-remove" :disabled="!selectedNetworks.length || busy || loading" @click="deleteVisible = true"><PortainerIcon name="trash" />删除</el-button>
          <el-button class="network-primary" :disabled="busy || loading" @click="router.push({ name: 'NetworkCreate', query: listQuery })"><PortainerIcon name="plus" />创建网络</el-button>
        </div>
      </header>
      <form class="network-filters" @submit.prevent>
        <el-select v-model="driverFilter" placeholder="全部驱动" aria-label="筛选网络驱动" :disabled="busy"><el-option label="全部驱动" value="" /><el-option v-for="driver in drivers" :key="driver" :label="driver" :value="driver" /></el-select>
        <el-select v-model="typeFilter" placeholder="全部类型" aria-label="筛选网络类型" :disabled="busy"><el-option label="全部类型" value="" /><el-option label="系统网络" value="system" /><el-option label="自定义网络" value="custom" /></el-select>
        <el-select v-model="attachableFilter" placeholder="手动连接：全部" aria-label="筛选手动连接" :disabled="busy"><el-option label="手动连接：全部" value="" /><el-option label="可手动连接" value="true" /><el-option label="不可手动连接" value="false" /></el-select>
        <el-button v-if="hasFilters" link :disabled="busy" @click="resetFilters">重置筛选</el-button>
        <span class="network-summary">自定义 <strong>{{ networks.filter(network => !isSystemNetwork(network)).length }}</strong><span>/</span>全部 <strong>{{ networks.length }}</strong></span>
      </form>
      <div v-if="listError" class="network-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading || busy" @click="loadNetworks">重试</el-button></div>
      <div ref="tableScroll" v-loading="loading" class="network-table-scroll" aria-label="网络列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="network-table" aria-label="网络列表">
          <colgroup><col class="network-selection-col" /><col class="network-name-col" /><col class="network-stack-col" /><col class="network-driver-col" /><col class="network-attachable-col" /><col class="network-ipam-col" /><col class="network-address-col" /><col class="network-address-col" /><col class="network-ipv6-col" /><col class="network-ipv6-col" /></colgroup>
          <thead><tr>
            <th scope="col" class="network-selection"><el-checkbox :model-value="allPageSelected" :indeterminate="somePageSelected && !allPageSelected" :disabled="!selectablePage.length || busy || loading" aria-label="选择当前页全部可删除网络" @change="togglePageSelection(Boolean($event))" /></th>
            <th v-for="column in networkColumns" :key="column.key" scope="col" :aria-sort="sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none'"><button class="network-sort" :disabled="busy" @click="changeSort(column.key)">{{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" /></button></th>
          </tr></thead>
          <tbody><tr v-for="network in pageNetworks" :key="network.Id" :class="{ selected: selectedIds.has(network.Id) }">
            <td class="network-selection"><el-checkbox :model-value="selectedIds.has(network.Id)" :disabled="!canRemoveNetwork(network) || busy || loading" :aria-label="`选择网络 ${network.Name}`" :title="isSystemNetwork(network) ? '系统网络不可删除' : network.Ingress ? 'Swarm 入口网络不可删除' : `选择 ${network.Name}`" @change="toggleSelection(network.Id, Boolean($event))" /></td>
            <td><div class="network-identity"><router-link :to="{ name: 'NetworkDetail', params: { id: network.Id }, query: listQuery }" :title="network.Name">{{ network.Name }}</router-link><span v-if="isSystemNetwork(network)" class="network-badge">系统</span><span v-else-if="network.Ingress" class="network-badge">入口网络</span></div></td>
            <td><span class="network-truncate network-muted" :title="networkStack(network)">{{ networkStack(network) || '—' }}</span></td>
            <td>{{ network.Driver }}</td>
            <td><span :class="network.Attachable ? 'network-true' : 'network-muted'">{{ network.Attachable ? '是' : '否' }}</span></td>
            <td>{{ network.IPAM?.Driver || 'default' }}</td>
            <td v-for="column in addressColumns" :key="column.key" class="network-address"><span :title="networkAddresses(network, column.family, column.field)">{{ networkAddresses(network, column.family, column.field) || '—' }}</span></td>
          </tr></tbody>
        </table>
        <div v-if="!pageNetworks.length && !loading" class="network-empty" role="status"><PortainerIcon :name="hasFilters ? 'search' : 'network'" /><p>{{ listError ? '暂时无法加载网络列表' : hasFilters ? '没有找到匹配的网络' : '暂无网络' }}</p><el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button><span v-else-if="!listError">创建网络后，可在这里查看和管理</span></div>
      </div>
      <footer class="networks-pagination">
        <div class="network-count" role="status"><span>共 {{ filteredNetworks.length }} 个网络</span><template v-if="selectedNetworks.length"><span class="network-selected-count">已选 {{ selectedNetworks.length }} 项</span><el-button link :disabled="busy" @click="selectedIds = new Set()">清除选择</el-button></template></div>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredNetworks.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" />
      </footer>
    </section>
    <NetworkDeleteDialog v-model="deleteVisible" :networks="selectedNetworks" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { dockerApi } from '@/api/docker'
import PortainerIcon from '@/components/PortainerIcon.vue'
import NetworkDeleteDialog from '@/components/networks/NetworkDeleteDialog.vue'
import { networkError } from '@/utils/networkFeedback'
import { canRemoveNetwork, filterNetworks, isSystemNetwork, networkAddresses, networkColumns, networkListQuery, networkStack, readNetworkListState, sortNetworks, type DockerNetwork, type NetworkSortKey } from '@/utils/networks'
import '@/styles/networks.css'

const router = useRouter()
const initial = readNetworkListState(useRoute().query)
const networks = ref<DockerNetwork[]>([])
const loading = ref(false), listError = ref(''), deleteVisible = ref(false)
const search = ref(initial.search), driverFilter = ref(initial.driver), typeFilter = ref(initial.type), attachableFilter = ref(initial.attachable)
const page = ref(initial.page), pageSize = ref(initial.size), sortKey = ref<NetworkSortKey>(initial.sort), descending = ref(initial.descending)
const selectedIds = ref(new Set<string>())
const tableScroll = ref<HTMLElement>()
const busy = computed(() => deleteVisible.value)
const hasFilters = computed(() => Boolean(search.value || driverFilter.value || typeFilter.value || attachableFilter.value))
const drivers = computed(() => [...new Set([...networks.value.map(network => network.Driver), ...(driverFilter.value ? [driverFilter.value] : [])])].sort())
const listQuery = computed(() => networkListQuery({ search: search.value, driver: driverFilter.value, type: typeFilter.value, attachable: attachableFilter.value, page: page.value, size: pageSize.value, sort: sortKey.value, descending: descending.value }))
const filteredNetworks = computed(() => sortNetworks(filterNetworks(networks.value, search.value, driverFilter.value, typeFilter.value, attachableFilter.value), sortKey.value, descending.value))
const pageNetworks = computed(() => filteredNetworks.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const selectablePage = computed(() => pageNetworks.value.filter(canRemoveNetwork))
const selectedNetworks = computed(() => networks.value.filter(network => selectedIds.value.has(network.Id) && canRemoveNetwork(network)))
const allPageSelected = computed(() => selectablePage.value.length > 0 && selectablePage.value.every(network => selectedIds.value.has(network.Id)))
const somePageSelected = computed(() => selectablePage.value.some(network => selectedIds.value.has(network.Id)))
const addressColumns = [{ key: 'v4subnet', family: 4, field: 'Subnet' }, { key: 'v4gateway', family: 4, field: 'Gateway' }, { key: 'v6subnet', family: 6, field: 'Subnet' }, { key: 'v6gateway', family: 6, field: 'Gateway' }] as const
const controller = new AbortController()
let disposed = false
watch([search, driverFilter, typeFilter, attachableFilter], () => { page.value = 1; selectedIds.value = new Set() })
watch(pageSize, () => { page.value = 1 })
watch(() => filteredNetworks.value.length, total => { if (!loading.value) page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch([page, pageSize, sortKey, descending, search, driverFilter, typeFilter, attachableFilter], () => tableScroll.value?.scrollTo({ top: 0 }))
function resetFilters() { search.value = ''; driverFilter.value = ''; typeFilter.value = ''; attachableFilter.value = '' }
function toggleSelection(id: string, checked: boolean) {
  const next = new Set(selectedIds.value)
  if (checked && networks.value.some(network => network.Id === id && canRemoveNetwork(network))) next.add(id)
  else next.delete(id)
  selectedIds.value = next
}
function togglePageSelection(checked: boolean) { for (const network of selectablePage.value) toggleSelection(network.Id, checked) }
function changeSort(key: NetworkSortKey) { descending.value = sortKey.value === key ? !descending.value : false; sortKey.value = key; page.value = 1 }
function onRemoved(ids: string[]) {
  const removed = new Set(ids)
  networks.value = networks.value.filter(network => !removed.has(network.Id))
  selectedIds.value = new Set([...selectedIds.value].filter(id => !removed.has(id)))
  void loadNetworks()
}
async function loadNetworks() {
  if (loading.value || disposed) return
  loading.value = true; listError.value = ''
  try {
    const result = await dockerApi.networks.list(controller.signal)
    if (disposed) return
    networks.value = result || []
    const existing = new Set(networks.value.filter(canRemoveNetwork).map(network => network.Id))
    selectedIds.value = new Set([...selectedIds.value].filter(id => existing.has(id)))
    page.value = Math.min(page.value, Math.max(1, Math.ceil(filteredNetworks.value.length / pageSize.value)))
  } catch (error) { if (!disposed) listError.value = `网络列表加载失败：${networkError(error)}` }
  finally { loading.value = false }
}
onMounted(loadNetworks)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
