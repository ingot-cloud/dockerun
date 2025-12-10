<template>
  <div class="deploy-scripts-view">
    <div class="page-header">
      <h1>Deploy Scripts</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Document /></el-icon>
          <span>Scripts</span>
        </div>
        <div class="header-right">
          <el-input
            v-model="searchText"
            placeholder="Search..."
            :prefix-icon="Search"
            class="search-input"
            clearable
          />
          <el-button
            type="danger"
            :icon="Delete"
            @click="handleBatchDelete"
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
            Add script
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="filteredScripts"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        
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
        
        <el-table-column label="Actions" width="200" fixed="right">
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
      </el-table>
      
      <div class="table-footer">
        <span class="total-items">{{ filteredScripts.length }} items</span>
      </div>
    </div>
    
    <!-- 编辑脚本对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="isEdit ? 'Edit script' : 'Add script'" 
      width="90%"
      :close-on-click-modal="false"
      class="script-dialog"
      top="5vh"
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
        
        <!-- 主脚本 -->
        <div class="form-section">
          <h3>Main Script</h3>
          <el-input
            v-model="form.main_script"
            type="textarea"
            :rows="12"
            placeholder="Enter your main execution script..."
            class="code-input"
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
              Add File
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
              Add Variable
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
      
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleSave">Save</el-button>
      </template>
    </el-dialog>
    
    <!-- 编辑文件内容对话框 -->
    <el-dialog 
      v-model="fileDialogVisible" 
      :title="`Edit File: ${currentFile?.file_name}`" 
      width="70%"
      class="file-dialog"
    >
      <el-input
        v-model="currentFile.content"
        type="textarea"
        :rows="20"
        placeholder="Enter file content..."
        class="code-input"
      />
      <template #footer>
        <el-button @click="fileDialogVisible = false">Close</el-button>
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
        v-model="output"
        type="textarea"
        :rows="20"
        readonly
        class="output-textarea"
      />
    </el-dialog>
    
    <!-- 模版选择对话框 -->
    <el-dialog 
      v-model="templatesVisible" 
      title="Choose a Template" 
      width="80%"
      class="templates-dialog"
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
    </el-dialog>
  </div>
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

const scripts = ref<DeployScript[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedScripts = ref<DeployScript[]>([])
const dialogVisible = ref(false)
const fileDialogVisible = ref(false)
const outputVisible = ref(false)
const output = ref('')
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
  files: []
})

const exampleScript = `#!/usr/bin/env bash

set -e

# 设置工作目录
WORK_DIR="\${WORK_DIR:-/opt/myapp}"
cd "$WORK_DIR"

# 创建必要的目录
mkdir -p data logs config

# 复制配置文件
# (配置文件应该已经在工作目录中)

# 启动服务
echo "Starting services..."
docker-compose up -d

echo "Deployment completed!"
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
  form.files = []
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
    const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'
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
                  } else if (data.type === 'error') {
                    output.value += '\n=== Execution failed ===\n'
                    output.value += data.message + '\n'
                    ElMessage.error('Script execution failed')
                  } else if (data.type === 'complete') {
                    output.value += '\n=== Execution completed successfully ===\n'
                    if (data.container_id) {
                      output.value += `Container ID: ${data.container_id}\n`
                    }
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

// 显示模版选择
const showTemplates = async () => {
  templatesVisible.value = true
  templatesLoading.value = true
  try {
    const res = await deployApi.getTemplates()
    templates.value = res.data as any
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to load templates')
    templatesVisible.value = false // 加载失败时关闭对话框
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

.script-form {
  max-height: 70vh;
  overflow-y: auto;
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
  min-height: 300px;
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
