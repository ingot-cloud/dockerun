<template>
  <div class="registry-resource registries-page">
    <header class="registry-heading">
      <h1>外部仓库</h1>
      <el-button text circle aria-label="刷新外部仓库列表" title="刷新外部仓库列表" :disabled="busy || loading" @click="loadRegistries"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button>
    </header>
    <section class="registry-card registry-list" aria-labelledby="registries-title">
      <header class="registries-toolbar">
        <h2 id="registries-title"><span class="registry-section-icon"><PortainerIcon name="registry" /></span>Registries</h2>
        <div class="registries-actions">
          <el-input v-model="search" class="registry-search" placeholder="搜索名称、URL 或用户名…" aria-label="搜索外部仓库" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input>
          <el-button class="registry-remove" :disabled="!selectedRegistries.length || busy || loading" @click="deleteVisible = true"><PortainerIcon name="trash" />删除</el-button>
          <el-button class="registry-primary" :disabled="busy || loading" @click="router.push({ name: 'RegistryCreate', query: listQuery })"><PortainerIcon name="plus" />添加 Registry</el-button>
        </div>
      </header>
      <form class="registry-filters" @submit.prevent>
        <el-select v-model="typeFilter" placeholder="全部类型" aria-label="筛选仓库类型" :disabled="busy"><el-option label="全部类型" value="" /><el-option v-for="provider in registryProviders" :key="provider.id" :label="provider.label" :value="provider.id" /></el-select>
        <el-select v-model="authFilter" placeholder="身份认证：全部" aria-label="筛选身份认证" :disabled="busy"><el-option label="身份认证：全部" value="" /><el-option label="已启用认证" value="true" /><el-option label="匿名访问" value="false" /></el-select>
        <el-select v-model="defaultFilter" placeholder="全部仓库" aria-label="筛选默认仓库" :disabled="busy"><el-option label="全部仓库" value="" /><el-option label="默认仓库" value="true" /><el-option label="非默认仓库" value="false" /></el-select>
        <el-button v-if="hasFilters" link :disabled="busy" @click="resetFilters">重置筛选</el-button>
        <span class="registry-summary">已启用认证 <strong>{{ registries.filter(item => item.username).length }}</strong><span>/</span>全部 <strong>{{ registries.length }}</strong></span>
      </form>
      <div v-if="listError" class="registry-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading || busy" @click="loadRegistries">重试</el-button></div>
      <div ref="tableScroll" v-loading="loading" class="registry-table-scroll" aria-label="外部仓库列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="registry-table" aria-label="外部仓库列表">
          <colgroup><col class="registry-selection-col" /><col class="registry-name-col" /><col class="registry-type-col" /><col /><col class="registry-auth-col" /><col class="registry-action-col" /></colgroup>
          <thead><tr>
            <th scope="col" class="registry-selection"><el-checkbox :model-value="allPageSelected" :indeterminate="somePageSelected && !allPageSelected" :disabled="!pageRegistries.length || busy || loading" aria-label="选择当前页全部仓库" @change="togglePageSelection(Boolean($event))" /></th>
            <th v-for="column in registryColumns" :key="column.key" scope="col" :aria-sort="sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none'"><button class="registry-sort" :disabled="busy" @click="changeSort(column.key)">{{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" /></button></th>
            <th scope="col">操作</th>
          </tr></thead>
          <tbody><tr v-for="registry in pageRegistries" :key="registry.id" :class="{ selected: selectedIds.has(registry.id) }">
            <td class="registry-selection"><el-checkbox :model-value="selectedIds.has(registry.id)" :disabled="busy || loading" :aria-label="`选择仓库 ${registry.name}`" @change="toggleSelection(registry.id, Boolean($event))" /></td>
            <td><div class="registry-identity"><router-link :to="{ name: 'RegistryDetail', params: { id: registry.id }, query: listQuery }" :title="registry.name">{{ registry.name }}</router-link><span v-if="registry.is_default" class="registry-badge">默认</span></div></td>
            <td>{{ registryType(registry.url) }}</td>
            <td><span class="registry-url" :title="registry.url">{{ registry.url }}</span><span v-if="registry.insecure" class="registry-insecure">跳过 TLS 验证</span></td>
            <td><span :class="registry.username ? 'registry-true' : 'registry-muted'">{{ registry.username ? '已启用' : '匿名访问' }}</span><span v-if="registry.username" class="registry-username" :title="registry.username">{{ registry.username }}</span></td>
            <td><router-link class="registry-browse" :to="{ name: 'RegistryImages', params: { id: registry.id }, query: listQuery }"><PortainerIcon name="box" />浏览镜像</router-link></td>
          </tr></tbody>
        </table>
        <div v-if="!pageRegistries.length && !loading" class="registry-empty" role="status"><PortainerIcon :name="hasFilters ? 'search' : 'registry'" /><p>{{ listError ? '暂时无法加载仓库列表' : hasFilters ? '没有找到匹配的仓库' : '暂无外部仓库' }}</p><el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button><span v-else-if="!listError">添加 Registry 后，可在这里管理连接并浏览镜像</span></div>
      </div>
      <footer class="registries-pagination">
        <div class="registry-count" role="status"><span>共 {{ filteredRegistries.length }} 个仓库</span><template v-if="selectedRegistries.length"><span class="registry-selected-count">已选 {{ selectedRegistries.length }} 项</span><el-button link :disabled="busy" @click="selectedIds = new Set()">清除选择</el-button></template></div>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredRegistries.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" />
      </footer>
    </section>
    <RegistryDeleteDialog v-model="deleteVisible" :registries="selectedRegistries" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { registryApi, type Registry } from '@/api/registry'
import PortainerIcon from '@/components/PortainerIcon.vue'
import RegistryDeleteDialog from '@/components/registries/RegistryDeleteDialog.vue'
import { filterRegistries, readRegistryListState, registryColumns, registryError, registryListQuery, registryProviders, registryType, sortRegistries, type RegistrySortKey } from '@/utils/registries'
import '@/styles/registries.css'

const router = useRouter()
const initial = readRegistryListState(useRoute().query)
const registries = ref<Registry[]>([])
const loading = ref(false), listError = ref(''), deleteVisible = ref(false)
const search = ref(initial.search), typeFilter = ref(initial.type), authFilter = ref(initial.authentication), defaultFilter = ref(initial.defaults)
const page = ref(initial.page), pageSize = ref(initial.size), sortKey = ref<RegistrySortKey>(initial.sort), descending = ref(initial.descending)
const selectedIds = ref(new Set<number>())
const tableScroll = ref<HTMLElement>()
const busy = computed(() => deleteVisible.value)
const hasFilters = computed(() => Boolean(search.value || typeFilter.value || authFilter.value || defaultFilter.value))
const listQuery = computed(() => registryListQuery({ search: search.value, type: typeFilter.value, authentication: authFilter.value, defaults: defaultFilter.value, page: page.value, size: pageSize.value, sort: sortKey.value, descending: descending.value }))
const filteredRegistries = computed(() => sortRegistries(filterRegistries(registries.value, search.value, typeFilter.value, authFilter.value, defaultFilter.value), sortKey.value, descending.value))
const pageRegistries = computed(() => filteredRegistries.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const selectedRegistries = computed(() => registries.value.filter(item => selectedIds.value.has(item.id)))
const allPageSelected = computed(() => pageRegistries.value.length > 0 && pageRegistries.value.every(item => selectedIds.value.has(item.id)))
const somePageSelected = computed(() => pageRegistries.value.some(item => selectedIds.value.has(item.id)))
const controller = new AbortController()
let disposed = false
watch([search, typeFilter, authFilter, defaultFilter], () => { page.value = 1; selectedIds.value = new Set() })
watch(pageSize, () => { page.value = 1 })
watch(() => filteredRegistries.value.length, () => { if (!loading.value) clampPage() })
watch([page, pageSize, sortKey, descending, search, typeFilter, authFilter, defaultFilter], () => tableScroll.value?.scrollTo({ top: 0 }))
function clampPage() { page.value = Math.min(page.value, Math.max(1, Math.ceil(filteredRegistries.value.length / pageSize.value))) }
function resetFilters() { search.value = ''; typeFilter.value = ''; authFilter.value = ''; defaultFilter.value = '' }
function toggleSelection(id: number, checked: boolean) { const next = new Set(selectedIds.value); if (checked) next.add(id); else next.delete(id); selectedIds.value = next }
function togglePageSelection(checked: boolean) { for (const item of pageRegistries.value) toggleSelection(item.id, checked) }
function changeSort(key: RegistrySortKey) { descending.value = sortKey.value === key ? !descending.value : false; sortKey.value = key; page.value = 1 }
function onRemoved(ids: number[]) { const removed = new Set(ids); registries.value = registries.value.filter(item => !removed.has(item.id)); selectedIds.value = new Set([...selectedIds.value].filter(id => !removed.has(id))) }
async function loadRegistries() {
  if (loading.value || disposed) return
  loading.value = true; listError.value = ''
  try {
    const result = await registryApi.list(controller.signal)
    if (disposed) return
    registries.value = (result || []).filter(item => item.kind !== 'builtin')
    const existing = new Set(registries.value.map(item => item.id))
    selectedIds.value = new Set([...selectedIds.value].filter(id => existing.has(id)))
    clampPage()
  } catch (error) { if (!disposed) listError.value = `仓库列表加载失败：${registryError(error)}` }
  finally { loading.value = false }
}
onMounted(loadRegistries)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
