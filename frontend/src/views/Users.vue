<template>
  <div class="page-heading">
    <div>
      <h1>用户管理</h1>
      <p>账号共用于网页和 docker login；授权变更会使已有令牌失效。</p>
    </div>
    <el-button type="primary" @click="edit()">创建用户</el-button>
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <section class="panel">
    <el-table :data="rows"
      ><el-table-column prop="username" label="用户名" /><el-table-column
        label="角色"
        ><template #default="{ row }">{{
          row.is_admin ? "管理员" : "普通用户"
        }}</template></el-table-column
      ><el-table-column label="状态"
        ><template #default="{ row }"
          ><span
            :class="row.enabled ? 'state state-success' : 'state state-neutral'"
            >{{ row.enabled ? "启用" : "停用" }}</span
          ></template
        ></el-table-column
      ><el-table-column label="仓库权限"
        ><template #default="{ row }">{{
          row.is_admin ? "全部权限" : access[row.registry_access]
        }}</template></el-table-column
      ><el-table-column label="操作"
        ><template #default="{ row }"
          ><el-button link type="primary" @click="edit(row)"
            >修改授权 / 重设密码</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
  <el-dialog
    v-model="visible"
    :title="form.id ? '修改用户' : '创建用户'"
    width="540px"
    ><el-alert
      v-if="error"
      :title="error"
      type="error"
      class="inline-error"
      :closable="false"
    /><el-form label-position="top" @submit.prevent="save"
      ><el-form-item label="用户名"
        ><el-input
          v-model="form.username"
          :disabled="!!form.id" /></el-form-item
      ><el-form-item :label="form.id ? '重设密码（留空保留）' : '初始密码'"
        ><el-input
          v-model="form.password"
          type="password"
          show-password
          autocomplete="new-password" /></el-form-item
      ><el-form-item label="角色"
        ><el-switch
          v-model="form.is_admin"
          active-text="管理员"
          inactive-text="普通用户" /></el-form-item
      ><el-form-item label="仓库权限"
        ><el-select v-model="form.registry_access" :disabled="form.is_admin"
          ><el-option
            v-for="(v, k) in access"
            :key="k"
            :value="k"
            :label="v" /></el-select></el-form-item
      ><el-form-item v-if="form.id" label="账号状态"
        ><el-switch
          v-model="form.enabled"
          active-text="启用"
          inactive-text="停用" /></el-form-item
      ><el-button type="primary" native-type="submit" :loading="busy"
        >保存</el-button
      ></el-form
    ></el-dialog
  >
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api, errorText } from "@/api/workspace";
const rows = ref<any[]>([]),
  form = ref<any>({}),
  visible = ref(false),
  busy = ref(false),
  error = ref("");
const access: Record<string, string> = {
  none: "无权限",
  read_only: "只读",
  read_write: "读写",
};
function edit(row?: any) {
  form.value = row
    ? { ...row, password: "" }
    : {
        username: "",
        password: "",
        is_admin: false,
        registry_access: "none",
        enabled: true,
      };
  visible.value = true;
}
async function load() {
  rows.value = await api("GET", "/users");
}
async function save() {
  busy.value = true;
  try {
    await api(
      form.value.id ? "PATCH" : "POST",
      form.value.id ? `/users/${form.value.id}` : "/users",
      form.value,
    );
    visible.value = false;
    await load();
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
onMounted(() => load().catch((e) => (error.value = errorText(e))));
</script>
