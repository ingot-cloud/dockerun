<template>
  <div class="volumes-view">
    <div class="page-header">
      <h1>Volume list</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><FolderOpened /></el-icon>
          <span>Volumes</span>
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
            type="primary"
            :icon="Plus"
            @click="createDialogVisible = true"
          >
            Add volume
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="filteredVolumes"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        
        <el-table-column label="Name" min-width="250">
          <template #default="{ row }">
            <span class="volume-name">{{ row.Name }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Stack" width="120">
          <template #default="{ row }">
            <el-tag
              v-if="row.Labels?.['com.docker.stack.namespace']"
              size="small"
              class="stack-tag"
            >
              {{ row.Labels['com.docker.stack.namespace'] }}
            </el-tag>
            <span v-else class="stack-text">-</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="Driver" label="Driver" width="120" />
        
        <el-table-column prop="Mountpoint" label="Mount point" min-width="350">
          <template #default="{ row }">
            <span class="mount-text">{{ row.Mountpoint }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="date-text">{{ formatDate(row.CreatedAt) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Ownership" width="150">
          <template #default>
            <div class="ownership-cell">
              <el-icon class="ownership-icon"><User /></el-icon>
              <span>administrators</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="table-footer">
        <span class="total-items">{{ filteredVolumes.length }} items</span>
      </div>
    </div>
    
    <!-- 创建卷对话框 -->
    <el-dialog 
      v-model="createDialogVisible" 
      title="Create volume" 
      width="600px"
      class="create-dialog"
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. my-volume" />
        </el-form-item>
        <el-form-item label="Driver">
          <el-select v-model="form.driver" style="width: 100%">
            <el-option label="local" value="local" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleCreate">Create volume</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  FolderOpened,
  Delete,
  Plus,
  User
} from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

const volumes = ref<any[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedVolumes = ref<any[]>([])
const createDialogVisible = ref(false)

const form = reactive({
  name: '',
  driver: 'local'
})

const filteredVolumes = computed(() => {
  if (!searchText.value) return volumes.value
  const search = searchText.value.toLowerCase()
  return volumes.value.filter(v => 
    v.Name?.toLowerCase().includes(search) ||
    v.Mountpoint?.toLowerCase().includes(search)
  )
})

const loadVolumes = async () => {
  loading.value = true
  try {
    const res: any = await dockerApi.volumes.list()
    volumes.value = res.Volumes || []
  } catch (error) {
    ElMessage.error('Failed to load volumes')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedVolumes.value = selection
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

const handleCreate = async () => {
  if (!form.name) {
    ElMessage.warning('Please enter volume name')
    return
  }
  
  try {
    await dockerApi.volumes.create(form.name, form.driver)
    ElMessage.success('Volume created successfully')
    createDialogVisible.value = false
    form.name = ''
    form.driver = 'local'
    loadVolumes()
  } catch (error) {
    ElMessage.error('Failed to create volume')
  }
}

const handleBatchRemove = async () => {
  if (selectedVolumes.value.length === 0) {
    ElMessage.warning('Please select at least one volume')
    return
  }
  
  ElMessageBox.confirm(
    `Are you sure you want to remove ${selectedVolumes.value.length} volume(s)?`,
    'Confirm',
    {
      confirmButtonText: 'Remove',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    for (const volume of selectedVolumes.value) {
      try {
        await dockerApi.volumes.remove(volume.Name, false)
      } catch (error) {
        console.error('Failed to remove volume:', volume.Name)
      }
    }
    ElMessage.success('Volumes removed')
    loadVolumes()
  })
}

onMounted(() => {
  loadVolumes()
})
</script>

<style scoped>
.volumes-view {
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

.volume-name {
  color: #4a9eff;
  cursor: pointer;
}

.volume-name:hover {
  text-decoration: underline;
}

.stack-tag {
  background-color: #4a9eff;
  border: none;
  color: #fff;
}

.stack-text,
.mount-text,
.date-text {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.ownership-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: rgba(255, 255, 255, 0.85);
}

.ownership-icon {
  font-size: 14px;
  color: #8a929c;
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

.create-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.create-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #2a2d31;
}

.create-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.create-dialog :deep(.el-dialog__body) {
  background-color: #2a2d31;
  color: rgba(255, 255, 255, 0.85);
}

.create-dialog :deep(.el-dialog__footer) {
  background-color: #2a2d31;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.create-dialog :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.create-dialog :deep(.el-input__wrapper),
.create-dialog :deep(.el-select__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.create-dialog :deep(.el-input__inner) {
  color: #fff;
}

.create-dialog :deep(.el-select__placeholder) {
  color: #8a929c;
}
</style>
