<template>
  <div class="page-heading">
    <div>
      <h1>仓库与维护</h1>
      <p>镜像内容、本机 Docker 缓存与离线材料分别持久化管理。</p>
    </div>
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <section class="panel">
    <h2>内置仓库状态</h2>
    <el-descriptions :column="2" border
      ><el-descriptions-item label="访问地址">{{
        status.public_url
      }}</el-descriptions-item
      ><el-descriptions-item label="引擎"
        >Distribution v3.1.2</el-descriptions-item
      ><el-descriptions-item label="镜像占用"
        >{{
          ((status.storage_bytes || 0) / 1024 / 1024).toFixed(1)
        }}
        MiB</el-descriptions-item
      ><el-descriptions-item label="维护状态">{{
        status.read_only ? "只读" : "可写"
      }}</el-descriptions-item></el-descriptions
    >
    <p class="muted" style="margin: 16px 0">
      访问地址及 TLS
      在安装配置中设置。维护前先停止推送；备份包含元数据、密钥、Registry
      内容及离线材料。
    </p>
    <div class="toolbar">
      <el-button @click="maintenance(!status.read_only)">{{
        status.read_only ? "结束只读维护" : "开启只读维护"
      }}</el-button
      ><el-button :disabled="!status.read_only" @click="gc"
        >清理未引用镜像层</el-button
      ><el-button :loading="busy" @click="backup">创建完整备份</el-button>
    </div>
    <p v-if="message">{{ message }}</p>
  </section>
  <section class="panel">
    <h2>完整备份任务</h2>
    <el-table :data="backups"
      ><el-table-column prop="id" label="任务" width="80" /><el-table-column
        label="状态"
        ><template #default="{ row }"
          ><StatusBadge :value="row.status" /></template></el-table-column
      ><el-table-column prop="file" label="完整备份文件" /><el-table-column
        prop="error"
        label="错误"
        show-overflow-tooltip
      /><el-table-column label="操作"
        ><template #default="{ row }"
          ><el-button
            link
            :disabled="row.status !== 'ready'"
            @click="download(`/system/backups/${row.id}/download`, row.file)"
            >下载</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
  <section class="panel">
    <h2>最近审计事件</h2>
    <el-table :data="events"
      ><el-table-column
        prop="created_at"
        label="时间"
        width="240" /><el-table-column
        prop="user_id"
        label="用户 ID"
        width="100" /><el-table-column
        prop="action"
        label="操作" /><el-table-column
        prop="resource"
        label="对象"
        show-overflow-tooltip
    /></el-table>
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { ElMessageBox } from "element-plus";
import { api, errorText, download } from "@/api/workspace";
import StatusBadge from "@/components/StatusBadge.vue";
const backups = ref<any[]>([]);
let timer: ReturnType<typeof setInterval>;
const status = ref<any>({}),
  events = ref<any[]>([]),
  error = ref(""),
  busy = ref(false),
  message = ref("");
async function load() {
  [status.value, events.value] = await Promise.all([
    api("GET", "/builtin/status"),
    api("GET", "/audit"),
  ]);
}
async function maintenance(v: boolean) {
  try {
    await api("POST", "/builtin/maintenance", { read_only: v });
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function gc() {
  try {
    await ElMessageBox.confirm(
      "将在只读窗口中运行垃圾回收。已保存的 manifest 及其引用会保留。",
      "仓库维护",
    );
    await api("POST", "/builtin/maintenance", { read_only: true, gc: true });
    await load();
  } catch (e: any) {
    if (e !== "cancel") error.value = errorText(e);
  }
}
async function backup() {
  busy.value = true;
  try {
    const r = await api("POST", "/system/backups");
    message.value = r.message;
    await load();
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
onMounted(() => {
  load().catch((e) => (error.value = errorText(e)));
  const update = () =>
    api("GET", "/system/backups")
      .then((r) => (backups.value = r))
      .catch((e) => (error.value = errorText(e)));
  update();
  timer = setInterval(update, 3000);
});
onUnmounted(() => clearInterval(timer));
</script>
