<template>
  <PageContainer title="Image Management">
    <ContentCard title="Images" :icon="Picture" :no-padding="true">
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
          @click="handleBatchRemove"
          :disabled="selectedImages.length === 0"
        >
          Remove
        </el-button>
        <el-button :icon="Download" @click="showImportDialog">
          Import
        </el-button>
        <el-button :icon="Upload" @click="showExportDialog">
          Export
        </el-button>
        <el-button type="primary" :icon="Download" @click="handlePullImage">
          Pull Image
        </el-button>
      </template>

      <DataTable
        :data="filteredImages"
        v-loading="loading"
        :total="filteredImages.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="ID" width="150">
          <template #default="{ row }">
            <code class="code-text">
              {{ row.Id.replace("sha256:", "").substring(0, 12) }}
            </code>
          </template>
        </el-table-column>

        <el-table-column label="Tags" min-width="350">
          <template #default="{ row }">
            <div class="tags-cell">
              <el-tag
                v-for="(tag, index) in row.RepoTags?.slice(0, 3)"
                :key="index"
                size="small"
                type="info"
              >
                {{ tag }}
              </el-tag>
              <el-tag
                v-if="!row.RepoTags || row.RepoTags.length === 0"
                size="small"
                type="warning"
              >
                &lt;none&gt;
              </el-tag>
              <span
                v-if="row.RepoTags && row.RepoTags.length > 3"
                class="text-muted"
              >
                +{{ row.RepoTags.length - 3 }} more
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Size" width="120">
          <template #default="{ row }">
            <span class="text-normal">{{ formatSize(row.Size) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Created" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.Created) }}</span>
          </template>
        </el-table-column>

        <el-table-column
          label="Quick Actions"
          width="100"
          align="center"
          fixed="right"
        >
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Remove" placement="top">
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

    <!-- Pull Image Dialog -->
    <FormDialog
      v-model="pullDialogVisible"
      title="Pull Image"
      :icon="Download"
      :loading="pulling"
      @confirm="handlePullConfirm"
      @cancel="pullDialogVisible = false"
    >
      <el-form label-width="120px">
        <div class="form-section">
          <h3>Image Information</h3>
          <el-form-item label="Image Name">
            <el-input
              v-model="pullImageName"
              placeholder="e.g. nginx:latest or mysql:8.0"
            />
            <div class="form-tip">
              Format: [registry/]repository[:tag]
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
  Picture,
  Search,
  Delete,
  Download,
  Upload,
} from "@element-plus/icons-vue";
import { dockerApi } from "@/api/docker";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const images = ref<any[]>([]);
const loading = ref(false);
const searchText = ref("");
const selectedImages = ref<any[]>([]);
const pullDialogVisible = ref(false);
const pullImageName = ref("");
const pulling = ref(false);
const importDialogVisible = ref(false);
const exportDialogVisible = ref(false);

const filteredImages = computed(() => {
  if (!searchText.value) return images.value;
  const search = searchText.value.toLowerCase();
  return images.value.filter((img) =>
    img.RepoTags?.some((tag: string) => tag.toLowerCase().includes(search))
  );
});

const loadImages = async () => {
  loading.value = true;
  try {
    images.value = (await dockerApi.images.list()) as any;
  } catch (error) {
    ElMessage.error("Failed to load images");
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (selection: any[]) => {
  selectedImages.value = selection;
};

const handlePullImage = () => {
  pullImageName.value = "";
  pullDialogVisible.value = true;
};

const handlePullConfirm = async () => {
  if (!pullImageName.value.trim()) {
    ElMessage.warning("Please enter image name");
    return;
  }

  pulling.value = true;
  try {
    await dockerApi.images.pull(pullImageName.value);
    ElMessage.success("Image pulled successfully");
    pullDialogVisible.value = false;
    loadImages();
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || "Failed to pull image");
  } finally {
    pulling.value = false;
  }
};

const handleBatchRemove = async () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning("Please select images to remove");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedImages.value.length} image(s)?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    let successCount = 0;
    for (const image of selectedImages.value) {
      try {
        await dockerApi.images.remove(image.Id, false);
        successCount++;
      } catch (error) {
        console.error("Failed to remove image:", image.Id);
      }
    }

    if (successCount > 0) {
      ElMessage.success(`${successCount} image(s) removed successfully`);
      selectedImages.value = [];
      loadImages();
    }
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error("Failed to remove images");
    }
  }
};

const handleRemove = async (image: any) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove this image?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    await dockerApi.images.remove(image.Id, false);
    ElMessage.success("Image removed successfully");
    loadImages();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove image"
      );
    }
  }
};

const formatSize = (bytes: number) => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
};

const formatDate = (timestamp: number) => {
  const date = new Date(timestamp * 1000);
  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

const showImportDialog = () => {
  importDialogVisible.value = true;
  ElMessage.info("Import feature: Please select a .tar file to import");
};

const showExportDialog = () => {
  if (selectedImages.value.length === 0) {
    ElMessage.warning("Please select images to export");
    return;
  }
  exportDialogVisible.value = true;
  ElMessage.info("Export feature: Selected images will be exported as .tar");
};

onMounted(() => {
  loadImages();
});
</script>

<style scoped>
.tags-cell {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  flex-wrap: wrap;
}
</style>
