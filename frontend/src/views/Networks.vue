<template>
  <PageContainer title="Network Management">
    <ContentCard title="Networks" :icon="Connection" :no-padding="true">
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
          :disabled="selectedNetworks.length === 0"
        >
          Remove
        </el-button>
        <el-button type="primary" :icon="Plus" @click="showCreateDialog">
          Add Network
        </el-button>
      </template>

      <DataTable
        :data="filteredNetworks"
        v-loading="loading"
        :total="filteredNetworks.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="Name" min-width="200">
          <template #default="{ row }">
            <span class="network-name">{{ row.Name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Type" width="120">
          <template #default="{ row }">
            <el-tag
              v-if="['bridge', 'host', 'none'].includes(row.Name)"
              type="info"
              size="small"
            >
              System
            </el-tag>
            <el-tag v-else type="success" size="small">Custom</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Driver" width="120">
          <template #default="{ row }">
            <span class="text-normal">{{ row.Driver }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Subnet" width="180">
          <template #default="{ row }">
            <code class="code-text">
              {{ row.IPAM?.Config?.[0]?.Subnet || "-" }}
            </code>
          </template>
        </el-table-column>

        <el-table-column label="Gateway" width="150">
          <template #default="{ row }">
            <span class="text-normal">
              {{ row.IPAM?.Config?.[0]?.Gateway || "-" }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="Attachable" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.Attachable ? 'success' : 'info'" size="small">
              {{ row.Attachable ? "Yes" : "No" }}
            </el-tag>
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
                  :disabled="['bridge', 'host', 'none'].includes(row.Name)"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>

    <!-- Create Network Dialog -->
    <FormDialog
      v-model="dialogVisible"
      title="Create Network"
      :icon="Connection"
      :loading="creating"
      @confirm="handleCreate"
      @cancel="dialogVisible = false"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="140px">
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="Name" prop="name">
            <el-input v-model="form.name" placeholder="e.g. my-network" />
          </el-form-item>
          <el-form-item label="Driver" prop="driver">
            <el-select v-model="form.driver" style="width: 100%">
              <el-option label="bridge" value="bridge" />
              <el-option label="overlay" value="overlay" />
              <el-option label="macvlan" value="macvlan" />
            </el-select>
          </el-form-item>
        </div>

        <div class="form-section">
          <h3>Network Configuration</h3>
          <el-form-item label="Subnet">
            <el-input
              v-model="form.subnet"
              placeholder="e.g. 172.20.0.0/16"
            />
            <div class="form-tip">
              Leave empty to use default subnet
            </div>
          </el-form-item>
          <el-form-item label="Gateway">
            <el-input v-model="form.gateway" placeholder="e.g. 172.20.0.1" />
            <div class="form-tip">
              Leave empty to use default gateway
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
  Connection,
  Search,
  Delete,
  Plus,
} from "@element-plus/icons-vue";
import { dockerApi } from "@/api/docker";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const networks = ref<any[]>([]);
const loading = ref(false);
const searchText = ref("");
const selectedNetworks = ref<any[]>([]);
const dialogVisible = ref(false);
const creating = ref(false);
const formRef = ref();

const form = ref({
  name: "",
  driver: "bridge",
  subnet: "",
  gateway: "",
});

const rules = {
  name: [{ required: true, message: "Please enter network name", trigger: "blur" }],
  driver: [{ required: true, message: "Please select driver", trigger: "change" }],
};

const filteredNetworks = computed(() => {
  if (!searchText.value) return networks.value;
  const search = searchText.value.toLowerCase();
  return networks.value.filter((net) =>
    net.Name.toLowerCase().includes(search)
  );
});

const loadNetworks = async () => {
  loading.value = true;
  try {
    networks.value = (await dockerApi.networks.list()) as any;
  } catch (error) {
    ElMessage.error("Failed to load networks");
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (selection: any[]) => {
  selectedNetworks.value = selection;
};

const showCreateDialog = () => {
  form.value = {
    name: "",
    driver: "bridge",
    subnet: "",
    gateway: "",
  };
  dialogVisible.value = true;
};

const handleCreate = async () => {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
  } catch {
    return;
  }

  creating.value = true;
  try {
    await dockerApi.networks.create(form.value);
    ElMessage.success("Network created successfully");
    dialogVisible.value = false;
    loadNetworks();
  } catch (error: any) {
    ElMessage.error(
      error.response?.data?.error || "Failed to create network"
    );
  } finally {
    creating.value = false;
  }
};

const handleBatchRemove = async () => {
  if (selectedNetworks.value.length === 0) {
    ElMessage.warning("Please select networks to remove");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedNetworks.value.length} network(s)?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    let successCount = 0;
    for (const network of selectedNetworks.value) {
      try {
        await dockerApi.networks.remove(network.Id);
        successCount++;
      } catch (error) {
        console.error("Failed to remove network:", network.Name);
      }
    }

    if (successCount > 0) {
      ElMessage.success(`${successCount} network(s) removed successfully`);
      selectedNetworks.value = [];
      loadNetworks();
    }
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error("Failed to remove networks");
    }
  }
};

const handleRemove = async (network: any) => {
  if (["bridge", "host", "none"].includes(network.Name)) {
    ElMessage.warning("Cannot remove system networks");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove network "${network.Name}"?`,
      "Confirm Removal",
      {
        type: "warning",
        confirmButtonText: "Remove",
        cancelButtonText: "Cancel",
      }
    );

    await dockerApi.networks.remove(network.Id);
    ElMessage.success("Network removed successfully");
    loadNetworks();
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(
        error.response?.data?.error || "Failed to remove network"
      );
    }
  }
};

onMounted(() => {
  loadNetworks();
});
</script>

<style scoped>
.network-name {
  color: var(--color-primary);
  font-weight: var(--font-weight-medium);
}
</style>
