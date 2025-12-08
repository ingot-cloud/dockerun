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
            @click="handleImport"
          >
            Import
          </el-button>
          <el-button
            :icon="Upload"
            @click="handleExport"
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
                type="danger"
                size="small"
                class="unused-tag"
              >
                Unused
              </el-tag>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="Size" width="120" align="right">
          <template #default="{ row }">
            <span class="size-text">{{ formatSize(row.Size) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="date-text">{{ formatDate(row.Created) }}</span>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="table-footer">
        <span class="total-items">{{ filteredImages.length }} items</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Picture,
  Delete,
  Download,
  Upload,
  Plus
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

const filteredImages = computed(() => {
  if (!searchText.value) return images.value
  const search = searchText.value.toLowerCase()
  return images.value.filter(img => 
    img.RepoTags?.some((tag: string) => tag.toLowerCase().includes(search)) ||
    img.Id.toLowerCase().includes(search)
  )
})

const loadImages = async () => {
  loading.value = true
  try {
    images.value = await dockerApi.images.list() as any
  } catch (error) {
    ElMessage.error('Failed to load images')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedImages.value = selection
}

const formatSize = (size: number) => {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  if (size < 1024 * 1024 * 1024) return (size / (1024 * 1024)).toFixed(1) + ' MB'
  return (size / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
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

const handleImport = () => {
  ElMessageBox.prompt(
    'Please enter the path to the image tar file to import',
    'Import Image',
    {
      confirmButtonText: 'Import',
      cancelButtonText: 'Cancel',
      inputPlaceholder: '/path/to/image.tar'
    }
  ).then(async ({ value }) => {
    if (!value) {
      ElMessage.warning('Please enter a file path')
      return
    }
    
    const loading = ElMessage({
      message: 'Importing image...',
      type: 'info',
      duration: 0
    })
    
    try {
      // 注意：实际实现需要后端支持文件上传或路径导入
      // 这里暂时使用 API 调用的方式
      await dockerApi.images.load(value)
      loading.close()
      ElMessage.success('Image imported successfully')
      loadImages()
    } catch (error: any) {
      loading.close()
      ElMessage.error(error.response?.data?.message || 'Failed to import image')
    }
  }).catch(() => {
    // 用户取消
  })
}

const handleExport = () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning('Please select at least one image to export')
    return
  }
  
  if (selectedImages.value.length > 1) {
    ElMessage.warning('Please select only one image to export')
    return
  }
  
  const image = selectedImages.value[0]
  const imageName = image.RepoTags?.[0] || image.Id.replace('sha256:', '').substring(0, 12)
  
  ElMessageBox.prompt(
    `Please enter the output path for the exported image`,
    'Export Image',
    {
      confirmButtonText: 'Export',
      cancelButtonText: 'Cancel',
      inputPlaceholder: `/tmp/${imageName}.tar`,
      inputValue: `/tmp/${imageName}.tar`
    }
  ).then(async ({ value }) => {
    if (!value) {
      ElMessage.warning('Please enter a file path')
      return
    }
    
    const loading = ElMessage({
      message: 'Exporting image...',
      type: 'info',
      duration: 0
    })
    
    try {
      // 注意：实际实现需要后端支持文件保存
      await dockerApi.images.save(image.Id, value)
      loading.close()
      ElMessage.success(`Image exported to ${value}`)
    } catch (error: any) {
      loading.close()
      ElMessage.error(error.response?.data?.message || 'Failed to export image')
    }
  }).catch(() => {
    // 用户取消
  })
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
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 400;
  color: #fff;
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
  font-size: 16px;
  font-weight: 400;
  color: #fff;
}

.header-icon {
  font-size: 18px;
  color: #8a929c;
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
  color: rgba(255, 255, 255, 0.85);
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

.id-text {
  color: #4a9eff;
  font-family: monospace;
  cursor: pointer;
}

.id-text:hover {
  text-decoration: underline;
}

.tags-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.image-tag {
  background-color: #4a9eff;
  border: none;
  color: #fff;
  font-size: 12px;
}

.unused-tag {
  background-color: #e67e22;
  border: none;
  color: #fff;
  font-size: 12px;
}

.size-text,
.date-text {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
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
</style>
