import type { TemplateParameter } from '@/api/applicationTemplates'

export type ApplicationDetailTab = 'environments' | 'releases' | 'configuration'
export function applicationDetailTab(value: unknown): ApplicationDetailTab {
  return value === 'releases' || value === 'configuration' ? value : 'environments'
}
export function applicationSetupStep(environments: {current_release_id?: number}[], targets: {current_release_id?: number}[] = []): number {
  if ([...environments, ...targets].some(item => item.current_release_id)) return 3
  return environments.length ? 2 : 1
}

export function environmentDeletionReason(environment: {deletion_blocked_reason?: string}): string {
  return typeof environment.deletion_blocked_reason === 'string' ? environment.deletion_blocked_reason : '环境任务状态尚未确认，请刷新后重试'
}

export function environmentTargetConfiguration(targets: number[], singleHost = false): {target_id: number; target_ids: string} {
  if (!targets.length) throw new Error('请选择部署目标')
  if (targets.some(id => !Number.isSafeInteger(id) || id <= 0)) throw new Error('请选择有效的部署目标')
  if (new Set(targets).size !== targets.length) throw new Error('部署目标不能重复')
  if (singleHost && targets.length > 1) throw new Error('此应用仅支持单台目标主机')
  return { target_id: targets[0]!, target_ids: JSON.stringify(targets.length > 1 ? targets : []) }
}

export interface EnvironmentParameterPair { id: number; key: string; value: string; schemaKey?: string }

export function environmentParameterPairs(parameters: Record<string, string> = {}, schema: TemplateParameter[] = []): EnvironmentParameterPair[] {
  const keys = new Set(schema.map(field => field.key))
  const derivedPort = keys.has('NACOS_PORT')
  const fields = schema.map(field => ({ key: field.key, value: Object.prototype.hasOwnProperty.call(parameters, field.key) ? parameters[field.key]! : '', schemaKey: field.key }))
  const extra = Object.entries(parameters).filter(([key]) => !keys.has(key) && !(derivedPort && key === 'NACOS_GRPC_PORT')).map(([key, value]) => ({ key, value }))
  return [...fields, ...extra].map((pair, index) => ({ ...pair, id: index + 1 }))
}

export function environmentParameterValues(pairs: EnvironmentParameterPair[], schema: TemplateParameter[] = []): Record<string, string> {
  const keys = new Set<string>(), entries: [string, string][] = []
  pairs.forEach((pair, index) => {
    const key = pair.key.trim()
    if (!key && pair.value === '') return
    if (!key) throw new Error(`请填写第 ${index + 1} 项运行参数的名称`)
    if (pair.schemaKey && pair.key !== pair.schemaKey) throw new Error('模板参数名称由应用定义固定，不能修改')
    if (key === 'NACOS_GRPC_PORT' && schema.some(field => field.key === 'NACOS_PORT') && pair.schemaKey !== key) throw new Error('gRPC 客户端端口由服务端口自动计算，请修改服务端口')
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) throw new Error(`运行参数名称「${key}」无效，请以字母或下划线开头，仅使用字母、数字和下划线`)
    if (keys.has(key)) throw new Error(`运行参数名称「${key}」重复`)
    keys.add(key)
    entries.push([key, pair.value])
  })
  return Object.fromEntries(entries)
}

export function isSensitiveEnvironmentParameter(key: string): boolean {
  return /password|passwd|secret|token|credential|private_key|api_key|access_key/i.test(key)
}

export function readOnlyEnvironmentParameters(parameters: Record<string,string> = {}, schema: TemplateParameter[] = []): Record<string,string> {
  const passwords=new Set(schema.filter(field=>field.type==='password').map(field=>field.key))
  return Object.fromEntries(Object.entries(parameters).map(([key,value])=>[key,passwords.has(key) || isSensitiveEnvironmentParameter(key)?'******':value]))
}

export function applicationRuntimeImages(environments:any[]=[],states:any[]=[],hosts:any[]=[]){
  return states.flatMap(state=>{
    const env=environments.find(item=>item.id===state.environment_id)
    if(!env)return []
    return (Array.isArray(state.containers)?state.containers:[]).map((container:any)=>({
      ...container,environment_id:env.id,environment_name:env.name,target_id:state.target_id,
      target_name:hosts.find(item=>item.id===state.target_id)?.name || (state.target_id<=1?'本机':`主机 #${state.target_id}`),
    }))
  })
}

export function runtimeImageLocation(container: {target_id?:number;image_id?:string}) {
  // 现有镜像详情只查询本机 Docker，不能用远程节点的 Image ID 打开本机详情。
  if (container.target_id !== 1 || !container.image_id?.trim()) return undefined
  return { name: 'ImageDetail', params: { id: container.image_id } }
}
