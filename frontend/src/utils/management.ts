export const statusLabels: Record<string, string> = {
  queued: '排队中', pending: '等待中', running: '执行中', processing: '处理中', success: '成功', failed: '失败',
  needs_attention: '需要核对', cancelled: '已取消', healthy: '健康', unhealthy: '不健康', unverified: '已启动，健康未验证',
  unknown: '状态未知', ready: '可用', validating: '校验中', loading: '镜像加载中', blocked: '等待上一台', paused: '已暂停',
  idle: '空闲', stopped: '已停止', enabled: '启用', disabled: '停用', expired: '已过期', revoked: '已撤销', online: '在线', offline: '离线', local: '本机',
}
export const sourceLabels: Record<string, string> = { registry: '仓库镜像', local: '本机镜像', bundle: '完整发布包', image: '镜像归档', legacy: '历史脚本', compose: '应用栈', simple: '简单容器', shell: '脚本', 'docker-compose': '应用栈' }
export const stageLabels: Record<string, string> = { queued: '排队中', blocked: '等待上一台', failed: '执行失败', legacy: '执行历史脚本', preflight: '预检查', prepare: '准备环境', validate: '校验配置', validating: '校验中', pull: '拉取镜像', pulling: '拉取镜像', load: '加载镜像', loading: '加载镜像', deploy: '部署服务', deploying: '部署服务', apply: '应用配置', health: '检查健康', verify: '验证结果', rollback: '恢复版本', recovery: '恢复版本', complete: '已结束', execute: '执行脚本', script: '执行脚本', snapshot: '保存快照', render: '生成配置', images: '准备镜像', start: '启动服务' }
Object.assign(stageLabels, { before_script: '执行部署前脚本', deploy_script: '执行部署命令', after_script: '执行部署后脚本', check_script: '执行部署检查', containers: '替换关联容器', runtime: '核对运行容器' })
Object.assign(sourceLabels, { command: '命令部署', direct: '镜像地址' })
Object.assign(statusLabels, { unchecked: '未执行健康检查', missing: '容器不存在', created: '已创建', restarting: '重启中', removing: '删除中', exited: '已退出', dead: '已失效' })
export function statusLabel(value?: string) { return statusLabels[value || 'unknown'] || `未知状态（${value}）` }
export function sourceLabel(value?: string) { return sourceLabels[value || ''] || (value ? `其他来源（${value}）` : '—') }
export function stageLabel(value?: string) { return stageLabels[value || ''] || (value ? `其他阶段（${value}）` : '—') }
export function formatDate(value?: string | number) {
  if (value == null || value === '') return '—'
  const date = new Date(value)
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(date) : '—'
}
export function fieldValue(item: any, path: string): unknown { return path.split('.').reduce((value, key) => value?.[key], item) }
export interface ListState { search: string; filter: string; page: number; size: number; sort: string; descending: boolean }
export function readListState(query: Record<string, unknown>, prefix = '', sorts: string[] = ['name'], filters: string[] = []) : ListState {
  const value = (key: string) => typeof query[prefix + key] === 'string' ? query[prefix + key] as string : ''
  const page = Number(value('page'))
  const sort=sorts.includes(value('sort'))?value('sort'):sorts[0] || 'name'
  return { search: value('search'), filter: filters.includes(value('filter')) ? value('filter') : '', page: Number.isSafeInteger(page) && page > 0 ? page : 1, size: [10, 25, 50, 100].includes(Number(value('size'))) ? Number(value('size')) : 10, sort, descending: value('direction') === 'desc' || (!['asc','desc'].includes(value('direction')) && sort.endsWith('_at')) }
}
export function filterAndSort(items: any[], search: string, searchKeys: string[], sort: string, descending: boolean) {
  const words = search.trim().toLocaleLowerCase()
  return items.filter(item => searchKeys.some(key => String(fieldValue(item, key) ?? '').toLocaleLowerCase().includes(words))).sort((a, b) => {
    let left = fieldValue(a, sort), right = fieldValue(b, sort)
    if (sort.endsWith('_at')) { left = Date.parse(String(left)) || 0; right = Date.parse(String(right)) || 0 }
    const order = typeof left === 'number' && typeof right === 'number' ? left - right : String(left ?? '').localeCompare(String(right ?? ''), 'zh-CN', { numeric: true, sensitivity: 'base' })
    return (descending ? -order : order) || String(a.id ?? a.name ?? '').localeCompare(String(b.id ?? b.name ?? ''), 'zh-CN', { numeric: true })
  })
}
export function parseObject(text: string, label: string): Record<string, unknown> {
  let value: unknown
  try { value = JSON.parse(text) } catch { throw new Error(`${label}必须是有效的 JSON 对象`) }
  if (!value || Array.isArray(value) || typeof value !== 'object') throw new Error(`${label}必须是 JSON 对象，不能是数组或其他值`)
  return value as Record<string, unknown>
}
export function parseArray(value: unknown): number[] { try { const list = typeof value === 'string' ? JSON.parse(value) : value; return Array.isArray(list) ? list.filter(item => Number.isSafeInteger(item) && item > 0) : [] } catch { return [] } }
export function hostState(host: { id: number; enabled: boolean; last_seen_at?: string }, now = Date.now()) { return host.id === 1 ? 'local' : !host.enabled ? 'disabled' : host.last_seen_at && now - Date.parse(host.last_seen_at) < 90000 ? 'online' : 'offline' }
export function tokenState(token: { revoked: boolean; expires_at: string }, now = Date.now()) { return token.revoked ? 'revoked' : !Number.isFinite(Date.parse(token.expires_at)) || Date.parse(token.expires_at) <= now ? 'expired' : 'ready' }
