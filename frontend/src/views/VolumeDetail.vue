<template>
  <div class="volume-resource volume-detail-page" v-loading="loading">
    <header class="volume-heading"><h1>卷详情<span v-if="volume" class="volume-heading-name">{{ volume.Name }}</span></h1><div class="volume-heading-actions"><router-link :to="{ name: 'Volumes', query: listQuery }">返回卷列表</router-link><el-button text circle aria-label="刷新卷详情" title="刷新卷详情" :disabled="busy || loading" @click="loadVolume"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button></div></header>
    <div v-if="detailError" class="volume-error" role="alert"><span>{{ detailError }}</span><el-button link :disabled="loading" @click="loadVolume">重试</el-button></div>
    <template v-if="volume">
      <section class="volume-card" aria-labelledby="volume-details-title">
        <header class="volume-card-header"><h2 id="volume-details-title"><span class="volume-section-icon"><PortainerIcon name="volume" /></span>卷详情</h2><el-tooltip content="此卷仍被容器引用，包含已停止的容器" :disabled="!useCount"><span><el-button class="volume-remove" :disabled="!!useCount || busy || loading || !!detailError" @click="deleteVisible = true"><PortainerIcon name="trash" />删除此卷</el-button></span></el-tooltip></header>
        <dl class="volume-details"><div><dt>名称 / ID</dt><dd>{{ volume.Name }}</dd></div><div><dt>创建时间</dt><dd>{{ formatVolumeDate(volume.CreatedAt) }}</dd></div><div><dt>挂载路径</dt><dd class="volume-mono">{{ volume.Mountpoint || '—' }}</dd></div><div><dt>驱动</dt><dd>{{ volume.Driver }}</dd></div><div><dt>范围</dt><dd>{{ volume.Scope || '—' }}</dd></div><div v-if="volumeStack(volume)"><dt>应用栈</dt><dd>{{ volumeStack(volume) }}</dd></div><div><dt>使用状态</dt><dd><span v-if="useCount === null" class="volume-muted">未知</span><span v-else-if="useCount === 0" class="volume-unused">未使用</span><span v-else>被 {{ useCount }} 个容器引用</span></dd></div></dl>
      </section>
      <section v-if="Object.keys(volume.Options || {}).length" class="volume-card" aria-labelledby="volume-options-title"><header class="volume-card-header"><h2 id="volume-options-title"><span class="volume-section-icon"><PortainerIcon name="volume" /></span>驱动选项</h2></header><dl class="volume-details volume-map"><div v-for="(value, key) in volume.Options" :key="key"><dt>{{ key }}</dt><dd>{{ displayVolumeOption(String(key), value) || '—' }}</dd></div></dl></section>
      <section v-if="Object.keys(volume.Labels || {}).length" class="volume-card" aria-labelledby="volume-labels-title"><header class="volume-card-header"><h2 id="volume-labels-title"><span class="volume-section-icon"><PortainerIcon name="tag" /></span>标签</h2></header><dl class="volume-details volume-map"><div v-for="(value, key) in volume.Labels" :key="key"><dt>{{ key }}</dt><dd>{{ value || '—' }}</dd></div></dl></section>
      <section v-if="Object.keys(volume.Status || {}).length" class="volume-card"><header class="volume-card-header"><h2>驱动状态</h2></header><dl class="volume-details volume-map"><div v-for="(value, key) in volume.Status" :key="key"><dt>{{ key }}</dt><dd>{{ displayVolumeOption(String(key), typeof value === 'string' ? value : JSON.stringify(value)) }}</dd></div></dl></section>
      <section class="volume-card" aria-labelledby="volume-containers-title">
        <header class="volume-card-header"><h2 id="volume-containers-title"><span class="volume-section-icon"><PortainerIcon name="box" /></span>引用此卷的容器<span class="volume-muted">{{ useCount === null ? '—' : useCount }}</span></h2></header>
        <div v-if="usageError" class="volume-warning" role="status"><span>无法完整读取容器引用状态：{{ usageError }}</span><el-button link :disabled="loading || busy" @click="loadVolume">重试</el-button></div>
        <div class="volume-reference-scroll"><table class="volume-reference-table" aria-label="引用此卷的容器"><thead><tr><th scope="col">容器名称</th><th scope="col">状态</th><th scope="col">容器内挂载路径</th><th scope="col">只读</th></tr></thead><tbody><tr v-for="(reference, index) in references" :key="`${reference.id}-${index}`"><td><router-link :to="{ name: 'ContainerDetail', params: { id: reference.id } }">{{ reference.name }}</router-link></td><td>{{ containerStateLabel(reference.state) }}</td><td class="volume-mono">{{ reference.destination }}</td><td>{{ reference.readOnly === null ? '未知' : reference.readOnly ? '是' : '否' }}</td></tr><tr v-if="!references.length"><td colspan="4" class="volume-reference-empty">{{ useCount === null ? '暂时无法确认容器引用状态' : '没有容器引用此卷（包含已停止的容器）' }}</td></tr></tbody></table></div>
      </section>
    </template>
    <VolumeDeleteDialog v-model="deleteVisible" :volumes="volume ? [volume] : []" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import PortainerIcon from '@/components/PortainerIcon.vue'
import VolumeDeleteDialog from '@/components/volumes/VolumeDeleteDialog.vue'
import { dockerApi } from '@/api/docker'
import { containerStateLabel } from '@/utils/containers'
import { displayVolumeOption, formatVolumeDate, readVolumeListState, volumeListQuery, volumeReferences, volumeStack, volumeUseCount, type DockerVolume, type VolumeContainer } from '@/utils/volumes'
import { volumeError } from '@/utils/volumeFeedback'
import '@/styles/volumes.css'
const route = useRoute(), router = useRouter()
const listQuery = computed(() => volumeListQuery(readVolumeListState(route.query)))
const volume = ref<DockerVolume | null>(null), containers = ref<VolumeContainer[] | null>(null)
const loading = ref(false), detailError = ref(''), usageError = ref(''), deleteVisible = ref(false)
const busy = computed(() => deleteVisible.value), references = computed(() => volume.value ? volumeReferences(volume.value, containers.value) : [])
const useCount = computed(() => volume.value ? volumeUseCount(volume.value, containers.value) : null)
let controller: AbortController | undefined, disposed = false
onBeforeRouteLeave(() => !busy.value)
onBeforeRouteUpdate(() => !busy.value)
function onRemoved() { deleteVisible.value = false; volume.value = null; void router.replace({ name: 'Volumes', query: listQuery.value }) }
async function loadVolume() {
  controller?.abort()
  const current = new AbortController(); controller = current
  loading.value = true; detailError.value = ''; usageError.value = ''
  try {
    const [volumeResult, containerResult] = await Promise.allSettled([dockerApi.volumes.inspect(String(route.params.name), current.signal), dockerApi.containers.list(true, current.signal)])
    if (current.signal.aborted || disposed) return
    if (volumeResult.status === 'fulfilled') volume.value = volumeResult.value
    else detailError.value = `卷详情加载失败：${volumeError(volumeResult.reason)}`
    if (containerResult.status === 'fulfilled') { containers.value = (containerResult.value as unknown as VolumeContainer[] | null) || []; if (containers.value.some(container => container.Mounts === undefined)) usageError.value = '部分容器未返回挂载信息' }
    else { containers.value = null; usageError.value = volumeError(containerResult.reason) }
  } finally { if (!current.signal.aborted) loading.value = false }
}
watch(() => route.params.name, () => { volume.value = null; containers.value = null; void loadVolume() }, { immediate: true })
onBeforeUnmount(() => { disposed = true; controller?.abort() })
</script>
