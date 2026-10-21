<template>
  <nav class="container-quick-actions" aria-label="容器快捷操作" :class="{ 'with-labels': labels }">
    <template v-for="view in containerViews" :key="view.key">
      <el-tooltip v-if="['logs', 'inspect'].includes(view.key) || container.State === 'running'" :content="view.label" placement="top">
        <router-link :to="{ name: routeNames[view.key], params: { id: container.Id }, query }" :aria-label="`${view.label} ${containerName(container)}`"><PortainerIcon :name="view.icon" /><span v-if="labels">{{ view.label }}</span></router-link>
      </el-tooltip>
    </template>
  </nav>
</template>
<script setup lang="ts">
import PortainerIcon from '@/components/PortainerIcon.vue'
import { containerName, containerViews, type Container } from '@/utils/containers'
defineProps<{ container: Container; query?: Record<string, string>; labels?: boolean }>()
const routeNames = { logs: 'ContainerLogs', inspect: 'ContainerInspect', stats: 'ContainerStats', exec: 'ContainerExec', attach: 'ContainerAttach' }
</script>
<style scoped>
.container-quick-actions { display: flex; align-items: center; gap: 7px; white-space: nowrap; }
a { display: inline-flex; align-items: center; gap: 5px; color: #35acd7; text-decoration: none; }
a:hover { color: #85d3ee; }
.portainer-icon { width: 14px; height: 14px; }
.with-labels { flex-wrap: wrap; gap: 20px; font-size: 12px; }
</style>
