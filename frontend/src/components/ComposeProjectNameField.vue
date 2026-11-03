<template>
  <el-form-item>
    <template #label><span class="compose-project-label">Compose 项目名称（可选）<DeploymentHelp label="查看 Compose 项目名称说明" title="Compose 项目名称"><p>用于将此环境的容器、默认网络和项目数据卷归入同一组。留空沿用系统生成的名称。</p><p>优先于 Compose 顶层 name 和运行参数中的 COMPOSE_PROJECT_NAME；脚本中的 COMPOSE_PROJECT_NAME 由系统按此配置提供。</p><p>仅在首次提交部署任务前可修改。提交后即固定名称，包括排队或失败的任务，保护已有资源和数据卷归属。</p><p>同一目标主机上的不同环境不能使用相同名称。须以小写字母或数字开头，仅含小写字母、数字、连字符和下划线，最多 100 个字符。</p></DeploymentHelp></span></template>
    <el-input :model-value="modelValue" :disabled="disabled || environment?.compose_project_name_locked" :placeholder="automaticName?`留空使用 ${automaticName}`:'留空，保存环境后自动生成'" maxlength="100" clearable aria-label="Compose 项目名称（可选）" @update:model-value="$emit('update:modelValue',$event)" />
    <p class="management-form-help compose-project-hint">{{environment?.compose_project_name_locked?'已提交部署任务，项目名称已固定':'部署前可设置，留空自动生成'}}<template v-if="effectiveName"> · 实际名称：<code>{{effectiveName}}</code></template></p>
  </el-form-item>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import DeploymentHelp from '@/components/DeploymentHelp.vue'
import { composeProjectName, type ProjectEnvironment } from '@/utils/composeProject'
const props=defineProps<{modelValue:string;environment?:ProjectEnvironment;disabled?:boolean}>()
defineEmits<{ 'update:modelValue':[value:string] }>()
const automaticName=computed(()=>composeProjectName(props.environment,''))
const effectiveName=computed(()=>composeProjectName(props.environment,props.modelValue))
</script>
<style scoped>
.compose-project-label{display:inline-flex;align-items:center;gap:6px}.compose-project-hint{overflow-wrap:anywhere}.compose-project-hint code{font-size:12px}
</style>
