<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="24"
    height="24"
    :viewBox="name === 'sort' ? '0 0 16 12' : '0 0 24 24'"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    class="portainer-icon"
  >
    <template v-if="name === 'list'">
      <path d="M3 5h.01" /><path d="M3 12h.01" /><path d="M3 19h.01" />
      <path d="M8 5h13" /><path d="M8 12h13" /><path d="M8 19h13" />
    </template>
    <template v-else-if="name === 'trash'">
      <path d="M10 11v6" /><path d="M14 11v6" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
      <path d="M3 6h18" /><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
    </template>
    <template v-else-if="name === 'upload'">
      <path d="M12 3v12" /><path d="m17 8-5-5-5 5" />
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
    </template>
    <template v-else-if="name === 'download'">
      <path d="M12 15V3" />
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <path d="m7 10 5 5 5-5" />
    </template>
    <template v-else-if="name === 'refresh'">
      <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
      <path d="M8 16H3v5" />
    </template>
    <template v-else-if="name === 'search'">
      <path d="m21 21-4.34-4.34" /><circle cx="11" cy="11" r="8" />
    </template>
    <template v-else-if="name === 'tag'">
      <path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z" />
      <circle cx="7.5" cy="7.5" r=".5" fill="currentColor" />
    </template>
    <path v-else-if="name === 'chevron-down'" d="m6 9 6 6 6-6" />
    <template v-else-if="name === 'sort'">
      <path fill="currentColor" stroke="none" d="M4.473.727a.593.593 0 1 0-1.186 0h1.186ZM3.88 11.274l-.406.432c.233.22.599.214.826-.013l-.42-.42Zm3.056-2.218a.593.593 0 1 0-.839-.839l.84.84Zm-5.45-.851a.593.593 0 1 0-.814.864l.813-.864Zm1.8-7.478v10.547h1.187V.727H3.287Zm2.811 7.49-2.636 2.637.839.84 2.636-2.638-.839-.839Zm-1.81 2.625L1.485 8.205l-.813.864 2.802 2.637.813-.864Z" />
      <path fill="currentColor" stroke="none" transform="translate(8)" d="M3.526 11.273a.593.593 0 0 0 1.187 0H3.526ZM4.12.727l.406-.432A.593.593 0 0 0 3.7.307l.42.42ZM1.063 2.944a.593.593 0 1 0 .84.839l-.84-.84Zm5.451.851a.593.593 0 1 0 .814-.864l-.814.864Zm-1.801 7.478V.727H3.526v10.546h1.187Zm-2.81-7.49 2.636-2.637L3.7.307 1.063 2.944l.84.839Zm1.81-2.624 2.801 2.636.814-.864L4.526.295l-.813.864Z" />
    </template>
  </svg>
</template>

<script setup lang="ts">
// 与 Portainer Images 页面使用的 Lucide SVG 路径及描边保持一致。
defineProps<{
  name: 'list' | 'trash' | 'upload' | 'download' | 'refresh' | 'search' | 'chevron-down' | 'sort' | 'tag'
}>()
</script>

<style scoped>
.portainer-icon { width: 16px; height: 16px; flex-shrink: 0; vertical-align: middle; }
</style>
