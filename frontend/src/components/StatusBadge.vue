<template>
  <span class="state" :class="`state-${entry[1]}`">{{ entry[0] }}</span>
</template>
<script setup lang="ts">
import { computed } from "vue";
const props = defineProps<{ value?: string }>();
const labels: Record<string, string[]> = {
  queued: ["排队中", "info"],
  running: ["执行中", "info"],
  success: ["成功", "success"],
  failed: ["失败", "danger"],
  needs_attention: ["需要核对", "warning"],
  cancelled: ["已取消", "neutral"],
  healthy: ["健康", "success"],
  unverified: ["已启动，未验证健康", "warning"],
  unknown: ["未知", "neutral"],
  ready: ["可用", "success"],
  validating: ["校验中", "info"],
  loading: ["镜像加载中", "info"],
  blocked: ["等待上一台", "neutral"],
  paused: ["已暂停", "warning"],
};
const entry = computed(
  () => labels[props.value || "unknown"] || [props.value || "未知", "neutral"],
);
</script>
