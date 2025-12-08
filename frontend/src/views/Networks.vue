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
          <template #default>
            <span class="text-normal">-</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IPv6 IPAM Gateway" width="150">
          <template #default>
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
      width="800px"
      class="create-dialog"
      :close-on-click-modal="false"
    >
      <el-form :model="form" label-width="180px" class="network-form">
        <div class="form-section">
          <h3>Name</h3>
          <el-form-item label="Name">
            <el-input v-model="form.name" placeholder="e.g. myNetwork" />
          </el-form-item>
        </div>
        
        <div class="form-section">
          <h3>Driver configuration</h3>
          <el-form-item label="Driver">
            <el-select v-model="form.driver" style="width: 100%">
              <el-option label="bridge" value="bridge" />
              <el-option label="overlay" value="overlay" />
              <el-option label="host" value="host" />
              <el-option label="macvlan" value="macvlan" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="Driver options">
            <div class="options-list">
              <div 
                v-for="(opt, index) in form.driver_options" 
                :key="index" 
                class="option-item"
              >
                <el-input v-model="opt.key" placeholder="name" class="option-key" />
                <el-input v-model="opt.value" placeholder="value" class="option-value" />
                <el-button 
                  :icon="Delete" 
                  type="danger" 
                  circle 
                  size="small"
                  @click="form.driver_options.splice(index, 1)"
                />
              </div>
              <el-button 
                :icon="Plus" 
                type="primary" 
                text 
                @click="form.driver_options.push({ key: '', value: '' })"
              >
                Add driver option
              </el-button>
            </div>
          </el-form-item>
        </div>
        
        <div class="form-section">
          <h3>IPv4 Network configuration</h3>
          <el-form-item label="Subnet">
            <el-input v-model="form.ipv4_subnet" placeholder="e.g. 172.20.0.0/16" />
          </el-form-item>
          
          <el-form-item label="Gateway">
            <el-input v-model="form.ipv4_gateway" placeholder="e.g. 172.20.10.11" />
          </el-form-item>
          
          <el-form-item label="IP range">
            <el-input v-model="form.ipv4_ip_range" placeholder="e.g. 172.20.10.128/25" />
          </el-form-item>
          
          <el-form-item label="Excluded IPs">
            <div class="options-list">
              <div 
                v-for="(_, index) in form.ipv4_excluded_ips" 
                :key="index" 
                class="option-item"
              >
                <el-input v-model="form.ipv4_excluded_ips[index]" placeholder="e.g. 172.20.10.1" />
                <el-button 
                  :icon="Delete" 
                  type="danger" 
                  circle 
                  size="small"
                  @click="form.ipv4_excluded_ips.splice(index, 1)"
                />
              </div>
              <el-button 
                :icon="Plus" 
                type="primary" 
                text 
                @click="form.ipv4_excluded_ips.push('')"
              >
                Add excluded IP
              </el-button>
            </div>
          </el-form-item>
        </div>
        
        <div class="form-section">
          <h3>IPv6 Network configuration</h3>
          <el-form-item label="Enable IPv6">
            <el-switch v-model="form.enable_ipv6" />
          </el-form-item>
          
          <template v-if="form.enable_ipv6">
            <el-form-item label="Subnet">
              <el-input v-model="form.ipv6_subnet" placeholder="e.g. 2001:db8::/48" />
            </el-form-item>
            
            <el-form-item label="Gateway">
              <el-input v-model="form.ipv6_gateway" placeholder="e.g. 2001:db8::1" />
            </el-form-item>
            
            <el-form-item label="IP range">
              <el-input v-model="form.ipv6_ip_range" placeholder="e.g. 2001:db8::/64" />
            </el-form-item>
            
            <el-form-item label="Excluded IPs">
              <div class="options-list">
                <div 
                  v-for="(_, index) in form.ipv6_excluded_ips" 
                  :key="index" 
                  class="option-item"
                >
                  <el-input v-model="form.ipv6_excluded_ips[index]" placeholder="e.g. 2001:db8::1" />
                  <el-button 
                    :icon="Delete" 
                    type="danger" 
                    circle 
                    size="small"
                    @click="form.ipv6_excluded_ips.splice(index, 1)"
                  />
                </div>
                <el-button 
                  :icon="Plus" 
                  type="primary" 
                  text 
                  @click="form.ipv6_excluded_ips.push('')"
                >
                  Add excluded IP
                </el-button>
              </div>
            </el-form-item>
          </template>
        </div>
        
        <div class="form-section">
          <h3>Advanced configuration</h3>
          <el-form-item label="Labels">
            <div class="options-list">
              <div 
                v-for="(label, index) in form.labels" 
                :key="index" 
                class="option-item"
              >
                <el-input v-model="label.key" placeholder="name" class="option-key" />
                <el-input v-model="label.value" placeholder="value" class="option-value" />
                <el-button 
                  :icon="Delete" 
                  type="danger" 
                  circle 
                  size="small"
                  @click="form.labels.splice(index, 1)"
                />
              </div>
              <el-button 
                :icon="Plus" 
                type="primary" 
                text 
                @click="form.labels.push({ key: '', value: '' })"
              >
                Add label
              </el-button>
            </div>
          </el-form-item>
          
          <el-form-item label="Isolated network">
            <el-switch v-model="form.internal" />
          </el-form-item>
          
          <el-form-item label="Enable manual container attachment">
            <el-switch v-model="form.attachable" />
          </el-form-item>
        </div>
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
  User
} from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

const networks = ref<any[]>([])
const loading = ref(false)
const searchText = ref('')
const selectedNetworks = ref<any[]>([])
const createDialogVisible = ref(false)

const form = reactive({
  name: '',
  driver: 'bridge',
  driver_options: [] as Array<{ key: string; value: string }>,
  ipv4_subnet: '',
  ipv4_gateway: '',
  ipv4_ip_range: '',
  ipv4_excluded_ips: [] as string[],
  enable_ipv6: false,
  ipv6_subnet: '',
  ipv6_gateway: '',
  ipv6_ip_range: '',
  ipv6_excluded_ips: [] as string[],
  labels: [] as Array<{ key: string; value: string }>,
  internal: false,
  attachable: false
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
    // 构建请求参数
    const params: any = {
      name: form.name,
      driver: form.driver,
      internal: form.internal,
      attachable: form.attachable,
      ipam: {
        driver: 'default',
        config: [] as any[]
      }
    }
    
    // Driver options
    if (form.driver_options.length > 0) {
      params.options = {}
      form.driver_options.forEach(opt => {
        if (opt.key && opt.value) {
          params.options[opt.key] = opt.value
        }
      })
    }
    
    // IPv4 configuration
    const ipv4Config: any = {}
    if (form.ipv4_subnet) ipv4Config.subnet = form.ipv4_subnet
    if (form.ipv4_gateway) ipv4Config.gateway = form.ipv4_gateway
    if (form.ipv4_ip_range) ipv4Config.ip_range = form.ipv4_ip_range
    if (form.ipv4_excluded_ips.length > 0) {
      ipv4Config.aux_addresses = {}
      form.ipv4_excluded_ips.forEach((ip, index) => {
        if (ip) ipv4Config.aux_addresses[`excluded_${index}`] = ip
      })
    }
    if (Object.keys(ipv4Config).length > 0) {
      params.ipam.config.push(ipv4Config)
    }
    
    // IPv6 configuration
    if (form.enable_ipv6) {
      params.enable_ipv6 = true
      const ipv6Config: any = {}
      if (form.ipv6_subnet) ipv6Config.subnet = form.ipv6_subnet
      if (form.ipv6_gateway) ipv6Config.gateway = form.ipv6_gateway
      if (form.ipv6_ip_range) ipv6Config.ip_range = form.ipv6_ip_range
      if (form.ipv6_excluded_ips.length > 0) {
        ipv6Config.aux_addresses = {}
        form.ipv6_excluded_ips.forEach((ip, index) => {
          if (ip) ipv6Config.aux_addresses[`excluded_${index}`] = ip
        })
      }
      if (Object.keys(ipv6Config).length > 0) {
        params.ipam.config.push(ipv6Config)
      }
    }
    
    // Labels
    if (form.labels.length > 0) {
      params.labels = {}
      form.labels.forEach(label => {
        if (label.key && label.value) {
          params.labels[label.key] = label.value
        }
      })
    }
    
    await dockerApi.networks.create(params)
    ElMessage.success('Network created successfully')
    createDialogVisible.value = false
    resetForm()
    loadNetworks()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || 'Failed to create network')
  }
}

const resetForm = () => {
  form.name = ''
  form.driver = 'bridge'
  form.driver_options = []
  form.ipv4_subnet = ''
  form.ipv4_gateway = ''
  form.ipv4_ip_range = ''
  form.ipv4_excluded_ips = []
  form.enable_ipv6 = false
  form.ipv6_subnet = ''
  form.ipv6_gateway = ''
  form.ipv6_ip_range = ''
  form.ipv6_excluded_ips = []
  form.labels = []
  form.internal = false
  form.attachable = false
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

.network-form {
  max-height: 600px;
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

.options-list {
  width: 100%;
}

.option-item {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}

.option-key {
  flex: 1;
}

.option-value {
  flex: 2;
}

.network-form :deep(.el-switch.is-checked .el-switch__core) {
  background-color: #4a9eff;
}

.network-form :deep(.el-button.is-text) {
  color: #4a9eff;
}

.network-form :deep(.el-button.is-text:hover) {
  color: #79bbff;
}
</style>
