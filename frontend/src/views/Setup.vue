<template>
  <div class="setup-container">
    <el-card class="setup-card">
      <template #header>
        <h2>初始设置</h2>
        <p>请设置管理员密码</p>
      </template>
      
      <el-form :model="form" :rules="rules" ref="formRef">
        <el-form-item prop="password">
          <el-input 
            v-model="form.password" 
            type="password"
            placeholder="新密码（至少6位）"
            prefix-icon="Lock"
            size="large"
          />
        </el-form-item>
        
        <el-form-item prop="confirmPassword">
          <el-input 
            v-model="form.confirmPassword" 
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            size="large"
            @keyup.enter="handleSetup"
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="primary" 
            size="large" 
            style="width: 100%"
            :loading="loading"
            @click="handleSetup"
          >
            设置密码
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
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
    callback(new Error('密码至少6位'))
  } else {
    callback()
  }
}

const validateConfirmPassword = (rule: any, value: any, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules = {
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { validator: validatePassword, trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const handleSetup = async () => {
  await formRef.value.validate()
  
  loading.value = true
  try {
    await authApi.setInitialPassword({ new_password: form.password })
    ElMessage.success('密码设置成功，请登录')
    router.push('/login')
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.setup-card {
  width: 400px;
}

.setup-card :deep(.el-card__header) {
  text-align: center;
  padding: 30px 0 20px;
}

.setup-card h2 {
  margin: 0 0 10px;
  color: #303133;
}

.setup-card p {
  margin: 0;
  color: #909399;
  font-size: 14px;
}
</style>

