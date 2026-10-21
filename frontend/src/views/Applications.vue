<template>
  <div class="page-heading">
    <div>
      <h1>应用</h1>
      <p>应用定义服务，环境保存目标、参数及发布策略。</p>
    </div>
    <el-button type="primary" @click="visible = true">创建应用</el-button>
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <section class="panel">
    <div class="toolbar">
      <el-input
        v-model="search"
        placeholder="搜索应用"
        clearable
        style="max-width: 320px"
      /><el-button @click="load">刷新</el-button>
    </div>
    <el-table
      :data="filtered"
      v-loading="loading"
      @row-click="(row: any) => $router.push(`/applications/${row.id}`)"
      ><el-table-column label="应用名称"
        ><template #default="{ row }"
          ><router-link :to="`/applications/${row.id}`">{{
            row.name
          }}</router-link></template
        ></el-table-column
      ><el-table-column prop="description" label="说明" /><el-table-column
        prop="kind"
        label="部署方式"
        width="150" /><el-table-column
        prop="created_at"
        label="创建时间"
        width="240" /></el-table
    ><el-empty
      v-if="!loading && !rows.length"
      description="创建一个应用，开始在线或离线发布"
    />
  </section>
  <el-dialog v-model="visible" title="创建应用" width="520px"
    ><el-alert
      v-if="error"
      :title="error"
      type="error"
      class="inline-error"
      :closable="false"
    /><el-form label-position="top" @submit.prevent="create"
      ><el-form-item label="应用名称"
        ><el-input v-model="form.name" maxlength="100" /></el-form-item
      ><el-form-item label="说明"
        ><el-input v-model="form.description" type="textarea" /></el-form-item
      ><el-button type="primary" native-type="submit" :loading="saving"
        >创建并配置环境</el-button
      ></el-form
    ></el-dialog
  >
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { api, errorText } from "@/api/workspace";
const router = useRouter(),
  rows = ref<any[]>([]),
  search = ref(""),
  visible = ref(false),
  error = ref(""),
  loading = ref(false),
  saving = ref(false),
  form = ref({ name: "", description: "" });
const filtered = computed(() =>
  rows.value.filter((r) => r.name.includes(search.value)),
);
async function load() {
  loading.value = true;
  try {
    rows.value = await api("GET", "/applications");
  } catch (e) {
    error.value = errorText(e);
  } finally {
    loading.value = false;
  }
}
async function create() {
  saving.value = true;
  try {
    const a = await api("POST", "/applications", form.value);
    router.push(`/applications/${a.id}`);
  } catch (e) {
    error.value = errorText(e);
  } finally {
    saving.value = false;
  }
}
onMounted(load);
</script>
