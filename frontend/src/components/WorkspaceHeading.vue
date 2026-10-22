<template>
  <header class="workspace-heading"><h1>{{ title }}</h1><div class="management-heading-actions"><slot /><el-button v-if="refreshable" text circle :disabled="loading || busy" :aria-label="`刷新${title}`" :title="`刷新${title}`" @click="emit('refresh')"><PortainerIcon name="refresh" :class="{'management-spin':loading}" /></el-button></div></header>
</template>
<script setup lang="ts">
import PortainerIcon from '@/components/PortainerIcon.vue'
withDefaults(defineProps<{title: string; loading?: boolean; busy?: boolean; refreshable?: boolean}>(), {refreshable:true})
const emit = defineEmits<{refresh: []}>()
</script>
