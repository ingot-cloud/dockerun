<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-header">
        <h1 class="logo">DOCKERUN</h1>
        <h2 class="title">Log in to your account</h2>
        <p class="subtitle">Welcome back! Please enter your details</p>
      </div>
      
      <el-form :model="form" :rules="rules" ref="formRef" class="login-form">
        <el-form-item label="Username" prop="username">
          <el-input 
            v-model="form.username" 
            placeholder="Enter your username"
            size="large"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="Password" prop="password">
          <el-input 
            v-model="form.password" 
            type="password"
            placeholder="Enter your password"
            size="large"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="primary" 
            size="large" 
            class="login-button"
            :loading="loading"
            @click="handleLogin"
          >
            Login
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref()
const loading = ref(false)

const form = reactive({
  username: 'admin',
  password: ''
})

const rules = {
  username: [{ required: true, message: 'Please enter username', trigger: 'blur' }],
  password: [{ required: true, message: 'Please enter password', trigger: 'blur' }]
}

onMounted(async () => {
  // 检查是否需要初始设置
  try {
    const res: any = await authApi.checkSetup()
    if (res.require_setup) {
      router.push('/setup')
    }
  } catch (error) {
    console.error(error)
  }
})

const handleLogin = async () => {
  await formRef.value.validate()
  
  loading.value = true
  try {
    await authStore.login(form.username, form.password)
    ElMessage.success('Login successful')
    router.push('/')
  } catch (error: any) {
    if (error.response?.data?.require_setup) {
      router.push('/setup')
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.login-container {
  width: 100%;
  max-width: 460px;
  background: rgba(42, 42, 56, 0.95);
  border-radius: 8px;
  padding: 48px 40px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.login-header {
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

.login-form {
  margin-top: 32px;
}

.login-form :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
  margin-bottom: 8px;
}

.login-form :deep(.el-input) {
  --el-input-bg-color: rgba(255, 255, 255, 0.08);
  --el-input-border-color: rgba(255, 255, 255, 0.15);
  --el-input-hover-border-color: rgba(56, 139, 253, 0.5);
  --el-input-focus-border-color: #388bfd;
  --el-input-text-color: #ffffff;
  --el-input-placeholder-color: rgba(255, 255, 255, 0.4);
}

.login-form :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
  transition: all 0.3s;
}

.login-form :deep(.el-input__wrapper:hover) {
  border-color: rgba(56, 139, 253, 0.5);
}

.login-form :deep(.el-input__wrapper.is-focus) {
  border-color: #388bfd;
  box-shadow: 0 0 0 2px rgba(56, 139, 253, 0.15);
}

.login-form :deep(.el-input__inner) {
  color: #ffffff;
}

.login-form :deep(.el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.4);
}

.login-button {
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

.login-button:hover {
  background: linear-gradient(135deg, #4a9aff 0%, #3d7ee6 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(56, 139, 253, 0.4);
}

.login-button:active {
  transform: translateY(0);
}

.login-form :deep(.el-form-item) {
  margin-bottom: 24px;
}

.login-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}
</style>
