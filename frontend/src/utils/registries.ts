import type { Registry } from '@/api/registry'

export const registryProviders = [
  { id: 'custom', label: 'Custom registry', description: '兼容 Docker Registry v2 的仓库', url: '' },
  { id: 'dockerhub', label: 'Docker Hub', description: 'Docker 官方镜像仓库', url: 'https://registry-1.docker.io' },
  { id: 'quay', label: 'Quay.io', description: 'Quay 容器镜像仓库', url: 'https://quay.io' },
  { id: 'github', label: 'GitHub', description: 'GitHub Container Registry', url: 'https://ghcr.io' },
  { id: 'gitlab', label: 'GitLab', description: 'GitLab.com 或自托管仓库', url: 'https://registry.gitlab.com' },
  { id: 'azure', label: 'Azure', description: 'Azure Container Registry', url: '' },
] as const
export type RegistryProvider = typeof registryProviders[number]['id']
export type RegistrySortKey = 'name' | 'type' | 'url' | 'authentication'
export const registryColumns: { key: RegistrySortKey; label: string }[] = [
  { key: 'name', label: '名称' }, { key: 'type', label: '类型' }, { key: 'url', label: 'URL' }, { key: 'authentication', label: '身份认证' },
]

export function registryProvider(url: string): RegistryProvider {
  try {
    const host = new URL(/^https?:\/\//i.test(url) ? url : `https://${url}`).hostname.toLowerCase()
    if (['registry-1.docker.io', 'index.docker.io', 'docker.io'].includes(host)) return 'dockerhub'
    if (host === 'quay.io') return 'quay'
    if (host === 'ghcr.io') return 'github'
    if (host === 'registry.gitlab.com') return 'gitlab'
    if (host.endsWith('.azurecr.io')) return 'azure'
  } catch { /* Invalid existing addresses remain editable as custom registries. */ }
  return 'custom'
}
export function registryType(url: string) { return registryProviders.find(provider => provider.id === registryProvider(url))!.label }
export function filterRegistries(items: Registry[], search = '', type = '', authentication = '', defaults = '') {
  const query = search.trim().toLocaleLowerCase()
  return items.filter(item => item.kind !== 'builtin'
    && (!query || [item.name, item.url, item.username, registryType(item.url)].some(value => value?.toLocaleLowerCase().includes(query)))
    && (!type || registryProvider(item.url) === type)
    && (!authentication || Boolean(item.username) === (authentication === 'true'))
    && (!defaults || item.is_default === (defaults === 'true')))
}
export function sortRegistries(items: Registry[], key: RegistrySortKey, descending: boolean) {
  const value = (item: Registry) => key === 'type' ? registryType(item.url) : key === 'authentication' ? String(Boolean(item.username)) : item[key]
  return [...items].sort((a, b) => value(a).localeCompare(value(b), 'zh-CN', { numeric: true }) * (descending ? -1 : 1))
}
export function normalizeRegistryUrl(value: string) {
  const input = value.trim()
  if (!input) throw new Error('请填写仓库地址')
  const url = new URL(/^[a-z][a-z\d+.-]*:\/\//i.test(input) ? input : `https://${input}`)
  if (!['http:', 'https:'].includes(url.protocol) || !url.hostname || url.username || url.password || url.search || url.hash || !/^\/*$/.test(url.pathname)) {
    throw new Error('请填写有效的 HTTP(S) 仓库地址，不包含镜像路径、凭据或查询参数')
  }
  return url.origin
}
export function emptyRegistryForm() {
  return { name: '', url: '', authentication: false, username: '', password: '', email: '', pull_username: '', pull_password: '', pull_only_confirmed: false, is_default: false, insecure: false }
}
export type RegistryForm = ReturnType<typeof emptyRegistryForm>
export function buildRegistryRequest(form: RegistryForm, editing = false): Partial<Registry> {
  if (!form.name.trim()) throw new Error('请填写仓库名称')
  if (form.authentication && !form.username.trim()) throw new Error('请填写认证用户名')
  if (form.authentication && !editing && !form.password) throw new Error('请填写密码或访问令牌')
  if (form.pull_username.trim() && (!form.pull_only_confirmed || (!editing && !form.pull_password))) throw new Error('请填写 Agent 拉取密码，并确认该账号已限制为只读')
  return { ...form, name: form.name.trim(), url: normalizeRegistryUrl(form.url), username: form.authentication ? form.username.trim() : '', password: form.authentication ? form.password : '', email: form.email.trim(), pull_username: form.pull_username.trim() }
}
export function registryError(error: unknown): string {
  const response = (error as { response?: { data?: { error?: string } }; code?: string })
  return response?.response?.data?.error || (response?.code === 'ECONNABORTED' ? '请求超时，请重试' : error instanceof Error ? error.message : '操作失败，请重试')
}
export function readRegistryListState(query: Record<string, unknown>) {
  const text = (key: string) => typeof query[key] === 'string' ? query[key] as string : ''
  const page = Number(text('page'))
  const size = Number(text('size'))
  return {
    search: text('search'), type: registryProviders.some(provider => provider.id === text('type')) ? text('type') : '',
    authentication: ['true', 'false'].includes(text('authentication')) ? text('authentication') : '',
    defaults: ['true', 'false'].includes(text('defaults')) ? text('defaults') : '',
    page: Number.isSafeInteger(page) && page > 0 ? page : 1, size: [10, 25, 50, 100].includes(size) ? size : 10,
    sort: registryColumns.some(column => column.key === text('sort')) ? text('sort') as RegistrySortKey : 'name' as RegistrySortKey,
    descending: text('direction') === 'desc',
  }
}
export function registryListQuery(state: ReturnType<typeof readRegistryListState>) {
  return { search: state.search || undefined, type: state.type || undefined, authentication: state.authentication || undefined, defaults: state.defaults || undefined, page: String(state.page), size: String(state.size), sort: state.sort, direction: state.descending ? 'desc' : 'asc' }
}
