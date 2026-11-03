<template>
  <section class="compose-service-settings" aria-labelledby="compose-service-title">
    <div class="compose-service-heading"><h3 id="compose-service-title">容器名称与网络</h3><DeploymentHelp label="查看容器名称与网络说明" title="容器名称与网络"><p>按服务设置，修改会同步到部署内容；保存配置不改变正在运行的容器，确认发布后才生效。</p><p>容器名称留空由 Compose 自动生成，也可以填写 ${变量名} 并在运行参数中配置。固定名称在同一主机上必须唯一，多副本服务应使用自动名称。</p><p>网络留空使用应用默认网络；选择已有网络会作为外部网络连接，不会创建或删除该网络。可多选，原有网络的别名和固定地址会保留。</p><p>远程目标和固定主机组可输入网络名称；部署前须在每台目标主机准备同名网络。</p></DeploymentHelp><el-button v-if="localTarget" text circle aria-label="刷新可选网络" title="刷新可选网络" :loading="loading" :disabled="disabled || loading" @click="loadNetworks"><PortainerIcon name="refresh" /></el-button></div>
    <div v-if="parseError" class="management-error" role="alert">{{parseError}}<span> 请在部署内容中修正 Compose 后继续。</span></div>
    <div v-if="networkError && localTarget" class="management-error" role="alert"><span>{{networkError}}</span><el-button link :disabled="loading || disabled" @click="loadNetworks">重试</el-button></div>
    <p v-if="!localTarget" class="compose-service-state">当前目标包含远程主机，请输入目标主机上的网络名称；本机网络列表不适用于此目标。</p>
    <div v-if="editError" class="management-error" role="alert">{{editError}}</div>
    <div v-for="service in settings.services" :key="service.name" class="compose-service-row">
      <h4>{{service.name}}</h4>
      <p v-if="service.editingReason" class="compose-service-state">{{service.editingReason}}</p>
      <div class="compose-service-fields">
        <el-form-item label="容器名称（可选）"><el-input :model-value="service.containerName" :disabled="disabled || !!service.editingReason" placeholder="留空由 Compose 自动生成" :aria-label="`${service.name} 的容器名称`" @update:model-value="setName(service.name, String($event))" /></el-form-item>
        <el-form-item label="连接网络（可选）"><el-select :model-value="service.networks" multiple filterable :allow-create="!localTarget" default-first-option :reserve-keyword="false" :loading="loading" :disabled="disabled || !!service.editingReason || !!service.networkEditingReason" placeholder="留空使用应用默认网络" :aria-label="`${service.name} 的连接网络`" @update:model-value="setNetworks(service.name, $event)">
          <el-option-group label="Compose 中的网络"><el-option v-for="option in settings.networks" :key="option.value" :value="option.value" :label="option.label" /></el-option-group>
          <el-option-group v-if="localTarget && networks.length" label="网络管理中的已有网络"><el-option v-for="network in networks" :key="network.Id" :value="`external:${network.Name}`" :label="`${network.Name} · ${network.Driver}`" :disabled="!!composeNetworkUnavailableReason(network)" :title="composeNetworkUnavailableReason(network) || network.Name" /></el-option-group>
        </el-select><span v-if="service.networkEditingReason" class="compose-service-state">{{service.networkEditingReason}}</span></el-form-item>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { api, errorText } from '@/api/workspace'
import type { DockerNetwork } from '@/utils/networks'
import { composeNetworkUnavailableReason, readComposeServices, restoreComposeFormatting, setComposeContainerName, setComposeServiceNetworks } from '@/utils/composeServices'
import DeploymentHelp from '@/components/DeploymentHelp.vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
const props=defineProps<{modelValue:string;localTarget:boolean;disabled?:boolean;baseline?:string}>()
const emit=defineEmits<{'update:modelValue':[value:string]}>()
const networks=ref<DockerNetwork[]>([]),loading=ref(false),networkError=ref(''),editError=ref('')
let controller:AbortController|undefined,disposed=false
let originalSource='',lastGenerated:string|undefined
const parsed=computed(()=>{try{return {settings:readComposeServices(props.modelValue),error:''}}catch(cause){return {settings:{services:[],networks:[]},error:errorText(cause)}}})
const settings=computed(()=>parsed.value.settings),parseError=computed(()=>parsed.value.error)
function applyChange(update:()=>string){try{if(props.modelValue!==lastGenerated)originalSource=props.modelValue;lastGenerated=restoreComposeFormatting(update(),originalSource);emit('update:modelValue',lastGenerated);editError.value=''}catch(cause){editError.value=errorText(cause)}}
function setName(service:string,value:string){applyChange(()=>setComposeContainerName(props.modelValue,service,value))}
function setNetworks(service:string,value:string[]){applyChange(()=>setComposeServiceNetworks(props.modelValue,service,value))}
async function loadNetworks(){
  if(loading.value || disposed || !props.localTarget)return
  controller=new AbortController();loading.value=true;networkError.value=''
  try{const result=await api<DockerNetwork[]|null>('GET','/networks',undefined,controller.signal);if(!disposed)networks.value=(result || []).sort((a,b)=>a.Name.localeCompare(b.Name))}catch(cause){if(!disposed && !controller.signal.aborted)networkError.value=`可选网络加载失败：${errorText(cause)}`}finally{loading.value=false}
}
watch(()=>props.localTarget,local=>{if(local)void loadNetworks()},{immediate:true})
watch(()=>props.modelValue,()=>{editError.value=''})
watch(()=>props.baseline,()=>{lastGenerated=undefined})
onBeforeUnmount(()=>{disposed=true;controller?.abort()})
</script>
<style scoped>
.compose-service-settings{margin-bottom:24px;padding-bottom:8px;border-bottom:1px solid #383838;min-width:0}.compose-service-heading{display:flex;align-items:center;flex-wrap:wrap;gap:6px;margin-bottom:12px}.compose-service-heading h3{font-size:15px;font-weight:600;margin:0}.compose-service-heading>.el-button{margin-left:auto}.compose-service-row{padding:12px 0;border-top:1px solid #303030}.compose-service-row h4{margin:0 0 12px;font-size:13px;font-weight:600;overflow-wrap:anywhere}.compose-service-fields{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.5fr);gap:16px}.compose-service-fields .el-select{width:100%;min-width:0}.compose-service-state{color:#aaa;font-size:12px;line-height:1.6;overflow-wrap:anywhere;margin:0 0 12px}.compose-service-fields .compose-service-state{margin:6px 0 0}.compose-service-settings .management-error{margin-bottom:12px}
@media(max-width:760px){.compose-service-fields{grid-template-columns:minmax(0,1fr);gap:0}}
</style>
