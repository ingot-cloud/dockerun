<template>
  <div class="deployment-content-editor">
    <div class="deployment-file-toolbar">
      <div class="deployment-file-navigation"><div class="deployment-file-tabs" role="group" aria-label="部署文件">
        <el-button :aria-pressed="activeKey==='main'" :class="{selected:activeKey==='main'}" @click="activeKey='main'">{{ filename }}</el-button>
        <el-button v-for="file in files" :key="file.id" :aria-pressed="activeKey===`config:${file.id}`" :class="{selected:activeKey===`config:${file.id}`}" @click="activeKey=`config:${file.id}`">{{file.path || '未命名文件'}}</el-button>
      </div><el-button :disabled="disabled || readonly" @click="addFile"><PortainerIcon name="plus" />添加文件</el-button></div>
      <label class="deployment-language">高亮语法<el-select v-model="overrides[activeKey]" placeholder="自动识别" clearable aria-label="编辑器高亮语法"><el-option v-for="language in languages" :key="language.value" :value="language.value" :label="language.label" /></el-select></label>
    </div>
    <div v-if="activeFile" class="deployment-file-name"><label :for="filenameId">文件名</label><el-input :id="filenameId" v-model="activeFile.path" :disabled="disabled || readonly" placeholder="configs/app.yaml" /><el-button type="danger" :disabled="disabled || readonly" @click="removeFile">删除文件</el-button></div>
    <CodeEditor v-model="content" :document-key="activeKey==='main'?filename:activeKey" :filename="activeFilename" :language="overrides[activeKey] || undefined" :label="activeFilename" :readonly="disabled || readonly" preserve-content fill />
    <p v-if="activeFile" class="management-form-help">配置文件保存在任务目录的 configs/ 下，内容按原文保留。敏感值请使用运行参数。</p>
    <FormDialog v-model="adding" title="添加配置文件" :draft="newFile" confirm-text="添加文件" @confirm="confirmFile" @cancel="adding=false"><el-form label-position="top" @submit.prevent="confirmFile"><el-form-item label="文件名" required><el-input v-model="newFile.path" placeholder="configs/app.yaml" /></el-form-item><p v-if="fileError" class="management-error" role="alert">{{fileError}}</p></el-form></FormDialog>
  </div>
</template>
<script setup lang="ts">
import { computed, defineAsyncComponent, ref, useId, watch } from 'vue'
import FormDialog from './FormDialog.vue'
import PortainerIcon from './PortainerIcon.vue'
import { releaseConfigValues, type ReleaseConfigFile } from '@/utils/releaseConfigFiles'
const CodeEditor=defineAsyncComponent(()=>import('./CodeEditor.vue'))
const props=defineProps<{modelValue:string;files:ReleaseConfigFile[];filename:string;disabled?:boolean;readonly?:boolean}>()
const emit=defineEmits<{'update:modelValue':[value:string];'update:files':[files:ReleaseConfigFile[]]}>()
const activeKey=ref('main'),overrides=ref<Record<string,string>>({}),adding=ref(false),newFile=ref({path:'configs/'}),fileError=ref('')
const filenameId=useId()
const languages=[{value:'shell',label:'Shell'},{value:'yaml',label:'YAML'},{value:'json',label:'JSON'},{value:'ini',label:'环境变量 / INI'},{value:'dockerfile',label:'Dockerfile'},{value:'xml',label:'XML'},{value:'html',label:'HTML'},{value:'css',label:'CSS'},{value:'javascript',label:'JavaScript'},{value:'typescript',label:'TypeScript'},{value:'python',label:'Python'},{value:'java',label:'Java'},{value:'go',label:'Go'},{value:'sql',label:'SQL'},{value:'powershell',label:'PowerShell'},{value:'markdown',label:'Markdown'},{value:'plaintext',label:'纯文本'}]
const activeFile=computed(()=>props.files.find(file=>activeKey.value===`config:${file.id}`))
const activeFilename=computed(()=>activeFile.value?.path || props.filename)
const content=computed({get:()=>activeFile.value?.content ?? props.modelValue,set:value=>{if(props.disabled || props.readonly)return;if(activeFile.value)activeFile.value.content=value;else emit('update:modelValue',value)}})
watch(()=>props.files,()=>{if(activeKey.value!=='main' && !activeFile.value)activeKey.value='main'},{deep:true})
function addFile(){newFile.value={path:'configs/'};fileError.value='';adding.value=true}
function confirmFile(){
  const id=Math.max(0,...props.files.map(file=>file.id))+1
  const files=[...props.files,{id,path:newFile.value.path,content:''}]
  try{releaseConfigValues(files)}catch(cause){fileError.value=cause instanceof Error?cause.message:String(cause);return}
  emit('update:files',files);activeKey.value=`config:${id}`;adding.value=false
}
function removeFile(){emit('update:files',props.files.filter(file=>file!==activeFile.value));activeKey.value='main'}
</script>
<style scoped>
.deployment-content-editor{display:flex;flex-direction:column;flex:1 1 0;min-height:0;min-width:0;width:100%;overflow:hidden;--color-bg-2:#202020;--color-bg-3:#171717;--color-border:#3b3b3b}
.deployment-file-toolbar,.deployment-file-tabs,.deployment-file-navigation,.deployment-file-name{display:flex;align-items:center;gap:8px;min-width:0}
.deployment-file-toolbar{justify-content:space-between;margin-bottom:12px}
.deployment-file-toolbar,.deployment-file-name,.deployment-content-editor>.management-form-help{flex-shrink:0}
.deployment-file-navigation{flex:1;overflow:hidden}
.deployment-file-navigation>.el-button{flex-shrink:0;margin:0}
.deployment-file-tabs{flex:1;overflow-x:auto;scrollbar-width:thin;padding-bottom:3px}
.deployment-file-tabs .el-button{margin:0;flex-shrink:0;max-width:220px;min-height:32px}
.deployment-file-tabs .el-button :deep(span){overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.deployment-file-tabs .selected{background:#24343c;border-color:#35acd7;color:#8ed3ed}
.deployment-language{display:flex;align-items:center;gap:8px;flex-shrink:0;color:#aaa;font-size:12px}
.deployment-language .el-select{width:155px}
.deployment-file-name{margin-bottom:12px;flex-wrap:nowrap}
.deployment-file-name label{flex-shrink:0;font-size:12px}
.deployment-file-name .el-input{flex:1;min-width:0}
@media(max-width:760px){.deployment-file-toolbar{align-items:stretch;flex-direction:column;gap:6px;margin-bottom:8px}.deployment-file-navigation{flex:auto;width:100%}.deployment-language{align-self:flex-end}.deployment-file-name{gap:6px}}
</style>
