<template>
  <div class="workspace-page">
    <WorkspaceHeading title="历史脚本" :loading="loading" :busy="busy" @refresh="loadScripts" />
    <ResourceList title="脚本列表" icon="Document" :items="scripts" :loading="loading" :error="listError" :busy="busy" :search-keys="['name','description','container_name']" :sorts="[{key:'name',label:'名称'},{key:'created_at',label:'创建时间'}]" :filter-options="[{value:'idle',label:'空闲'},{value:'running',label:'执行中'},{value:'failed',label:'失败'},{value:'stopped',label:'已停止'}]" filter-key="status" empty-text="暂无历史脚本" @refresh="loadScripts" @selection-change="selectedScripts=$event">
      <template #actions><el-button type="danger" :disabled="!selectedScripts.length || busy || loading" @click="deleteVisible=true">删除</el-button><el-button :disabled="busy" @click="showTemplates">旧脚本模板</el-button><el-button type="primary" :disabled="busy" @click="handleCreate">添加脚本</el-button></template>
      <el-table-column type="selection" width="48" :selectable="(row:DeployScript)=>row.status!=='running' && !busy && !loading" /><el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip /><el-table-column prop="description" label="说明" min-width="200" show-overflow-tooltip /><el-table-column label="文件数量" width="100"><template #default="{row}">{{row.files?.length || 0}}</template></el-table-column><el-table-column label="关联容器" min-width="180"><template #default="{row}"><router-link v-if="row.container_id" :to="`/containers/${row.container_id}`">{{row.container_name || row.container_id.slice(0,12)}}</router-link><span v-else class="management-muted">—</span></template></el-table-column><el-table-column label="状态" width="140"><template #default="{row}"><StatusBadge :value="row.status || 'idle'" /></template></el-table-column><el-table-column label="创建时间" width="180"><template #default="{row}">{{formatDate(row.created_at)}}</template></el-table-column><el-table-column label="操作" width="170"><template #default="{row}"><div class="management-row-actions"><el-button link type="primary" :disabled="busy || row.status==='running'" @click="handleExecute(row)">执行</el-button><el-button link :disabled="busy || row.status==='running'" @click="handleEdit(row)">编辑</el-button></div></template></el-table-column>
      <template #feedback><div v-if="actionError" class="management-error" role="alert"><span>{{actionError}}</span><el-button link @click="actionError=''">关闭</el-button></div></template><template #count><span v-if="selectedScripts.length">已选 {{selectedScripts.length}} 项</span></template>
    </ResourceList>
    <p class="management-description">此入口用于兼容已有独立脚本。部署新服务请使用应用模板；执行中的脚本暂不可编辑或删除。<router-link to="/application-templates">前往应用模板</router-link></p>
    <FormDialog v-model="dialogVisible" :draft="form" :title="isEdit?'编辑脚本':'添加脚本'" :icon="Document" :loading="saving" @confirm="handleSave" @cancel="dialogVisible=false" extra-wide>
      <div v-if="formError" class="management-error management-feedback" role="alert">{{formError}}</div>
      <el-form :model="form" label-position="top" :disabled="saving" @submit.prevent="handleSave">
        <div class="form-section"><h3>基本信息</h3><el-form-item label="脚本名称" required><el-input v-model="form.name" placeholder="例如：部署网关服务" maxlength="100" /></el-form-item><el-form-item label="说明"><el-input v-model="form.description" placeholder="填写脚本用途（可选）" /></el-form-item><el-form-item label="工作目录"><el-input v-model="form.work_dir" placeholder="例如：/ingot-data/dockerun/my-app" /><p class="management-form-help">脚本及文件保存在该目录；目录不存在时由服务端按执行规则创建。</p></el-form-item></div>
        <div class="form-section"><h3>执行方式</h3><p class="management-form-help">历史脚本仅支持手动执行。镜像推送自动发布请在应用的环境配置中开启。</p></div>
        <div class="form-section"><h3>主执行脚本</h3><CodeEditor v-model="form.main_script" :label="`${form.name || '未命名脚本'} · 主执行脚本`" language="shell" height="360px" :readonly="saving" /><p class="management-form-help">负责准备目录、复制配置和启动服务；请确认脚本实际执行的操作。</p></div>
        <div class="form-section"><div class="script-section-heading"><h3>附加文件</h3><el-button :icon="Plus" @click="addFile">添加文件</el-button></div><div class="script-files-scroll"><el-table :data="form.files" empty-text="暂无附加文件" max-height="360"><el-table-column label="文件名称" min-width="180"><template #default="{row}"><el-input v-model="row.file_name" placeholder="例如：docker-compose.yml" aria-label="文件名称" /></template></el-table-column><el-table-column label="相对路径" min-width="190"><template #default="{row}"><el-input v-model="row.file_path" placeholder="例如：./docker-compose.yml" aria-label="文件相对路径" /></template></el-table-column><el-table-column label="类型" width="140"><template #default="{row}"><el-select v-model="row.file_type" aria-label="文件类型"><el-option label="应用栈定义" value="compose" /><el-option label="配置文件" value="config" /><el-option label="执行脚本" value="script" /><el-option label="其他文件" value="other" /></el-select></template></el-table-column><el-table-column label="说明" min-width="170"><template #default="{row}"><el-input v-model="row.description" placeholder="文件用途（可选）" aria-label="文件说明" /></template></el-table-column><el-table-column label="操作" width="130" fixed="right"><template #default="{row,$index}"><el-button link :aria-label="`编辑 ${row.file_name || '文件'} 内容`" @click="editFileContent(row)">编辑</el-button><el-button link type="danger" :aria-label="`移除 ${row.file_name || '文件'}`" @click="form.files?.splice($index,1)">移除</el-button></template></el-table-column></el-table></div></div>
        <div class="form-section"><div class="script-section-heading"><h3>环境变量</h3><el-button :icon="Plus" @click="addEnvVar">添加变量</el-button></div><div v-for="(_,key) in form.env_vars" :key="key" class="script-env-row"><el-input :model-value="key" @change="updateEnvKey(key,$event)" placeholder="变量名称" aria-label="环境变量名称" /><el-input v-model="form.env_vars![key]" type="password" show-password placeholder="变量值" aria-label="环境变量值" /><el-button :icon="Delete" :aria-label="`移除变量 ${key}`" @click="deleteEnvVar(key)" /></div><p v-if="!Object.keys(form.env_vars || {}).length" class="management-description">暂无环境变量</p></div><button type="submit" hidden>保存脚本</button>
      </el-form>
    </FormDialog>
    <FormDialog v-model="fileDialogVisible" :draft="currentFile" :title="`编辑文件：${currentFile.file_name || '未命名文件'}`" extra-wide confirm-text="应用到脚本" @confirm="applyFile" @cancel="fileDialogVisible=false"><CodeEditor v-model="currentFile.content" :label="currentFile.file_name || '未命名文件'" :language="getFileLanguage(currentFile)" height="400px" /><p class="management-form-help">应用后仍需保存整个脚本，文件内容才会提交到服务端。</p></FormDialog>
    <FormDialog v-model="outputVisible" title="脚本执行输出" extra-wide :show-footer="false"><el-input ref="outputTextarea" v-model="output" type="textarea" :rows="18" readonly aria-label="脚本执行日志" class="script-output" /><p class="management-form-help">关闭输出不会停止服务端任务，可在部署记录中继续查看。</p><template #footer><router-link v-if="executionTask" :to="`/deployments/${executionTask}`">查看持久化部署记录</router-link><el-button @click="outputVisible=false">关闭输出</el-button></template></FormDialog>
    <FormDialog v-model="templatesVisible" title="选择脚本模板" :show-footer="false" extra-wide><div v-if="templatesError" class="management-error management-feedback" role="alert"><span>{{templatesError}}</span><el-button link @click="showTemplates">重试</el-button></div><div v-loading="templatesLoading" class="script-template-grid"><button v-for="template in templates" :key="template.id" class="script-template" @click="useTemplate(template)"><el-icon><DocumentCopy /></el-icon><span><strong>{{template.name}}</strong><small>{{template.description}}</small></span></button><p v-if="!templates.length && !templatesLoading && !templatesError" class="management-description">暂无可用模板</p></div></FormDialog>
    <ManagementDeleteDialog v-model="deleteVisible" :items="selectedScripts.map(script=>({id:script.id!,name:script.name}))" kind="脚本" endpoint="/deploy-scripts" warning="将删除脚本配置及附加文件记录，此操作无法撤销。" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick, defineAsyncComponent } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Delete, Plus, DocumentCopy } from '@element-plus/icons-vue'
import { deployApi, type DeployScript, type DeployScriptFile, type ScriptTemplate } from '@/api/deploy'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { errorText } from '@/api/workspace'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import FormDialog from '@/components/FormDialog.vue'
const CodeEditor=defineAsyncComponent(()=>import('@/components/CodeEditor.vue'))

const auth=useAuthStore(),router=useRouter()
const scripts = ref<DeployScript[]>([])
const loading = ref(false)
const listError=ref(''), actionError=ref(''), formError=ref(''), saving=ref(false), editing=ref(false), executing=ref(false), confirming=ref(false), deleteVisible=ref(false), executionTask=ref(0), templatesError=ref('')
const busy=computed(()=>saving.value || editing.value || executing.value || confirming.value || deleteVisible.value)
let disposed=false, executionController:AbortController|undefined, currentFileIndex=-1
const selectedScripts = ref<DeployScript[]>([])
const dialogVisible = ref(false)
const fileDialogVisible = ref(false)
const outputVisible = ref(false)
const output = ref('')
const outputTextarea = ref()
const isEdit = ref(false)
const templatesVisible = ref(false)
const templates = ref<ScriptTemplate[]>([])
const templatesLoading = ref(false)
const currentId = ref(0)
const currentFile = ref<DeployScriptFile>({
  file_name: '',
  file_path: '',
  file_type: 'other',
  content: ''
})

const form = reactive<DeployScript>({
  name: '',
  description: '',
  script_type: 'shell',
  main_script: '',
  work_dir: '/ingot-data/dockerun',
  env_vars: {},
  files: [],
  auto_deploy: false,
  image_name: '',
  image_tag: ''
})

// docker-manager.sh 脚本内容
const dockerManagerScript = `#!/usr/bin/env bash

###############################################################################
# Docker 容器管理脚本 V2
# 配置文件驱动的简洁架构
# 
# 架构说明:
#   1. 配置文件 (.env) - 定义环境变量和所有配置项
#   2. 执行脚本 (docker-run.sh) - 通用执行脚本(自动调用)
#   3. 管理脚本 (本脚本) - 提供容器管理功能
#
# 用法:
#   方式 1 (推荐): 使用配置文件
#     ./docker-manager.sh <配置文件.env> <命令> [选项]
#     ./docker-manager.sh myapp.env start
#     ./docker-manager.sh myapp.env stop
#     ./docker-manager.sh myapp.env logs -f
#
#   方式 2 (高级): 使用自定义执行脚本
#     ./docker-manager.sh <自定义脚本.sh> <命令> [选项]
#     ./docker-manager.sh myapp-custom-run.sh start
###############################################################################

set -e

# 颜色输出
RED='\\033[0;31m'
GREEN='\\033[0;32m'
YELLOW='\\033[1;33m'
BLUE='\\033[0;34m'
CYAN='\\033[0;36m'
NC='\\033[0m'

# 工具函数
log_info() { echo -e "\${BLUE}[INFO]\${NC} $1"; }
log_success() { echo -e "\${GREEN}[✓]\${NC} $1"; }
log_warning() { echo -e "\${YELLOW}[⚠]\${NC} $1"; }
log_error() { echo -e "\${RED}[✗]\${NC} $1"; }
log_step() { echo -e "\${CYAN}[→]\${NC} $1"; }
print_separator() { echo -e "\${BLUE}════════════════════════════════════════════════════════════════\${NC}"; }

# 检查 Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行"
        exit 1
    fi
}

# 启动容器
start_container() {
    local run_script="$1"
    if [ -n "$CONTAINER_NAME" ]; then
        if docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
            if docker ps --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
                log_warning "容器 [$CONTAINER_NAME] 已在运行中"
                show_status
                return 0
            else
                log_info "启动已存在的容器 [$CONTAINER_NAME]..."
                docker start "$CONTAINER_NAME"
                log_success "容器启动成功"
                show_status
                return 0
            fi
        fi
    fi
    log_step "执行启动脚本: $run_script"
    bash "$run_script"
    if [ $? -eq 0 ]; then
        log_success "容器启动成功"
        echo ""
        show_status
    else
        log_error "容器启动失败"
        exit 1
    fi
}

# 停止容器
stop_container() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    log_step "停止容器: $CONTAINER_NAME"
    if ! docker ps --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_warning "容器未运行"
        return 0
    fi
    docker stop "$CONTAINER_NAME"
    log_success "容器已停止"
}

# 重启容器
restart_container() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    log_step "重启容器: $CONTAINER_NAME"
    if docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        docker restart "$CONTAINER_NAME"
        log_success "容器已重启"
        show_status
    else
        log_error "容器不存在"
        return 1
    fi
}

# 删除容器
remove_container() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    log_step "删除容器: $CONTAINER_NAME"
    if ! docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_warning "容器不存在"
        return 0
    fi
    if docker ps --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_info "停止容器..."
        docker stop "$CONTAINER_NAME"
    fi
    docker rm "$CONTAINER_NAME"
    log_success "容器已删除"
}

# 删除容器和镜像
remove_all() {
    if [ -z "$CONTAINER_NAME" ] || [ -z "$IMAGE_NAME" ]; then
        log_error "未定义 CONTAINER_NAME 或 IMAGE_NAME"
        exit 1
    fi
    
    # 先删除容器
    if docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_step "删除容器: $CONTAINER_NAME"
        docker stop "$CONTAINER_NAME" 2>/dev/null || true
        docker rm "$CONTAINER_NAME"
        log_success "容器已删除"
    fi
    
    # 删除镜像
    if docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "^\${IMAGE_NAME}$"; then
        log_step "删除镜像: $IMAGE_NAME"
        docker rmi "$IMAGE_NAME"
        log_success "镜像已删除"
    else
        log_warning "镜像不存在: $IMAGE_NAME"
    fi
}

# 查看状态
show_status() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    print_separator
    echo -e "\${CYAN}容器状态: $CONTAINER_NAME\${NC}"
    print_separator
    if docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        docker ps -a --filter "name=^\${CONTAINER_NAME}$" --format "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}"
    else
        log_warning "容器不存在"
    fi
    print_separator
}

# 查看日志
show_logs() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    local follow="\${1:-false}"
    local lines="\${2:-100}"
    if ! docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_error "容器不存在"
        return 1
    fi
    if [ "$follow" = "true" ] || [ "$follow" = "-f" ]; then
        log_info "实时查看日志（Ctrl+C 退出）..."
        docker logs -f --tail "$lines" "$CONTAINER_NAME"
    else
        docker logs --tail "$lines" "$CONTAINER_NAME"
    fi
}

# 进入容器
exec_shell() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    local shell="\${1:-bash}"
    if ! docker ps --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_error "容器未运行"
        return 1
    fi
    log_info "进入容器 (shell: $shell)..."
    docker exec -it "$CONTAINER_NAME" "$shell" || docker exec -it "$CONTAINER_NAME" sh
}

# 查看容器信息
inspect_container() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    if ! docker ps -a --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_error "容器不存在"
        return 1
    fi
    docker inspect "$CONTAINER_NAME"
}

# 查看容器资源使用
show_stats() {
    [ -z "$CONTAINER_NAME" ] && { log_error "未定义 CONTAINER_NAME"; exit 1; }
    if ! docker ps --format '{{.Names}}' | grep -q "^\${CONTAINER_NAME}$"; then
        log_error "容器未运行"
        return 1
    fi
    docker stats "$CONTAINER_NAME"
}

# 显示帮助
show_help() {
    cat << EOF
Docker 容器管理脚本 V2

用法:
  \$(basename "$0") <配置文件.env> <命令> [选项]

命令:
  start                启动容器
  stop                 停止容器
  restart              重启容器
  remove, rm           删除容器
  remove-all, rmi      删除容器和镜像
  status, ps           查看容器状态
  logs [lines]         查看日志（默认最后 100 行）
  logs -f [lines]      实时查看日志
  exec [shell]         进入容器（默认 bash）
  inspect              查看容器详细信息
  stats                查看容器资源使用情况
  help                 显示帮助信息

示例:
  ./docker-manager.sh app.env start
  ./docker-manager.sh app.env logs -f
  ./docker-manager.sh app.env exec
EOF
}

# 主程序
main() {
    [ $# -lt 1 ] && { show_help; exit 1; }
    check_docker
    
    local first_arg="$1"
    local command="\${2:-help}"
    shift 2 || shift || true
    
    [ "$first_arg" = "help" ] && { show_help; exit 0; }
    
    local script_dir="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
    local run_script=""
    local config_file=""
    
    if [[ "$first_arg" == *.env ]]; then
        config_file="$first_arg"
        run_script="$script_dir/docker-run.sh"
        [ ! -f "$run_script" ] && { log_error "通用执行脚本不存在: $run_script"; exit 1; }
        [ ! -f "$config_file" ] && { log_error "配置文件不存在: $config_file"; exit 1; }
        log_info "使用配置文件: $config_file"
        log_info "使用通用执行脚本: docker-run.sh"
        source "$config_file"
        export CONFIG_FILE="$config_file"
    else
        run_script="$first_arg"
        [ ! -f "$run_script" ] && { log_error "执行脚本不存在: $run_script"; exit 1; }
        log_info "使用自定义执行脚本: $run_script"
        config_file=\$(grep -E "^CONFIG_FILE=" "$run_script" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'")
        if [ -n "$config_file" ] && [ -f "$config_file" ]; then
            log_info "加载配置文件: $config_file"
            source "$config_file"
        fi
        source "$run_script"
    fi
    
    case "$command" in
        start) start_container "$run_script" ;;
        stop) stop_container ;;
        restart) restart_container ;;
        remove|rm) remove_container ;;
        status|ps) show_status ;;
        logs) show_logs "$@" ;;
        exec) exec_shell "$@" ;;
        inspect) inspect_container ;;
        stats) show_stats ;;
        help|--help|-h) show_help ;;
        *) log_error "未知命令: $command"; echo ""; show_help; exit 1 ;;
    esac
}

main "$@"
`

// docker-run.sh 脚本内容
const dockerRunScript = `#!/usr/bin/env bash

###############################################################################
# Docker 容器通用运行脚本
# 通过环境变量 CONFIG_FILE 指定配置文件
###############################################################################

[ -z "$CONFIG_FILE" ] && { echo "错误: 未设置 CONFIG_FILE 环境变量"; exit 1; }

SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"

if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
elif [ -f "$SCRIPT_DIR/$CONFIG_FILE" ]; then
    source "$SCRIPT_DIR/$CONFIG_FILE"
else
    echo "错误: 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

# 检查必需的变量
[ -z "$CONTAINER_NAME" ] && { echo "错误: 未定义 CONTAINER_NAME"; exit 1; }
[ -z "$IMAGE_NAME" ] && { echo "错误: 未定义 IMAGE_NAME"; exit 1; }

# 创建必要的目录
if [ -n "$VOLUMES" ]; then
    while IFS= read -r volume; do
        [ -z "$volume" ] && continue
        local host_path=\$(echo "$volume" | cut -d':' -f1)
        if [[ "$host_path" == /* ]] && [ ! -e "$host_path" ]; then
            echo "创建目录: $host_path"
            mkdir -p "$host_path"
        fi
    done <<< "$VOLUMES"
fi

# 检查镜像
echo "检查镜像: $IMAGE_NAME"
if ! docker image inspect "$IMAGE_NAME" &> /dev/null; then
    echo "镜像不存在,尝试拉取..."
    docker pull "$IMAGE_NAME" || { echo "错误: 镜像拉取失败"; exit 1; }
    echo "镜像拉取成功"
else
    echo "镜像已存在"
fi

echo "开始启动容器: $CONTAINER_NAME"

# 构建 docker run 命令
CMD="docker run -d"
CMD="$CMD --name $CONTAINER_NAME"

[ -n "$CONTAINER_HOSTNAME" ] && CMD="$CMD --hostname $CONTAINER_HOSTNAME"
CMD="$CMD --restart \${RESTART_POLICY:-unless-stopped}"

# 网络配置
if [ -n "$NETWORK_NAME" ]; then
    CMD="$CMD --network $NETWORK_NAME"
    [ -n "$CONTAINER_IP" ] && CMD="$CMD --ip $CONTAINER_IP"
elif [ -n "$NETWORK_MODE" ]; then
    CMD="$CMD --network $NETWORK_MODE"
fi

# 端口映射
if [ -n "$PORTS" ]; then
    for port in $PORTS; do
        CMD="$CMD -p $port"
    done
fi

# 环境变量
if [ -n "$ENV_VARS" ]; then
    while IFS= read -r env_var; do
        [ -z "$env_var" ] && continue
        CMD="$CMD -e \\"$env_var\\""
    done <<< "$ENV_VARS"
fi

# 卷挂载
if [ -n "$VOLUMES" ]; then
    while IFS= read -r volume; do
        [ -z "$volume" ] && continue
        CMD="$CMD -v $volume"
    done <<< "$VOLUMES"
fi

if [ -n "$NAMED_VOLUMES" ]; then
    while IFS= read -r volume; do
        [ -z "$volume" ] && continue
        CMD="$CMD -v $volume"
    done <<< "$NAMED_VOLUMES"
fi

[ "$MOUNT_DOCKER_SOCKET" = "true" ] && CMD="$CMD -v /var/run/docker.sock:/var/run/docker.sock"
if [ "$MOUNT_TIMEZONE" = "true" ]; then
    CMD="$CMD -v /etc/localtime:/etc/localtime:ro"
    CMD="$CMD -v /etc/timezone:/etc/timezone:ro"
fi

# 资源限制
[ -n "$CPU_LIMIT" ] && CMD="$CMD --cpus $CPU_LIMIT"
[ -n "$CPU_SHARES" ] && CMD="$CMD --cpu-shares $CPU_SHARES"
[ -n "$MEMORY_LIMIT" ] && CMD="$CMD --memory $MEMORY_LIMIT"
[ -n "$MEMORY_RESERVATION" ] && CMD="$CMD --memory-reservation $MEMORY_RESERVATION"

# 健康检查
if [ "$HEALTH_CHECK_ENABLED" = "true" ] && [ -n "$HEALTH_CHECK_CMD" ]; then
    CMD="$CMD --health-cmd=\\"$HEALTH_CHECK_CMD\\""
    CMD="$CMD --health-interval=\${HEALTH_CHECK_INTERVAL:-30s}"
    CMD="$CMD --health-timeout=\${HEALTH_CHECK_TIMEOUT:-10s}"
    CMD="$CMD --health-retries=\${HEALTH_CHECK_RETRIES:-3}"
    [ -n "$HEALTH_CHECK_START_PERIOD" ] && CMD="$CMD --health-start-period=$HEALTH_CHECK_START_PERIOD"
fi

# 用户配置
[ -n "$RUN_AS_USER" ] && CMD="$CMD --user $RUN_AS_USER"
[ -n "$WORKDIR" ] && CMD="$CMD --workdir $WORKDIR"

# 日志配置
CMD="$CMD --log-driver \${LOG_DRIVER:-json-file}"
if [ "\${LOG_DRIVER:-json-file}" = "json-file" ]; then
    CMD="$CMD --log-opt max-size=\${LOG_MAX_SIZE:-10m}"
    CMD="$CMD --log-opt max-file=\${LOG_MAX_FILE:-3}"
fi

# 主机名映射
if [ -n "$EXTRA_HOSTS" ]; then
    while IFS= read -r host; do
        [ -z "$host" ] && continue
        CMD="$CMD --add-host $host"
    done <<< "$EXTRA_HOSTS"
fi

# DNS 配置
if [ -n "$DNS_SERVERS" ]; then
    for dns in $DNS_SERVERS; do
        CMD="$CMD --dns $dns"
    done
fi

# 安全配置
[ "$PRIVILEGED" = "true" ] && CMD="$CMD --privileged"
[ "$READ_ONLY" = "true" ] && CMD="$CMD --read-only"

# 容器标签
if [ -n "$LABELS" ]; then
    while IFS= read -r label; do
        [ -z "$label" ] && continue
        CMD="$CMD --label $label"
    done <<< "$LABELS"
fi

# 额外参数
[ -n "$EXTRA_DOCKER_ARGS" ] && CMD="$CMD $EXTRA_DOCKER_ARGS"

# 镜像名称
CMD="$CMD $IMAGE_NAME"

# 容器命令
[ -n "$CONTAINER_CMD" ] && CMD="$CMD $CONTAINER_CMD"

echo ""
echo "执行命令:"
echo "$CMD"
echo ""

eval "$CMD"

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ 容器创建成功!"
    exit 0
else
    echo ""
    echo "✗ 容器创建失败!"
    exit 1
fi
`

// app.env 配置文件模板
const appEnvTemplate = `# ==============================================================================
# Docker 容器配置文件
# 这个文件定义了容器运行所需的所有环境变量和配置项
# ==============================================================================

# ==================== 基本配置 ====================
# 容器名称（必需）
CONTAINER_NAME=""

# 镜像名称（必需）
IMAGE_NAME=""

# 容器主机名（可选）
CONTAINER_HOSTNAME=""

# 重启策略: no, always, unless-stopped, on-failure
RESTART_POLICY="always"

# ==================== 网络配置 ====================
# 网络模式: bridge, host, none
NETWORK_MODE="bridge"

# 自定义网络名称（可选，如果设置则忽略 NETWORK_MODE）
NETWORK_NAME=""

# 静态 IP 地址（可选，需要使用自定义网络）
CONTAINER_IP=""

# ==================== 端口映射 ====================
# 格式: host_port:container_port
# 多个端口用空格分隔
PORTS=""

# ==================== 环境变量 ====================
# 应用环境变量（每行一个）
ENV_VARS="
TZ=Asia/Shanghai
"

# ==================== 卷挂载 ====================
# 主机路径挂载
# 格式: host_path:container_path[:options]
VOLUMES="
/data/myapp:/app/data
"

# 命名卷
NAMED_VOLUMES=""

# ==================== 特殊挂载 ====================
# Docker Socket（用于容器管理其他容器）
MOUNT_DOCKER_SOCKET="false"

# 时区文件
MOUNT_TIMEZONE="true"

# ==================== 资源限制 ====================
# CPU 限制（核心数）
CPU_LIMIT=""

# 内存限制
MEMORY_LIMIT=""

# ==================== 健康检查 ====================
# 启用健康检查
HEALTH_CHECK_ENABLED="false"

# 健康检查命令
HEALTH_CHECK_CMD=""

# 检查间隔
HEALTH_CHECK_INTERVAL="30s"

# 超时时间
HEALTH_CHECK_TIMEOUT="10s"

# 重试次数
HEALTH_CHECK_RETRIES="3"

# 启动等待时间
HEALTH_CHECK_START_PERIOD="60s"

# ==================== 日志配置 ====================
# 日志驱动: json-file, syslog, journald
LOG_DRIVER="json-file"

# 日志选项
LOG_MAX_SIZE="10m"
LOG_MAX_FILE="3"

# ==================== 安全配置 ====================
# 特权模式（慎用）
PRIVILEGED="false"

# 只读根文件系统
READ_ONLY="false"

# ==================== 其他配置 ====================
# 容器标签（每行一个）
LABELS=""

# 容器启动命令（可选，会覆盖 Dockerfile 中的 CMD）
CONTAINER_CMD=""

# 其他 Docker run 参数
EXTRA_DOCKER_ARGS=""
`

// 主部署脚本
const exampleScript = `#!/usr/bin/env bash

set -e

echo "========================================"
echo "  Docker 容器部署脚本"
echo "========================================"

# 切换到工作目录
cd "\${WORK_DIR:-.}"

echo ""
echo "工作目录: \$(pwd)"
echo ""

# 使用 docker-manager.sh 启动容器
# 用法: bash docker-manager.sh <配置文件.env> <命令>
bash docker-manager.sh app.env start

echo ""
echo "========================================"
echo "  部署完成！"
echo "========================================"
echo ""
echo "常用命令:"
echo "  查看状态: bash docker-manager.sh app.env status"
echo "  查看日志: bash docker-manager.sh app.env logs -f"
echo "  停止容器: bash docker-manager.sh app.env stop"
echo "  重启容器: bash docker-manager.sh app.env restart"
echo "  进入容器: bash docker-manager.sh app.env exec"
echo "  删除容器: bash docker-manager.sh app.env rm"
echo "  删除容器和镜像: bash docker-manager.sh app.env rmi"
echo ""
`

const loadScripts = async () => {
  if(loading.value || disposed)return;
  loading.value=true;listError.value='';
  try {const result=await deployApi.list() as any;if(!disposed){scripts.value=result || [];selectedScripts.value=[]}}
  catch(cause){if(!disposed)listError.value=errorText(cause)}
  finally{if(!disposed)loading.value=false}
}
const scrollToBottom = async () => { await nextTick(); const textarea=outputTextarea.value?.textarea;if(textarea)textarea.scrollTop=textarea.scrollHeight }
const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

const handleCreate = () => {
  if(busy.value)return;formError.value='';
  isEdit.value = false
  form.name = ''
  form.description = ''
  form.script_type = 'shell'
  form.main_script = exampleScript
  form.work_dir = '/ingot-data/dockerun'
  form.env_vars = {}
  form.auto_deploy = false
  form.image_name = ''
  form.image_tag = ''
  
  // 默认包含通用的 Docker 部署管理脚本
  form.files = [
    {
      file_name: 'docker-manager.sh',
      file_path: './docker-manager.sh',
      file_type: 'script',
      content: dockerManagerScript,
      description: 'Docker 容器管理脚本 - 提供启动、停止、重启等管理功能'
    },
    {
      file_name: 'docker-run.sh',
      file_path: './docker-run.sh',
      file_type: 'script',
      content: dockerRunScript,
      description: 'Docker 容器通用运行脚本 - 根据配置文件启动容器'
    },
    {
      file_name: 'app.env',
      file_path: './app.env',
      file_type: 'config',
      content: appEnvTemplate,
      description: '容器配置文件 - 定义容器名称、镜像、端口、环境变量等'
    }
  ]
  
  dialogVisible.value = true
}

const handleEdit = async (script: DeployScript) => {
  if(busy.value)return;formError.value='';
  editing.value=true
  isEdit.value = true
  currentId.value = script.id!
  
  try {
    const data = await deployApi.get(script.id!) as any
    if(disposed)return
    
    
    // 重置表单
    form.name = data.name || ''
    form.description = data.description || ''
    form.script_type = data.script_type || 'shell'
    form.main_script = data.main_script || ''
    form.work_dir = data.work_dir || '/ingot-data/dockerun'
    form.env_vars = data.env_vars || {}
    form.files = data.files || []
    form.auto_deploy = data.auto_deploy || false
    form.image_name = data.image_name || ''
    form.image_tag = data.image_tag || ''

    dialogVisible.value = true
  } catch (error: any) {
    if(!disposed)actionError.value=errorText(error)
  } finally { editing.value=false }
}

const handleSave = async () => {
  if(saving.value)return;formError.value='';
  if(!form.name.trim() || !form.main_script.trim()){formError.value='请填写脚本名称和主执行脚本';return}
  saving.value=true;
  try {const payload={...form,name:form.name.trim()};if(isEdit.value)await deployApi.update(currentId.value,payload);else await deployApi.create(payload);ElMessage.success('脚本已保存');dialogVisible.value=false;await loadScripts()}
  catch(cause){formError.value=errorText(cause)}finally{saving.value=false}
}
const handleExecute = async (script:DeployScript) => {
  if(busy.value)return;confirming.value=true;
  try {await ElMessageBox.confirm(`确定执行「${script.name}」？脚本可能修改目标主机上的文件和容器。`,'执行部署脚本',{confirmButtonText:'确认执行',cancelButtonText:'取消',type:'warning'})}
  catch{return}finally{confirming.value=false}
  if(disposed)return;executing.value=true;output.value='=== 正在启动执行 ===\n';executionTask.value=0;outputVisible.value=true;executionController=new AbortController();
  try {
    const response=await fetch(`/api/deploy-scripts/${script.id}/execute`,{method:'POST',headers:{Authorization:`Bearer ${localStorage.getItem('token') || ''}`,Accept:'text/event-stream'},signal:executionController.signal});
    if(response.status===401){auth.logout();await router.replace('/login');throw new Error('登录已过期，请重新登录')}
    if(!response.ok){const failure=await response.json().catch(()=>null);throw new Error(failure?.error || `执行请求失败（${response.status}）`)}
    const reader=response.body?.getReader();if(!reader)throw new Error('服务端未返回执行日志');
    const decoder=new TextDecoder();let buffer='',terminal=false;
    const process=(message:string)=>{for(const line of message.split('\n')){if(!line.startsWith('data:'))continue;let event:any;try{event=JSON.parse(line.slice(5).trim())}catch{continue}if(event.task_id)executionTask.value=event.task_id;if(event.type==='log')output.value+=`${event.data}\n`;else if(event.type==='error'){terminal=true;output.value+=`\n=== 执行失败 ===\n${event.message || ''}\n`;ElMessage.error('脚本执行失败，请查看日志')}else if(event.type==='complete'){terminal=true;output.value+='\n=== 执行完成 ===\n';ElMessage.success('脚本执行完成')}void scrollToBottom()}};
    while(!disposed){const {done,value}=await reader.read();if(done)break;buffer+=decoder.decode(value,{stream:true});buffer=buffer.replace(/\r\n/g,'\n');const messages=buffer.split('\n\n');buffer=messages.pop() || '';for(const message of messages)process(message)}
    if(buffer.trim())process(buffer);if(!terminal && !disposed)output.value+='\n日志连接已结束，任务结果请在部署记录中核对。\n';
  }catch(cause){if(!disposed){output.value+=`\n=== 日志连接失败 ===\n${errorText(cause)}\n服务端任务可能仍在运行，请查看部署记录。`;ElMessage.error('日志连接失败，请查看部署记录')}}
  finally{executing.value=false;if(!disposed)await loadScripts()}
}
const onRemoved=(ids:number[])=>{scripts.value=scripts.value.filter(script=>!ids.includes(script.id!));selectedScripts.value=[]}
const addFile = () => {
  form.files!.push({
    file_name: '',
    file_path: './',
    file_type: 'other',
    content: '',
    description: ''
  })
}

const editFileContent = (file: DeployScriptFile) => {
  currentFileIndex=(form.files || []).indexOf(file)
  currentFile.value = {...file}
  fileDialogVisible.value = true
}

const applyFile=()=>{if(currentFileIndex>=0 && form.files?.[currentFileIndex])form.files[currentFileIndex]={...currentFile.value};fileDialogVisible.value=false}
const addEnvVar = () => { let index=1;while(`ENV_VAR_${index}` in form.env_vars!)index++;form.env_vars![`ENV_VAR_${index}`]='' }

const updateEnvKey = (oldKey: string, newKey: string) => {
  if (oldKey === newKey) return
  if(!/^[A-Za-z_][A-Za-z0-9_]*$/.test(newKey)){ElMessage.warning('变量名称须以字母或下划线开头，仅包含字母、数字及下划线');return}
  if (newKey in form.env_vars!) {
    ElMessage.warning('环境变量名称已存在')
    return
  }
  const val = form.env_vars![oldKey]
  delete form.env_vars![oldKey]
  form.env_vars![newKey] = val
}

const deleteEnvVar = (key: string) => {
  delete form.env_vars![key]
}

// 根据文件类型和文件名自动判断语言
const getFileLanguage = (file: DeployScriptFile) => {
  if (!file) return 'plaintext'
  
  // 根据文件类型判断
  if (file.file_type === 'compose') {
    return 'yaml'
  }
  if (file.file_type === 'script') {
    return 'shell'
  }
  
  // 根据文件扩展名判断
  const fileName = file.file_name?.toLowerCase() || ''
  const filePath = file.file_path?.toLowerCase() || ''
  const fullName = fileName || filePath
  
  if (fullName.endsWith('.yml') || fullName.endsWith('.yaml')) {
    return 'yaml'
  }
  if (fullName.endsWith('.sh')) {
    return 'shell'
  }
  if (fullName.endsWith('.json')) {
    return 'json'
  }
  if (fullName.endsWith('.xml')) {
    return 'xml'
  }
  if (fullName.endsWith('.properties') || fullName.endsWith('.conf') || fullName.endsWith('.config')) {
    return 'ini'
  }
  if (fullName.endsWith('.env')) {
    return 'shell'
  }
  if (fullName.endsWith('.js') || fullName.endsWith('.mjs')) {
    return 'javascript'
  }
  if (fullName.endsWith('.ts')) {
    return 'typescript'
  }
  if (fullName.endsWith('.py')) {
    return 'python'
  }
  if (fullName.endsWith('.java')) {
    return 'java'
  }
  if (fullName.endsWith('.go')) {
    return 'go'
  }
  if (fullName.endsWith('.sql')) {
    return 'sql'
  }
  if (fullName.endsWith('.md') || fullName.endsWith('.markdown')) {
    return 'markdown'
  }
  if (fullName.endsWith('.html') || fullName.endsWith('.htm')) {
    return 'html'
  }
  if (fullName.endsWith('.css')) {
    return 'css'
  }
  if (fullName.endsWith('.dockerfile') || fullName === 'dockerfile') {
    return 'dockerfile'
  }
  
  // 默认使用纯文本
  return 'plaintext'
}

// 显示模版选择
const showTemplates = async () => {
  if(templatesLoading.value)return;templatesError.value='';
  templatesVisible.value = true
  templatesLoading.value = true
  try {
    const data = await deployApi.getTemplates() as any
    templates.value = data || []
  } catch (error: any) {
    templatesError.value=errorText(error)
  } finally {
    templatesLoading.value = false
  }
}

// 使用模版
const useTemplate = (template: ScriptTemplate) => {
  form.name = template.name
  form.description = template.description
  form.script_type = 'shell'
  form.main_script = template.main_script
  form.work_dir = template.work_dir || '/ingot-data/dockerun'
  form.env_vars = {...template.env_vars}
  form.files = (template.files || []).map(f => ({
    file_name: f.file_name || '',
    file_path: f.file_path || '',
    file_type: (f.file_type as any) || 'other',
    content: f.content || '',
    description: f.description || ''
  }))
  
  templatesVisible.value = false
  dialogVisible.value = true
  isEdit.value = false
  formError.value='';form.auto_deploy=false;form.image_name='';form.image_tag=''
}

onBeforeUnmount(()=>{disposed=true;executionController?.abort()})
onMounted(() => {
  loadScripts()
})
</script>

<style scoped>
.script-section-heading { display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:14px; }.script-section-heading h3 { margin:0; }.script-files-scroll { overflow:auto; }.script-files-scroll .el-table { min-width:820px; }.script-env-row { display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr) 32px;gap:10px;margin-bottom:12px; }.script-env-row .el-button { margin:0;padding:0; }.script-template-grid { display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;min-height:100px; }.script-template { display:flex;align-items:flex-start;gap:12px;padding:16px;border:1px solid #414141;border-radius:6px;background:#242424;color:#ddd;text-align:left;cursor:pointer; }.script-template:hover { border-color:#35acd7;background:#22343c; }.script-template .el-icon { font-size:22px;color:#9cbac7; }.script-template strong { display:block;font-size:14px;font-weight:500; }.script-template small { display:block;margin-top:8px;font-size:12px;color:#999;line-height:20px; }.script-output :deep(textarea) { font:12px/1.8 ui-monospace,SFMono-Regular,Menlo,monospace; }@media(max-width:600px){.script-template-grid { grid-template-columns:minmax(0,1fr); }.script-env-row { grid-template-columns:minmax(0,1fr) 32px; }.script-env-row > .el-input:nth-child(2) { grid-column:1;grid-row:2; }.script-env-row > .el-button { grid-column:2;grid-row:1 / 3; }}
</style>
