<template>
  <PageContainer title="外部仓库管理">
    <ContentCard title="外部仓库" :icon="DataBoard" :no-padding="true">
      <template #actions>
        <el-input
          v-model="searchText"
          placeholder="搜索…"
          :prefix-icon="Search"
          style="width: 240px"
          clearable
        />
        <el-button
          type="danger"
          :icon="Delete"
          @click="handleBatchRemove"
          :disabled="selectedItems.length === 0"
        >
          删除
        </el-button>
        <el-button type="primary" :icon="Plus" @click="showAddDialog">
          Add Registry
        </el-button>
      </template>

      <DataTable
        :data="filteredRegistries"
        v-loading="loading"
        :total="filteredRegistries.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="名称" min-width="180">
          <template #default="{ row }">
            <span class="registry-name" @click="viewImages(row)">
              {{ row.name }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="URL" min-width="300">
          <template #default="{ row }">
            <code class="code-text">{{ row.url }}</code>
          </template>
        </el-table-column>

        <el-table-column label="用户名" width="150">
          <template #default="{ row }">
            <span class="text-normal">{{ row.username || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Default" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="success" size="small">
              Yes
            </el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="Insecure" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.insecure" type="warning" size="small">
              Yes
            </el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column
          label="快捷操作"
          width="150"
          align="center"
          fixed="right"
        >
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="View Images" placement="top">
                <el-button
                  link
                  :icon="View"
                  @click="viewImages(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Edit" placement="top">
                <el-button
                  link
                  :icon="Edit"
                  @click="editRegistry(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  link
                  :icon="Delete"
                  @click="deleteRegistry(row)"
                  class="action-btn danger-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>

    <!-- 添加/编辑对话框 -->
    <FormDialog
      v-model="dialogVisible"
      :title="dialogMode === 'add' ? 'Create Registry' : 'Edit Registry'"
      :icon="DataBoard"
      :loading="submitting"
      @confirm="submitForm"
      @cancel="dialogVisible = false"
      wide
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px">
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="e.g. My Registry" />
          </el-form-item>
          <el-form-item label="仓库地址" prop="url">
            <el-input
              v-model="form.url"
              placeholder="https://registry.example.com"
            />
            <div class="form-tip">
              Full URL of the Docker registry (e.g.,
              https://registry-1.docker.io for Docker Hub)
            </div>
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Authentication</h3>
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="Optional" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="Optional"
              show-password
            />
            <div class="form-tip">通过 HTTPS 传输，并在服务器加密保存</div>
          </el-form-item>
          <el-form-item label="Email">
            <el-input v-model="form.email" placeholder="Optional" />
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Agent 专用拉取账号</h3>
          <p class="form-tip">
            在外部 Registry 创建只读账号后填写；Agent 不使用上面的管理凭据。
          </p>
          <el-form-item label="拉取用户名"
            ><el-input v-model="form.pull_username" /></el-form-item
          ><el-form-item label="拉取密码"
            ><el-input
              v-model="form.pull_password"
              type="password"
              show-password
              placeholder="留空保留已有凭据" /></el-form-item
          ><el-form-item label="权限确认"
            ><el-checkbox v-model="form.pull_only_confirmed"
              >已在外部 Registry 将该账号限制为只读</el-checkbox
            ></el-form-item
          >
        </div>
        <div class="form-section">
          <h3>Options</h3>
          <el-form-item label="Default Registry">
            <el-switch v-model="form.is_default" />
            <div class="form-tip">
              Set as default registry for image operations
            </div>
          </el-form-item>
          <el-form-item label="Insecure">
            <el-switch v-model="form.insecure" />
            <div class="form-tip">
              Allow insecure HTTPS connections (not recommended for production)
            </div>
          </el-form-item>
        </div>
      </el-form>
    </FormDialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Plus,
  View,
  Edit,
  Delete,
  DataBoard,
  Search,
} from "@element-plus/icons-vue";
import { registryApi, type Registry } from "@/api/registry";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const router = useRouter();
const loading = ref(false);
const registries = ref<Registry[]>([]);
const searchText = ref("");
const selectedItems = ref<Registry[]>([]);
const dialogVisible = ref(false);
const dialogMode = ref<"add" | "edit">("add");
const submitting = ref(false);
const formRef = ref();

const form = ref({
  id: 0,
  name: "",
  url: "",
  pull_username: "",
  pull_password: "",
  pull_only_confirmed: false,
  username: "",
  password: "",
  email: "",
  is_default: false,
  insecure: false,
});

const rules = {
  name: [{ required: true, message: "Please enter a name", trigger: "blur" }],
  url: [{ required: true, message: "Please enter a URL", trigger: "blur" }],
};

const filteredRegistries = computed(() => {
  if (!searchText.value) {
    return registries.value;
  }
  const search = searchText.value.toLowerCase();
  return registries.value.filter(
    (item) =>
      item.name.toLowerCase().includes(search) ||
      item.url.toLowerCase().includes(search) ||
      (item.username && item.username.toLowerCase().includes(search)),
  );
});

const fetchRegistries = async () => {
  loading.value = true;
  try {
    const response = await registryApi.list();
    registries.value = (response || []).filter(
      (r: any) => r.kind !== "builtin",
    );
  } catch (error: any) {
    console.error("Failed to fetch registries:", error);
    ElMessage.error(
      error.response?.data?.error || "Failed to fetch registry list",
    );
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (val: Registry[]) => {
  selectedItems.value = val;
};

const handleBatchRemove = async () => {
  if (selectedItems.value.length === 0) {
    ElMessage.warning("Please select registries to remove");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedItems.value.length} registry(ies)?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      },
    );

    for (const item of selectedItems.value) {
      await registryApi.delete(item.id);
    }

    ElMessage.success("Registries removed successfully");
    selectedItems.value = [];
    fetchRegistries();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove registries",
      );
    }
  }
};

const showAddDialog = () => {
  dialogMode.value = "add";
  form.value = {
    id: 0,
    name: "",
    url: "",
    pull_username: "",
    pull_password: "",
    pull_only_confirmed: false,
    username: "",
    password: "",
    email: "",
    is_default: false,
    insecure: false,
  };
  dialogVisible.value = true;
};

const editRegistry = (row: Registry) => {
  dialogMode.value = "edit";
  form.value = {
    id: row.id,
    name: row.name,
    url: row.url,
    pull_username: row.pull_username || "",
    pull_password: "",
    pull_only_confirmed: row.pull_only_confirmed || false,
    username: row.username,
    password: "",
    email: row.email || "",
    is_default: row.is_default,
    insecure: row.insecure,
  };
  dialogVisible.value = true;
};

const submitForm = async () => {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
  } catch {
    return;
  }

  submitting.value = true;
  try {
    if (dialogMode.value === "add") {
      await registryApi.create(form.value);
      ElMessage.success("Registry created successfully");
    } else {
      await registryApi.update(form.value.id, form.value);
      ElMessage.success("Registry updated successfully");
    }
    dialogVisible.value = false;
    await fetchRegistries();
  } catch (error: any) {
    console.error("Failed to submit form:", error);
    ElMessage.error(error.response?.data?.error || "Operation failed");
  } finally {
    submitting.value = false;
  }
};

const deleteRegistry = async (row: Registry) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove registry "${row.name}"?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      },
    );
    await registryApi.delete(row.id);
    ElMessage.success("Registry removed successfully");
    fetchRegistries();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove registry",
      );
    }
  }
};

const viewImages = (row: Registry) => {
  router.push(`/registry/${row.id}/images`);
};

const formatDate = (date: string) => {
  if (!date) return "-";
  return new Date(date).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

onMounted(() => {
  fetchRegistries();
});
</script>

<style scoped>
.registry-name {
  color: var(--color-primary);
  cursor: pointer;
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
}

.registry-name:hover {
  color: #79bbff;
  text-decoration: underline;
}
</style>
