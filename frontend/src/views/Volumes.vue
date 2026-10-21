<template>
  <PageContainer title="卷管理">
    <ContentCard title="卷" :icon="FolderOpened" :no-padding="true">
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
          :disabled="selectedVolumes.length === 0"
        >
          删除
        </el-button>
        <el-button type="primary" :icon="Plus" @click="showCreateDialog">
          Add Volume
        </el-button>
      </template>

      <DataTable
        :data="filteredVolumes"
        v-loading="loading"
        :total="filteredVolumes.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="名称" min-width="250">
          <template #default="{ row }">
            <span class="volume-name">{{ row.Name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Driver" width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.Driver }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Mount Point" min-width="300">
          <template #default="{ row }">
            <code class="code-text">{{ row.Mountpoint }}</code>
          </template>
        </el-table-column>

        <el-table-column label="Scope" width="120">
          <template #default="{ row }">
            <span class="text-normal">{{ row.Scope }}</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.CreatedAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column
          label="快捷操作"
          width="100"
          align="center"
          fixed="right"
        >
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="删除" placement="top">
                <el-button
                  link
                  :icon="Delete"
                  @click="handleRemove(row)"
                  class="action-btn danger-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>

    <!-- Create Volume Dialog -->
    <FormDialog
      v-model="dialogVisible"
      title="Create Volume"
      :icon="FolderOpened"
      :loading="creating"
      @confirm="handleCreate"
      @cancel="dialogVisible = false"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px">
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="e.g. my-volume" />
            <div class="form-tip">
              Leave empty to generate a random name
            </div>
          </el-form-item>
          <el-form-item label="Driver">
            <el-select v-model="form.driver" style="width: 100%">
              <el-option label="local" value="local" />
            </el-select>
            <div class="form-tip">
              Storage driver for the volume
            </div>
          </el-form-item>
        </div>
      </el-form>
    </FormDialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  FolderOpened,
  Search,
  Delete,
  Plus,
} from "@element-plus/icons-vue";
import { dockerApi } from "@/api/docker";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const volumes = ref<any[]>([]);
const loading = ref(false);
const searchText = ref("");
const selectedVolumes = ref<any[]>([]);
const dialogVisible = ref(false);
const creating = ref(false);
const formRef = ref();

const form = ref({
  name: "",
  driver: "local",
});

const rules = {};

const filteredVolumes = computed(() => {
  if (!searchText.value) return volumes.value;
  const search = searchText.value.toLowerCase();
  return volumes.value.filter((vol) =>
    vol.Name.toLowerCase().includes(search)
  );
});

const loadVolumes = async () => {
  loading.value = true;
  try {
    const response = (await dockerApi.volumes.list()) as any;
    volumes.value = response.Volumes || [];
  } catch (error) {
    ElMessage.error("Failed to load volumes");
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (selection: any[]) => {
  selectedVolumes.value = selection;
};

const showCreateDialog = () => {
  form.value = {
    name: "",
    driver: "local",
  };
  dialogVisible.value = true;
};

const handleCreate = async () => {
  creating.value = true;
  try {
    await dockerApi.volumes.create(form.value.name, form.value.driver);
    ElMessage.success("Volume created successfully");
    dialogVisible.value = false;
    loadVolumes();
  } catch (error: any) {
    ElMessage.error(
      error.response?.data?.error || "Failed to create volume"
    );
  } finally {
    creating.value = false;
  }
};

const handleBatchRemove = async () => {
  if (selectedVolumes.value.length === 0) {
    ElMessage.warning("Please select volumes to remove");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedVolumes.value.length} volume(s)?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    let successCount = 0;
    for (const volume of selectedVolumes.value) {
      try {
        await dockerApi.volumes.remove(volume.Name);
        successCount++;
      } catch (error) {
        console.error("Failed to remove volume:", volume.Name);
      }
    }

    if (successCount > 0) {
      ElMessage.success(`${successCount} volume(s) removed successfully`);
      selectedVolumes.value = [];
      loadVolumes();
    }
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error("Failed to remove volumes");
    }
  }
};

const handleRemove = async (volume: any) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove volume "${volume.Name}"?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    await dockerApi.volumes.remove(volume.Name);
    ElMessage.success("Volume removed successfully");
    loadVolumes();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove volume"
      );
    }
  }
};

const formatDate = (dateStr: string) => {
  if (!dateStr) return "-";
  return new Date(dateStr).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

onMounted(() => {
  loadVolumes();
});
</script>

<style scoped>
.volume-name {
  color: var(--color-primary);
  font-weight: var(--font-weight-medium);
}
</style>
