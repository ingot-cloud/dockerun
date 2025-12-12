<template>
  <div class="setup-page">
    <div class="setup-container">
      <div class="setup-header">
        <h1 class="logo">DOCKERUN</h1>
        <h2 class="title">Initial Setup</h2>
        <p class="subtitle">Configure your system settings</p>
      </div>
      
      <el-form :model="form" :rules="rules" ref="formRef" class="setup-form" label-position="top">
        <div class="form-section">
          <h3 class="section-title">Admin Password</h3>
          <el-form-item label="New Password" prop="password">
            <el-input 
              v-model="form.password" 
              type="password"
              placeholder="Password (at least 6 characters)"
              size="large"
              show-password
            />
          </el-form-item>
          
          <el-form-item label="Confirm Password" prop="confirmPassword">
            <el-input 
              v-model="form.confirmPassword" 
              type="password"
              placeholder="Confirm your password"
              size="large"
              show-password
            />
          </el-form-item>
        </div>

        <div class="form-section">
          <h3 class="section-title">System Configuration</h3>
          <el-form-item label="Deploy Work Directory" prop="deployWorkDir">
            <el-input 
              v-model="form.deployWorkDir" 
              placeholder="e.g., /opt/dockerun"
              size="large"
            >
              <template #prepend>
                <el-icon><Folder /></el-icon>
              </template>
            </el-input>
            <div class="form-tip">Directory for deployment script files</div>
          </el-form-item>

          <el-form-item label="Max Script Executors" prop="maxScriptExecutors">
            <el-input-number 
              v-model="form.maxScriptExecutors" 
              :min="1"
              :max="20"
              size="large"
              style="width: 100%;"
            />
            <div class="form-tip">Maximum number of concurrent script executions</div>
          </el-form-item>

          <el-form-item label="Script Timeout (seconds)" prop="scriptTimeout">
            <el-input-number 
              v-model="form.scriptTimeout" 
              :min="60"
              :max="7200"
              :step="60"
              size="large"
              style="width: 100%;"
            />
            <div class="form-tip">Maximum execution time for scripts</div>
          </el-form-item>
        </div>
        
        <el-form-item>
          <el-button 
            type="primary" 
            size="large" 
            class="setup-button"
            :loading="loading"
            @click="handleSetup"
          >
            Complete Setup
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Folder } from '@element-plus/icons-vue'
import { authApi } from '@/api/auth'

const router = useRouter()
const formRef = ref()
const loading = ref(false)

const form = reactive({
  password: '',
  confirmPassword: '',
  deployWorkDir: '/opt/dockerun',
  maxScriptExecutors: 5,
  scriptTimeout: 3600
})

const validatePassword = (rule: any, value: any, callback: any) => {
  if (value.length < 6) {
    callback(new Error('Password must be at least 6 characters'))
  } else {
    callback()
  }
}

const validateConfirmPassword = (rule: any, value: any, callback: any) => {
  if (value !== form.password) {
    callback(new Error('Passwords do not match'))
  } else {
    callback()
  }
}

const rules = {
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { validator: validatePassword, trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: 'Please confirm password', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ],
  deployWorkDir: [
    { required: true, message: 'Please enter deploy work directory', trigger: 'blur' }
  ],
  maxScriptExecutors: [
    { required: true, message: 'Please set max script executors', trigger: 'blur' }
  ],
  scriptTimeout: [
    { required: true, message: 'Please set script timeout', trigger: 'blur' }
  ]
}

const handleSetup = async () => {
  await formRef.value.validate()
  
  loading.value = true
  try {
    await authApi.setInitialPassword({ 
      new_password: form.password,
      deploy_work_dir: form.deployWorkDir,
      max_script_executors: form.maxScriptExecutors,
      script_timeout: form.scriptTimeout
    })
    ElMessage.success('Setup completed successfully, please login')
    router.push('/login')
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.setup-container {
  width: 100%;
  max-width: 600px;
  background: rgba(42, 42, 56, 0.95);
  border-radius: 8px;
  padding: 48px 40px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  max-height: 90vh;
  overflow-y: auto;
}

.setup-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 32px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 16px 0;
  letter-spacing: 2px;
}

.title {
  font-size: 24px;
  font-weight: 600;
  color: #ffffff;
  margin: 0 0 8px 0;
}

.subtitle {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
  margin: 0;
}

.setup-form {
  margin-top: 24px;
}

.form-section {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.form-section:last-of-type {
  border-bottom: none;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  margin: 0 0 20px 0;
}

.form-tip {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
  margin-top: 4px;
}

.setup-form :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
  margin-bottom: 8px;
}

.setup-form :deep(.el-input) {
  --el-input-bg-color: rgba(255, 255, 255, 0.08);
  --el-input-border-color: rgba(255, 255, 255, 0.15);
  --el-input-hover-border-color: rgba(56, 139, 253, 0.5);
  --el-input-focus-border-color: #388bfd;
  --el-input-text-color: #ffffff;
  --el-input-placeholder-color: rgba(255, 255, 255, 0.4);
}

.setup-form :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
  transition: all 0.3s;
}

.setup-form :deep(.el-input__wrapper:hover) {
  border-color: rgba(56, 139, 253, 0.5);
}

.setup-form :deep(.el-input__wrapper.is-focus) {
  border-color: #388bfd;
  box-shadow: 0 0 0 2px rgba(56, 139, 253, 0.15);
}

.setup-form :deep(.el-input__inner) {
  color: #ffffff;
}

.setup-form :deep(.el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.4);
}

.setup-form :deep(.el-input-group__prepend) {
  background-color: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.6);
}

.setup-form :deep(.el-input-number) {
  width: 100%;
}

.setup-form :deep(.el-input-number .el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.setup-form :deep(.el-input-number__decrease),
.setup-form :deep(.el-input-number__increase) {
  background-color: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.85);
}

.setup-form :deep(.el-input-number__decrease:hover),
.setup-form :deep(.el-input-number__increase:hover) {
  color: #388bfd;
}

.setup-button {
  width: 100%;
  height: 48px;
  background: linear-gradient(135deg, #388bfd 0%, #2c6dd5 100%);
  border: none;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.5px;
  margin-top: 8px;
  transition: all 0.3s;
}

.setup-button:hover {
  background: linear-gradient(135deg, #4a9aff 0%, #3d7ee6 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(56, 139, 253, 0.4);
}

.setup-button:active {
  transform: translateY(0);
}

.setup-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.setup-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
  margin-top: 24px;
}
</style>
