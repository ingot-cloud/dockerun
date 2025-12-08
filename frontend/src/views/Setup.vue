<template>
  <div class="setup-page">
    <div class="setup-container">
      <div class="setup-header">
        <h1 class="logo">DOCKERUN</h1>
        <h2 class="title">Initial Setup</h2>
        <p class="subtitle">Please set your admin password</p>
      </div>
      
      <el-form :model="form" :rules="rules" ref="formRef" class="setup-form">
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
            @keyup.enter="handleSetup"
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="primary" 
            size="large" 
            class="setup-button"
            :loading="loading"
            @click="handleSetup"
          >
            Set Password
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
import { authApi } from '@/api/auth'

const router = useRouter()
const formRef = ref()
const loading = ref(false)

const form = reactive({
  password: '',
  confirmPassword: ''
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
  ]
}

const handleSetup = async () => {
  await formRef.value.validate()
  
  loading.value = true
  try {
    await authApi.setInitialPassword({ new_password: form.password })
    ElMessage.success('Password set successfully, please login')
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
  max-width: 460px;
  background: rgba(42, 42, 56, 0.95);
  border-radius: 8px;
  padding: 48px 40px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.setup-header {
  text-align: center;
  margin-bottom: 40px;
}

.logo {
  font-size: 32px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 24px 0;
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
  margin-top: 32px;
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
  margin-bottom: 24px;
}

.setup-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}
</style>
