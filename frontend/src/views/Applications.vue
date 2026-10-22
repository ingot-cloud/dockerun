<template>
  <div class="workspace-page">
    <WorkspaceHeading title="应用" :loading="loading" :busy="saving" @refresh="load()" />
    <ResourceList title="应用列表" icon="Grid" :items="rows" :loading="loading" :error="error" :busy="saving" :search-keys="['name','description','id']" :sorts="[{key:'name',label:'名称'},{key:'created_at',label:'创建时间'}]" empty-text="暂无应用" @refresh="load()">
      <template #actions><el-button type="primary" :disabled="loading" @click="openCreate"><PortainerIcon name="plus" />创建应用</el-button></template>
      <template #default="{query}"><el-table-column label="应用名称" prop="name" min-width="180"><template #default="{row}"><router-link :to="{path:`/applications/${row.id}`,query}">{{ row.name }}</router-link></template></el-table-column><el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip /><el-table-column label="部署方式" width="150"><template #default="{row}">{{ sourceLabel(row.kind) }}</template></el-table-column><el-table-column label="创建时间" width="180"><template #default="{row}">{{ formatDate(row.created_at) }}</template></el-table-column></template>
      <template #empty><span>创建应用后，配置部署环境与发布版本。</span></template>
    </ResourceList>
    <FormDialog v-model="visible" title="创建应用" :loading="saving" confirm-text="创建并配置环境" @confirm="create" @cancel="visible=false">
      <div v-if="createError" class="management-error management-feedback" role="alert">{{ createError }}</div>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="saving" @submit.prevent="create"><el-form-item label="应用名称" prop="name"><el-input v-model="form.name" placeholder="输入应用名称" maxlength="100" /></el-form-item><el-form-item label="说明"><el-input v-model="form.description" type="textarea" placeholder="说明应用的用途（可选）" :rows="3" /></el-form-item><button type="submit" hidden>创建应用</button></el-form>
    </FormDialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { api,errorText } from '@/api/workspace'
import { useResource } from '@/composables/useResource'
import { formatDate,sourceLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import FormDialog from '@/components/FormDialog.vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
const router=useRouter(),route=useRoute()
const {data:rows,loading,error,load}=useResource<any[]>('/applications',[])
const visible=ref(false),saving=ref(false),createError=ref(''),formRef=ref<FormInstance>(),form=ref({name:'',description:''})
const rules={name:[{required:true,whitespace:true,message:'请输入应用名称',trigger:'blur'}]}
function openCreate(){form.value={name:'',description:''};createError.value='';formRef.value?.clearValidate();visible.value=true}
async function create(){if(saving.value || !await formRef.value?.validate().catch(()=>false) || saving.value)return;saving.value=true;createError.value='';try{const result=await api('POST','/applications',{...form.value,name:form.value.name.trim()});visible.value=false;ElMessage.success('应用已创建');await router.push({path:`/applications/${result.id}`,query:route.query})}catch(cause){createError.value=errorText(cause)}finally{saving.value=false}}
</script>
