<template>
  <section class="registry-card registry-list maintenance-history" aria-labelledby="maintenance-history-title">
    <header class="registries-toolbar">
      <h2 id="maintenance-history-title"><span class="registry-section-icon"><PortainerIcon name="logs" /></span>任务与审计</h2>
      <div class="registries-actions"><el-input v-model="search" class="registry-search" :placeholder="tab === 'audit' ? '搜索操作、对象或用户…' : '搜索任务、文件或失败原因…'" aria-label="搜索任务与审计记录" clearable><template #prefix><PortainerIcon name="search" /></template></el-input></div>
    </header>
    <div class="maintenance-history-controls">
      <div class="maintenance-tabs" role="tablist" aria-label="维护记录类型"><button v-for="(item, index) in tabs" :id="`maintenance-${item.key}-tab`" :key="item.key" role="tab" :aria-selected="tab === item.key" :tabindex="tab === item.key ? 0 : -1" aria-controls="maintenance-history-panel" :class="{ active: tab === item.key }" @click="tab = item.key" @keydown.right="moveTab(index + 1, $event)" @keydown.left="moveTab(index - 1, $event)">{{ item.label }}<span>{{ item.count }}</span></button></div>
      <el-select v-if="tab !== 'audit'" v-model="statusFilter" aria-label="筛选任务状态" placeholder="全部状态"><el-option label="全部状态" value="" /><el-option label="排队 / 执行中" value="active" /><el-option label="已完成" value="complete" /><el-option label="失败" value="failed" /></el-select>
      <el-select v-else v-model="category" aria-label="筛选审计类别" placeholder="全部操作"><el-option label="全部操作" value="" /><el-option v-for="item in auditCategories" :key="item.value" :label="item.label" :value="item.value" /></el-select>
      <el-button v-if="search || statusFilter || category" link @click="resetFilters">重置筛选</el-button>
      <span class="maintenance-history-hint">{{ polling ? '任务执行中，自动更新' : tab === 'audit' ? '最近 200 条审计事件' : '最近 100 条任务记录' }}</span>
    </div>
    <div v-if="currentError" class="registry-error" role="alert"><span>{{ currentError }}</span><el-button link :disabled="loading || busy" @click="emit('refresh')">重试</el-button></div>
    <div id="maintenance-history-panel" ref="scrollRegion" v-loading="loading" class="registry-table-scroll" role="tabpanel" :aria-labelledby="`maintenance-${tab}-tab`" tabindex="0" :aria-busy="loading">
      <table v-if="tab !== 'audit'" class="registry-table maintenance-jobs-table" :aria-label="tab === 'backups' ? '完整备份任务' : '镜像层清理任务'">
        <colgroup><col class="maintenance-id-col" /><col class="maintenance-status-col" /><col /><col class="maintenance-date-col" /><col class="maintenance-user-col" /><col class="maintenance-actions-col" /></colgroup>
        <thead><tr><th scope="col" :aria-sort="sort === 'id' ? (descending ? 'descending' : 'ascending') : 'none'"><button class="registry-sort" @click="changeSort('id')">任务<PortainerIcon name="sort" :class="{ 'sort-active': sort === 'id' }" /></button></th><th scope="col">状态</th><th scope="col">{{ tab === 'backups' ? '备份文件 / 结果' : '清理内容 / 结果' }}</th><th scope="col" :aria-sort="sort === 'created_at' ? (descending ? 'descending' : 'ascending') : 'none'"><button class="registry-sort" @click="changeSort('created_at')">创建时间<PortainerIcon name="sort" :class="{ 'sort-active': sort === 'created_at' }" /></button></th><th scope="col">创建人</th><th scope="col">操作</th></tr></thead>
        <tbody><tr v-for="job in pageJobs" :key="job.id"><td class="maintenance-job-id">#{{ job.id }}</td><td><StatusBadge :value="job.status" /></td><td><span class="maintenance-file" :title="job.file">{{ tab === 'backups' ? job.file || (job.status === 'failed' ? '备份未完成' : '正在准备备份文件') : '清理未引用镜像层' }}</span><span v-if="job.status === 'queued'" class="maintenance-job-note">等待维护窗口，后台会自动执行</span><span v-else-if="job.status === 'running'" class="maintenance-job-note">后台执行中，离开页面后仍会继续</span><details v-if="job.error" class="maintenance-job-error"><summary>查看失败原因</summary><p>{{ job.error }}</p></details></td><td class="maintenance-date" :title="`最后更新：${builtinTime(job.updated_at)}`">{{ builtinTime(job.created_at) }}</td><td class="registry-muted">#{{ job.user_id }}</td><td><el-button v-if="tab === 'backups' && job.status === 'ready'" link :loading="downloading === job.id" :disabled="Boolean(downloading) || busy" @click="emit('download', job)"><PortainerIcon name="download" />{{ downloading === job.id ? '下载中' : '下载' }}</el-button><el-button v-else-if="job.status === 'failed'" link :disabled="busy || loading || (tab === 'collections' && !canCollect) || activeKind" :title="tab === 'collections' && !canCollect ? '请先开启只读维护，并等待当前维护任务完成' : activeKind ? '已有同类任务等待执行或执行中' : '创建新的任务重试'" @click="emit('retry', job.kind)"><PortainerIcon name="refresh" />重新执行</el-button><span v-else class="registry-muted">—</span></td></tr></tbody>
      </table>
      <table v-else class="registry-table maintenance-audit-table" aria-label="审计事件"><colgroup><col class="maintenance-date-col" /><col class="maintenance-user-col" /><col class="maintenance-audit-action-col" /><col /></colgroup><thead><tr><th scope="col">时间</th><th scope="col">用户 ID</th><th scope="col">操作</th><th scope="col">对象</th></tr></thead><tbody><tr v-for="event in pageAudit" :key="event.id"><td class="maintenance-date">{{ builtinTime(event.created_at) }}</td><td class="registry-muted">#{{ event.user_id }}</td><td>{{ maintenanceActions[event.action] || event.action }}<code class="maintenance-job-note">{{ event.action }}</code></td><td><span class="maintenance-file" :title="event.resource">{{ event.resource || '—' }}</span></td></tr></tbody></table>
      <div v-if="!total && !loading" class="registry-empty" role="status"><PortainerIcon :name="hasFilters ? 'search' : 'logs'" /><p>{{ currentError ? '暂时无法加载记录' : hasFilters ? '没有找到匹配的记录' : tab === 'backups' ? '暂无完整备份任务' : tab === 'collections' ? '暂无镜像层清理任务' : '暂无审计事件' }}</p><el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button><span v-else-if="!currentError">{{ tab === 'backups' ? '创建完整备份后，可在这里查看进度并下载结果' : tab === 'collections' ? '清理任务将在后台执行，完成后保留执行结果' : '仓库维护与账号操作会记录在这里' }}</span></div>
    </div>
    <footer class="registries-pagination"><div class="registry-count" role="status">共 {{ total }} 条{{ tab === 'audit' ? '审计事件' : '任务记录' }}<span v-if="lastUpdated" class="maintenance-updated">更新于 {{ lastUpdated }}</span></div><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="total" :pager-count="5" :disabled="loading" layout="sizes, prev, pager, next" /></footer>
  </section>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AuditEvent, MaintenanceJob, MaintenanceKind } from '@/api/maintenance'
import { filterMaintenanceAudit, filterMaintenanceJobs, isActiveMaintenanceJob, maintenanceActions } from '@/utils/maintenance'
import { builtinTime } from '@/utils/builtin'
import PortainerIcon from '@/components/PortainerIcon.vue'
import StatusBadge from '@/components/StatusBadge.vue'
const props = defineProps<{ backups: MaintenanceJob[]; collections: MaintenanceJob[]; events: AuditEvent[]; loading: boolean; errors: { backups: string; collections: string; audit: string }; busy: boolean; polling: boolean; downloading: number; canCollect: boolean; lastUpdated: string }>()
const emit = defineEmits<{ refresh: []; download: [job: MaintenanceJob]; retry: [kind: MaintenanceKind] }>()
const tab = ref<'backups' | 'collections' | 'audit'>('backups'), search = ref(''), statusFilter = ref(''), category = ref('')
const page = ref(1), pageSize = ref(10), sort = ref<'id' | 'created_at'>('id'), descending = ref(true)
const scrollRegion = ref<HTMLElement>()
const tabs = computed(() => [{ key: 'backups' as const, label: '完整备份', count: props.backups.length }, { key: 'collections' as const, label: '镜像层清理', count: props.collections.length }, { key: 'audit' as const, label: '审计事件', count: props.events.length }])
const auditCategories = [{ value: 'registry', label: '仓库与清理' }, { value: 'backup', label: '完整备份' }, { value: 'user', label: '用户管理' }, { value: 'token', label: '访问令牌' }, { value: 'deployment', label: '部署' }, { value: 'bundle', label: '发布包' }, { value: 'environment', label: '应用环境' }, { value: 'target', label: '目标主机' }]
const currentJobs = computed(() => tab.value === 'backups' ? props.backups : props.collections)
const activeKind = computed(() => currentJobs.value.some(isActiveMaintenanceJob))
const jobs = computed(() => filterMaintenanceJobs(currentJobs.value, search.value, statusFilter.value).slice().sort((a, b) => ((sort.value === 'id' ? a.id - b.id : Date.parse(a.created_at) - Date.parse(b.created_at)) || a.id - b.id) * (descending.value ? -1 : 1)))
const audit = computed(() => filterMaintenanceAudit(props.events, search.value, category.value))
const total = computed(() => tab.value === 'audit' ? audit.value.length : jobs.value.length)
const pageJobs = computed(() => jobs.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const pageAudit = computed(() => audit.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const currentError = computed(() => props.errors[tab.value])
const hasFilters = computed(() => Boolean(search.value || statusFilter.value || category.value))
watch(tab, () => { resetFilters(); page.value = 1 })
watch([search, statusFilter, category, pageSize], () => { page.value = 1 })
watch(total, () => { page.value = Math.min(page.value, Math.max(1, Math.ceil(total.value / pageSize.value))) })
watch([page, tab, search, statusFilter, category, pageSize, sort, descending], () => scrollRegion.value?.scrollTo({ top: 0 }))
function resetFilters() { search.value = ''; statusFilter.value = ''; category.value = '' }
function changeSort(value: 'id' | 'created_at') { descending.value = sort.value === value ? !descending.value : true; sort.value = value; page.value = 1 }
function moveTab(index: number, event: KeyboardEvent) { event.preventDefault(); tab.value = tabs.value[(index + tabs.value.length) % tabs.value.length].key; document.getElementById(`maintenance-${tab.value}-tab`)?.focus() }
defineExpose({ show: (kind: MaintenanceKind) => { tab.value = kind === 'backup' ? 'backups' : 'collections'; resetFilters(); page.value = 1 } })
</script>
