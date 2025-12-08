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
        
        <el-table-column prop="script_type" label="Type" width="150">
          <template #default="{ row }">
            <el-tag class="type-tag" size="small">
              {{ row.script_type }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small" class="status-tag">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="Container" width="150">
          <template #default="{ row }">
            <span class="container-id">
              {{ row.container_id ? row.container_id.substring(0, 12) : '-' }}
            </span>
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
      width="70%"
      :close-on-click-modal="false"
      class="script-dialog"
    >
      <el-form :model="form" label-width="120px" style="margin-top: 10px;">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. deploy-nginx" />
        </el-form-item>
        
        <el-form-item label="Description">
          <el-input v-model="form.description" placeholder="Script description" />
        </el-form-item>
        
        <el-form-item label="Type">
          <el-radio-group v-model="form.script_type">
            <el-radio label="shell">Shell Script</el-radio>
            <el-radio label="docker-compose">Docker Compose</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item label="Script">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="15"
            placeholder="Enter your script content..."
            class="code-input"
          />
        </el-form-item>
        
        <el-form-item 
          v-if="form.script_type === 'docker-compose'" 
          label="Compose File"
        >
          <el-input
            v-model="form.compose_file"
            type="textarea"
            :rows="15"
            placeholder="Enter docker-compose.yml content..."
            class="code-input"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleSave">Save</el-button>
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
  User
} from '@element-plus/icons-vue'
import { deployApi, type DeployScript } from '@/api/deploy'

const scripts = ref<DeployScript[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedScripts = ref<DeployScript[]>([])
const dialogVisible = ref(false)
const outputVisible = ref(false)
const output = ref('')
const isEdit = ref(false)
const currentId = ref(0)

const form = reactive<DeployScript>({
  name: '',
  description: '',
  script_type: 'shell',
  content: '',
  compose_file: ''
})

const exampleScript = `#!/usr/bin/env bash

VERSION=latest
IMAGE_NAME=nginx
SERVICE_NAME=my-nginx

# Stop and remove old container
docker ps -q --filter name="\${SERVICE_NAME}" | xargs -r docker rm -f

# Pull latest image
docker pull \${IMAGE_NAME}:\${VERSION}

# Start new container
docker run -d --name \${SERVICE_NAME} \\
  --restart always \\
  -p 8080:80 \\
  \${IMAGE_NAME}:\${VERSION}
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
  form.content = exampleScript
  form.compose_file = ''
  dialogVisible.value = true
}

const handleEdit = async (script: DeployScript) => {
  isEdit.value = true
  currentId.value = script.id!
  
  try {
    const data = await deployApi.get(script.id!) as any
    form.name = data.name
    form.description = data.description
    form.script_type = data.script_type
    form.content = data.content
    form.compose_file = data.compose_file || ''
    dialogVisible.value = true
  } catch (error) {
    ElMessage.error('Failed to load script')
  }
}

const handleSave = async () => {
  if (!form.name || !form.content) {
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
    const loading = ElMessage({
      message: 'Executing script...',
      type: 'info',
      duration: 0
    })
    
    try {
      const res: any = await deployApi.execute(script.id!)
      loading.close()
      output.value = res.output || 'Execution completed'
      outputVisible.value = true
      loadScripts()
    } catch (error: any) {
      loading.close()
      output.value = error.response?.data?.output || 'Execution failed'
      outputVisible.value = true
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

.container-id {
  color: #4a9eff;
  font-family: monospace;
  font-size: 12px;
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
}

.script-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.script-dialog :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.script-dialog :deep(.el-input__wrapper),
.script-dialog :deep(.el-select__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.script-dialog :deep(.el-input__inner),
.script-dialog :deep(.el-textarea__inner) {
  color: #fff;
}

.script-dialog :deep(.el-radio__label) {
  color: rgba(255, 255, 255, 0.85);
}

.script-dialog :deep(.el-radio__input.is-checked .el-radio__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.code-input :deep(.el-textarea__inner) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  background-color: #1e1e1e;
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
