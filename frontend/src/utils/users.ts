export type RegistryAccess = 'none' | 'read_only' | 'read_write'
export interface ManagedUser {
  id: number
  username: string
  is_admin: boolean
  enabled: boolean
  registry_access: RegistryAccess
  created_at: string
  updated_at: string
}
export type UserEditorMode = 'create' | 'authorization' | 'password'
export interface UserForm {
  username: string
  password: string
  confirmPassword: string
  is_admin: boolean
  enabled: boolean
  registry_access: RegistryAccess
}
export const registryAccessLabels: Record<RegistryAccess, string> = { none: '无权限', read_only: '只读', read_write: '读写' }
export const registryAccessDescriptions: Record<RegistryAccess, string> = { none: '不可访问内置仓库', read_only: '可拉取镜像', read_write: '可拉取和推送镜像' }
export const userColumns = [
  { key: 'username', label: '用户名' },
  { key: 'role', label: '角色' },
  { key: 'enabled', label: '状态' },
  { key: 'access', label: '仓库权限' },
  { key: 'created_at', label: '创建时间' },
] as const
export type UserSortKey = typeof userColumns[number]['key']

export function effectiveRegistryAccess(user: Pick<ManagedUser, 'is_admin' | 'registry_access'>) {
  return user.is_admin ? 'all' : user.registry_access
}
export function filterUsers(users: ManagedUser[], search: string, role = '', status = '', access = '') {
  const query = search.trim().toLocaleLowerCase()
  return users.filter(user => user.username.toLocaleLowerCase().includes(query)
    && (!role || user.is_admin === (role === 'admin'))
    && (!status || user.enabled === (status === 'enabled'))
    && (!access || effectiveRegistryAccess(user) === access))
}
export function sortUsers(users: ManagedUser[], key: UserSortKey, descending: boolean) {
  const ranks = { none: 0, read_only: 1, read_write: 2, all: 3 }
  const value = (user: ManagedUser): string | number => {
    if (key === 'role') return Number(user.is_admin)
    if (key === 'enabled') return Number(user.enabled)
    if (key === 'access') return ranks[effectiveRegistryAccess(user)] ?? -1
    if (key === 'created_at') return Date.parse(user.created_at) || 0
    return user.username
  }
  return [...users].sort((a, b) => {
    const left = value(a), right = value(b)
    const order = typeof left === 'number' && typeof right === 'number' ? left - right : String(left).localeCompare(String(right), 'zh-CN', { numeric: true, sensitivity: 'base' })
    return (descending ? -order : order) || a.id - b.id
  })
}
export function isLastEnabledAdmin(user: ManagedUser, users: ManagedUser[]) {
  return user.enabled && user.is_admin && users.filter(item => item.enabled && item.is_admin).length === 1
}
export function emptyUserForm(user?: ManagedUser): UserForm {
  return { username: user?.username ?? '', password: '', confirmPassword: '', is_admin: user?.is_admin ?? false, enabled: user?.enabled ?? true, registry_access: user?.registry_access ?? 'none' }
}
const bytes = (value: string) => new TextEncoder().encode(value).length
export function validateUsername(value: string, users: ManagedUser[]) {
  const name = value.trim()
  if (!name) return '请输入用户名'
  if (bytes(name) > 100) return '用户名不能超过 100 字节'
  if (users.some(user => user.username === name)) return '用户名已存在，请使用其他名称'
  return ''
}
export function validatePassword(value: string) {
  if (!value) return '请输入密码'
  return bytes(value) < 6 || bytes(value) > 72 ? '密码须为 6–72 字节，中文通常占 3 字节' : ''
}
export function authorizationChanged(form: UserForm, user: ManagedUser) {
  return form.is_admin !== user.is_admin || form.enabled !== user.enabled || form.registry_access !== user.registry_access
}
export function buildUserRequest(form: UserForm, mode: UserEditorMode, users: ManagedUser[], user?: ManagedUser) {
  if (mode === 'create') {
    const error = validateUsername(form.username, users) || validatePassword(form.password)
    if (error) throw new Error(error)
  } else if (!user) throw new Error('用户已不存在，请刷新列表')
  if (mode !== 'authorization') {
    const error = validatePassword(form.password)
    if (error) throw new Error(error)
    if (form.password !== form.confirmPassword) throw new Error('两次输入的密码不一致')
  }
  if (mode === 'password') return { password: form.password }
  if (!Object.prototype.hasOwnProperty.call(registryAccessLabels, form.registry_access)) throw new Error('请选择有效的仓库权限')
  if (mode === 'create') return { username: form.username.trim(), password: form.password, is_admin: form.is_admin, registry_access: form.registry_access }
  if (isLastEnabledAdmin(user!, users) && (!form.enabled || !form.is_admin)) throw new Error('不能停用或降级最后一个可用管理员')
  if (!authorizationChanged(form, user!)) throw new Error('授权信息未发生更改')
  return { is_admin: form.is_admin, enabled: form.enabled, registry_access: form.registry_access }
}
export function userDate(value: string) {
  const date = new Date(value)
  if (!value || !Number.isFinite(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}
export function userError(error: unknown): string {
  const response = error as { response?: { status?: number; data?: { error?: string } }; code?: string }
  if (typeof response?.response?.data?.error === 'string') return response.response.data.error
  if (response?.code === 'ECONNABORTED') return '请求超时，请重试'
  if (response?.code === 'ERR_NETWORK') return '无法连接服务，请检查网络后重试'
  if (response?.response?.status === 403) return '只有管理员可以管理用户'
  if (response?.response) return `请求失败（${response.response.status}），请重试`
  return error instanceof Error ? error.message : '操作失败，请重试'
}
