import { api } from './workspace'
import type { DeploymentScripts, DeploymentExecution } from '@/utils/releases'
export interface TemplateParameter { key: string; label: string; type: string; default: string; required: boolean; help: string }
export interface ApplicationDefinition { source?: string; registry_id?: number; platform?: string; execution?: DeploymentExecution; compose: string; configs: Record<string,string>; parameters: TemplateParameter[]; stateful: boolean; topology: string; storage: string; initialization: string; instructions: string; endpoints: Array<{name:string;port:string;protocol:string;path:string}> }
export interface ApplicationTemplate { id: string; name: string; version: string; category: string; description: string; source: string; definition: ApplicationDefinition; scripts: DeploymentScripts }
export const applicationTemplatesApi = {
  list: (signal?:AbortSignal) => api<ApplicationTemplate[]>('GET','/managed-templates',undefined,signal),
  create: (id:string,data:unknown) => api('POST',`/managed-templates/${encodeURIComponent(id)}/applications`,data),
  save: (id:number|string,data:unknown) => api('POST',`/applications/${id}/template`,data),
}
