<template>
  <WorkspaceEditor :title="template?`从 ${template.name} 创建应用`:'从模板创建应用'" :loading="loading" :busy="saving" @refresh="load">
    <template #navigation><ApplicationSetupSteps :step="created?2:0" /></template>
    <template v-if="template" #actions><router-link v-if="created" :to="`/applications/${created.application.id}`">查看已创建应用</router-link><el-button type="primary" :disabled="loading" :loading="saving" @click="create">{{created?'继续准备发布':'创建并准备发布'}}</el-button></template>
    <template v-if="error || actionError" #feedback>
      <div v-if="error" class="management-error" role="alert"><span>{{error}}</span><el-button link :disabled="saving" @click="load">重试</el-button></div>
      <div v-if="actionError" class="management-error" role="alert">{{actionError}}</div>
    </template>
  <div class="workspace-page workspace-editor-page template-create-page">
    <template v-if="template">
      <section class="panel"><h2>{{template.name}} · {{template.version}}</h2><p class="management-description">{{template.description}}</p><p class="management-form-help">{{template.definition.instructions}}</p><dl class="template-notes"><div v-if="template.definition.storage"><dt>持久化</dt><dd>{{template.definition.storage}}</dd></div><div v-if="template.definition.initialization"><dt>首次初始化</dt><dd>{{template.definition.initialization}}</dd></div><div><dt>更新方式</dt><dd>默认手动发布。首次发布成功后，可在环境中单独配置镜像自动发布。</dd></div></dl></section>
      <el-form class="template-create-form" label-position="top" :disabled="saving || !!created" @submit.prevent="create">
        <section class="panel"><h2>应用与首个环境</h2><div class="template-form-grid"><el-form-item label="应用名称" required><el-input v-model="form.name" placeholder="例如：基础服务 Nacos" maxlength="100" /></el-form-item><el-form-item label="环境名称" required><el-input v-model="form.environment" placeholder="例如：测试环境、生产环境" maxlength="100" /></el-form-item><el-form-item label="目标主机" required><el-select v-model="form.target_id" aria-label="目标主机"><el-option v-for="host in hosts" :key="host.id" :label="host.name" :value="host.id" :disabled="!host.enabled" /></el-select><p class="management-form-help">{{template.definition.topology==='single_host'?'整套服务部署在一台主机上。三节点模板不会自动跨主机分布。':'创建后可在环境配置中添加固定主机组。'}}</p></el-form-item><el-form-item label="镜像仓库" required><el-select v-model="form.registry_id" aria-label="镜像仓库" @change="changeRegistry"><el-option v-for="registry in registries" :key="registry.id" :label="registry.name" :value="registry.id" /></el-select><p class="management-form-help">{{selectedRegistry?.kind==='builtin'?'默认使用内置仓库，无需配置 Webhook。':'使用已配置的外部镜像仓库。'}} 镜像须提前存在所选仓库中。</p></el-form-item></div><el-form-item label="应用说明"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item></section>
        <section class="panel"><h2>镜像与运行参数</h2><EnvironmentParameterPairs v-model="parameterPairs" :schema="template.definition.parameters || []" :allow-extra="false" :disabled="saving || !!created" /></section>
      </el-form>
      <section class="panel"><h2>高级配置预览</h2><p class="management-form-help">创建后应用持有独立配置，可在发布向导中修改。凭据仅保存在环境参数中，不写入公共模板。</p><el-collapse><el-collapse-item title="应用栈与非敏感配置文件" name="definition"><pre class="management-code">{{template.definition.compose}}</pre><pre v-if="Object.keys(template.definition.configs || {}).length" class="management-code" style="margin-top:12px">{{JSON.stringify(template.definition.configs,null,2)}}</pre></el-collapse-item><el-collapse-item v-if="hasScripts(template.scripts)" title="部署脚本" name="scripts"><DeploymentScriptEditor :model-value="copyScripts(template.scripts)" disabled /></el-collapse-item></el-collapse></section>
    </template>
    <p v-else-if="loading" class="management-description">正在加载模板和目标主机…</p>
  </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { computed,onBeforeUnmount,onMounted,ref } from 'vue'
import { useRoute,useRouter } from 'vue-router'
import { applicationTemplatesApi,type ApplicationTemplate } from '@/api/applicationTemplates'
import { api,errorText } from '@/api/workspace'
import { copyScripts,defaultReleaseRegistry,hasScripts,type ReleaseRegistry } from '@/utils/releases'
import { templateValues,validateTemplateValues,retargetTemplateParameterPairs } from '@/utils/applicationTemplates'
import { environmentParameterPairs,environmentParameterValues,type EnvironmentParameterPair } from '@/utils/applicationDetail'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import ApplicationSetupSteps from '@/components/ApplicationSetupSteps.vue'
import EnvironmentParameterPairs from '@/components/EnvironmentParameterPairs.vue'
import DeploymentScriptEditor from '@/components/DeploymentScriptEditor.vue'
import { useUnsavedChanges } from '@/composables/useUnsavedChanges'
const route=useRoute(),router=useRouter(),template=ref<ApplicationTemplate>(),registries=ref<ReleaseRegistry[]>([]),hosts=ref<any[]>([]),loading=ref(false),saving=ref(false),error=ref(''),actionError=ref(''),created=ref<any>()
const form=ref({name:'',description:'',environment:'默认环境',target_id:1,registry_id:0}),parameterPairs=ref<EnvironmentParameterPair[]>([]),controller=new AbortController();let disposed=false,initialized=false,previousRegistry=0
const selectedRegistry=computed(()=>registries.value.find(item=>item.id===form.value.registry_id))
const { markSaved } = useUnsavedChanges(() => ({ form: form.value, parameters: parameterPairs.value.map(({key,value})=>({key,value})) }), { enabled: () => !!template.value && !created.value, busy: () => saving.value })
async function load(){if(loading.value || saving.value)return;loading.value=true;error.value='';try{const [items,regs,nodes]=await Promise.all([applicationTemplatesApi.list(controller.signal),api<ReleaseRegistry[]>('GET','/registries',undefined,controller.signal),api<any[]>('GET','/targets',undefined,controller.signal)]);if(disposed)return;const item=items.find(item=>item.id===route.params.id);if(!item)throw new Error('应用模板不存在，请返回模板列表重新选择');template.value=item;registries.value=regs;hosts.value=nodes;if(!initialized){form.value.registry_id=defaultReleaseRegistry(regs);form.value.name=item.name;form.value.description=item.description;parameterPairs.value=environmentParameterPairs(templateValues(item,selectedRegistry.value),item.definition.parameters || []);previousRegistry=form.value.registry_id;initialized=true;markSaved()}}catch(cause){if(!disposed)error.value=errorText(cause)}finally{loading.value=false}}
function changeRegistry(){if(!template.value)return;parameterPairs.value=retargetTemplateParameterPairs(template.value.definition.parameters || [],parameterPairs.value,registries.value.find(item=>item.id===previousRegistry),selectedRegistry.value);previousRegistry=form.value.registry_id}
async function create(){if(saving.value || !template.value)return;actionError.value='';try{if(!form.value.name.trim() || !form.value.environment.trim())throw new Error('请填写应用名称和环境名称');const schema=template.value.definition.parameters || [],values=environmentParameterValues(parameterPairs.value,schema);validateTemplateValues(schema,values);saving.value=true;if(!created.value)created.value=await applicationTemplatesApi.create(template.value.id,{...form.value,template_version:template.value.version,name:form.value.name.trim(),environment:form.value.environment.trim(),parameters:values});markSaved();saving.value=false;await router.replace({path:`/applications/${created.value.application.id}/deploy`,query:{environment:String(created.value.environment.id),prepare:'template'}})}catch(cause){actionError.value=errorText(cause)}finally{saving.value=false}}
onMounted(load);onBeforeUnmount(()=>{disposed=true;controller.abort()})
</script>
<style scoped>
.template-create-page :deep(.el-select) { width:100%; }
.template-create-form { display:flex;flex-direction:column;gap:16px; }
.template-form-grid { display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 20px; }
.template-notes { margin:16px 0 0;line-height:20px;font-size:12px; }
.template-notes > div { display:grid;grid-template-columns:90px minmax(0,1fr);gap:12px;margin-top:10px; }
.template-notes dt { color:#aaa; }.template-notes dd { margin:0;overflow-wrap:anywhere; }
@media(max-width:760px){.template-create-form{gap:12px}.template-form-grid{grid-template-columns:minmax(0,1fr)}.template-notes > div{grid-template-columns:minmax(0,1fr);gap:4px}}
</style>
