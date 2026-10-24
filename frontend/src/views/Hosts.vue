<template>
  <div class="workspace-page">
    <WorkspaceHeading title="目标主机" :loading="loading" :busy="busy" @refresh="load()" />
    <ResourceList title="主机列表" icon="Monitor" :items="rows" :loading="loading" :error="error" :busy="busy" :search-keys="['name','platform','id']" :sorts="[{key:'name',label:'名称'},{key:'last_seen_at',label:'最近连接'}]" :filter-options="states" :filter-by="row=>hostState(row)" empty-text="暂无目标主机" @refresh="load()">
      <template #actions><el-button type="primary" @click="openCreate">添加主机</el-button></template>
      <el-table-column prop="name" label="主机名称" min-width="190" show-overflow-tooltip /><el-table-column label="平台" min-width="150"><template #default="{row}">{{row.platform || '—'}}</template></el-table-column><el-table-column label="连接状态" width="130"><template #default="{row}"><StatusBadge :value="hostState(row)" /></template></el-table-column><el-table-column label="最近连接" width="180"><template #default="{row}">{{formatDate(row.last_seen_at)}}</template></el-table-column><el-table-column label="操作" width="140"><template #default="{row}"><span v-if="row.id===1" class="management-muted">本机无需凭据</span><el-button v-else link :type="row.enabled?'danger':'primary'" :disabled="busy" @click="toggle(row)">{{row.enabled?'停用凭据':'启用凭据'}}</el-button></template></el-table-column>
    </ResourceList>
    <p class="management-description">本机可独立使用。远端代理（Agent）主动连接控制台获取任务，停用凭据后无法继续领取任务。</p>
    <FormDialog v-model="visible" :draft="credential ? undefined : form" title="添加目标主机" :loading="busy" :show-footer="!credential" confirm-text="创建连接凭据" @confirm="create" @cancel="visible=false">
      <div v-if="formError" class="management-error management-feedback" role="alert">{{formError}}</div>
      <template v-if="!credential"><el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="create"><el-form-item label="主机名称" prop="name"><el-input v-model="form.name" placeholder="输入便于识别的主机名称" maxlength="100" :disabled="busy" /></el-form-item><button type="submit" hidden>创建连接凭据</button></el-form><p class="management-description">凭据仅在创建时展示，请复制到目标主机的运行环境中。</p></template>
      <div v-else class="management-secret"><p>凭据已创建，仅展示一次。请妥善保存完整命令，勿分享给他人。</p><pre class="management-code">{{credential}}</pre><el-button @click="copyCredential">复制连接命令</el-button><el-button @click="credential='';visible=false">已保存，关闭</el-button></div>
    </FormDialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage,ElMessageBox,type FormInstance } from 'element-plus'
import { useClipboard } from '@vueuse/core'
import { api,errorText } from '@/api/workspace'
import { useResource } from '@/composables/useResource'
import { formatDate,hostState,statusLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import FormDialog from '@/components/FormDialog.vue'
import StatusBadge from '@/components/StatusBadge.vue'
const {data:rows,loading,error,load}=useResource<any[]>('/targets',[],{interval:30000,active:()=>true})
const busy=ref(false),visible=ref(false),credential=ref(''),formError=ref(''),form=ref({name:''}),formRef=ref<FormInstance>()
const {copy}=useClipboard({legacy:true}), rules={name:[{required:true,whitespace:true,message:'请输入主机名称',trigger:'blur'}]}
const states=['local','online','offline','disabled'].map(value=>({value,label:statusLabel(value)}))
function openCreate(){credential.value='';formError.value='';form.value.name='';visible.value=true}
async function copyCredential(){try{await copy(credential.value);ElMessage.success('连接命令已复制')}catch{ElMessage.error('复制失败，请重试')}}
async function create(){if(busy.value || !await formRef.value?.validate().catch(()=>false) || busy.value)return;busy.value=true;formError.value='';try{const result=await api('POST','/targets',{name:form.value.name.trim()});credential.value=`DOCKERUN_URL=${location.origin}\nDOCKERUN_AGENT_TOKEN=${result.token}\ndockerun agent --data-dir /var/lib/dockerun-agent`;await load();ElMessage.success('主机凭据已创建')}catch(cause){formError.value=errorText(cause)}finally{busy.value=false}}
async function toggle(row:any){if(busy.value)return;busy.value=true;try{await ElMessageBox.confirm(`确定${row.enabled?'停用':'启用'}「${row.name}」的连接凭据？${row.enabled?'停用后代理将无法领取新任务。':'启用后代理可重新连接并领取任务。'}`,row.enabled?'停用主机凭据':'启用主机凭据',{confirmButtonText:'确认',cancelButtonText:'取消',type:'warning'})}catch{busy.value=false;return}try{await api('PATCH',`/targets/${row.id}`,{enabled:!row.enabled});ElMessage.success('主机凭据已更新');await load()}catch(cause){error.value=errorText(cause)}finally{busy.value=false}}
</script>
