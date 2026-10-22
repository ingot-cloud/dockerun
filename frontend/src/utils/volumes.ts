export interface DockerVolume {
  Name: string
  Driver: string
  Mountpoint: string
  CreatedAt?: string
  Scope?: string
  Labels?: Record<string, string> | null
  Options?: Record<string, string> | null
  Status?: Record<string, unknown> | null
}
export interface VolumeMount { Type?: string; Name?: string; Source?: string; Destination?: string; RW?: boolean }
export interface VolumeContainer { Id: string; Names?: string[] | null; State?: string; Mounts?: VolumeMount[] | null }
export interface VolumeReference { id: string; name: string; state: string; destination: string; readOnly: boolean | null }
export function volumeReferences(volume: DockerVolume, containers: VolumeContainer[] | null): VolumeReference[] {
  return (containers || []).flatMap(container => (container.Mounts || [])
    .filter(mount => mount.Type === 'volume' && (mount.Name ? mount.Name === volume.Name : Boolean(volume.Mountpoint) && mount.Source === volume.Mountpoint))
    .map(mount => ({ id: container.Id, name: container.Names?.[0]?.replace(/^\//, '') || container.Id.slice(0, 12), state: container.State || '', destination: mount.Destination || '—', readOnly: typeof mount.RW === 'boolean' ? !mount.RW : null })))
}
export function volumeUseCount(volume: DockerVolume, containers: VolumeContainer[] | null): number | null {
  if (!containers) return null
  const count = new Set(volumeReferences(volume, containers).map(reference => reference.id)).size
  return count || !containers.some(container => container.Mounts === undefined) ? count : null
}
export const volumeStack = (volume: DockerVolume) => volume.Labels?.['com.docker.compose.project'] || volume.Labels?.['com.docker.stack.namespace'] || ''
export const volumeColumns = [
  { key: 'Name', label: '名称' }, { key: 'stack', label: '应用栈' }, { key: 'Driver', label: '驱动' },
  { key: 'Mountpoint', label: '挂载路径' }, { key: 'CreatedAt', label: '创建时间' }, { key: 'Scope', label: '范围' },
] as const
export type VolumeSortKey = typeof volumeColumns[number]['key']
export function volumeUsage(volume: DockerVolume, containers: VolumeContainer[] | null) {
  const count = volumeUseCount(volume, containers)
  return count === null ? 'unknown' : count > 0 ? 'used' : 'unused'
}
export function filterVolumes(volumes: DockerVolume[], containers: VolumeContainer[] | null, search: string, driver = '', usage = '') {
  const query = search.trim().toLowerCase()
  return volumes.filter(volume => (!driver || volume.Driver === driver) && (!usage || volumeUsage(volume, containers) === usage)
    && (!query || [volume.Name, volume.Driver, volume.Mountpoint, volume.Scope, volumeStack(volume), ...Object.entries(volume.Labels || {}).flat()].some(value => value?.toLowerCase().includes(query))))
}
export function sortVolumes(volumes: DockerVolume[], key: VolumeSortKey, descending: boolean) {
  const value = (volume: DockerVolume) => key === 'stack' ? volumeStack(volume) : volume[key] || ''
  return [...volumes].sort((a, b) => {
    const left = value(a), right = value(b)
    const difference = key === 'CreatedAt' ? (Date.parse(left) || 0) - (Date.parse(right) || 0) : left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
    return (difference || a.Name.localeCompare(b.Name)) * (descending ? -1 : 1)
  })
}
export function formatVolumeDate(value?: string) {
  return !value || Number.isNaN(Date.parse(value)) ? '—' : new Date(value).toLocaleString('zh-CN', { hour12: false })
}
/** Inspect may contain CIFS passwords in the comma-separated mount options. */
export function displayVolumeOption(key: string, value: string) {
  if (/(password|passwd|secret|token|credential)/i.test(key)) return '••••••'
  return value.replace(/(^|,)(\s*(?:password|passwd|pass|secret|token)\s*=)[^,]*/gi, '$1$2••••••')
}
export interface VolumeListState { search: string; driver: string; usage: string; page: number; size: number; sort: VolumeSortKey; descending: boolean }
export function readVolumeListState(query: Record<string, unknown>): VolumeListState {
  const value = (key: string) => typeof query[key] === 'string' ? query[key] as string : ''
  const page = Number(value('page'))
  return { search: value('search'), driver: value('driver'), usage: ['used', 'unused', 'unknown'].includes(value('usage')) ? value('usage') : '',
    page: Number.isSafeInteger(page) && page > 0 ? page : 1, size: [10, 25, 50, 100].includes(Number(value('size'))) ? Number(value('size')) : 10,
    sort: volumeColumns.some(column => column.key === value('sort')) ? value('sort') as VolumeSortKey : 'Name', descending: value('direction') === 'desc' }
}
export function volumeListQuery(state: VolumeListState) {
  return { search: state.search || undefined, driver: state.driver || undefined, usage: state.usage || undefined, page: String(state.page), size: String(state.size), sort: state.sort, direction: state.descending ? 'desc' : 'asc' }
}
export interface VolumePair { key: string; value: string }
export interface VolumeForm {
  name: string; driver: string; mode: 'custom' | 'nfs' | 'cifs'; options: VolumePair[]; labels: VolumePair[]
  nfs: { address: string; version: 'nfs' | 'nfs4'; path: string; options: string }
  cifs: { address: string; share: string; version: string; username: string; password: string }
}
export interface VolumeCreateRequest { name: string; driver: string; options: Record<string, string>; labels: Record<string, string> }
export function emptyVolumeForm(): VolumeForm {
  return { name: '', driver: 'local', mode: 'custom', options: [], labels: [], nfs: { address: '', version: 'nfs4', path: '', options: 'rw,noatime,rsize=8192,wsize=8192,tcp,timeo=14' }, cifs: { address: '', share: '', version: '3.0', username: '', password: '' } }
}
function pairsToRecord(pairs: VolumePair[], label: string) {
  const keys = new Set<string>(), entries: [string, string][] = []
  for (const pair of pairs) {
    const key = pair.key.trim()
    if (!key) throw new Error(`请填写${label}名称，或移除空白行`)
    if (keys.has(key)) throw new Error(`${label}名称「${key}」重复`)
    keys.add(key); entries.push([key, pair.value.trim()])
  }
  return Object.fromEntries(entries)
}
function serverAddress(value: string, label: string) {
  const address = value.trim()
  if (!address || /[\s,\/@=\u0000]/.test(address)) throw new Error(`请填写有效的 ${label} 服务器主机名或 IP 地址，不包含协议和路径`)
  return address
}
export function buildVolumeRequest(form: VolumeForm): VolumeCreateRequest {
  const name = form.name.trim(), driver = form.driver.trim() || 'local'
  if (name && !/^[a-zA-Z0-9][a-zA-Z0-9_.-]+$/.test(name)) throw new Error('卷名称至少 2 个字符，需以字母或数字开头，且只包含字母、数字、下划线、点或连字符')
  if (/\s/.test(driver)) throw new Error('驱动名称不能包含空格')
  if (form.mode !== 'custom' && driver !== 'local') throw new Error('NFS 和 CIFS 卷需要使用 local 驱动')
  let options: Record<string, string>
  if (form.mode === 'nfs') {
    const address = serverAddress(form.nfs.address, 'NFS'), path = form.nfs.path.trim(), mountOptions = form.nfs.options.trim()
    if (!path.startsWith('/') || /[\u0000\r\n]/.test(path)) throw new Error('NFS 挂载路径必须是以 / 开头的绝对路径')
    if (!['nfs', 'nfs4'].includes(form.nfs.version)) throw new Error('请选择有效的 NFS 版本')
    if (/(^|,)\s*addr\s*=/.test(mountOptions) || /[\u0000\r\n]/.test(mountOptions)) throw new Error('NFS 选项中无需填写 addr，且不能包含换行')
    options = { type: form.nfs.version, device: `:${path}`, o: `addr=${address}${mountOptions ? `,${mountOptions}` : ''}` }
  } else if (form.mode === 'cifs') {
    const address = serverAddress(form.cifs.address, 'CIFS'), share = form.cifs.share.trim().replace(/^\/+/, ''), username = form.cifs.username.trim()
    if (!share || /[\u0000\r\n]/.test(share)) throw new Error('请填写 CIFS 共享目录名称')
    if (!['1.0', '2.0', '2.1', '3.0', '3.1.1'].includes(form.cifs.version)) throw new Error('请选择有效的 CIFS 版本')
    if (!username) throw new Error('请填写 CIFS 用户名')
    if (/[,\u0000\r\n]/.test(username + form.cifs.password)) throw new Error('CIFS 用户名和密码不能包含逗号或换行；可通过驱动选项配置 credentials 文件')
    options = { type: 'cifs', device: `//${address}/${share}`, o: `addr=${address},vers=${form.cifs.version},username=${username}${form.cifs.password ? `,password=${form.cifs.password}` : ''}` }
  } else options = pairsToRecord(form.options, '驱动选项')
  return { name, driver, options, labels: pairsToRecord(form.labels, '标签') }
}
