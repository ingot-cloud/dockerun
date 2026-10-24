export interface DeploymentScripts { before: string; deploy: string; after: string }
export interface ReleaseRegistry { id: number; name: string; url: string; kind?: string; is_default?: boolean }
export const emptyScripts = (): DeploymentScripts => ({ before: '', deploy: '', after: '' })
export function copyScripts(value?: Partial<DeploymentScripts> | null): DeploymentScripts {
  return { before: value?.before || '', deploy: value?.deploy || '', after: value?.after || '' }
}
export function hasScripts(value?: Partial<DeploymentScripts>): boolean {
  return !!(value?.before?.trim() || value?.deploy?.trim() || value?.after?.trim())
}
export function defaultReleaseRegistry(registries: ReleaseRegistry[]): number {
  return registries.find(item => item.kind === 'builtin')?.id || registries.find(item => item.is_default)?.id || registries[0]?.id || 0
}
export function registryHost(registry?: ReleaseRegistry): string {
  if (!registry?.url) return ''
  try { return new URL(registry.url.includes('://') ? registry.url : `https://${registry.url}`).host } catch { return '' }
}
export function imageRepository(reference: string): string {
  const name = reference.split('@')[0]!.replace(/:[^/]*$/, '')
  const first = name.split('/')[0] || ''
  return first.includes('.') || first.includes(':') || first === 'localhost' ? name.slice(first.length + 1) : name
}
export function releaseImages(release?: { images?: string }): Array<{ reference: string }> {
  try { const images = JSON.parse(release?.images || '[]'); return Array.isArray(images) ? images : [] } catch { return [] }
}
export function automationIssue(environment: any, releases: any[]): string {
  const base = releases.find(item => item.id === environment?.current_release_id)
  if (!base) return '请先在此环境完成首次手动发布，再开启自动发布。'
  if (base.source !== 'registry' || base.registry_id !== environment.registry_id) return '请先使用所选仓库在此环境手动发布一次，再开启自动发布。'
  if (!environment.repository?.trim()) return '请填写当前版本中需要监听的镜像名称。'
  if (!releaseImages(base).some(image => imageRepository(image.reference) === environment.repository.trim())) return '镜像名称必须包含在此环境当前成功版本中。'
  return ''
}
