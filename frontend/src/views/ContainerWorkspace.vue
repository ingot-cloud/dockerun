<template>
  <div class="container-resource container-workspace">
    <header class="resource-heading"><h1>{{ route.meta.title }}<small v-if="details">{{ details.Name?.replace(/^\//, '') }}</small></h1><div class="resource-heading-actions"><router-link :to="{ name: 'ContainerDetail', params: { id }, query: listQuery }">返回容器详情</router-link><router-link :to="{ name: 'Containers', query: listQuery }">返回容器列表</router-link></div></header>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-if="loading" class="resource-empty" v-loading="true">正在加载容器信息…</div>
    <template v-else-if="details">
      <ContainerLogs v-if="view === 'logs'" :key="id + view" :id="id" :name="name" />
      <ContainerInspect v-else-if="view === 'inspect'" :key="id + view" :details="details" />
      <ContainerStatistics v-else-if="view === 'stats'" :key="id + view" :id="id" :name="name" :running="details.State?.Status === 'running'" />
      <ContainerConsole v-else-if="view === 'exec' || view === 'attach'" :key="id + view" :id="id" :details="details" :mode="view" />
    </template>
    <el-button v-else-if="error" @click="load">重试</el-button>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { dockerApi } from '@/api/docker'
import { containerListQuery, readContainerListState, type ContainerDetails } from '@/utils/containers'
import { containerError } from '@/utils/containerFeedback'
import ContainerLogs from '@/components/containers/ContainerLogs.vue'
import ContainerInspect from '@/components/containers/ContainerInspect.vue'
import ContainerStatistics from '@/components/containers/ContainerStatistics.vue'
import ContainerConsole from '@/components/containers/ContainerConsole.vue'
import '@/styles/containers.css'
const route = useRoute()
const id = computed(() => String(route.params.id)), view = computed(() => String(route.meta.containerView))
const listQuery = computed(() => containerListQuery(readContainerListState(route.query)))
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
h1 small { margin-left: 12px; font-size: 12px; font-weight: 400; color: #999; }
</style>
