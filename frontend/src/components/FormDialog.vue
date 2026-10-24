<template>
  <el-dialog v-bind="$attrs" :model-value="modelValue" align-center append-to-body destroy-on-close :close-on-click-modal="false" :close-on-press-escape="!loading" :show-close="!loading" :before-close="beforeClose" :class="['form-dialog', {wide, 'extra-wide':extraWide}]" :title="title" @update:model-value="emit('update:modelValue', $event)">
    <template #header><div class="dialog-header"><el-icon v-if="icon"><component :is="icon" /></el-icon><span>{{ title }}</span></div></template>
    <div class="dialog-body"><slot /></div>
    <template v-if="$slots.footer || showFooter" #footer><div class="dialog-footer"><slot name="footer"><el-button :disabled="loading" @click="cancel">{{ cancelText }}</el-button><el-button type="primary" :loading="loading" @click="emit('confirm')">{{ confirmText }}</el-button></slot></div></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { watch, type Component } from 'vue'
import { useUnsavedChanges } from '@/composables/useUnsavedChanges'
const props=withDefaults(defineProps<{modelValue?:boolean; draft?:unknown; title?: string; icon?:Component; loading?:boolean; wide?:boolean; extraWide?:boolean; showFooter?:boolean; cancelText?:string; confirmText?:string}>(),{modelValue:false,showFooter:true,cancelText:'取消',confirmText:'保存更改'})
const emit=defineEmits<{cancel:[];confirm:[];'update:modelValue':[value:boolean]}>()
const { markSaved, confirmDiscard } = useUnsavedChanges(() => props.draft, { enabled: () => props.modelValue && props.draft !== undefined, busy: () => Boolean(props.loading) })
watch(() => props.modelValue, open => { if(open)markSaved() }, { flush:'post', immediate:true })
async function beforeClose(done:()=>void){if(!props.loading && await confirmDiscard())done()}
async function cancel(){if(!props.loading && await confirmDiscard())emit('cancel')}
</script>
<style>
.form-dialog.el-dialog { width: 580px; padding: 0; background: #202020; border: 1px solid #383838; border-radius: 10px; }.form-dialog.wide { width: 720px; }.form-dialog.extra-wide { width: 1000px; }.form-dialog .el-dialog__header { margin: 0; padding: 18px 24px; }.form-dialog .el-dialog__body { padding: 0; max-height: min(68vh,calc(100dvh - 150px)); overflow: auto; }.form-dialog .dialog-header { display: flex; align-items: center; gap: 10px; font-size: 16px; font-weight: 600; }.form-dialog .dialog-header .el-icon { color:#c6d4e5; }.form-dialog .dialog-body { padding: 20px 24px; }.form-dialog .el-dialog__footer { padding: 16px 24px; }.form-dialog .dialog-footer { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: 10px; }.form-dialog .dialog-footer .el-button { margin: 0; }.form-dialog .form-section { margin-bottom: 24px; padding-bottom: 20px; border-bottom: 1px solid #333; }.form-dialog .form-section:last-child { margin-bottom: 0; border: 0; }.form-dialog .form-section h3 { margin: 0 0 16px; font-size: 14px; font-weight: 500; }.form-dialog .form-tip { flex-basis: 100%; margin-top: 6px; color: #999; font-size: 12px; line-height: 20px; }.form-dialog .el-form-item { margin-bottom: 22px; }.form-dialog .el-form-item__content { min-width: 0; }.form-dialog .el-select { width: 100%; }@media(max-width:760px){.form-dialog .dialog-body { padding:16px; }.form-dialog .el-dialog__header,.form-dialog .el-dialog__footer { padding:16px; }.form-dialog .el-form-item { display: block; }.form-dialog .el-form-item__label { display: block; width: auto !important; text-align: left; padding:0; }.form-dialog .el-form-item__content { margin-left:0 !important; }}
</style>
