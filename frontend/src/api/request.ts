import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

declare module 'axios' {
  interface AxiosRequestConfig { silentError?: boolean }
}

const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

// 请求拦截器
request.interceptors.request.use(
  config => {
    const authStore = useAuthStore()
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    if (axios.isCancel(error)) return Promise.reject(error)
    if (error.response) {
      const { status, data } = error.response
      
      if (status === 401 && error.config?.url !== '/auth/login') {
        const authStore = useAuthStore()
        authStore.logout()
        router.push('/login')
        ElMessage.error('认证失败，请重新登录')
      } else if (!error.config?.silentError) {
        ElMessage.error(data.error || '请求失败')
      }
    } else if (!error.config?.silentError) {
      ElMessage.error('网络错误')
    }
    
    return Promise.reject(error)
  }
)

export default request

