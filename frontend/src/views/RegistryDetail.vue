<template>
  <div class="registry-resource registry-detail-page">
    <header class="registry-heading">
      <h1>{{ creating ? '添加 Registry' : '仓库详情' }}<span v-if="registry" class="registry-heading-name">{{ registry.name }}</span></h1>
      <div class="registry-heading-actions"><el-button v-if="registry" :disabled="busy || loading" @click="router.push({ name: 'RegistryImages', params: { id: registry.id }, query: route.query })"><PortainerIcon name="box" />浏览镜像</el-button><el-button link :disabled="busy" @click="goBack">返回仓库列表</el-button></div>
    </header>
    <div v-if="loadError" class="registry-error" role="alert"><span>{{ loadError }}</span><el-button link @click="loadRegistry">重试</el-button></div>
    <div v-loading="loading" class="registry-detail-body" :aria-busy="loading">
      <template v-if="creating || registry">
        <section v-if="creating" class="registry-card" aria-labelledby="provider-title">
          <header class="registry-card-header"><h2 id="provider-title"><span class="registry-section-icon"><PortainerIcon name="registry" /></span>仓库提供商</h2></header>
          <div class="registry-provider-grid" role="group" aria-label="选择仓库提供商">
            <button v-for="provider in registryProviders" :key="provider.id" type="button" class="registry-provider" :class="{ active: selectedProvider === provider.id }" :aria-pressed="selectedProvider === provider.id" :disabled="busy" @click="selectProvider(provider.id)"><PortainerIcon :name="provider.id === 'custom' ? 'registry' : 'box'" /><span><strong>{{ provider.label }}</strong><small>{{ provider.description }}</small></span><span v-if="selectedProvider === provider.id" class="registry-provider-check" aria-hidden="true">✓</span></button>
          </div>
        </section>
        <section class="registry-card" aria-labelledby="registry-details-title">
          <header class="registry-card-header"><h2 id="registry-details-title"><span class="registry-section-icon"><PortainerIcon name="registry" /></span>{{ activeProvider.label }} 配置</h2></header>
          <el-form ref="formRef" :model="form" :rules="rules" label-width="140px" class="registry-config-form" :disabled="busy" @submit.prevent="save">
            <el-form-item label="名称" prop="name"><el-input v-model="form.name" placeholder="例如 Production registry" maxlength="200" /></el-form-item>
            <el-form-item label="仓库地址" prop="url"><el-input v-model="form.url" :placeholder="selectedProvider === 'azure' ? 'https://myregistry.azurecr.io' : 'https://registry.example.com'" /><p class="registry-form-help">填写仓库域名和端口；未指定协议时默认使用 HTTPS，不包含镜像名称。</p></el-form-item>
            <el-form-item label="身份认证"><el-switch v-model="form.authentication" aria-label="启用身份认证" /><p class="registry-form-help">{{ form.authentication ? '使用用户名和密码或访问令牌连接仓库。' : '使用匿名访问连接仓库。' }}</p></el-form-item>
            <template v-if="form.authentication">
              <el-form-item label="用户名" prop="username"><el-input v-model="form.username" autocomplete="off" placeholder="仓库用户名" maxlength="100" /></el-form-item>
              <el-form-item :label="selectedProvider === 'github' ? '访问令牌' : '密码 / 访问令牌'" prop="password"><el-input v-model="form.password" type="password" show-password autocomplete="new-password" :placeholder="registry?.username ? '留空保留已有密码' : '输入密码或访问令牌'" /><p class="registry-form-help">{{ selectedProvider === 'github' ? '使用具有所需包访问权限的 Personal Access Token。' : '密码在服务器加密保存。' }}</p></el-form-item>
              <el-form-item label="邮箱" prop="email"><el-input v-model="form.email" placeholder="可选" maxlength="200" /></el-form-item>
            </template>
            <div class="registry-form-section registry-options">
              <h3>仓库选项</h3>
              <el-form-item label="默认仓库"><el-switch v-model="form.is_default" aria-label="设为默认仓库" /><p class="registry-form-help">将此仓库用于默认的镜像操作。</p></el-form-item>
              <el-form-item label="跳过 TLS 验证"><el-switch v-model="form.insecure" aria-label="跳过 TLS 证书验证" /><p class="registry-form-help">适用于使用自签名证书的 HTTPS 仓库；使用 HTTP 时请在地址中明确填写 http://。</p></el-form-item>
            </div>
            <div class="registry-form-section">
              <h3>Agent 专用拉取账号</h3>
              <p class="registry-form-help">在外部仓库创建只读账号后填写，Agent 使用该账号拉取镜像。</p>
              <el-form-item label="拉取用户名" class="registry-pull-username"><el-input v-model="form.pull_username" placeholder="可选，仅用于 Agent 拉取镜像" maxlength="100" /></el-form-item>
              <el-form-item label="拉取密码"><el-input v-model="form.pull_password" type="password" show-password autocomplete="new-password" :placeholder="registry?.pull_username ? '留空保留已有拉取密码' : '输入拉取密码或访问令牌'" /></el-form-item>
              <el-form-item label="只读权限"><el-checkbox v-model="form.pull_only_confirmed">已在外部仓库将该账号限制为只读</el-checkbox></el-form-item>
            </div>
          </el-form>
        </section>
        <section v-if="registry" class="registry-card" aria-labelledby="registry-info-title"><header class="registry-card-header"><h2 id="registry-info-title">仓库信息</h2></header><dl class="registry-details"><div><dt>ID</dt><dd>{{ registry.id }}</dd></div><div><dt>创建时间</dt><dd>{{ formatDate(registry.created_at) }}</dd></div><div><dt>更新时间</dt><dd>{{ formatDate(registry.updated_at) }}</dd></div></dl></section>
      </template>
    </div>
    <div v-if="actionError" class="registry-error" role="alert"><span>{{ actionError }}</span><el-button link @click="actionError = ''">关闭</el-button></div>
    <div v-if="testSuccess" class="registry-test-success" role="status">Registry API 连接成功；目录和镜像标签的读取结果请在“浏览镜像”中查看。</div>
    <footer v-if="creating || registry" class="registry-card registry-form-footer">
      <el-button v-if="registry" class="registry-remove" :disabled="busy || loading" @click="deleteVisible = true"><PortainerIcon name="trash" />删除仓库</el-button><span v-else />
      <div><el-button :disabled="busy || loading" :loading="testing" @click="testConnection">测试连接</el-button><el-button :disabled="busy" @click="goBack">取消</el-button><el-button class="registry-primary" :loading="saving" :disabled="busy || loading" @click="save">{{ creating ? '添加 Registry' : '保存更改' }}</el-button></div>
    </footer>
    <RegistryDeleteDialog v-model="deleteVisible" :registries="registry ? [registry] : []" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { registryApi, type Registry } from '@/api/registry'
import PortainerIcon from '@/components/PortainerIcon.vue'
import RegistryDeleteDialog from '@/components/registries/RegistryDeleteDialog.vue'
import { buildRegistryRequest, emptyRegistryForm, normalizeRegistryUrl, registryError, registryProvider, registryProviders, type RegistryProvider } from '@/utils/registries'
import '@/styles/registries.css'

const route = useRoute(), router = useRouter()
const creating = computed(() => route.name === 'RegistryCreate')
const registry = ref<Registry | null>(null)
const form = ref(emptyRegistryForm()), formRef = ref<FormInstance>()
const selectedProvider = ref<RegistryProvider>('custom')
const activeProvider = computed(() => registryProviders.find(provider => provider.id === selectedProvider.value)!)
const loading = ref(false), saving = ref(false), testing = ref(false), deleteVisible = ref(false)
const loadError = ref(''), actionError = ref(''), testSuccess = ref(false)
const busy = computed(() => saving.value || testing.value || deleteVisible.value)
let controller: AbortController | undefined
let disposed = false
const rules: FormRules = {
  name: [{ required: true, whitespace: true, message: '请填写仓库名称', trigger: 'blur' }],
  url: [{ validator: (_rule, value, callback) => { try { normalizeRegistryUrl(value); callback() } catch { callback(new Error('请填写有效的仓库地址，例如 https://registry.example.com')) } }, trigger: 'blur' }],
  username: [{ validator: (_rule, value, callback) => callback(form.value.authentication && !value.trim() ? new Error('请填写认证用户名') : undefined), trigger: 'blur' }],
  password: [{ validator: (_rule, value, callback) => callback(form.value.authentication && !value && (!registry.value?.username || form.value.username.trim() !== registry.value.username) ? new Error('请填写密码或访问令牌') : undefined), trigger: 'blur' }],
  email: [{ type: 'email', message: '请填写有效的邮箱地址', trigger: 'blur' }],
}
watch(form, () => { testSuccess.value = false }, { deep: true })
function selectProvider(id: RegistryProvider) {
  selectedProvider.value = id
  const provider = registryProviders.find(item => item.id === id)!
  form.value.url = provider.url
  if (!form.value.name.trim() || registryProviders.some(item => item.label === form.value.name)) form.value.name = id === 'custom' ? '' : provider.label
  form.value.authentication = id !== 'custom'
  formRef.value?.clearValidate()
}
function goBack() { void router.push({ name: 'Registry', query: route.query }) }
function onRemoved(ids: number[]) { if (registry.value && ids.includes(registry.value.id)) goBack() }
function formatDate(value: string) { return value ? new Date(value).toLocaleString('zh-CN') : '—' }
async function loadRegistry() {
  controller?.abort()
  const current = new AbortController(); controller = current
  registry.value = null; form.value = emptyRegistryForm(); selectedProvider.value = 'custom'
  loadError.value = ''; actionError.value = ''; testSuccess.value = false; loading.value = false
  if (creating.value) return
  loading.value = true
  try {
    const result = await registryApi.get(Number(route.params.id), current.signal)
    if (current.signal.aborted || disposed) return
    if (result.kind === 'builtin') throw new Error('内置仓库请通过系统设置管理')
    registry.value = result
    selectedProvider.value = registryProvider(result.url)
    form.value = { ...emptyRegistryForm(), name: result.name, url: result.url, authentication: Boolean(result.username), username: result.username, email: result.email || '', pull_username: result.pull_username || '', pull_only_confirmed: result.pull_only_confirmed || false, is_default: result.is_default, insecure: result.insecure }
    formRef.value?.clearValidate()
  } catch (error) { if (!current.signal.aborted && !disposed) loadError.value = `仓库详情加载失败：${registryError(error)}` }
  finally { if (!current.signal.aborted) loading.value = false }
}
async function save() {
  if (busy.value || loading.value) return
  const context = controller
  if (!await formRef.value?.validate().catch(() => false)) return
  if (disposed || context !== controller) return
  actionError.value = ''; testSuccess.value = false
  try {
    const payload = buildRegistryRequest(form.value, !creating.value)
    if (payload.pull_username && (!registry.value?.pull_username || payload.pull_username !== registry.value.pull_username) && !payload.pull_password) throw new Error('请填写新账号的 Agent 拉取密码')
    saving.value = true
    if (creating.value) { await registryApi.create(payload); if (!disposed && context === controller) { ElMessage.success('仓库已添加'); goBack() } }
    else {
      const updated = await registryApi.update(registry.value!.id, payload)
      if (disposed || context !== controller) return
      registry.value = updated; form.value.password = ''; form.value.pull_password = ''
      selectedProvider.value = registryProvider(updated.url)
      ElMessage.success('仓库配置已保存')
    }
  } catch (error) { if (!disposed && context === controller) actionError.value = registryError(error) }
  finally { saving.value = false }
}
async function testConnection() {
  if (busy.value || loading.value) return
  const context = controller
  actionError.value = ''; testSuccess.value = false
  try {
    const url = normalizeRegistryUrl(form.value.url)
    if (form.value.authentication && (!form.value.username.trim() || (!form.value.password && (!registry.value?.username || form.value.username.trim() !== registry.value.username)))) throw new Error('请填写认证用户名和密码或访问令牌')
    testing.value = true
    await registryApi.test({ id: registry.value?.id, url, username: form.value.authentication ? form.value.username.trim() : '', password: form.value.authentication ? form.value.password : '', insecure: form.value.insecure, authentication: form.value.authentication })
    if (!disposed && context === controller) testSuccess.value = true
  } catch (error) { if (!disposed && context === controller) actionError.value = `连接测试失败：${registryError(error)}` }
  finally { testing.value = false }
}
watch(() => [route.name, route.params.id], loadRegistry, { immediate: true })
onBeforeUnmount(() => { disposed = true; controller?.abort() })
</script>
