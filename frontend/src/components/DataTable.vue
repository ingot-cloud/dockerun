<template>
  <div class="data-table-wrapper">
    <div class="data-table-scroll" tabindex="0" aria-label="数据列表滚动区域">
      <el-table ref="table" v-bind="tableAttrs" :data="pageRows" class="data-table" height="100%" :row-key="String($attrs['row-key'] || 'id')" :row-class-name="rowClassName" empty-text="暂无数据" @sort-change="sortChange" @selection-change="emit('selection-change',$event)"><slot /></el-table>
    </div>
    <footer v-if="showFooter" class="management-pagination"><slot name="footer"><span role="status">共 {{ total ?? rows.length }} {{ totalText }}</span><el-pagination v-model:current-page="state.page" v-model:page-size="state.size" :page-sizes="[10,25,50,100]" :total="rows.length" :pager-count="5" layout="sizes, prev, pager, next" /></slot></footer>
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref, useAttrs, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { TableInstance } from 'element-plus'
import { filterAndSort, readListState } from '@/utils/management'
defineOptions({inheritAttrs:false})
const props=withDefaults(defineProps<{total?:number; totalText?:string; showFooter?:boolean; rowClassName?: (row:any,rowIndex:number)=>string}>(), {totalText:'项',showFooter:true})
const emit=defineEmits<{'selection-change':[rows:any[]]}>()
const attrs=useAttrs(), route=useRoute(), router=useRouter(), table=ref<TableInstance>()
const tableAttrs=computed(()=>{const {data, 'onSelectionChange': _selection, ...rest}=attrs; return rest})
const state=reactive(readListState(route.query,'table_',['name','created_at','status','id']))
const rows=computed(()=>filterAndSort((attrs.data as any[]) || [],'', ['name'],state.sort,state.descending))
const pageRows=computed(()=>props.showFooter?rows.value.slice((state.page-1)*state.size,state.page*state.size):rows.value)
watch(()=>[attrs.data,state.size,state.sort,state.descending],()=>{state.page=1;table.value?.clearSelection();emit('selection-change',[])})
watch(()=>state.page,()=>{table.value?.clearSelection();table.value?.setScrollTop(0)})
watch(state,()=>{void router.replace({query:{...route.query,table_page:String(state.page),table_size:String(state.size),table_sort:state.sort,table_direction:state.descending?'desc':'asc'}})},{flush:'post'})
function sortChange(value:{prop:string;order:string|null}){if(value.order){state.sort=value.prop;state.descending=value.order==='descending'}}
</script>
<style scoped>
.data-table-wrapper { display: flex; flex-direction: column; flex: 1; min-height: 0; }.data-table-scroll { flex: 1; min-height: 160px; overflow: auto; }.data-table { min-width: 900px; }.data-table-scroll:focus-visible { outline: 2px solid #33a7d5; outline-offset: -2px; }
</style>
