export function templateDeletionID(template: { id: string; source: string }): number | null {
  if (template.source !== 'custom' || !/^custom-[1-9]\d*$/.test(template.id)) return null
  const id = Number(template.id.slice(7))
  return Number.isSafeInteger(id) && id <= 0xffffffff ? id : null
}

export function templateDeletionReason(template: { id: string; source: string }) {
  if (template.source !== 'custom') return '内置模板不能删除，可删除自行保存的应用模板'
  return templateDeletionID(template) === null ? '模板编号无效，请刷新后重试' : ''
}

export function bundleDeletionReason(bundle: { status?: string }) {
  if (bundle.status === 'ready' || bundle.status === 'failed') return ''
  return ['queued', 'validating', 'loading'].includes(bundle.status || '') ? '材料正在导入，请等待导入结束后再删除' : '导入状态未知，请刷新后重试'
}

export interface WorkspaceDeleteTarget { id: number; name: string }

// 确认弹窗使用独立快照，去重并拒绝无效编号，避免批量请求误删或重复提交。
export function workspaceDeleteTargets<T>(items: readonly T[], identify: (item: T) => number | null, name: (item: T) => string): WorkspaceDeleteTarget[] {
  const targets: WorkspaceDeleteTarget[] = [], seen = new Set<number>()
  for (const item of items) {
    const id = identify(item)
    if (id === null || !Number.isSafeInteger(id) || id <= 0 || id > 0xffffffff || seen.has(id)) continue
    seen.add(id)
    targets.push({ id, name: name(item) })
  }
  return targets
}
