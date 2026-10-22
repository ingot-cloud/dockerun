<template>
  <el-dialog :model-value="modelValue" title="删除仓库" width="540px" class="registry-delete-dialog" append-to-body :close-on-click-modal="false" :close-on-press-escape="!deleting" :show-close="!deleting" @update:model-value="emit('update:modelValue', $event)" @closed="reset">
    <p class="registry-delete-description">确定删除以下 {{ targets.length }} 个仓库？仅移除此连接配置，仓库中的镜像不会被删除。</p>
    <ul class="registry-delete-targets"><li v-for="registry in targets" :key="registry.id"><strong>{{ registry.name }}</strong><span :title="registry.url">{{ removed.has(registry.id) ? '已删除' : registry.url }}</span></li></ul>
    <div v-if="failures.length" class="registry-delete-failures" role="alert"><strong>{{ failures.length }} 个仓库删除失败</strong><p v-for="failure in failures" :key="failure.id">{{ failure.name }}：{{ failure.message }}</p></div>
    <template #footer><el-button :disabled="deleting" @click="emit('update:modelValue', false)">{{ failures.length ? '关闭' : '取消' }}</el-button><el-button type="danger" :loading="deleting" :disabled="!pending.length" @click="remove">{{ failures.length ? '重试失败项' : '确认删除' }}</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { registryApi, type Registry } from '@/api/registry'
import { registryError } from '@/utils/registries'
const props = defineProps<{ modelValue: boolean; registries: Registry[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; removed: [ids: number[]] }>()
const targets = ref<Registry[]>([])
const removed = ref(new Set<number>())
const deleting = ref(false)
const failures = ref<{ id: number; name: string; message: string }[]>([])
const pending = computed(() => targets.value.filter(registry => !removed.value.has(registry.id)))
let disposed = false
watch(() => props.modelValue, open => { if (open) { reset(); targets.value = props.registries.filter(registry => registry.kind !== 'builtin') } })
function reset() { failures.value = []; removed.value = new Set(); targets.value = [] }
async function remove() {
  if (deleting.value || !pending.value.length) return
  deleting.value = true
  failures.value = []
  const succeeded: number[] = []
  try {
    for (const registry of pending.value) {
      if (disposed) break
      try { await registryApi.delete(registry.id); removed.value.add(registry.id); succeeded.push(registry.id) }
      catch (error) { failures.value.push({ id: registry.id, name: registry.name, message: registryError(error) }) }
    }
    if (disposed) return
    if (succeeded.length) { emit('removed', succeeded); ElMessage.success(`已删除 ${succeeded.length} 个仓库`) }
    if (!failures.value.length) emit('update:modelValue', false)
  } finally { deleting.value = false }
}
onBeforeUnmount(() => { disposed = true })
</script>
<style>
.registry-delete-dialog { --el-dialog-bg-color: #242424; border-radius: 10px; }
.registry-delete-dialog .el-dialog__body { padding: 20px 24px; }
.registry-delete-description { color: #bcbcbc; font-size: 13px; line-height: 1.7; margin-bottom: 16px; }
.registry-delete-targets { list-style: none; padding: 0; margin: 0; max-height: 220px; overflow: auto; }
.registry-delete-targets li { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 0; border-bottom: 1px solid #3b3b3b; }
.registry-delete-targets strong { overflow-wrap: anywhere; font-size: 13px; font-weight: 500; min-width: 0; }.registry-delete-targets span { font-size: 11px; color: #999; overflow-wrap: anywhere; max-width: 55%; }
.registry-delete-failures { margin-top: 16px; padding: 12px; border-radius: 6px; background: #40272b; color: #ffc0c7; font-size: 12px; line-height: 1.7; max-height: 180px; overflow: auto; }.registry-delete-failures p { margin-top: 8px; overflow-wrap: anywhere; }
</style>
