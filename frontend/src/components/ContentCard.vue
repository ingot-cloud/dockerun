<template>
  <div class="content-card">
    <div class="card-header" v-if="$slots.header || title">
      <slot name="header">
        <div class="header-left">
          <el-icon class="header-icon" v-if="icon">
            <component :is="icon" />
          </el-icon>
          <span class="header-title">{{ title }}</span>
        </div>
        <div class="header-right">
          <slot name="actions"></slot>
        </div>
      </slot>
    </div>
    
    <div class="card-body" :class="{ 'no-padding': noPadding }">
      <slot></slot>
    </div>

    <div class="card-footer" v-if="$slots.footer">
      <slot name="footer"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  title?: string
  icon?: Component
  noPadding?: boolean
}>()
</script>

<style scoped>
.content-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  transition: box-shadow var(--transition-base);
}

.content-card:hover {
  box-shadow: var(--shadow-md);
}

/* ========== Card Header ========== */
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-lg) var(--spacing-xl);
  border-bottom: 1px solid var(--color-border-lighter);
  background: var(--color-bg-card);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.header-icon {
  font-size: var(--font-size-xl);
  color: var(--color-primary);
}

.header-title {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  line-height: var(--line-height-tight);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

/* ========== Card Body ========== */
.card-body {
  padding: var(--spacing-xl);
}

.card-body.no-padding {
  padding: 0;
}

/* ========== Card Footer ========== */
.card-footer {
  padding: var(--spacing-md) var(--spacing-xl);
  border-top: 1px solid var(--color-border-lighter);
  background: rgba(0, 0, 0, 0.2);
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--spacing-md);
  }

  .header-right {
    width: 100%;
    flex-wrap: wrap;
  }
}
</style>
