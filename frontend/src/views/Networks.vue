<template>
  <div class="networks-view">
    <div class="page-header">
      <h1>Network list</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Connection /></el-icon>
          <span>Networks</span>
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
            Add network
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="filteredNetworks"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        
        <el-table-column label="Name" >
          <template #default="{ row }">
            <span class="network-name">{{ row.Name }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Stack" width="120">
          <template #default="{ row }">
            <el-tag
              v-if="['bridge', 'host', 'none'].includes(row.Name)"
              type="info"
              size="small"
              class="system-tag"
            >
              System
            </el-tag>
            <span v-else class="stack-text">{{ row.Stack || '-' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="Driver" label="Driver" width="120" />
        
        <el-table-column label="Attachable" width="120">
          <template #default="{ row }">
            <span class="bool-text">{{ row.Attachable ? 'true' : 'false' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPAM Driver" width="120">
          <template #default="{ row }">
            <span class="text-normal">{{ row.IPAM?.Driver || 'default' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPv4 IPAM Subnet" width="150">
          <template #default="{ row }">
            <span class="text-normal">{{ row.IPAM?.Config?.[0]?.Subnet || '-' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPv4 IPAM Gateway" width="150">
          <template #default="{ row }">
            <span class="text-normal">{{ row.IPAM?.Config?.[0]?.Gateway || '-' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPv6 IPAM Subnet" width="150">
          <template #default="{ row }">
            <span class="text-normal">-</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPv6 IPAM Gateway" width="150">
          <template #default="{ row }">
            <span class="text-normal">-</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Ownership" width="150">
          <template #default="{ row }">
            <div class="ownership-cell">
              <el-icon class="ownership-icon"><User /></el-icon>
              <span>{{ ['bridge', 'host', 'none'].includes(row.Name) ? 'public' : 'administrators' }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="table-footer">
        <span class="total-items">{{ filteredNetworks.length }} items</span>
      </div>
    </div>
    
    <!-- 创建网络对话框 -->
    <el-dialog 
      v-model="createDialogVisible" 
      title="Create network" 
      width="600px"
      class="create-dialog"
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. my-network" />
        </el-form-item>
        <el-form-item label="Driver">
          <el-select v-model="form.driver" style="width: 100%">
            <el-option label="bridge" value="bridge" />
            <el-option label="overlay" value="overlay" />
            <el-option label="host" value="host" />
            <el-option label="macvlan" value="macvlan" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleCreate">Create network</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Connection,
  Delete,
  Plus,
  User,
  Download,
  Upload,
  MoreFilled
} from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

const networks = ref<any[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedNetworks = ref<any[]>([])
const createDialogVisible = ref(false)

const form = reactive({
  name: '',
  driver: 'bridge'
})

const filteredNetworks = computed(() => {
  if (!searchText.value) return networks.value
  const search = searchText.value.toLowerCase()
  return networks.value.filter(n => 
    n.Name?.toLowerCase().includes(search) ||
    n.Driver?.toLowerCase().includes(search)
  )
})

const loadNetworks = async () => {
  loading.value = true
  try {
    networks.value = await dockerApi.networks.list() as any
  } catch (error) {
    ElMessage.error('Failed to load networks')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedNetworks.value = selection
}

const handleCreate = async () => {
  if (!form.name) {
    ElMessage.warning('Please enter network name')
    return
  }
  
  try {
    await dockerApi.networks.create(form.name, form.driver)
    ElMessage.success('Network created successfully')
    createDialogVisible.value = false
    form.name = ''
    form.driver = 'bridge'
    loadNetworks()
  } catch (error) {
    ElMessage.error('Failed to create network')
  }
}

const handleBatchRemove = async () => {
  if (selectedNetworks.value.length === 0) {
    ElMessage.warning('Please select at least one network')
    return
  }
  
  ElMessageBox.confirm(
    `Are you sure you want to remove ${selectedNetworks.value.length} network(s)?`,
    'Confirm',
    {
      confirmButtonText: 'Remove',
      cancelButtonText: 'Cancel',
      type: 'warning'
    }
  ).then(async () => {
    for (const network of selectedNetworks.value) {
      if (!['bridge', 'host', 'none'].includes(network.Name)) {
        try {
          await dockerApi.networks.remove(network.Id)
        } catch (error) {
          console.error('Failed to remove network:', network.Name)
        }
      }
    }
    ElMessage.success('Networks removed')
    loadNetworks()
  })
}

onMounted(() => {
  loadNetworks()
})
</script>

<style scoped>
.networks-view {
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
  padding: 10px;
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

.network-name {
  color: #4a9eff;
  cursor: pointer;
}

.network-name:hover {
  text-decoration: underline;
}

.system-tag {
  background-color: #4a9eff;
  border: none;
  color: #fff;
}

.stack-text,
.bool-text,
.text-normal {
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
}

.create-dialog :deep(.el-dialog__title) {
  color: #fff;
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
</style>
