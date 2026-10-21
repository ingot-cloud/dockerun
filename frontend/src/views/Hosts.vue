<template>
  <div class="page-heading">
    <div>
      <h1>目标主机</h1>
      <p>单机模式可独立使用；远端 Agent 主动连接控制台获取本机任务。</p>
    </div>
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <section class="panel">
    <el-form inline @submit.prevent="create"
      ><el-form-item label="主机名称"><el-input v-model="name" /></el-form-item
      ><el-button type="primary" native-type="submit"
        >创建 Agent 凭据</el-button
      ></el-form
    >
    <pre v-if="credential" class="code-block">{{ credential }}</pre>
  </section>
  <section class="panel">
    <el-table :data="rows"
      ><el-table-column prop="name" label="主机" /><el-table-column
        prop="platform"
        label="平台"
      /><el-table-column label="连接状态"
        ><template #default="{ row }">{{
          row.id === 1
            ? "本机"
            : row.last_seen_at &&
                Date.now() - Date.parse(row.last_seen_at) < 90000
              ? "在线"
              : "离线"
        }}</template></el-table-column
      ><el-table-column prop="last_seen_at" label="最近连接" /><el-table-column
        label="操作"
        ><template #default="{ row }"
          ><el-button
            v-if="row.id !== 1"
            link
            type="danger"
            @click="disable(row)"
            >停用凭据</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api, errorText } from "@/api/workspace";
const rows = ref<any[]>([]),
  name = ref(""),
  credential = ref(""),
  error = ref("");
async function load() {
  rows.value = await api("GET", "/targets");
}
async function create() {
  try {
    const r = await api("POST", "/targets", { name: name.value });
    credential.value = `DOCKERUN_URL=${location.origin}\nDOCKERUN_AGENT_TOKEN=${r.token}\ndockerun agent --data-dir /var/lib/dockerun-agent`;
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function disable(row: any) {
  await api("PATCH", `/targets/${row.id}`, { enabled: false });
  await load();
}
onMounted(() => load().catch((e) => (error.value = errorText(e))));
</script>
