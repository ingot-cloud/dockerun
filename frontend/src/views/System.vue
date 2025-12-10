<template>
  <div class="system-view">
    <div class="page-header">
      <h1>System</h1>
    </div>

    <!-- Docker Information Card -->
    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Monitor /></el-icon>
          <span>Docker Environment</span>
        </div>
        <div class="header-right">
          <el-button :icon="Refresh" @click="loadSystemInfo" :loading="loading">
            Refresh
          </el-button>
        </div>
      </div>
      
      <div class="card-body">
        <div v-if="loading" class="loading-state">
          <el-icon class="loading-icon"><Loading /></el-icon>
          <span>Loading system information...</span>
        </div>
        
        <div v-else-if="dockerInfo" class="info-container">
          <!-- Docker Version Section -->
          <div class="info-section">
            <div class="section-title">
              <el-icon><Box /></el-icon>
              <span>Docker</span>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <div class="item-label">Client Version</div>
                <div class="item-value">
                  <el-tag type="success" size="large">{{ dockerInfo.docker_version || 'N/A' }}</el-tag>
                </div>
              </div>
              <div class="info-item">
                <div class="item-label">Server Version</div>
                <div class="item-value">
                  <el-tag type="success" size="large">{{ dockerInfo.server_version || 'N/A' }}</el-tag>
                </div>
              </div>
              <div class="info-item">
                <div class="item-label">API Version</div>
                <div class="item-value">{{ dockerInfo.docker_api_version || 'N/A' }}</div>
              </div>
              <div class="info-item">
                <div class="item-label">Docker Compose</div>
                <div class="item-value">
                  <el-tag 
                    :type="dockerInfo.compose_version === 'Not Installed' ? 'warning' : 'success'"
                    size="large"
                  >
                    {{ dockerInfo.compose_version || 'N/A' }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>

          <!-- Server Information Section -->
          <div class="info-section">
            <div class="section-title">
              <el-icon><Platform /></el-icon>
              <span>Server Information</span>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <div class="item-label">Operating System</div>
                <div class="item-value">{{ dockerInfo.operating_system || 'N/A' }}</div>
              </div>
              <div class="info-item">
                <div class="item-label">Kernel Version</div>
                <div class="item-value">{{ dockerInfo.kernel_version || 'N/A' }}</div>
              </div>
              <div class="info-item">
                <div class="item-label">Architecture</div>
                <div class="item-value">{{ dockerInfo.server_arch || dockerInfo.arch || 'N/A' }}</div>
              </div>
              <div class="info-item">
                <div class="item-label">Host OS</div>
                <div class="item-value">{{ dockerInfo.server_os || dockerInfo.os || 'N/A' }}</div>
              </div>
            </div>
          </div>

          <!-- Resources Section -->
          <div class="info-section">
            <div class="section-title">
              <el-icon><DataAnalysis /></el-icon>
              <span>Resources</span>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <div class="item-label">Total Memory</div>
                <div class="item-value">
                  <el-tag type="info" size="large">{{ dockerInfo.total_memory || 'N/A' }}</el-tag>
                </div>
              </div>
              <div class="info-item">
                <div class="item-label">CPUs</div>
                <div class="item-value">
                  <el-tag type="info" size="large">{{ dockerInfo.cpus || 'N/A' }}</el-tag>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <div v-else class="error-state">
          <el-icon class="error-icon"><WarningFilled /></el-icon>
          <h3>Failed to Load System Information</h3>
          <p>Unable to connect to Docker daemon or retrieve system information.</p>
          <el-button type="primary" @click="loadSystemInfo">Retry</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Monitor,
  Refresh,
  Loading,
  WarningFilled,
  Box,
  Platform,
  DataAnalysis
} from '@element-plus/icons-vue'
import { systemApi, type DockerInfo } from '@/api/system'

const loading = ref(false)
const dockerInfo = ref<DockerInfo | null>(null)

const loadSystemInfo = async () => {
  loading.value = true
  try {
    const res = await systemApi.getSystemInfo()
    console.log(res)
    dockerInfo.value = res as any
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to load system information')
    dockerInfo.value = null
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadSystemInfo()
})
</script>

<style scoped>
.system-view {
  width: 100%;
  max-width: 1200px;
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

.header-right :deep(.el-button) {
  height: 32px;
  padding: 0 15px;
  font-size: 13px;
}

.card-body {
  padding: 20px;
  background-color: #23252a;
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.loading-icon {
  font-size: 48px;
  color: #4a9eff;
  margin-bottom: 16px;
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.loading-state span {
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
}

.error-icon {
  font-size: 64px;
  color: #f56c6c;
  margin-bottom: 20px;
}

.error-state h3 {
  margin: 0 0 12px 0;
  font-size: 20px;
  font-weight: 500;
  color: #fff;
}

.error-state p {
  margin: 0 0 24px 0;
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
}

.info-container {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.info-section {
  padding-bottom: 32px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.info-section:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
  margin-bottom: 20px;
}

.section-title .el-icon {
  font-size: 20px;
  color: #4a9eff;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
  background-color: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.2s;
}

.info-item:hover {
  background-color: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.1);
}

.item-label {
  color: #8a929c;
  font-size: 13px;
  font-weight: 400;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.item-value {
  color: rgba(255, 255, 255, 0.95);
  font-size: 15px;
  font-weight: 500;
  word-break: break-word;
}

.item-value :deep(.el-tag) {
  font-weight: 500;
}

.item-value :deep(.el-tag.el-tag--large) {
  height: 32px;
  padding: 0 16px;
  font-size: 14px;
}

/* Dark mode tag styles */
:deep(.el-tag--success) {
  background-color: rgba(103, 194, 58, 0.15);
  border-color: rgba(103, 194, 58, 0.3);
  color: #67c23a;
}

:deep(.el-tag--info) {
  background-color: rgba(144, 147, 153, 0.15);
  border-color: rgba(144, 147, 153, 0.3);
  color: #909399;
}

:deep(.el-tag--warning) {
  background-color: rgba(230, 162, 60, 0.15);
  border-color: rgba(230, 162, 60, 0.3);
  color: #e6a23c;
}
</style>
