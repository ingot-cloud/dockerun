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
  status?: string
  created_at?: string
  updated_at?: string
}

export const deployApi = {
  list: () => request.get('/deploy-scripts'),
  get: (id: number) => request.get(`/deploy-scripts/${id}`),
  create: (data: DeployScript) => request.post('/deploy-scripts', data),
  update: (id: number, data: DeployScript) => request.put(`/deploy-scripts/${id}`, data),
  delete: (id: number) => request.delete(`/deploy-scripts/${id}`),
  execute: (id: number) => request.post(`/deploy-scripts/${id}/execute`)
}

