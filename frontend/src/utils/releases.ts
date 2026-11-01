export interface DeploymentScripts { before: string; deploy: string; after: string }
export interface DeploymentExecution { mode: 'command'|'compose'; command: string; work_dir: string; check_command: string; timeout_seconds: number }
export interface ReleaseRegistry { id: number; name: string; url: string; kind?: string; is_default?: boolean }
export const emptyScripts = (): DeploymentScripts => ({ before: '', deploy: '', after: '' })
export function copyScripts(value?: Partial<DeploymentScripts> | null): DeploymentScripts {
  return { before: value?.before || '', deploy: value?.deploy || '', after: value?.after || '' }
}
export function hasScripts(value?: Partial<DeploymentScripts>): boolean {
  return !!(value?.before?.trim() || value?.deploy?.trim() || value?.after?.trim())
}

export function deploymentParameterReferences(content: string): string[] {
  return [...new Set([...content.matchAll(/\$(?:\{([A-Za-z_][A-Za-z0-9_]*)|([A-Za-z_][A-Za-z0-9_]*))/g)]
    .map(match => match[1] || match[2] || '')
    .filter(key => !['WORK_DIR', 'COMPOSE_FILE', 'COMPOSE_PROJECT_NAME'].includes(key) && !key.startsWith('DOCKERUN_')))]
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
export function automationIssue(environment: any, releases: any[], application?: any): string {
  if (!environment.repository?.trim()) return '请填写需要监听的镜像名称。'
  const definition = application?.definition
  if (definition?.execution?.mode === 'command') {
    if (!/\$(DOCKERUN_IMAGE\b|\{DOCKERUN_IMAGE[}:?+%-])/.test(definition.execution.command || '')) return '请先在部署命令中使用 $DOCKERUN_IMAGE，并保存部署配置；可用 ${DOCKERUN_IMAGE:-$IMAGE} 兼容手动发布。'
    return ''
  }
  if (definition?.compose?.trim()) {
    if (['local', 'bundle'].includes(definition.source)) return '请先保存使用仓库镜像的在线部署配置。'
    if (definition.source === 'registry' && definition.registry_id && definition.registry_id !== environment.registry_id) return '请选择与已保存部署配置一致的镜像仓库。'
    return '' // Compose 镜像引用和环境参数由服务端解析校验。
  }
  const base = releases.find(item => item.id === environment?.current_release_id)
  if (!base) return '请先在此环境完成首次手动发布，再开启自动发布。'
  if (base.source !== 'registry' || base.registry_id !== environment.registry_id) return '请先使用所选仓库在此环境手动发布一次，再开启自动发布。'
  if (!releaseImages(base).some(image => imageRepository(image.reference) === environment.repository.trim())) return '镜像名称必须包含在此环境当前成功版本中。'
  return ''
}

// 复制历史版本生成独立草稿；后续保存与编辑不会修改原版本。
export function deploymentDraftFromRelease(version: any) {
  const scripts = copyScripts(version.scripts)
  const execution = { command: '', work_dir: '', check_command: '', timeout_seconds: 0, ...version.execution, mode: version.execution?.mode || 'compose' }
  if (execution.mode === 'compose' && !execution.command && scripts.deploy) {
    execution.command = scripts.deploy
    scripts.deploy = ''
  }
  return {
    compose: version.compose || '',
    execution,
    configs: JSON.parse(version.configs || '{}'),
    source: version.source === 'command' ? 'direct' : version.source === 'bundle' && execution.mode !== 'command' ? 'local' : version.source || 'direct',
    registry_id: version.registry_id || 0,
    platform: version.platform || '',
    scripts,
  }
}

export function releaseDeletionReason(version: {deletion_blocked_reason?: string}): string {
  return typeof version.deletion_blocked_reason === 'string' ? version.deletion_blocked_reason : '版本引用状态尚未确认，请刷新后重试'
}
