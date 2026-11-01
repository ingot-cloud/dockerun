<template><span class="state" :class="`state-${tone}`" :title="value">{{ label || statusLabel(value) }}</span></template>
<script setup lang="ts">
import { computed } from 'vue'
import { statusLabel } from '@/utils/management'
const props = defineProps<{ value?: string; label?: string }>()
const tone = computed(() => ['success','healthy','ready','enabled','online','local'].includes(props.value || '') ? 'success' : ['failed','unhealthy'].includes(props.value || '') ? 'danger' : ['paused','unverified','needs_attention','processing','expired'].includes(props.value || '') ? 'warning' : ['queued','pending','running','loading','validating'].includes(props.value || '') ? 'info' : 'neutral')
</script>
