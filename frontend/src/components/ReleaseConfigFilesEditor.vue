<template>
  <div ref="editor" class="release-config-files">
    <p class="management-form-help release-config-help">文件名以 configs/ 开头，例如 configs/app.yaml，可使用子目录。内容按原文保存；密码、令牌等请在目标环境运行参数中配置。</p>
    <p v-if="!modelValue.length" class="management-form-help release-config-empty">暂无非敏感配置文件，可按需添加。</p>
    <section v-for="(file,index) in modelValue" :key="file.id" class="release-config-file" :aria-label="`配置文件 ${index + 1}`">
      <div class="release-config-file-heading">
        <el-form-item label="文件名" required class="release-config-filename"><el-input v-model="file.path" :aria-label="`配置文件 ${index + 1} 文件名`" placeholder="例如：configs/app.yaml" :disabled="disabled" autocomplete="off" /></el-form-item>
        <el-button type="danger" :aria-label="`删除配置文件 ${file.path || index + 1}`" :disabled="disabled" @click="remove(file.id)"><PortainerIcon name="trash" />删除文件</el-button>
      </div>
      <CodeEditor v-model="file.content" :filename="file.path" preserve-content :label="`${file.path || '未命名文件'} · 文件内容`" :readonly="disabled" />
    </section>
    <el-button :disabled="disabled" @click="add"><PortainerIcon name="plus" />添加文件</el-button>
  </div>
</template>
<script setup lang="ts">
import { defineAsyncComponent, nextTick, ref } from 'vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
import type { ReleaseConfigFile } from '@/utils/releaseConfigFiles'
const props = defineProps<{modelValue:ReleaseConfigFile[];disabled?:boolean}>()
const emit = defineEmits<{'update:modelValue':[files:ReleaseConfigFile[]]}>()
const CodeEditor = defineAsyncComponent(() => import('@/components/CodeEditor.vue'))
const editor = ref<HTMLElement>()
async function add() {
  if (props.disabled) return
  const id = props.modelValue.reduce((max,file) => Math.max(max,file.id),0) + 1
  emit('update:modelValue',[...props.modelValue,{id,path:'configs/',content:''}])
  await nextTick()
  const inputs = editor.value?.querySelectorAll<HTMLInputElement>('.release-config-filename input')
  const input = inputs?.[inputs.length - 1]
  input?.focus()
  input?.setSelectionRange(input.value.length,input.value.length)
}
function remove(id:number) {
  if (!props.disabled) emit('update:modelValue',props.modelValue.filter(file => file.id !== id))
}
</script>
<style scoped>
.release-config-files { min-width:0; width:100%; }
.release-config-help { margin:0 0 12px; }
.release-config-empty { margin:0 0 12px; }
.release-config-file { min-width:0; margin-bottom:16px; padding:12px; border:1px solid #383838; border-radius:6px; }
.release-config-file-heading { display:flex; align-items:flex-end; gap:12px; min-width:0; margin-bottom:12px; }
.release-config-filename { flex:1; min-width:0; margin-bottom:0; }
.release-config-file-heading > .el-button { flex-shrink:0; margin:0; }
.release-config-filename :deep(.el-form-item__content) { min-width:0; }
@media(max-width:480px) {
  .release-config-file { padding:10px; }
  .release-config-file-heading { gap:8px; }
}
</style>
