<template>
  <section class="container-binding-editor" :class="{'container-binding-editor--compact':compact}">
    <el-form-item label="关联容器">
      <template v-if="compact" #label><span class="container-binding-label">关联容器<DeploymentHelp label="查看关联容器说明" title="关联容器说明"><p>指定本环境在目标主机上的容器，可引用已保存的环境参数，例如 $APP_NAME。</p><p>不同环境不能关联同一主机的同名容器；关联后展示实际镜像名称和 Image ID。</p></DeploymentHelp></span></template>
      <el-select :model-value="names || []" multiple filterable allow-create default-first-option :disabled="disabled" placeholder="输入容器名称后按回车，例如 test-nginx 或 $APP_NAME" aria-label="关联容器名称" @update:model-value="emit('update:names',$event)">
        <el-option v-for="name in names || []" :key="name" :value="name" :label="name" />
      </el-select>
      <p v-if="!compact" class="management-form-help">明确指定本环境在目标主机上的容器，可引用已保存的环境参数。不同环境不能关联同一主机的同名容器；关联后展示实际镜像名称和 Image ID。</p>
    </el-form-item>
    <div v-if="command" class="container-binding-policy"><el-switch :model-value="replace" :disabled="disabled" active-text="部署时由系统替换关联容器" aria-label="部署时由系统替换关联容器" @update:model-value="emit('update:replace',Boolean($event))" /><DeploymentHelp v-if="compact" label="查看容器替换说明" title="容器替换说明"><template v-if="replace"><p>手动发布及历史版本部署：先准备镜像，再停止并删除已关联的旧容器并执行脚本；离线镜像只加载，不删除、不拉取。</p><p>自动发布才清理已存在且可删除的同 tag 镜像引用，再拉取固定摘要。</p><p>容器不存在时直接部署，数据卷保留；失败不会自动恢复旧容器。</p></template><p v-else>未开启系统替换，容器替换仍由脚本负责。</p></DeploymentHelp></div>
    <p v-if="!compact && command && replace" class="management-warning management-feedback" role="status">手动发布及历史版本部署：先准备镜像，再停止并删除已关联的旧容器并执行脚本；离线镜像只加载，不删除、不拉取。自动发布才清理已存在且可删除的同 tag 镜像引用，再拉取固定摘要。容器不存在时直接部署，数据卷保留；失败不会自动恢复旧容器。</p>
    <p v-else-if="!compact" class="management-form-help">{{command?'未开启系统替换，容器替换仍由脚本负责。':'Compose 使用应用项目自动更新服务，不需要开启命令容器替换。'}}</p>
  </section>
</template>
<script setup lang="ts">
import DeploymentHelp from '@/components/DeploymentHelp.vue'
defineProps<{names?:string[];replace?:boolean;disabled?:boolean;command?:boolean;compact?:boolean}>()
const emit=defineEmits<{'update:names':[names:string[]];'update:replace':[replace:boolean]}>()
</script>
<style scoped>
.container-binding-editor{margin:16px 0}.container-binding-editor :deep(.el-select){width:100%}.container-binding-editor :deep(.el-form-item){margin-bottom:12px}
.container-binding-editor--compact{margin:0 0 18px}.container-binding-label,.container-binding-policy{display:flex;align-items:center;gap:6px;flex-wrap:wrap;min-width:0}
</style>
