<template>
  <div class="registry-resource builtin-page">
    <header class="registry-heading">
      <h1>内置仓库</h1>
      <div class="registry-heading-actions">
        <el-button :disabled="!loaded" @click="usageVisible = true"><PortainerIcon name="terminal" />客户端使用</el-button>
        <el-button text circle aria-label="刷新内置仓库" title="刷新内置仓库" :disabled="loading" @click="load"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button>
      </div>
    </header>
    <section class="registry-card builtin-overview" aria-label="仓库信息">
      <span class="registry-section-icon"><PortainerIcon name="registry" /></span>
      <div class="builtin-endpoint"><span class="registry-muted">仓库地址</span><div><code :title="data?.public_url">{{ loaded ? host : '—' }}</code><el-button v-if="loaded" link aria-label="复制仓库地址" title="复制仓库地址" @click="copyText(host, '仓库地址')"><el-icon><CopyDocument /></el-icon></el-button></div></div>
      <div class="builtin-overview-item"><span class="registry-muted">当前权限</span><strong>{{ permissionLabel }}</strong></div>
      <div class="builtin-overview-item"><span class="registry-muted">仓库状态</span><span v-if="loaded" class="builtin-state" :class="{ maintenance: data?.read_only }"><i />{{ data?.read_only ? '只读维护中' : '运行中' }}</span><span v-else class="registry-muted">{{ loading ? '加载中…' : '暂不可用' }}</span></div>
      <router-link v-if="auth.user?.is_admin" class="builtin-settings" to="/settings">仓库与维护<PortainerIcon name="chevron-down" /></router-link>
    </section>
    <div v-if="data?.read_only" class="registry-warning" role="status">仓库正在只读维护，仍可浏览和拉取镜像，推送与删除暂不可用。</div>
    <section class="registry-card registry-list builtin-list" aria-labelledby="builtin-list-title">
      <header class="registries-toolbar">
        <h2 id="builtin-list-title"><span class="registry-section-icon"><PortainerIcon :name="tab === 'images' ? 'box' : 'logs'" /></span>{{ tab === 'images' ? '镜像列表' : '最近提交' }}</h2>
        <div class="registries-actions"><el-input v-model="search" class="registry-search" :placeholder="tab === 'images' ? '搜索镜像名称…' : '搜索镜像、标签或摘要…'" aria-label="搜索内置仓库" clearable><template #prefix><PortainerIcon name="search" /></template></el-input></div>
      </header>
      <div class="builtin-list-controls">
        <div class="builtin-tabs" role="tablist" aria-label="仓库视图">
          <button id="builtin-images-tab" role="tab" :aria-selected="tab === 'images'" :tabindex="tab === 'images' ? 0 : -1" aria-controls="builtin-images-panel" :class="{ active: tab === 'images' }" @click="tab = 'images'" @keydown.right="switchTab('events', $event)" @keydown.left="switchTab('events', $event)">镜像<span>{{ repos.length }}</span></button>
          <button id="builtin-events-tab" role="tab" :aria-selected="tab === 'events'" :tabindex="tab === 'events' ? 0 : -1" aria-controls="builtin-events-panel" :class="{ active: tab === 'events' }" @click="tab = 'events'" @keydown.right="switchTab('images', $event)" @keydown.left="switchTab('images', $event)">最近提交<span>{{ events.length }}</span></button>
        </div>
        <el-select v-if="tab === 'images'" v-model="namespace" aria-label="筛选命名空间" placeholder="全部命名空间"><el-option label="全部命名空间" value="" /><el-option label="无命名空间" value="__root__" /><el-option v-for="item in namespaces" :key="item" :label="item" :value="item" /></el-select>
        <el-button v-if="search || namespace" link @click="resetFilters">重置筛选</el-button>
        <span class="builtin-list-hint">{{ tab === 'images' ? '选择镜像查看标签与版本' : '保留最近 500 条提交记录' }}</span>
      </div>
      <div v-if="listError" class="registry-error" role="alert"><span>仓库加载失败：{{ listError }}</span><el-button link :disabled="loading" @click="load">重试</el-button></div>
      <div v-if="tab === 'images'" id="builtin-images-panel" ref="tableScroll" v-loading="loading" role="tabpanel" aria-labelledby="builtin-images-tab" class="registry-table-scroll" tabindex="0" :aria-busy="loading">
        <table class="registry-table builtin-repository-table" aria-label="内置仓库镜像列表">
          <colgroup><col /><col class="builtin-namespace-col" /><col class="builtin-ref-col" /><col class="builtin-time-col" /><col class="builtin-action-col" /></colgroup>
          <thead><tr>
            <th scope="col" :aria-sort="sort === 'name' ? (descending ? 'descending' : 'ascending') : 'none'"><button class="registry-sort" @click="changeSort('name')">镜像名称<PortainerIcon name="sort" :class="{ 'sort-active': sort === 'name' }" /></button></th>
            <th scope="col">命名空间</th><th scope="col">最近提交引用</th>
            <th scope="col" :aria-sort="sort === 'updated' ? (descending ? 'descending' : 'ascending') : 'none'"><button class="registry-sort" @click="changeSort('updated')">最近提交时间<PortainerIcon name="sort" :class="{ 'sort-active': sort === 'updated' }" /></button></th><th scope="col">操作</th>
          </tr></thead>
          <tbody><tr v-for="name in pageRepos" :key="name">
            <td><div class="registry-image-name"><PortainerIcon name="box" /><button class="builtin-name-link" :title="name" :disabled="loading" @click="openRepository(name)">{{ name }}</button></div></td>
            <td><span class="registry-muted">{{ repositoryNamespace(name) || '—' }}</span></td><td><span class="builtin-truncate" :title="latest.get(name)?.reference">{{ latest.get(name)?.reference || '—' }}</span></td><td class="builtin-time">{{ builtinTime(latest.get(name)?.created_at) }}</td>
            <td><div class="builtin-row-actions"><el-button link :disabled="loading" @click="openRepository(name)"><PortainerIcon name="tag" />查看标签</el-button><el-button link :disabled="loading" :aria-label="`复制 ${name} 的镜像名称`" title="复制完整镜像名称" @click="copyText(`${host}/${name}`, '镜像名称')"><el-icon><CopyDocument /></el-icon></el-button></div></td>
          </tr></tbody>
        </table>
        <div v-if="!pageRepos.length && !loading" class="registry-empty" role="status"><PortainerIcon :name="search || namespace ? 'search' : 'box'" /><p>{{ listError ? '暂时无法加载镜像列表' : search || namespace ? '没有找到匹配的镜像' : '仓库中还没有镜像' }}</p><el-button v-if="search || namespace" link @click="resetFilters">重置筛选</el-button><el-button v-else-if="loaded" link @click="usageVisible = true">查看客户端使用方法</el-button></div>
      </div>
      <div v-else id="builtin-events-panel" ref="tableScroll" v-loading="loading" role="tabpanel" aria-labelledby="builtin-events-tab" class="registry-table-scroll" tabindex="0" :aria-busy="loading">
        <table class="registry-table builtin-events-table" aria-label="最近镜像提交">
          <colgroup><col /><col class="builtin-ref-col" /><col class="builtin-digest-col" /><col class="builtin-user-col" /><col class="builtin-time-col" /></colgroup>
          <thead><tr><th scope="col">镜像 / 来源</th><th scope="col">标签 / 引用</th><th scope="col">镜像摘要</th><th scope="col">发布人 ID</th><th scope="col">提交时间</th></tr></thead>
          <tbody><tr v-for="event in pageEvents" :key="event.id"><td><button class="builtin-name-link" :title="event.repository" :disabled="loading" @click="openRepository(event.repository, event.digest)">{{ event.repository }}</button><span class="builtin-event-source">{{ event.imported ? '发布包导入' : '客户端推送' }}</span></td><td><span class="builtin-truncate" :title="event.reference">{{ event.reference }}</span></td><td><div class="builtin-digest"><code class="builtin-truncate" :title="event.digest">{{ event.digest }}</code><el-button link :aria-label="`复制 ${event.repository} 的镜像摘要`" @click="copyText(event.digest, '镜像摘要')"><el-icon><CopyDocument /></el-icon></el-button></div></td><td>#{{ event.user_id }}</td><td class="builtin-time">{{ builtinTime(event.created_at) }}</td></tr></tbody>
        </table>
        <div v-if="!pageEvents.length && !loading" class="registry-empty" role="status"><PortainerIcon :name="search ? 'search' : 'logs'" /><p>{{ listError ? '暂时无法加载提交记录' : search ? '没有找到匹配的提交' : '暂无镜像提交记录' }}</p><el-button v-if="search" link @click="resetFilters">清除搜索</el-button></div>
      </div>
      <footer class="registries-pagination"><div class="registry-count" role="status">共 {{ total }} {{ tab === 'images' ? '个镜像' : '条提交' }}<span v-if="tab === 'images'" class="builtin-pagination-hint">提交信息来自最近 500 条记录</span></div><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="total" :pager-count="5" :disabled="loading" layout="sizes, prev, pager, next" /></footer>
    </section>
    <BuiltinUsageDialog v-model="usageVisible" :host="host" :public-url="data?.public_url || ''" :username="auth.user?.username || ''" :can-push="canPush" :read-only="Boolean(data?.read_only)" :permission="permissionLabel" />
    <BuiltinRepositoryDrawer v-model="drawerVisible" :repository="selected" :initial-reference="selectedReference" :host="host" :admin="Boolean(auth.user?.is_admin)" :read-only="Boolean(data?.read_only)" @removed="load" />
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import { builtinApi, type BuiltinCatalog } from '@/api/builtin'
import { useAuthStore } from '@/stores/auth'
import PortainerIcon from '@/components/PortainerIcon.vue'
import BuiltinUsageDialog from '@/components/builtin/BuiltinUsageDialog.vue'
import BuiltinRepositoryDrawer from '@/components/builtin/BuiltinRepositoryDrawer.vue'
import { builtinError, builtinPermission, builtinTime, filterBuiltinRepositories, latestImageEvents, registryHost, repositoryNamespace } from '@/utils/builtin'
import '@/styles/registries.css'
import '@/styles/builtin.css'

const auth = useAuthStore()
const data = ref<BuiltinCatalog>(), loaded = ref(false), loading = ref(false), listError = ref('')
const tab = ref<'images' | 'events'>('images'), search = ref(''), namespace = ref('')
const sort = ref<'name' | 'updated'>('name'), descending = ref(false), page = ref(1), pageSize = ref(10)
const usageVisible = ref(false), drawerVisible = ref(false), selected = ref(''), selectedReference = ref('')
const tableScroll = ref<HTMLElement>()
const repos = computed(() => data.value?.catalog?.repositories || [])
const events = computed(() => data.value?.events || [])
const host = computed(() => registryHost(data.value?.public_url || '', location.host))
const permission = computed(() => data.value?.permission || auth.user?.registry_access || 'none')
const permissionLabel = computed(() => builtinPermission(Boolean(auth.user?.is_admin), permission.value))
const canPush = computed(() => Boolean(auth.user?.is_admin || permission.value === 'read_write'))
const latest = computed(() => latestImageEvents(events.value))
const namespaces = computed(() => [...new Set(repos.value.map(repositoryNamespace).filter(Boolean))].sort())
const filteredRepos = computed(() => filterBuiltinRepositories(repos.value, search.value, namespace.value, sort.value, descending.value, latest.value))
const filteredEvents = computed(() => { const query = search.value.trim().toLowerCase(); return events.value.filter(event => `${event.repository} ${event.reference} ${event.digest}`.toLowerCase().includes(query)) })
const pageRepos = computed(() => filteredRepos.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const pageEvents = computed(() => filteredEvents.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const total = computed(() => tab.value === 'images' ? filteredRepos.value.length : filteredEvents.value.length)
const { copy } = useClipboard({ legacy: true })
const controller = new AbortController()
let disposed = false
watch([tab, search, namespace, pageSize], () => { page.value = 1; if (tab.value === 'events') namespace.value = '' })
watch(total, () => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total.value / pageSize.value))) })
watch([page, tab, search, namespace, sort, descending, pageSize], () => tableScroll.value?.scrollTo({ top: 0 }))
function resetFilters() { search.value = ''; namespace.value = '' }
function changeSort(value: 'name' | 'updated') { descending.value = sort.value === value ? !descending.value : value === 'updated'; sort.value = value; page.value = 1 }
function switchTab(value: 'images' | 'events', event: KeyboardEvent) { event.preventDefault(); tab.value = value; document.getElementById(`builtin-${value}-tab`)?.focus() }
function openRepository(name: string, reference = '') { selected.value = name; selectedReference.value = reference; drawerVisible.value = true }
async function copyText(value: string, label: string) { try { await copy(value); ElMessage.success(`${label}已复制`) } catch { ElMessage.error('复制失败，请重试') } }
async function load() {
  if (loading.value || disposed) return
  loading.value = true; listError.value = ''
  try { const result = await builtinApi.catalog(controller.signal); if (!disposed) { data.value = result; loaded.value = true } }
  catch (error) { if (!disposed) listError.value = builtinError(error) }
  finally { if (!disposed) loading.value = false }
}
onMounted(load)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
