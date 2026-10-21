<template>
  <main class="login-page">
    <div class="login-container">
      <header class="login-header">
        <h1 class="logo">DOCKERUN</h1>
        <h2 class="title">登录您的账号</h2>
        <p class="subtitle">欢迎回来，请输入您的账号信息</p>
      </header>

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

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

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
  try {
    await authStore.login(form.username, form.password)
    ElMessage.success('登录成功')
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
  min-height: 100svh;
  background: #111111;
  color: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
}

.login-container {
  width: 100%;
  max-width: 472px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  font-size: 38px;
  line-height: 46px;
  font-weight: 800;
  margin: 0 0 28px;
  letter-spacing: -0.8px;
}

.title {
  font-size: 20px;
  line-height: 28px;
  font-weight: 500;
  margin: 0 0 8px;
}

.subtitle {
  font-size: 14px;
  line-height: 22px;
  color: #b3b3b3;
  margin: 0;
}

.login-form {
  padding: 44px 26px 34px;
  background: #212121;
  border-radius: 8px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 24px;
}

.login-form :deep(.el-form-item:last-of-type) {
  margin-bottom: 28px;
}

.login-form :deep(.el-form-item__label) {
  color: #f5f5f5;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  padding: 0;
  margin-bottom: 8px;
}

.login-form :deep(.el-input) {
  --el-input-bg-color: #171717;
  --el-input-border-color: #383838;
  --el-input-hover-border-color: #666666;
  --el-input-focus-border-color: #ededed;
  --el-input-text-color: #f5f5f5;
  --el-input-placeholder-color: #999999;
  --el-input-icon-color: #b3b3b3;
  height: 36px;
}

.login-form :deep(.el-input__wrapper) {
  background: var(--el-input-bg-color);
  border: 1px solid var(--el-input-border-color);
  border-radius: 5px;
  padding: 0 12px;
  transition: border-color 0.15s;
}

.login-form :deep(.el-input__wrapper:hover) {
  border-color: var(--el-input-hover-border-color);
}

.login-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--el-input-focus-border-color);
}

.login-form :deep(.el-form-item.is-error .el-input__wrapper) {
  border-color: var(--el-color-danger);
}

.login-form :deep(.el-input__inner) {
  color: var(--el-input-text-color);
  font-size: 14px;
}

.login-form :deep(.el-input__inner::placeholder) {
  color: var(--el-input-placeholder-color);
}

.login-form :deep(.el-input__inner:-webkit-autofill) {
  -webkit-text-fill-color: #f5f5f5;
  box-shadow: 0 0 0 1000px #171717 inset;
  caret-color: #f5f5f5;
}

.login-button {
  --el-button-text-color: #212121;
  --el-button-bg-color: #f5f5f5;
  --el-button-border-color: #f5f5f5;
  --el-button-hover-text-color: #111111;
  --el-button-hover-bg-color: #e0e0e0;
  --el-button-hover-border-color: #e0e0e0;
  --el-button-active-text-color: #111111;
  --el-button-active-bg-color: #cccccc;
  --el-button-active-border-color: #cccccc;
  --el-button-disabled-text-color: #666666;
  --el-button-disabled-bg-color: #cccccc;
  --el-button-disabled-border-color: #cccccc;
  width: 100%;
  height: 46px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  transition: background-color 0.15s, border-color 0.15s;
}

.login-button:focus-visible {
  outline: 2px solid #f5f5f5;
  outline-offset: 4px;
}

@media (max-width: 520px) {
  .login-page {
    padding: 32px 20px;
  }

  .login-form {
    padding: 32px 24px;
  }

  .logo {
    font-size: 34px;
  }
}
</style>
