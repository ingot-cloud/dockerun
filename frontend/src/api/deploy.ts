import request from './request'

export interface DeployScriptFile {
  id?: number
  deploy_script_id?: number
  file_name: string
  file_path: string
  file_type: 'compose' | 'config' | 'script' | 'other'
  content: string
  description?: string
  created_at?: string
  updated_at?: string
}

export interface DeployScript {
  id?: number
  name: string
  description: string
  script_type: 'shell' | 'docker-compose'
  main_script: string
  work_dir?: string
  env_vars?: Record<string, string>
  files?: DeployScriptFile[]
  container_id?: string
  container_name?: string
  status?: string
  created_at?: string
  updated_at?: string
  
  // Webhook 自动部署配置
  auto_deploy?: boolean
  image_name?: string
  image_tag?: string
}

export interface TemplateFile {
  file_name: string
  file_path: string
  file_type: string
  content: string
  description?: string
}

export interface ScriptTemplate {
  id: string
  name: string
  description: string
  category: string
  icon: string
  main_script: string
  work_dir: string
  env_vars: Record<string, string>
  files: TemplateFile[]
}

export const deployApi = {
  list: () => request.get('/deploy-scripts'),
  get: (id: number) => request.get(`/deploy-scripts/${id}`),
  create: (data: DeployScript) => request.post('/deploy-scripts', data),
  update: (id: number, data: DeployScript) => request.put(`/deploy-scripts/${id}`, data),
  delete: (id: number) => request.delete(`/deploy-scripts/${id}`),
  execute: (id: number) => request.post(`/deploy-scripts/${id}/execute`),
  
  // 模版相关
  getTemplates: () => request.get<ScriptTemplate[]>('/templates'),
  getTemplate: (id: string) => request.get<ScriptTemplate>(`/templates/${id}`)
}

