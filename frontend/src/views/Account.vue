<template>
  <div class="page-heading">
    <div>
      <h1>个人账号</h1>
      <p>
        {{ auth.user?.username }} ·
        {{ auth.user?.is_admin ? "管理员" : "普通用户" }}
      </p>
    </div>
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <div class="split-grid">
    <section class="panel">
      <h2>修改密码</h2>
      <el-form label-position="top" @submit.prevent="password"
        ><el-form-item label="原密码"
          ><el-input
            v-model="oldPassword"
            type="password"
            show-password /></el-form-item
        ><el-form-item label="新密码"
          ><el-input
            v-model="newPassword"
            type="password"
            show-password /></el-form-item
        ><el-button native-type="submit" type="primary"
          >更新并重新登录</el-button
        ></el-form
      >
    </section>
    <section class="panel">
      <h2>自动化令牌</h2>
      <p class="muted">
        权限不能超过当前账号；令牌仅在创建时显示一次。撤权、改密或停用账号后失效。
      </p>
      <el-input
        v-model="name"
        placeholder="令牌用途"
        style="margin: 16px 0"
      /><el-checkbox-group v-model="scopes"
        ><el-checkbox value="pull">拉取</el-checkbox
        ><el-checkbox
          value="push"
          :disabled="
            !auth.user?.is_admin && auth.user?.registry_access !== 'read_write'
          "
          >推送</el-checkbox
        ><el-checkbox v-if="auth.user?.is_admin" value="deploy"
          >发布应用</el-checkbox
        ></el-checkbox-group
      ><el-button type="primary" style="margin-top: 16px" @click="createToken"
        >创建 30 天令牌</el-button
      >
      <pre v-if="secret" class="code-block" style="margin-top: 16px">{{
        secret
      }}</pre>
    </section>
  </div>
  <section class="panel">
    <h2>已签发令牌</h2>
    <el-table :data="tokens"
      ><el-table-column prop="name" label="名称" /><el-table-column
        prop="scopes"
        label="权限"
      /><el-table-column prop="expires_at" label="有效期" /><el-table-column
        label="状态"
        ><template #default="{ row }">{{
          row.revoked ? "已撤销" : "有效"
        }}</template></el-table-column
      ><el-table-column label="操作"
        ><template #default="{ row }"
          ><el-button
            link
            type="danger"
            :disabled="row.revoked"
            @click="revoke(row)"
            >撤销</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { api, errorText } from "@/api/workspace";
import { authApi } from "@/api/auth";
import { encryptPassword } from "@/utils/crypto";
import { useAuthStore } from "@/stores/auth";
const auth = useAuthStore(),
  router = useRouter(),
  error = ref(""),
  oldPassword = ref(""),
  newPassword = ref(""),
  tokens = ref<any[]>([]),
  scopes = ref<string[]>(["pull"]),
  name = ref(""),
  secret = ref("");
async function password() {
  try {
    await authApi.changePassword({
      old_password: encryptPassword(oldPassword.value),
      new_password: encryptPassword(newPassword.value),
    });
    auth.logout();
    router.push("/login");
  } catch (e) {
    error.value = errorText(e);
  }
}
async function load() {
  tokens.value = await api("GET", "/auth/tokens");
}
async function createToken() {
  try {
    const r = await api("POST", "/auth/tokens", {
      name: name.value,
      scopes: scopes.value,
      days: 30,
    });
    secret.value = r.token;
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function revoke(row: any) {
  await api("DELETE", `/auth/tokens/${row.id}`);
  await load();
}
onMounted(() => load().catch((e) => (error.value = errorText(e))));
</script>
