<template>
  <div class="code-editor-wrapper">
    <div v-if="label" class="editor-label">{{ label }}</div>
    <vue-monaco-editor
      v-model:value="editorValue"
      :language="language"
      :theme="theme"
      :options="editorOptions"
      :height="height"
      @mount="handleMount"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'

interface Props {
  modelValue: string
  language?: string
  label?: string
  height?: string | number
  readonly?: boolean
  minimap?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  language: 'shell',
  height: '400px',
  readonly: false,
  minimap: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const editorValue = ref(props.modelValue)

// 监听外部变化
watch(() => props.modelValue, (newVal) => {
  if (newVal !== editorValue.value) {
    editorValue.value = newVal
  }
})

// 监听内部变化
watch(editorValue, (newVal) => {
  emit('update:modelValue', newVal)
})

const theme = computed(() => 'vs-dark')

const editorOptions = computed(() => ({
  automaticLayout: true,
  formatOnType: true,
  formatOnPaste: true,
  readOnly: props.readonly,
  minimap: {
    enabled: props.minimap
  },
  scrollBeyondLastLine: false,
  fontSize: 14,
  lineNumbers: 'on',
  roundedSelection: false,
  scrollbar: {
    verticalScrollbarSize: 10,
    horizontalScrollbarSize: 10
  },
  folding: true,
  renderLineHighlight: 'all',
  selectOnLineNumbers: true,
  tabSize: 2,
  wordWrap: 'on'
}))

const handleMount = (_editor: any) => {
  // 编辑器挂载后的回调，可以进行额外配置
  console.log('Monaco Editor mounted')
}
</script>

<style scoped>
.code-editor-wrapper {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: var(--border-radius);
  overflow: hidden;
  background: var(--color-bg-3);
}

.editor-label {
  padding: 8px 12px;
  background: var(--color-bg-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 13px;
  font-weight: 500;
}
</style>
