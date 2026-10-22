export interface NetworkIPAMConfig {
  Subnet?: string
  Gateway?: string
  IPRange?: string
  AuxiliaryAddresses?: Record<string, string> | null
}

export interface DockerNetwork {
  Id: string
  Name: string
  Driver: string
  Scope?: string
  Created?: string
  Internal?: boolean
  Attachable?: boolean
  Ingress?: boolean
  EnableIPv6?: boolean
  IPAM?: { Driver?: string; Options?: Record<string, string> | null; Config?: NetworkIPAMConfig[] | null } | null
  Labels?: Record<string, string> | null
  Options?: Record<string, string> | null
  Containers?: Record<string, { Name: string; EndpointID?: string; MacAddress?: string; IPv4Address?: string; IPv6Address?: string }> | null
}

export const networkColumns = [
  { key: 'name', label: '名称' }, { key: 'stack', label: '应用栈' }, { key: 'driver', label: '驱动' },
  { key: 'attachable', label: '可手动连接' }, { key: 'ipam', label: 'IPAM 驱动' },
  { key: 'ipv4Subnet', label: 'IPv4 子网' }, { key: 'ipv4Gateway', label: 'IPv4 网关' },
  { key: 'ipv6Subnet', label: 'IPv6 子网' }, { key: 'ipv6Gateway', label: 'IPv6 网关' },
] as const
export type NetworkSortKey = typeof networkColumns[number]['key']
export const isSystemNetwork = (network: DockerNetwork) => ['bridge', 'host', 'none'].includes(network.Name)
export const canRemoveNetwork = (network: DockerNetwork) => !isSystemNetwork(network) && !network.Ingress
export const networkStack = (network: DockerNetwork) => network.Labels?.['com.docker.stack.namespace'] || network.Labels?.['com.docker.compose.project'] || ''
export const networkConfigs = (network: DockerNetwork, family: 4 | 6) => (network.IPAM?.Config || []).filter(config => Boolean(config.Subnet) && config.Subnet!.includes(':') === (family === 6))
export const networkAddresses = (network: DockerNetwork, family: 4 | 6, field: 'Subnet' | 'Gateway') => networkConfigs(network, family).map(config => config[field]).filter(Boolean).join(', ')

function columnValue(network: DockerNetwork, key: NetworkSortKey): string {
  if (key === 'name') return network.Name
  if (key === 'stack') return networkStack(network)
  if (key === 'driver') return network.Driver
  if (key === 'attachable') return String(Boolean(network.Attachable))
  if (key === 'ipam') return network.IPAM?.Driver || 'default'
  return networkAddresses(network, key.startsWith('ipv6') ? 6 : 4, key.endsWith('Gateway') ? 'Gateway' : 'Subnet')
}
export function filterNetworks(networks: DockerNetwork[], search: string, driver = '', type = '', attachable = '') {
  const query = search.trim().toLowerCase()
  return networks.filter(network => (!driver || network.Driver === driver)
    && (!type || (isSystemNetwork(network) ? 'system' : 'custom') === type)
    && (!attachable || String(Boolean(network.Attachable)) === attachable)
    && (!query || [network.Id, network.Name, network.Driver, network.Scope, networkStack(network),
      ...Object.entries(network.Labels || {}).flat(), ...(network.IPAM?.Config || []).flatMap(config => [config.Subnet, config.Gateway, config.IPRange]),
    ].some(value => value?.toLowerCase().includes(query))))
}
export function sortNetworks(networks: DockerNetwork[], key: NetworkSortKey, descending: boolean) {
  return [...networks].sort((a, b) => (columnValue(a, key).localeCompare(columnValue(b, key), undefined, { numeric: true, sensitivity: 'base' }) || a.Id.localeCompare(b.Id)) * (descending ? -1 : 1))
}

export interface NetworkListState { search: string; driver: string; type: string; attachable: string; page: number; size: number; sort: NetworkSortKey; descending: boolean }
export function readNetworkListState(query: Record<string, unknown>): NetworkListState {
  const value = (key: string) => typeof query[key] === 'string' ? query[key] as string : ''
  const page = Number(value('page'))
  return {
    search: value('search'), driver: value('driver'), type: ['system', 'custom'].includes(value('type')) ? value('type') : '',
    attachable: ['true', 'false'].includes(value('attachable')) ? value('attachable') : '',
    page: Number.isSafeInteger(page) && page > 0 ? page : 1,
    size: [10, 25, 50, 100].includes(Number(value('size'))) ? Number(value('size')) : 10,
    sort: networkColumns.some(column => column.key === value('sort')) ? value('sort') as NetworkSortKey : 'name', descending: value('direction') === 'desc',
  }
}
export function networkListQuery(state: NetworkListState) {
  return { search: state.search || undefined, driver: state.driver || undefined, type: state.type || undefined, attachable: state.attachable || undefined, page: String(state.page), size: String(state.size), sort: state.sort, direction: state.descending ? 'desc' : 'asc' }
}

export interface NetworkPair { key: string; value: string }
export interface NetworkAddressForm { subnet: string; gateway: string; range: string; excluded: NetworkPair[] }
export interface NetworkForm { name: string; driver: string; options: NetworkPair[]; labels: NetworkPair[]; internal: boolean; attachable: boolean; enableIPv6: boolean; ipv4: NetworkAddressForm; ipv6: NetworkAddressForm }
export interface NetworkCreateRequest {
  name: string; driver: string; options: Record<string, string>; labels: Record<string, string>
  internal: boolean; attachable: boolean; enable_ipv6: boolean
  ipam?: { Driver: string; Config: NetworkIPAMConfig[] }
}
export function emptyNetworkForm(): NetworkForm {
  return { name: '', driver: 'bridge', options: [], labels: [], internal: false, attachable: false, enableIPv6: false,
    ipv4: { subnet: '', gateway: '', range: '', excluded: [] }, ipv6: { subnet: '', gateway: '', range: '', excluded: [] } }
}
function pairsToRecord(pairs: NetworkPair[], label: string): Record<string, string> {
  const entries: [string, string][] = []
  const keys = new Set<string>()
  for (const pair of pairs) {
    const key = pair.key.trim()
    if (!key) throw new Error(`请填写${label}的名称，或移除空白行`)
    if (keys.has(key)) throw new Error(`${label}名称「${key}」重复`)
    keys.add(key)
    entries.push([key, pair.value.trim()])
  }
  return Object.fromEntries(entries)
}

function parseIP(value: string, family: 4 | 6): bigint | null {
  if (family === 4) {
    const parts = value.split('.')
    if (parts.length !== 4 || parts.some(part => !/^(0|[1-9]\d{0,2})$/.test(part) || Number(part) > 255)) return null
    return parts.reduce((address, part) => (address << 8n) + BigInt(part), 0n)
  }
  // URL's host parser validates and normalizes compressed IPv6, including embedded IPv4.
  try {
    if (!value.includes(':') || /[\[\]\s/%]/.test(value)) return null
    const normalized = new URL(`http://[${value}]/`).hostname.slice(1, -1)
    const halves = normalized.split('::')
    const left = halves[0] ? halves[0].split(':') : []
    const right = halves[1] ? halves[1].split(':') : []
    const parts = halves.length === 2 ? [...left, ...Array(8 - left.length - right.length).fill('0'), ...right] : left
    return parts.reduce((address, part) => (address << 16n) + BigInt(`0x${part}`), 0n)
  } catch { return null }
}
function parseCIDR(value: string, family: 4 | 6) {
  const [ip, prefix, extra] = value.split('/')
  const address = parseIP(ip, family)
  if (extra !== undefined || prefix === undefined || !/^\d{1,3}$/.test(prefix) || Number(prefix) > (family === 4 ? 32 : 128) || address === null) return null
  return { address, prefix: Number(prefix), shift: BigInt((family === 4 ? 32 : 128) - Number(prefix)) }
}
function addressConfig(form: NetworkAddressForm, family: 4 | 6): NetworkIPAMConfig | undefined {
  const subnet = form.subnet.trim(), gateway = form.gateway.trim(), range = form.range.trim()
  const auxiliary = pairsToRecord(form.excluded, `IPv${family} 排除 IP`)
  if (!subnet && !gateway && !range && !Object.keys(auxiliary).length) return undefined
  const cidr = parseCIDR(subnet, family)
  if (!cidr) throw new Error(`请填写有效的 IPv${family} 子网（CIDR 格式）`)
  if ((cidr.address >> cidr.shift) << cidr.shift !== cidr.address) throw new Error(`IPv${family} 子网必须是网络地址，例如 ${family === 4 ? '172.20.0.0/16' : 'fd00::/64'}`)
  const contains = (address: bigint) => address >> cidr.shift === cidr.address >> cidr.shift
  if (gateway) {
    const address = parseIP(gateway, family)
    if (address === null || !contains(address)) throw new Error(`IPv${family} 网关必须是子网内的有效 IP 地址`)
  }
  if (range) {
    const pool = parseCIDR(range, family)
    if (!pool || pool.prefix < cidr.prefix || !contains(pool.address) || (pool.address >> pool.shift) << pool.shift !== pool.address) throw new Error(`IPv${family} IP 范围必须是子网内的有效 CIDR 网络地址`)
  }
  for (const [name, ip] of Object.entries(auxiliary)) {
    const address = parseIP(ip, family)
    if (address === null || !contains(address)) throw new Error(`IPv${family} 排除 IP「${name}」必须是子网内的有效 IP 地址`)
  }
  return { Subnet: subnet, ...(gateway ? { Gateway: gateway } : {}), ...(range ? { IPRange: range } : {}), ...(Object.keys(auxiliary).length ? { AuxiliaryAddresses: auxiliary } : {}) }
}
export function buildNetworkRequest(form: NetworkForm): NetworkCreateRequest {
  const name = form.name.trim()
  if (!name || !/^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/.test(name)) throw new Error('网络名称需以字母或数字开头，且只包含字母、数字、下划线、点或连字符')
  if (['bridge', 'host', 'none', 'ingress'].includes(name)) throw new Error('不能使用系统保留的网络名称')
  if (!['bridge', 'overlay', 'macvlan', 'ipvlan'].includes(form.driver)) throw new Error('请选择有效的网络驱动')
  const options = pairsToRecord(form.options, '驱动选项')
  if (['macvlan', 'ipvlan'].includes(form.driver) && !options.parent) throw new Error(`${form.driver} 驱动需要添加 parent 选项并填写父网卡，例如 eth0`)
  const configs = [addressConfig(form.ipv4, 4), ...(form.enableIPv6 ? [addressConfig(form.ipv6, 6)] : [])].filter((value): value is NetworkIPAMConfig => Boolean(value))
  return { name, driver: form.driver, options, labels: pairsToRecord(form.labels, '标签'), internal: form.internal, attachable: form.attachable, enable_ipv6: form.enableIPv6,
    ...(configs.length ? { ipam: { Driver: 'default', Config: configs } } : {}) }
}
