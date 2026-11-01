<template>
  <span class="deployment-help" @keydown.esc.stop="visible=false">
    <el-popover v-model:visible="visible" placement="bottom-start" :width="320" trigger="click" popper-class="deployment-help-popover" popper-style="max-width: calc(100vw - 32px); --el-bg-color-overlay: #252525; --el-border-color-light: #414141; --el-text-color-regular: #ccc;">
      <template #reference><el-button native-type="button" class="deployment-help-button" text circle :aria-label="label"><PortainerIcon name="info" /></el-button></template>
      <div class="deployment-help-content" @keydown.esc.stop="visible=false">
        <strong>{{title}}</strong>
        <slot />
      </div>
    </el-popover>
  </span>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
defineProps<{label:string;title:string}>()
const visible=ref(false)
</script>
<style scoped>
.deployment-help{display:inline-flex;align-items:center;flex-shrink:0;vertical-align:middle}.deployment-help-button{width:20px;height:24px;padding:2px;color:#999}.deployment-help-content{font-size:12px;line-height:1.6;overflow-wrap:anywhere}.deployment-help-content>strong{display:block;margin-bottom:6px;color:#ddd}.deployment-help-content :deep(p){margin:0 0 8px}.deployment-help-content :deep(p:last-child){margin-bottom:0}
</style>
