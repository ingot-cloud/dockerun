<template><div class="workspace-page page-container"><WorkspaceHeading v-if="title || $slots.header" :title="title || ''" :loading="loading" :refreshable="refreshable" @refresh="emit('refresh')"><slot name="header" /></WorkspaceHeading><div class="page-content"><slot /></div></div></template>
<script setup lang="ts">
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
withDefaults(defineProps<{title?: string; loading?: boolean; refreshable?: boolean}>(), {refreshable:false})
const emit = defineEmits<{refresh: []}>()
</script>
<style scoped>
.page-content { display: flex; flex-direction: column; flex: 1; min-width: 0; min-height: 0; gap: 16px; }
</style>
