import { isAxiosError } from 'axios'
import { ElMessage } from 'element-plus'
export function containerError(error: unknown): string {
  if (isAxiosError(error)) {
    let data = error.response?.data
    if (typeof data === 'string') { try { data = JSON.parse(data) } catch { data = undefined } }
    if (typeof data?.error === 'string') return data.error
    if (error.code === 'ECONNABORTED') return '请求超时，请重试'
    return error.response ? `请求失败（${error.response.status}），请重试` : '无法连接服务，请检查网络后重试'
  }
  return error instanceof Error ? error.message : '操作失败，请重试'
}
export async function copyContainerText(value: string) {
  try { await navigator.clipboard.writeText(value); ElMessage.success('已复制到剪贴板') }
  catch { ElMessage.error('复制失败，请检查浏览器剪贴板权限') }
}
export function downloadContainerText(name: string, value: string) {
  const url = URL.createObjectURL(new Blob([value], { type: 'text/plain;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url; link.download = name; link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}
