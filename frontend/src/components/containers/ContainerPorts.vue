<template>
  <div class="published-ports">
    <a v-for="(port, index) in ports.slice(0, 3)" :key="index" :href="containerPortURL(port, hostname)" target="_blank" rel="noopener noreferrer" :title="`${port.IP || '全部地址'}:${port.PublicPort} → ${port.PrivatePort}/${port.Type}`">{{ port.PublicPort }}:{{ port.PrivatePort }}<span>↗</span></a>
    <el-popover v-if="ports.length > 3" placement="top" :width="320" trigger="click" popper-class="container-overflow-popover">
      <template #reference><button class="more-ports" aria-label="查看全部端口映射">另有 {{ ports.length - 3 }} 项</button></template>
      <strong>端口映射</strong><ul><li v-for="(port, index) in ports" :key="index"><a :href="containerPortURL(port, hostname)" target="_blank" rel="noopener noreferrer">{{ port.IP || '全部地址' }}:{{ port.PublicPort }} → {{ port.PrivatePort }}/{{ port.Type }} ↗</a></li></ul>
    </el-popover>
    <span v-if="!ports.length" class="muted">—</span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { containerPortURL, publishedPorts, type Container } from '@/utils/containers'
const props = defineProps<{ container: Container }>()
const ports = computed(() => publishedPorts(props.container))
const hostname = window.location.hostname
</script>
<style scoped>
.published-ports { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 2px 7px; max-height: 42px; overflow: hidden; font-size: 11px; line-height: 19px; }
a { color: #35acd7; text-decoration: none; white-space: nowrap; }
a:hover { text-decoration: underline; }a > span { margin-left: 2px; }
.more-ports { justify-self: start; border: 0; border-radius: 10px; background: #384355; color: #edf1f6; padding: 0 7px; font: inherit; line-height: 18px; cursor: pointer; white-space: nowrap; }
.muted { color: #888; }
</style>
