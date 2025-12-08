<template>
  <el-container class="main-layout">
    <el-aside width="200px">
      <div class="logo">
        <h2>DockeRun</h2>
      </div>
      
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409eff"
      >
        <el-menu-item index="/containers">
          <el-icon><Box /></el-icon>
          <span>容器管理</span>
        </el-menu-item>
        
        <el-menu-item index="/images">
          <el-icon><Picture /></el-icon>
          <span>镜像管理</span>
        </el-menu-item>
        
        <el-menu-item index="/networks">
          <el-icon><Connection /></el-icon>
          <span>网络管理</span>
        </el-menu-item>
        
        <el-menu-item index="/volumes">
          <el-icon><FolderOpened /></el-icon>
          <span>卷管理</span>
        </el-menu-item>
        
        <el-menu-item index="/deploy-scripts">
          <el-icon><Document /></el-icon>
          <span>部署脚本</span>
        </el-menu-item>
        
        <el-menu-item index="/system">
          <el-icon><Setting /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    
    <el-container>
      <el-header>
        <div class="header-content">
          <span></span>
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><User /></el-icon>
              {{ authStore.user?.username }}
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="changePassword">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
  
  <!-- 修改密码对话框 -->
  <el-dialog v-model="changePasswordVisible" title="修改密码" width="400px">
    <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef">
      <el-form-item label="原密码" prop="old_password">
        <el-input v-model="passwordForm.old_password" type="password" />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input v-model="passwordForm.new_password" type="password" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="changePasswordVisible = false">取消</el-button>
      <el-button type="primary" @click="handleChangePassword">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const activeMenu = computed(() => route.path)
const changePasswordVisible = ref(false)
const passwordFormRef = ref()

const passwordForm = reactive({
  old_password: '',
  new_password: ''
})

const passwordRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' }
  ]
}

onMounted(() => {
  if (!authStore.user) {
    authStore.fetchCurrentUser()
  }
})

const handleCommand = (command: string) => {
  if (command === 'logout') {
    ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }).then(() => {
      authStore.logout()
      router.push('/login')
    })
  } else if (command === 'changePassword') {
    changePasswordVisible.value = true
  }
}

const handleChangePassword = async () => {
  await passwordFormRef.value.validate()
  
  try {
    await authApi.changePassword(passwordForm)
    ElMessage.success('密码修改成功')
    changePasswordVisible.value = false
    passwordForm.old_password = ''
    passwordForm.new_password = ''
  } catch (error) {
    console.error(error)
  }
}
</script>

<style scoped>
.main-layout {
  height: 100vh;
}

.el-aside {
  background-color: #304156;
  color: #fff;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #2b3947;
}

.logo h2 {
  margin: 0;
  color: #fff;
  font-size: 20px;
}

.el-header {
  background-color: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  padding: 0 20px;
}

.header-content {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-info {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

.el-main {
  background-color: #f0f2f5;
  padding: 20px;
}
</style>

