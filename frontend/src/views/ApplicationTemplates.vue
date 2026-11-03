<template>
  <div class="workspace-page">
    <WorkspaceHeading title="应用模板" :loading="loading" :busy="busy" @refresh="load()" />
    <p class="management-description">选择模板创建应用，也可下载标准 ZIP 或导入其他实例的模板。模板更新不会修改已有应用。</p>
    <ResourceList ref="list" title="应用模板列表" icon="Collection" :items="items" :loading="loading" :error="error" :busy="busy" selection :search-keys="['name','description','version','key']" :sorts="[{key:'name',label:'名称'},{key:'version',label:'版本'}]" :filter-options="[{value:'business',label:'业务应用'},{value:'middleware',label:'中间件'}]" filter-key="category" filter-label="全部类型" empty-text="暂无应用模板" @refresh="load()" @selection-change="selected=$event">
      <template #actions>
        <el-button type="danger" :disabled="!canDeleteSelection || busy || loading" :title="deleteSelectionReason" @click="openDeleteTargets(selected)">批量删除</el-button>
        <el-button :disabled="!selected.length || busy || loading" :loading="downloading && downloadingIDs.length>1" @click="downloadTemplates(selected)">批量下载</el-button>
        <el-button type="primary" :disabled="busy" @click="importVisible=true">导入模板</el-button>
      </template>
      <template #feedback><div v-if="operationError" class="management-error management-feedback" role="alert">{{operationError}}</div></template>
      <template #default="{query}">
        <el-table-column label="模板名称" min-width="210"><template #default="{row}"><router-link :to="{path:`/application-templates/${row.id}/create`,query}">{{row.name}}</router-link><div class="management-muted template-origin">{{templateSource(row.source)}} · {{row.version}}</div></template></el-table-column>
        <el-table-column prop="key" label="模板标识" min-width="180" show-overflow-tooltip />
        <el-table-column label="类型" width="110"><template #default="{row}">{{templateCategory(row.category)}}</template></el-table-column>
        <el-table-column prop="description" label="用途" min-width="250" show-overflow-tooltip />
        <el-table-column label="部署方式" width="155"><template #default="{row}">{{row.definition.topology==='single_host'?'单台目标主机':'单机或固定主机组'}}</template></el-table-column>
        <el-table-column label="操作" width="235"><template #default="{row}"><div class="management-row-actions">
          <router-link :to="{path:`/application-templates/${row.id}/create`,query}">创建应用</router-link>
          <el-button link :disabled="busy" :loading="downloadingIDs.includes(row.id)" :aria-label="`下载应用模板 ${row.name}`" @click="downloadTemplates([row])">下载</el-button>
          <el-tooltip :content="templateDeletionReason(row)" :disabled="!templateDeletionReason(row)"><span><el-button link type="danger" :disabled="busy || !!templateDeletionReason(row)" :aria-label="`删除应用模板 ${row.name}`" @click="openDelete(row)">删除</el-button></span></el-tooltip>
        </div></template></el-table-column>
      </template>
    </ResourceList>
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="应用模板" endpoint="/managed-templates" warning="将删除所选自建或导入模板的指定版本，无法撤销。已有应用使用独立的配置快照，不受影响。" @removed="onRemoved" />
    <TemplateImportDialog v-model="importVisible" @imported="onImported" />
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import { ElMessage } from 'element-plus'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import TemplateImportDialog from '@/components/TemplateImportDialog.vue'
import { useResource } from '@/composables/useResource'
import { templateCategory } from '@/utils/applicationTemplates'
import { templateSource,templateArchiveFilename } from '@/utils/templatePackages'
import { applicationTemplatesApi,type ApplicationTemplate,type TemplateImportResult } from '@/api/applicationTemplates'
import { errorText } from '@/api/workspace'
import { templateDeletionID,templateDeletionReason,workspaceDeleteTargets } from '@/utils/workspaceDeletion'
const {data:templateRows,loading,error,load}=useResource<ApplicationTemplate[]>('/managed-templates',[])
const list=ref<InstanceType<typeof ResourceList>>(),deleteVisible=ref(false),importVisible=ref(false),downloading=ref(false),operationError=ref(''),downloadingIDs=ref<string[]>([])
const deleteTargets=ref<{id:number;name:string}[]>([]),removedIds=ref(new Set<number>()),selected=ref<ApplicationTemplate[]>([])
const busy=computed(()=>deleteVisible.value||importVisible.value||downloading.value)
const items=computed(()=>templateRows.value.filter(row=>!removedIds.value.has(templateDeletionID(row) || 0)))
const canDeleteSelection=computed(()=>selected.value.length>0 && selected.value.every(row=>!templateDeletionReason(row)))
const deleteSelectionReason=computed(()=>!selected.value.length?'请先勾选要删除的自建或导入模板':!canDeleteSelection.value?'所选项包含内置模板，请仅勾选可删除的模板':'删除选中的自建或导入模板')
function openDeleteTargets(rows:ApplicationTemplate[]){if(busy.value)return;const targets=workspaceDeleteTargets(rows,templateDeletionID,row=>`${row.name} · ${row.version}`);if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function openDelete(row:ApplicationTemplate){openDeleteTargets([row])}
function onRemoved(ids:number[]){removedIds.value=new Set([...removedIds.value,...ids])}
async function onImported(result:TemplateImportResult){operationError.value='';ElMessage.success(`已导入 ${result.imported} 个模板${result.skipped?`，跳过 ${result.skipped} 个`:''}`);await load();list.value?.clearSelection()}
async function downloadTemplates(rows:ApplicationTemplate[]){
  if(busy.value||!rows.length)return
  if(rows.length>100){operationError.value='单次最多下载 100 个模板，请减少选择';return}
  const snapshot=[...rows];operationError.value='';downloading.value=true;downloadingIDs.value=snapshot.map(row=>row.id)
  try{
    const blob=await applicationTemplatesApi.export(downloadingIDs.value)
    const url=URL.createObjectURL(blob),anchor=document.createElement('a');anchor.href=url;anchor.download=snapshot.length===1?templateArchiveFilename(snapshot[0].key,snapshot[0].version):templateArchiveFilename();document.body.appendChild(anchor);anchor.click();anchor.remove();setTimeout(()=>URL.revokeObjectURL(url),1000)
  }catch(cause){operationError.value=errorText(cause)}finally{downloading.value=false;downloadingIDs.value=[]}
}
</script>
<style scoped>
.template-origin { font-size:11px; }
</style>
