import type { ApplicationDefinition, ApplicationTemplate, TemplateParameter } from '@/api/applicationTemplates'
import { registryHost, type ReleaseRegistry } from '@/utils/releases'
export function templateCategory(value?:string) { return value === 'middleware' ? '中间件' : '业务应用' }
export function templateValues(template: ApplicationTemplate, registry?: ReleaseRegistry): Record<string,string> {
  const host=registryHost(registry)
  return Object.fromEntries((template.definition.parameters || []).map(field=>{
    let value=field.type==='password'?'':field.default || ''
    if(field.type==='image' && value && host) {const first=value.split('/')[0] || '';if(!value.includes('/') || !first.includes('.') && !first.includes(':') && first!=='localhost')value=`${host}/${value}`}
    return [field.key,value]
  }))
}
export function validateTemplateValues(schema:TemplateParameter[],values:Record<string,string>) {
  for(const field of schema) {
    const value=values[field.key] || ''
    if(field.required && !value.trim())throw new Error(`请填写${field.label}`)
    if(field.type==='port' && value && (!/^\d+$/.test(value) || Number(value)<1 || Number(value)>65535))throw new Error(`${field.label}必须为 1–65535 的端口`)
    if(field.type==='path' && value && !value.startsWith('/'))throw new Error(`${field.label}必须是目标主机的绝对路径`)
  }
  if(values.NACOS_AUTH_TOKEN) {try{if(atob(values.NACOS_AUTH_TOKEN).length<32)throw new Error()}catch{throw new Error('Nacos 认证令牌必须为 Base64 编码且解码后至少 32 字节')}}
  if(values.NACOS_PORT && Number(values.NACOS_PORT)>64535)throw new Error('Nacos 服务端口不能超过 64535，需预留端口 +1000 用于 gRPC')
}
export function maskTemplateParameters(values:Record<string,string>,schema:TemplateParameter[] = []) {
  const keys=new Set(schema.filter(item=>item.type==='password').map(item=>item.key))
  return Object.fromEntries(Object.entries(values).map(([key,value])=>[key,keys.has(key)||/password|passwd|secret|token|credential|private_key/i.test(key)?'******':value]))
}
export function templateEndpoints(definition:ApplicationDefinition,parameters:Record<string,string>,host:string) {
  return (definition.endpoints || []).filter(item=>parameters[item.port]).map(item=>({name:item.name,address:`${item.protocol}://${host}:${parameters[item.port]}${item.path || ''}`}))
}
export function retargetTemplateImages(schema:TemplateParameter[],values:Record<string,string>,oldRegistry?:ReleaseRegistry,newRegistry?:ReleaseRegistry) {
  const previous=registryHost(oldRegistry),next=registryHost(newRegistry),copy={...values}
  if(previous && next)for(const field of schema)if(field.type==='image' && copy[field.key]?.startsWith(`${previous}/`))copy[field.key]=`${next}/${copy[field.key]!.slice(previous.length+1)}`
  return copy
}
