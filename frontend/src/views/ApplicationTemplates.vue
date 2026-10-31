<template>
  <div class="workspace-page">
    <WorkspaceHeading title="应用模板" :loading="loading" :busy="deleteVisible" @refresh="load()" />
    <p class="management-description">选择模板，填写镜像和环境参数后创建应用并发布。中间件默认手动更新，模板更新不会修改已有应用。</p>
    <ResourceList title="应用模板列表" icon="Collection" :items="items" :loading="loading" :error="error" :busy="deleteVisible" selection :selectable="row=>!templateDeletionReason(row)" :search-keys="['name','description','version']" :sorts="[{key:'name',label:'名称'},{key:'version',label:'版本'}]" :filter-options="[{value:'business',label:'业务应用'},{value:'middleware',label:'中间件'}]" filter-key="category" filter-label="全部类型" empty-text="暂无应用模板" @refresh="load()" @selection-change="selected=$event">
      <template #actions><el-button type="danger" :disabled="!selected.length || deleteVisible || loading" :title="selected.length ? '删除选中的自建模板' : '请先勾选要删除的自建模板'" @click="openDeleteTargets(selected)">批量删除</el-button></template>
      <template #default="{query}"><el-table-column label="模板名称" min-width="220"><template #default="{row}"><router-link :to="{path:`/application-templates/${row.id}/create`,query}">{{row.name}}</router-link><div class="management-muted" style="font-size:11px">{{row.source==='custom'?'自定义模板':'内置模板'}} · {{row.version}}</div></template></el-table-column><el-table-column label="类型" width="120"><template #default="{row}">{{templateCategory(row.category)}}</template></el-table-column><el-table-column prop="description" label="用途" min-width="300" show-overflow-tooltip /><el-table-column label="部署方式" width="160"><template #default="{row}">{{row.definition.topology==='single_host'?'单台目标主机':'单机或固定主机组'}}</template></el-table-column><el-table-column label="操作" width="200"><template #default="{row}"><div class="management-row-actions"><router-link :to="{path:`/application-templates/${row.id}/create`,query}">创建应用</router-link><el-tooltip :content="templateDeletionReason(row)" :disabled="!templateDeletionReason(row)"><span><el-button link type="danger" :disabled="deleteVisible || !!templateDeletionReason(row)" :aria-label="`删除应用模板 ${row.name}`" @click="openDelete(row)">删除</el-button></span></el-tooltip></div></template></el-table-column></template>
    </ResourceList>
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="应用模板" endpoint="/managed-templates" warning="将删除所选自建模板的指定版本，无法撤销。已有应用使用独立的配置快照，不受影响。" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import { useResource } from '@/composables/useResource'
import { templateCategory } from '@/utils/applicationTemplates'
import type { ApplicationTemplate } from '@/api/applicationTemplates'
import { templateDeletionID,templateDeletionReason,workspaceDeleteTargets } from '@/utils/workspaceDeletion'
const {data:templateRows,loading,error,load}=useResource<ApplicationTemplate[]>('/managed-templates',[])
const deleteVisible=ref(false),deleteTargets=ref<{id:number;name:string}[]>([]),removedIds=ref(new Set<number>()),selected=ref<ApplicationTemplate[]>([])
const items=computed(()=>templateRows.value.filter(row=>!removedIds.value.has(templateDeletionID(row) || 0)))
function openDeleteTargets(rows:ApplicationTemplate[]){if(deleteVisible.value)return;const targets=workspaceDeleteTargets(rows,templateDeletionID,row=>`${row.name} · ${row.version}`);if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function openDelete(row:ApplicationTemplate){openDeleteTargets([row])}
function onRemoved(ids:number[]){removedIds.value=new Set([...removedIds.value,...ids])}
</script>
