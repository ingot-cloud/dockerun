<template>
  <div class="code-editor-wrapper">
    <div class="editor-label"><span>{{ label || '代码编辑' }}</span><el-button text :aria-label="`${readonly ? '放大查看' : '放大编辑'}：${label || '代码'}`" @click="openExpanded"><el-icon><FullScreen /></el-icon>{{ readonly ? '放大查看' : '放大编辑' }}</el-button></div>
    <vue-monaco-editor
      v-model:value="editorValue"
      :language="language"
      :theme="theme"
      :options="editorOptions"
      :height="height"
    />
    <el-dialog v-model="expanded" :title="expandedTitle" class="code-editor-dialog" :style="dialogStyle" append-to-body destroy-on-close :close-on-click-modal="false" :close-on-press-escape="false" @opened="expandedEditor?.focus()" @closed="expandedEditor=undefined">
      <p class="code-editor-dialog-help">{{ readonly ? '此内容为只读，可在放大窗口中查看。' : '修改会同步到当前表单，仍需保存或发布后生效。' }} 拖动右下角可调整窗口大小。</p>
      <div class="code-editor-dialog-content"><vue-monaco-editor v-model:value="editorValue" :language="language" :theme="theme" :options="editorOptions" height="100%" @mount="handleExpandedMount" /></div>
      <template #footer><el-button type="primary" @click="expanded=false">{{readonly ? '完成查看' : '完成编辑'}}</el-button></template>
      <button type="button" class="code-editor-resize-handle" aria-label="调整编辑窗口大小" title="拖动调整大小，或使用方向键调整" @pointerdown="startResize" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" @keydown="resizeWithKeyboard"><el-icon><BottomRight /></el-icon></button>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import "monaco-editor/esm/nls.messages.zh-cn.js";
import { ref, computed, getCurrentInstance, onMounted, onBeforeUnmount } from "vue";
import { FullScreen, BottomRight } from '@element-plus/icons-vue';
import {
  VueMonacoEditor,
  install as MonacoInstall,
} from "@guolao/vue-monaco-editor";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";
import "monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution";
import "monaco-editor/esm/vs/basic-languages/shell/shell.contribution";
import "monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
self.MonacoEnvironment = { getWorker: () => new EditorWorker() };
const app = getCurrentInstance()?.appContext.app;
if (app) MonacoInstall(app, { monaco });

interface Props {
  modelValue: string;
  language?: string;
  label?: string;
  height?: string | number;
  readonly?: boolean;
  minimap?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  language: "shell",
  height: "400px",
  readonly: false,
  minimap: false,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
}>();

const editorValue = computed({get: () => props.modelValue || '', set: value => { if (!props.readonly) emit('update:modelValue', value) }});
const expanded = ref(false);
const expandedTitle = computed(() => `${props.readonly ? '放大查看' : '放大编辑'} · ${props.label || '代码'}`);
const layout = ref({width: 1000, height: 700, left: 12, top: 12});
const dialogStyle = computed(() => Object.fromEntries(Object.entries(layout.value).map(([key, value]) => [key, `${value}px`])));
let expandedEditor: monaco.editor.IStandaloneCodeEditor | undefined;
let resizeStart: {id:number; x:number; y:number; width:number; height:number} | undefined;
function constrainLayout(next = layout.value) {
  const availableWidth = Math.max(1, window.innerWidth - 24), availableHeight = Math.max(1, window.innerHeight - 24);
  const minWidth = Math.min(480, availableWidth), minHeight = Math.min(320, availableHeight);
  const left = Math.max(12, Math.min(next.left, window.innerWidth - minWidth - 12));
  const top = Math.max(12, Math.min(next.top, window.innerHeight - minHeight - 12));
  layout.value = {left, top, width: Math.max(minWidth, Math.min(next.width, window.innerWidth - left - 12)), height: Math.max(minHeight, Math.min(next.height, window.innerHeight - top - 12))};
}
function openExpanded() {
  const width = Math.min(1100, window.innerWidth - 24), height = Math.min(800, window.innerHeight - 24);
  constrainLayout({width, height, left: (window.innerWidth-width)/2, top: (window.innerHeight-height)/2});
  expanded.value = true;
}
function startResize(event: PointerEvent) {
  if (event.button !== 0) return;
  event.preventDefault();
  resizeStart = {id:event.pointerId, x:event.clientX, y:event.clientY, width:layout.value.width, height:layout.value.height};
  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
}
function resize(event: PointerEvent) {
  if (!resizeStart || event.pointerId !== resizeStart.id) return;
  constrainLayout({...layout.value, width:resizeStart.width+event.clientX-resizeStart.x, height:resizeStart.height+event.clientY-resizeStart.y});
}
function stopResize() { resizeStart = undefined; }
function resizeWithKeyboard(event: KeyboardEvent) {
  if (!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(event.key)) return;
  event.preventDefault();event.stopPropagation();
  constrainLayout({...layout.value, width:layout.value.width+(event.key==='ArrowRight'?20:event.key==='ArrowLeft'?-20:0), height:layout.value.height+(event.key==='ArrowDown'?20:event.key==='ArrowUp'?-20:0)});
}
function handleExpandedMount(editor: monaco.editor.IStandaloneCodeEditor) { expandedEditor = editor; }
function handleViewportResize() { if(expanded.value)constrainLayout(); }
onMounted(() => window.addEventListener('resize', handleViewportResize));
onBeforeUnmount(() => window.removeEventListener('resize', handleViewportResize));

const theme = computed(() => "vs-dark");

const editorOptions = computed(() => ({
  automaticLayout: true,
  formatOnType: true,
  formatOnPaste: true,
  readOnly: props.readonly,
  ariaLabel: props.label || '代码编辑器',
  minimap: {
    enabled: props.minimap,
  },
  scrollBeyondLastLine: false,
  fontSize: 12,
  lineNumbers: "on",
  roundedSelection: false,
  scrollbar: {
    verticalScrollbarSize: 10,
    horizontalScrollbarSize: 10,
  },
  folding: true,
  renderLineHighlight: "all",
  selectOnLineNumbers: true,
  tabSize: 2,
  wordWrap: "on",
}));

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
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 12px;
  background: var(--color-bg-2);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 12px;
  font-weight: 500;
}
.editor-label > span { min-width: 0; overflow-wrap: anywhere; }
.editor-label .el-button { flex-shrink: 0; margin: 0; }
</style>
<style>
.code-editor-dialog.el-dialog { position: fixed; display: flex; flex-direction: column; margin: 0; padding: 0; background: #202020; border: 1px solid #383838; border-radius: 10px; max-width: calc(100vw - 24px); max-height: calc(100dvh - 24px); }
.code-editor-dialog .el-dialog__header { flex-shrink: 0; margin: 0; padding: 16px 48px 16px 20px; }
.code-editor-dialog .el-dialog__title { font-size: 16px; overflow-wrap: anywhere; }
.code-editor-dialog .el-dialog__body { display: flex; flex-direction: column; flex: 1; min-height: 0; max-height: none; gap: 12px; padding: 16px 20px; overflow: hidden; }
.code-editor-dialog-help { flex-shrink: 0; margin: 0; font-size: 12px; line-height: 20px; color: #999; }
.code-editor-dialog-content { flex: 1; min-width: 0; min-height: 0; overflow: hidden; border: 1px solid #383838; border-radius: 4px; }
.code-editor-dialog .el-dialog__footer { flex-shrink: 0; padding: 12px 32px 12px 20px; }
.code-editor-resize-handle { position: absolute; right: 2px; bottom: 2px; display: grid; place-items: center; width: 24px; height: 24px; padding: 0; border: 0; border-radius: 4px; background: transparent; color: #aaa; cursor: nwse-resize; touch-action: none; }
.code-editor-resize-handle:hover { background: #303030; color: #ebebeb; }.code-editor-resize-handle:focus-visible { outline: 2px solid #33a7d5; outline-offset: -2px; }
@media(max-width:760px) { .code-editor-dialog .el-dialog__body { padding: 12px; }.code-editor-dialog .el-dialog__header { padding-left: 12px; }.code-editor-dialog .el-dialog__footer { padding-left: 12px; } }
</style>
