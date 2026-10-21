<template>
  <PageContainer title="容器管理">
    <ContentCard title="容器" :icon="Box" :no-padding="true">
      <template #actions>
        <el-input
          v-model="searchText"
          placeholder="搜索…"
          :prefix-icon="Search"
          style="width: 240px"
          clearable
        />
        <el-button
          type="success"
          :icon="VideoPlay"
          @click="handleBatchAction('start')"
          :disabled="selectedContainers.length === 0"
        >
          Start
        </el-button>
        <el-button
          type="warning"
          :icon="VideoPause"
          @click="handleBatchAction('stop')"
          :disabled="selectedContainers.length === 0"
        >
          Stop
        </el-button>
        <el-button
          type="primary"
          :icon="Refresh"
          @click="handleBatchAction('restart')"
          :disabled="selectedContainers.length === 0"
        >
          Restart
        </el-button>
        <el-button
          type="danger"
          :icon="Delete"
          @click="handleBatchAction('remove')"
          :disabled="selectedContainers.length === 0"
        >
          删除
        </el-button>
      </template>

      <DataTable
        :data="filteredContainers"
        v-loading="loading"
        :total="filteredContainers.length"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="名称" min-width="200">
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon :class="['status-icon', getStateClass(row.State)]">
                <SuccessFilled v-if="row.State === 'running'" />
                <CircleClose v-else />
              </el-icon>
              <span class="container-name">
                {{ row.Names?.[0]?.replace("/", "") || "-" }}
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="State" label="运行状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStateType(row.State)" size="small">
              {{ row.State }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="镜像" min-width="250">
          <template #default="{ row }">
            <code class="code-text">{{ row.Image }}</code>
          </template>
        </el-table-column>

        <el-table-column label="IP Address" width="140">
          <template #default="{ row }">
            <span class="text-normal">{{ getIPAddress(row) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="端口" min-width="180">
          <template #default="{ row }">
            <div class="ports-cell">
              <div
                v-for="(port, index) in row.Ports"
                :key="index"
                class="port-item"
              >
                <el-icon><Link /></el-icon>
                <span>{{ formatPort(port) }}</span>
              </div>
              <span v-if="!row.Ports || row.Ports.length === 0" class="text-muted">-</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatDate(row.Created) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="快捷操作" width="150" align="center" fixed="right">
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Logs" placement="top">
                <el-button
                  link
                  :icon="Document"
                  @click="handleLogs(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Stats" placement="top">
                <el-button
                  link
                  :icon="DataAnalysis"
                  @click="handleStats(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Terminal" placement="top">
                <el-button
                  link
                  :icon="Monitor"
                  @click="handleTerminal(row)"
                  class="action-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </DataTable>
    </ContentCard>

    <!-- 日志对话框 -->
    <FormDialog
      v-model="logsVisible"
      :title="`Container Logs - ${currentContainer?.Names?.[0]?.replace('/', '')}`"
      :icon="Document"
      :show-footer="false"
      extra-wide
    >
      <div class="logs-container">
        <el-input
          v-model="logs"
          type="textarea"
          :rows="25"
          readonly
          class="logs-textarea"
        />
      </div>
    </FormDialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Refresh,
  Search,
  Box,
  SuccessFilled,
  CircleClose,
  Document,
  DataAnalysis,
  Monitor,
  Link,
  VideoPlay,
  VideoPause,
  Delete,
} from "@element-plus/icons-vue";
import { dockerApi } from "@/api/docker";
import PageContainer from "@/components/PageContainer.vue";
import ContentCard from "@/components/ContentCard.vue";
import DataTable from "@/components/DataTable.vue";
import FormDialog from "@/components/FormDialog.vue";

const containers = ref<any[]>([]);
const loading = ref(false);
const logsVisible = ref(false);
const logs = ref("");
const currentContainer = ref<any>(null);
const searchText = ref("");
const selectedContainers = ref<any[]>([]);

const filteredContainers = computed(() => {
  if (!searchText.value) return containers.value;
  const search = searchText.value.toLowerCase();
  return containers.value.filter(
    (c) =>
      c.Names?.[0]?.toLowerCase().includes(search) ||
      c.Image?.toLowerCase().includes(search)
  );
});

const loadContainers = async () => {
  loading.value = true;
  try {
    containers.value = (await dockerApi.containers.list(true)) as any;
  } catch (error) {
    ElMessage.error("Failed to load containers");
  } finally {
    loading.value = false;
  }
};

const handleSelectionChange = (selection: any[]) => {
  selectedContainers.value = selection;
};

const getStateClass = (state: string) => {
  return state === "running" ? "running" : "stopped";
};

const getStateType = (state: string) => {
  const map: Record<string, any> = {
    running: "success",
    exited: "info",
    paused: "warning",
    dead: "danger",
  };
  return map[state] || "info";
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

const getIPAddress = (container: any) => {
  return container.NetworkSettings?.Networks?.bridge?.IPAddress || "-";
};

const formatPort = (port: any) => {
  if (port.PublicPort) {
    return `${port.PublicPort}:${port.PrivatePort}`;
  }
  return `${port.PrivatePort}`;
};

const handleLogs = async (container: any) => {
  currentContainer.value = container;
  try {
    logs.value = (await dockerApi.containers.logs(container.Id, "500")) as any;
    logsVisible.value = true;
  } catch (error) {
    ElMessage.error("Failed to get logs");
  }
};

const handleStats = (_container: any) => {
  ElMessage.info("Stats feature coming soon");
};

const handleTerminal = (_container: any) => {
  ElMessage.info("Terminal feature coming soon");
};

const handleBatchAction = async (action: string) => {
  if (selectedContainers.value.length === 0) {
    ElMessage.warning("Please select at least one container");
    return;
  }

  const actionMap: Record<string, string> = {
    start: "start",
    stop: "stop",
    restart: "restart",
    remove: "remove",
  };

  const actionName = actionMap[action];
  if (!actionName) {
    ElMessage.error("Unknown action");
    return;
  }

  // 如果是删除操作，需要确认
  if (action === "remove") {
    try {
      await ElMessageBox.confirm(
        `Are you sure you want to remove ${selectedContainers.value.length} container(s)?`,
        "Confirm Removal",
        {
          confirmButtonText: "Remove",
          cancelButtonText: "Cancel",
          type: "warning",
        }
      );
    } catch {
      return;
    }
  }

  const loadingMsg = ElMessage({
    message: `Processing ${action} action...`,
    type: "info",
    duration: 0,
  });

  let successCount = 0;
  let failCount = 0;

  for (const container of selectedContainers.value) {
    try {
      const id = container.Id;

      switch (action) {
        case "start":
          await dockerApi.containers.start(id);
          break;
        case "stop":
          await dockerApi.containers.stop(id);
          break;
        case "restart":
          await dockerApi.containers.restart(id);
          break;
        case "remove":
          await dockerApi.containers.remove(id, false);
          break;
      }

      successCount++;
    } catch (error) {
      console.error(`Failed to ${action} container:`, container.Names?.[0]);
      failCount++;
    }
  }

  loadingMsg.close();

  if (successCount > 0) {
    ElMessage.success(`${successCount} container(s) ${action}ed successfully`);
    selectedContainers.value = [];
    loadContainers();
  }

  if (failCount > 0) {
    ElMessage.warning(`${failCount} container(s) failed`);
  }
};

onMounted(() => {
  loadContainers();
});
</script>

<style scoped>
.name-cell {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.status-icon {
  font-size: 10px;
  flex-shrink: 0;
  transition: color var(--transition-fast);
}

.status-icon.running {
  color: var(--color-success);
}

.status-icon.stopped {
  color: var(--color-text-secondary);
}

.container-name {
  color: var(--color-primary);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.container-name:hover {
  color: #79bbff;
  text-decoration: underline;
}

.ports-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--color-primary);
  font-size: var(--font-size-xs);
}

.port-item .el-icon {
  font-size: 14px;
}

.logs-container {
  padding: var(--spacing-md);
  background: rgba(0, 0, 0, 0.3);
  border-radius: var(--radius-md);
}

.logs-textarea :deep(.el-textarea__inner) {
  font-family: "Monaco", "Menlo", "Consolas", "Courier New", monospace;
  font-size: var(--font-size-xs);
  line-height: 1.6;
  background: #1e1e1e;
  color: #d4d4d4;
  border: 1px solid var(--color-border-base);
}

.logs-textarea :deep(.el-textarea__inner)::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.logs-textarea :deep(.el-textarea__inner)::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.logs-textarea :deep(.el-textarea__inner)::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
}
</style>
