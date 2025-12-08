<template>
  <div class="system-view">
    <el-row :gutter="20">
      <el-col :span="24">
        <el-card>
          <template #header>
            <span>Docker状态</span>
          </template>
          
          <el-descriptions :column="2" border>
            <el-descriptions-item label="Docker状态">
              <el-tag :type="dockerStatus ? 'success' : 'danger'">
                {{ dockerStatus ? '已安装' : '未安装' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="操作">
              <el-button 
                type="primary" 
                size="small" 
                @click="checkStatus"
              >
                检查状态
              </el-button>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      
      <el-col :span="24" style="margin-top: 20px;">
        <el-card>
          <template #header>
            <span>系统信息</span>
          </template>
          
          <el-descriptions :column="2" border v-if="systemInfo">
            <el-descriptions-item label="操作系统">
              {{ systemInfo.os }}
            </el-descriptions-item>
            <el-descriptions-item label="架构">
              {{ systemInfo.arch }}
            </el-descriptions-item>
            <el-descriptions-item label="发行版">
              {{ systemInfo.distribution }}
            </el-descriptions-item>
            <el-descriptions-item label="版本">
              {{ systemInfo.version }}
            </el-descriptions-item>
            <el-descriptions-item label="网络连接">
              <el-tag :type="systemInfo.has_internet ? 'success' : 'warning'">
                {{ systemInfo.has_internet ? '可访问互联网' : '无网络连接' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Docker">
              <el-tag :type="systemInfo.docker_exists ? 'success' : 'info'">
                {{ systemInfo.docker_exists ? '已安装' : '未安装' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Docker Compose">
              <el-tag :type="systemInfo.compose_exists ? 'success' : 'info'">
                {{ systemInfo.compose_exists ? '已安装' : '未安装' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
          
          <div style="margin-top: 20px;">
            <el-button type="primary" @click="loadSystemInfo">
              刷新信息
            </el-button>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="24" style="margin-top: 20px;" v-if="systemInfo && !systemInfo.docker_exists">
        <el-card>
          <template #header>
            <span>Docker安装</span>
          </template>
          
          <el-form :model="installForm" label-width="120px">
            <el-form-item label="安装方式">
              <el-radio-group v-model="installForm.method">
                <el-radio label="online" :disabled="!systemInfo?.has_internet">
                  在线安装
                </el-radio>
                <el-radio label="offline-rpm">
                  离线安装 (RPM)
                </el-radio>
                <el-radio label="offline-deb">
                  离线安装 (DEB)
                </el-radio>
                <el-radio label="offline-binary">
                  离线安装 (二进制)
                </el-radio>
              </el-radio-group>
            </el-form-item>
            
            <el-form-item 
              v-if="installForm.method === 'online'" 
              label="Linux发行版"
            >
              <el-select v-model="installForm.distribution" style="width: 300px">
                <el-option label="Ubuntu/Debian" value="ubuntu" />
                <el-option label="CentOS/RHEL" value="centos" />
                <el-option label="Fedora" value="fedora" />
              </el-select>
            </el-form-item>
            
            <el-form-item 
              v-if="installForm.method.startsWith('offline')" 
              label="安装包路径"
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
                开始安装
              </el-button>
              <el-alert
                style="margin-top: 10px"
                title="注意：安装Docker需要root权限，请确保当前用户有sudo权限"
                type="warning"
                :closable="false"
              />
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
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
      ElMessage.success('Docker运行正常')
    } else {
      ElMessage.warning('Docker未运行或未安装')
    }
  } catch (error) {
    ElMessage.error('检查Docker状态失败')
  }
}

const loadSystemInfo = async () => {
  try {
    systemInfo.value = await systemApi.getSystemInfo()
  } catch (error) {
    ElMessage.error('加载系统信息失败')
  }
}

const handleInstall = async () => {
  if (installForm.method.startsWith('offline') && !installForm.package_path) {
    ElMessage.warning('请输入安装包路径')
    return
  }
  
  installing.value = true
  try {
    await systemApi.installDocker(installForm)
    ElMessage.success('Docker安装成功，请重启系统后生效')
    loadSystemInfo()
    checkStatus()
  } catch (error) {
    ElMessage.error('Docker安装失败，请查看日志')
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
  max-width: 1200px;
}
</style>

