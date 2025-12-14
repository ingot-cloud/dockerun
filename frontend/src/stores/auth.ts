import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { encryptPassword } from '@/utils/crypto'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const user = ref<any>(null)

  const isAuthenticated = computed(() => !!token.value)

  const login = async (username: string, password: string) => {
    // 加密密码
    const encryptedPassword = encryptPassword(password)
    const res: any = await authApi.login({ username, password: encryptedPassword })
    token.value = res.token
    user.value = res.user
    localStorage.setItem('token', res.token)
  }

  const logout = () => {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  const fetchCurrentUser = async () => {
    try {
      user.value = await authApi.getCurrentUser()
    } catch (error) {
      logout()
    }
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
    fetchCurrentUser
  }
})

