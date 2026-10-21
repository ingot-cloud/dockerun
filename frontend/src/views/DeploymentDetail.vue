<template>
  <div class="page-heading">
    <div>
      <h1>部署 #{{ route.params.id }}</h1>
      <p>状态、服务健康和恢复结果分别记录。</p>
    </div>
    <StatusBadge :value="task.status" />
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <div class="stats-grid">
    <section class="stat-card">
      阶段<strong style="font-size: 20px">{{ task.stage || "—" }}</strong>
    </section>
    <section class="stat-card">
      版本<strong style="font-size: 20px">#{{ task.release_id || "—" }}</strong>
    </section>
    <section class="stat-card">
      恢复结果
      <div style="margin-top: 16px">
        <StatusBadge :value="task.recovery_status || 'unknown'" />
      </div>
    </section>
    <section class="stat-card">
      触发身份<strong style="font-size: 20px">用户 #{{ task.user_id }}</strong>
    </section>
  </div>
  <section class="panel">
    <h2>执行结果</h2>
    <p>{{ task.result || "任务正在等待或执行" }}</p>
    <pre
      v-if="task.actual_images"
      class="code-block"
      style="margin-top: 16px"
      >{{ task.actual_images }}</pre
    >
  </section>
  <section v-if="nodes.length > 1" class="panel">
    <h2>逐机发布结果</h2>
    <el-table :data="nodes"
      ><el-table-column prop="target_id" label="主机 ID" /><el-table-column
        label="任务"
        ><template #default="{ row }"
          ><router-link :to="`/deployments/${row.id}`"
            >#{{ row.id }}</router-link
          ></template
        ></el-table-column
      ><el-table-column label="状态"
        ><template #default="{ row }"
          ><StatusBadge :value="row.status" /></template></el-table-column
      ><el-table-column prop="result" label="结果"
    /></el-table>
  </section>
  <section class="panel">
    <h2>持久化日志</h2>
    <div class="code-block task-logs">
      <div v-for="log in logs" :key="log.id" class="task-line">
        <small
          >{{ new Date(log.created_at).toLocaleTimeString() }} ·
          {{ log.stage }}</small
        ><span>{{ log.message }}</span>
      </div>
      <p v-if="!logs.length" class="muted">等待任务输出…</p>
    </div>
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";
import { api, errorText } from "@/api/workspace";
import StatusBadge from "@/components/StatusBadge.vue";
const route = useRoute(),
  task = ref<any>({}),
  logs = ref<any[]>([]),
  error = ref(""),
  nodes = ref<any[]>([]);
let timer: ReturnType<typeof setTimeout>;
let alive = true;
async function load() {
  try {
    const d = await api(
      "GET",
      `/deployments/${route.params.id}?after=${logs.value[logs.value.length - 1]?.id || 0}`,
    );
    task.value = d.deployment;
    nodes.value = d.nodes || [];
    logs.value.push(...d.logs);
    error.value = "";
    if (
      alive &&
      (["queued", "running", "blocked"].includes(task.value.status) ||
        nodes.value.some((n) =>
          ["queued", "running", "blocked"].includes(n.status),
        ))
    )
      timer = setTimeout(load, 1500);
  } catch (e) {
    error.value = errorText(e);
    if (alive) timer = setTimeout(load, 5000);
  }
}
onMounted(load);
onUnmounted(() => {
  alive = false;
  clearTimeout(timer);
});
</script>
