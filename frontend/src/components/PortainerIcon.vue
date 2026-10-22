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
    <template v-else-if="name === 'box'">
      <path d="m12 3 9 5v8l-9 5-9-5V8z" /><path d="m3 8 9 5 9-5M12 13v8M7.5 5.5l9 5" />
    </template>
    <path v-else-if="name === 'play'" d="m7 4 14 8-14 8z" />
    <path v-else-if="name === 'kill'" d="m5 19 14-14" />
    <template v-else-if="name === 'pause'"><path d="M6 4h3v16H6zM15 4h3v16h-3z" /></template>
    <template v-else-if="name === 'info'"><circle cx="12" cy="12" r="9" /><path d="M12 11v6M12 7h.01" /></template>
    <template v-else-if="name === 'terminal'"><path d="m4 5 7 7-7 7M13 19h7" /></template>
    <path v-else-if="name === 'attach'" d="m21 11-8 8a6 6 0 0 1-8-8L15 2a4 4 0 0 1 6 6L10 18a2 2 0 0 1-3-3L17 5" />
    <rect v-else-if="name === 'stop'" x="5" y="5" width="14" height="14" rx="1" />
    <template v-else-if="name === 'logs'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><path d="M14 2v6h6M8 13h8M8 17h6" />
    </template>
    <template v-else-if="name === 'inspect'">
      <path d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M21 16v3a2 2 0 0 1-2 2h-3M8 21H5a2 2 0 0 1-2-2v-3" /><circle cx="12" cy="12" r="3" />
    </template>
    <template v-else-if="name === 'chart'">
      <path d="M3 3v18h18M8 16v-5M13 16V7M18 16v-8" />
    </template>
    <template v-else-if="name === 'link'">
      <path d="M10 13a5 5 0 0 0 7 .1l3-3a5 5 0 0 0-7-7l-1.7 1.7M14 11a5 5 0 0 0-7-.1l-3 3a5 5 0 0 0 7 7l1.7-1.7" />
    </template>
    <template v-else-if="name === 'volume'"><ellipse cx="12" cy="5" rx="8" ry="3" /><path d="M4 5v14c0 1.66 3.58 3 8 3s8-1.34 8-3V5M4 12c0 1.66 3.58 3 8 3s8-1.34 8-3" /></template>
    <template v-else-if="name === 'registry'"><rect x="3" y="3" width="18" height="7" rx="2" /><rect x="3" y="14" width="18" height="7" rx="2" /><path d="M7 6.5h.01M7 17.5h.01M11 6.5h6M11 17.5h6" /></template>
    <path v-else-if="name === 'plus'" d="M12 5v14M5 12h14" />
    <template v-else-if="name === 'network'"><rect x="9" y="2" width="6" height="6" rx="1" /><rect x="2" y="16" width="6" height="6" rx="1" /><rect x="16" y="16" width="6" height="6" rx="1" /><path d="M12 8v4M5 16v-4h14v4" /></template>
    <path v-else-if="name === 'chevron-down'" d="m6 9 6 6 6-6" />
    <template v-else-if="name === 'sort'">
      <path fill="currentColor" stroke="none" d="M4.473.727a.593.593 0 1 0-1.186 0h1.186ZM3.88 11.274l-.406.432c.233.22.599.214.826-.013l-.42-.42Zm3.056-2.218a.593.593 0 1 0-.839-.839l.84.84Zm-5.45-.851a.593.593 0 1 0-.814.864l.813-.864Zm1.8-7.478v10.547h1.187V.727H3.287Zm2.811 7.49-2.636 2.637.839.84 2.636-2.638-.839-.839Zm-1.81 2.625L1.485 8.205l-.813.864 2.802 2.637.813-.864Z" />
      <path fill="currentColor" stroke="none" transform="translate(8)" d="M3.526 11.273a.593.593 0 0 0 1.187 0H3.526ZM4.12.727l.406-.432A.593.593 0 0 0 3.7.307l.42.42ZM1.063 2.944a.593.593 0 1 0 .84.839l-.84-.84Zm5.451.851a.593.593 0 1 0 .814-.864l-.814.864Zm-1.801 7.478V.727H3.526v10.546h1.187Zm-2.81-7.49 2.636-2.637L3.7.307 1.063 2.944l.84.839Zm1.81-2.624 2.801 2.636.814-.864L4.526.295l-.813.864Z" />
    </template>
  </svg>
</template>

<script setup lang="ts">
// 资源页面共用的 Lucide 风格线性图标。
defineProps<{
  name: 'list' | 'trash' | 'upload' | 'download' | 'refresh' | 'search' | 'chevron-down' | 'sort' | 'tag' | 'box' | 'play' | 'stop' | 'kill' | 'pause' | 'info' | 'terminal' | 'attach' | 'logs' | 'inspect' | 'chart' | 'link' | 'plus' | 'network' | 'volume' | 'registry'
}>()
</script>

<style scoped>
.portainer-icon { width: 16px; height: 16px; flex-shrink: 0; vertical-align: middle; }
</style>
