<template>
  <div class="compact-items">
    <span v-for="(item, index) in items.slice(0, limit)" :key="index" :title="item">{{ item }}</span>
    <el-popover v-if="items.length > limit" placement="top" :width="320" trigger="click" popper-class="container-overflow-popover">
      <template #reference><button class="more-items" :aria-label="`查看全部${label}`">另有 {{ items.length - limit }} 项</button></template>
      <strong>{{ label }}</strong><ul><li v-for="(item, index) in items" :key="index">{{ item }}</li></ul>
    </el-popover>
    <span v-if="!items.length" class="muted">—</span>
  </div>
</template>
<script setup lang="ts">
withDefaults(defineProps<{ items: string[]; limit?: number; label: string }>(), { limit: 1 })
</script>
<style scoped>
.compact-items { display: flex; flex-direction: column; align-items: flex-start; gap: 2px; max-height: 44px; overflow: hidden; font-size: 12px; line-height: 19px; }
.compact-items > span { display: block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.more-items { border: 0; border-radius: 10px; background: #384355; color: #edf1f6; padding: 0 7px; font: inherit; line-height: 18px; cursor: pointer; white-space: nowrap; }
.muted { color: #888; }
</style>
<style>
.container-overflow-popover { background: #252525 !important; color: #ddd !important; border-color: #555 !important; font-size: 12px !important; }
.container-overflow-popover ul { padding: 0; list-style: none; margin: 8px 0 0; max-height: 240px; overflow: auto; }
.container-overflow-popover li { padding: 5px 0; overflow-wrap: anywhere; }
</style>
