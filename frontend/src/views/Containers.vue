<template>
  <div class="containers-view">
    <div class="page-header">
      <h1>Container list</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Box /></el-icon>
          <span>Containers</span>
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
            type="success"
            :icon="VideoPlay"
            @click="handleBatchAction('start')"
          >
            Start
          </el-button>
          <el-button
            type="warning"
            :icon="VideoPause"
            @click="handleBatchAction('stop')"
          >
            Stop
          </el-button>
          <el-button
            type="primary"
            :icon="Refresh"
            @click="handleBatchAction('restart')"
          >
            Restart
          </el-button>
          <el-button
            :icon="More"
          >
            Kill
          </el-button>
          <el-button
            :icon="VideoPlay"
          >
            Pause
          </el-button>
          <el-button
            :icon="VideoPlay"
          >
            Resume
          </el-button>
          <el-button
            type="danger"
            :icon="Delete"
            @click="handleBatchAction('remove')"
          >
            Remove
          </el-button>
        </div>
      </div>
      
      <el-table
        :data="filteredContainers"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        
        <el-table-column label="Name" min-width="180">
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon :class="['status-icon', getStateClass(row.State)]">
                <SuccessFilled v-if="row.State === 'running'" />
                <CircleClose v-else />
              </el-icon>
              <span class="container-name">{{ row.Names?.[0]?.replace('/', '') || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="State" label="State" width="100">
          <template #default="{ row }">
            <el-tag :type="getStateType(row.State)" size="small" class="state-tag">
              {{ row.State }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="Quick Actions" width="150" align="center">
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Logs" placement="top">
                <el-button
                  link
                  :icon="Document"
                  class="action-btn"
                  @click="handleLogs(row)"
                />
              </el-tooltip>
              <el-tooltip content="Stats" placement="top">
                <el-button
                  link
                  :icon="DataAnalysis"
                  class="action-btn"
                  @click="handleStats(row)"
                />
              </el-tooltip>
              <el-tooltip content="Exec Console" placement="top">
                <el-button
                  link
                  :icon="Monitor"
                  class="action-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="Stack" label="Stack" width="100">
          <template #default="{ row }">
            <span class="stack-text">{{ row.Stack || '-' }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="Image" label="Image" min-width="280">
          <template #default="{ row }">
            <span class="image-text">{{ row.Image }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="date-text">{{ formatDate(row.Created) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="IP Address" width="130">
          <template #default="{ row }">
            <span class="ip-text">{{ getIPAddress(row) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="Published Ports" min-width="200">
          <template #default="{ row }">
            <div class="ports-cell">
              <div v-for="(port, index) in row.Ports" :key="index" class="port-item">
                <el-icon class="port-icon"><Link /></el-icon>
                <span>{{ formatPort(port) }}</span>
              </div>
              <span v-if="!row.Ports || row.Ports.length === 0" class="no-ports">-</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="Ownership" width="150">
          <template #default="{ row }">
            <div class="ownership-cell">
              <el-icon class="ownership-icon"><User /></el-icon>
              <span>administrators</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="table-footer">
        <span class="total-items">{{ filteredContainers.length }} items</span>
      </div>
    </div>
    
    <!-- 日志对话框 -->
    <el-dialog
      v-model="logsVisible"
      :title="`Container logs - ${currentContainer?.Names?.[0]?.replace('/', '')}`"
      width="80%"
      class="logs-dialog"
    >
      <el-input
        v-model="logs"
        type="textarea"
        :rows="20"
        readonly
        class="logs-textarea"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  Search,
  Box,
  SuccessFilled,
  CircleClose,
  Document,
  DataAnalysis,
  Monitor,
  Link,
  User,
  VideoPlay,
  VideoPause,
  More,
  Delete
} from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

const containers = ref<any[]>([])
const loading = ref(false)
const logsVisible = ref(false)
const logs = ref('')
const currentContainer = ref<any>(null)
const searchText = ref('')
const selectedContainers = ref<any[]>([])

const filteredContainers = computed(() => {
  if (!searchText.value) return containers.value
  const search = searchText.value.toLowerCase()
  return containers.value.filter(c => 
    c.Names?.[0]?.toLowerCase().includes(search) ||
    c.Image?.toLowerCase().includes(search)
  )
})

const loadContainers = async () => {
  loading.value = true
  try {
    containers.value = await dockerApi.containers.list(true) as any
  } catch (error) {
    ElMessage.error('Failed to load containers')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedContainers.value = selection
}

const getStateClass = (state: string) => {
  return state === 'running' ? 'running' : 'stopped'
}

const getStateType = (state: string) => {
  const map: Record<string, any> = {
    running: 'success',
    exited: 'info',
    paused: 'warning',
    dead: 'danger'
  }
  return map[state] || 'info'
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

const getIPAddress = (container: any) => {
  return container.NetworkSettings?.Networks?.bridge?.IPAddress || '-'
}

const formatPort = (port: any) => {
  if (port.PublicPort) {
    return `${port.PublicPort}:${port.PrivatePort}`
  }
  return `${port.PrivatePort}`
}

const handleLogs = async (container: any) => {
  currentContainer.value = container
  try {
    logs.value = await dockerApi.containers.logs(container.Id, '500') as any
    logsVisible.value = true
  } catch (error) {
    ElMessage.error('Failed to get logs')
  }
}

const handleStats = (container: any) => {
  ElMessage.info('Stats feature coming soon')
}

const handleBatchAction = async (action: string) => {
  if (selectedContainers.value.length === 0) {
    ElMessage.warning('Please select at least one container')
    return
  }
  
  const actionMap: Record<string, string> = {
    start: 'start',
    stop: 'stop',
    restart: 'restart',
    kill: 'kill',
    pause: 'pause',
    resume: 'unpause',
    remove: 'remove'
  }
  
  const actionName = actionMap[action]
  if (!actionName) {
    ElMessage.error('Unknown action')
    return
  }
  
  // 如果是删除操作，需要确认
  if (action === 'remove') {
    try {
      await ElMessageBox.confirm(
        `Are you sure you want to remove ${selectedContainers.value.length} container(s)?`,
        'Confirm',
        {
          confirmButtonText: 'Remove',
          cancelButtonText: 'Cancel',
          type: 'warning'
        }
      )
    } catch {
      return
    }
  }
  
  const loading = ElMessage({
    message: `Processing ${action} action...`,
    type: 'info',
    duration: 0
  })
  
  let successCount = 0
  let failCount = 0
  
  for (const container of selectedContainers.value) {
    try {
      const id = container.Id
      
      switch (action) {
        case 'start':
          await dockerApi.containers.start(id)
          break
        case 'stop':
          await dockerApi.containers.stop(id)
          break
        case 'restart':
          await dockerApi.containers.restart(id)
          break
        case 'kill':
          await dockerApi.containers.stop(id) // 使用 stop API，后端可以实现 kill
          break
        case 'pause':
          // 暂停功能需要后端支持
          ElMessage.warning('Pause feature is not implemented yet')
          break
        case 'resume':
          // 恢复功能需要后端支持
          ElMessage.warning('Resume feature is not implemented yet')
          break
        case 'remove':
          await dockerApi.containers.remove(id, false)
          break
      }
      
      successCount++
    } catch (error) {
      console.error(`Failed to ${action} container:`, container.Names?.[0])
      failCount++
    }
  }
  
  loading.close()
  
  if (successCount > 0) {
    ElMessage.success(`${successCount} container(s) ${action}ed successfully`)
    loadContainers()
  }
  
  if (failCount > 0) {
    ElMessage.warning(`${failCount} container(s) failed`)
  }
}

onMounted(() => {
  loadContainers()
})
</script>

<style scoped>
.containers-view {
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
  background-color: #23252a;
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

.data-table :deep(.el-table__header-wrapper) {
  background-color: #23252a;
}

.data-table :deep(.el-table__body-wrapper) {
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

.data-table :deep(.el-table__empty-block) {
  background-color: #23252a;
}

.data-table :deep(.el-checkbox__inner) {
  background-color: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.data-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-icon {
  font-size: 10px;
  flex-shrink: 0;
}

.status-icon.running {
  color: #2ecc71;
}

.status-icon.stopped {
  color: #95a5a6;
}

.container-name {
  color: #4a9eff;
  font-weight: 400;
  cursor: pointer;
}

.container-name:hover {
  text-decoration: underline;
}

.state-tag {
  border: none;
  font-size: 12px;
}

.quick-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.action-btn {
  font-size: 18px;
  color: #8a929c;
  padding: 4px;
}

.action-btn:hover {
  color: #4a9eff;
}

.stack-text,
.image-text,
.date-text,
.ip-text {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.ports-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #4a9eff;
  font-size: 12px;
}

.port-icon {
  font-size: 14px;
}

.no-ports {
  font-size: 12px;
  color: #8a929c;
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
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.total-items {
  font-size: 13px;
  color: #8a929c;
}

.logs-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.logs-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #2a2d31;
}

.logs-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.logs-dialog :deep(.el-dialog__body) {
  background-color: #2a2d31;
}

.logs-textarea :deep(.el-textarea__inner) {
  background-color: #1e1e1e;
  border-color: rgba(255, 255, 255, 0.15);
  color: #fff;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}
</style>
