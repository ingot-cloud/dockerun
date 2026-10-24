<template>
  <div class="template-parameters">
    <el-form-item v-for="field in fields" :key="field.key" :label="field.label" :required="field.required" :class="{ 'template-parameter-wide':field.type==='image' }">
      <div class="template-parameter-control"><el-input :model-value="modelValue[field.key] || ''" :type="field.type==='password'?'password':'text'" :show-password="field.type==='password'" :disabled="disabled" :placeholder="field.type==='password'?'填写此环境的凭据':field.default || `请输入${field.label}`" :aria-label="field.label" autocomplete="off" @update:model-value="(value:string)=>update(field.key,value)" /><el-button v-if="field.type==='password'" :disabled="disabled" :aria-label="`生成${field.label}`" @click="generate(field.key)">生成</el-button></div>
      <p v-if="field.help" class="management-form-help">{{field.help}}</p><p v-if="field.key==='NACOS_PORT' && modelValue.NACOS_PORT" class="management-form-help">对应 gRPC 客户端端口：{{Number(modelValue.NACOS_PORT)+1000}}，部署时一并映射。</p>
      <el-button v-if="allowExtra && !schema.some(item=>item.key===field.key)" link :disabled="disabled" @click="remove(field.key)">移除此参数</el-button>
    </el-form-item>
    <div v-if="allowExtra" class="template-parameter-wide"><div class="template-parameter-control"><el-input v-model="extraKey" placeholder="附加参数名称，例如 APP_PROFILE" aria-label="附加参数名称" :disabled="disabled" /><el-button :disabled="disabled" @click="add">添加参数</el-button></div><p class="management-form-help">额外参数可用于脚本或自定义应用栈，仅保存在此环境。系统计算的 gRPC 端口不支持单独修改。</p><p v-if="extraError" class="management-form-help" role="alert">{{extraError}}</p></div>
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import type { TemplateParameter } from '@/api/applicationTemplates'
const props=defineProps<{schema:TemplateParameter[];modelValue:Record<string,string>;disabled?:boolean;allowExtra?:boolean}>()
const emit=defineEmits<{'update:modelValue':[value:Record<string,string>]}>()
const extraKey=ref(''),extraError=ref('')
const fields=computed(()=>[...props.schema,...(props.allowExtra?Object.keys(props.modelValue).filter(key=>key!=='NACOS_GRPC_PORT' && !props.schema.some(item=>item.key===key)).map(key=>({key,label:key,type:'password',default:'',required:false,help:'附加环境参数，值默认隐藏。'})):[])])
function add(){const key=extraKey.value.trim();extraError.value='';if(!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key) || key==='NACOS_GRPC_PORT'){extraError.value='请输入有效的变量名称，以字母或下划线开头';return}if(key in props.modelValue){extraError.value='此参数已存在';return}update(key,'');extraKey.value=''}
function remove(key:string){const values={...props.modelValue};delete values[key];emit('update:modelValue',values)}
function update(key:string,value:string){if(!props.disabled)emit('update:modelValue',{...props.modelValue,[key]:value})}
function generate(key:string){const bytes=crypto.getRandomValues(new Uint8Array(32));update(key,key==='NACOS_AUTH_TOKEN'?btoa(String.fromCharCode(...bytes)):Array.from(bytes,b=>b.toString(16).padStart(2,'0')).join(''))}
</script>
<style scoped>
.template-parameters { display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 20px;min-width:0;width:100%; }
.template-parameters :deep(.el-form-item__content) { display:block;min-width:0; }
.template-parameter-wide { grid-column:1 / -1; }
.template-parameter-control { display:flex;gap:8px;min-width:0;width:100%; }
.template-parameter-control .el-input { min-width:0; }
.template-parameter-control .el-button { margin:0; }
@media(max-width:760px){.template-parameters{grid-template-columns:minmax(0,1fr)}}
</style>
