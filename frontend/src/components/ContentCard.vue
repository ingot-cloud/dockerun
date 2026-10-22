<template><section class="management-card content-card"><header v-if="$slots.header || title" class="management-toolbar"><slot name="header"><h2><span v-if="icon" class="management-section-icon"><el-icon><component :is="icon" /></el-icon></span>{{ title }}</h2><div class="management-actions"><slot name="actions" /></div></slot></header><div class="card-body" :class="{'no-padding':noPadding}"><slot /></div><footer v-if="$slots.footer" class="management-pagination"><slot name="footer" /></footer></section></template>
<script setup lang="ts">
import type { Component } from 'vue'
defineProps<{title?: string; icon?: Component; noPadding?: boolean}>()
</script>
<style scoped>
.content-card { display: flex; flex-direction: column; flex: 1; min-height: 0; }.card-body { padding: 20px; min-height: 0; min-width: 0; display: flex; flex-direction: column; flex: 1; }.card-body.no-padding { padding: 0; }@media(max-width:760px) { .card-body { padding: 16px; } }
</style>
