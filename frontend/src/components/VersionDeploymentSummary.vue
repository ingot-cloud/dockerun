<template>
  <div class="version-deployment-summary">
    <section class="panel version-deployment-card">
      <h2>部署版本 {{version.version}}</h2>
      <p class="management-description">使用此版本保存的部署内容，以及目标环境已保存的运行参数与附加脚本配置。</p>
      <dl class="version-deployment-values">
        <dt>目标环境</dt><dd>{{environment?.name || '请选择部署环境'}}</dd>
        <dt>环境当前版本</dt><dd>{{currentVersion || '尚未部署'}}</dd>
        <template v-if="environment && version.execution?.mode!=='command'"><dt>Compose 项目</dt><dd>{{composeProjectName(environment)}}</dd></template>
        <dt>部署方式</dt><dd>{{version.execution?.mode==='command'?'命令部署':'Compose 部署'}}</dd>
        <template v-if="version.source!=='command'"><dt>版本来源</dt><dd>{{sourceLabel(version.source,version.bundle_id)}}</dd></template>
        <template v-if="version.platform"><dt>目标平台</dt><dd>{{version.platform}}</dd></template>
        <template v-if="version.execution?.work_dir"><dt>工作目录</dt><dd>{{version.execution.work_dir}}</dd></template>
        <template v-if="version.execution?.timeout_seconds"><dt>任务超时</dt><dd>{{version.execution.timeout_seconds}} 秒</dd></template>
        <template v-if="policy.names.length"><dt>关联容器</dt><dd>{{policy.names.join('、')}}</dd><dt>容器替换</dt><dd>{{policy.replace && version.execution?.mode==='command'?'先准备此版本镜像，再停止并删除已关联的旧容器，执行此版本脚本；不清理镜像':'由脚本或 Compose 管理'}}</dd></template>
        <template v-if="version.bundle_id"><dt>离线镜像</dt><dd>从此版本保留的离线包重新加载并核对 Image ID，不拉取仓库。</dd></template>
        <template v-if="hasScripts(scripts) || environment?.script_mode==='disabled'"><dt>附加脚本来源</dt><dd>{{environment?.script_mode==='disabled'?'不执行附加脚本':environment?.script_mode==='override'?'使用此环境的独立附加脚本':'使用版本中的应用附加脚本'}}</dd></template>
      </dl>
    </section>
    <section v-if="Object.keys(parameters).length" class="panel version-deployment-card">
      <h2>运行参数<span class="management-muted"> · 只读</span></h2>
      <dl v-if="Object.keys(parameters).length" class="version-deployment-values"><template v-for="(value,key) in parameters" :key="key"><dt>{{key}}</dt><dd>{{value===''?'（空值）':value}}</dd></template></dl>
      <p v-else class="management-description">此环境未配置运行参数。</p>
    </section>
    <section class="panel version-deployment-card">
      <el-collapse v-model="expanded" class="version-deployment-details">
        <el-collapse-item title="部署内容与配置文件 · 只读" name="content">
          <el-tabs v-model="activeFile" class="version-deployment-files"><el-tab-pane v-for="file in documents" :key="file.name" :label="file.name" :name="file.name" /></el-tabs>
          <div v-if="currentDocument.name==='compose.yaml'" class="version-deployment-compose"><DeploymentComposePreview :content="currentDocument.content" :images="versionImages" title="版本部署配置" description="展示此版本的部署内容。预检查后显示基于所选环境解析的脱敏配置，确认前仍需核对目标和运行参数。" /></div>
          <CodeEditor v-else :model-value="currentDocument.content" :filename="currentDocument.name" :document-key="currentDocument.name" :label="currentDocument.name" :height="320" readonly />
        </el-collapse-item>
        <el-collapse-item v-if="scriptDocuments.length" title="启动命令、检查与附加脚本 · 只读" name="scripts">
          <CodeEditor v-for="script in scriptDocuments" :key="script.name" :model-value="script.content" :label="script.name" :document-key="script.name" language="shell" :height="240" readonly />
        </el-collapse-item>
      </el-collapse>
    </section>
  </div>
</template>
<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { sourceLabel } from '@/utils/management'
import { hasScripts, type DeploymentScripts } from '@/utils/releases'
import { composeProjectName } from '@/utils/composeProject'
import DeploymentComposePreview from '@/components/DeploymentComposePreview.vue'
const CodeEditor=defineAsyncComponent(()=>import('./CodeEditor.vue'))
const props=defineProps<{version:any;environment?:any;currentVersion?:string;parameters:Record<string,string>;scripts:DeploymentScripts;containerPolicy?:{names:string[];replace:boolean}}>()
const policy=computed(()=>props.containerPolicy || {names:props.environment?.container_names || [],replace:props.environment?.replace_containers || false})
const expanded=ref<string[]>([]),activeFile=ref('')
const documents=computed(()=>{
  let configs:Record<string,string>={}
  try{configs=JSON.parse(props.version.configs || '{}') || {}}catch{}
  const command=props.version.execution?.mode==='command'
  return [{name:command?'deploy.sh':'compose.yaml',content:command?props.version.execution?.command || '':props.version.compose || ''},...Object.entries(configs).map(([name,content])=>({name,content}))]
})
watch(documents,files=>{if(!files.some(file=>file.name===activeFile.value))activeFile.value=files[0]?.name || ''},{immediate:true})
const currentDocument=computed(()=>documents.value.find(file=>file.name===activeFile.value) || documents.value[0]!)
const versionImages=computed(()=>{try{const images=JSON.parse(props.version.images || '[]');return Array.isArray(images)?images:[]}catch{return []}})
const scriptDocuments=computed(()=>[
  {name:'Compose 启动命令',content:props.version.execution?.mode!=='command'?props.version.execution?.command:''},
  {name:'部署检查命令',content:props.version.execution?.check_command},
  {name:'部署前脚本',content:props.scripts.before},
  {name:'历史启动脚本',content:props.scripts.deploy},
  {name:'部署后脚本',content:props.scripts.after},
].filter(script=>script.content?.trim()))
</script>
<style scoped>
.version-deployment-summary{display:flex;flex-direction:column;gap:16px;min-width:0}
.version-deployment-card{padding:16px;min-width:0}
.version-deployment-card h2{font-size:15px;margin:0 0 12px}
.version-deployment-card h2>.management-muted{font-size:12px;font-weight:400}
.version-deployment-values{display:grid;grid-template-columns:minmax(100px,180px) minmax(0,1fr);gap:10px 16px;margin:0;font-size:13px}
.version-deployment-values dt{color:#999;overflow-wrap:anywhere}.version-deployment-values dd{margin:0;white-space:pre-wrap;overflow-wrap:anywhere}
.version-deployment-files{min-width:0}.version-deployment-details :deep(.el-collapse-item__header){font-size:13px}
.version-deployment-compose{display:flex;height:clamp(320px,55dvh,560px);min-width:0;min-height:0}
.version-deployment-details :deep(.code-editor-wrapper)+:deep(.code-editor-wrapper){margin-top:16px}
@media(max-width:760px){.version-deployment-summary{gap:12px}.version-deployment-card{padding:12px}.version-deployment-values{grid-template-columns:minmax(90px,35%) minmax(0,1fr);gap:10px}}
</style>
