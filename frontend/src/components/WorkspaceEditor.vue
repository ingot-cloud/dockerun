<template>
  <div class="workspace-editor" :class="{'workspace-editor--fill':fill}">
    <div class="workspace-editor-header">
      <div class="workspace-editor-page workspace-editor-header-content">
        <WorkspaceHeading class="workspace-editor-heading" :title="title" :loading="loading" :busy="busy" :refreshable="refreshable" @refresh="emit('refresh')">
          <template #actions><div v-if="$slots.actions" class="workspace-editor-actions"><slot name="actions" /></div></template>
        </WorkspaceHeading>
        <div v-if="$slots.navigation" class="workspace-editor-navigation"><slot name="navigation" /></div>
        <div v-if="$slots.feedback" class="workspace-editor-feedback"><slot name="feedback" /></div>
      </div>
    </div>
    <div ref="scroll" class="workspace-editor-scroll"><slot /></div>
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
const props = withDefaults(defineProps<{ title: string; loading?: boolean; busy?: boolean; refreshable?: boolean; resetKey?: string | number; fill?:boolean }>(), { refreshable: true })
const emit = defineEmits<{ refresh: [] }>()
const scroll = ref<HTMLDivElement>()
watch(() => props.resetKey, () => scroll.value?.scrollTo({ top: 0 }), { flush: 'post' })
</script>
