<template>
  <div class="webhook-configs-view">
    <div class="page-header">
      <h1>Webhook Auto Deploy</h1>
    </div>

    <div class="content-card">
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon"><Connection /></el-icon>
          <span>Webhooks</span>
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
            @click="handleBatchDelete"
            :disabled="selectedItems.length === 0"
          >
            Remove
          </el-button>
          <el-button type="primary" :icon="Plus" @click="showCreateDialog">
            Add webhook
          </el-button>
        </div>
      </div>

      <!-- 配置列表 -->
      <el-table
        :data="filteredConfigs"
        v-loading="loading"
        class="data-table"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" fixed="left" />

        <el-table-column label="Status" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="handleToggleEnabled(row)"
            />
          </template>
        </el-table-column>

        <el-table-column prop="name" label="Name" min-width="160">
          <template #default="{ row }">
            <span class="webhook-name">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Image" min-width="220">
          <template #default="{ row }">
            <div class="image-info">
              <code class="image-code">{{ row.image_name }}</code>
              <el-tag v-if="row.image_tag" size="small" type="info" class="image-tag">
                {{ row.image_tag }}
              </el-tag>
              <el-tag v-else size="small" type="warning" class="image-tag">
                Any Tag
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Deploy Script" min-width="140">
          <template #default="{ row }">
            <span class="text-normal">{{ row.deploy_script?.name || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Registry" width="140">
          <template #default="{ row }">
            <span class="text-normal">{{ row.registry?.name || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Triggers" width="90" align="center">
          <template #default="{ row }">
            <el-tag type="primary" size="small">{{ row.trigger_count }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Last Triggered" width="180">
          <template #default="{ row }">
            <span class="text-normal">
              {{ row.last_triggered_at ? formatTime(row.last_triggered_at) : '-' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="Quick Actions" width="150" align="center" fixed="right">
          <template #default="{ row }">
            <div class="quick-actions">
              <el-tooltip content="Webhook URL" placement="top">
                <el-button
                  link
                  :icon="Link"
                  @click="showWebhookUrl(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Logs" placement="top">
                <el-button
                  link
                  :icon="Document"
                  @click="viewLogs(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Edit" placement="top">
                <el-button
                  link
                  :icon="Edit"
                  @click="handleEdit(row)"
                  class="action-btn"
                />
              </el-tooltip>
              <el-tooltip content="Remove" placement="top">
                <el-button
                  link
                  :icon="Delete"
                  @click="handleDelete(row)"
                  class="action-btn danger-btn"
                />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <span class="total-items">{{ filteredConfigs.length }} items</span>
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? 'Edit webhook' : 'Create webhook'"
      width="700px"
      class="create-dialog"
      :close-on-click-modal="false"
    >
      <el-form :model="form" label-width="140px" :rules="rules" ref="formRef" class="webhook-form">
        <!-- 基本信息 -->
        <div class="form-section">
          <h3>Basic Information</h3>
          <el-form-item label="Name" prop="name">
            <el-input v-model="form.name" placeholder="e.g. Auto Deploy MyApp" />
          </el-form-item>

          <el-form-item label="Description">
            <el-input
              v-model="form.description"
              type="textarea"
              :rows="2"
              placeholder="Optional description"
            />
          </el-form-item>

          <el-form-item label="Enabled">
            <el-switch v-model="form.enabled" />
          </el-form-item>
        </div>

        <!-- 镜像配置 -->
        <div class="form-section">
          <h3>Image Configuration</h3>
          <el-form-item label="Registry">
            <el-select v-model="form.registry_id" placeholder="Select registry" clearable>
              <el-option
                v-for="registry in registries"
                :key="registry.id"
                :label="registry.name"
                :value="registry.id"
              />
            </el-select>
            <div class="form-tip">Select registry if image is in a private registry</div>
          </el-form-item>

          <el-form-item label="Image Name" prop="image_name">
            <el-input v-model="form.image_name" placeholder="e.g. namespace/repo or com.example/app" />
            <div class="form-tip">Supports wildcards *, e.g. namespace/* or */app</div>
          </el-form-item>

          <el-form-item label="Image Tag">
            <el-input v-model="form.image_tag" placeholder="e.g. latest, v1.0.0 (leave empty for any tag)" />
            <div class="form-tip">Leave empty to match any tag</div>
          </el-form-item>
        </div>

        <!-- 部署配置 -->
        <div class="form-section">
          <h3>Deploy Configuration</h3>
          <el-form-item label="Deploy Script" prop="deploy_script_id">
            <el-select v-model="form.deploy_script_id" placeholder="Select deploy script">
              <el-option
                v-for="script in deployScripts"
                :key="script.id"
                :label="script.name"
                :value="script.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="Remove Old Container">
            <el-switch v-model="form.remove_old_container" />
            <div class="form-tip">Automatically stop and remove old containers using this image before deployment</div>
          </el-form-item>

          <el-form-item label="Remove Old Image">
            <el-switch v-model="form.remove_old_image" />
            <div class="form-tip">Automatically remove old image before deployment</div>
          </el-form-item>
        </div>

        <!-- 安全配置 -->
        <div class="form-section">
          <h3>Security</h3>
          <el-form-item label="Webhook Secret">
            <el-input
              v-model="form.secret"
              placeholder="Optional, for webhook signature verification"
              show-password
            />
            <div class="form-tip">When configured, only accept webhook requests with valid signature</div>
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          {{ isEditing ? 'Update' : 'Create' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Webhook URL 对话框 -->
    <el-dialog
      v-model="urlDialogVisible"
      title="Webhook Configuration"
      width="850px"
      class="create-dialog webhook-url-dialog"
      :close-on-click-modal="false"
    >
      <div class="webhook-url-container">
        <el-alert type="info" :closable="false" style="margin-bottom: 20px">
          <template #title>
            Configure this URL in your Docker Registry webhook settings
          </template>
        </el-alert>

        <!-- Webhook URL -->
        <div class="config-section">
          <h4>📌 Webhook URL</h4>
          <el-input
            v-model="webhookUrl"
            readonly
            class="webhook-url-input"
          >
            <template #append>
              <el-button @click="copyUrl" :icon="CopyDocument">Copy</el-button>
            </template>
          </el-input>
        </div>

        <!-- Registry 配置文件示例 -->
        <div class="config-section" style="margin-top: 24px">
          <h4>⚙️ Docker Registry Configuration (config.yml)</h4>
          <p class="config-description">
            For self-hosted Docker Registry, add the following to your <code>config.yml</code>:
          </p>
          <div class="config-code-wrapper">
            <pre class="config-code">{{ registryConfigYaml }}</pre>
            <el-button
              class="copy-config-btn"
              size="small"
              :icon="CopyDocument"
              @click="copyConfig"
            >
              Copy Config
            </el-button>
          </div>
        </div>

        <!-- 配置指南 -->
        <div class="webhook-guide" style="margin-top: 24px">
          <h4>📝 Platform-Specific Guides</h4>
          <el-collapse accordion>
            <el-collapse-item title="🐳 Docker Hub" name="dockerhub">
              <ol>
                <li>Go to your repository page</li>
                <li>Click <strong>Webhooks</strong> tab</li>
                <li>Click <strong>Create webhook</strong></li>
                <li>Paste the webhook URL</li>
                <li>Click <strong>Create</strong></li>
              </ol>
            </el-collapse-item>

            <el-collapse-item title="⚓ Harbor" name="harbor">
              <ol>
                <li>Go to your project</li>
                <li>Click <strong>Webhooks</strong> tab</li>
                <li>Click <strong>+ NEW WEBHOOK</strong></li>
                <li>Name: <code>DockeRun Auto Deploy</code></li>
                <li>Event Type: <strong>Push Artifact</strong></li>
                <li>Endpoint URL: Paste the webhook URL</li>
                <li v-if="currentConfig?.secret">
                  Auth Header: <code>{{ currentConfig.secret }}</code>
                </li>
                <li>Click <strong>TEST ENDPOINT</strong> to verify</li>
                <li>Click <strong>CONTINUE</strong></li>
              </ol>
            </el-collapse-item>

            <el-collapse-item title="🦊 GitLab Container Registry" name="gitlab">
              <ol>
                <li>Go to your project</li>
                <li>Click <strong>Settings</strong> → <strong>Webhooks</strong></li>
                <li>URL: Paste the webhook URL</li>
                <li v-if="currentConfig?.secret">
                  Secret token: <code>{{ currentConfig.secret }}</code>
                </li>
                <li>Trigger: Check <strong>Push events</strong></li>
                <li>Click <strong>Add webhook</strong></li>
              </ol>
            </el-collapse-item>

            <el-collapse-item title="🏗️ Self-Hosted Registry" name="selfhosted">
              <ol>
                <li>Edit your Registry's <code>config.yml</code></li>
                <li>Add the <code>notifications</code> section shown above</li>
                <li>Restart your Registry service:
                  <pre style="margin-top: 8px; padding: 8px; background: rgba(0,0,0,0.2); border-radius: 4px; font-size: 12px;">docker restart registry</pre>
                </li>
                <li>Test by pushing an image to verify webhook triggers</li>
              </ol>
            </el-collapse-item>
          </el-collapse>

          <div v-if="currentConfig?.secret" class="secret-notice">
            <el-icon><Lock /></el-icon>
            <span>
              <strong>Secret Key:</strong> <code class="secret-code">{{ currentConfig.secret }}</code>
            </span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="urlDialogVisible = false">Close</el-button>
        <el-button type="success" @click="copyConfig" :icon="CopyDocument">
          Copy Config
        </el-button>
        <el-button type="primary" @click="copyUrl" :icon="CopyDocument">
          Copy URL
        </el-button>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog
      v-model="logsDialogVisible"
      title="Webhook Execution Logs"
      width="900px"
      class="create-dialog"
      :close-on-click-modal="false"
    >
      <el-table :data="logs" style="width: 100%" max-height="500" class="data-table">
        <el-table-column label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Image" min-width="220">
          <template #default="{ row }">
            <code class="image-code">{{ row.image_name }}:{{ row.image_tag }}</code>
          </template>
        </el-table-column>

        <el-table-column label="Message" min-width="250">
          <template #default="{ row }">
            <div class="log-message">{{ row.message }}</div>
          </template>
        </el-table-column>

        <el-table-column label="Duration" width="90" align="center">
          <template #default="{ row }">
            <span class="text-normal">{{ row.execution_time }}s</span>
          </template>
        </el-table-column>

        <el-table-column label="Triggered At" width="180">
          <template #default="{ row }">
            <span class="text-normal">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="logsDialogVisible = false">Close</el-button>
        <el-button type="primary" @click="refreshLogs" :icon="Refresh">
          Refresh
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  Link,
  Document,
  CopyDocument,
  Lock,
  Refresh,
  Search,
  Connection
} from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import { webhookApi, type WebhookConfig, type WebhookLog, type WebhookURL } from '../api/webhook'
import { registryApi, type Registry } from '../api/registry'
import { deployApi, type DeployScript } from '../api/deploy'

const configs = ref<WebhookConfig[]>([])
const registries = ref<Registry[]>([])
const deployScripts = ref<DeployScript[]>([])
const logs = ref<WebhookLog[]>([])
const selectedItems = ref<WebhookConfig[]>([])

const dialogVisible = ref(false)
const urlDialogVisible = ref(false)
const logsDialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const loading = ref(false)
const searchText = ref('')
const webhookUrl = ref('')
const currentConfig = ref<WebhookConfig | null>(null)

// 过滤配置
const filteredConfigs = computed(() => {
  if (!searchText.value) return configs.value
  const search = searchText.value.toLowerCase()
  return configs.value.filter(config =>
    config.name.toLowerCase().includes(search) ||
    config.image_name.toLowerCase().includes(search) ||
    config.deploy_script?.name.toLowerCase().includes(search)
  )
})

const formRef = ref()
const form = ref<Partial<WebhookConfig>>({
  name: '',
  description: '',
  enabled: true,
  registry_id: undefined,
  image_name: '',
  image_tag: '',
  deploy_script_id: undefined,
  remove_old_image: true,
  remove_old_container: true,
  secret: ''
})

const rules = {
  name: [{ required: true, message: 'Please enter webhook name', trigger: 'blur' }],
  image_name: [{ required: true, message: 'Please enter image name', trigger: 'blur' }],
  deploy_script_id: [{ required: true, message: 'Please select deploy script', trigger: 'change' }]
}

const fetchConfigs = async () => {
  loading.value = true
  try {
    const data = await webhookApi.list()
    configs.value = data as unknown as WebhookConfig[]
  } catch (error: any) {
    ElMessage.error('Failed to load configurations: ' + error.message)
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: WebhookConfig[]) => {
  selectedItems.value = selection
}

const handleBatchDelete = async () => {
  if (selectedItems.value.length === 0) {
    ElMessage.warning('Please select webhooks to remove')
    return
  }

  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove ${selectedItems.value.length} webhook(s)?`,
      'Confirm Remove',
      { type: 'warning' }
    )

    for (const config of selectedItems.value) {
      await webhookApi.delete(config.id!)
    }

    ElMessage.success('Webhooks removed successfully')
    selectedItems.value = []
    fetchConfigs()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to remove webhooks: ' + error.message)
    }
  }
}

const fetchRegistries = async () => {
  try {
    const data = await registryApi.list()
    registries.value = data as unknown as Registry[]
  } catch (error: any) {
    console.error('Failed to load registries:', error)
  }
}

const fetchDeployScripts = async () => {
  try {
    const data = await deployApi.list()
    deployScripts.value = data as unknown as DeployScript[]
  } catch (error: any) {
    ElMessage.error('Failed to load deploy scripts: ' + error.message)
  }
}

const showCreateDialog = () => {
  isEditing.value = false
  form.value = {
    name: '',
    description: '',
    enabled: true,
    registry_id: undefined,
    image_name: '',
    image_tag: '',
    deploy_script_id: undefined,
    remove_old_image: true,
    remove_old_container: true,
    secret: ''
  }
  dialogVisible.value = true
}

const handleEdit = async (config: WebhookConfig) => {
  isEditing.value = true
  form.value = { ...config }
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    saving.value = true
    try {
      if (isEditing.value && form.value.id) {
        await webhookApi.update(form.value.id, form.value)
        ElMessage.success('Webhook updated successfully')
      } else {
        await webhookApi.create(form.value)
        ElMessage.success('Webhook created successfully')
      }
      dialogVisible.value = false
      fetchConfigs()
    } catch (error: any) {
      ElMessage.error('Failed to save webhook: ' + error.message)
    } finally {
      saving.value = false
    }
  })
}

const handleToggleEnabled = async (config: WebhookConfig) => {
  try {
    await webhookApi.update(config.id!, { enabled: config.enabled })
    ElMessage.success(config.enabled ? 'Enabled' : 'Disabled')
  } catch (error: any) {
    ElMessage.error('Failed to update: ' + error.message)
    config.enabled = !config.enabled
  }
}

const handleDelete = async (config: WebhookConfig) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove webhook "${config.name}"?`,
      'Confirm Remove',
      { type: 'warning' }
    )
    await webhookApi.delete(config.id!)
    ElMessage.success('Webhook removed successfully')
    fetchConfigs()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to remove webhook: ' + error.message)
    }
  }
}

const showWebhookUrl = async (config: WebhookConfig) => {
  currentConfig.value = config
  try {
    const data = await webhookApi.getUrl(config.id!) as unknown as WebhookURL
    webhookUrl.value = data.url
    urlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('Failed to get URL: ' + error.message)
  }
}

const { copy: copyToClipboard } = useClipboard()

const copyUrl = async () => {
  try {
    await copyToClipboard(webhookUrl.value)
    ElMessage.success('Webhook URL copied to clipboard')
  } catch (error) {
    ElMessage.error('Failed to copy URL')
  }
}

// 生成 Registry 配置文件内容
const registryConfigYaml = computed(() => {
  if (!webhookUrl.value) return ''
  
  const config = `notifications:
  endpoints:
    - name: dockerun
      disabled: false
      url: ${webhookUrl.value}
      timeout: 5s
      threshold: 5
      backoff: 1s${currentConfig.value?.secret ? `
      headers:
        Authorization: [${currentConfig.value.secret}]` : ''}
      events:
        - push
        - delete`
  
  return config
})

const copyConfig = async () => {
  try {
    await copyToClipboard(registryConfigYaml.value)
    ElMessage.success('Registry configuration copied to clipboard')
  } catch (error) {
    ElMessage.error('Failed to copy configuration')
  }
}

const viewLogs = async (config: WebhookConfig) => {
  currentConfig.value = config
  await refreshLogs()
  logsDialogVisible.value = true
}

const refreshLogs = async () => {
  if (!currentConfig.value) return
  try {
    const data = await webhookApi.listLogs(currentConfig.value.id)
    logs.value = data as unknown as WebhookLog[]
  } catch (error: any) {
    ElMessage.error('Failed to load logs: ' + error.message)
  }
}

const getStatusType = (status: string) => {
  const types: Record<string, any> = {
    pending: 'info',
    processing: 'warning',
    success: 'success',
    failed: 'danger'
  }
  return types[status] || 'info'
}

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: 'Pending',
    processing: 'Processing',
    success: 'Success',
    failed: 'Failed'
  }
  return texts[status] || status
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchConfigs()
  fetchRegistries()
  fetchDeployScripts()
})
</script>

<style scoped>
.webhook-configs-view {
  min-height: 100vh;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 300;
  color: #ffffff;
}

.content-card {
  background-color: #23252a;
  border-radius: 8px;
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 500;
  color: #ffffff;
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
  width: 250px;
}

.data-table {
  background: transparent;
}

.webhook-name {
  color: #4a9eff;
  font-weight: 500;
}

.image-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-code {
  background: rgba(36, 150, 237, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
  color: #2496ED;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}

.image-tag {
  margin: 0;
}

.text-normal {
  color: #c0c4cc;
}

.quick-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.action-btn {
  color: #4a9eff;
  padding: 4px;
}

.action-btn:hover {
  color: #6bb3ff;
}

.danger-btn {
  color: #f56c6c;
}

.danger-btn:hover {
  color: #f78989;
}

.table-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.total-items {
  color: #8a929c;
  font-size: 14px;
}

/* 对话框样式 */
.create-dialog :deep(.el-dialog) {
  background: #1a1f2e;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.create-dialog :deep(.el-dialog__title) {
  color: #ffffff;
  font-size: 18px;
  font-weight: 500;
}

.webhook-form {
  max-height: 60vh;
  overflow-y: auto;
  padding-right: 10px;
}

.form-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.form-section:first-child {
  margin-top: 24px;
}

.form-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.form-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 500;
  color: #4a9eff;
}

.form-tip {
  font-size: 12px;
  color: #8a929c;
  margin-top: 4px;
  margin-left: 4px;
  line-height: 1.4;
}

.webhook-url-container {
  padding: 10px 0;
}

.webhook-url-input {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 13px;
}

.webhook-url-input :deep(.el-input__inner) {
  font-family: 'Monaco', 'Courier New', monospace;
}

.config-section h4 {
  margin: 0 0 12px 0;
  color: #4a9eff;
  font-size: 15px;
  font-weight: 500;
}

.config-description {
  margin: 0 0 12px 0;
  color: #8a929c;
  font-size: 13px;
}

.config-description code {
  background: rgba(74, 158, 255, 0.1);
  padding: 2px 6px;
  border-radius: 3px;
  color: #4a9eff;
  font-size: 12px;
}

.config-code-wrapper {
  position: relative;
  background: #151820;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  padding: 16px;
}

.config-code {
  margin: 0;
  padding: 0;
  color: #c0c4cc;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre;
  overflow-x: auto;
}

.config-code::-webkit-scrollbar {
  height: 6px;
}

.config-code::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 3px;
}

.config-code::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.copy-config-btn {
  position: absolute;
  top: 12px;
  right: 12px;
}

.webhook-guide h4 {
  margin-top: 0;
  margin-bottom: 12px;
  color: #4a9eff;
  font-size: 15px;
  font-weight: 500;
}

.webhook-guide :deep(.el-collapse) {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  overflow: hidden;
}

.webhook-guide :deep(.el-collapse-item__header) {
  background: rgba(255, 255, 255, 0.03);
  color: #c0c4cc;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  padding: 0 16px;
  height: 44px;
  line-height: 44px;
  font-size: 14px;
}

.webhook-guide :deep(.el-collapse-item__content) {
  background: rgba(0, 0, 0, 0.2);
  padding: 16px 20px;
  color: #c0c4cc;
}

.webhook-guide ol {
  margin: 0;
  padding-left: 20px;
}

.webhook-guide ol li {
  margin: 10px 0;
  color: #c0c4cc;
  font-size: 13px;
  line-height: 1.6;
}

.webhook-guide ol li strong {
  color: #ffffff;
  font-weight: 500;
}

.webhook-guide ol li code {
  background: rgba(74, 158, 255, 0.1);
  padding: 2px 6px;
  border-radius: 3px;
  color: #4a9eff;
  font-size: 12px;
}

.secret-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 12px;
  background: rgba(245, 108, 108, 0.1);
  border: 1px solid rgba(245, 108, 108, 0.3);
  border-radius: 6px;
  color: #f56c6c;
  font-size: 13px;
}

.secret-notice strong {
  color: #ffffff;
}

.secret-code {
  background: rgba(0, 0, 0, 0.3);
  padding: 2px 8px;
  border-radius: 3px;
  color: #f78989;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}

.log-message {
  font-size: 13px;
  color: #c0c4cc;
  word-break: break-all;
}

/* 滚动条样式 */
.webhook-form::-webkit-scrollbar {
  width: 6px;
}

.webhook-form::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 3px;
}

.webhook-form::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.webhook-form::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}
</style>

