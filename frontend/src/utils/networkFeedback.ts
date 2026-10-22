import { isAxiosError } from 'axios'
export function networkError(error: unknown): string {
  if (isAxiosError(error)) return error.response?.data?.error || (error.code === 'ECONNABORTED' ? '请求超时，请重试' : error.response ? `请求失败（${error.response.status}），请重试` : '无法连接服务，请检查网络后重试')
  return error instanceof Error ? error.message : '操作失败，请重试'
}
