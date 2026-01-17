<template>
  <PageContainer title="Deploy Scripts">
    <ContentCard title="Scripts" :icon="Document" :no-padding="true">
      <template #actions>
        <el-input
          v-model="searchText"
          placeholder="Search..."
          :prefix-icon="Search"
          style="width: 240px"
          clearable
        />
        <el-button
          type="danger"
          :icon="Delete"
          @click="handleBatchDelete"
          :disabled="selectedScripts.length === 0"
        >
          Remove
        </el-button>
        <el-button
          type="success"
          :icon="DocumentCopy"
          @click="showTemplates"
        >
          From Template
        </el-button>
        <el-button
          type="primary"
          :icon="Plus"
          @click="handleCreate"
        >
          Add Script
        </el-button>
      </template>
      
      <DataTable
        :data="filteredScripts"
        v-loading="loading"
        :total="filteredScripts.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left"/>
        
        <el-table-column prop="name" label="Name" width="200">
          <template #default="{ row }">
            <span class="script-name">{{ row.name }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="description" label="Description" min-width="250" />
        
        <el-table-column label="Files" width="100" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">
              {{ row.files?.length || 0 }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="Container" width="200">
          <template #default="{ row }">
            <div v-if="row.container_id" class="container-info">
              <div class="container-name">{{ row.container_name || 'Unknown' }}</div>
              <div class="container-id">{{ row.container_id.substring(0, 12) }}</div>
            </div>
            <span v-else class="no-container">-</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small" class="status-tag">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="date-text">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Quick Actions" width="200" fixed="right">
          <template #default="{ row }">
            <div class="actions-cell">
              <el-button
                type="success"
                size="small"
                :icon="VideoPlay"
                @click="handleExecute(row)"
              >
                Execute
              </el-button>
              <el-button
                type="primary"
                size="small"
                :icon="Edit"
                @click="handleEdit(row)"
              >
                Edit
              </el-button>
            </div>
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>
    
    <!-- 编辑脚本对话框 -->
    <FormDialog
      v-model="dialogVisible"
      :title="isEdit ? 'Edit Script' : 'Add Script'"
      :icon="Document"
      @confirm="handleSave"
      @cancel="dialogVisible = false"
      extra-wide
    >
      <el-form :model="form" label-width="120px" class="script-form">
        <!-- 基本信息 -->
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="Name">
            <el-input v-model="form.name" placeholder="e.g. deploy-nginx" />
          </el-form-item>
          
          <el-form-item label="Description">
            <el-input v-model="form.description" placeholder="Script description" />
          </el-form-item>
          
          <el-form-item label="Work Directory">
            <el-input v-model="form.work_dir" placeholder="/ingot-data/dockerun/my-app" />
            <div class="form-tip">Directory where files will be saved and script will be executed. If not exists, it will be created automatically.</div>
          </el-form-item>
        </div>
        
        <!-- Webhook 自动部署 -->
        <div class="form-section">
          <h3>Webhook Auto Deploy</h3>
          <el-form-item label="Enable">
            <el-switch v-model="form.auto_deploy" />
            <div class="form-tip">
              When enabled, this script will be triggered automatically when the configured image is pushed
            </div>
          </el-form-item>
          
          <template v-if="form.auto_deploy">
            <el-form-item label="Image Name">
              <el-input 
                v-model="form.image_name" 
                placeholder="e.g. myapp or */myapp (supports wildcard)"
              />
              <div class="form-tip">
                Supports wildcard *, e.g. namespace/* or */app
              </div>
            </el-form-item>
            
            <el-form-item label="Image Tag">
              <el-input 
                v-model="form.image_tag" 
                placeholder="e.g. latest or v1.* (leave empty for any tag)"
              />
              <div class="form-tip">
                Leave empty to match any tag, or use wildcard for pattern matching
              </div>
            </el-form-item>
          </template>
        </div>
        
        <!-- 主脚本 -->
        <div class="form-section">
          <h3>Main Script</h3>
          <CodeEditor
            v-model="form.main_script"
            language="shell"
            height="500px"
            :minimap="true"
          />
          <div class="form-tip">This is the main script that will be executed. It should handle directory creation, file copying, and service startup.</div>
        </div>
        
        <!-- 文件管理 -->
        <div class="form-section">
          <div class="section-header">
            <h3>Files</h3>
            <el-button 
              :icon="Plus" 
              type="primary" 
              size="small"
              @click="addFile"
            >
              Add
            </el-button>
          </div>
          
          <el-table :data="form.files" class="files-table" max-height="400">
            <el-table-column label="File Name" width="200">
              <template #default="{ row }">
                <el-input v-model="row.file_name" placeholder="e.g. docker-compose.yml" size="small" />
              </template>
            </el-table-column>
            
            <el-table-column label="File Path" width="250">
              <template #default="{ row }">
                <el-input v-model="row.file_path" placeholder="e.g. ./docker-compose.yml" size="small" />
              </template>
            </el-table-column>
            
            <el-table-column label="Type" width="150">
              <template #default="{ row }">
                <el-select v-model="row.file_type" size="small">
                  <el-option label="Docker Compose" value="compose" />
                  <el-option label="Config" value="config" />
                  <el-option label="Script" value="script" />
                  <el-option label="Other" value="other" />
                </el-select>
              </template>
            </el-table-column>
            
            <el-table-column label="Description" min-width="200">
              <template #default="{ row }">
                <el-input v-model="row.description" placeholder="File description" size="small" />
              </template>
            </el-table-column>
            
            <el-table-column label="Actions" width="120" fixed="right">
              <template #default="{ row, $index }">
                <el-button 
                  :icon="Edit" 
                  size="small"
                  @click="editFileContent(row)"
                >
                </el-button>
                <el-button 
                  :icon="Delete" 
                  type="danger" 
                  size="small"
                  @click="form.files?.splice($index, 1)"
                />
              </template>
            </el-table-column>
          </el-table>
        </div>
        
        <!-- 环境变量 -->
        <div class="form-section">
          <div class="section-header">
            <h3>Environment Variables</h3>
            <el-button 
              :icon="Plus" 
              type="primary" 
              size="small"
              @click="addEnvVar"
            >
              Add
            </el-button>
          </div>
          
          <div class="env-vars-list">
            <div 
              v-for="(_, key, index) in form.env_vars" 
              :key="index" 
              class="env-var-item"
            >
              <el-input 
                :model-value="key"
                @update:model-value="updateEnvKey(key, $event)"
                placeholder="VARIABLE_NAME" 
                class="env-key" 
              />
              <span class="env-equals">=</span>
              <el-input 
                v-model="form.env_vars![key]"
                placeholder="value" 
                class="env-value" 
              />
              <el-button 
                :icon="Delete" 
                type="danger" 
                circle 
                size="small"
                @click="deleteEnvVar(key)"
              />
            </div>
            <div v-if="Object.keys(form.env_vars || {}).length === 0" class="empty-tip">
              No environment variables defined
            </div>
          </div>
        </div>
      </el-form>
    </FormDialog>
    
    <!-- 编辑文件内容对话框 -->
    <!-- 文件内容编辑对话框 -->
    <el-dialog 
      v-model="fileDialogVisible" 
      :title="`Edit File: ${currentFile?.file_name}`" 
      width="80%"
      class="file-dialog"
    >
      <CodeEditor
        v-model="currentFile.content"
        :language="getFileLanguage(currentFile)"
        height="600px"
        :minimap="true"
      />
      <template #footer>
        <el-button @click="fileDialogVisible = false">Close</el-button>
        <el-button type="primary" @click="fileDialogVisible = false">Save</el-button>
      </template>
    </el-dialog>
    
    <!-- 执行输出对话框 -->
    <el-dialog 
      v-model="outputVisible" 
      title="Execution output" 
      width="70%"
      class="output-dialog"
    >
      <el-input
        ref="outputTextarea"
        v-model="output"
        type="textarea"
        :rows="20"
        readonly
        class="output-textarea"
      />
    </el-dialog>
    
    <!-- 模版选择对话框 -->
    <FormDialog
      v-model="templatesVisible"
      title="Choose a Template"
      :icon="DocumentCopy"
      :show-footer="false"
      extra-wide
    >
      <div v-loading="templatesLoading" class="templates-container">
        <div 
          v-for="template in templates" 
          :key="template.id" 
          class="template-card"
          @click="useTemplate(template)"
        >
          <div class="template-icon">{{ template.icon }}</div>
          <div class="template-content">
            <h3>{{ template.name }}</h3>
            <p class="template-category">{{ template.category }}</p>
            <p class="template-description">{{ template.description }}</p>
          </div>
        </div>
        <div v-if="templates.length === 0 && !templatesLoading" class="no-templates">
          No templates available
        </div>
      </div>
    </FormDialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Document,
  Delete,
  Plus,
  VideoPlay,
  Edit,
  DocumentCopy
} from '@element-plus/icons-vue'
import { deployApi, type DeployScript, type DeployScriptFile, type ScriptTemplate } from '@/api/deploy'
import PageContainer from '@/components/PageContainer.vue'
import ContentCard from '@/components/ContentCard.vue'
import DataTable from '@/components/DataTable.vue'
import FormDialog from '@/components/FormDialog.vue'
import CodeEditor from '@/components/CodeEditor.vue'

const scripts = ref<DeployScript[]>([])
const loading = ref(false)
const searchText = ref('')
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

const filteredScripts = computed(() => {
  if (!searchText.value) return scripts.value
  const search = searchText.value.toLowerCase()
  return scripts.value.filter(s => 
    s.name?.toLowerCase().includes(search) ||
    s.description?.toLowerCase().includes(search)
  )
})

const loadScripts = async () => {
  loading.value = true
  try {
    scripts.value = await deployApi.list() as any
  } catch (error) {
    ElMessage.error('Failed to load scripts')
  } finally {
    loading.value = false
  }
}

// 滚动到输出底部
const scrollToBottom = () => {
  setTimeout(() => {
    if (outputTextarea.value && outputTextarea.value.textarea) {
      const textarea = outputTextarea.value.textarea
      textarea.scrollTop = textarea.scrollHeight
    }
  }, 0)
}

const handleSelectionChange = (selection: DeployScript[]) => {
  selectedScripts.value = selection
}

const getStatusType = (status: string) => {
  const map: Record<string, any> = {
    idle: 'info',
    running: 'warning',
    stopped: 'success',
    failed: 'danger'
  }
  return map[status] || 'info'
}

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
  isEdit.value = false
  form.name = ''
  form.description = ''
  form.script_type = 'shell'
  form.main_script = exampleScript
  form.work_dir = '/ingot-data/dockerun'
  form.env_vars = {}
  
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
  isEdit.value = true
  currentId.value = script.id!
  
  try {
    const data = await deployApi.get(script.id!) as any
    
    console.log('Loaded script data:', data)
    
    // 重置表单
    form.name = data.name || ''
    form.description = data.description || ''
    form.script_type = data.script_type || 'shell'
    form.main_script = data.main_script || ''
    form.work_dir = data.work_dir || '/ingot-data/dockerun'
    form.env_vars = data.env_vars || {}
    form.files = data.files || []
    
    dialogVisible.value = true
  } catch (error: any) {
    console.error('Failed to load script:', error)
    ElMessage.error(error.response?.data?.error || error.message || 'Failed to load script')
  }
}

const handleSave = async () => {
  if (!form.name || !form.main_script) {
    ElMessage.warning('Please fill in required fields')
    return
  }
  
  try {
    if (isEdit.value) {
      await deployApi.update(currentId.value, form)
      ElMessage.success('Script updated successfully')
    } else {
      await deployApi.create(form)
      ElMessage.success('Script created successfully')
    }
    dialogVisible.value = false
    loadScripts()
  } catch (error) {
    ElMessage.error('Failed to save script')
  }
}

const handleExecute = async (script: DeployScript) => {
  ElMessageBox.confirm(
    'Are you sure you want to execute this script?',
    'Confirm',
    {
      confirmButtonText: 'Execute',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    // 清空输出并打开输出对话框
    output.value = '=== Starting execution ===\n'
    outputVisible.value = true
    
    const token = localStorage.getItem('token')
    const baseURL = (import.meta as any).env?.VITE_API_BASE_URL || '/api'
    const url = `${window.location.origin}${baseURL}/deploy-scripts/${script.id}/execute`
    
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Accept': 'text/event-stream'
        }
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const reader = response.body?.getReader()
      const decoder = new TextDecoder()
      
      if (reader) {
        let buffer = ''
        
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          
          const chunk = decoder.decode(value, { stream: true })
          buffer += chunk
          console.log('Received chunk:', chunk)
          
          // 处理完整的消息（以 \n\n 分隔）
          const messages = buffer.split('\n\n')
          buffer = messages.pop() || '' // 保留最后一个不完整的消息
          
          for (const message of messages) {
            if (message.trim() === '') continue
            
            const lines = message.split('\n')
            for (const line of lines) {
              if (line.startsWith('data: ')) {
                try {
                  const data = JSON.parse(line.substring(6))
                  console.log('Parsed SSE data:', data)
                  
                  if (data.type === 'log') {
                    output.value += data.data + '\n'
                    scrollToBottom()
                  } else if (data.type === 'error') {
                    output.value += '\n=== Execution failed ===\n'
                    output.value += data.message + '\n'
                    scrollToBottom()
                    ElMessage.error('Script execution failed')
                  } else if (data.type === 'complete') {
                    output.value += '\n=== Execution completed successfully ===\n'
                    if (data.container_id) {
                      output.value += `Container ID: ${data.container_id}\n`
                    }
                    scrollToBottom()
                    ElMessage.success('Script executed successfully')
                  }
                } catch (e) {
                  console.error('Failed to parse SSE data:', line, e)
                }
              }
            }
          }
        }
      }
      
      loadScripts()
    } catch (error: any) {
      output.value += '\n=== Error ===\n' + error.message
      ElMessage.error('Failed to execute script')
    }
  })
}

const handleBatchDelete = async () => {
  if (selectedScripts.value.length === 0) {
    ElMessage.warning('Please select at least one script')
    return
  }
  
  ElMessageBox.confirm(
    `Are you sure you want to remove ${selectedScripts.value.length} script(s)?`,
    'Confirm',
    {
      confirmButtonText: 'Remove',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    for (const script of selectedScripts.value) {
      try {
        await deployApi.delete(script.id!)
      } catch (error) {
        console.error('Failed to delete script:', script.id)
      }
    }
    ElMessage.success('Scripts removed')
    loadScripts()
  })
}

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
  currentFile.value = file
  fileDialogVisible.value = true
}

const addEnvVar = () => {
  const key = `ENV_VAR_${Object.keys(form.env_vars!).length + 1}`
  form.env_vars![key] = ''
}

const updateEnvKey = (oldKey: string, newKey: string) => {
  if (oldKey === newKey) return
  if (newKey in form.env_vars!) {
    ElMessage.warning('Variable name already exists')
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
  templatesVisible.value = true
  templatesLoading.value = true
  try {
    const data = await deployApi.getTemplates() as any
    templates.value = data || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to load templates')
    templatesVisible.value = false // 加载失败时关闭对话框
    templates.value = [] // 确保是空数组而不是 undefined
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
  form.env_vars = template.env_vars || {}
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
}

onMounted(() => {
  loadScripts()
})
</script>

<style scoped>
.deploy-scripts-view {
  width: 100%;
  height: 100%;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 400;
  color: #fff;
}

.content-card {
  background-color: #23252a;
  border-radius: 2px;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 15px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 400;
  color: #fff;
}

.header-icon {
  font-size: 18px;
  color: #8a929c;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-input {
  width: 250px;
}

.search-input :deep(.el-input__wrapper) {
  background-color: #2a2d31;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
}

.search-input :deep(.el-input__inner) {
  color: #fff;
}

.header-right :deep(.el-button) {
  height: 32px;
  padding: 0 15px;
  font-size: 13px;
}

.data-table {
  width: 100%;
  background-color: #23252a;
}

.data-table :deep(.el-table__inner-wrapper) {
  background-color: #23252a;
}

.data-table :deep(thead) {
  color: #8a929c;
}

.data-table :deep(.el-table__header th) {
  background-color: #23252a !important;
  color: #8a929c;
  font-weight: 400;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.data-table :deep(.el-table tr) {
  background-color: #23252a;
  color: #fff;
}

.data-table :deep(.el-table__row) {
  background-color: #23252a !important;
}

.data-table :deep(.el-table__row:hover > td) {
  background-color: rgba(74, 158, 255, 0.08) !important;
}

.data-table :deep(.el-table td) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background-color: transparent !important;
}

.data-table :deep(.el-checkbox__inner) {
  background-color: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.data-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.script-name {
  color: #4a9eff;
  cursor: pointer;
  font-weight: 500;
}

.script-name:hover {
  text-decoration: underline;
}

.type-tag {
  background-color: #4a9eff;
  border: none;
  color: #fff;
}

.status-tag {
  border: none;
}

.date-text {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.actions-cell {
  display: flex;
  gap: 8px;
}

.table-footer {
  padding: 12px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #23252a;
}

.total-items {
  font-size: 13px;
  color: #8a929c;
}

.script-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.script-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #2a2d31;
}

.script-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.script-dialog :deep(.el-dialog__body) {
  background-color: #2a2d31;
  color: rgba(255, 255, 255, 0.85);
}

.script-dialog :deep(.el-dialog__footer) {
  background-color: #2a2d31;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.form-section {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.form-section:last-child {
  border-bottom: none;
}

.form-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 0 20px 0 0;
}

.section-header h3 {
  margin: 0;
}

.script-form :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.script-form :deep(.el-input__wrapper),
.script-form :deep(.el-select__wrapper),
.script-form :deep(.el-textarea__inner) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.script-form :deep(.el-input__inner),
.script-form :deep(.el-textarea__inner) {
  color: #fff;
}

.script-form :deep(.el-radio__label) {
  color: rgba(255, 255, 255, 0.85);
}

.script-form :deep(.el-radio__input.is-checked .el-radio__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: #8a929c;
}

.code-input :deep(.el-textarea__inner) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  background-color: #1e1e1e !important;
  line-height: 1.6;
}

.files-table {
  background-color: transparent;
}

.files-table :deep(.el-table__inner-wrapper) {
  background-color: rgba(255, 255, 255, 0.02);
}

.files-table :deep(.el-table__header th) {
  background-color: rgba(255, 255, 255, 0.05) !important;
  color: #8a929c;
}

.files-table :deep(.el-table tr) {
  background-color: transparent;
  color: #fff;
}

.files-table :deep(.el-table__row) {
  background-color: transparent !important;
}

.files-table :deep(.el-table__row:hover > td) {
  background-color: rgba(74, 158, 255, 0.08) !important;
}

.files-table :deep(.el-table td) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background-color: transparent !important;
  color: rgba(255, 255, 255, 0.85);
}

.files-table :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.files-table :deep(.el-input__inner) {
  color: #fff;
}

.files-table :deep(.el-select__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.files-table :deep(.el-select__placeholder) {
  color: #8a929c;
}

.files-table :deep(.el-select__input) {
  color: #fff;
}

.env-vars-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.env-var-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.env-key {
  flex: 1;
}

.env-equals {
  color: #8a929c;
  font-weight: bold;
}

.env-value {
  flex: 2;
}

.empty-tip {
  padding: 20px;
  text-align: center;
  color: #8a929c;
  font-size: 14px;
}

.file-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.file-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.file-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.output-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.output-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.output-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.output-textarea :deep(.el-textarea__inner) {
  background-color: #1e1e1e;
  border-color: rgba(255, 255, 255, 0.15);
  color: #fff;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}
</style>

<style>
/* 全局样式：Select 下拉菜单 dark 模式 */
.el-select-dropdown {
  background-color: #2a2d31 !important;
  border: 1px solid rgba(255, 255, 255, 0.15) !important;
}

.el-select-dropdown__item {
  color: rgba(255, 255, 255, 0.85) !important;
}

.el-select-dropdown__item:hover {
  background-color: rgba(74, 158, 255, 0.15) !important;
}

.el-select-dropdown__item.is-selected {
  color: #4a9eff !important;
  font-weight: 500;
}

.el-popper {
  background-color: #2a2d31 !important;
  border: 1px solid rgba(255, 255, 255, 0.15) !important;
}

.el-popper__arrow::before {
  background-color: #2a2d31 !important;
  border: 1px solid rgba(255, 255, 255, 0.15) !important;
}

/* 模版对话框样式 */
.templates-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.templates-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  padding-bottom: var(--spacing-xl);
}

.template-card {
  display: flex;
  gap: 15px;
  padding: 20px;
  background-color: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-card:hover {
  background-color: rgba(74, 158, 255, 0.08);
  border-color: #4a9eff;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(74, 158, 255, 0.15);
}

.template-icon {
  font-size: 36px;
  flex-shrink: 0;
}

.template-content {
  flex: 1;
}

.template-content h3 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
}

.template-category {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: #8a929c;
}

.template-description {
  margin: 0;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.5;
}

.no-templates {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #8a929c;
  font-size: 14px;
}

.no-container {
  color: #8a929c;
  font-size: 13px;
}

.container-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.container-name {
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.container-id {
  color: #8a929c;
  font-size: 12px;
  font-family: monospace;
}
</style>
