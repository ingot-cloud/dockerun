<template>
  <div class="data-table-wrapper">
    <el-table
      v-bind="$attrs"
      class="data-table"
      :row-class-name="rowClassName"
      :header-cell-style="headerCellStyle"
      stripe
    >
      <slot></slot>
    </el-table>
    
    <div class="table-footer" v-if="showFooter">
      <slot name="footer">
        <span class="total-count">{{ total }} {{ totalText }}</span>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CSSProperties } from 'vue'

withDefaults(defineProps<{
  total?: number
  totalText?: string
  showFooter?: boolean
  rowClassName?: (row: any, rowIndex: number) => string
}>(), {
  totalText: 'items',
  showFooter: true,
  rowClassName: () => ''
})

const headerCellStyle: CSSProperties = {
  background: 'rgba(0, 0, 0, 0.2)',
  color: 'rgba(255, 255, 255, 0.85)',
  fontWeight: '600',
  fontSize: '13px',
  padding: '12px 0',
  height: '48px'
}
</script>

<style>
/* ========== 全局表格样式（不使用 scoped） ========== */
.data-table-wrapper {
  width: 100%;
}

.data-table {
  width: 100%;
  font-size: var(--font-size-sm);
}

/* 单元格样式 */
.data-table .el-table__cell {
  padding: 12px 0;
  height: auto;
}

.data-table .el-table__body-wrapper .el-table__cell {
  color: var(--color-text-regular);
}

/* 表头样式 */
.data-table .el-table__header-wrapper {
  border-radius: var(--radius-md) var(--radius-md) 0 0;
}

/* 斑马纹 */
.data-table .el-table__row--striped {
  background-color: rgba(255, 255, 255, 0.02);
}

/* 悬停效果 */
.data-table .el-table__body tr:hover > td {
  background-color: var(--color-bg-hover) !important;
}

/* 选中行 */
.data-table .el-table__body tr.current-row > td {
  background-color: rgba(64, 158, 255, 0.15) !important;
}

/* 边框 */
.data-table .el-table__inner-wrapper::before {
  display: none;
}

.data-table td.el-table__cell,
.data-table th.el-table__cell.is-leaf {
  border-bottom: 1px solid var(--color-border-lighter);
}

/* 空状态 */
.data-table .el-table__empty-block {
  padding: var(--spacing-3xl) 0;
}

.data-table .el-table__empty-text {
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

/* 加载状态 */
.data-table .el-loading-mask {
  background-color: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(2px);
}

/* ========== 表格底部 ========== */
.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-md) var(--spacing-xl);
  border-top: 1px solid var(--color-border-lighter);
  background: rgba(0, 0, 0, 0.2);
}

.total-count {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  font-weight: var(--font-weight-medium);
}

/* ========== 快速操作按钮 ========== */
.quick-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.quick-actions .el-button {
  margin: 0;
  padding: 6px;
  font-size: 16px;
}

.quick-actions .el-button.is-link {
  color: var(--color-text-secondary);
  transition: color var(--transition-fast);
}

.quick-actions .el-button.is-link:hover {
  color: var(--color-primary);
}

.quick-actions .el-button.danger-btn:hover {
  color: var(--color-danger);
}

/* ========== 通用单元格样式 ========== */
.text-normal {
  color: var(--color-text-regular);
  font-size: var(--font-size-sm);
}

.text-muted {
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

.code-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: var(--font-size-xs);
  color: var(--color-text-regular);
  background: var(--color-bg-hover);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
</style>
