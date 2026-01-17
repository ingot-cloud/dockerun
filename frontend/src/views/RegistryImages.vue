<template>
  <PageContainer>
    <template #header>
      <div style="display: flex; align-items: center; gap: 12px">
        <el-button link @click="goBack" style="font-size: 20px">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <h1 style="margin: 0; font-size: var(--font-size-3xl); font-weight: var(--font-weight-semibold)">
          {{ registryName }} - Images
        </h1>
      </div>
    </template>

    <ContentCard title="Images" :icon="Box" :no-padding="true">
      <template #actions>
        <el-input
          v-model="searchKeyword"
          placeholder="Search images..."
          :prefix-icon="Search"
          style="width: 240px"
          clearable
          @input="handleSearch"
        />
        <el-button :icon="Refresh" @click="fetchImages" :loading="loading">
          Refresh
        </el-button>
      </template>

      <DataTable
        :data="images"
        v-loading="loading"
        :total="totalImages"
        :show-footer="false"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-content">
              <div class="tags-header">
                <strong>Tags:</strong>
                <span class="text-muted">
                  ({{ row.tags.length }} tag{{ row.tags.length !== 1 ? "s" : "" }})
                </span>
              </div>
              <div class="tags-list">
                <el-tag
                  v-for="tag in row.tags"
                  :key="tag"
                  size="small"
                  closable
                  @close="deleteImage(row.name, tag)"
                  style="margin: 4px"
                >
                  {{ tag }}
                </el-tag>
                <el-empty
                  v-if="row.tags.length === 0"
                  description="No tags"
                  :image-size="60"
                />
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Image Name" min-width="400">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon style="color: var(--color-primary)"><Box /></el-icon>
              <code class="code-text">{{ row.name }}</code>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Tags" width="120" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">
              {{ row.tags.length }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Quick Actions" width="120" align="center">
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Copy Image Name" placement="top">
                <el-button
                  link
                  :icon="CopyDocument"
                  @click="copyImageName(row.name)"
                  class="action-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </DataTable>

      <template #footer v-if="totalImages > 0">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="totalImages"
          layout="sizes, prev, pager, next"
          background
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </template>
    </ContentCard>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowLeft,
  Search,
  Refresh,
  Box,
  CopyDocument,
} from "@element-plus/icons-vue";
import { registryApi, type RegistryImage } from "@/api/registry";
import { useClipboard } from "@vueuse/core";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";

const router = useRouter();
const route = useRoute();
const loading = ref(false);
const images = ref<RegistryImage[]>([]);
const searchKeyword = ref("");
const registryName = ref("");
const registryId = ref(Number(route.params.id));

const currentPage = ref(1);
const pageSize = ref(20);
const totalImages = ref(0);

const { copy: copyToClipboard } = useClipboard();

let searchTimer: any = null;

const fetchImages = async () => {
  loading.value = true;
  try {
    const result = await registryApi.images.list(
      registryId.value,
      currentPage.value,
      pageSize.value,
      searchKeyword.value
    );
    images.value = result.items || [];
    totalImages.value = result.total || 0;
    registryName.value = (result as any).registry_name || "";
  } catch (error: any) {
    ElMessage.error(
      error.response?.data?.error || "Failed to load registry images"
    );
    images.value = [];
    totalImages.value = 0;
  } finally {
    loading.value = false;
  }
};

const handleSearch = () => {
  if (searchTimer) {
    clearTimeout(searchTimer);
  }
  searchTimer = setTimeout(() => {
    currentPage.value = 1;
    fetchImages();
  }, 500);
};

const handlePageChange = () => {
  fetchImages();
};

const handleSizeChange = () => {
  currentPage.value = 1;
  fetchImages();
};

const goBack = () => {
  router.push("/registry");
};

const copyImageName = async (name: string) => {
  try {
    await copyToClipboard(name);
    ElMessage.success("Image name copied to clipboard");
  } catch {
    ElMessage.error("Failed to copy image name");
  }
};

const deleteImage = async (name: string, tag: string) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete ${name}:${tag}?`,
      "Confirm Deletion",
      {
        confirmButtonText: "Delete",
        cancelButtonText: "Cancel",
        type: "warning",
      }
    );

    await registryApi.images.delete(registryId.value, name, tag);
    ElMessage.success("Image deleted successfully");
    fetchImages();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to delete image"
      );
    }
  }
};

onMounted(() => {
  fetchImages();
});
</script>

<style scoped>
.expand-content {
  padding: var(--spacing-lg) var(--spacing-2xl);
  background: rgba(0, 0, 0, 0.2);
}

.tags-header {
  margin-bottom: var(--spacing-md);
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
}
</style>
