<template>
  <div class="page-heading">
    <div>
      <h1>部署概览</h1>
      <p>镜像交付、发布结果与运行健康，一处确认。</p>
    </div>
    <el-button type="primary" @click="$router.push('/applications')"
      >发布应用</el-button
    >
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <div class="stats-grid">
    <router-link
      v-for="card in cards"
      :key="card.key"
      class="stat-card"
      :to="card.to"
      style="color: inherit; text-decoration: none"
      ><span class="muted">{{ card.title }}</span
      ><strong>{{ data.counts?.[card.key] ?? "—" }}</strong></router-link
    >
  </div>
  <section class="panel">
    <h2>应用环境</h2>
    <el-table :data="data.environments || []"
      ><el-table-column
        prop="application_id"
        label="应用 ID"
        width="100" /><el-table-column
        prop="name"
        label="部署环境" /><el-table-column
        prop="current_release_id"
        label="当前版本 ID" /><el-table-column label="目标主机"
        ><template #default="{ row }">{{
          row.target_id > 1 ? `主机 #${row.target_id}` : "本机"
        }}</template></el-table-column
      ><el-table-column label="健康"
        ><template #default="{ row }"
          ><StatusBadge :value="row.health" /></template></el-table-column
    ></el-table>
  </section>
  <section class="panel">
    <h2>从镜像到运行</h2>
    <p class="muted">
      选择仓库镜像、本机镜像或完整发布包，确认环境参数和差异后执行。生产环境默认手动发布。镜像导入与应用部署分别进行。
    </p>
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api, errorText } from "@/api/workspace";
import StatusBadge from "@/components/StatusBadge.vue";
const data = ref<any>({});
const error = ref("");
const cards = [
  { key: "applications", title: "应用", to: "/applications" },
  { key: "deployments", title: "部署记录", to: "/deployments" },
  { key: "bundles", title: "离线发布包", to: "/bundles" },
  { key: "image_events", title: "镜像提交", to: "/builtin" },
];
onMounted(async () => {
  try {
    data.value = await api("GET", "/overview");
  } catch (e) {
    error.value = errorText(e);
  }
});
</script>
