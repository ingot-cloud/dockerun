import type { AuditEvent, MaintenanceJob, RegistryMaintenanceStatus } from '@/api/maintenance'

export const maintenanceActions: Record<string, string> = {
  'registry.maintenance': '切换只读维护', 'registry.gc.create': '创建镜像层清理', 'registry.gc.complete': '镜像层清理完成', 'registry.gc.failed': '镜像层清理失败',
  'backup.create': '创建完整备份', 'backup.ready': '完整备份可下载', 'backup.failed': '完整备份失败',
  'user.create': '创建用户', 'user.authorization_or_password': '修改账号权限或密码', 'token.create': '创建访问令牌', 'token.revoke': '撤销访问令牌',
  'deployment.create': '创建部署', 'environment.update': '修改应用环境', 'target.create': '创建目标主机', 'target.update': '修改目标主机',
  'bundle.import': '导入发布包', 'bundle.ready': '发布包就绪', 'bundle.publish': '发布镜像版本', 'image.import': '导入镜像',
}
export function isActiveMaintenanceJob(job: Pick<MaintenanceJob, 'status'>) { return job.status === 'queued' || job.status === 'running' }
export function needsMaintenancePolling(backups: MaintenanceJob[], collections: MaintenanceJob[], operation?: string) { return Boolean(operation || [...backups, ...collections].some(isActiveMaintenanceJob)) }
export function maintenanceStorage(bytes: number | null | undefined) {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return '—'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const index = bytes ? Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1) : 0
  return `${index ? (bytes / 1024 ** index).toFixed(1) : bytes} ${units[index]}`
}
export function maintenanceTransport(status: Pick<RegistryMaintenanceStatus, 'public_url' | 'tls'>) {
  if (status.tls) return '服务端 TLS'
  try { return new URL(status.public_url).protocol === 'https:' ? 'HTTPS 入口 / 代理 TLS' : 'HTTP' } catch { return '—' }
}
export function filterMaintenanceJobs(jobs: MaintenanceJob[], search: string, status: string) {
  const query = search.trim().toLowerCase()
  return jobs.filter(job => `${job.id} ${job.file} ${job.error} ${job.user_id}`.toLowerCase().includes(query) && (!status || (status === 'active' ? isActiveMaintenanceJob(job) : status === 'complete' ? ['ready', 'success'].includes(job.status) : job.status === status)))
}
export function filterMaintenanceAudit(events: AuditEvent[], search: string, category: string) {
  const query = search.trim().toLowerCase()
  return events.filter(event => `${event.action} ${maintenanceActions[event.action] || ''} ${event.resource} ${event.user_id}`.toLowerCase().includes(query) && (!category || event.action.startsWith(`${category}.`)))
}
export function maintenanceError(error: unknown): string {
  const response = error as { response?: { status?: number; data?: { error?: string } }; code?: string }
  if (typeof response?.response?.data?.error === 'string') return response.response.data.error
  if (response?.response?.status === 403) return '只有管理员可以管理仓库与维护任务'
  if (response?.response?.status === 404) return '任务或备份文件已不存在，请刷新后重试'
  if (response?.code === 'ECONNABORTED') return '请求超时，请刷新任务状态后重试'
  if (response?.response) return `请求失败（${response.response.status}），请重试`
  return error instanceof Error ? error.message : '操作失败，请重试'
}
