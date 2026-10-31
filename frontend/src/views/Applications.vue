<template>
  <div class="workspace-page">
    <WorkspaceHeading title="应用" :loading="loading" :busy="busy" @refresh="load()" />
    <ResourceList title="应用列表" icon="Grid" :items="rows" :loading="loading" :error="error" :busy="busy" selection :search-keys="['name','description','id']" :sorts="[{key:'name',label:'名称'},{key:'created_at',label:'创建时间'}]" :filter-options="[{value:'business',label:'业务应用'},{value:'middleware',label:'中间件'}]" :filter-by="row=>row.category || 'business'" filter-label="全部类型" empty-text="暂无应用" @refresh="load()" @selection-change="selected=$event">
      <template #actions><el-button type="danger" :disabled="!selected.length || busy || loading" :title="selected.length ? '删除选中的应用' : '请先勾选要删除的应用'" @click="openDeleteTargets(selected)">批量删除</el-button><el-button :disabled="loading || busy" @click="router.push('/application-templates')">从模板创建</el-button><el-button type="primary" :disabled="loading || busy" @click="openCreate"><PortainerIcon name="plus" />创建应用</el-button></template>
      <template #default="{query}"><el-table-column label="应用名称" prop="name" min-width="180"><template #default="{row}"><router-link :to="{path:`/applications/${row.id}`,query}">{{ row.name }}</router-link></template></el-table-column><el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip /><el-table-column label="应用类型" width="120"><template #default="{row}">{{templateCategory(row.category)}}</template></el-table-column><el-table-column label="部署方式" width="150"><template #default="{row}">{{ sourceLabel(row.kind) }}</template></el-table-column><el-table-column label="创建时间" width="180"><template #default="{row}">{{ formatDate(row.created_at) }}</template></el-table-column><el-table-column label="操作" width="100"><template #default="{row}"><el-button link type="danger" :disabled="busy" :aria-label="`删除应用 ${row.name}`" @click="openDelete(row)">删除</el-button></template></el-table-column></template>
      <template #empty><span>创建应用后，配置部署环境与发布版本。</span></template>
    </ResourceList>
    <FormDialog v-model="visible" :draft="form" title="创建应用" :loading="saving" confirm-text="创建并配置环境" @confirm="create" @cancel="visible=false">
      <ApplicationSetupSteps :step="0" />
      <p class="management-description application-create-help">创建应用后，继续选择部署主机、配置首个环境，再准备镜像并首次发布。</p>
      <div v-if="createError" class="management-error management-feedback" role="alert">{{ createError }}</div>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="saving" @submit.prevent="create"><el-form-item label="应用名称" prop="name"><el-input v-model="form.name" placeholder="输入应用名称" maxlength="100" /></el-form-item><el-form-item label="说明"><el-input v-model="form.description" type="textarea" placeholder="说明应用的用途（可选）" :rows="3" /></el-form-item><button type="submit" hidden>创建应用</button></el-form>
    </FormDialog>
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="应用" endpoint="/applications" warning="将删除应用及其环境、发布版本、部署记录和日志，无法撤销。已部署的容器、网络、卷和镜像会保留；删除后无法继续通过此应用管理或回滚。排队、执行或恢复中的发布任务会阻止删除。" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { api,errorText } from '@/api/workspace'
import { useResource } from '@/composables/useResource'
import { templateCategory } from '@/utils/applicationTemplates'
import { formatDate,sourceLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import FormDialog from '@/components/FormDialog.vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import ApplicationSetupSteps from '@/components/ApplicationSetupSteps.vue'
import { workspaceDeleteTargets } from '@/utils/workspaceDeletion'
const router=useRouter(),route=useRoute()
const {data:applicationRows,loading,error,load}=useResource<any[]>('/applications',[])
const deleteVisible=ref(false),deleteTargets=ref<{id:number;name:string}[]>([]),removedIds=ref(new Set<number>()),selected=ref<{id:number;name:string}[]>([])
const rows=computed(()=>applicationRows.value.filter(row=>!removedIds.value.has(row.id)))
const visible=ref(false),saving=ref(false),createError=ref(''),formRef=ref<FormInstance>(),form=ref({name:'',description:''})
const rules={name:[{required:true,whitespace:true,message:'请输入应用名称',trigger:'blur'}]}
const busy=computed(()=>saving.value || deleteVisible.value)
function openDeleteTargets(items:{id:number;name:string}[]){if(busy.value)return;const targets=workspaceDeleteTargets(items,row=>row.id,row=>row.name);if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function openDelete(row:{id:number;name:string}){openDeleteTargets([row])}
function onRemoved(ids:number[]){removedIds.value=new Set([...removedIds.value,...ids])}
function openCreate(){form.value={name:'',description:''};createError.value='';formRef.value?.clearValidate();visible.value=true}
async function create(){if(saving.value || !await formRef.value?.validate().catch(()=>false) || saving.value)return;saving.value=true;createError.value='';try{const result=await api('POST','/applications',{...form.value,name:form.value.name.trim()});visible.value=false;ElMessage.success('应用已创建，接下来配置首个环境');await router.push({path:`/applications/${result.id}`,query:{...route.query,app_view:'environments',setup:'environment'}})}catch(cause){createError.value=errorText(cause)}finally{saving.value=false}}
</script>
<style scoped>
.application-create-help { margin:12px 0 20px; }
</style>
