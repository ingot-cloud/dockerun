<template>
  <WorkspaceEditor title="创建卷" :busy="creating" :refreshable="false">
    <template #actions><el-button :disabled="creating" @click="router.push({ name: 'Volumes', query: listQuery })">取消</el-button><el-button native-type="submit" form="volume-create-form" type="primary" :disabled="creating" :loading="creating">{{ creating ? '创建中…' : '创建卷' }}</el-button></template>
    <template v-if="createError" #feedback><div ref="errorElement" class="management-error" role="alert" tabindex="-1">{{ createError }}</div></template>
    <div class="volume-resource volume-create-page workspace-page workspace-editor-page">
    <form id="volume-create-form" class="volume-card volume-create-form" @submit.prevent="createVolume">
      <fieldset :disabled="creating">
        <div class="volume-form-row"><label for="volume-name">名称</label><el-input id="volume-name" v-model="form.name" placeholder="例如 my-volume；留空自动生成名称" autocomplete="off" :disabled="creating" /></div>
        <p class="volume-form-help volume-name-help">留空时由 Docker 生成随机名称。</p>
        <section class="volume-form-section" aria-labelledby="volume-driver-title">
          <h2 id="volume-driver-title">驱动配置</h2>
          <div class="volume-form-row"><label for="volume-driver">驱动</label><el-select id="volume-driver" v-model="form.driver" filterable allow-create default-first-option :reserve-keyword="false" :disabled="creating" placeholder="选择或输入已安装的卷驱动"><el-option v-for="driver in drivers" :key="driver" :value="driver" :label="driver" /></el-select></div>
          <p class="volume-form-help">local 适用于本机存储、NFS 和 CIFS；也可输入 Docker 主机上已安装的卷驱动名称。</p>
          <template v-if="form.mode === 'custom'"><h3>驱动选项</h3><VolumeKeyValues v-model="form.options" label="驱动选项" :disabled="creating" /></template>
          <template v-if="form.driver === 'local'">
            <div class="volume-toggle-row"><label for="volume-nfs">使用 NFS 卷</label><el-switch id="volume-nfs" :model-value="form.mode === 'nfs'" :disabled="creating" @change="form.mode = $event ? 'nfs' : 'custom'" /></div>
            <div class="volume-toggle-row"><label for="volume-cifs">使用 CIFS 卷</label><el-switch id="volume-cifs" :model-value="form.mode === 'cifs'" :disabled="creating" @change="form.mode = $event ? 'cifs' : 'custom'" /></div>
          </template>
        </section>
        <section v-if="form.mode === 'nfs'" class="volume-form-section" aria-labelledby="volume-nfs-title">
          <h2 id="volume-nfs-title">NFS 配置</h2><p class="volume-form-help">Docker 主机需要安装 nfs-utils，并能访问 NFS 服务器。</p>
          <div class="volume-remote-fields">
            <div class="volume-form-row"><label for="volume-nfs-address">服务器地址 <span class="volume-required">*</span></label><el-input id="volume-nfs-address" v-model="form.nfs.address" placeholder="例如 192.168.1.10 或 nas.example.com" :disabled="creating" /></div>
            <div class="volume-form-row"><label for="volume-nfs-version">NFS 版本</label><el-select id="volume-nfs-version" v-model="form.nfs.version" :disabled="creating"><el-option label="NFS v4" value="nfs4" /><el-option label="NFS v3" value="nfs" /></el-select></div>
            <div class="volume-form-row"><label for="volume-nfs-path">挂载路径 <span class="volume-required">*</span></label><el-input id="volume-nfs-path" v-model="form.nfs.path" placeholder="例如 /export/data" :disabled="creating" /></div>
            <div class="volume-form-row"><label for="volume-nfs-options">挂载选项</label><el-input id="volume-nfs-options" v-model="form.nfs.options" placeholder="例如 rw,noatime,tcp" :disabled="creating" /></div>
          </div>
        </section>
        <section v-if="form.mode === 'cifs'" class="volume-form-section" aria-labelledby="volume-cifs-title">
          <h2 id="volume-cifs-title">CIFS 配置</h2><p class="volume-form-help">Docker 主机需要安装 cifs-utils，并能访问共享目录。</p>
          <div class="volume-remote-fields">
            <div class="volume-form-row"><label for="volume-cifs-address">服务器地址 <span class="volume-required">*</span></label><el-input id="volume-cifs-address" v-model="form.cifs.address" placeholder="例如 192.168.1.10" :disabled="creating" /></div>
            <div class="volume-form-row"><label for="volume-cifs-share">共享目录 <span class="volume-required">*</span></label><el-input id="volume-cifs-share" v-model="form.cifs.share" placeholder="例如 shared/data" :disabled="creating" /></div>
            <div class="volume-form-row"><label for="volume-cifs-version">CIFS 版本</label><el-select id="volume-cifs-version" v-model="form.cifs.version" :disabled="creating"><el-option v-for="version in ['1.0', '2.0', '2.1', '3.0', '3.1.1']" :key="version" :label="`CIFS ${version}`" :value="version" /></el-select></div>
            <div class="volume-form-row"><label for="volume-cifs-username">用户名 <span class="volume-required">*</span></label><el-input id="volume-cifs-username" v-model="form.cifs.username" autocomplete="off" :disabled="creating" /></div>
            <div class="volume-form-row"><label for="volume-cifs-password">密码</label><el-input id="volume-cifs-password" v-model="form.cifs.password" type="password" show-password autocomplete="new-password" :disabled="creating" /></div>
          </div>
          <p class="volume-form-help">CIFS 凭据会保存为 Docker 卷驱动选项。</p>
        </section>
        <section class="volume-form-section" aria-labelledby="volume-labels-title"><h2 id="volume-labels-title">标签</h2><VolumeKeyValues v-model="form.labels" label="标签" :disabled="creating" /></section>
      </fieldset>
    </form>
    </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'
import VolumeKeyValues from '@/components/volumes/VolumeKeyValues.vue'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import { useUnsavedChanges } from '@/composables/useUnsavedChanges'
import { buildVolumeRequest, emptyVolumeForm, volumeListQuery, readVolumeListState } from '@/utils/volumes'
import { volumeError } from '@/utils/volumeFeedback'
import '@/styles/volumes.css'
const router = useRouter(), listQuery = volumeListQuery(readVolumeListState(useRoute().query))
const form = ref(emptyVolumeForm()), drivers = ref(['local']), creating = ref(false), createError = ref(''), errorElement = ref<HTMLElement>()
const controller = new AbortController()
let disposed = false
watch(() => form.value.driver, driver => { if (driver !== 'local') form.value.mode = 'custom' })
const { markSaved } = useUnsavedChanges(() => form.value, { busy: () => creating.value })
onMounted(async () => { try { const response = await dockerApi.volumes.list(controller.signal); if (!disposed) drivers.value = [...new Set(['local', ...(response.Volumes || []).map(volume => volume.Driver)])].sort() } catch { /* Manual driver entry remains available. */ } })
onBeforeUnmount(() => { disposed = true; controller.abort() })
async function createVolume() {
  if (creating.value) return
  createError.value = ''
  try {
    const request = buildVolumeRequest(form.value)
    creating.value = true
    const result = await dockerApi.volumes.create(request)
    if (disposed) return
    ElMessage.success('卷创建成功'); markSaved(); creating.value = false
    await router.replace({ name: 'VolumeDetail', params: { name: result.Name }, query: listQuery })
  } catch (error) { if (!disposed) { createError.value = volumeError(error); await nextTick(); errorElement.value?.focus({ preventScroll: true }) } }
  finally { creating.value = false }
}
</script>
