import type { BuiltinImageEvent, BuiltinManifest } from '@/api/builtin'

const collator = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' })
export function registryHost(publicUrl: string, fallback: string) {
  try { return new URL(publicUrl).host || fallback } catch { return fallback }
}
export function registryConnectionHelp(publicUrl: string) {
  try {
    const url = new URL(publicUrl)
    const hostname = url.hostname.replace(/^\[|\]$/g, '').replace(/\.$/, '').toLowerCase()
    const loopback = hostname === 'localhost' || hostname === '::1' || /^127\./.test(hostname)
    const desktopHost = `host.docker.internal${url.port ? `:${url.port}` : ''}`
    return {
      http: url.protocol === 'http:', loopback,
      desktopHost, desktopUrl: `${url.protocol}//${desktopHost}`,
      daemonConfig: JSON.stringify({ 'insecure-registries': [loopback ? desktopHost : url.host] }, null, 2),
    }
  } catch { return { http: false, loopback: false, desktopHost: '', desktopUrl: '', daemonConfig: '' } }
}
export function registryLoginCommand(host: string, username: string) {
  const quote = (value: string) => /^[a-zA-Z0-9_.:@\[\]-]+$/.test(value) ? value : `'${value.replace(/'/g, `'"'"'`)}'`
  return `docker login${username ? ` -u ${quote(username)}` : ''} ${quote(host)}`
}
export function imageReference(host: string, name: string, reference: string) {
  return `${host}/${name}${reference.startsWith('sha256:') ? '@' : ':'}${reference}`
}
export function latestImageEvents(events: BuiltinImageEvent[]) {
  const result = new Map<string, BuiltinImageEvent>()
  for (const event of events) {
    const previous = result.get(event.repository)
    if (!previous || Date.parse(event.created_at) > Date.parse(previous.created_at) || (event.created_at === previous.created_at && event.id > previous.id)) result.set(event.repository, event)
  }
  return result
}
export function repositoryNamespace(name: string) {
  const slash = name.lastIndexOf('/')
  return slash < 0 ? '' : name.slice(0, slash)
}
export function filterBuiltinRepositories(names: string[], search: string, namespace: string, sort: 'name' | 'updated', descending: boolean, latest: Map<string, BuiltinImageEvent>) {
  const query = search.trim().toLowerCase()
  return names.filter(name => name.toLowerCase().includes(query) && (!namespace || (namespace === '__root__' ? !repositoryNamespace(name) : repositoryNamespace(name) === namespace)))
    .sort((a, b) => {
      const comparison = sort === 'updated' ? (Date.parse(latest.get(a)?.created_at || '') || 0) - (Date.parse(latest.get(b)?.created_at || '') || 0) : collator.compare(a, b)
      return (comparison || collator.compare(a, b)) * (descending ? -1 : 1)
    })
}
export function builtinPermission(admin: boolean, permission: string) {
  return admin ? '管理员' : permission === 'read_write' ? '读写' : permission === 'read_only' ? '只读' : '无访问权限'
}
export function builtinTime(value?: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return '—'
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(value))
}
export function manifestSize(detail: BuiltinManifest) {
  if (detail.manifest.manifests) return '多平台镜像'
  const size = (detail.manifest.config?.size || 0) + (detail.manifest.layers || []).reduce((total, layer) => total + (layer.size || 0), 0)
  if (!size) return '—'
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 ** 3) return `${(size / 1024 ** 2).toFixed(1)} MB`
  return `${(size / 1024 ** 3).toFixed(2)} GB`
}
export function builtinError(error: unknown): string {
  const response = error as { response?: { status?: number; data?: { error?: string; catalog?: { errors?: { message?: string }[] } } }; code?: string }
  const data = response?.response?.data
  if (typeof data?.error === 'string' && data.error) return data.error
  const messages = data?.catalog?.errors?.map(item => item.message).filter(Boolean)
  if (messages?.length) return messages.join('；')
  if (response?.response?.status === 403) return '此账号没有仓库访问权限，请联系管理员'
  if (response?.response?.status === 404) return '镜像或标签已不存在，请刷新列表'
  if (response?.response?.status === 503) return '内置仓库未启用或正在维护，请稍后重试'
  if (response?.code === 'ECONNABORTED') return '请求超时，请重试'
  if (response?.response) return `请求失败（${response.response.status}），请重试`
  return error instanceof Error ? error.message : '操作失败，请重试'
}
