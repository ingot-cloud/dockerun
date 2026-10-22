<template>
  <main class="auth-page">
    <div class="auth-card">
      <header class="auth-header">
        <div class="auth-brand"><span>D</span>Dockerun</div>
        <h1>登录工作台</h1>
        <p>使用工作台账号登录。</p>
      </header>

      <div v-if="error" class="management-error" role="alert">{{ error }}</div>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        hide-required-asterisk
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="form.username"
            name="username"
            autocomplete="username"
            autocapitalize="off"
            :spellcheck="false"
            placeholder="请输入用户名"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            name="password"
            autocomplete="current-password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>

        <el-button
          type="primary"
          native-type="submit"
          class="login-button"
          :loading="loading"
        >
          {{ loading ? '登录中…' : '登录' }}
        </el-button>
      </el-form>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'
import { errorText } from '@/api/workspace'
import '@/styles/auth.css'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const error = ref('')

const form = reactive({
  username: '',
  password: ''
})

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
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
  if (!formRef.value || loading.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid || loading.value) return

  loading.value = true
  error.value = ''
  try {
    await authStore.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/')
  } catch (cause: any) {
    error.value = errorText(cause)
    if (cause.response?.data?.require_setup) {
      router.push('/setup')
    }
  } finally {
    loading.value = false
  }
}
</script>
