<template>
  <div class="containers-view">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>容器列表</span>
          <el-button type="primary" @click="loadContainers">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>
      
      <el-table :data="containers" v-loading="loading" border>
        <el-table-column prop="Names" label="名称" width="200">
          <template #default="{ row }">
            {{ row.Names?.[0]?.replace('/', '') || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="Image" label="镜像" />
        <el-table-column prop="State" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStateType(row.State)">
              {{ row.State }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="Status" label="运行状态" width="150" />
        <el-table-column label="端口" width="180">
          <template #default="{ row }">
            <div v-for="(port, index) in row.Ports" :key="index">
              {{ formatPort(port) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="350" fixed="right">
          <template #default="{ row }">
            <el-button 
              v-if="row.State !== 'running'" 
              type="success" 
              size="small" 
              @click="handleStart(row)"
            >
              启动
            </el-button>
            <el-button 
              v-if="row.State === 'running'" 
              type="warning" 
              size="small" 
              @click="handleStop(row)"
            >
              停止
            </el-button>
            <el-button 
              type="primary" 
              size="small" 
              @click="handleRestart(row)"
            >
              重启
            </el-button>
            <el-button 
              type="info" 
              size="small" 
              @click="handleLogs(row)"
            >
              日志
            </el-button>
            <el-button 
              type="danger" 
              size="small" 
              @click="handleRemove(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 日志对话框 -->
    <el-dialog v-model="logsVisible" :title="`容器日志 - ${currentContainer?.Names?.[0]}`" width="80%">
      <el-input
        v-model="logs"
        type="textarea"
        :rows="20"
        readonly
        style="font-family: monospace;"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { dockerApi } from '@/api/docker'

const containers = ref<any[]>([])
const loading = ref(false)
const logsVisible = ref(false)
const logs = ref('')
const currentContainer = ref<any>(null)

const loadContainers = async () => {
  loading.value = true
  try {
    containers.value = await dockerApi.containers.list(true) as any
  } catch (error) {
    ElMessage.error('加载容器列表失败')
  } finally {
    loading.value = false
  }
}

const getStateType = (state: string) => {
  const map: Record<string, any> = {
    running: 'success',
    exited: 'info',
    paused: 'warning',
    dead: 'danger'
  }
  return map[state] || ''
}

const formatPort = (port: any) => {
  if (port.PublicPort) {
    return `${port.IP || '0.0.0.0'}:${port.PublicPort}->${port.PrivatePort}/${port.Type}`
  }
  return `${port.PrivatePort}/${port.Type}`
}

const handleStart = async (container: any) => {
  try {
    await dockerApi.containers.start(container.Id)
    ElMessage.success('容器启动成功')
    loadContainers()
  } catch (error) {
    ElMessage.error('容器启动失败')
  }
}

const handleStop = async (container: any) => {
  try {
    await dockerApi.containers.stop(container.Id)
    ElMessage.success('容器停止成功')
    loadContainers()
  } catch (error) {
    ElMessage.error('容器停止失败')
  }
}

const handleRestart = async (container: any) => {
  try {
    await dockerApi.containers.restart(container.Id)
    ElMessage.success('容器重启成功')
    loadContainers()
  } catch (error) {
    ElMessage.error('容器重启失败')
  }
}

const handleLogs = async (container: any) => {
  currentContainer.value = container
  try {
    logs.value = await dockerApi.containers.logs(container.Id, '500') as any
    logsVisible.value = true
  } catch (error) {
    ElMessage.error('获取日志失败')
  }
}

const handleRemove = async (container: any) => {
  ElMessageBox.confirm('确定要删除这个容器吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await dockerApi.containers.remove(container.Id, true)
      ElMessage.success('容器删除成功')
      loadContainers()
    } catch (error) {
      ElMessage.error('容器删除失败')
    }
  })
}

onMounted(() => {
  loadContainers()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

