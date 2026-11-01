<template>
  <div ref="editor" class="environment-parameter-pairs" :class="{'environment-parameter-pairs--readonly':readonly}">
    <div v-if="modelValue.length" class="environment-parameter-columns" aria-hidden="true"><span>参数名称</span><span>参数值</span><span v-if="!readonly" /></div>
    <p v-else class="management-form-help environment-parameter-empty">{{readonly?'暂无运行参数。':'暂无运行参数，可按需添加。'}}</p>
    <div v-for="(pair, index) in modelValue" :key="pair.id" class="environment-parameter-row">
      <div class="environment-parameter-key-control">
        <el-input v-model="pair.key" class="environment-parameter-name" :aria-label="`运行参数 ${index + 1} 名称`" :readonly="readonly || !!field(pair)" :title="field(pair)?'模板参数名称由应用定义固定':''" placeholder="例如：DB_HOST" :disabled="disabled" autocomplete="off" />
      </div>
      <div class="environment-parameter-value-control">
        <el-input v-model="pair.value" :aria-label="field(pair)?.label || `运行参数 ${index + 1} 值`" :aria-required="field(pair)?.required || undefined" :type="sensitive(pair)?'password':'text'" :show-password="sensitive(pair) && !readonly" :readonly="readonly" :inputmode="field(pair)?.type==='port'?'numeric':undefined" :placeholder="placeholder(pair)" :disabled="disabled" autocomplete="off" />
        <el-button v-if="!readonly && sensitive(pair)" :aria-label="`生成${field(pair)?.label || pair.key}`" :disabled="disabled" @click="generate(pair)">生成</el-button>
      </div>
      <el-tooltip v-if="!readonly" :disabled="!field(pair)" content="模板参数由应用定义固定，不能删除；请编辑参数值。">
        <span class="environment-parameter-remove-wrap"><el-button class="environment-parameter-remove" :aria-label="`删除运行参数 ${pair.key || index + 1}`" :disabled="disabled || !!field(pair)" @click="remove(pair)"><PortainerIcon name="trash" /></el-button></span>
      </el-tooltip>
      <p v-if="field(pair)" class="management-form-help environment-parameter-help"><span :class="{'environment-parameter-required':field(pair)?.required}">{{field(pair)?.label}}{{field(pair)?.required?'（必填）':''}}</span><span v-if="field(pair)?.help"> · {{field(pair)?.help}}</span><span v-if="grpcPort(pair)"> · 对应 gRPC 客户端端口：{{grpcPort(pair)}}，部署时一并映射。</span></p>
    </div>
    <el-button v-if="allowExtra && !readonly" link :disabled="disabled" @click="add"><PortainerIcon name="plus" />添加参数</el-button>
    <p v-if="!readonly" class="management-form-help">{{help}}</p>
    <p v-if="schema.length && !readonly" class="management-form-help">模板参数名称固定，标记「必填」的参数必须填写{{allowExtra?'；附加参数可添加、修改或删除。':'。'}}</p>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { isSensitiveEnvironmentParameter, type EnvironmentParameterPair } from '@/utils/applicationDetail'
import type { TemplateParameter } from '@/api/applicationTemplates'
const props = withDefaults(defineProps<{modelValue: EnvironmentParameterPair[]; schema?: TemplateParameter[]; disabled?: boolean; readonly?: boolean; allowExtra?: boolean; help?: string}>(), { schema: () => [], allowExtra: true, help: '名称以字母或下划线开头，可包含数字；参数仅保存在本环境。密码、令牌等敏感参数默认隐藏。' })
const emit = defineEmits<{'update:modelValue': [value: EnvironmentParameterPair[]]}>()
const editor = ref<HTMLElement>()
const fields = computed(() => new Map(props.schema.map(field => [field.key, field])))
function field(pair: EnvironmentParameterPair) { return pair.schemaKey ? fields.value.get(pair.schemaKey) : undefined }
function sensitive(pair: EnvironmentParameterPair) { return field(pair)?.type === 'password' || isSensitiveEnvironmentParameter(pair.key) }
function placeholder(pair: EnvironmentParameterPair) {
  const definition = field(pair)
  return definition?.type === 'password' ? '填写此环境的凭据' : definition?.default || (definition?.required ? `请输入${definition.label}` : '参数值，可留空')
}
function grpcPort(pair: EnvironmentParameterPair) {
  const port = Number(pair.value)
  return field(pair)?.key === 'NACOS_PORT' && /^\d+$/.test(pair.value) && port > 0 && port <= 64535 ? port + 1000 : 0
}
async function add() {
  if (props.disabled || props.readonly || !props.allowExtra) return
  const id = props.modelValue.reduce((max, pair) => Math.max(max, pair.id), 0) + 1
  emit('update:modelValue', [...props.modelValue, { id, key: '', value: '' }])
  await nextTick()
  const inputs = editor.value?.querySelectorAll<HTMLInputElement>('.environment-parameter-name input')
  inputs?.[inputs.length - 1]?.focus()
}
function remove(pair: EnvironmentParameterPair) {
  if (!props.disabled && !props.readonly && !field(pair)) emit('update:modelValue', props.modelValue.filter(row => row.id !== pair.id))
}
function generate(pair: EnvironmentParameterPair) {
  if (props.disabled || props.readonly || !sensitive(pair)) return
  const bytes = crypto.getRandomValues(new Uint8Array(32))
  pair.value = pair.key === 'NACOS_AUTH_TOKEN' ? btoa(String.fromCharCode(...bytes)) : Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
}
</script>
<style scoped>
.environment-parameter-pairs { min-width:0; width:100%; }
.environment-parameter-columns, .environment-parameter-row { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,1.4fr) 32px; gap:8px; align-items:start; margin-bottom:8px; }
.environment-parameter-columns { color:#999; font-size:12px; line-height:20px; }
.environment-parameter-row .el-input { min-width:0; }
.environment-parameter-key-control, .environment-parameter-value-control { display:flex; gap:8px; min-width:0; }
.environment-parameter-value-control .el-button { margin:0; }
.environment-parameter-remove-wrap { display:inline-flex; }
.environment-parameter-remove { width:32px; height:32px; margin:0; padding:0; color:#f1a7ac; }
.environment-parameter-empty { margin:0 0 8px; }
.environment-parameter-help { grid-column:1 / -1; margin:0 0 6px; overflow-wrap:anywhere; }
.environment-parameter-required { color:#d6d6d6; }
.environment-parameter-pairs--readonly .environment-parameter-columns, .environment-parameter-pairs--readonly .environment-parameter-row { grid-template-columns:minmax(0,1fr) minmax(0,1.4fr); }
@media(max-width:480px) {
  .environment-parameter-columns { display:none; }
  .environment-parameter-row { grid-template-columns:minmax(0,1fr) 32px; }
  .environment-parameter-pairs--readonly .environment-parameter-row { grid-template-columns:minmax(0,1fr); }
  .environment-parameter-key-control { grid-column:1; }
  .environment-parameter-value-control { grid-column:1; grid-row:2; }
  .environment-parameter-remove-wrap { grid-column:2; grid-row:1 / 3; align-self:center; }
  .environment-parameter-row + .environment-parameter-row { margin-top:12px; }
}
</style>
