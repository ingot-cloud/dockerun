import request from './request'

export interface BuiltinImageEvent {
  id: number
  repository: string
  reference: string
  digest: string
  user_id: number
  created_at: string
  imported: boolean
}
export interface BuiltinCatalog {
  catalog: { repositories: string[] }
  events: BuiltinImageEvent[]
  public_url: string
  read_only: boolean
  permission: string
}
export interface BuiltinManifest {
  digest: string
  release_ids: number[]
  manifest: {
    schemaVersion?: number
    mediaType?: string
    config?: { size?: number; digest?: string }
    layers?: { size?: number; digest?: string; mediaType?: string }[]
    manifests?: { digest: string; size?: number; platform?: { os?: string; architecture?: string; variant?: string } }[]
  }
}
export const builtinApi = {
  catalog: (signal?: AbortSignal): Promise<BuiltinCatalog> => request.get('/builtin/catalog', { signal }),
  tags: (name: string, signal?: AbortSignal): Promise<{ name: string; tags: string[] | null }> => request.get('/builtin/tags', { params: { name }, signal }),
  manifest: (name: string, reference: string, signal?: AbortSignal): Promise<BuiltinManifest> => request.get('/builtin/tags', { params: { name, reference }, signal }),
  remove: (name: string, digest: string): Promise<{ message: string }> => request.delete('/builtin/manifests', { data: { name, digest } }),
}
