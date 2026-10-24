<template>
  <WorkspaceEditor title="创建网络" :busy="creating" :refreshable="false">
    <template #actions><el-button :disabled="creating" @click="router.push({ name: 'Networks', query: listQuery })">取消</el-button><el-button native-type="submit" form="network-create-form" type="primary" :disabled="creating" :loading="creating">{{ creating ? '创建中…' : '创建网络' }}</el-button></template>
    <template v-if="createError" #feedback><div ref="errorElement" class="management-error" role="alert" tabindex="-1">{{ createError }}</div></template>
    <div class="network-resource network-create-page workspace-page workspace-editor-page">
    <form id="network-create-form" class="network-card network-create-form" @submit.prevent="createNetwork">
      <fieldset :disabled="creating">
        <div class="network-form-row"><label for="network-name">名称 <span class="network-required">*</span></label><el-input id="network-name" v-model="form.name" placeholder="例如 my-network" autocomplete="off" :disabled="creating" required /></div>
        <section class="network-form-section" aria-labelledby="driver-title">
          <h2 id="driver-title">驱动配置</h2>
          <div class="network-form-row"><label for="network-driver">驱动</label><el-select id="network-driver" v-model="form.driver" :disabled="creating"><el-option v-for="driver in ['bridge', 'overlay', 'macvlan', 'ipvlan']" :key="driver" :value="driver" :label="driver" /></el-select></div>
          <p class="network-form-help">{{ driverHelp[form.driver] }}</p>
          <h3>驱动选项</h3><NetworkKeyValues v-model="form.options" label="驱动选项" :disabled="creating" value-placeholder="例如 parent 对应 eth0" />
        </section>
        <section class="network-form-section" aria-labelledby="ipv4-title">
          <h2 id="ipv4-title">IPv4 网络配置</h2><p class="network-form-help">留空时由 Docker 自动分配子网和网关。</p>
          <div class="network-address-form">
            <div class="network-form-row"><label for="network-v4-subnet">子网</label><el-input id="network-v4-subnet" v-model="form.ipv4.subnet" placeholder="例如 172.20.0.0/16" :disabled="creating" /></div>
            <div class="network-form-row"><label for="network-v4-gateway">网关</label><el-input id="network-v4-gateway" v-model="form.ipv4.gateway" placeholder="例如 172.20.0.1" :disabled="creating" /></div>
            <div class="network-form-row"><label for="network-v4-range">IP 范围</label><el-input id="network-v4-range" v-model="form.ipv4.range" placeholder="例如 172.20.10.0/24" :disabled="creating" /></div>
          </div>
          <NetworkKeyValues v-model="form.ipv4.excluded" label="IPv4 排除 IP" :disabled="creating" value-placeholder="例如 172.20.0.10" />
        </section>
        <section class="network-form-section" aria-labelledby="ipv6-title">
          <div class="network-section-heading"><h2 id="ipv6-title">IPv6 网络配置</h2><el-switch v-model="form.enableIPv6" aria-label="启用 IPv6" :disabled="creating" /></div>
          <template v-if="form.enableIPv6">
            <p class="network-form-help">填写 IPv6 子网，或留空使用 Docker 的自动分配配置。</p>
            <div class="network-address-form">
              <div class="network-form-row"><label for="network-v6-subnet">子网</label><el-input id="network-v6-subnet" v-model="form.ipv6.subnet" placeholder="例如 fd00:20::/64" :disabled="creating" /></div>
              <div class="network-form-row"><label for="network-v6-gateway">网关</label><el-input id="network-v6-gateway" v-model="form.ipv6.gateway" placeholder="例如 fd00:20::1" :disabled="creating" /></div>
              <div class="network-form-row"><label for="network-v6-range">IP 范围</label><el-input id="network-v6-range" v-model="form.ipv6.range" placeholder="例如 fd00:20::/80" :disabled="creating" /></div>
            </div>
            <NetworkKeyValues v-model="form.ipv6.excluded" label="IPv6 排除 IP" :disabled="creating" value-placeholder="例如 fd00:20::10" />
          </template>
        </section>
        <section class="network-form-section" aria-labelledby="advanced-title">
          <h2 id="advanced-title">高级配置</h2><h3>标签</h3><NetworkKeyValues v-model="form.labels" label="标签" :disabled="creating" />
          <div class="network-toggle-row"><label for="network-internal">隔离网络<small>仅允许网络内部通信，禁止通过此网络访问外部网络。</small></label><el-switch id="network-internal" v-model="form.internal" :disabled="creating" /></div>
          <div class="network-toggle-row"><label for="network-attachable">允许手动连接容器<small>允许独立容器连接至此网络，适用于需要手动连接容器的 overlay 网络。</small></label><el-switch id="network-attachable" v-model="form.attachable" :disabled="creating" /></div>
        </section>
      </fieldset>
    </form>
    </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'
import NetworkKeyValues from '@/components/networks/NetworkKeyValues.vue'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import { useUnsavedChanges } from '@/composables/useUnsavedChanges'
import { buildNetworkRequest, emptyNetworkForm, networkListQuery, readNetworkListState } from '@/utils/networks'
import { networkError } from '@/utils/networkFeedback'
import '@/styles/networks.css'
const route = useRoute(), router = useRouter()
const listQuery = networkListQuery(readNetworkListState(route.query))
const form = ref(emptyNetworkForm())
const creating = ref(false), createError = ref(''), errorElement = ref<HTMLElement>()
const driverHelp: Record<string, string> = { bridge: 'bridge 用于同一主机上的容器通信，是独立容器的默认选择。', overlay: 'overlay 用于跨主机通信，需要 Docker 已启用 Swarm。', macvlan: 'macvlan 将容器连接到物理网络；请添加 parent 驱动选项并指定父网卡。', ipvlan: 'ipvlan 让容器共享父网卡的 MAC 地址；请添加 parent 驱动选项并指定父网卡。' }
const { markSaved } = useUnsavedChanges(() => form.value, { busy: () => creating.value })
async function createNetwork() {
  if (creating.value) return
  createError.value = ''
  try {
    const request = buildNetworkRequest(form.value)
    creating.value = true
    const result = await dockerApi.networks.create(request)
    if (result.Warning) ElMessage.warning(result.Warning)
    ElMessage.success('网络创建成功')
    markSaved()
    creating.value = false
    await router.replace({ name: 'NetworkDetail', params: { id: result.Id }, query: listQuery })
  } catch (error) { createError.value = networkError(error); await nextTick(); errorElement.value?.focus({ preventScroll: true }) }
  finally { creating.value = false }
}
</script>
