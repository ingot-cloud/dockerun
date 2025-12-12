<template>
  <div class="registry-view">
    <div class="page-header">
      <h1>Registry list</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><DataBoard /></el-icon>
          <span>Registries</span>
        </div>
        <div class="header-right">
          <el-input
            v-model="searchText"
            placeholder="Search..."
            :prefix-icon="Search"
            class="search-input"
            clearable
          />
          <el-button
            type="danger"
            :icon="Delete"
            @click="handleBatchRemove"
            :disabled="selectedItems.length === 0"
          >
            Remove
          </el-button>
          <el-button type="primary" :icon="Plus" @click="showAddDialog">
            Add registry
          </el-button>
        </div>
      </div>

      <el-table
        :data="filteredRegistries"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="Name" min-width="180">
          <template #default="{ row }">
            <span class="registry-name" @click="viewImages(row)">{{
              row.name
            }}</span>
          </template>
        </el-table-column>

        <el-table-column label="URL" min-width="300">
          <template #default="{ row }">
            <span class="text-normal">{{ row.url }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Username" width="150">
          <template #default="{ row }">
            <span class="text-normal">{{ row.username || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Default" width="100" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.is_default"
              type="success"
              size="small"
              class="default-tag"
            >
              Yes
            </el-tag>
            <span v-else class="text-normal">-</span>
          </template>
        </el-table-column>

        <el-table-column label="Insecure" width="100" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.insecure"
              type="warning"
              size="small"
              class="insecure-tag"
            >
              Yes
            </el-tag>
            <span v-else class="text-normal">-</span>
          </template>
        </el-table-column>

        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column
          label="Quick Actions"
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
              <el-tooltip content="Remove" placement="top">
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
      </el-table>

      <div class="table-footer">
        <span class="total-items">{{ filteredRegistries.length }} items</span>
      </div>
    </div>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'add' ? 'Create registry' : 'Edit registry'"
      width="700px"
      class="create-dialog"
      :close-on-click-modal="false"
    >
      <el-form
        :model="form"
        :rules="rules"
        ref="formRef"
        label-width="140px"
        class="registry-form"
      >
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="Name" prop="name">
            <el-input v-model="form.name" placeholder="e.g. My Registry" />
          </el-form-item>
          <el-form-item label="Registry URL" prop="url">
            <el-input
              v-model="form.url"
              placeholder="https://registry.example.com"
            />
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Authentication</h3>
          <el-form-item label="Username">
            <el-input v-model="form.username" placeholder="Optional" />
          </el-form-item>
          <el-form-item label="Password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="Optional"
              show-password
            />
          </el-form-item>
          <el-form-item label="Email">
            <el-input v-model="form.email" placeholder="Optional" />
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Options</h3>
          <el-form-item label="Default Registry">
            <el-switch v-model="form.is_default" />
            <span class="form-tip"
              >Set as default registry for image operations</span
            >
          </el-form-item>
          <el-form-item label="Insecure">
            <el-switch v-model="form.insecure" />
            <span class="form-tip">Allow insecure HTTPS connections</span>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">
          {{ dialogMode === "add" ? "Create" : "Update" }}
        </el-button>
      </template>
    </el-dialog>
  </div>
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
      (item.username && item.username.toLowerCase().includes(search))
  );
});

const fetchRegistries = async () => {
  loading.value = true;
  try {
    const response = await registryApi.list();
    registries.value = response || [];
  } catch (error: any) {
    console.error("Failed to fetch registries:", error);
    ElMessage.error(
      error.response?.data?.error || "Failed to fetch registry list"
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
      "Confirm removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    for (const item of selectedItems.value) {
      await registryApi.delete(item.id);
    }

    ElMessage.success("Registries removed successfully");
    fetchRegistries();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove registries"
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
    await fetchRegistries(); // 重新获取列表
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
      "Confirm removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );
    await registryApi.delete(row.id);
    ElMessage.success("Registry removed successfully");
    fetchRegistries();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove registry"
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
.registry-view {
  padding: 20px;
  background-color: #1b1d21;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 400;
  color: #fff;
}

.content-card {
  background-color: #23252a;
  border-radius: 4px;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background-color: #2a2d31;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
}

.header-icon {
  font-size: 20px;
  color: #4a9eff;
}

.header-right {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-input {
  width: 280px;
}

.search-input :deep(.el-input__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.search-input :deep(.el-input__inner) {
  color: #fff;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: #8a929c;
}

.data-table {
  width: 100%;
}

.data-table :deep(.el-table__header-wrapper) {
  background-color: #2a2d31;
}

.data-table :deep(.el-table__header th) {
  background-color: #2a2d31 !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
  padding: 12px 0;
}

.data-table :deep(.el-table__body-wrapper) {
  background-color: #23252a;
}

.data-table :deep(.el-table__body) {
  background-color: #23252a;
  color: #fff;
}

.data-table :deep(.el-table__row) {
  background-color: #23252a !important;
}

.data-table :deep(.el-table__row:hover > td) {
  background-color: rgba(74, 158, 255, 0.08) !important;
}

.data-table :deep(.el-table td) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background-color: transparent !important;
}

.data-table :deep(.el-checkbox__inner) {
  background-color: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.data-table :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #4a9eff;
  border-color: #4a9eff;
}

.registry-name {
  color: #4a9eff;
  cursor: pointer;
}

.registry-name:hover {
  text-decoration: underline;
}

.text-normal {
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.default-tag {
  background-color: #67c23a;
  border: none;
  color: #fff;
}

.insecure-tag {
  background-color: #e6a23c;
  border: none;
  color: #fff;
}

.quick-actions {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #4a9eff;
  font-size: 16px;
  padding: 4px;
}

.action-btn:hover {
  color: #79bbff;
}

.danger-btn {
  color: #f56c6c;
}

.danger-btn:hover {
  color: #f89898;
}

.table-footer {
  padding: 12px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #23252a;
}

.total-items {
  font-size: 13px;
  color: #8a929c;
}

/* Dialog Styles */
.create-dialog :deep(.el-dialog) {
  background-color: #2a2d31;
}

.create-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background-color: #2a2d31;
}

.create-dialog :deep(.el-dialog__title) {
  color: #fff;
}

.create-dialog :deep(.el-dialog__body) {
  background-color: #2a2d31;
  color: rgba(255, 255, 255, 0.85);
}

.create-dialog :deep(.el-dialog__footer) {
  background-color: #2a2d31;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.create-dialog :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.85);
}

.create-dialog :deep(.el-input__wrapper),
.create-dialog :deep(.el-select__wrapper) {
  background-color: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: none;
}

.create-dialog :deep(.el-input__inner) {
  color: #fff;
}

.create-dialog :deep(.el-input__inner::placeholder) {
  color: #8a929c;
}

.registry-form {
  max-height: 600px;
  overflow-y: auto;
}

.form-section {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.form-section:first-child {
  margin-top: 24px;
}

.form-section:last-child {
  border-bottom: none;
}

.form-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 500;
  color: #fff;
}

.form-tip {
  display: block;
  margin-left: 10px;
  font-size: 12px;
  color: #8a929c;
}

.registry-form :deep(.el-switch.is-checked .el-switch__core) {
  background-color: #4a9eff;
}
</style>
