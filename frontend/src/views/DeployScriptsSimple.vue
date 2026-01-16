<template>
  <PageContainer title="Deploy Scripts">
    <ContentCard title="Scripts" :icon="Document" :no-padding="true">
      <template #actions>
        <el-input
          v-model="searchText"
          placeholder="Search..."
          :prefix-icon="Search"
          style="width: 240px"
          clearable
        />
        <el-button
          type="danger"
          :icon="Delete"
          @click="handleBatchDelete"
          :disabled="selectedScripts.length === 0"
        >
          Remove
        </el-button>
        <el-button type="primary" :icon="Plus" @click="handleCreate">
          Add Script
        </el-button>
      </template>

      <DataTable
        :data="filteredScripts"
        v-loading="loading"
        :total="filteredScripts.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="Name" width="200">
          <template #default="{ row }">
            <span class="script-name">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Description" min-width="250">
          <template #default="{ row }">
            <span class="text-normal">{{ row.description || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Auto Deploy" width="120" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.auto_deploy"
              type="success"
              size="small"
            >
              Enabled
            </el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="Image" width="200">
          <template #default="{ row }">
            <code v-if="row.image_name" class="code-text">
              {{ row.image_name }}{{ row.image_tag ? `:${row.image_tag}` : '' }}
            </code>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Actions" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              type="success"
              size="small"
              :icon="VideoPlay"
              @click="handleExecute(row)"
            >
              Execute
            </el-button>
            <el-button
              type="primary"
              size="small"
              :icon="Edit"
              @click="handleEdit(row)"
              link
            />
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>

    <!-- Edit Dialog -->
    <FormDialog
      v-model="dialogVisible"
      :title="isEdit ? 'Edit Script' : 'Add Script'"
      :icon="Document"
      :loading="saving"
      @confirm="handleSave"
      @cancel="dialogVisible = false"
      extra-wide
    >
      <el-form :model="form" ref="formRef" label-width="140px">
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="Name">
            <el-input v-model="form.name" placeholder="e.g. deploy-nginx" />
          </el-form-item>
          <el-form-item label="Description">
            <el-input v-model="form.description" placeholder="Script description" />
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Webhook Auto Deploy</h3>
          <el-form-item label="Enable">
            <el-switch v-model="form.auto_deploy" />
            <div class="form-tip">
              Allow webhook to trigger this script automatically
            </div>
          </el-form-item>
          <el-form-item label="Image Name" v-if="form.auto_deploy">
            <el-input v-model="form.image_name" placeholder="e.g. myapp or */myapp" />
            <div class="form-tip">
              Supports wildcard *, e.g. namespace/* or */app
            </div>
          </el-form-item>
          <el-form-item label="Image Tag" v-if="form.auto_deploy">
            <el-input v-model="form.image_tag" placeholder="e.g. latest or v1.*" />
            <div class="form-tip">
              Leave empty to match any tag
            </div>
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Script Content</h3>
          <el-form-item label="Deploy Script">
            <el-input
              v-model="form.script_content"
              type="textarea"
              :rows="15"
              placeholder="#!/bin/bash&#10;&#10;# Your deployment script here"
            />
            <div class="form-tip">
              Bash script to execute for deployment
            </div>
          </el-form-item>
        </div>
      </el-form>
    </FormDialog>

    <!-- Execute Log Dialog -->
    <FormDialog
      v-model="logDialogVisible"
      :title="`Execution Log - ${currentScript?.name}`"
      :icon="Document"
      :show-footer="false"
      extra-wide
    >
      <div class="log-container">
        <el-scrollbar height="400px">
          <pre class="log-content">{{ executionLog }}</pre>
        </el-scrollbar>
      </div>
    </FormDialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Document,
  Search,
  Delete,
  Plus,
  VideoPlay,
  Edit,
} from "@element-plus/icons-vue";
import { deployScriptApi, type DeployScript } from "@/api/deploy";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const scripts = ref<DeployScript[]>([]);
const loading = ref(false);
const searchText = ref("");
const selectedScripts = ref<DeployScript[]>([]);
const dialogVisible = ref(false);
const logDialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const formRef = ref();
const currentScript = ref<DeployScript | null>(null);
const executionLog = ref("");

const form = ref({
  id: 0,
  name: "",
  description: "",
  script_content: "",
  auto_deploy: false,
  image_name: "",
  image_tag: "",
});

const filteredScripts = computed(() => {
  if (!searchText.value) return scripts.value;
  const search = searchText.value.toLowerCase();
  return scripts.value.filter(
    (s) =>
      s.name.toLowerCase().includes(search) ||
      s.description?.toLowerCase().includes(search)
  );
});

const loadScripts = async () => {
  loading.value = true;
  try {
    scripts.value = await deployScriptApi.list();
  } catch (error) {
    ElMessage.error("Failed to load scripts");
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (selection: DeployScript[]) => {
  selectedScripts.value = selection;
};

const handleCreate = () => {
  isEdit.value = false;
  form.value = {
    id: 0,
    name: "",
    description: "",
    script_content: "#!/bin/bash\n\n# Your deployment script here\n",
    auto_deploy: false,
    image_name: "",
    image_tag: "",
  };
  dialogVisible.value = true;
};

const handleEdit = (script: DeployScript) => {
  isEdit.value = true;
  form.value = {
    id: script.id,
    name: script.name,
    description: script.description || "",
    script_content: script.script_content || "",
    auto_deploy: script.auto_deploy || false,
    image_name: script.image_name || "",
    image_tag: script.image_tag || "",
  };
  dialogVisible.value = true;
};

const handleSave = async () => {
  saving.value = true;
  try {
    if (isEdit.value) {
      await deployScriptApi.update(form.value.id, form.value);
      ElMessage.success("Script updated successfully");
    } else {
      await deployScriptApi.create(form.value);
      ElMessage.success("Script created successfully");
    }
    dialogVisible.value = false;
    loadScripts();
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || "Failed to save script");
  } finally {
    saving.value = false;
  }
};

const handleExecute = async (script: DeployScript) => {
  currentScript.value = script;
  executionLog.value = "Starting execution...\n";
  logDialogVisible.value = true;

  try {
    const result = await deployScriptApi.execute(script.id);
    executionLog.value = result.output || "Execution completed";
    ElMessage.success("Script executed successfully");
    loadScripts();
  } catch (error: any) {
    executionLog.value += "\n\nError: " + (error.response?.data?.error || "Execution failed");
    ElMessage.error("Script execution failed");
  }
};

const handleBatchDelete = async () => {
  if (selectedScripts.value.length === 0) {
    ElMessage.warning("Please select scripts to remove");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedScripts.value.length} script(s)?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    for (const script of selectedScripts.value) {
      await deployScriptApi.delete(script.id);
    }

    ElMessage.success("Scripts removed successfully");
    selectedScripts.value = [];
    loadScripts();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error("Failed to remove scripts");
    }
  }
};

const getStatusType = (status: string) => {
  const map: Record<string, any> = {
    idle: "info",
    running: "warning",
    success: "success",
    failed: "danger",
  };
  return map[status] || "info";
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
  loadScripts();
});
</script>

<style scoped>
.script-name {
  color: var(--color-primary);
  font-weight: var(--font-weight-medium);
}

.log-container {
  background: rgba(0, 0, 0, 0.3);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}

.log-content {
  font-family: "Monaco", "Menlo", "Consolas", "Courier New", monospace;
  font-size: var(--font-size-xs);
  line-height: 1.6;
  color: var(--color-text-regular);
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>
