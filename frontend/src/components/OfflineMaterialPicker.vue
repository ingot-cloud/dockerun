<template>
  <div class="offline-material-picker">
    <div class="offline-material-controls">
      <slot name="source" />
      <div v-if="showMaterials" class="offline-material-selection">
        <div class="offline-material-label">
          <label for="deployment-offline-material">离线包</label>
          <DeploymentHelp :key="kind" label="查看离线包导入说明" title="离线包说明">
            <p>{{kind==='image'?'支持 docker save 镜像归档，导入后复用当前部署脚本。':'完整发布包包含镜像和部署定义。'}}</p>
            <p>导入只准备镜像，确认部署后才替换旧容器。</p>
            <router-link to="/bundles">管理离线材料</router-link>
          </DeploymentHelp>
        </div>
        <el-select id="deployment-offline-material" :model-value="modelValue" :disabled="disabled || importing" filterable clearable placeholder="选择已导入的离线包" no-match-text="没有找到匹配的离线包" no-data-text="暂无可用离线包，请先导入" aria-label="本次发布的离线包" @update:model-value="emit('update:modelValue',$event)">
          <el-option v-for="item in readyMaterials" :key="item.id" :value="item.id" :label="item.name" :title="item.name" />
        </el-select>
        <el-button :disabled="disabled || importing" :loading="importing" @click="fileInput?.click()">{{importing?'正在导入':'导入新包'}}</el-button>
        <input ref="fileInput" type="file" accept=".tar,.tar.gz,.tgz" aria-label="上传本次发布的离线包" hidden @change="importFile" />
      </div>
    </div>
    <div v-if="stage" class="offline-material-progress" role="status"><span>{{stage}}</span><el-progress v-if="progress!==null" :percentage="progress" :stroke-width="4" /></div>
    <p v-if="error" class="management-error management-feedback" role="alert">{{error}}</p>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { api, errorText } from '@/api/workspace'
import { uploadOfflineMaterial, type OfflineMaterial } from '@/api/bundles'
import DeploymentHelp from '@/components/DeploymentHelp.vue'
const props=withDefaults(defineProps<{modelValue?:number;kind:'bundle'|'image';materials:OfflineMaterial[];disabled?:boolean;showMaterials?:boolean}>(),{showMaterials:true})
const emit=defineEmits<{'update:modelValue':[value:number|undefined];imported:[material:OfflineMaterial];busy:[value:boolean]}>()
const fileInput=ref<HTMLInputElement>(),importing=ref(false),stage=ref(''),progress=ref<number|null>(null),error=ref('')
const controller=new AbortController()
const readyMaterials=computed(()=>props.materials.filter(item=>item.status==='ready' && (item.kind==='image')===(props.kind==='image')))
watch([()=>props.kind,()=>props.showMaterials],()=>{stage.value='';progress.value=null;error.value=''})
onBeforeUnmount(()=>controller.abort())
async function importFile(event:Event){
  const input=event.target as HTMLInputElement,file=input.files?.[0];input.value=''
  if(!file || props.disabled || importing.value)return
  importing.value=true;emit('busy',true);emit('update:modelValue',undefined);error.value=''
  try{
    let material=await uploadOfflineMaterial(file,props.kind,(message,value)=>{stage.value=message;progress.value=value},controller.signal)
    if(material.status==='failed'){await api('POST',`/bundles/${material.id}/retry`,undefined,controller.signal);material={...material,status:'queued'}}
    while(!['ready','failed'].includes(material.status)){
      stage.value='文件已上传，正在校验并加载离线镜像…';progress.value=null
      await new Promise<void>((resolve,reject)=>{const abort=()=>{clearTimeout(timer);reject(new DOMException('操作已取消','AbortError'))};const timer=setTimeout(()=>{controller.signal.removeEventListener('abort',abort);resolve()},1500);controller.signal.addEventListener('abort',abort,{once:true})})
      const rows=await api<OfflineMaterial[]>('GET','/bundles',undefined,controller.signal)
      const current=rows.find(item=>item.id===material.id)
      if(!current)throw new Error('导入记录不存在，请重新导入')
      material=current
    }
    if(material.status==='failed')throw new Error(material.error || '离线镜像导入失败，请重新选择文件重试')
    emit('imported',material);emit('update:modelValue',material.id);stage.value='离线材料已就绪，预览并确认后部署。';progress.value=null
  }catch(cause){if(!controller.signal.aborted){error.value=errorText(cause);stage.value='';progress.value=null}}finally{importing.value=false;emit('busy',false)}
}
</script>
<style scoped>
.offline-material-picker{display:flex;flex-direction:column;gap:8px;min-width:0;flex-shrink:0}.offline-material-controls{display:flex;flex-wrap:wrap;gap:8px 12px;align-items:center;min-width:0}.offline-material-selection{display:flex;flex:0 1 392px;align-items:center;gap:8px;min-width:0;max-width:100%}.offline-material-label{display:flex;align-items:center;gap:4px;flex-shrink:0;font-size:12px}.offline-material-selection>.el-select{width:260px;min-width:0;flex:0 1 260px}.offline-material-selection>.el-button{flex-shrink:0;margin:0}.offline-material-progress{font-size:12px;color:#aaa;display:grid;gap:6px;max-width:480px}.offline-material-picker p{margin:0}@media(max-width:640px){.offline-material-selection{display:grid;grid-template-columns:64px minmax(0,1fr)}.offline-material-selection>.el-select{width:100%}.offline-material-selection>.el-button{grid-column:2;justify-self:start}}
</style>
