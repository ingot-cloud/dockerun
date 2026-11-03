import { api } from './workspace'
import request from './request'
import type { DeploymentScripts, DeploymentExecution } from '@/utils/releases'
export interface TemplateParameter { key: string; label: string; type: string; default: string; required: boolean; help: string }
export interface ApplicationDefinition { source?: string; registry_id?: number; platform?: string; execution?: DeploymentExecution; compose: string; configs: Record<string,string>; parameters: TemplateParameter[]; stateful: boolean; topology: string; storage: string; initialization: string; instructions: string; endpoints: Array<{name:string;port:string;protocol:string;path:string}> }
export interface ApplicationTemplate { id: string; key: string; name: string; version: string; category: string; description: string; source: string; definition: ApplicationDefinition; scripts: DeploymentScripts }
export interface TemplateImportEntry { index:number; key:string; name:string; version:string; description:string; category:string; status:'new'|'new_version'|'identical'|'conflict'; message:string; parameter_count:number; config_count:number; script_stages:string[]; files:Record<string,string> }
export interface TemplateImportPreview { id:string; expires_at:string; entries:TemplateImportEntry[] }
export interface TemplateImportChoice { index:number; action:'import'|'skip'|'rename'; key?:string; name?:string }
export interface TemplateImportResult { imported:number; skipped:number; ids:string[] }
export const applicationTemplatesApi = {
  list: (signal?:AbortSignal) => api<ApplicationTemplate[]>('GET','/managed-templates',undefined,signal),
  create: (id:string,data:unknown) => api('POST',`/managed-templates/${encodeURIComponent(id)}/applications`,data),
  save: (id:number|string,data:unknown) => api('POST',`/applications/${id}/template`,data),
  preview: async (file:File,progress:(percent:number)=>void):Promise<TemplateImportPreview> => {
    const data=new FormData();data.append('file',file)
    return await request.post('/managed-templates/imports',data,{silentError:true,timeout:90000,onUploadProgress:event=>{if(event.total)progress(Math.min(100,Math.round(event.loaded/event.total*100)))}}) as unknown as TemplateImportPreview
  },
  confirm: (id:string,items:TemplateImportChoice[]) => api<TemplateImportResult>('POST',`/managed-templates/imports/${encodeURIComponent(id)}/commit`,{items}),
  cancel: (id:string) => api('DELETE',`/managed-templates/imports/${encodeURIComponent(id)}`),
  export: async (ids:string[]):Promise<Blob> => {
    try { return await request.post('/managed-templates/export',{ids},{silentError:true,responseType:'blob',timeout:90000}) as unknown as Blob }
    catch(cause:any){
      if(cause?.response?.data instanceof Blob){let body:any;try{body=JSON.parse(await cause.response.data.text())}catch{};throw new Error(body?.error || '模板下载失败，请重试')}
      throw cause
    }
  },
}
