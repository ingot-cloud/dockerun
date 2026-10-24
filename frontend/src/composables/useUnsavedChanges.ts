import { computed, onBeforeUnmount, ref } from 'vue'
import { useRouter, type Router } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { createUnsavedChangesGuard, draftSnapshot, type UnsavedDraft } from '@/utils/unsavedChanges'

const guards = new WeakMap<Router, ReturnType<typeof installGuard>>()
function installGuard(router: Router) {
  const guard = createUnsavedChangesGuard(async () => {
    try {
      await ElMessageBox.confirm('当前内容尚未保存，离开后将丢失修改。是否放弃修改？', '有未保存的修改', {
        confirmButtonText: '放弃修改', cancelButtonText: '继续编辑', type: 'warning',
        customClass: 'unsaved-changes-confirm', confirmButtonClass: 'unsaved-changes-discard',
        closeOnClickModal: false, closeOnHashChange: false, distinguishCancelAndClose: true, autofocus: false,
      })
      return true
    } catch { return false }
  })
  const removeRouteGuard = router.beforeEach((to, from) => to.path === from.path ? !guard.busy() : guard.canLeave())
  const beforeUnload = (event: BeforeUnloadEvent) => {
    if (!guard.dirty()) return
    event.preventDefault()
    event.returnValue = ''
  }
  window.addEventListener('beforeunload', beforeUnload)
  return { guard, dispose() { removeRouteGuard(); window.removeEventListener('beforeunload', beforeUnload); guards.delete(router) } }
}

export function useUnsavedChanges(source: () => unknown, options: { enabled?: () => boolean; busy?: () => boolean } = {}) {
  const baseline = ref(draftSnapshot(source()))
  const dirty = computed(() => draftSnapshot(source()) !== baseline.value)
  const draft: UnsavedDraft = {
    active: options.enabled || (() => true), dirty: () => dirty.value, busy: options.busy || (() => false),
  }
  const router = useRouter()
  let installed = guards.get(router)
  if (!installed) { installed = installGuard(router); guards.set(router, installed) }
  const unregister = installed.guard.register(draft)
  onBeforeUnmount(() => { unregister(); if (!installed.guard.size) installed.dispose() })
  return {
    dirty,
    markSaved() { baseline.value = draftSnapshot(source()) },
    confirmDiscard: () => installed.guard.canLeave([draft]),
  }
}
