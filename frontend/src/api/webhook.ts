import request from './request'

export interface WebhookConfig {
  id?: number
  name: string
  description?: string
  enabled: boolean
  registry_id?: number
  registry?: {
    id: number
    name: string
    url: string
  }
  image_name: string
  image_tag?: string
  deploy_script_id: number
  deploy_script?: {
    id: number
    name: string
  }
  remove_old_image: boolean
  remove_old_container: boolean
  secret?: string
  last_triggered_at?: string
  trigger_count: number
  created_at?: string
  updated_at?: string
}

export interface WebhookLog {
  id: number
  webhook_config_id: number
  webhook_config?: WebhookConfig
  image_name: string
  image_tag: string
  status: 'pending' | 'processing' | 'success' | 'failed'
  message: string
  started_at?: string
  completed_at?: string
  execution_time: number
  created_at: string
}

export interface WebhookURL {
  url: string
  note: string
}

export const webhookApi = {
  // 列出所有配置
  list: () => {
    return request<WebhookConfig[]>({
      url: '/webhook-configs',
      method: 'GET'
    })
  },

  // 获取单个配置
  get: (id: number) => {
    return request<WebhookConfig>({
      url: `/webhook-configs/${id}`,
      method: 'GET'
    })
  },

  // 创建配置
  create: (data: Partial<WebhookConfig>) => {
    return request<WebhookConfig>({
      url: '/webhook-configs',
      method: 'POST',
      data
    })
  },

  // 更新配置
  update: (id: number, data: Partial<WebhookConfig>) => {
    return request<WebhookConfig>({
      url: `/webhook-configs/${id}`,
      method: 'PUT',
      data
    })
  },

  // 删除配置
  delete: (id: number) => {
    return request({
      url: `/webhook-configs/${id}`,
      method: 'DELETE'
    })
  },

  // 获取 Webhook URL
  getUrl: (id: number) => {
    return request<WebhookURL>({
      url: `/webhook-configs/${id}/url`,
      method: 'GET'
    })
  },

  // 列出日志
  listLogs: (configId?: number) => {
    return request<WebhookLog[]>({
      url: '/webhook-logs',
      method: 'GET',
      params: configId ? { config_id: configId } : undefined
    })
  }
}

