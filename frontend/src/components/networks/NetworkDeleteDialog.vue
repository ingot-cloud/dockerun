<template>
  <el-dialog :model-value="modelValue" title="删除网络" width="540px" class="network-delete-dialog" append-to-body :close-on-click-modal="false" :close-on-press-escape="!deleting" :show-close="!deleting" @update:model-value="emit('update:modelValue', $event)" @closed="reset">
    <p class="network-delete-description">确定删除以下 {{ targets.length }} 个网络？已连接容器的网络无法删除，请先断开容器连接。</p>
    <ul class="network-delete-targets"><li v-for="network in targets" :key="network.Id"><strong>{{ network.Name }}</strong><span>{{ removed.has(network.Id) ? '已删除' : `${network.Driver} · ${network.Id.slice(0, 12)}` }}</span></li></ul>
    <div v-if="failures.length" class="network-delete-failures" role="alert"><strong>{{ failures.length }} 个网络删除失败</strong><p v-for="failure in failures" :key="failure.id">{{ failure.name }}：{{ failure.message }}</p></div>
    <template #footer><el-button :disabled="deleting" @click="emit('update:modelValue', false)">{{ failures.length ? '关闭' : '取消' }}</el-button><el-button type="danger" :loading="deleting" :disabled="!pending.length" @click="remove">{{ failures.length ? '重试失败项' : '确认删除' }}</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'
import { canRemoveNetwork, type DockerNetwork } from '@/utils/networks'
import { networkError } from '@/utils/networkFeedback'
const props = defineProps<{ modelValue: boolean; networks: DockerNetwork[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; removed: [ids: string[]] }>()
const targets = ref<DockerNetwork[]>([])
const removed = ref(new Set<string>())
const deleting = ref(false)
const failures = ref<{ id: string; name: string; message: string }[]>([])
const pending = computed(() => targets.value.filter(network => !removed.value.has(network.Id)))
let disposed = false
watch(() => props.modelValue, open => { if (open) { reset(); targets.value = props.networks.filter(canRemoveNetwork) } })
function reset() { failures.value = []; removed.value = new Set(); targets.value = [] }
async function remove() {
  if (deleting.value || !pending.value.length) return
  deleting.value = true
  failures.value = []
  const succeeded: string[] = []
  try {
    for (const network of pending.value) {
      if (disposed) break
      try { await dockerApi.networks.remove(network.Id); removed.value.add(network.Id); succeeded.push(network.Id) }
      catch (error) { failures.value.push({ id: network.Id, name: network.Name, message: networkError(error) }) }
    }
    if (disposed) return
    if (succeeded.length) { emit('removed', succeeded); ElMessage.success(`已删除 ${succeeded.length} 个网络`) }
    if (!failures.value.length) emit('update:modelValue', false)
  } finally { deleting.value = false }
}
onBeforeUnmount(() => { disposed = true })
</script>
<style>
.network-delete-dialog { --el-dialog-bg-color: #242424; border-radius: 10px; }
.network-delete-dialog .el-dialog__body { padding: 20px 24px; }
.network-delete-description { color: #bcbcbc; font-size: 13px; line-height: 1.7; margin-bottom: 16px; }
.network-delete-targets { list-style: none; padding: 0; margin: 0; max-height: 220px; overflow: auto; }
.network-delete-targets li { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 0; border-bottom: 1px solid #3b3b3b; }
.network-delete-targets strong { overflow-wrap: anywhere; font-size: 13px; font-weight: 500; }.network-delete-targets span { font-size: 11px; color: #999; white-space: nowrap; }
.network-delete-failures { margin-top: 16px; padding: 12px; border-radius: 6px; background: #40272b; color: #ffc0c7; font-size: 12px; line-height: 1.7; max-height: 180px; overflow: auto; }.network-delete-failures p { margin-top: 8px; overflow-wrap: anywhere; }
</style>
