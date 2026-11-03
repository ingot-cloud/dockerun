import { isAlias, isMap, isScalar, isSeq, parseDocument, YAMLMap } from 'yaml'
import type { DockerNetwork } from '@/utils/networks'

export interface ComposeServiceSettings {
  name: string
  containerName: string
  networks: string[]
  networkMode: string
  editingReason: string
  networkEditingReason: string
}
export interface ComposeNetworkOption { value: string; label: string }
const managedNetworkMarker = 'x-dockerun-managed'
const hasMerge = (node: unknown) => isMap(node) && node.items.some(pair => isScalar(pair.key) && pair.key.source === '<<')

function composeDocument(source: string) {
  const doc = parseDocument(source, { merge: true })
  if (doc.errors.length) throw new Error(`Compose 语法错误：${doc.errors[0]!.message}`)
  if (!isMap(doc.contents) || !isMap(doc.get('services', true))) throw new Error('请在 Compose 中定义 services 服务')
  return doc
}

function networkKeys(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String)
  return value && typeof value === 'object' ? Object.keys(value) : []
}

export function readComposeServices(source: string): { services: ComposeServiceSettings[]; networks: ComposeNetworkOption[] } {
  const doc = composeDocument(source)
  const raw = doc.toJS({ maxAliasCount: 100 })
  const services = Object.entries(raw.services).map(([name, value]) => {
    const node = doc.getIn(['services', name], true)
    const service = value as Record<string, unknown> | null
    if (!service || typeof service !== 'object' || Array.isArray(service)) throw new Error(`服务「${name}」的配置必须是对象`)
    const editingReason = !isMap(node) || node.anchor || hasMerge(node) ? '此服务使用 YAML 锚点或合并，请在部署内容中编辑。' : ''
    const networkNode = isMap(node) ? node.get('networks', true) : undefined
    return {
      name, containerName: String(service.container_name || ''), networkMode: String(service.network_mode || ''),
      networks: networkKeys(service.networks).map(key => `compose:${key}`), editingReason,
      networkEditingReason: service.network_mode ? `已设置网络模式 ${service.network_mode}，请在部署内容中修改网络模式。`
        : isAlias(networkNode) || (isMap(networkNode) || isSeq(networkNode)) && networkNode.anchor || hasMerge(networkNode) ? '网络配置使用 YAML 锚点或合并，请在部署内容中编辑。' : '',
    }
  })
  if (!services.length) throw new Error('Compose 至少需要一个服务')
  const keys = new Set(['default', ...Object.keys(raw.networks || {}), ...services.flatMap(service => service.networks.map(value => value.slice(8)))])
  const networks = [...keys].map(key => {
    const network = raw.networks?.[key]
    return { value: `compose:${key}`, label: network?.[managedNetworkMarker] === true ? `${network.name}（外部网络）`
      : key === 'default' && !network ? '应用默认网络（自动创建）' : `${key}${network?.name ? ` → ${network.name}` : ''}${network?.external ? '（外部网络）' : ''}` }
  })
  return { services, networks }
}

function editableService(source: string, name: string, networks = false) {
  const doc = composeDocument(source)
  const service = readComposeServices(source).services.find(service => service.name === name)
  if (!service) throw new Error('服务已变化，请重新选择服务')
  if (service.editingReason || networks && service.networkEditingReason) throw new Error(service.editingReason || service.networkEditingReason)
  const node = doc.getIn(['services', name], true)
  if (!isMap(node)) throw new Error('请在部署内容中编辑此服务')
  return { doc, node, service }
}

export function setComposeContainerName(source: string, serviceName: string, name: string): string {
  const { doc, node, service } = editableService(source, serviceName)
  if (name === service.containerName) return source
  if (name) node.set('container_name', name)
  else node.delete('container_name')
  return doc.toString({ lineWidth: 0 })
}

export function setComposeServiceNetworks(source: string, serviceName: string, selected: string[]): string {
  const { doc, node, service } = editableService(source, serviceName, true)
  if (JSON.stringify(selected) === JSON.stringify(service.networks)) return source
  let definitions = doc.get('networks', true)
  if (isScalar(definitions) && definitions.value == null) definitions = undefined
  if (definitions != null && (!isMap(definitions) || definitions.anchor || hasMerge(definitions))) throw new Error('顶层网络配置使用了特殊 YAML 结构，请在部署内容中编辑。')
  const keys: string[] = []
  for (const value of selected) {
    if (value.startsWith('compose:')) {
      const key = value.slice(8)
      if (!readComposeServices(source).networks.some(network => network.value === value)) throw new Error('所选 Compose 网络已变化，请重新选择')
      keys.push(key)
      continue
    }
    const name = value.startsWith('external:') ? value.slice(9) : value
    if (!name || /[\s\x00]/.test(name)) throw new Error('请填写有效的外部网络名称')
    if (!isMap(definitions)) { doc.set('networks', doc.createNode<Record<string, unknown>>({})); definitions = doc.get('networks', true) }
    if (!isMap(definitions)) throw new Error('无法编辑网络配置')
    let key = definitions.items.find(pair => {
      const config = pair.value
      return isMap(config) && config.get('external') === true && String(config.get('name') || pair.key) === name
    })?.key
    if (key === undefined) {
      const base = `dockerun_external_${name.replace(/[^A-Za-z0-9_-]/g, '_')}`
      let available = base, index = 2
      while (definitions.has(available)) available = `${base}_${index++}`
      definitions.set(available, doc.createNode({ name, external: true, [managedNetworkMarker]: true }))
      key = available
    }
    keys.push(String(key))
  }
  const unique = [...new Set(keys)]
  if (!unique.length) node.delete('networks')
  else {
    const previous = node.get('networks', true)
    if (isMap(previous)) {
      const next = new YAMLMap<string, unknown>(doc.schema)
      for (const key of unique) next.set(key, previous.has(key) ? previous.get(key, true) : null)
      node.set('networks', next)
    } else node.set('networks', doc.createNode(unique))
  }
  // 只清理界面生成且已无服务引用的声明，保留用户原有网络和连接属性。
  if (isMap(definitions)) {
    const raw = doc.toJS({ maxAliasCount: 100 })
    const referenced = new Set(Object.values(raw.services).flatMap((service: any) => service.networks == null && !service.network_mode ? ['default'] : networkKeys(service.networks)))
    for (const pair of [...definitions.items]) {
      if (isScalar(pair.key) && isMap(pair.value) && pair.value.get(managedNetworkMarker) === true && !referenced.has(String(pair.key.value))) definitions.delete(pair.key.value)
    }
    if (!definitions.items.length) doc.delete('networks')
  }
  return doc.toString({ lineWidth: 0 })
}

export function restoreComposeFormatting(updated: string, original: string): string {
  const normalize = (value: any): any => Array.isArray(value) ? value.map(normalize)
    : value && typeof value === 'object' ? Object.fromEntries(Object.entries(value).sort(([a], [b]) => a.localeCompare(b)).map(([key, item]) => [key, normalize(item)])) : value
  const snapshot = (source: string) => JSON.stringify(normalize(composeDocument(source).toJS({ maxAliasCount: 100 })))
  return snapshot(updated) === snapshot(original) ? original : updated
}

export function composeNetworkUnavailableReason(network: DockerNetwork): string {
  if (network.Ingress) return '入口网络不能连接普通应用容器'
  if (['bridge', 'host', 'none'].includes(network.Name) || ['host', 'null'].includes(network.Driver)) return '系统网络请在 Compose 中通过 network_mode 配置'
  if (network.Scope === 'swarm' && !network.Attachable) return '此 Swarm 网络未允许独立容器连接'
  return ''
}
