<template>
  <header class="workspace-heading">
    <div class="workspace-heading-title">
      <el-button v-if="back" class="workspace-back" text circle :disabled="busy" :aria-label="back.label" :title="back.label" @click="goBack"><PortainerIcon name="back" /></el-button>
      <h1 :title="title">{{ title }}</h1>
    </div>
    <div class="management-heading-actions"><slot /><el-button v-if="refreshable" text circle :disabled="loading || busy" :aria-label="`刷新${title}`" :title="`刷新${title}`" @click="emit('refresh')"><PortainerIcon name="refresh" :class="{'management-spin':loading}" /></el-button><slot name="actions" /></div>
  </header>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { workspaceBack } from '@/utils/navigation'
const props = withDefaults(defineProps<{title: string; loading?: boolean; busy?: boolean; refreshable?: boolean}>(), {refreshable:true})
const emit = defineEmits<{refresh: []}>()
const route = useRoute(), router = useRouter()
const back = computed(() => workspaceBack(route, router.options.history.state.back, path => router.resolve(path)))
function goBack() {
  if (props.busy || !back.value) return
  if (back.value.history) router.back()
  else void router.push(back.value.to)
}
</script>
