<template>
  <div class="network-resource network-detail-page" v-loading="loading">
    <header class="network-heading"><h1>网络详情<span v-if="network" class="network-heading-name">{{ network.Name }}</span></h1><div class="network-heading-actions"><router-link :to="{ name: 'Networks', query: listQuery }">返回网络列表</router-link><el-button text circle aria-label="刷新网络详情" title="刷新网络详情" :disabled="busy || loading" @click="loadNetwork"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button></div></header>
    <div v-if="detailError" class="network-error" role="alert"><span>{{ detailError }}</span><el-button link :disabled="loading" @click="loadNetwork">重试</el-button></div>
    <template v-if="network">
      <section class="network-card" aria-labelledby="network-details-title">
        <header class="network-card-header"><h2 id="network-details-title"><span class="network-section-icon"><PortainerIcon name="network" /></span>网络详情</h2><el-tooltip :content="deleteReason" :disabled="!deleteReason"><span><el-button class="network-remove" :disabled="!!deleteReason || busy || loading || !!detailError" @click="deleteVisible = true"><PortainerIcon name="trash" />删除此网络</el-button></span></el-tooltip></header>
        <dl class="network-details">
          <div><dt>名称</dt><dd>{{ network.Name }}<span v-if="isSystemNetwork(network)" class="network-badge">系统</span><span v-else-if="network.Ingress" class="network-badge">入口网络</span></dd></div>
          <div><dt>ID</dt><dd class="network-mono">{{ network.Id }}</dd></div>
          <div><dt>驱动</dt><dd>{{ network.Driver }}</dd></div>
          <div><dt>范围</dt><dd>{{ network.Scope || '—' }}</dd></div>
          <div v-if="network.Created"><dt>创建时间</dt><dd>{{ formatCreated(network.Created) }}</dd></div>
          <div v-if="networkStack(network)"><dt>应用栈</dt><dd>{{ networkStack(network) }}</dd></div>
          <div><dt>可手动连接</dt><dd>{{ network.Attachable ? '是' : '否' }}</dd></div>
          <div><dt>隔离网络</dt><dd>{{ network.Internal ? '是' : '否' }}</dd></div>
          <div><dt>IPv6</dt><dd>{{ network.EnableIPv6 ? '已启用' : '未启用' }}</dd></div>
          <div><dt>地址分配驱动</dt><dd>{{ network.IPAM?.Driver || 'default' }}</dd></div>
        </dl>
        <div v-if="network.IPAM?.Config?.length" class="network-detail-addresses"><section v-for="(config, index) in network.IPAM.Config" :key="index" class="network-ipam-block"><h3>{{ config.Subnet?.includes(':') ? 'IPv6' : 'IPv4' }} 网络配置</h3><dl class="network-details"><div><dt>子网</dt><dd class="network-mono">{{ config.Subnet || '—' }}</dd></div><div><dt>网关</dt><dd class="network-mono">{{ config.Gateway || '—' }}</dd></div><div><dt>IP 范围</dt><dd class="network-mono">{{ config.IPRange || '—' }}</dd></div><div><dt>排除 IP</dt><dd><template v-if="Object.keys(config.AuxiliaryAddresses || {}).length"><span v-for="(ip, name) in config.AuxiliaryAddresses" :key="name" class="network-detail-entry">{{ name }}：{{ ip }}</span></template><span v-else>—</span></dd></div></dl></section></div>
      </section>
      <section v-if="Object.keys(network.Options || {}).length" class="network-card" aria-labelledby="network-options-title"><header class="network-card-header"><h2 id="network-options-title"><span class="network-section-icon"><PortainerIcon name="network" /></span>网络选项</h2></header><dl class="network-details network-map"><div v-for="(value, key) in network.Options" :key="key"><dt>{{ key }}</dt><dd>{{ value || '—' }}</dd></div></dl></section>
      <section v-if="Object.keys(network.IPAM?.Options || {}).length" class="network-card"><header class="network-card-header"><h2>地址分配选项</h2></header><dl class="network-details network-map"><div v-for="(value, key) in network.IPAM?.Options" :key="key"><dt>{{ key }}</dt><dd>{{ value || '—' }}</dd></div></dl></section>
      <section v-if="Object.keys(network.Labels || {}).length" class="network-card" aria-labelledby="network-labels-title"><header class="network-card-header"><h2 id="network-labels-title"><span class="network-section-icon"><PortainerIcon name="tag" /></span>标签</h2></header><dl class="network-details network-map"><div v-for="(value, key) in network.Labels" :key="key"><dt>{{ key }}</dt><dd>{{ value || '—' }}</dd></div></dl></section>
      <section class="network-card" aria-labelledby="network-containers-title">
        <header class="network-card-header"><h2 id="network-containers-title"><span class="network-section-icon"><PortainerIcon name="box" /></span>已连接的容器<span class="network-muted">{{ endpoints.length }}</span></h2></header>
        <div v-if="actionError" class="network-error" role="alert">{{ actionError }}</div>
        <div class="network-endpoint-scroll"><table class="network-endpoint-table" aria-label="网络关联容器"><thead><tr><th scope="col">容器名称</th><th scope="col">IPv4 地址</th><th scope="col">IPv6 地址</th><th scope="col">MAC 地址</th><th scope="col">操作</th></tr></thead><tbody><tr v-for="endpoint in endpoints" :key="endpoint.id"><td><router-link :to="{ name: 'ContainerDetail', params: { id: endpoint.id } }">{{ endpoint.Name || endpoint.id.slice(0, 12) }}</router-link></td><td>{{ endpoint.IPv4Address || '—' }}</td><td>{{ endpoint.IPv6Address || '—' }}</td><td>{{ endpoint.MacAddress || '—' }}</td><td><el-button link type="danger" :disabled="busy || loading || !!detailError || isSystemNetwork(network)" :loading="disconnecting === endpoint.id" @click="disconnect(endpoint.id, endpoint.Name)">断开连接</el-button></td></tr><tr v-if="!endpoints.length"><td colspan="5" class="network-endpoint-empty">暂无容器连接到此网络</td></tr></tbody></table></div>
        <form v-if="allowConnect" class="network-connect-form" @submit.prevent="connect">
          <label for="network-connect-container">连接容器</label><el-input id="network-connect-container" v-model="containerInput" placeholder="输入容器名称或 ID" :disabled="busy || loading || !!detailError" /><el-button class="network-primary" native-type="submit" :loading="connecting" :disabled="busy || loading || !!detailError || !containerInput.trim()"><PortainerIcon name="link" />连接</el-button>
        </form>
      </section>
    </template>
    <NetworkDeleteDialog v-model="deleteVisible" :networks="network ? [network] : []" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PortainerIcon from '@/components/PortainerIcon.vue'
import NetworkDeleteDialog from '@/components/networks/NetworkDeleteDialog.vue'
import { dockerApi } from '@/api/docker'
import { isSystemNetwork, networkListQuery, networkStack, readNetworkListState, type DockerNetwork } from '@/utils/networks'
import { networkError } from '@/utils/networkFeedback'
import '@/styles/networks.css'
const route = useRoute(), router = useRouter()
const listQuery = computed(() => networkListQuery(readNetworkListState(route.query)))
const network = ref<DockerNetwork | null>(null)
const loading = ref(false), detailError = ref(''), actionError = ref(''), deleteVisible = ref(false)
const disconnecting = ref(''), connecting = ref(false), confirming = ref(false), containerInput = ref('')
const busy = computed(() => deleteVisible.value || Boolean(disconnecting.value) || connecting.value || confirming.value)
const endpoints = computed(() => Object.entries(network.value?.Containers || {}).map(([id, endpoint]) => ({ id, ...endpoint })))
const deleteReason = computed(() => !network.value ? '' : isSystemNetwork(network.value) ? '系统网络不可删除' : network.value.Ingress ? 'Swarm 入口网络不可删除' : endpoints.value.length ? '请先断开全部容器连接' : '')
const allowConnect = computed(() => network.value && !isSystemNetwork(network.value) && !network.value.Ingress && (network.value.Driver !== 'overlay' || network.value.Attachable))
const formatCreated = (value: string) => Number.isNaN(Date.parse(value)) ? value : new Date(value).toLocaleString('zh-CN', { hour12: false })
let controller: AbortController | undefined
let disposed = false
onBeforeRouteLeave(() => !busy.value)
onBeforeRouteUpdate(() => !busy.value)
function onRemoved() { deleteVisible.value = false; network.value = null; void router.replace({ name: 'Networks', query: listQuery.value }) }
async function loadNetwork() {
  controller?.abort()
  const current = new AbortController(); controller = current
  loading.value = true; detailError.value = ''
  try { const result = await dockerApi.networks.inspect(String(route.params.id), current.signal); if (!current.signal.aborted && !disposed) network.value = result }
  catch (error) { if (!current.signal.aborted && !disposed) detailError.value = `网络详情加载失败：${networkError(error)}` }
  finally { if (!current.signal.aborted) loading.value = false }
}
async function disconnect(id: string, name: string) {
  if (busy.value || !network.value) return
  confirming.value = true
  try { await ElMessageBox.confirm(`确定将容器「${name || id.slice(0, 12)}」从网络「${network.value.Name}」断开？依赖此网络的通信会中断。`, '断开容器连接', { confirmButtonText: '断开连接', cancelButtonText: '取消', type: 'warning' }) }
  catch { return }
  finally { confirming.value = false }
  if (disposed) return
  const networkId = network.value.Id
  disconnecting.value = id; actionError.value = ''
  try { await dockerApi.networks.disconnect(networkId, id); if (disposed || network.value?.Id !== networkId) return; ElMessage.success('已断开容器连接'); await loadNetwork() }
  catch (error) { if (!disposed) actionError.value = `断开连接失败：${networkError(error)}` }
  finally { disconnecting.value = '' }
}
async function connect() {
  if (busy.value || !network.value || !containerInput.value.trim() || !allowConnect.value) return
  const networkId = network.value.Id
  connecting.value = true; actionError.value = ''
  try { await dockerApi.networks.connect(networkId, containerInput.value.trim()); if (disposed || network.value?.Id !== networkId) return; containerInput.value = ''; ElMessage.success('容器已连接到网络'); await loadNetwork() }
  catch (error) { if (!disposed) actionError.value = `连接失败：${networkError(error)}` }
  finally { connecting.value = false }
}
watch(() => route.params.id, () => { network.value = null; actionError.value = ''; containerInput.value = ''; void loadNetwork() }, { immediate: true })
onBeforeUnmount(() => { disposed = true; controller?.abort() })
</script>
