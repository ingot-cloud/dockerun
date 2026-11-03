export interface Container {
  Id: string
  Names?: string[] | null
  Image: string
  ImageID?: string
  ImageReference?: string
  State: string
  Status?: string
  Created: number
  Labels?: Record<string, string> | null
  NetworkSettings?: { Networks?: Record<string, { NetworkID?: string; IPAddress?: string; GlobalIPv6Address?: string; Gateway?: string; MacAddress?: string }> | null }
  Ports?: { IP?: string; PrivatePort: number; PublicPort?: number; Type: string }[] | null
}

export interface ContainerDetails {
  Id: string
  Name?: string
  Created: string
  Image?: string
  ImageReference?: string
  RestartCount?: number
  State?: { Status?: string; Running?: boolean; ExitCode?: number; Error?: string; StartedAt?: string; FinishedAt?: string; Health?: { Status: string; FailingStreak: number; Log?: { Start: string; End: string; ExitCode: number; Output: string }[] } }
  Config?: { Image?: string; Entrypoint?: string[] | null; Cmd?: string[] | null; WorkingDir?: string; User?: string; Env?: string[] | null; Labels?: Record<string, string> | null; Tty?: boolean; OpenStdin?: boolean }
  HostConfig?: { RestartPolicy?: { Name?: string; MaximumRetryCount?: number }; PortBindings?: Record<string, { HostIp: string; HostPort: string }[] | null> }
  NetworkSettings?: Container['NetworkSettings']
  Mounts?: { Type: string; Source: string; Destination: string; RW: boolean }[]
}

export interface ContainerStats {
  read?: string
  cpu_stats?: { cpu_usage?: { total_usage?: number; percpu_usage?: number[] }; system_cpu_usage?: number; online_cpus?: number }
  precpu_stats?: { cpu_usage?: { total_usage?: number }; system_cpu_usage?: number }
  memory_stats?: { usage?: number; limit?: number; stats?: { inactive_file?: number; total_inactive_file?: number } }
  networks?: Record<string, { rx_bytes?: number; tx_bytes?: number }>
  pids_stats?: { current?: number }
  blkio_stats?: { io_service_bytes_recursive?: { op: string; value: number }[] }
}

export type ContainerAction = 'start' | 'stop' | 'kill' | 'restart' | 'pause' | 'resume' | 'remove'
export type ContainerSortKey = 'name' | 'State' | 'stack' | 'Image' | 'Created'

export const containerStates: Record<string, string> = {
  running: '运行中', exited: '已停止', created: '已创建', paused: '已暂停',
  restarting: '重启中', removing: '移除中', dead: '异常',
}
export const containerStateLabel = (state: string) => containerStates[state] || '未知'
export const containerName = (container: Container) => container.Names?.[0]?.replace(/^\//, '') || container.Id.slice(0, 12)
export const containerImage = (container: Pick<Container,'Image'|'ImageReference'>) => container.ImageReference || container.Image
export const containerStack = (container: Container) => container.Labels?.['com.docker.compose.project'] || container.Labels?.['com.docker.stack.namespace'] || ''
export const containerIPs = (container: Pick<Container, 'NetworkSettings'>) => [...new Set(Object.values(container.NetworkSettings?.Networks || {}).flatMap(network => [network.IPAddress, network.GlobalIPv6Address].filter((ip): ip is string => Boolean(ip))))]

export function canContainerAction(container: Container, action: ContainerAction): boolean {
  if (action === 'start') return ['created', 'exited'].includes(container.State)
  if (action === 'stop') return ['running', 'restarting'].includes(container.State)
  if (action === 'restart') return ['running', 'exited'].includes(container.State)
  if (action === 'kill') return ['running', 'paused', 'restarting'].includes(container.State)
  if (action === 'pause') return container.State === 'running'
  if (action === 'resume') return container.State === 'paused'
  return ['created', 'exited', 'dead'].includes(container.State)
}

export function filterContainers(containers: Container[], search: string, state: string): Container[] {
  const query = search.trim().toLowerCase()
  return containers.filter(container => (!state || container.State === state) && (!query || [
    container.Id, ...(container.Names || []), containerImage(container), container.Image, container.ImageID || '', containerStack(container), containerStateLabel(container.State),
    ...containerIPs(container), ...(container.Ports || []).map(port => `${port.PublicPort || ''}:${port.PrivatePort}/${port.Type}`),
  ].some(value => value.toLowerCase().includes(query))))
}

export function sortContainers(containers: Container[], key: ContainerSortKey, descending: boolean): Container[] {
  const value = (container: Container) => key === 'name' ? containerName(container) : key === 'stack' ? containerStack(container) : key === 'Image' ? containerImage(container) : container[key]
  return [...containers].sort((a, b) => {
    const left = value(a), right = value(b)
    return (typeof left === 'number' && typeof right === 'number' ? left - right : String(left).localeCompare(String(right), 'zh-CN', { numeric: true })) * (descending ? -1 : 1)
  })
}

export function containerMetrics(stats: ContainerStats) {
  const cpuDelta = (stats.cpu_stats?.cpu_usage?.total_usage || 0) - (stats.precpu_stats?.cpu_usage?.total_usage || 0)
  const systemDelta = (stats.cpu_stats?.system_cpu_usage || 0) - (stats.precpu_stats?.system_cpu_usage || 0)
  const cpus = stats.cpu_stats?.online_cpus || stats.cpu_stats?.cpu_usage?.percpu_usage?.length || 1
  const usage = stats.memory_stats?.usage || 0
  const cache = stats.memory_stats?.stats?.total_inactive_file ?? stats.memory_stats?.stats?.inactive_file ?? 0
  const memory = Math.max(0, usage - (cache < usage ? cache : 0))
  const limit = stats.memory_stats?.limit || 0
  const networks = Object.values(stats.networks || {})
  return {
    cpu: cpuDelta >= 0 && systemDelta > 0 ? cpuDelta / systemDelta * cpus * 100 : null,
    memory, limit, memoryPercent: limit > 0 ? Math.min(100, memory / limit * 100) : 0,
    received: networks.reduce((total, network) => total + (network.rx_bytes || 0), 0),
    sent: networks.reduce((total, network) => total + (network.tx_bytes || 0), 0),
    pids: stats.pids_stats?.current ?? null,
    cache: cache < usage ? cache : 0,
    read: (stats.blkio_stats?.io_service_bytes_recursive || []).filter(item => item.op.toLowerCase() === 'read').reduce((sum, item) => sum + item.value, 0),
    written: (stats.blkio_stats?.io_service_bytes_recursive || []).filter(item => item.op.toLowerCase() === 'write').reduce((sum, item) => sum + item.value, 0),
  }
}

export const containerActions: { key: ContainerAction; label: string; icon: 'play' | 'stop' | 'kill' | 'refresh' | 'pause' | 'trash' }[] = [
  { key: 'start', label: '启动', icon: 'play' }, { key: 'stop', label: '停止', icon: 'stop' },
  { key: 'kill', label: '强制停止', icon: 'kill' }, { key: 'restart', label: '重启', icon: 'refresh' },
  { key: 'pause', label: '暂停', icon: 'pause' }, { key: 'resume', label: '恢复', icon: 'play' },
  { key: 'remove', label: '删除', icon: 'trash' },
]
export type ContainerView = 'logs' | 'inspect' | 'stats' | 'exec' | 'attach'
export const containerViews: { key: ContainerView; label: string; icon: 'logs' | 'info' | 'chart' | 'terminal' | 'attach' }[] = [
  { key: 'logs', label: '日志', icon: 'logs' }, { key: 'inspect', label: '检查', icon: 'info' },
  { key: 'stats', label: '统计', icon: 'chart' }, { key: 'exec', label: '执行控制台', icon: 'terminal' },
  { key: 'attach', label: '附加控制台', icon: 'attach' },
]

export function publishedPorts(container: Container) {
  const seen = new Set<string>()
  return (container.Ports || []).filter(port => {
    if (!port.PublicPort) return false
    const key = `${port.PublicPort}:${port.PrivatePort}/${port.Type}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}
export function containerPortURL(port: NonNullable<Container['Ports']>[number], hostname: string) {
  const host = !port.IP || ['0.0.0.0', '::'].includes(port.IP) ? hostname : port.IP
  return `http://${host.includes(':') ? `[${host}]` : host}:${port.PublicPort}`
}
export function parseContainerCommand(value: string): string[] {
  const words: string[] = []
  let word = '', quote = '', escaping = false, started = false
  for (const character of value.trim()) {
    if (escaping) { word += character; escaping = false; started = true; continue }
    if (character === '\\' && quote !== "'") { escaping = true; started = true; continue }
    if (quote) { if (character === quote) quote = ''; else word += character; started = true; continue }
    if (character === '"' || character === "'") { quote = character; started = true; continue }
    if (/\s/.test(character)) { if (started) { words.push(word); word = ''; started = false }; continue }
    word += character; started = true
  }
  if (quote || escaping) throw new Error('命令中的引号或转义符不完整')
  if (started) words.push(word)
  if (!words[0]) throw new Error('请填写执行命令')
  return words
}

export interface ContainerListState { search: string; state: string; page: number; size: number; sort: ContainerSortKey; descending: boolean }
export function readContainerListState(query: Record<string, unknown>): ContainerListState {
  const size = Number(query.size), page = Number(query.page)
  return { search: typeof query.search === 'string' ? query.search : '', state: typeof query.state === 'string' && query.state in containerStates ? query.state : '', page: Number.isSafeInteger(page) && page > 0 ? page : 1, size: [10, 25, 50, 100].includes(size) ? size : 25, sort: ['name', 'State', 'stack', 'Image', 'Created'].includes(String(query.sort)) ? query.sort as ContainerSortKey : 'name', descending: query.order === 'desc' }
}
export function containerListQuery(state: ContainerListState) { return { search: state.search, state: state.state, page: String(state.page), size: String(state.size), sort: state.sort, order: state.descending ? 'desc' : 'asc' } }
