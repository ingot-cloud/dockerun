<template>
  <div class="page-heading">
    <div>
      <h1>内置镜像仓库</h1>
      <p>使用 Dockerun 账号登录，权限由管理员统一分配。</p>
    </div>
    <el-button @click="load">刷新</el-button>
  </div>
  <el-alert
    v-if="error"
    :title="error"
    :type="
      auth.user?.registry_access === 'none' && !auth.user?.is_admin
        ? 'warning'
        : 'error'
    "
    :closable="false"
    class="inline-error"
  /><template v-if="loaded"
    ><div class="split-grid">
      <section class="panel">
        <h2>仓库镜像</h2>
        <div class="toolbar">
          <el-input v-model="search" placeholder="搜索镜像名称" clearable />
        </div>
        <el-table
          :data="repos.filter((r) => r.includes(search))"
          @row-click="(name: string) => tags(name)"
          ><el-table-column label="镜像名称"
            ><template #default="{ row }"
              ><el-button
                link
                type="primary"
                class="mono"
                @click.stop="tags(row)"
                >{{ row }}</el-button
              ></template
            ></el-table-column
          ></el-table
        ><el-empty v-if="!repos.length" description="仓库中还没有镜像" />
      </section>
      <section class="panel">
        <h2>客户端使用</h2>
        <p class="muted">
          当前权限：{{
            auth.user?.is_admin
              ? "管理员"
              : auth.user?.registry_access === "read_write"
                ? "读写"
                : "只读"
          }}
        </p>
        <pre class="code-block" style="margin-top: 16px">
docker login {{ host }}
docker pull {{ host }}/team/app:v1</pre
        >
        <pre
          v-if="
            auth.user?.is_admin || auth.user?.registry_access === 'read_write'
          "
          class="code-block"
          style="margin-top: 12px"
        >
docker tag app:v1 {{ host }}/team/app:v1
docker push {{ host }}/team/app:v1</pre
        >
        <el-alert
          title="普通用户推送默认只生成镜像版本。自动部署需由管理员为指定环境授权。"
          type="info"
          :closable="false"
          style="margin-top: 16px"
        />
      </section>
    </div>
    <section v-if="selected" class="panel">
      <h2>{{ selected }} · 标签</h2>
      <div class="toolbar">
        <el-button v-for="tag in tagList" :key="tag" @click="manifest(tag)">{{
          tag
        }}</el-button>
      </div>
      <el-button
        v-if="detail?.digest && auth.user?.is_admin"
        type="danger"
        plain
        style="margin-bottom: 16px"
        @click="removeManifest"
        >删除未受保护版本</el-button
      >
      <pre v-if="detail" class="code-block">{{ detail }}</pre>
    </section>
    <section class="panel">
      <h2>最近镜像提交</h2>
      <el-table :data="data.events || []"
        ><el-table-column prop="repository" label="镜像" /><el-table-column
          prop="reference"
          label="标签 / 引用" /><el-table-column
          prop="digest"
          label="digest"
          show-overflow-tooltip
          class-name="mono" /><el-table-column
          prop="user_id"
          label="发布人 ID"
          width="120" /><el-table-column
          prop="created_at"
          label="时间"
          width="240"
      /></el-table></section
  ></template>
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessageBox } from "element-plus";
import { api, errorText } from "@/api/workspace";
import { useAuthStore } from "@/stores/auth";
const auth = useAuthStore(),
  data = ref<any>({}),
  repos = ref<string[]>([]),
  error = ref(""),
  loaded = ref(false),
  search = ref(""),
  selected = ref(""),
  tagList = ref<string[]>([]),
  detail = ref<any>(null);
const host = computed(
  () =>
    data.value.public_url?.replace(/^https?:\/\//, "").replace(/\/$/, "") ||
    location.host,
);
async function load() {
  try {
    data.value = await api("GET", "/builtin/catalog");
    repos.value = data.value.catalog?.repositories || [];
    loaded.value = true;
    error.value = "";
  } catch (e) {
    error.value = errorText(e);
  }
}
async function tags(name: string) {
  selected.value = name;
  detail.value = null;
  try {
    const r = await api(
      "GET",
      `/builtin/tags?name=${encodeURIComponent(name)}`,
    );
    tagList.value = r.tags || [];
  } catch (e) {
    error.value = errorText(e);
  }
}
async function manifest(tag: string) {
  try {
    detail.value = await api(
      "GET",
      `/builtin/tags?name=${encodeURIComponent(selected.value)}&reference=${encodeURIComponent(tag)}`,
    );
  } catch (e) {
    error.value = errorText(e);
  }
}
async function removeManifest() {
  try {
    await ElMessageBox.confirm(
      `删除 ${selected.value}@${detail.value.digest}？当前及保留版本受保护。`,
      "删除镜像版本",
      { type: "warning" },
    );
    await api("DELETE", "/builtin/manifests", {
      name: selected.value,
      digest: detail.value.digest,
    });
    await tags(selected.value);
    await load();
  } catch (e) {
    if (e !== "cancel") error.value = errorText(e);
  }
}
onMounted(load);
</script>
