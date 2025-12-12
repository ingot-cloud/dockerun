<template>
  <div class="registry-images-view">
    <div class="page-header">
      <div class="header-title">
        <el-button link @click="goBack" class="back-btn">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <h1>{{ registryName }} - Images</h1>
      </div>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Box /></el-icon>
          <span>Images</span>
        </div>
        <div class="header-right">
          <el-input
            v-model="searchKeyword"
            placeholder="Search images..."
            :prefix-icon="Search"
            class="search-input"
            clearable
            @input="handleSearch"
          />
          <el-button
            :icon="Refresh"
            @click="fetchImages"
            :loading="loading"
          >
            Refresh
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="images"
        v-loading="loading"
        class="data-table"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-content">
              <div class="tags-header">
                <strong>Tags:</strong>
                <span class="tags-count">({{ row.tags.length }} tag{{ row.tags.length !== 1 ? 's' : '' }})</span>
              </div>
              <div class="tags-list">
                <el-tag
                  v-for="tag in row.tags"
                  :key="tag"
                  class="tag-item"
                  closable
                  @close="deleteImage(row.name, tag)"
                >
                  {{ tag }}
                </el-tag>
                <el-empty v-if="row.tags.length === 0" description="No tags" :image-size="60" />
              </div>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="Image Name" min-width="400">
          <template #default="{ row }">
            <div class="image-name-cell">
              <el-icon class="image-icon"><Box /></el-icon>
              <span class="image-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="Tags" width="120" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small" class="tags-count-tag">
              {{ row.tags.length }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="Quick Actions" width="120" align="center">
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Copy Image Name" placement="top">
                <el-button
                  link
                  :icon="CopyDocument"
                  @click="copyImageName(row.name)"
                  class="action-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty 
        v-if="!loading && images.length === 0" 
        description="No images found"
        :image-size="100"
        class="empty-state"
      />
      
      <div class="table-footer" v-if="totalImages > 0">
        <span class="total-items">{{ totalImages }} items</span>
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="totalImages"
          layout="sizes, prev, pager, next"
          class="pagination"
          background
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Search, Refresh, Box, CopyDocument } from '@element-plus/icons-vue'
import { registryApi, type RegistryImage } from '@/api/registry'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const images = ref<RegistryImage[]>([])
const searchKeyword = ref('')
const registryName = ref('')
const registryId = ref(Number(route.params.id))

// 分页相关
const currentPage = ref(1)
const pageSize = ref(20)
const totalImages = ref(0)

let searchTimer: any = null

const fetchImages = async () => {
  loading.value = true
  try {
    // 调用后端分页 API
    const result = await registryApi.images.list(
      registryId.value,
      currentPage.value,
      pageSize.value,
      searchKeyword.value || undefined
    )
    
    images.value = result.items || []
    totalImages.value = result.total || 0
    console.log('Fetched images:', {
      items: images.value.length,
      total: totalImages.value,
      page: currentPage.value,
      pageSize: pageSize.value
    })
    
    // 获取 Registry 名称
    if (!registryName.value) {
      const registry = await registryApi.get(registryId.value)
      registryName.value = registry?.name || 'Unknown'
    }
  } catch (error: any) {
    console.error('Failed to fetch images:', error)
    ElMessage.error(error.response?.data?.error || 'Failed to fetch image list')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    // 搜索时重置到第一页
    currentPage.value = 1
    fetchImages()
  }, 500)
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  fetchImages()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  fetchImages()
}

const deleteImage = async (imageName: string, tag: string) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete image "${imageName}:${tag}"? This operation cannot be undone!`,
      'Confirm deletion',
      {
        type: 'warning',
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel'
      }
    )
    
    await registryApi.images.delete(registryId.value, imageName, tag)
    ElMessage.success('Image deleted successfully')
    fetchImages()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || 'Failed to delete image')
    }
  }
}

const copyImageName = async (imageName: string) => {
  try {
    const registry = await registryApi.get(registryId.value)
    const registryUrl = registry?.url?.replace(/^https?:\/\//, '') || ''
    const fullImageName = registryUrl ? `${registryUrl}/${imageName}` : imageName
    await navigator.clipboard.writeText(fullImageName)
    ElMessage.success('Copied to clipboard')
  } catch (error) {
    console.error('Failed to copy:', error)
    ElMessage.error('Failed to copy')
  }
}

const goBack = () => {
  router.push('/registry')
}

onMounted(async () => {
  // 先获取 Registry 信息
  try {
    const registry = await registryApi.get(registryId.value)
    registryName.value = registry?.name || 'Unknown'
    console.log('Registry info:', registry)
  } catch (error) {
    console.error('Failed to get registry info:', error)
  }
  
  fetchImages()
})
</script>

<style scoped>
.registry-images-view {
  padding: 20px;
  background-color: #1b1d21;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 20px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  color: #4a9eff;
  font-size: 20px;
  padding: 0;
}

.back-btn:hover {
  color: #79bbff;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 400;
  color: #fff;
}

.content-card {
  background-color: #23252a;
  border-radius: 4px;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background-color: #2a2d31;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
}

.header-icon {
  font-size: 20px;
  color: #4a9eff;
}

.header-right {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-input {
  width: 300px;
}

.search-input :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.search-input :deep(.el-input__inner) {
  color: #fff;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: #8a929c;
}

.data-table {
  width: 100%;
}

.data-table :deep(.el-table__header-wrapper) {
  background-color: #2a2d31;
}

.data-table :deep(.el-table__header th) {
  background-color: #2a2d31 !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
  padding: 12px 0;
}

.data-table :deep(.el-table__body-wrapper) {
  background-color: #23252a;
}

.data-table :deep(.el-table__body) {
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

.data-table :deep(.el-table__expand-icon) {
  color: #8a929c;
}

.data-table :deep(.el-table__expand-icon:hover) {
  color: #4a9eff;
}

.image-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-icon {
  color: #4a9eff;
  font-size: 16px;
}

.image-name {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.tags-count-tag {
  background-color: rgba(74, 158, 255, 0.2);
  border: 1px solid #4a9eff;
  color: #4a9eff;
}

.quick-actions {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #4a9eff;
  font-size: 16px;
  padding: 4px;
}

.action-btn:hover {
  color: #79bbff;
}

.expand-content {
  padding: 20px 48px;
  background-color: rgba(255, 255, 255, 0.02);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.tags-header {
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.85);
}

.tags-count {
  color: #8a929c;
  font-size: 12px;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  background-color: rgba(74, 158, 255, 0.15);
  border: 1px solid rgba(74, 158, 255, 0.3);
  color: #4a9eff;
  cursor: pointer;
}

.tag-item:hover {
  background-color: rgba(74, 158, 255, 0.25);
}

.tag-item :deep(.el-tag__close) {
  color: #4a9eff;
}

.tag-item :deep(.el-tag__close:hover) {
  background-color: #f56c6c;
  color: #fff;
}

.empty-state {
  padding: 60px 0;
  background-color: #23252a;
}

.empty-state :deep(.el-empty__description) {
  color: #8a929c;
}

.table-footer {
  padding: 12px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #23252a;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.total-items {
  font-size: 13px;
  color: #8a929c;
}

.pagination {
  --el-pagination-bg-color: transparent;
  --el-pagination-button-disabled-bg-color: transparent;
}

.pagination :deep(.el-pager li) {
  background-color: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.pagination :deep(.el-pager li:hover) {
  color: #4a9eff;
}

.pagination :deep(.el-pager li.is-active) {
  background-color: #4a9eff;
  color: #fff;
  border-color: #4a9eff;
}

.pagination :deep(.btn-prev),
.pagination :deep(.btn-next) {
  background-color: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.pagination :deep(.btn-prev:hover),
.pagination :deep(.btn-next:hover) {
  color: #4a9eff;
}

.pagination :deep(.btn-prev:disabled),
.pagination :deep(.btn-next:disabled) {
  color: rgba(255, 255, 255, 0.3);
}

.pagination :deep(.el-select .el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.pagination :deep(.el-select .el-input__inner) {
  color: rgba(255, 255, 255, 0.85);
}
</style>
