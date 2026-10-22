<template>
  <div class="user-resource users-page">
    <header class="users-heading">
      <h1>用户管理</h1>
      <el-button text circle aria-label="刷新用户列表" title="刷新用户列表" :disabled="loading || editorVisible" @click="loadUsers"><PortainerIcon name="refresh" :class="{ spinning: loading }" /></el-button>
    </header>
    <section class="user-card users-list" aria-labelledby="users-title">
      <header class="users-toolbar">
        <h2 id="users-title"><span class="user-section-icon"><el-icon><User /></el-icon></span>用户列表</h2>
        <div class="users-actions">
          <el-input v-model="search" class="user-search" placeholder="搜索用户名…" aria-label="搜索用户" clearable><template #prefix><PortainerIcon name="search" /></template></el-input>
          <el-button class="user-primary" :disabled="loading || editorVisible" @click="openEditor('create')"><PortainerIcon name="plus" />创建用户</el-button>
        </div>
      </header>
      <form class="users-filters" @submit.prevent>
        <el-select v-model="roleFilter" placeholder="全部角色" aria-label="筛选用户角色"><el-option label="全部角色" value="" /><el-option label="管理员" value="admin" /><el-option label="普通用户" value="member" /></el-select>
        <el-select v-model="statusFilter" placeholder="全部状态" aria-label="筛选账号状态"><el-option label="全部状态" value="" /><el-option label="启用" value="enabled" /><el-option label="停用" value="disabled" /></el-select>
        <el-select v-model="accessFilter" placeholder="全部仓库权限" aria-label="筛选仓库权限"><el-option label="全部仓库权限" value="" /><el-option label="全部权限" value="all" /><el-option v-for="(label, value) in registryAccessLabels" :key="value" :label="label" :value="value" /></el-select>
        <el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button>
        <span class="users-summary">启用 <strong>{{ users.filter(user => user.enabled).length }}</strong><span>/</span>全部 <strong>{{ users.length }}</strong></span>
      </form>
      <div v-if="listError" class="users-error" role="alert"><span>{{ listError }}</span><el-button link :disabled="loading" @click="loadUsers">重试</el-button></div>
      <div ref="tableScroll" v-loading="loading" class="users-table-scroll" aria-label="用户列表滚动区域" tabindex="0" :aria-busy="loading">
        <table class="users-table" aria-label="用户列表">
          <colgroup><col class="user-name-col" /><col class="user-role-col" /><col class="user-status-col" /><col class="user-access-col" /><col class="user-date-col" /><col class="user-actions-col" /></colgroup>
          <thead><tr>
            <th v-for="column in userColumns" :key="column.key" scope="col" :aria-sort="sortKey === column.key ? (descending ? 'descending' : 'ascending') : 'none'"><button class="user-sort" @click="changeSort(column.key)">{{ column.label }}<PortainerIcon name="sort" :class="{ 'sort-active': sortKey === column.key }" /></button></th>
            <th scope="col">操作</th>
          </tr></thead>
          <tbody><tr v-for="user in pageUsers" :key="user.id">
            <td><div class="user-identity"><span class="user-avatar" aria-hidden="true"><el-icon><User /></el-icon></span><button class="user-name" :title="user.username" :disabled="loading || editorVisible" @click="openEditor('authorization', user)">{{ user.username }}</button><span v-if="user.id === auth.user?.id" class="user-badge">当前账号</span></div></td>
            <td><span :class="user.is_admin ? 'user-role user-role--admin' : 'user-muted'">{{ user.is_admin ? '管理员' : '普通用户' }}</span><span v-if="isLastEnabledAdmin(user, users)" class="user-cell-help">最后一个可用管理员</span></td>
            <td><span class="user-state" :class="user.enabled ? 'user-state--enabled' : 'user-state--disabled'">{{ user.enabled ? '启用' : '停用' }}</span></td>
            <td><span>{{ user.is_admin ? '全部权限' : registryAccessLabels[user.registry_access] || '未知权限' }}</span><span class="user-cell-help">{{ user.is_admin ? '管理平台、拉取与推送' : registryAccessDescriptions[user.registry_access] || '请检查用户授权' }}</span></td>
            <td class="user-date">{{ userDate(user.created_at) }}</td>
            <td><div class="user-row-actions"><el-button link :disabled="loading || editorVisible" :aria-label="`修改 ${user.username} 的授权`" @click="openEditor('authorization', user)">修改授权</el-button><span aria-hidden="true"></span><el-button link :disabled="loading || editorVisible" :aria-label="`重设 ${user.username} 的密码`" @click="openEditor('password', user)">重设密码</el-button></div></td>
          </tr></tbody>
        </table>
        <div v-if="!pageUsers.length && !loading" class="users-empty" role="status">
          <PortainerIcon v-if="hasFilters" name="search" /><el-icon v-else><User /></el-icon>
          <p>{{ listError ? '暂时无法加载用户列表' : hasFilters ? '没有找到匹配的用户' : '暂无用户' }}</p>
          <el-button v-if="hasFilters" link @click="resetFilters">重置筛选</el-button><span v-else-if="!listError">创建用户后，可在这里管理账号与授权</span>
        </div>
      </div>
      <footer class="users-pagination">
        <div class="users-count" role="status">共 {{ filteredUsers.length }} 个用户<span v-if="hasFilters">（全部 {{ users.length }} 个）</span></div>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 25, 50, 100]" :total="filteredUsers.length" :pager-count="5" :disabled="loading || editorVisible" layout="sizes, prev, pager, next" />
      </footer>
    </section>
    <p class="users-footnote"><PortainerIcon name="info" />账号共用于网页登录和 docker login；修改授权或密码后，已有登录与令牌将失效。</p>
    <UserEditorDialog v-model="editorVisible" :mode="editorMode" :user="editingUser" :users="users" :current-user-id="auth.user?.id" @saved="onSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { usersApi } from '@/api/users'
import { useAuthStore } from '@/stores/auth'
import PortainerIcon from '@/components/PortainerIcon.vue'
import UserEditorDialog from '@/components/users/UserEditorDialog.vue'
import { filterUsers, isLastEnabledAdmin, registryAccessDescriptions, registryAccessLabels, sortUsers, userColumns, userDate, userError, type ManagedUser, type UserEditorMode, type UserSortKey } from '@/utils/users'
import '@/styles/users.css'

const auth = useAuthStore(), router = useRouter()
const users = ref<ManagedUser[]>([])
const loading = ref(false), listError = ref(''), editorVisible = ref(false)
const search = ref(''), roleFilter = ref(''), statusFilter = ref(''), accessFilter = ref('')
const page = ref(1), pageSize = ref(10), sortKey = ref<UserSortKey>('username'), descending = ref(false)
const editorMode = ref<UserEditorMode>('create'), editingUser = ref<ManagedUser>()
const tableScroll = ref<HTMLElement>()
const hasFilters = computed(() => Boolean(search.value || roleFilter.value || statusFilter.value || accessFilter.value))
const filteredUsers = computed(() => sortUsers(filterUsers(users.value, search.value, roleFilter.value, statusFilter.value, accessFilter.value), sortKey.value, descending.value))
const pageUsers = computed(() => filteredUsers.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const controller = new AbortController()
let disposed = false
watch([search, roleFilter, statusFilter, accessFilter, pageSize], () => { page.value = 1 })
watch(() => filteredUsers.value.length, total => { if (!loading.value) page.value = Math.min(page.value, Math.max(1, Math.ceil(total / pageSize.value))) })
watch([page, pageSize, sortKey, descending, search, roleFilter, statusFilter, accessFilter], () => tableScroll.value?.scrollTo({ top: 0 }))
function resetFilters() { search.value = ''; roleFilter.value = ''; statusFilter.value = ''; accessFilter.value = '' }
function changeSort(key: UserSortKey) { descending.value = sortKey.value === key ? !descending.value : false; sortKey.value = key; page.value = 1 }
function openEditor(mode: UserEditorMode, user?: ManagedUser) {
  if (loading.value || editorVisible.value) return
  editorMode.value = mode; editingUser.value = user; editorVisible.value = true
}
async function loadUsers() {
  if (loading.value || disposed) return
  loading.value = true; listError.value = ''
  try {
    const result = await usersApi.list(controller.signal)
    if (disposed) return
    users.value = result || []
    page.value = Math.min(page.value, Math.max(1, Math.ceil(filteredUsers.value.length / pageSize.value)))
  } catch (error) { if (!disposed) listError.value = `用户列表加载失败：${userError(error)}` }
  finally { loading.value = false }
}
function onSaved(user: ManagedUser, mode: UserEditorMode) {
  if (mode !== 'create' && user.id === auth.user?.id) {
    auth.logout()
    ElMessage.success('当前账号已更新，请重新登录')
    void router.replace('/login')
    return
  }
  const index = users.value.findIndex(item => item.id === user.id)
  if (index < 0) users.value.push(user)
  else users.value[index] = user
  ElMessage.success(mode === 'create' ? `用户 ${user.username} 已创建` : mode === 'password' ? '密码已重设' : '用户授权已更新')
  void loadUsers()
}
onMounted(loadUsers)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
