<template>
  <div class="page-heading">
    <div>
      <h1>{{ data.application?.name || "应用详情" }}</h1>
      <p>
        {{
          data.application?.description ||
          "独立管理各环境的版本、健康与发布授权。"
        }}
      </p>
    </div>
    <el-button
      type="primary"
      @click="$router.push(`/applications/${id}/deploy`)"
      >发布应用</el-button
    >
  </div>
  <el-alert v-if="error" :title="error" type="error" class="inline-error" />
  <section class="panel">
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">部署环境</h2>
      <el-button @click="edit()">添加环境</el-button>
    </div>
    <el-table :data="data.environments || []"
      ><el-table-column prop="name" label="环境" /><el-table-column
        label="目标主机"
        ><template #default="{ row }">{{
          row.target_id > 1 ? `#${row.target_id}` : "本机"
        }}</template></el-table-column
      ><el-table-column
        prop="current_release_id"
        label="当前版本"
      /><el-table-column label="健康"
        ><template #default="{ row }"
          ><StatusBadge :value="row.health" /></template></el-table-column
      ><el-table-column label="发布方式"
        ><template #default="{ row }">{{
          row.auto_deploy ? "自动（受推送人限制）" : "手动"
        }}</template></el-table-column
      ><el-table-column label="操作" width="220"
        ><template #default="{ row }"
          ><el-button link type="primary" @click="edit(row)">配置</el-button
          ><el-button link @click="clone(row)">复制</el-button
          ><el-button
            link
            @click="$router.push(`/deployments?environment_id=${row.id}`)"
            >记录</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
  <section class="panel">
    <h2>发布版本</h2>
    <el-table :data="data.releases || []"
      ><el-table-column prop="version" label="版本" /><el-table-column
        prop="source"
        label="来源"
        width="100"
      /><el-table-column prop="platform" label="平台" /><el-table-column
        prop="created_at"
        label="创建时间"
      /><el-table-column label="操作" width="190"
        ><template #default="{ row }"
          ><el-button
            link
            type="primary"
            @click="
              $router.push(`/applications/${id}/deploy?release=${row.id}`)
            "
            >晋升 / 恢复</el-button
          ><el-button link @click="inspect = row">查看</el-button
          ><el-button
            v-if="row.source === 'bundle'"
            link
            type="primary"
            @click="publishBundle(row)"
            >镜像入库</el-button
          ></template
        ></el-table-column
      ></el-table
    >
  </section>
  <el-dialog v-model="visible" title="环境配置" width="680px"
    ><el-alert
      v-if="error"
      :title="error"
      type="error"
      class="inline-error"
      :closable="false"
    /><el-form label-position="top"
      ><el-form-item label="环境名称"
        ><el-input
          v-model="form.name"
          placeholder="test / prod" /></el-form-item
      ><el-form-item label="目标主机"
        ><el-select v-model="form.target_id"
          ><el-option
            v-for="h in hosts"
            :key="h.id"
            :value="h.id"
            :label="h.name" /></el-select></el-form-item
      ><el-form-item label="固定主机组（按选择顺序逐台部署，失败暂停）"
        ><el-select v-model="targetIds" multiple
          ><el-option
            v-for="h in hosts"
            :key="h.id"
            :value="h.id"
            :label="h.name" /></el-select></el-form-item
      ><el-form-item
        label="环境参数（JSON，仅保存在本环境，不会被发布包默认值覆盖）"
        ><el-input
          v-model="params"
          type="textarea"
          :rows="5"
          class="mono" /></el-form-item
      ><el-form-item
        ><el-checkbox v-model="form.stateless">无状态应用</el-checkbox
        ><el-checkbox v-model="form.auto_rollback" :disabled="!form.stateless"
          >失败自动恢复</el-checkbox
        ></el-form-item
      ><el-form-item
        ><el-switch
          v-model="form.auto_deploy"
          active-text="允许镜像提交后自动发布"
          aria-label="允许镜像提交后自动发布" /></el-form-item
      ><template v-if="form.auto_deploy"
        ><el-alert
          title="默认仅管理员推送可触发；普通用户需在下方明确授权。生产环境建议手动发布。"
          type="warning"
          :closable="false" /><el-form-item label="来源仓库"
          ><el-select v-model="form.registry_id"
            ><el-option
              v-for="r in registries"
              :key="r.id"
              :value="r.id"
              :label="r.name" /></el-select></el-form-item
        ><el-form-item label="镜像名称（不含仓库域名）"
          ><el-input
            v-model="form.repository"
            placeholder="team/gateway" /></el-form-item
        ><el-form-item label="标签规则"
          ><el-input
            v-model="form.tag_pattern"
            placeholder="* / test-*" /></el-form-item
        ><el-form-item label="允许触发的普通用户"
          ><el-select v-model="publishers" multiple
            ><el-option
              v-for="u in users.filter((u) => !u.is_admin)"
              :key="u.id"
              :value="u.id"
              :label="
                u.username
              " /></el-select></el-form-item></template></el-form
    ><template #footer
      ><el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="save">保存环境</el-button></template
    ></el-dialog
  >
  <el-dialog
    :model-value="!!inspect"
    title="不可变版本"
    width="800px"
    @close="inspect = null"
  >
    <pre class="code-block">{{ inspect }}</pre>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute } from "vue-router";
import request from "@/api/request";
import { api, errorText } from "@/api/workspace";
import StatusBadge from "@/components/StatusBadge.vue";
const id = useRoute().params.id,
  data = ref<any>({}),
  error = ref(""),
  visible = ref(false),
  form = ref<any>({}),
  params = ref("{}"),
  publishers = ref<number[]>([]),
  users = ref<any[]>([]),
  registries = ref<any[]>([]),
  hosts = ref<any[]>([]),
  inspect = ref<any>(null),
  targetIds = ref<number[]>([]);
async function load() {
  try {
    data.value = await api("GET", `/applications/${id}`);
  } catch (e) {
    error.value = errorText(e);
  }
}
function edit(row?: any) {
  form.value = row
    ? { ...row }
    : {
        name: "test",
        target_id: 1,
        stateless: false,
        auto_rollback: false,
        auto_deploy: false,
        registry_id: 0,
        repository: "",
        tag_pattern: "*",
      };
  targetIds.value = JSON.parse(row?.target_ids || "[]");
  params.value = JSON.stringify(row?.parameters || {}, null, 2);
  publishers.value = JSON.parse(row?.allowed_publishers || "[]");
  visible.value = true;
}
function clone(row: any) {
  edit({ ...row, id: 0, name: row.name + "-copy", auto_deploy: false });
}
async function save() {
  try {
    form.value.target_ids = JSON.stringify(targetIds.value);
    form.value.parameters = JSON.parse(params.value);
    form.value.allowed_publishers = JSON.stringify(publishers.value);
    await api("PUT", `/applications/${id}/environments`, form.value);
    visible.value = false;
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
async function publishBundle(row: any) {
  try {
    await request.post(
      "/bundles/publish",
      { release_id: row.id },
      { timeout: 0 },
    );
    await load();
  } catch (e) {
    error.value = errorText(e);
  }
}
onMounted(async () => {
  await load();
  const results = await Promise.allSettled([
    api("GET", "/users"),
    api("GET", "/registries"),
    api("GET", "/targets"),
  ]);
  users.value = results[0].status === "fulfilled" ? results[0].value : [];
  registries.value = results[1].status === "fulfilled" ? results[1].value : [];
  hosts.value =
    results[2].status === "fulfilled"
      ? results[2].value
      : [{ id: 1, name: "本机" }];
});
</script>
