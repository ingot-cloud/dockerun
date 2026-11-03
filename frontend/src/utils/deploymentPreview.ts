import { isMap, parseDocument, stringify } from 'yaml'

export interface DeploymentPreviewImage { reference: string; original?: string; digest?: string; id?: string }
export interface DeploymentServicePreview {
  name: string; containerName: string; image: string; digest: string; imageID: string
  ports: string[]; networks: string[]; mounts: string[]; configs: string[]
  restart: string; healthcheck: string; environmentCount: number
}

const record = (value: unknown): Record<string, any> => value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, any> : {}
const names = (value: unknown): string[] => Array.isArray(value) ? value.map(String) : Object.keys(record(value))
const duration = (value: unknown) => typeof value === 'number' ? `${value / 1e9} 秒` : String(value)

function portLabel(value: any): string {
  if (typeof value === 'string' || typeof value === 'number') {
    const parts = String(value).split(':')
    return parts.length > 1 ? `${parts.slice(0, -1).join(':')} → ${parts[parts.length - 1]}` : `容器端口 ${value}`
  }
  const port = record(value), protocol = String(port.protocol || 'tcp').toUpperCase()
  return port.published ? `${port.host_ip ? `${port.host_ip}:` : ''}${port.published} → ${port.target}/${protocol}` : `容器端口 ${port.target}/${protocol}`
}

export function deploymentComposePreview(content: string, images: DeploymentPreviewImage[] | null = []): { yaml: string; services: DeploymentServicePreview[] } {
  const doc = parseDocument(content, { merge: true })
  if (doc.errors.length || !isMap(doc.contents)) throw new Error('部署配置格式无效，无法生成服务概览')
  const raw = record(doc.toJS({ maxAliasCount: 100 }))
  if (!Object.keys(record(raw.services)).length) throw new Error('部署配置中没有可展示的服务')
  const services = Object.entries(record(raw.services)).map(([name, value]) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`服务「${name}」配置无效，无法生成概览`)
    const service = record(value)
    const reference = String(service.image || '')
    const matches = (images || []).filter(image => image.reference === reference || image.original === reference || image.id === reference)
    const image = matches.find(image => image.reference === reference || image.original === reference) || (matches.length === 1 ? matches[0] : undefined)
    const imageName = image?.reference || reference || '未设置'
    const digest = image?.digest || (reference.includes('@') ? reference.split('@').slice(-1)[0] || '' : '')
    const networks = service.network_mode ? [`网络模式：${service.network_mode}`] : (names(service.networks).length ? names(service.networks) : ['default']).map(key => {
      const network = record(raw.networks?.[key])
      return `${network.name || (key === 'default' ? '应用默认网络' : key)}${network.external ? '（外部网络）' : ''}`
    })
    const mounts = (Array.isArray(service.volumes) ? service.volumes : []).map((value: any) => {
      if (typeof value === 'string') return value.replace(/:/, ' → ')
      const mount = record(value), source = record(raw.volumes?.[mount.source]).name || mount.source || '匿名卷'
      return `${source} → ${mount.target}${mount.read_only ? '（只读）' : ''}`
    })
    const configs = (Array.isArray(service.configs) ? service.configs : []).map((value: any) => typeof value === 'string' ? value : `${value.source}${value.target ? ` → ${value.target}` : ''}`)
    const health = record(service.healthcheck)
    const healthcheck = health.disable || health.test?.[0] === 'NONE' ? '已停用' : Object.keys(health).length ? ['已配置', health.interval ? `间隔 ${duration(health.interval)}` : '', health.timeout ? `超时 ${duration(health.timeout)}` : '', health.retries != null ? `重试 ${health.retries} 次` : ''].filter(Boolean).join(' · ') : '未配置'
    const policy = service.restart || service.deploy?.restart_policy?.condition
    const restartLabels: Record<string, string> = {no:'不自动重启',always:'始终重启','on-failure':'失败时重启','unless-stopped':'除非手动停止，否则重启',none:'不自动重启',any:'始终重启'}
    return { name, containerName: String(service.container_name || '由 Compose 自动生成'), image: imageName, digest,
      imageID: image?.id || (reference.startsWith('sha256:') ? reference : ''), ports: (Array.isArray(service.ports) ? service.ports : []).map(portLabel),
      networks, mounts, configs, restart: restartLabels[policy] || policy || '未设置', healthcheck,
      environmentCount: Array.isArray(service.environment) ? service.environment.length : Object.keys(record(service.environment)).length,
    }
  })
  // 保留所有解析字段和值，只调整展示格式；不改写版本或执行文件。
  return { yaml: stringify(raw, { indent: 2, lineWidth: 0 }), services }
}
