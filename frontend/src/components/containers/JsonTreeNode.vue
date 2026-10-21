<template>
  <div class="json-node">
    <div v-if="isObject" class="json-branch"><button class="json-toggle" :aria-expanded="expanded" :aria-label="`${expanded ? '收起' : '展开'} ${label || '根节点'}`" @click="expanded = !expanded">{{ expanded ? '▾' : '▸' }}</button><span v-if="label" class="json-key">{{ label }}: </span><span>{{ Array.isArray(value) ? '[' : '{' }}</span><span v-if="!expanded" class="json-summary">{{ entries.length }} 项 {{ Array.isArray(value) ? ']' : '}' }}</span></div>
    <template v-if="isObject && expanded"><div class="json-children"><JsonTreeNode v-for="[key, entry] in entries" :key="key" :label="Array.isArray(value) ? '' : key" :value="entry" :depth="depth + 1" /></div><div class="json-close">{{ Array.isArray(value) ? ']' : '}' }}</div></template>
    <div v-else-if="!isObject" class="json-leaf"><span v-if="label" class="json-key">{{ label }}: </span><span :class="`json-${value === null ? 'null' : typeof value}`">{{ JSON.stringify(value) }}</span></div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
const props = withDefaults(defineProps<{ value: unknown; label?: string; depth?: number }>(), { label: '', depth: 0 })
const expanded = ref(true)
const isObject = computed(() => props.value !== null && typeof props.value === 'object')
const entries = computed(() => isObject.value ? Object.entries(props.value as Record<string, unknown>).sort(([a], [b]) => Array.isArray(props.value) ? Number(a) - Number(b) : a.localeCompare(b)) : [])
</script>
<style scoped>
.json-node { font: 12px/1.8 Menlo, Monaco, Consolas, monospace; color: #a9b5ce; overflow-wrap: anywhere; }.json-children { padding-left: 18px; }.json-toggle { width: 18px; border: 0; padding: 0; background: transparent; color: #b8c9dc; cursor: pointer; font: inherit; }.json-key { color: #ccd3df; }.json-string { color: #93bbff; }.json-number, .json-boolean { color: #c3a8f2; }.json-null, .json-summary { color: #8d98aa; }.json-close, .json-leaf { padding-left: 18px; }
</style>
