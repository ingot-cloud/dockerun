<template>
  <div class="page-heading">
    <div>
      <h1>发布包</h1>
      <p>完整材料导入后成为待发布版本，随后选择环境部署。</p>
    </div>
  </div>
  <section class="panel">
    <h2>上传交付材料</h2>
    <el-radio-group v-model="kind" style="margin-bottom: 16px"
      ><el-radio-button value="bundle">完整发布包</el-radio-button
      ><el-radio-button value="image"
        >普通镜像归档</el-radio-button
      ></el-radio-group
    >
    <p class="muted">
      支持 .tar.gz；按 SHA-256 标识上传，重复选择同一文件可从已上传偏移继续。
    </p>
    <div class="toolbar" style="margin-top: 16px">
      <input
        type="file"
        accept=".tar,.tar.gz,.tgz"
        aria-label="选择完整发布包"
        :disabled="busy"
        @change="select"
      /><el-button
        type="primary"
        :disabled="!file"
        :loading="busy"
        @click="upload"
        >{{ busy ? "正在上传" : "校验并导入" }}</el-button
      >
    </div>
    <el-progress v-if="progress !== null" :percentage="progress" />
    <p class="muted">{{ stage }}</p>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      class="inline-error"
      :closable="false"
    /><el-divider /><el-form inline @submit.prevent="importDirectory"
      ><el-form-item label="服务器导入目录中的文件"
        ><el-input
          v-model="serverFile"
          placeholder="application-release.tar.gz"
          style="width: 300px" /></el-form-item
      ><el-button native-type="submit" :disabled="!serverFile"
        >导入服务器文件</el-button
      ></el-form
    >
  </section>
  <section class="panel">
    <h2>导入记录</h2>
    <el-table :data="rows"
      ><el-table-column prop="id" label="#" width="65" /><el-table-column
        prop="name"
        label="发布包"
      /><el-table-column label="状态"
        ><template #default="{ row }"
          ><StatusBadge :value="row.status" /></template></el-table-column
      ><el-table-column label="本机镜像" min-width="160"
        ><template #default="{ row }"
          ><span>{{ row.local_loaded ? "已加载" : "由目标节点确认" }}</span>
          <p v-if="row.local_load_note" class="muted">
            {{ row.local_load_note }}
          </p></template
        ></el-table-column
      ><el-table-column
        prop="sha256"
        label="SHA-256"
        show-overflow-tooltip
        class-name="mono"
      /><el-table-column
        prop="error"
        label="错误"
        show-overflow-tooltip
      /><el-table-column label="操作" width="210"
        ><template #default="{ row }"
          ><el-button
            link
            type="primary"
            :disabled="row.status !== 'ready' || row.kind === 'image'"
            @click="createVersion(row)"
            >创建待发布版本</el-button
          ><el-button v-if="row.status === 'failed'" link @click="retry(row)"
            >重试导入</el-button
          ><el-button
            link
            @click="download(`/bundles/${row.id}/download`, row.name)"
            >下载</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { api, errorText, download } from "@/api/workspace";
import request from "@/api/request";
import CryptoJS from "crypto-js";
import StatusBadge from "@/components/StatusBadge.vue";
const rows = ref<any[]>([]),
  file = ref<File>(),
  busy = ref(false),
  progress = ref<number | null>(null),
  stage = ref(""),
  error = ref(""),
  serverFile = ref(""),
  kind = ref("bundle"),
  router = useRouter();
let timer: ReturnType<typeof setInterval>;
function select(e: Event) {
  file.value = (e.target as HTMLInputElement).files?.[0];
  progress.value = null;
}
async function load() {
  try {
    rows.value = await api("GET", "/bundles");
  } catch (e) {
    error.value = errorText(e);
  }
}
async function upload() {
  if (!file.value) return;
  busy.value = true;
  error.value = "";
  try {
    stage.value = "计算文件校验值…";
    const hasher = CryptoJS.algo.SHA256.create();
    for (let start = 0; start < file.value.size; start += 8 * 1024 * 1024) {
      const bytes = new Uint8Array(
        await file.value.slice(start, start + 8 * 1024 * 1024).arrayBuffer(),
      );
      hasher.update(CryptoJS.lib.WordArray.create(bytes as any));
    }
    const hash = hasher.finalize().toString();
    let { offset } = await api("GET", `/bundles/uploads/${hash}`);
    while (offset < file.value.size) {
      stage.value = "上传材料（可续传）";
      const end = Math.min(offset + 8 * 1024 * 1024, file.value.size);
      const chunk = file.value.slice(offset, end);
      const r: any = await request.put(`/bundles/uploads/${hash}`, chunk, {
        headers: {
          "Content-Type": "application/octet-stream",
          "Content-Range": `bytes ${offset}-${end - 1}/${file.value.size}`,
        },
        timeout: 120000,
      });
      offset = r.offset;
      progress.value = Math.floor((offset / file.value.size) * 100);
    }
    await api("POST", `/bundles/uploads/${hash}/complete`, {
      kind: kind.value,
      name: file.value.name,
    });
    stage.value = "上传完成，服务器正在校验和加载镜像；在下方查看导入结果。";
    await load();
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
async function importDirectory() {
  try {
    await api("POST", "/bundles/import-directory", { file: serverFile.value, kind: kind.value });
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function retry(row: any) {
  try {
    await api("POST", `/bundles/${row.id}/retry`);
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function createVersion(b: any) {
  try {
    const r = await api("POST", `/bundles/${b.id}/version`, {});
    router.push(`/applications/${r.release.application_id}`);
  } catch (e) {
    error.value = errorText(e);
  }
}
onMounted(() => {
  load();
  timer = setInterval(load, 3000);
});
onUnmounted(() => clearInterval(timer));
</script>
