<template>
  <section class="container-binding-editor">
    <el-form-item label="关联容器">
      <el-select :model-value="names || []" multiple filterable allow-create default-first-option :disabled="disabled" placeholder="输入容器名称后按回车，例如 test-nginx 或 $APP_NAME" aria-label="关联容器名称" @update:model-value="emit('update:names',$event)">
        <el-option v-for="name in names || []" :key="name" :value="name" :label="name" />
      </el-select>
      <p class="management-form-help">明确指定本环境在目标主机上的容器，可引用已保存的环境参数。不同环境不能关联同一主机的同名容器；关联后展示实际镜像名称和 Image ID。</p>
    </el-form-item>
    <el-switch v-if="command" :model-value="replace" :disabled="disabled" active-text="部署时由系统替换关联容器" aria-label="部署时由系统替换关联容器" @update:model-value="emit('update:replace',Boolean($event))" />
    <p v-if="command && replace" class="management-warning management-feedback" role="status">仅处理已关联容器：先停止并删除容器，再清理可删除的同 tag 本地镜像、拉取本次镜像并执行脚本。数据卷保留；拉取或执行失败不会自动恢复旧容器。</p>
    <p v-else class="management-form-help">{{command?'未开启系统替换，容器替换仍由脚本负责。':'Compose 使用应用项目自动更新服务，不需要开启命令容器替换。'}}</p>
  </section>
</template>
<script setup lang="ts">
defineProps<{names?:string[];replace?:boolean;disabled?:boolean;command?:boolean}>()
const emit=defineEmits<{'update:names':[names:string[]];'update:replace':[replace:boolean]}>()
</script>
<style scoped>
.container-binding-editor{margin:16px 0}.container-binding-editor :deep(.el-select){width:100%}.container-binding-editor :deep(.el-form-item){margin-bottom:12px}
</style>
