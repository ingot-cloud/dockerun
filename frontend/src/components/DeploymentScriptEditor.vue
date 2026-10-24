<template>
  <div class="deployment-script-editor">
    <p class="management-form-help">脚本在实际目标主机的任务目录中由 Bash 执行，环境参数同时作为脚本环境变量。退出码非零会终止发布；预检查页面不会执行脚本。</p>
    <el-tabs v-model="stage">
      <el-tab-pane v-for="item in stages" :key="item.key" :name="item.key" :label="`${item.label}${modelValue[item.key]?.trim() ? ' · 已配置' : ''}`" />
    </el-tabs>
    <p class="management-form-help script-stage-help">{{ stages.find(item => item.key === stage)?.help }}</p>
    <CodeEditor :model-value="modelValue[stage]" language="shell" height="240px" :readonly="disabled" :label="stages.find(item => item.key === stage)?.label" @update:model-value="update" />
    <el-collapse class="script-variables">
      <el-collapse-item title="可用变量与部署脚本示例" name="variables">
        <p class="management-form-help"><code>WORK_DIR</code>：任务目录；<code>COMPOSE_FILE</code>：锁定镜像后的应用栈文件；<code>COMPOSE_PROJECT_NAME</code>：本应用环境的项目名；<code>DOCKERUN_ENVIRONMENT</code>、<code>DOCKERUN_VERSION</code>：环境和版本。</p>
        <p class="management-form-help">自定义部署脚本需要启动上述应用栈，系统仍会核对服务和镜像身份。脚本及其生成文件仅属于当前任务，持久数据请使用环境参数指定的挂载目录。自动恢复只恢复上一版本的服务，不会撤销脚本对外部系统的修改。</p>
        <pre class="management-code script-example">{{ example }}</pre>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import CodeEditor from '@/components/CodeEditor.vue'
import type { DeploymentScripts } from '@/utils/releases'
const props = defineProps<{ modelValue: DeploymentScripts; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: DeploymentScripts] }>()
const stage = ref<keyof DeploymentScripts>('before')
const stages: Array<{key: keyof DeploymentScripts; label: string; help: string}> = [
  { key: 'before', label: '部署前脚本', help: '镜像准备完成后执行，可准备挂载目录或外部资源，随后系统检查端口、挂载、网络和卷。留空则跳过。' },
  { key: 'deploy', label: '部署脚本', help: '留空由系统启动 Compose。填写后由此脚本接管启动步骤，请使用系统生成的应用栈文件和项目名。' },
  { key: 'after', label: '部署后脚本', help: '服务启动并通过检查后执行，可运行额外检查或初始化命令。失败会将本次发布标为失败。留空则跳过。' },
]
const example = 'docker compose --env-file "$WORK_DIR/empty.env" \\\n  --project-name "$COMPOSE_PROJECT_NAME" --project-directory "$WORK_DIR" \\\n  -f "$COMPOSE_FILE" up -d --pull never --no-build --wait --wait-timeout 120'
function update(value: string) { if (!props.disabled) emit('update:modelValue', { ...props.modelValue, [stage.value]: value }) }
</script>
<style scoped>
.deployment-script-editor { min-width: 0; width: 100%; }
.script-stage-help { margin: 0 0 12px; }
.script-variables { margin-top: 12px; }
.script-example { margin-top: 12px; }
.script-variables code { overflow-wrap: anywhere; }
.deployment-script-editor :deep(.el-tabs__item) { font-size: 12px; padding: 0 12px; }
</style>
