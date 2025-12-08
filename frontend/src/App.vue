<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  width: 100%;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background-color: #1e1e1e;
  color: #fff;
}

/* Element Plus 深色主题覆盖 */
.el-button--primary {
  --el-button-bg-color: #388bfd;
  --el-button-border-color: #388bfd;
  --el-button-hover-bg-color: #4a9eff;
  --el-button-hover-border-color: #4a9eff;
}

.el-button--success {
  --el-button-bg-color: #2ecc71;
  --el-button-border-color: #2ecc71;
}

.el-button--warning {
  --el-button-bg-color: #f39c12;
  --el-button-border-color: #f39c12;
}

.el-button--danger {
  --el-button-bg-color: #e74c3c;
  --el-button-border-color: #e74c3c;
}

.el-button--info {
  --el-button-bg-color: #5dade2;
  --el-button-border-color: #5dade2;
}

.el-button--default {
  --el-button-bg-color: rgba(255, 255, 255, 0.08);
  --el-button-border-color: rgba(255, 255, 255, 0.2);
  --el-button-text-color: #fff;
  --el-button-hover-bg-color: rgba(255, 255, 255, 0.12);
  --el-button-hover-border-color: rgba(255, 255, 255, 0.3);
  --el-button-hover-text-color: #fff;
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
}

::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* 输入框全局样式 */
.el-input__wrapper {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none !important;
}

.el-input__wrapper:hover {
  border-color: rgba(56, 139, 253, 0.5);
}

.el-input__wrapper.is-focus {
  border-color: #388bfd;
}

.el-input__inner {
  color: #fff;
}

.el-input__inner::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

/* 下拉框样式 */
.el-select-dropdown {
  background-color: #2a2d31;
  border-color: rgba(255, 255, 255, 0.1);
}

.el-select-dropdown__item {
  color: #fff;
}

.el-select-dropdown__item:hover {
  background-color: rgba(74, 158, 255, 0.1);
}

.el-select-dropdown__item.is-selected {
  color: #4a9eff;
  background-color: rgba(74, 158, 255, 0.15);
}

/* 对话框样式 */
.el-dialog {
  background-color: #2a2d31;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.el-dialog__header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.el-dialog__title {
  color: #fff;
}

.el-dialog__body {
  color: #fff;
}

.el-dialog__footer {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

/* 消息提示样式 */
.el-message {
  background-color: #2a2d31;
  border-color: rgba(255, 255, 255, 0.1);
}

.el-message__content {
  color: #fff;
}

/* 加载样式 */
.el-loading-mask {
  background-color: rgba(30, 30, 30, 0.8);
}
</style>

