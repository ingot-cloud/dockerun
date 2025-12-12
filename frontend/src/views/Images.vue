<template>
  <div class="images-view">
    <div class="page-header">
      <h1>Image list</h1>
    </div>

    <div class="pull-section" v-if="showPullSection">
      <div class="pull-card">
        <div class="pull-header">
          <div class="header-left">
            <el-icon class="header-icon"><Download /></el-icon>
            <span>Pull image</span>
          </div>
        </div>
        <div class="pull-body">
          <div class="form-row">
            <label>Registry</label>
            <el-select v-model="pullRegistry" class="form-input">
              <el-option label="Docker Hub (anonymous)" value="dockerhub" />
            </el-select>
          </div>
          <div class="form-row">
            <label>Image</label>
            <el-input 
              v-model="pullImageName" 
              placeholder="e.g. my-image:my-tag"
              class="form-input"
            />
          </div>
          <div class="form-actions">
            <el-button @click="showPullSection = false">Cancel</el-button>
            <el-button type="primary" :loading="pulling" @click="handlePull">
              Pull the image
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <div class="content-card" v-else>
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Picture /></el-icon>
          <span>Images</span>
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
            @click="handleBatchRemove"
          >
            Remove
          </el-button>
          <el-button
            :icon="Download"
            @click="openImportDialog"
          >
            Import
          </el-button>
          <el-button
            :icon="Upload"
            @click="openExportDialog"
          >
            Export
          </el-button>
          <el-button
            type="primary"
            :icon="Plus"
            @click="showPullSection = true"
          >
            Build a new image
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="filteredImages"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        
        <el-table-column label="Id" width="150">
          <template #default="{ row }">
            <span class="id-text">{{ row.Id.replace('sha256:', '').substring(0, 12) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Tags" min-width="350">
          <template #default="{ row }">
            <div class="tags-cell">
              <el-tag
                v-for="(tag, index) in row.RepoTags?.slice(0, 3)"
                :key="index"
                class="image-tag"
                size="small"
              >
                {{ tag }}
              </el-tag>
              <el-tag
                v-if="!row.RepoTags || row.RepoTags.length === 0"
                class="image-tag unused-tag"
                size="small"
              >
                &lt;none&gt;
              </el-tag>
              <span v-if="row.RepoTags && row.RepoTags.length > 3" class="more-tags">
                +{{ row.RepoTags.length - 3 }} more
              </span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="Size" width="120">
          <template #default="{ row }">
            <span class="size-text">{{ formatSize(row.Size) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="date-text">{{ formatDate(row.Created) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Quick Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-dropdown trigger="click">
              <el-button type="primary" text :icon="MoreFilled" class="more-btn" />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleRemove(row)">
                    <el-icon><Delete /></el-icon>
                    Remove
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 导入镜像对话框 -->
    <el-dialog
      v-model="showImportDialog"
      title="Import Image"
      width="700px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="dark-dialog"
    >
      <div class="import-dialog-content">
        <div v-if="!importing">
          <el-button @click="selectFileToImport" type="primary" :icon="FolderOpened">
            Select Image File (.tar)
          </el-button>
          <p v-if="selectedFile" class="selected-file">
            Selected: {{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})
          </p>
        </div>

        <div v-if="importing" class="log-container">
          <div class="log-header">
            <el-icon class="loading-icon"><Loading /></el-icon>
            <span>Importing...</span>
          </div>
          <el-scrollbar ref="importScrollbar" class="log-scrollbar">
            <pre class="log-content">{{ importLog }}</pre>
          </el-scrollbar>
        </div>
      </div>

      <template #footer>
        <el-button @click="closeImportDialog" :disabled="importing">Cancel</el-button>
        <el-button 
          type="primary" 
          @click="startImport" 
          :loading="importing"
          :disabled="!selectedFile || importing"
        >
          Import
        </el-button>
      </template>
    </el-dialog>

    <!-- 导出镜像对话框 -->
    <el-dialog
      v-model="showExportDialog"
      title="Export Image"
      width="700px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="dark-dialog"
    >
      <div class="export-dialog-content">
        <div v-if="!exporting" class="export-info">
          <p><strong>Image:</strong> {{ exportImageName }}</p>
          <p><strong>ID:</strong> {{ exportImageId }}</p>
        </div>

        <div v-if="exporting" class="progress-container">
          <div class="progress-header">
            <el-icon class="loading-icon"><Loading /></el-icon>
            <span>Exporting...</span>
          </div>
          <el-progress 
            :percentage="exportProgress" 
            :status="exportProgress === 100 ? 'success' : undefined"
          />
          <p class="progress-text">{{ exportStatus }}</p>
        </div>
      </div>

      <template #footer>
        <el-button @click="closeExportDialog" :disabled="exporting">Cancel</el-button>
        <el-button 
          type="primary" 
          @click="startExport" 
          :loading="exporting"
          :disabled="exporting"
        >
          Export
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Picture, 
  Search, 
  Delete, 
  Download, 
  Upload, 
  Plus, 
  MoreFilled,
  FolderOpened,
  Loading
} from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

const images = ref<any[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedImages = ref<any[]>([])
const showPullSection = ref(false)
const pullRegistry = ref('dockerhub')
const pullImageName = ref('')
const pulling = ref(false)

// 导入相关
const showImportDialog = ref(false)
const selectedFile = ref<File | null>(null)
const importing = ref(false)
const importLog = ref('')
const importScrollbar = ref<any>(null)

// 导出相关
const showExportDialog = ref(false)
const exporting = ref(false)
const exportProgress = ref(0)
const exportStatus = ref('')
const exportImageId = ref('')
const exportImageName = ref('')

const filteredImages = computed(() => {
  if (!searchText.value) return images.value
  
  const search = searchText.value.toLowerCase()
  return images.value.filter(image => {
    const tags = image.RepoTags?.join(' ').toLowerCase() || ''
    const id = image.Id.toLowerCase()
    return tags.includes(search) || id.includes(search)
  })
})

const loadImages = async () => {
  loading.value = true
  try {
    const response = await dockerApi.images.list()
    images.value = response || []
  } catch (error) {
    ElMessage.error('Failed to load images')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedImages.value = selection
}

const handleRemove = async (image: any) => {
  ElMessageBox.confirm(
    'Are you sure you want to remove this image?',
    'Confirm',
    {
      confirmButtonText: 'Remove',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await dockerApi.images.remove(image.Id, false)
      ElMessage.success('Image removed')
      loadImages()
    } catch (error) {
      ElMessage.error('Failed to remove image')
    }
  })
}

const formatSize = (bytes: number) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

const formatDate = (timestamp: number) => {
  const date = new Date(timestamp * 1000)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

const handlePull = async () => {
  if (!pullImageName.value) {
    ElMessage.warning('Please enter image name')
    return
  }
  
  pulling.value = true
  try {
    await dockerApi.images.pull(pullImageName.value)
    ElMessage.success('Image pulled successfully')
    showPullSection.value = false
    pullImageName.value = ''
    loadImages()
  } catch (error) {
    ElMessage.error('Failed to pull image')
  } finally {
    pulling.value = false
  }
}

const handleBatchRemove = async () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning('Please select at least one image')
    return
  }
  
  ElMessageBox.confirm(
    `Are you sure you want to remove ${selectedImages.value.length} image(s)?`,
    'Confirm',
    {
      confirmButtonText: 'Remove',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    for (const image of selectedImages.value) {
      try {
        await dockerApi.images.remove(image.Id, false)
      } catch (error) {
        console.error('Failed to remove image:', image.Id)
      }
    }
    ElMessage.success('Images removed')
    loadImages()
  })
}

// ============ 导入功能 ============

const openImportDialog = () => {
  showImportDialog.value = true
  selectedFile.value = null
  importLog.value = ''
  importing.value = false
}

const closeImportDialog = () => {
  if (importing.value) return
  showImportDialog.value = false
  selectedFile.value = null
  importLog.value = ''
}

const selectFileToImport = () => {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.tar,.tar.gz,.tgz'
  
  input.onchange = (e: any) => {
    const file = e.target?.files?.[0]
    if (file) {
      selectedFile.value = file
    }
  }
  
  input.click()
}

const scrollToBottom = async () => {
  await nextTick()
  if (importScrollbar.value) {
    const scrollElement = importScrollbar.value.$el.querySelector('.el-scrollbar__wrap')
    if (scrollElement) {
      scrollElement.scrollTop = scrollElement.scrollHeight
    }
  }
}

const startImport = async () => {
  if (!selectedFile.value) return
  
  importing.value = true
  importLog.value = ''
  
  try {
    // 创建 FormData
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    
    // 使用 fetch API 来处理 SSE
    const response = await fetch('/api/images/import', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: formData
    })
    
    if (!response.ok) {
      throw new Error('Failed to import image')
    }
    
    const reader = response.body?.getReader()
    const decoder = new TextDecoder()
    
    if (reader) {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        const chunk = decoder.decode(value, { stream: true })
        const lines = chunk.split('\n')
        
        for (const line of lines) {
          if (line.startsWith('data:')) {
            try {
              const data = JSON.parse(line.substring(5))
              if (data.message) {
                // 处理 Docker 返回的 JSON Lines
                try {
                  const jsonLines = data.message.split('\n').filter((l: string) => l.trim())
                  for (const jsonLine of jsonLines) {
                    const logData = JSON.parse(jsonLine)
                    if (logData.stream) {
                      importLog.value += logData.stream
                    } else if (logData.status) {
                      importLog.value += logData.status + '\n'
                    }
                  }
                } catch {
                  importLog.value += data.message
                }
                await scrollToBottom()
              }
            } catch (e) {
              console.error('Failed to parse SSE data:', e)
            }
          } else if (line.startsWith('event:')) {
            const event = line.substring(6).trim()
            if (event === 'complete') {
              ElMessage.success('Image imported successfully')
              setTimeout(() => {
                showImportDialog.value = false
                loadImages()
              }, 1000)
            } else if (event === 'error') {
              ElMessage.error('Failed to import image')
            }
          }
        }
      }
    }
  } catch (error: any) {
    console.error('Import error:', error)
    ElMessage.error(error.message || 'Failed to import image')
  } finally {
    importing.value = false
  }
}

// ============ 导出功能 ============

const openExportDialog = () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning('Please select an image to export')
    return
  }
  
  if (selectedImages.value.length > 1) {
    ElMessage.warning('Please select only one image to export')
    return
  }
  
  const image = selectedImages.value[0]
  exportImageId.value = image.Id
  exportImageName.value = image.RepoTags?.[0] || image.Id.replace('sha256:', '').substring(0, 12)
  exportProgress.value = 0
  exportStatus.value = ''
  exporting.value = false
  showExportDialog.value = true
}

const closeExportDialog = () => {
  if (exporting.value) return
  showExportDialog.value = false
}

const startExport = async () => {
  exporting.value = true
  exportProgress.value = 0
  exportStatus.value = 'Preparing to export...'
  
  try {
    const imageName = exportImageName.value.replace(/[/:]/g, '_')
    
    // 使用 XMLHttpRequest 来跟踪下载进度
    const token = localStorage.getItem('token')
    const xhr = new XMLHttpRequest()
    
    xhr.open('GET', `/api/images/${exportImageId.value}/export`)
    xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.responseType = 'blob'
    
    // 进度监听
    xhr.onprogress = (event) => {
      if (event.lengthComputable) {
        const percentComplete = (event.loaded / event.total) * 100
        exportProgress.value = Math.round(percentComplete)
        exportStatus.value = `Downloaded: ${formatSize(event.loaded)} / ${formatSize(event.total)}`
      } else {
        exportStatus.value = `Downloaded: ${formatSize(event.loaded)}`
        // 模拟进度
        exportProgress.value = Math.min(90, exportProgress.value + 1)
      }
    }
    
    // 完成监听
    xhr.onload = () => {
      if (xhr.status === 200) {
        exportProgress.value = 100
        exportStatus.value = 'Export completed'
        
        // 创建下载链接
        const blob = xhr.response
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `${imageName}.tar`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
        
        ElMessage.success('Image exported successfully')
        setTimeout(() => {
          showExportDialog.value = false
        }, 1000)
      } else {
        throw new Error('Failed to export image')
      }
    }
    
    // 错误监听
    xhr.onerror = () => {
      throw new Error('Network error during export')
    }
    
    // 发送请求
    xhr.send()
  } catch (error: any) {
    console.error('Export error:', error)
    ElMessage.error(error.message || 'Failed to export image')
    exporting.value = false
  } finally {
    // exporting 会在 xhr.onload 中设置为 false
  }
}

onMounted(() => {
  loadImages()
})
</script>

<style scoped>
.images-view {
  width: 100%;
  height: 100%;
}

.page-header {
  padding: 20px 0;
  margin-bottom: 20px;
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
  color: #ffffff;
  margin: 0;
}

.pull-section {
  margin-bottom: 20px;
}

.pull-card {
  background-color: #23252a;
  border-radius: 2px;
  overflow: hidden;
}

.pull-header {
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
  font-size: 14px;
  font-weight: 500;
  color: #fff;
}

.header-icon {
  font-size: 16px;
  color: #388bfd;
}

.pull-body {
  padding: 20px;
}

.form-row {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.form-row label {
  width: 120px;
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
}

.form-input {
  flex: 1;
  max-width: 500px;
}

.form-input :deep(.el-input__wrapper),
.form-input :deep(.el-select__wrapper) {
  background-color: #2a2d31;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
}

.form-input :deep(.el-input__inner) {
  color: #fff;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
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

.more-btn {
  width: 32px;
  padding: 0;
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
}

.id-text {
  font-family: monospace;
  color: #8a929c;
  font-size: 13px;
}

.tags-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.image-tag {
  background-color: rgba(56, 139, 253, 0.15);
  border: 1px solid rgba(56, 139, 253, 0.3);
  color: #388bfd;
}

.unused-tag {
  background-color: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #8a929c;
}

.more-tags {
  color: #8a929c;
  font-size: 12px;
}

.size-text, .date-text {
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
}

/* 对话框样式 */
.dark-dialog :deep(.el-dialog) {
  background-color: #1e1e1e;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.dark-dialog :deep(.el-dialog__header) {
  background-color: #252526;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 16px 20px;
}

.dark-dialog :deep(.el-dialog__title) {
  color: #fff;
  font-size: 16px;
  font-weight: 500;
}

.dark-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: rgba(255, 255, 255, 0.7);
}

.dark-dialog :deep(.el-dialog__body) {
  padding: 20px;
  background-color: #1e1e1e;
  color: #fff;
}

.dark-dialog :deep(.el-dialog__footer) {
  background-color: #252526;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  padding: 16px 20px;
}

.import-dialog-content,
.export-dialog-content {
  min-height: 200px;
  padding: 20px 0;
}

.selected-file {
  margin-top: 16px;
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
}

.log-container {
  margin-top: 16px;
}

.log-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: #388bfd;
  font-size: 14px;
  font-weight: 500;
}

.loading-icon {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.log-scrollbar {
  height: 400px;
  background-color: #0d1117;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}

.log-scrollbar :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}

.log-content {
  padding: 12px;
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #c9d1d9;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.export-info {
  padding: 16px;
  background-color: rgba(56, 139, 253, 0.1);
  border: 1px solid rgba(56, 139, 253, 0.3);
  border-radius: 4px;
}

.export-info p {
  margin: 8px 0;
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
}

.export-info strong {
  color: #fff;
  margin-right: 8px;
}

.progress-container {
  padding: 20px 0;
}

.progress-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  color: #388bfd;
  font-size: 14px;
  font-weight: 500;
}

.progress-text {
  margin-top: 12px;
  text-align: center;
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
}

.dark-dialog :deep(.el-progress__text) {
  color: #fff;
}

.dark-dialog :deep(.el-button) {
  background-color: #2a2d31;
  border-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.dark-dialog :deep(.el-button:hover) {
  background-color: #33363a;
  border-color: rgba(255, 255, 255, 0.2);
}

.dark-dialog :deep(.el-button--primary) {
  background-color: #388bfd;
  border-color: #388bfd;
}

.dark-dialog :deep(.el-button--primary:hover) {
  background-color: #4a9aff;
  border-color: #4a9aff;
}

.dark-dialog :deep(.el-button:disabled) {
  background-color: #1e1e1e;
  border-color: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.3);
}
</style>
