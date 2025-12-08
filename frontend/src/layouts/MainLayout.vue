<template>
  <el-container class="main-layout">
    <el-aside width="200px" class="sidebar">
      <div class="logo">
        <h2>DOCKERUN</h2>
        <span class="edition">COMMUNITY EDITION</span>
      </div>
      
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#23252a"
        text-color="#8a929c"
        active-text-color="#4a9eff"
        class="sidebar-menu"
      >
        <el-menu-item index="/containers">
          <el-icon><Box /></el-icon>
          <span>Containers</span>
        </el-menu-item>
        
        <el-menu-item index="/images">
          <el-icon><Picture /></el-icon>
          <span>Images</span>
        </el-menu-item>
        
        <el-menu-item index="/networks">
          <el-icon><Connection /></el-icon>
          <span>Networks</span>
        </el-menu-item>
        
        <el-menu-item index="/volumes">
          <el-icon><FolderOpened /></el-icon>
          <span>Volumes</span>
        </el-menu-item>
        
        <el-menu-item index="/deploy-scripts">
          <el-icon><Document /></el-icon>
          <span>Deploy Scripts</span>
        </el-menu-item>
        
        <el-menu-item index="/system">
          <el-icon><Setting /></el-icon>
          <span>System</span>
        </el-menu-item>
      </el-menu>
      
      <div class="sidebar-footer">
        <div class="version-info">Community Edition</div>
      </div>
    </el-aside>
    
    <el-container class="content-wrapper">
      <el-header class="top-header">
        <div class="header-content">
          <span></span>
          <div class="user-section">
            <el-dropdown @command="handleCommand" class="user-dropdown">
              <span class="user-info">
                <el-icon class="user-icon"><User /></el-icon>
                <span class="username">{{ authStore.user?.username || 'admin' }}</span>
                <el-icon class="arrow-icon"><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu class="user-menu">
                  <el-dropdown-item command="changePassword">
                    <el-icon><Lock /></el-icon>
                    Change Password
                  </el-dropdown-item>
                  <el-dropdown-item command="logout" divided>
                    <el-icon><SwitchButton /></el-icon>
                    Logout
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>
      
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
  
  <!-- 修改密码对话框 -->
  <el-dialog 
    v-model="changePasswordVisible" 
    title="Change Password" 
    width="400px"
    class="change-password-dialog"
  >
    <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef">
      <el-form-item label="Old Password" prop="old_password">
        <el-input v-model="passwordForm.old_password" type="password" show-password />
      </el-form-item>
      <el-form-item label="New Password" prop="new_password">
        <el-input v-model="passwordForm.new_password" type="password" show-password />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="changePasswordVisible = false">Cancel</el-button>
      <el-button type="primary" @click="handleChangePassword">Confirm</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  User, 
  Lock, 
  SwitchButton,
  Box,
  Picture,
  Connection,
  FolderOpened,
  Document,
  Setting,
  ArrowDown
} from '@element-plus/icons-vue'
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
  old_password: [{ required: true, message: 'Please enter old password', trigger: 'blur' }],
  new_password: [
    { required: true, message: 'Please enter new password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' }
  ]
}

onMounted(() => {
  if (!authStore.user) {
    authStore.fetchCurrentUser()
  }
})

const handleCommand = (command: string) => {
  if (command === 'logout') {
    ElMessageBox.confirm('Are you sure you want to logout?', 'Confirm', {
      confirmButtonText: 'Yes',
      cancelButtonText: 'No',
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
    ElMessage.success('Password changed successfully')
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
  background-color: #1e1e1e;
}

.sidebar {
  background-color: #23252a;
  color: #fff;
  display: flex;
  flex-direction: column;
}

.logo {
  height: 60px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: #1a1c20;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 12px 0;
}

.logo h2 {
  margin: 0;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 1px;
}

.logo .edition {
  font-size: 10px;
  color: #8a929c;
  margin-top: 2px;
  letter-spacing: 0.5px;
}

.sidebar-menu {
  border: none;
  flex: 1;
}

.sidebar-menu :deep(.el-menu-item) {
  height: 44px;
  line-height: 44px;
  border-left: 3px solid transparent;
  transition: all 0.3s;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: rgba(255, 255, 255, 0.05);
  color: #4a9eff;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: rgba(74, 158, 255, 0.1);
  border-left-color: #4a9eff;
  color: #4a9eff;
}

.sidebar-menu :deep(.el-icon) {
  margin-right: 8px;
  font-size: 16px;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.version-info {
  font-size: 11px;
  color: #8a929c;
  text-align: center;
}

.content-wrapper {
  display: flex;
  flex-direction: column;
}

.top-header {
  background-color: #2a2d31;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  padding: 0 24px;
  height: 60px !important;
}

.header-content {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-section {
  display: flex;
  align-items: center;
}

.user-dropdown {
  cursor: pointer;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
  transition: all 0.3s;
  color: #fff;
}

.user-info:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.user-icon {
  font-size: 18px;
}

.username {
  font-size: 14px;
  font-weight: 500;
}

.arrow-icon {
  font-size: 12px;
  margin-left: 4px;
}

.main-content {
  background-color: #1e1e1e;
  padding: 24px;
  overflow-y: auto;
}

/* 对话框样式 */
.change-password-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.change-password-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.change-password-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.change-password-dialog :deep(.el-dialog__body) {
  color: #fff;
}

.change-password-dialog :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.change-password-dialog :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.change-password-dialog :deep(.el-input__inner) {
  color: #fff;
}

/* 用户菜单样式 */
.user-menu {
  background-color: #2a2d31;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.user-menu :deep(.el-dropdown-menu__item) {
  color: #fff;
}

.user-menu :deep(.el-dropdown-menu__item:hover) {
  background-color: rgba(74, 158, 255, 0.1);
  color: #4a9eff;
}

.user-menu :deep(.el-icon) {
  margin-right: 8px;
}
</style>
