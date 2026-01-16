<template>
  <el-dialog
    top="10vh"
    draggable
    v-bind="$attrs"
    :close-on-click-modal="false"
    :class="['form-dialog', { wide: wide, 'extra-wide': extraWide }]"
  >
    <template #header>
      <div class="dialog-header">
        <el-icon class="dialog-icon" v-if="icon">
          <component :is="icon" />
        </el-icon>
        <span class="dialog-title">{{ title }}</span>
      </div>
    </template>

    <div class="dialog-body">
      <slot></slot>
    </div>

    <template #footer v-if="$slots.footer || showFooter">
      <div class="dialog-footer">
        <slot name="footer">
          <el-button @click="$emit('cancel')">{{ cancelText }}</el-button>
          <el-button
            type="primary"
            :loading="loading"
            @click="$emit('confirm')"
          >
            {{ confirmText }}
          </el-button>
        </slot>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { Component } from "vue";

withDefaults(
  defineProps<{
    title?: string;
    icon?: Component;
    loading?: boolean;
    wide?: boolean;
    extraWide?: boolean;
    showFooter?: boolean;
    cancelText?: string;
    confirmText?: string;
  }>(),
  {
    showFooter: true,
    cancelText: "Cancel",
    confirmText: "Confirm",
  }
);

defineEmits<{
  cancel: [];
  confirm: [];
}>();
</script>

<style>
/* ========== 对话框基础样式 ========== */
.form-dialog.el-dialog {
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-2xl);
  overflow: hidden;
  background: var(--color-bg-card);
}

.form-dialog .el-dialog__header {
  margin: 0;
  padding: var(--spacing-xl) var(--spacing-xl);
  border-bottom: 1px solid var(--color-border-lighter);
  background: var(--color-bg-card);
}

.form-dialog .el-dialog__body {
  padding: 0;
  overflow: hidden;
  max-height: 65vh;
}

.form-dialog .el-dialog__footer {
  padding: 0;
  border-top: 1px solid var(--color-border-lighter);
}

/* 对话框宽度 */
.form-dialog.wide.el-dialog {
  width: 800px;
  max-width: 90vw;
}

.form-dialog.extra-wide.el-dialog {
  width: 1000px;
  max-width: 95vw;
}

/* ========== 对话框头部 ========== */
.dialog-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.dialog-icon {
  font-size: var(--font-size-xl);
  color: var(--color-primary);
}

.dialog-title {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  line-height: var(--line-height-tight);
}

/* ========== 对话框内容 ========== */
.dialog-body {
  padding: var(--spacing-xl);
  height: 70vh;
  overflow-y: auto;
}

/* ========== 对话框底部 ========== */
.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--spacing-sm);
  padding: var(--spacing-lg) var(--spacing-xl) 0;
}

/* ========== 表单样式 ========== */
.dialog-body .el-form {
  max-width: 100%;
}

.dialog-body .el-form-item {
  margin-bottom: var(--spacing-lg);
}

.dialog-body .el-form-item__label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
  line-height: var(--line-height-normal);
  padding-bottom: var(--spacing-xs);
}

.dialog-body .el-form-item__content {
  line-height: var(--line-height-normal);
}

/* 表单分组 */
.form-section {
  margin-bottom: var(--spacing-2xl);
  padding-bottom: var(--spacing-xl);
  border-bottom: 1px solid var(--color-border-lighter);
}

.form-section:last-child {
  margin-bottom: 0;
  padding-bottom: var(--spacing-xl);
  border-bottom: none;
}

.form-section h3 {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  margin: 0 0 var(--spacing-lg) 0;
  line-height: var(--line-height-tight);
}

/* 表单提示 */
.form-tip {
  margin-top: var(--spacing-xs);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  line-height: var(--line-height-normal);
}

/* ========== 输入框样式 ========== */
.dialog-body .el-input__inner,
.dialog-body .el-textarea__inner,
.dialog-body .el-select {
  font-size: var(--font-size-sm);
}

.dialog-body .el-input__inner:focus,
.dialog-body .el-textarea__inner:focus {
  border-color: var(--color-primary);
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .form-dialog .el-dialog {
    width: 95vw !important;
    margin: var(--spacing-lg);
  }

  .dialog-body {
    padding: var(--spacing-md);
  }

  .form-section {
    margin-bottom: var(--spacing-lg);
  }
}
</style>
