<template>
  <div class="system-view">
    <div class="page-header">
      <h1>System</h1>
    </div>

    <!-- Docker Status Card -->
    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Monitor /></el-icon>
          <span>Docker Status</span>
        </div>
      </div>
      
      <div class="card-body">
        <div class="info-row">
          <div class="info-label">Docker Status</div>
          <div class="info-value">
            <el-tag :type="dockerStatus ? 'success' : 'danger'" size="large">
              {{ dockerStatus ? 'Installed' : 'Not Installed' }}
            </el-tag>
          </div>
        </div>
        <div class="info-row">
          <div class="info-label">Action</div>
          <div class="info-value">
            <el-button type="primary" @click="checkStatus">
              Check Status
            </el-button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- System Info Card -->
    <div class="content-card" style="margin-top: 20px;">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><InfoFilled /></el-icon>
          <span>System Information</span>
        </div>
        <div class="header-right">
          <el-button :icon="Refresh" @click="loadSystemInfo">
            Refresh
          </el-button>
        </div>
      </div>
      
      <div class="card-body" v-if="systemInfo">
        <div class="info-grid">
          <div class="info-item">
            <div class="item-label">Operating System</div>
            <div class="item-value">{{ systemInfo.os }}</div>
          </div>
          <div class="info-item">
            <div class="item-label">Architecture</div>
            <div class="item-value">{{ systemInfo.arch }}</div>
          </div>
          <div class="info-item">
            <div class="item-label">Distribution</div>
            <div class="item-value">{{ systemInfo.distribution }}</div>
          </div>
          <div class="info-item">
            <div class="item-label">Version</div>
            <div class="item-value">{{ systemInfo.version }}</div>
          </div>
          <div class="info-item">
            <div class="item-label">Internet Connection</div>
            <div class="item-value">
              <el-tag :type="systemInfo.has_internet ? 'success' : 'warning'">
                {{ systemInfo.has_internet ? 'Connected' : 'Offline' }}
              </el-tag>
            </div>
          </div>
          <div class="info-item">
            <div class="item-label">Docker</div>
            <div class="item-value">
              <el-tag :type="systemInfo.docker_exists ? 'success' : 'info'">
                {{ systemInfo.docker_exists ? 'Installed' : 'Not Installed' }}
              </el-tag>
            </div>
          </div>
          <div class="info-item">
            <div class="item-label">Docker Compose</div>
            <div class="item-value">
              <el-tag :type="systemInfo.compose_exists ? 'success' : 'info'">
                {{ systemInfo.compose_exists ? 'Installed' : 'Not Installed' }}
              </el-tag>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Docker Installation Card -->
    <div 
      class="content-card" 
      style="margin-top: 20px;" 
      v-if="systemInfo && !systemInfo.docker_exists"
    >
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Download /></el-icon>
          <span>Docker Installation</span>
        </div>
      </div>
      
      <div class="card-body">
        <el-form :model="installForm" label-width="150px" class="install-form">
          <el-form-item label="Installation Method">
            <el-radio-group v-model="installForm.method">
              <el-radio label="online" :disabled="!systemInfo?.has_internet">
                Online Installation
              </el-radio>
              <el-radio label="offline-rpm">
                Offline (RPM)
              </el-radio>
              <el-radio label="offline-deb">
                Offline (DEB)
              </el-radio>
              <el-radio label="offline-binary">
                Offline (Binary)
              </el-radio>
            </el-radio-group>
          </el-form-item>
          
          <el-form-item 
            v-if="installForm.method === 'online'" 
            label="Linux Distribution"
          >
            <el-select v-model="installForm.distribution" style="width: 300px">
              <el-option label="Ubuntu/Debian" value="ubuntu" />
              <el-option label="CentOS/RHEL" value="centos" />
              <el-option label="Fedora" value="fedora" />
            </el-select>
          </el-form-item>
          
          <el-form-item 
            v-if="installForm.method.startsWith('offline')" 
            label="Package Path"
          >
            <el-input 
              v-model="installForm.package_path" 
              placeholder="/path/to/docker-package"
              style="width: 500px"
            />
          </el-form-item>
          
          <el-form-item>
            <el-button 
              type="primary" 
              @click="handleInstall"
              :loading="installing"
            >
              Start Installation
            </el-button>
          </el-form-item>
          
          <el-alert
            title="Note: Installing Docker requires root privileges. Please ensure the current user has sudo access."
            type="warning"
            :closable="false"
          />
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Monitor,
  InfoFilled,
  Refresh,
  Download
} from '@element-plus/icons-vue'
import { systemApi, type InstallDockerParams } from '@/api/system'

const dockerStatus = ref(false)
const systemInfo = ref<any>(null)
const installing = ref(false)

const installForm = reactive<InstallDockerParams>({
  method: 'online',
  distribution: 'ubuntu',
  package_path: ''
})

const checkStatus = async () => {
  try {
    const res: any = await systemApi.checkDockerStatus()
    dockerStatus.value = res.available
    if (res.available) {
      ElMessage.success('Docker is running')
    } else {
      ElMessage.warning('Docker is not running or not installed')
    }
  } catch (error) {
    ElMessage.error('Failed to check Docker status')
  }
}

const loadSystemInfo = async () => {
  try {
    systemInfo.value = await systemApi.getSystemInfo()
  } catch (error) {
    ElMessage.error('Failed to load system info')
  }
}

const handleInstall = async () => {
  if (installForm.method.startsWith('offline') && !installForm.package_path) {
    ElMessage.warning('Please enter package path')
    return
  }
  
  installing.value = true
  try {
    await systemApi.installDocker(installForm)
    ElMessage.success('Docker installed successfully, please restart the system')
    loadSystemInfo()
    checkStatus()
  } catch (error) {
    ElMessage.error('Docker installation failed, please check logs')
  } finally {
    installing.value = false
  }
}

onMounted(() => {
  checkStatus()
  loadSystemInfo()
})
</script>

<style scoped>
.system-view {
  width: 100%;
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

.header-right :deep(.el-button) {
  height: 32px;
  padding: 0 15px;
  font-size: 13px;
}

.card-body {
  padding: 20px;
}

.info-row {
  display: flex;
  padding: 15px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  width: 200px;
  color: #8a929c;
  font-size: 14px;
}

.info-value {
  flex: 1;
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 15px;
  background-color: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.item-label {
  color: #8a929c;
  font-size: 13px;
}

.item-value {
  color: rgba(255, 255, 255, 0.85);
  font-size: 14px;
  font-weight: 500;
}

.install-form {
  max-width: 800px;
}

.install-form :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.install-form :deep(.el-input__wrapper),
.install-form :deep(.el-select__wrapper) {
  background-color: #2a2d31;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
}

.install-form :deep(.el-input__inner) {
  color: #fff;
}

.install-form :deep(.el-radio__label) {
  color: rgba(255, 255, 255, 0.85);
}

.install-form :deep(.el-radio__input.is-checked .el-radio__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.install-form :deep(.el-radio__input.is-disabled .el-radio__inner) {
  background-color: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.install-form :deep(.el-radio__input.is-disabled + .el-radio__label) {
  color: rgba(255, 255, 255, 0.3);
}

.install-form :deep(.el-alert) {
  background-color: rgba(230, 162, 60, 0.1);
  border: 1px solid rgba(230, 162, 60, 0.3);
  margin-top: 10px;
}

.install-form :deep(.el-alert__title) {
  color: #e6a23c;
  font-size: 13px;
}
</style>
