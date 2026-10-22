import request from './request'

export interface LoginParams {
  username: string
  password: string
}

export interface SetPasswordParams {
  new_password: string
  deploy_work_dir?: string
  max_script_executors?: number
  script_timeout?: number
}

export interface ChangePasswordParams {
  old_password: string
  new_password: string
}

export const authApi = {
  // 登录
  login: (data: LoginParams) => request.post('/auth/login', data, {silentError:true}),
  
  // 检查初始设置
  checkSetup: () => request.get('/auth/check-setup'),
  
  // 设置初始密码
  setInitialPassword: (data: SetPasswordParams) => request.post('/auth/setup', data, {silentError:true}),
  
  // 修改密码
  changePassword: (data: ChangePasswordParams) => request.post('/auth/change-password', data, {silentError:true}),
  
  // 获取当前用户信息
  getCurrentUser: () => request.get('/auth/me')
}

