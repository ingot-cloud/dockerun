<template>
  <div class="page-heading">
    <div>
      <h1>部署记录</h1>
      <p>任务独立于浏览器运行，刷新后可继续查看。</p>
    </div>
    <el-button @click="load">刷新</el-button>
  </div>
  <el-alert v-if="error" :title="error" type="error" />
  <section class="panel">
    <el-table :data="rows"
      ><el-table-column label="任务" width="90"
        ><template #default="{ row }"
          ><router-link :to="`/deployments/${row.id}`"
            >#{{ row.id }}</router-link
          ></template
        ></el-table-column
      ><el-table-column prop="environment_id" label="环境 ID" /><el-table-column
        prop="release_id"
        label="版本 ID" /><el-table-column label="状态"
        ><template #default="{ row }"
          ><StatusBadge :value="row.status" /></template></el-table-column
      ><el-table-column prop="stage" label="当前阶段" /><el-table-column
        label="触发"
        ><template #default="{ row }"
          >{{ row.automatic ? "镜像提交" : "手动" }} · 用户 #{{
            row.user_id
          }}</template
        ></el-table-column
      ><el-table-column
        prop="result"
        label="结果"
        min-width="220"
        show-overflow-tooltip /><el-table-column
        prop="created_at"
        label="时间"
        width="230"
    /></el-table>
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";
import { api, errorText } from "@/api/workspace";
import StatusBadge from "@/components/StatusBadge.vue";
const rows = ref<any[]>([]),
  error = ref(""),
  route = useRoute();
let timer: ReturnType<typeof setInterval>;
async function load() {
  try {
    rows.value = await api(
      "GET",
      "/deployments" +
        (route.query.environment_id
          ? `?environment_id=${route.query.environment_id}`
          : ""),
    );
  } catch (e) {
    error.value = errorText(e);
  }
}
onMounted(() => {
  load();
  timer = setInterval(load, 5000);
});
onUnmounted(() => clearInterval(timer));
</script>
