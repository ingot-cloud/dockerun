<template>
  <WorkspaceEditor :class="{ 'container-fitted-editor': fittedWorkspace }" :title="`${route.meta.title}${details ? ` · ${name}` : ''}`" :refreshable="false">
    <template v-if="details" #actions><span class="workspace-editor-status" role="status" aria-label="容器状态">{{ containerStateLabel(details.State?.Status || '') }}</span></template>
    <template v-if="error" #feedback><div class="management-error" role="alert"><span>{{ error }}</span><el-button link :disabled="loading" @click="load">重试</el-button></div></template>
    <div class="container-resource container-workspace workspace-editor-page" :class="fittedWorkspace ? 'container-fitted-page' : 'workspace-page'">
    <div v-if="loading" class="resource-empty" v-loading="true">正在加载容器信息…</div>
    <template v-else-if="details">
      <ContainerLogs v-if="view === 'logs'" :key="id + view" :id="id" :name="name" />
      <ContainerInspect v-else-if="view === 'inspect'" :key="id + view" :details="details" />
      <ContainerStatistics v-else-if="view === 'stats'" :key="id + view" :id="id" :name="name" :running="details.State?.Status === 'running'" />
      <ContainerConsole v-else-if="view === 'exec' || view === 'attach'" :key="id + view" :id="id" :details="details" :mode="view" />
    </template>
    </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { dockerApi } from '@/api/docker'
import { containerStateLabel, type ContainerDetails } from '@/utils/containers'
import { containerError } from '@/utils/containerFeedback'
import ContainerLogs from '@/components/containers/ContainerLogs.vue'
import ContainerInspect from '@/components/containers/ContainerInspect.vue'
import ContainerStatistics from '@/components/containers/ContainerStatistics.vue'
import ContainerConsole from '@/components/containers/ContainerConsole.vue'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import '@/styles/containers.css'
const route = useRoute()
const id = computed(() => String(route.params.id)), view = computed(() => String(route.meta.containerView))
const fittedWorkspace = computed(() => ['logs', 'exec', 'attach'].includes(view.value))
const details = ref<ContainerDetails>(), loading = ref(false), error = ref('')
const name = computed(() => details.value?.Name?.replace(/^\//, '') || id.value.slice(0, 12))
let controller: AbortController | undefined
async function load() {
  controller?.abort(); const current = new AbortController(); controller = current; loading.value = true; error.value = ''; details.value = undefined
  try { const value = await dockerApi.containers.inspect(id.value, current.signal); if (!current.signal.aborted) details.value = value }
  catch (err) { if (!current.signal.aborted) error.value = containerError(err) }
  finally { if (!current.signal.aborted) loading.value = false }
}
watch(id, load, { immediate: true })
onBeforeUnmount(() => controller?.abort())
</script>
<style scoped>
.container-fitted-editor :deep(.workspace-editor-scroll) { display: flex; flex-direction: column; overflow: hidden; }
.container-fitted-editor :deep(.workspace-editor-page) { max-width: none; }
.container-fitted-editor :deep(.workspace-heading h1) { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; }
.container-fitted-page { flex: 1; min-height: 0; overflow: hidden; }
</style>
