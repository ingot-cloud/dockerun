<template>
  <section class="management-card resource-list" :aria-label="title">
    <header class="management-toolbar">
      <h2><span class="management-section-icon"><el-icon><component :is="icon || 'List'" /></el-icon></span>{{ title }}</h2>
      <div class="management-actions">
        <el-input v-model="state.search" class="management-search" :placeholder="placeholder || '搜索名称或说明…'" :aria-label="`搜索${title}`" clearable :disabled="busy"><template #prefix><PortainerIcon name="search" /></template></el-input>
        <slot name="actions" />
      </div>
    </header>
    <div class="management-filters">
      <el-select v-if="filterOptions.length" v-model="state.filter" :placeholder="filterLabel" :aria-label="`筛选${title}`" :disabled="busy"><el-option :label="filterLabel" value="" /><el-option v-for="option in filterOptions" :key="option.value" :label="option.label" :value="option.value" /></el-select>
      <el-select v-model="state.sort" :aria-label="`${title}排序字段`" :disabled="busy"><el-option v-for="sort in sorts" :key="sort.key" :label="`按${sort.label}排序`" :value="sort.key" /></el-select>
      <el-button text :disabled="busy" :aria-label="state.descending ? '改为升序排列' : '改为降序排列'" @click="state.descending = !state.descending"><PortainerIcon name="sort" />{{ state.descending ? '降序' : '升序' }}</el-button>
      <el-button v-if="hasFilters" link :disabled="busy" @click="reset">重置筛选</el-button>
      <span class="management-summary"><slot name="summary">全部 {{ items.length }} 项</slot></span>
    </div>
    <div v-if="error" class="management-error" role="alert"><span>{{ error }}</span><el-button link :disabled="loading || busy" @click="emit('refresh')">重试</el-button></div>
    <slot name="feedback" />
    <div ref="scroll" v-loading="loading" class="resource-list-table" :aria-label="`${title}滚动区域`" tabindex="0" :aria-busy="loading">
      <el-table ref="table" :data="pageItems" :row-key="rowKey" height="100%" :empty-text="loading ? '加载中…' : error ? '暂时无法加载数据' : hasFilters ? '没有找到匹配的数据' : emptyText" @selection-change="emit('selection-change', $event)" @sort-change="onSort">
        <slot :items="pageItems" :query="listQuery" />
        <template #empty><div v-if="!loading" class="management-empty"><PortainerIcon :name="hasFilters ? 'search' : 'list'" /><p>{{ error ? '暂时无法加载数据' : hasFilters ? '没有找到匹配的数据' : emptyText }}</p><el-button v-if="hasFilters" link @click="reset">重置筛选</el-button><slot v-else-if="!error" name="empty" /></div></template>
      </el-table>
    </div>
    <footer class="management-pagination">
      <span role="status">共 {{ filtered.length }} 项<span v-if="hasFilters" class="management-muted">（全部 {{ items.length }} 项）</span><slot name="count" /></span>
      <el-pagination v-model:current-page="state.page" v-model:page-size="state.size" :page-sizes="[10, 25, 50, 100]" :total="filtered.length" :pager-count="5" :disabled="busy || loading" layout="sizes, prev, pager, next" />
    </footer>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { TableInstance } from 'element-plus'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { fieldValue, filterAndSort, readListState } from '@/utils/management'
const props = withDefaults(defineProps<{ title: string; icon?: string; items: any[]; loading?: boolean; busy?: boolean; error?: string; placeholder?: string; emptyText?: string; searchKeys?: string[]; sorts?: {key: string; label: string}[]; filterOptions?: {value: string; label: string}[]; filterKey?: string; filterBy?: (item: any) => string; filterLabel?: string; queryPrefix?: string; rowKey?: string }>(), { emptyText: '暂无数据', searchKeys: () => ['name', 'description'], sorts: () => [{key: 'name', label: '名称'}], filterOptions: () => [], filterLabel: '全部状态', queryPrefix: '', rowKey: 'id' })
const emit = defineEmits<{ refresh: []; 'selection-change': [items: any[]] }>()
const route = useRoute(), router = useRouter()
const state = reactive(readListState(route.query, props.queryPrefix, props.sorts.map(sort => sort.key), props.filterOptions.map(filter => filter.value)))
const scroll = ref<HTMLElement>(), table = ref<TableInstance>()
const hasFilters = computed(() => Boolean(state.search || state.filter))
const filtered = computed(() => filterAndSort(props.items, state.search, props.searchKeys, state.sort, state.descending).filter(item => !state.filter || (props.filterBy ? props.filterBy(item) : String(fieldValue(item, props.filterKey || 'status'))) === state.filter))
const pageItems = computed(() => filtered.value.slice((state.page - 1) * state.size, state.page * state.size))
let disposed = false
watch(() => [state.search, state.filter, state.sort, state.descending, state.size], () => { state.page = 1; clearSelection() })
watch(() => filtered.value.length, total => { if (!props.loading) state.page = Math.min(state.page, Math.max(1, Math.ceil(total / state.size))) })
watch(() => props.loading, loading => { if (!loading) state.page = Math.min(state.page, Math.max(1, Math.ceil(filtered.value.length / state.size))) })
watch(pageItems, () => { clearSelection(); table.value?.setScrollTop(0); scroll.value?.scrollTo({top:0}) })
const listQuery = computed(() => {
  const query={...route.query}
  const values={search:state.search,filter:state.filter,sort:state.sort,direction:state.descending?'desc':'asc',page:state.page>1?String(state.page):'',size:state.size!==10?String(state.size):''}
  for(const [key,value] of Object.entries(values)){if(value)query[props.queryPrefix+key]=value;else delete query[props.queryPrefix+key]}
  return query
})
watch(state,()=>{if(!disposed && JSON.stringify(listQuery.value)!==JSON.stringify(route.query))void router.replace({query:listQuery.value})},{flush:'post'})
function reset() { state.search = ''; state.filter = '' }
function clearSelection() { table.value?.clearSelection(); emit('selection-change', []) }
function onSort(value: {prop: string; order: string | null}) { if (!value.order) return; state.sort = value.prop; state.descending = value.order === 'descending' }
onBeforeUnmount(() => { disposed = true })
defineExpose({clearSelection})
</script>
