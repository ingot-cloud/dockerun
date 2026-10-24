export function draftSnapshot(value: unknown): string {
  return JSON.stringify(value, (_key, item) => item && typeof item === 'object' && !Array.isArray(item)
    ? Object.fromEntries(Object.entries(item).sort(([a], [b]) => a.localeCompare(b)))
    : item) ?? ''
}

export interface UnsavedDraft {
  active: () => boolean
  dirty: () => boolean
  busy: () => boolean
}

// 同一页面的表单和嵌套弹窗共用一次离开确认。
export function createUnsavedChangesGuard(confirm: () => Promise<boolean>) {
  const drafts = new Set<UnsavedDraft>()
  let pending: Promise<boolean> | undefined
  const active = (items: Iterable<UnsavedDraft>) => [...items].filter(item => item.active())
  const busy = () => active(drafts).some(item => item.busy())
  const dirty = () => active(drafts).some(item => item.dirty())
  async function canLeave(items: Iterable<UnsavedDraft> = drafts): Promise<boolean> {
    const current = active(items)
    if (current.some(item => item.busy())) return false
    if (!current.some(item => item.dirty())) return true
    pending ??= Promise.resolve().then(confirm).catch(() => false).finally(() => { pending = undefined })
    const accepted = await pending
    // 确认只授权这次离开；后续导航被取消时，草稿仍保持未保存状态。
    return accepted && !active(items).some(item => item.busy())
  }
  return {
    register(draft: UnsavedDraft) { drafts.add(draft); return () => drafts.delete(draft) },
    get size() { return drafts.size },
    busy, dirty, canLeave,
  }
}
