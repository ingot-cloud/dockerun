<template>
  <FormDialog :model-value="modelValue" :title="`删除${kind}`" :loading="deleting" :show-footer="false" @update:model-value="(value:boolean)=>{if(!deleting)emit('update:modelValue',value)}"><p class="management-description">确定删除以下 {{targets.length}} 项{{kind}}？{{warning}}</p><ul class="management-delete-targets"><li v-for="target in targets" :key="target.id"><strong>{{target.name}}</strong><span>{{removed.has(target.id)?'已删除':`#${target.id}`}}</span></li></ul><div v-if="failures.length" class="management-error" role="alert"><div><p v-for="failure in failures" :key="failure.id">{{failure.name}}：{{failure.error}}</p></div></div><template #footer><el-button :disabled="deleting" @click="emit('update:modelValue',false)">{{failures.length?'关闭':'取消'}}</el-button><el-button type="danger" :loading="deleting" :disabled="!pending.length" @click="remove">{{failures.length?'重试失败项':'确认删除'}}</el-button></template></FormDialog>
</template>
<script setup lang="ts">
import { computed,onBeforeUnmount,ref,watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api,errorText } from '@/api/workspace'
import FormDialog from '@/components/FormDialog.vue'
interface Target {id:number;name:string}
const props=defineProps<{modelValue:boolean;items:Target[];kind:string;endpoint:string;warning:string}>()
const emit=defineEmits<{'update:modelValue':[value:boolean];removed:[ids:number[]]}>()
const targets=ref<Target[]>([]),removed=ref(new Set<number>()),deleting=ref(false),failures=ref<(Target & {error:string})[]>([])
const pending=computed(()=>targets.value.filter(target=>!removed.value.has(target.id)))
let disposed=false
watch(()=>props.modelValue,open=>{if(open){targets.value=props.items.map(item=>({id:item.id,name:item.name}));removed.value=new Set();failures.value=[]}})
async function remove(){if(deleting.value || !pending.value.length)return;deleting.value=true;failures.value=[];const succeeded:number[]=[];try{for(const target of pending.value){if(disposed)break;try{await api('DELETE',`${props.endpoint}/${target.id}`);removed.value.add(target.id);succeeded.push(target.id)}catch(cause){failures.value.push({...target,error:errorText(cause)})}}if(disposed)return;if(succeeded.length){emit('removed',succeeded);ElMessage.success(`已删除 ${succeeded.length} 项${props.kind}`)}if(!failures.value.length)emit('update:modelValue',false)}finally{deleting.value=false}}
onBeforeUnmount(()=>{disposed=true})
</script>
<style scoped>
.management-delete-targets { list-style:none; margin:16px 0; max-height:220px; overflow:auto; }.management-delete-targets li { display:flex; justify-content:space-between; gap:16px; padding:10px 0; border-bottom:1px solid #333; }.management-delete-targets strong { font-size:13px; font-weight:500; overflow-wrap:anywhere; }.management-delete-targets span { color:#999; font-size:12px; white-space:nowrap; }
</style>
