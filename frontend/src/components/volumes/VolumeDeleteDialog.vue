<template>
  <el-dialog :model-value="modelValue" title="删除卷" width="540px" class="volume-delete-dialog" append-to-body :close-on-click-modal="false" :close-on-press-escape="!deleting" :show-close="!deleting" @update:model-value="emit('update:modelValue', $event)" @closed="reset">
    <p class="volume-delete-description">确定删除以下 {{ targets.length }} 个卷？删除会永久移除卷中的数据，无法撤销。仍被容器引用的卷无法删除，包括已停止的容器。</p>
    <ul class="volume-delete-targets"><li v-for="volume in targets" :key="volume.Name"><strong>{{ volume.Name }}</strong><span>{{ removed.has(volume.Name) ? '已删除' : volume.Driver }}</span></li></ul>
    <div v-if="failures.length" class="volume-delete-failures" role="alert"><strong>{{ failures.length }} 个卷删除失败</strong><p v-for="failure in failures" :key="failure.name">{{ failure.name }}：{{ failure.message }}</p></div>
    <template #footer><el-button :disabled="deleting" @click="emit('update:modelValue', false)">{{ failures.length ? '关闭' : '取消' }}</el-button><el-button type="danger" :loading="deleting" :disabled="!pending.length" @click="remove">{{ failures.length ? '重试失败项' : '确认删除' }}</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'
import type { DockerVolume } from '@/utils/volumes'
import { volumeError } from '@/utils/volumeFeedback'
const props = defineProps<{ modelValue: boolean; volumes: DockerVolume[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; removed: [names: string[]] }>()
const targets = ref<DockerVolume[]>([]), removed = ref(new Set<string>()), deleting = ref(false)
const failures = ref<{ name: string; message: string }[]>([])
const pending = computed(() => targets.value.filter(volume => !removed.value.has(volume.Name)))
let disposed = false
watch(() => props.modelValue, open => { if (open) { reset(); targets.value = [...props.volumes] } })
function reset() { failures.value = []; removed.value = new Set(); targets.value = [] }
async function remove() {
  if (deleting.value || !pending.value.length) return
  deleting.value = true; failures.value = []
  const succeeded: string[] = []
  try {
    for (const volume of pending.value) {
      if (disposed) break
      try { await dockerApi.volumes.remove(volume.Name, false); removed.value.add(volume.Name); succeeded.push(volume.Name) }
      catch (error) { failures.value.push({ name: volume.Name, message: volumeError(error) }) }
    }
    if (disposed) return
    if (succeeded.length) { emit('removed', succeeded); ElMessage.success(`已删除 ${succeeded.length} 个卷`) }
    if (!failures.value.length) emit('update:modelValue', false)
  } finally { deleting.value = false }
}
onBeforeUnmount(() => { disposed = true })
</script>
<style>
.volume-delete-dialog { --el-dialog-bg-color: #242424; border-radius: 10px; }.volume-delete-dialog .el-dialog__body { padding: 20px 24px; }
.volume-delete-description { color: #bcbcbc; font-size: 13px; line-height: 1.7; margin-bottom: 16px; }
.volume-delete-targets { list-style: none; padding: 0; margin: 0; max-height: 220px; overflow: auto; }.volume-delete-targets li { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 0; border-bottom: 1px solid #3b3b3b; }.volume-delete-targets strong { overflow-wrap: anywhere; font-size: 13px; font-weight: 500; }.volume-delete-targets span { font-size: 11px; color: #999; white-space: nowrap; }
.volume-delete-failures { margin-top: 16px; padding: 12px; border-radius: 6px; background: #40272b; color: #ffc0c7; font-size: 12px; line-height: 1.7; max-height: 180px; overflow: auto; }.volume-delete-failures p { margin-top: 8px; overflow-wrap: anywhere; }
</style>
