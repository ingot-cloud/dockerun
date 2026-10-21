<template>
  <section class="resource-card"><div class="resource-card-header"><h2><span class="resource-icon"><PortainerIcon name="info" /></span>检查</h2><el-radio-group v-model="mode" size="small" aria-label="检查视图"><el-radio-button value="tree">树形</el-radio-button><el-radio-button value="text">文本</el-radio-button></el-radio-group></div><JsonTreeNode v-if="mode === 'tree'" :value="details" /><pre v-else class="inspect-text" tabindex="0" aria-label="容器原始 JSON">{{ JSON.stringify(details, null, 2) }}</pre></section>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
import JsonTreeNode from './JsonTreeNode.vue'
import type { ContainerDetails } from '@/utils/containers'
defineProps<{ details: ContainerDetails }>()
const mode = ref('tree')
</script>
<style scoped>
.inspect-text { white-space: pre; overflow: auto; color: #a9bfe1; font: 12px/1.8 Menlo, Monaco, Consolas, monospace; }
</style>
