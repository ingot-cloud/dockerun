<template>
  <div class="page-container">
    <div class="page-header" v-if="$slots.header || title">
      <slot name="header">
        <h1 class="page-title">{{ title }}</h1>
      </slot>
    </div>
    <div class="page-content">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
}>()
</script>

<style scoped>
.page-container {
  min-height: 100vh;
  /* padding: var(--spacing-2xl); */
  background: var(--color-bg-page);
}

.page-header {
  padding: var(--spacing-xl);
  /* margin-bottom: var(--spacing-xl); */
}

.page-title {
  font-size: var(--font-size-3xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  margin: 0;
  line-height: var(--line-height-tight);
}

.page-content {
  /* Content area */
}
</style>
