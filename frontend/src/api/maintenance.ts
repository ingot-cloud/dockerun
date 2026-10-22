import request from './request'

export interface RegistryMaintenanceStatus {
  enabled: boolean
  read_only: boolean
  manual_read_only: boolean
  operation: string
  public_url: string
  engine: string
  storage_bytes: number | null
  storage_error?: string
  tls: boolean
}
export type MaintenanceKind = 'backup' | 'registry_gc'
export interface MaintenanceJob {
  id: number
  kind: MaintenanceKind
  status: string
  file: string
  error: string
  user_id: number
  created_at: string
  updated_at: string
}
export interface AuditEvent {
  id: number
  user_id: number
  action: string
  resource: string
  created_at: string
}
export interface MaintenanceSubmission { job_id: number; reused?: boolean; message: string }
export const maintenanceApi = {
  status: (signal?: AbortSignal): Promise<RegistryMaintenanceStatus> => request.get('/builtin/status', { signal, silentError: true }),
  setReadOnly: (value: boolean): Promise<{ read_only: boolean; manual_read_only: boolean; operation: string }> => request.post('/builtin/maintenance', { read_only: value }, { silentError: true }),
  collect: (): Promise<MaintenanceSubmission> => request.post('/builtin/maintenance', { gc: true }, { silentError: true }),
  backup: (): Promise<MaintenanceSubmission> => request.post('/system/backups', undefined, { silentError: true }),
  backups: (signal?: AbortSignal): Promise<MaintenanceJob[]> => request.get('/system/backups', { signal, silentError: true }),
  collections: (signal?: AbortSignal): Promise<MaintenanceJob[]> => request.get('/builtin/maintenance', { signal, silentError: true }),
  audit: (signal?: AbortSignal): Promise<AuditEvent[]> => request.get('/audit', { signal, silentError: true }),
  download: async (id: number, signal?: AbortSignal): Promise<Blob> => {
    try { return await request.get(`/system/backups/${id}/download`, { signal, silentError: true, responseType: 'blob', timeout: 0 }) as unknown as Blob }
    catch (error) {
      const data = (error as { response?: { data?: unknown } }).response?.data
      if (data instanceof Blob) {
        const text = await data.text()
        try { const parsed = JSON.parse(text); if (typeof parsed.error === 'string') throw new Error(parsed.error) }
        catch (parsedError) { if (parsedError instanceof Error && !(parsedError instanceof SyntaxError)) throw parsedError }
      }
      throw error
    }
  },
}
