import request from './request'

export interface DeployScript {
  id?: number
  name: string
  description: string
  script_type: 'shell' | 'docker-compose'
  content: string
  compose_file?: string
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

