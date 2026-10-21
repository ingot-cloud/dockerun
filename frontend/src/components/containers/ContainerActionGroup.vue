<template>
  <div class="container-action-group" role="group" aria-label="容器操作">
    <el-tooltip v-for="action in containerActions" :key="action.key" :content="hint(action.key)" placement="top">
      <span class="action-wrapper"><el-button :class="{ destructive: action.key === 'remove' }" :disabled="busy || !targets(action.key).length" :loading="loading === action.key" @click="$emit('action', action.key)"><PortainerIcon v-if="loading !== action.key" :name="action.icon" />{{ action.label }}</el-button></span>
    </el-tooltip>
  </div>
</template>
<script setup lang="ts">
import PortainerIcon from '@/components/PortainerIcon.vue'
import { canContainerAction, containerActions, type Container, type ContainerAction } from '@/utils/containers'
const props = defineProps<{ containers: Container[]; busy?: boolean; loading?: string }>()
defineEmits<{ action: [action: ContainerAction] }>()
const targets = (action: ContainerAction) => props.containers.filter(container => canContainerAction(container, action))
function hint(action: ContainerAction) {
  const label = containerActions.find(item => item.key === action)!.label
  if (!props.containers.length) return `请先选择要${label}的容器`
  const count = targets(action).length
  if (!count) return action === 'remove' ? '请先停止容器，再执行删除' : `所选容器当前状态不支持${label}`
  return `对 ${count} 个容器执行${label}${count < props.containers.length ? '，跳过状态不适用的容器' : ''}`
}
</script>
<style scoped>
.container-action-group { display: inline-flex; flex-shrink: 0; white-space: nowrap; }
.action-wrapper { display: inline-flex; }
.action-wrapper + .action-wrapper { margin-left: -1px; }
.action-wrapper .el-button { margin: 0; height: 30px; padding: 0 10px; border-radius: 0; border: 1px solid #555; color: #ddd; background: transparent; font-size: 12px; font-weight: 400; }
.action-wrapper:first-child .el-button { border-radius: 5px 0 0 5px; }
.action-wrapper:last-child .el-button { border-radius: 0 5px 5px 0; }
.action-wrapper .el-button:not(.is-disabled):hover { position: relative; z-index: 1; background: #353535; border-color: #999; color: #fff; }
.action-wrapper .el-button.is-disabled { opacity: .4; }
.action-wrapper .el-button.destructive { background: #73312c; border-color: #8d4039; color: #ffd0ca; }
.action-wrapper .el-button.destructive:not(.is-disabled):hover { background: #943e35; border-color: #b55a4f; }
.action-wrapper :deep(.el-button > span) { gap: 5px; }
.portainer-icon { width: 13px; height: 13px; }
</style>
