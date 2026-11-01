<template>
  <WorkspaceEditor fill :title="existing?`部署版本 · ${data.application?.name || '应用'}`:`${editingConfiguration?'编辑部署配置 ·':'发布'} ${data.application?.name || '应用'}`" :loading="loading" :busy="busy" @refresh="loadOptions">
    <template #actions>
      <el-button v-if="preview" :disabled="busy" @click="preview=null">{{existing?'返回选择环境':'继续编辑'}}</el-button>
      <el-button v-else-if="!existing" :type="editingConfiguration?'primary':undefined" :disabled="loading || busy || !selectedEnv || !hasChanges || (locked && !targetParametersDirty)" @click="saveConfiguration">保存配置</el-button>
      <el-button v-if="!preview" :type="existing || !editingConfiguration?'primary':undefined" :loading="busy" :disabled="loading || !selectedEnv || (!!existing && !existingRelease)" @click="preflight">{{existing?'部署此版本':'预览发布'}}</el-button>
      <el-button v-else type="primary" :loading="busy" @click="deploy">{{existing?'确认部署':'确认发布'}}</el-button>
    </template>
    <template #navigation>
      <div class="deployment-context"><label for="deployment-environment">部署环境</label><el-select id="deployment-environment" v-model="environment" :disabled="busy || !!preview" aria-label="部署环境"><el-option v-for="env in data.environments || []" :key="env.id" :value="env.id" :label="env.name" /></el-select><span class="management-muted">{{targetSummary}}</span><span v-if="existingRelease" class="management-muted">发布版本：{{existingRelease.version}}</span></div>
      <div v-if="!preview && !existing" class="deployment-tabs" role="tablist" aria-label="发布配置分组"><button v-for="tab in tabs" :id="`deployment-tab-${tab.key}`" :key="tab.key" type="button" role="tab" :aria-selected="activeTab===tab.key" :aria-controls="`deployment-panel-${tab.key}`" :class="{active:activeTab===tab.key,invalid:invalidTab===tab.key}" @click="activeTab=tab.key">{{tab.label}}<span v-if="tab.key==='parameters'" class="deployment-tab-count">{{targetParameterPairs.length}}</span><span v-if="tab.key==='scripts'" class="deployment-tab-count">{{scriptCount?`已配置 ${scriptCount} 项`:'未配置'}}</span><span v-if="tabDirty(tab.key)" class="deployment-dirty" aria-label="有未保存的修改">•</span></button></div>
      <p v-else-if="existing" class="deployment-preview-heading">可以部署到其他环境，也可以重新部署历史版本。仅需选择目标环境，以下信息只读展示。</p><p v-else class="deployment-preview-heading">确认本次发布的目标、部署内容与脚本。</p>
    </template>
    <template v-if="error" #feedback><el-alert :title="error" type="error" :closable="false" /></template>
    <div v-loading="loading" class="workspace-page workspace-editor-page form-page deployment-workspace" :class="{'deployment-workspace--version':!!existing}">
      <div v-if="!loading && !data.environments?.length" class="management-warning"><span>请先添加部署环境，选择目标主机。</span><router-link :to="{path:`/applications/${id}`,query:route.query}">前往配置环境</router-link></div>
      <template v-if="existing">
        <el-alert v-if="preview" :title="preview.message" type="success" :closable="false" />
        <VersionDeploymentSummary v-if="existingRelease" :version="versionSummary" :environment="selectedEnv" :current-version="currentVersionName" :parameters="versionParameters" :scripts="preview?.scripts || plannedScripts" :container-policy="preview?.container_policy" />
      </template>
      <div v-else v-show="!preview" class="deployment-edit-panels">
        <section v-show="activeTab==='content'" id="deployment-panel-content" role="tabpanel" aria-labelledby="deployment-tab-content" class="panel deployment-editor-panel">
          <div class="deployment-content-settings">
            <div class="deployment-content-heading">
              <div class="deployment-mode-controls">
                <el-radio-group :model-value="displayMode" :disabled="busy || !!existing" aria-label="部署方式" @update:model-value="execution.mode=String($event)"><el-radio-button value="command">命令部署</el-radio-button><el-radio-button value="compose">Compose 部署</el-radio-button></el-radio-group>
                <DeploymentHelp label="查看部署配置说明" title="部署配置说明">
                  <p v-if="copiedFrom">已复制版本 {{copiedFrom}}。保存后更新应用部署配置，原版本保持不变。</p>
                  <p v-else-if="bundleMode">选择或导入完整发布包，使用包内的部署定义。</p>
                  <p>保存配置供后续手动和自动发布使用，不执行部署；历史版本与已提交任务保留原快照。</p>
                </DeploymentHelp>
                <span v-if="copiedFrom" class="deployment-copy-source management-muted" :title="`基于版本 ${copiedFrom}`">基于 {{copiedFrom}}</span>
              </div>
              <el-button v-if="displayMode==='command' && !locked" :disabled="busy || !selectedEnv" @click="insertDockerRun">插入 docker run 示例</el-button>
            </div>
            <OfflineMaterialPicker v-model="bundleId" :show-materials="offlineMode" :kind="form.source==='bundle'?'bundle':'image'" :materials="bundles" :disabled="busy || loading" @imported="onMaterialImported" @busy="importing=$event">
              <template #source><div class="deployment-image-source"><label for="deployment-image-source">镜像来源</label><el-select id="deployment-image-source" v-model="form.source" :disabled="busy" aria-label="镜像来源"><template v-if="displayMode==='command'"><el-option value="direct" label="由部署脚本准备镜像" /><el-option value="local" label="离线镜像归档" /></template><template v-else><el-option value="direct" label="按镜像地址拉取" /><el-option value="registry" label="已配置仓库（锁定镜像摘要）" /><el-option value="local" label="目标主机已有镜像" /></template><el-option value="bundle" :label="displayMode==='command'?'完整发布包中的镜像':'完整发布包'" /></el-select></div></template>
            </OfflineMaterialPicker>
            <div v-if="offlineCommand && offlineImages.length>1" class="deployment-image-source deployment-image-source--primary"><label for="deployment-primary-image">脚本主镜像</label><el-select id="deployment-primary-image" v-model="primaryImage" :disabled="busy" placeholder="选择通过 $DOCKERUN_IMAGE 使用的镜像" aria-label="脚本主镜像"><el-option v-for="image in offlineImages" :key="image.reference" :value="image.reference" :label="image.reference" :title="image.reference" /></el-select></div>
          </div>
          <template v-if="!bundleMode">
            <DeploymentContentEditor v-model="mainContent" v-model:files="displayFiles" :filename="displayMode==='command'?'deploy.sh':'compose.yaml'" :disabled="busy || loading" :readonly="locked" />
            <div class="deployment-references">
              <DeploymentHelp label="查看参数引用说明" title="参数引用说明">
                <template v-if="displayMode==='command'"><p>使用 "$变量名" 引用运行参数，通过 "$DOCKERUN_IMAGE" 使用本次离线镜像或自动发布镜像。</p><p>在高级配置中关联容器并开启系统替换后，后续发布及历史版本部署会自动替换旧容器。</p></template>
                <p v-else>通过 ${变量名} 引用运行参数，支持多服务 Compose 定义。</p>
              </DeploymentHelp>
              <span class="management-muted" :title="referenceSummary">{{referenceSummary}}</span><el-button link type="primary" @click="activeTab='parameters'">查看参数</el-button><el-button v-if="!locked && missingReferences.length" link :disabled="busy" @click="addReferencedParameters">补充引用参数</el-button>
            </div>
          </template>
        </section>
        <section v-show="activeTab==='parameters'" id="deployment-panel-parameters" role="tabpanel" aria-labelledby="deployment-tab-parameters" class="panel deployment-panel-scroll">
          <div class="deployment-parameter-heading"><div class="deployment-section-title"><h2>运行参数</h2><DeploymentHelp label="查看运行参数说明" title="运行参数说明"><p>参数供部署命令、Compose 和前后置脚本使用，预览前会保存到当前环境。</p><p>传入容器时请使用 -e 或 Compose environment。</p><p>名称以字母或下划线开头，可包含数字；参数仅保存在本环境。密码、令牌等敏感参数默认隐藏。</p><p v-if="parameterSchema.length">模板参数名称固定，标记「必填」的参数必须填写；附加参数可添加、修改或删除。</p></DeploymentHelp></div><div class="deployment-parameter-actions"><el-button :disabled="busy || !targetParametersDirty" @click="resetTargetParameters">还原修改</el-button><el-button :loading="savingParameters" :disabled="busy || !targetParametersDirty" @click="saveParameters">保存参数</el-button></div></div>
          <EnvironmentParameterPairs v-model="targetParameterPairs" :schema="parameterSchema" :disabled="busy || loading" :show-help="false" />
        </section>
        <section v-show="activeTab==='scripts'" id="deployment-panel-scripts" role="tabpanel" aria-labelledby="deployment-tab-scripts" class="panel deployment-editor-panel">
          <div class="deployment-section-title deployment-script-heading"><h2>前后置脚本</h2><DeploymentHelp label="查看前后置脚本说明" title="前后置脚本说明"><p>{{scriptSource}}</p><p>脚本在实际执行节点运行。前置失败则停止部署，部署失败则跳过后置脚本。</p><p v-if="scriptStage==='deploy'">此历史脚本只读。Compose 未填写新启动命令时沿用此脚本；新启动命令在高级配置中编辑。</p></DeploymentHelp><span class="management-muted deployment-script-source">{{scriptSourceLabel}}</span><span v-if="scriptStage==='deploy'" class="management-muted deployment-script-source">只读</span></div>
          <el-tabs v-model="scriptStage"><el-tab-pane label="前置脚本" name="before" /><el-tab-pane label="后置脚本" name="after" /><el-tab-pane v-if="scriptDraft.deploy" label="历史启动脚本" name="deploy" /></el-tabs>
          <CodeEditor v-model="currentScript" :document-key="`${scriptKey}:${scriptStage}`" language="shell" :label="scriptStage==='before'?'前置脚本':scriptStage==='after'?'后置脚本':'历史启动脚本'" fill :readonly="busy || loading || locked || scriptStage==='deploy' || selectedEnv?.script_mode==='disabled'" />
        </section>
        <section v-show="activeTab==='advanced'" id="deployment-panel-advanced" role="tabpanel" aria-labelledby="deployment-tab-advanced" class="panel deployment-panel-scroll">
          <h2>高级配置</h2><el-form class="deployment-advanced-form" label-position="top" :disabled="busy || loading">
            <template v-if="!existing">
              <el-form-item v-if="form.source==='registry' && execution.mode!=='command'" label="镜像仓库"><el-select v-model="form.registry_id"><el-option v-for="registry in registries" :key="registry.id" :value="registry.id" :label="registry.name" /></el-select></el-form-item>
              <ContainerBindingEditor v-if="execution.mode==='command'" v-model:names="containerNames" v-model:replace="replaceContainers" command compact :disabled="busy" />
              <el-form-item><template #label><span class="deployment-field-label">版本名称（可选）<DeploymentHelp label="查看版本名称说明" title="版本名称说明"><p>仅用于本次发布，不随部署配置保存。留空自动生成。</p></DeploymentHelp></span></template><el-input v-model="form.version" placeholder="留空自动生成" aria-label="版本名称（可选）" /></el-form-item>
              <template v-if="!bundleMode">
                <el-form-item><template #label><span class="deployment-field-label">脚本工作目录（可选）<DeploymentHelp label="查看脚本工作目录说明" title="脚本工作目录说明"><p>留空使用任务目录，可用 $WORK_DIR 和运行参数。</p><p>自定义目录须在执行节点中存在，配置文件仍位于任务目录的 configs/ 下。</p></DeploymentHelp></span></template><el-input v-model="execution.work_dir" placeholder="留空使用任务目录" aria-label="脚本工作目录（可选）" /></el-form-item>
                <el-form-item><template #label><span class="deployment-field-label">任务超时（秒）<DeploymentHelp label="查看任务超时说明" title="任务超时说明"><p>0 表示使用系统设置，各阶段共享超时。</p></DeploymentHelp></span></template><el-input-number v-model="execution.timeout_seconds" :min="0" :max="86400" aria-label="任务超时（秒）" /></el-form-item>
                <el-form-item v-if="execution.mode==='compose'"><template #label><span class="deployment-field-label">Compose 启动命令（可选）<DeploymentHelp label="查看 Compose 启动命令说明" title="Compose 启动命令说明"><p>留空由系统启动。自定义时使用 "$COMPOSE_FILE" 和 "$COMPOSE_PROJECT_NAME"，系统继续核对服务和镜像。</p></DeploymentHelp></span></template><CodeEditor v-model="composeCommand" language="shell" label="Compose 启动命令" :readonly="busy" /></el-form-item>
                <el-form-item><template #label><span class="deployment-field-label">部署检查命令（可选）<DeploymentHelp label="查看部署检查命令说明" title="部署检查命令说明"><p>{{execution.mode==='command'?'未配置时只记录命令执行结果。命令部署不自动回滚。':'服务启动检查通过后执行，失败则停止后置脚本。'}}</p></DeploymentHelp></span></template><CodeEditor v-model="execution.check_command" language="shell" label="部署检查命令" /></el-form-item>
                <el-form-item v-if="execution.mode!=='command'" label="目标平台（可选）"><el-select v-model="form.platform" clearable><el-option label="Linux amd64" value="linux/amd64" /><el-option label="Linux arm64" value="linux/arm64" /></el-select></el-form-item>
              </template>
            </template>
          </el-form>
        </section>
      </div>
      <section v-if="preview && !existing" class="panel deployment-preview">
        <el-alert :title="preview.message" type="success" :closable="false" />
        <p class="management-description">{{preview.environment}} · {{targetSummary}}。{{preview.execution?.mode==='command'?'确认后执行命令与前后置脚本。':'普通替换允许短暂停机。'}}</p>
        <el-tabs v-model="previewTab" class="deployment-preview-tabs"><el-tab-pane label="本次部署" name="content" /><el-tab-pane label="当前部署" name="previous" /><el-tab-pane label="运行参数" name="parameters" /><el-tab-pane label="脚本快照" name="scripts" /><el-tab-pane label="高级配置" name="execution" /></el-tabs>
        <div v-show="previewTab==='content'" class="deployment-preview-panel"><CodeEditor :model-value="preview.execution?.mode==='command'?preview.execution.command:preview.compose || ''" :language="preview.execution?.mode==='command'?'shell':'yaml'" label="本次部署内容" fill readonly /></div>
        <div v-show="previewTab==='previous'" class="deployment-preview-panel"><CodeEditor :model-value="preview.previous_execution?.mode==='command'?preview.previous_execution.command:preview.previous_compose || '首次部署'" :language="preview.previous_execution?.mode==='command'?'shell':'yaml'" label="当前部署内容" fill readonly /><pre class="management-code deployment-preview-images">{{preview.images}}</pre></div>
        <div v-show="previewTab==='parameters'" class="deployment-preview-panel deployment-panel-scroll"><p class="management-form-help">敏感参数值已隐藏。</p><dl class="deployment-preview-parameters"><template v-for="(value,key) in preview.parameters || {}" :key="key"><dt>{{key}}</dt><dd>{{value === '' ? '（空值）' : value}}</dd></template></dl></div>
        <div v-show="previewTab==='scripts'" class="deployment-preview-panel"><DeploymentScriptEditor v-if="hasScripts(preview.scripts)" :model-value="copyScripts(preview.scripts)" disabled fill /><p v-else class="management-description">本次未配置前后置脚本。</p></div>
        <div v-show="previewTab==='execution'" class="deployment-preview-panel"><p v-if="preview.container_policy?.names?.length" class="management-description">关联容器：{{preview.container_policy.names.join('、')}} · {{preview.container_policy.replace?'部署时由系统停止并删除后替换':'替换由脚本负责'}}</p><p class="management-description">工作目录：{{preview.execution?.work_dir || '任务目录'}} · 超时：{{preview.execution?.timeout_seconds || '系统设置'}}</p><el-tabs v-model="previewExecutionTab"><el-tab-pane label="启动命令" name="command" /><el-tab-pane label="检查命令" name="check_command" /></el-tabs><CodeEditor :model-value="preview.execution?.[previewExecutionTab] || ''" language="shell" :label="previewExecutionTab==='command'?'本次启动命令':'部署检查命令'" :document-key="previewExecutionTab" fill readonly /></div>
      </section>
    </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, defineAsyncComponent } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { api, errorText } from '@/api/workspace'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import DeploymentContentEditor from '@/components/DeploymentContentEditor.vue'
import VersionDeploymentSummary from '@/components/VersionDeploymentSummary.vue'
import DeploymentScriptEditor from '@/components/DeploymentScriptEditor.vue'
import EnvironmentParameterPairs from '@/components/EnvironmentParameterPairs.vue'
import ContainerBindingEditor from '@/components/ContainerBindingEditor.vue'
import OfflineMaterialPicker from '@/components/OfflineMaterialPicker.vue'
import DeploymentHelp from '@/components/DeploymentHelp.vue'
import type { OfflineMaterial } from '@/api/bundles'
import { releaseConfigFiles, releaseConfigValues, releaseConfigDraft, type ReleaseConfigFile } from '@/utils/releaseConfigFiles'
import { copyScripts, defaultReleaseRegistry, deploymentDraftFromRelease, deploymentParameterReferences, emptyScripts, hasScripts, type DeploymentScripts } from '@/utils/releases'
import { validateTemplateValues } from '@/utils/applicationTemplates'
import { readOnlyEnvironmentParameters, environmentParameterValues, environmentParameterPairs, type EnvironmentParameterPair } from '@/utils/applicationDetail'
import { createDeploymentParameterDraft, deploymentParametersChanged, deploymentParameterDraftsSnapshot, type DeploymentParameterDraft } from '@/utils/deploymentParameters'
import type { TemplateParameter } from '@/api/applicationTemplates'
import { useUnsavedChanges } from '@/composables/useUnsavedChanges'
const CodeEditor=defineAsyncComponent(()=>import('@/components/CodeEditor.vue'))
type Tab='content'|'parameters'|'scripts'|'advanced'
const previewTab=ref('content'),previewExecutionTab=ref<'command'|'check_command'>('command')
const tabs:{key:Tab;label:string}[]=[{key:'content',label:'部署内容'},{key:'parameters',label:'运行参数'},{key:'scripts',label:'前后置脚本'},{key:'advanced',label:'高级配置'}]
const route=useRoute(),router=useRouter(),id=route.params.id
const editingConfiguration=ref(route.query.mode==='edit' || !!route.query.edit_release),copiedFrom=ref(''),copyPending=ref(false)
const importing=ref(false),submitting=ref(false),busy=computed(()=>submitting.value || importing.value)
const loading=ref(false),error=ref(''),defaultsLoaded=ref(false),activeTab=ref<Tab>('content'),invalidTab=ref<Tab>(),data=ref<any>({}),environment=ref<number>(),registries=ref<any[]>([]),targets=ref<any[]>([]),bundles=ref<any[]>([]),bundleId=ref<number>(),existing=ref<number|undefined>(Number(route.query.release)||undefined),preview=ref<any>(null),releaseId=ref(0),savingParameters=ref(false)
const form=ref({source:'direct',registry_id:0,version:'',platform:'',compose:'services:\n  app:\n    image: ${IMAGE}\n    restart: unless-stopped\n    ports:\n      - "${HOST_PORT}:80"\n'})
const execution=ref({mode:'command',command:'',work_dir:'',check_command:'',timeout_seconds:0}),composeCommand=ref('')
const effectiveExecution=computed(()=>({...execution.value,command:execution.value.mode==='command'?execution.value.command:composeCommand.value}))
const configFiles=ref<ReleaseConfigFile[]>([]),parameterDrafts=ref<Record<string,DeploymentParameterDraft>>({})
const optionsController=new AbortController();let disposed=false,preparedSignature='',executionKey=''
let preparedParameters:Record<string,string>|undefined,preparedScripts:DeploymentScripts|undefined,preparedContainers:{names:string[];replace:boolean}|undefined
const selectedEnv=computed(()=>data.value.environments?.find((env:any)=>env.id===environment.value))
const existingRelease=computed(()=>data.value.releases?.find((version:any)=>version.id===existing.value))
const bundleMode=computed(()=>execution.value.mode!=='command' && form.value.source==='bundle')
const offlineCommand=computed(()=>execution.value.mode==='command' && ['local','bundle'].includes(form.value.source))
const offlineMode=computed(()=>bundleMode.value || offlineCommand.value)
const primaryImage=ref('')
const offlineImages=computed<Array<{reference:string;id:string;platform?:string}>>(()=>{const material=bundles.value.find(item=>item.id===bundleId.value);try{const manifest=JSON.parse(material?.manifest || '[]');return material?.kind==='image'?manifest:(manifest.images || []).map((image:any)=>({...image,platform:image.platform || manifest.platform}))}catch{return []}})
watch([offlineImages,bundleId],()=>{if(!offlineImages.value.some(image=>image.reference===primaryImage.value))primaryImage.value=offlineImages.value.length===1?offlineImages.value[0]!.reference:'';if(offlineCommand.value && offlineImages.value.length)form.value.platform=offlineImages.value[0]?.platform || ''})
function onMaterialImported(material:OfflineMaterial){bundles.value=[material,...bundles.value.filter(item=>item.id!==material.id)]}
watch(()=>form.value.source,()=>{bundleId.value=undefined;primaryImage.value=''})
watch([()=>execution.value.mode,()=>form.value.source],()=>{if(execution.value.mode==='command' && !['direct','local','bundle'].includes(form.value.source))form.value.source='direct'})
const locked=computed(()=>!!existing.value || bundleMode.value)
const displayMode=computed(()=>existing.value?(existingRelease.value?.execution?.mode || 'compose'):execution.value.mode)
const mainContent=computed({get:()=>existing.value?(displayMode.value==='command'?existingRelease.value?.execution?.command || '':existingRelease.value?.compose || ''):(execution.value.mode==='command'?execution.value.command:form.value.compose),set:value=>{if(locked.value)return;if(execution.value.mode==='command')execution.value.command=value;else form.value.compose=value}})
const existingFiles=computed(()=>{try{return releaseConfigFiles(JSON.parse(existingRelease.value?.configs || '{}'))}catch{return []}})
const displayFiles=computed({get:()=>existing.value?existingFiles.value:configFiles.value,set:value=>{if(!locked.value)configFiles.value=value}})
const targetSummary=computed(()=>{const env=selectedEnv.value;if(!env)return '请选择部署环境';let ids:number[]=[];try{ids=JSON.parse(env.target_ids || '[]')}catch{};if(!ids.length)ids=[env.target_id || 1];return '目标：'+ids.map(targetId=>targets.value.find((target:any)=>target.id===targetId)?.name || (targetId<=1?'本机':`主机 #${targetId}`)).join('、')})
const parameterSchema=computed<TemplateParameter[]>(()=>data.value.application?.definition?.parameters || [])
const versionParameters=computed(()=>readOnlyEnvironmentParameters(preview.value?.parameters || selectedEnv.value?.parameters || {},parameterSchema.value))
const currentVersionName=computed(()=>data.value.releases?.find((version:any)=>version.id===selectedEnv.value?.current_release_id)?.version || (selectedEnv.value?.current_release_id?`版本 #${selectedEnv.value.current_release_id}`:''))
const versionSummary=computed(()=>({...existingRelease.value,execution:preview.value?.execution || existingRelease.value?.execution,compose:preview.value?.compose || existingRelease.value?.compose}))
const targetParameterDraft=computed(()=>environment.value?parameterDrafts.value[environment.value]:undefined)
const targetParameterPairs=computed({get:()=>targetParameterDraft.value?.pairs || [],set:(pairs:EnvironmentParameterPair[])=>{if(targetParameterDraft.value)targetParameterDraft.value.pairs=pairs}})
const targetParametersDirty=computed(()=>deploymentParametersChanged(targetParameterDraft.value))
watch(existing,()=>{preview.value=null})
watch([selectedEnv,parameterSchema],([env,schema])=>{if(env && !deploymentParametersChanged(parameterDrafts.value[env.id]))parameterDrafts.value[env.id]=createDeploymentParameterDraft(env.parameters || {},schema)},{immediate:true})
const containerDrafts=ref<Record<string,{names:string[];replace:boolean}>>({}),containerBaselines=ref<Record<string,string>>({})
watch(selectedEnv,env=>{if(!env)return;const value={names:[...(env.container_names || [])],replace:!!env.replace_containers};if(!containerDrafts.value[env.id] || JSON.stringify(containerDrafts.value[env.id])===containerBaselines.value[env.id]){containerDrafts.value[env.id]=value;containerBaselines.value[env.id]=JSON.stringify(value)}},{immediate:true,deep:true})
const containerNames=computed({get:()=>containerDrafts.value[String(environment.value)]?.names || [],set:(names:string[])=>{const draft=containerDrafts.value[String(environment.value)];if(draft){if(!draft.names.length && names.length)draft.replace=true;draft.names=names;if(!names.length)draft.replace=false}}})
const replaceContainers=computed({get:()=>containerDrafts.value[String(environment.value)]?.replace || false,set:(value:boolean)=>{const draft=containerDrafts.value[String(environment.value)];if(draft)draft.replace=value}})
const containerChanges=computed(()=>Object.fromEntries(Object.entries(containerDrafts.value).filter(([key,value])=>JSON.stringify(value)!==containerBaselines.value[key])))
useUnsavedChanges(()=>containerChanges.value,{enabled:()=>defaultsLoaded.value,busy:()=>busy.value})
const plannedScripts=computed(()=>selectedEnv.value?.script_mode==='disabled'?emptyScripts():copyScripts(selectedEnv.value?.script_mode==='override'?selectedEnv.value.scripts:(existing.value?existingRelease.value?.scripts:data.value.application?.scripts)))
const scriptDrafts=ref<Record<string,DeploymentScripts>>({}),scriptBaselines=ref<Record<string,string>>({}),scriptStage=ref<'before'|'after'|'deploy'>('before')
const scriptKey=computed(()=>existing.value?`release:${existing.value}:${environment.value}`:selectedEnv.value?.script_mode==='override'?`environment:${environment.value}`:selectedEnv.value?.script_mode==='disabled'?`disabled:${environment.value}`:'application')
watch([scriptKey,plannedScripts],([key,scripts])=>{if(!scriptDrafts.value[key] || JSON.stringify(scriptDrafts.value[key])===scriptBaselines.value[key]){scriptDrafts.value[key]=copyScripts(scripts);scriptBaselines.value[key]=JSON.stringify(scripts)}},{immediate:true})
const scriptDraft=computed(()=>scriptDrafts.value[scriptKey.value] || emptyScripts())
const currentScript=computed({get:()=>scriptDraft.value[scriptStage.value],set:value=>{if(!locked.value && scriptStage.value!=='deploy' && selectedEnv.value?.script_mode!=='disabled')scriptDraft.value[scriptStage.value]=value}})
const scriptChanges=computed(()=>Object.fromEntries(Object.entries(scriptDrafts.value).filter(([key,value])=>JSON.stringify(value)!==scriptBaselines.value[key])))
const scriptCount=computed(()=>Number(!!scriptDraft.value.before.trim())+Number(!!scriptDraft.value.after.trim()))
const scriptSource=computed(()=>selectedEnv.value?.script_mode==='disabled'?'此环境已停用前后置脚本；部署命令仍会执行。':selectedEnv.value?.script_mode==='override'?'使用此环境的独立脚本。':existing.value?'沿用发布版本保存的脚本。':'应用默认脚本，新版本使用此配置。')
const scriptSourceLabel=computed(()=>selectedEnv.value?.script_mode==='disabled'?'附加脚本已停用':selectedEnv.value?.script_mode==='override'?'环境独立脚本':existing.value?'版本脚本':'应用默认脚本')
const formDraft=computed(()=>({form:{source:form.value.source,registry_id:form.value.registry_id,platform:form.value.platform,compose:form.value.compose},execution:effectiveExecution.value,commandDraft:execution.value.command,configs:releaseConfigDraft(configFiles.value),copyPending:copyPending.value}))
const {markSaved,dirty:configurationDirty}=useUnsavedChanges(()=>formDraft.value,{enabled:()=>defaultsLoaded.value,busy:()=>busy.value})
useUnsavedChanges(()=>deploymentParameterDraftsSnapshot(parameterDrafts.value),{enabled:()=>defaultsLoaded.value,busy:()=>busy.value})
useUnsavedChanges(()=>scriptChanges.value,{enabled:()=>defaultsLoaded.value,busy:()=>busy.value})
const {markSaved:markReleaseSaved,dirty:releaseDirty}=useUnsavedChanges(()=>({version:form.value.version,bundle:bundleId.value,primaryImage:primaryImage.value}),{enabled:()=>defaultsLoaded.value,busy:()=>busy.value})
const hasChanges=computed(()=>configurationDirty.value || targetParametersDirty.value || !!Object.keys(scriptChanges.value).length || !!Object.keys(containerChanges.value).length)
const references=computed(()=>deploymentParameterReferences(mainContent.value))
const missingReferences=computed(()=>references.value.filter(key=>!targetParameterPairs.value.some(pair=>pair.key===key)))
const referenceSummary=computed(()=>references.value.length?`已引用参数：${references.value.join('、')}`:'当前文件没有引用参数')
const contentState=computed(()=>JSON.stringify({mode:execution.value.mode,command:execution.value.command,compose:form.value.compose,composeCommand:composeCommand.value,files:releaseConfigDraft(configFiles.value)}))
const advancedState=computed(()=>JSON.stringify({source:form.value.source,registry:form.value.registry_id,platform:form.value.platform,workDir:execution.value.work_dir,check:execution.value.check_command,timeout:execution.value.timeout_seconds}))
const contentBaseline=ref(contentState.value),advancedBaseline=ref(advancedState.value)
function markConfigurationSaved(){markSaved();contentBaseline.value=contentState.value;advancedBaseline.value=advancedState.value}
function tabDirty(tab:Tab){if(tab==='parameters')return targetParametersDirty.value;if(tab==='scripts')return JSON.stringify(scriptDraft.value)!==scriptBaselines.value[scriptKey.value];return tab==='content'?copyPending.value || contentState.value!==contentBaseline.value:advancedState.value!==advancedBaseline.value || releaseDirty.value || !!containerChanges.value[String(environment.value)]}
function addReferencedParameters(){targetParameterPairs.value=[...targetParameterPairs.value,...environmentParameterPairs(Object.fromEntries(missingReferences.value.map(key=>[key,''])))];activeTab.value='parameters'}
function invalidatePreview(){preview.value=null;preparedParameters=undefined;preparedScripts=undefined;preparedContainers=undefined;invalidTab.value=undefined}
watch([formDraft,()=>deploymentParameterDraftsSnapshot(parameterDrafts.value),scriptChanges,containerChanges,environment,existing,()=>form.value.version,bundleId,primaryImage],invalidatePreview,{deep:true,flush:'sync'})
function targetParameterValues(){try{const values=environmentParameterValues(targetParameterPairs.value,parameterSchema.value);if(parameterSchema.value.length)validateTemplateValues(parameterSchema.value,values);return values}catch(cause){activeTab.value='parameters';invalidTab.value='parameters';throw cause}}
function resetTargetParameters(){if(busy.value || !selectedEnv.value)return;parameterDrafts.value[selectedEnv.value.id]=createDeploymentParameterDraft(selectedEnv.value.parameters || {},parameterSchema.value)}
async function persistTargetParameters(values:Record<string,string>){if(!targetParametersDirty.value || !selectedEnv.value || !targetParameterDraft.value)return;const env=selectedEnv.value;savingParameters.value=true;try{const saved=await api<{parameters:Record<string,string>}>('PATCH',`/applications/${id}/environments/${env.id}/parameters`,{parameters:values,expected_parameters:targetParameterDraft.value.savedParameters});env.parameters=saved.parameters;parameterDrafts.value[env.id]=createDeploymentParameterDraft(saved.parameters,parameterSchema.value)}finally{savingParameters.value=false}}
async function saveParameters(){if(busy.value || !!existing.value)return;submitting.value=true;error.value='';try{await persistTargetParameters(targetParameterValues());ElMessage.success('环境运行参数已保存')}catch(cause){error.value=errorText(cause)}finally{submitting.value=false}}
function validateConfiguration(){
  if(!selectedEnv.value)throw new Error('请选择部署环境')
  if(existing.value && !existingRelease.value)throw new Error('所选发布版本不存在，请返回应用刷新后重试')
  if(locked.value)return
  if(!mainContent.value.trim()){activeTab.value='content';invalidTab.value='content';throw new Error(execution.value.mode==='command'?'请填写部署命令':'请填写 Compose 定义')}
  if(!Number.isInteger(execution.value.timeout_seconds)||execution.value.timeout_seconds<0||execution.value.timeout_seconds>86400){activeTab.value='advanced';invalidTab.value='advanced';throw new Error('任务超时必须为 0 至 86400 秒')}
  try{releaseConfigValues(configFiles.value)}catch(cause){activeTab.value='content';invalidTab.value='content';throw cause}
}
async function persistConfiguration(){
  if(locked.value)return
  const saved=await api<any>('PUT',`/applications/${id}/deployment-config`,{environment_id:environment.value,source:form.value.source,registry_id:form.value.registry_id,platform:form.value.platform,compose:form.value.compose,execution:effectiveExecution.value,configs:releaseConfigValues(configFiles.value),scripts:scriptDraft.value,...(execution.value.mode==='command'?{container_names:containerNames.value,replace_containers:replaceContainers.value}:{})})
  data.value.application=saved.application;Object.assign(selectedEnv.value,saved.environment)
  containerBaselines.value[String(environment.value)]=JSON.stringify(containerDrafts.value[String(environment.value)])
  copyPending.value=false;scriptBaselines.value[scriptKey.value]=JSON.stringify(scriptDraft.value);markConfigurationSaved()
}
async function saveConfiguration(){if(busy.value || loading.value)return;submitting.value=true;error.value='';try{validateConfiguration();await persistTargetParameters(targetParameterValues());await persistConfiguration();ElMessage.success('配置已保存，供后续手动和自动发布使用，尚未执行部署')}catch(cause){handleFailure(cause)}finally{submitting.value=false}}
function editVersion(version:any){
  if(!version)return
  try{
    const draft=deploymentDraftFromRelease(version)
    existing.value=undefined;editingConfiguration.value=true;copiedFrom.value=version.version;copyPending.value=true
    form.value={...form.value,version:'',source:draft.source,registry_id:draft.registry_id || defaultReleaseRegistry(registries.value),platform:draft.platform,compose:draft.compose}
    execution.value={...draft.execution};composeCommand.value=draft.execution.mode==='compose'?draft.execution.command:''
    if(draft.execution.mode==='compose')execution.value.command=''
    configFiles.value=releaseConfigFiles(draft.configs)
    if(selectedEnv.value?.script_mode!=='override' && selectedEnv.value?.script_mode!=='disabled'){
      if(scriptBaselines.value[scriptKey.value]===undefined)scriptBaselines.value[scriptKey.value]=JSON.stringify(plannedScripts.value)
      scriptDrafts.value[scriptKey.value]=copyScripts(draft.scripts)
    }
    activeTab.value='content';error.value=''
    void router.replace({query:{...route.query,release:undefined,edit_release:String(version.id),mode:'edit',environment:environment.value}})
  }catch(cause){error.value=errorText(cause)}
}
function handleFailure(cause:unknown){error.value=errorText(cause);if(error.value.includes('语法')){activeTab.value=error.value.includes('部署检查')?'advanced':error.value.includes('部署前')||error.value.includes('部署后')?'scripts':'content';invalidTab.value=activeTab.value}}
async function insertDockerRun(){
  if(execution.value.command.trim()){try{await ElMessageBox.confirm('插入示例会替换当前部署命令。是否继续？','替换部署命令',{confirmButtonText:'插入示例',cancelButtonText:'继续编辑',type:'warning'})}catch{return}}
  execution.value.command='docker run -d \\\n  --name "$APP_NAME" \\\n  --restart unless-stopped \\\n  -p "$HOST_PORT:80" \\\n  "${DOCKERUN_IMAGE:-$IMAGE}"\n'
  if(!containerNames.value.length){containerNames.value=['$APP_NAME'];replaceContainers.value=true}
  const defaults:Record<string,string>={APP_NAME:`app-${id}-${environment.value}`,HOST_PORT:'8080',IMAGE:'nginx:latest'}
  targetParameterPairs.value=[...targetParameterPairs.value,...environmentParameterPairs(Object.fromEntries(Object.entries(defaults).filter(([key])=>!targetParameterPairs.value.some(pair=>pair.key===key))))]
}
async function preflight(){
  if(busy.value || loading.value)return;submitting.value=true;error.value='';invalidTab.value=undefined
  try{
    validateConfiguration()
    const parameters=existing.value?{...(selectedEnv.value.parameters || {})}:targetParameterValues()
    if(!existing.value && offlineMode.value && !bundleId.value){activeTab.value='content';throw new Error('请选择或导入本次发布的离线包')}
    if(!existing.value && offlineCommand.value && offlineImages.value.length>1 && !primaryImage.value){activeTab.value='content';throw new Error('请选择此离线包中的脚本主镜像')}
    if(!existing.value)await persistTargetParameters(parameters)
    if(!locked.value)await persistConfiguration()
    const signature=JSON.stringify({version:form.value.version,existing:existing.value,bundle:bundleId.value,primaryImage:primaryImage.value,environment:environment.value,parameters:selectedEnv.value.parameters,draft:formDraft.value,scripts:scriptDraft.value})
    if(signature===preparedSignature && releaseId.value){ /* 重试复用已准备版本。 */ }
    else if(existing.value)releaseId.value=existing.value
    else if(bundleMode.value){const result=await api<any>('POST',`/bundles/${bundleId.value}/version`,{application_id:Number(id)});releaseId.value=result.release.id}
    else{const result=await api<any>('POST',`/applications/${id}/releases?environment_id=${environment.value}`,{...form.value,source:execution.value.mode==='command' && !offlineCommand.value?'command':form.value.source,bundle_id:offlineCommand.value?bundleId.value:undefined,trigger_image:offlineCommand.value?primaryImage.value:undefined,execution:effectiveExecution.value,configs:JSON.stringify(releaseConfigValues(configFiles.value)),save_definition:true});releaseId.value=result.id}
    preparedSignature=signature
    const expectedParameters={...(selectedEnv.value.parameters || {})},expectedScripts=bundleMode.value && !existing.value?undefined:copyScripts(existing.value?plannedScripts.value:scriptDraft.value)
    const result=await api<any>('POST','/deployments/preflight',{environment_id:environment.value,release_id:releaseId.value,restore_configuration:false,expected_parameters:expectedParameters,expected_scripts:expectedScripts})
    preparedParameters=expectedParameters;preparedScripts=copyScripts(result.scripts);preparedContainers=result.container_policy;executionKey=crypto.randomUUID();previewTab.value='content';preview.value=result
  }catch(cause){handleFailure(cause)}finally{submitting.value=false}
}
async function deploy(){if(busy.value || !preview.value || !releaseId.value)return;submitting.value=true;error.value='';try{const task=await api<any>('POST','/deployments',{environment_id:environment.value,release_id:releaseId.value,restore_configuration:false,idempotency_key:executionKey,expected_parameters:preparedParameters,expected_scripts:preparedScripts,expected_containers:preparedContainers});markReleaseSaved();submitting.value=false;await router.replace(`/deployments/${task.id}`)}catch(cause){error.value=errorText(cause)}finally{submitting.value=false}}
async function loadOptions(){
  if(loading.value || busy.value)return;loading.value=true;error.value=''
  try{
    const results=await Promise.allSettled([api<any>('GET',`/applications/${id}`,undefined,optionsController.signal),api<any>('GET',`/applications/${id}/sources`,undefined,optionsController.signal),existing.value?Promise.resolve([]):api<any[]>('GET','/bundles',undefined,optionsController.signal)])
    if(disposed)return
    const [application,registryOptions,bundleOptions]=results
    if(application.status==='rejected')throw application.reason
    data.value=application.value
    if(existing.value && !existingRelease.value)throw new Error('所选发布版本不存在，请返回应用刷新后重试')
    if(registryOptions.status==='fulfilled'){registries.value=registryOptions.value.registries || [];targets.value=registryOptions.value.targets || []}
    if(bundleOptions.status==='fulfilled')bundles.value=bundleOptions.value || []
    if(registryOptions.status==='rejected'||bundleOptions.status==='rejected')error.value=existing.value?'部分目标主机信息加载失败，可刷新重试。':'部分仓库或发布包选项加载失败，可刷新重试；当前编辑内容已保留。'
    if(!data.value.environments?.some((env:any)=>env.id===environment.value))environment.value=data.value.environments?.find((env:any)=>env.id===Number(route.query.environment))?.id || data.value.environments?.[0]?.id
    if(!form.value.registry_id)form.value.registry_id=defaultReleaseRegistry(registries.value)
    if(!defaultsLoaded.value){
      const definition=data.value.application?.definition
      if(definition?.compose || definition?.execution?.mode){form.value.compose=definition.compose || form.value.compose;configFiles.value=releaseConfigFiles(definition.configs || {});execution.value={...execution.value,...definition.execution,mode:definition.execution?.mode || 'compose'};if(execution.value.mode==='compose'){composeCommand.value=execution.value.command;execution.value.command=''};form.value.source=definition.source || (data.value.application.template_id || route.query.prepare==='template'?'registry':'direct');form.value.registry_id=definition.registry_id || selectedEnv.value?.registry_id || form.value.registry_id;form.value.platform=definition.platform || ''}
      defaultsLoaded.value=true;markConfigurationSaved()
      if(route.query.edit_release){
        const version=data.value.releases?.find((item:any)=>item.id===Number(route.query.edit_release))
        if(!version)throw new Error('要编辑的历史版本不存在，请返回应用后刷新重试')
        editVersion(version)
      }else if(!existing.value && !definition?.compose && !definition?.execution?.mode){
        const current=data.value.releases?.find((item:any)=>item.id===selectedEnv.value?.current_release_id) || data.value.releases?.[0]
        if(current)editVersion(current)
      }
    }
  }catch(cause){if(!disposed)error.value=errorText(cause)}finally{loading.value=false}
}
onMounted(loadOptions)
onBeforeUnmount(()=>{disposed=true;optionsController.abort()})
</script>
<style scoped>
.deployment-workspace.deployment-workspace--version{display:block;overflow:auto;scrollbar-width:thin}
.deployment-workspace--version>.el-alert{margin-bottom:16px}
.deployment-workspace{--color-bg-2:#202020;--color-bg-3:#171717;--color-border:#3b3b3b}
.deployment-context,.deployment-content-heading,.deployment-references,.deployment-parameter-heading,.deployment-parameter-actions{display:flex;align-items:center;flex-wrap:wrap;gap:8px}
.deployment-mode-controls{display:flex;align-items:center;flex-wrap:wrap;gap:8px;min-width:0}.deployment-copy-source{max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:12px}
.deployment-section-title,.deployment-field-label{display:flex;align-items:center;gap:6px;flex-wrap:wrap;min-width:0}.deployment-section-title{flex-shrink:0}.deployment-section-title h2{margin:0}.deployment-script-heading{margin-bottom:8px}.deployment-script-source{font-size:12px}.deployment-script-heading+.el-tabs{margin-top:0}.deployment-advanced-form{margin-top:12px}.deployment-advanced-form :deep(.el-form-item){margin-bottom:18px}
.deployment-context{font-size:12px;padding:2px 0 12px;gap:10px}.deployment-context .el-select{width:180px}
.deployment-tabs{display:flex;align-items:center;flex-wrap:wrap;gap:4px 22px;border-bottom:1px solid #303030}
.deployment-tabs button{display:flex;align-items:center;gap:6px;border:0;border-bottom:2px solid transparent;border-radius:0;background:transparent;color:#aaa;padding:10px 0;font:inherit;font-size:13px;cursor:pointer}
.deployment-tabs button.active{color:#35acd7;border-bottom-color:#35acd7}.deployment-tabs button.invalid{color:#f1a7ac}.deployment-tabs button:focus-visible{outline:2px solid #35acd7;outline-offset:2px}
.deployment-tab-count{font-size:11px;color:#999}.deployment-dirty{color:#e6c48e}
.deployment-image-source{display:flex;align-items:center;gap:8px;flex-shrink:1;font-size:12px;min-width:0;max-width:100%}.deployment-image-source>label{flex:0 0 auto}.deployment-image-source>.el-select{width:220px;max-width:100%;min-width:0;flex:0 1 220px}.deployment-image-source--primary>.el-select{width:340px;flex-basis:340px}.deployment-content-heading{justify-content:space-between;margin-bottom:16px}.deployment-references{margin-top:12px;font-size:12px;overflow-wrap:anywhere}.deployment-references>.management-muted{flex:1;min-width:0}
.deployment-parameter-heading{justify-content:space-between;margin-bottom:16px}.deployment-parameter-heading h2{margin:0}
.deployment-workspace .management-form-help{margin:12px 0 16px}.deployment-workspace .panel{background:#202020;border:1px solid #383838;border-radius:10px;min-width:0;margin:0}
.deployment-workspace :deep(.el-form-item__content){min-width:0}.deployment-workspace :deep(.el-form-item .code-editor-wrapper){width:100%}
.deployment-preview{display:flex;flex-direction:column;gap:16px}.deployment-preview-heading{font-size:12px;color:#aaa;margin:0}
.deployment-preview-parameters{display:grid;grid-template-columns:minmax(100px,1fr) minmax(0,2fr);gap:10px;margin:0}.deployment-preview-parameters dt{color:#aaa;overflow-wrap:anywhere}.deployment-preview-parameters dd{margin:0;white-space:pre-wrap;overflow-wrap:anywhere}
@media(max-width:760px){.deployment-tabs{gap:4px 16px}.deployment-content-heading{gap:12px}.deployment-context .management-muted{width:100%}}

.deployment-workspace { flex:1;min-height:0;overflow:hidden; }
.deployment-workspace>.deployment-edit-panels { display:flex;flex-direction:column;flex:1 1 0;min-height:0;min-width:0;overflow:hidden; }
.deployment-edit-panels>.panel,.deployment-workspace>.deployment-preview { flex:1 1 0;min-height:0;min-width:0;padding:16px; }
.deployment-editor-panel { display:flex;flex-direction:column;overflow:hidden; }
.deployment-content-settings{display:flex;flex-direction:column;gap:10px;flex-shrink:0;min-width:0;margin-bottom:12px}.deployment-content-settings>.deployment-content-heading,.deployment-content-settings>.management-form-help{margin:0}
.deployment-editor-panel>.deployment-content-heading,.deployment-editor-panel>h2,.deployment-editor-panel>.management-form-help,.deployment-editor-panel>.el-tabs,.deployment-references { flex-shrink:0; }
.deployment-editor-panel>.management-form-help { margin:6px 0 0;white-space:normal;overflow-wrap:anywhere; }
.deployment-editor-panel>h2 { margin-bottom:4px; }
.deployment-editor-panel>.el-tabs { margin-top:6px; }
.deployment-content-heading { margin-bottom:12px; }
.deployment-references { flex-wrap:nowrap;margin-top:8px; }
.deployment-references>.management-muted { overflow:hidden;text-overflow:ellipsis;white-space:nowrap; }
.deployment-panel-scroll { overflow:auto;scrollbar-width:thin; }
.deployment-preview { overflow:hidden;gap:10px; }
.deployment-preview>.el-alert,.deployment-preview>.management-description,.deployment-preview>.el-tabs { flex-shrink:0; }
.deployment-preview-panel { display:flex;flex-direction:column;flex:1 1 0;min-height:0;min-width:0;overflow:hidden; }
.deployment-preview-panel.deployment-panel-scroll { overflow:auto; }
.deployment-preview-panel>.el-tabs,.deployment-preview-panel>.management-description { flex-shrink:0; }
.deployment-preview-images { flex:0 1 auto;max-height:120px;overflow:auto;margin:8px 0 0; }
@media(max-width:760px){.deployment-edit-panels>.panel,.deployment-workspace>.deployment-preview{padding:12px}.deployment-content-heading{gap:6px;margin-bottom:8px}.deployment-references .el-button{font-size:11px}.deployment-tabs{gap:0 12px}.deployment-tabs button{padding:7px 0}.deployment-tab-count{font-size:11px}}
@media(max-width:640px){.deployment-image-source>label{flex-basis:64px}}
@media(max-height:600px){.deployment-editor-panel{overflow:auto}.deployment-editor-panel>.deployment-content-editor,.deployment-editor-panel>.code-editor-wrapper{flex:1 0 240px}.deployment-preview-panel>.code-editor-wrapper{min-height:120px}.deployment-preview{overflow:auto}.deployment-preview-panel{flex-shrink:0;min-height:200px}}
@media(max-width:760px) and (max-height:900px){.deployment-editor-panel{overflow:auto}.deployment-editor-panel>.deployment-content-editor,.deployment-editor-panel>.code-editor-wrapper{flex:1 0 280px}}
</style>
