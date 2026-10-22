<template>
  <el-dialog
    :model-value="modelValue" :title="title" width="580px" class="user-editor user-resource"
    append-to-body align-center destroy-on-close :close-on-click-modal="false" :close-on-press-escape="!saving" :show-close="!saving"
    @update:model-value="close" @closed="reset"
  >
    <div v-if="user" class="user-editor-identity">
      <span class="user-avatar"><el-icon><User /></el-icon></span>
      <div><strong>{{ user.username }}</strong><span>{{ user.is_admin ? '管理员' : '普通用户' }} · {{ user.enabled ? '启用' : '停用' }}</span></div>
      <span v-if="isSelf" class="user-badge">当前账号</span>
    </div>
    <p class="user-editor-description">{{ description }}</p>
    <div v-if="submitError" class="user-form-error" role="alert">{{ submitError }}</div>
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="saving" @submit.prevent="save">
      <el-form-item v-if="mode === 'create'" label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="输入用户名" autocomplete="off" autofocus />
      </el-form-item>
      <template v-if="mode !== 'authorization'">
        <el-form-item :label="mode === 'create' ? '初始密码' : '新密码'" prop="password">
          <el-input v-model="form.password" type="password" show-password autocomplete="new-password" placeholder="输入 6–72 字节密码" @input="revalidateConfirmation" />
          <p class="user-field-help">密码长度为 6–72 字节，中文通常占 3 字节。</p>
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="form.confirmPassword" type="password" show-password autocomplete="new-password" placeholder="再次输入密码" />
        </el-form-item>
      </template>
      <template v-if="mode !== 'password'">
        <el-form-item label="用户角色" prop="is_admin">
          <el-radio-group v-model="form.is_admin" class="user-role-options" :disabled="protectedAdmin">
            <el-radio :value="false" border><span>普通用户<small>按授权访问内置镜像仓库</small></span></el-radio>
            <el-radio :value="true" border><span>管理员<small>管理平台及全部镜像仓库</small></span></el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="仓库权限" prop="registry_access">
          <div v-if="form.is_admin" class="user-access-all"><el-icon><CircleCheck /></el-icon><span>全部权限<small>管理员拥有内置仓库的拉取与推送权限</small></span></div>
          <template v-else>
            <el-select v-model="form.registry_access" aria-label="仓库权限">
              <el-option v-for="(label, value) in registryAccessLabels" :key="value" :label="`${label} · ${registryAccessDescriptions[value]}`" :value="value" />
            </el-select>
            <p class="user-field-help">权限用于内置仓库，支持网页与 Docker 客户端访问。</p>
          </template>
        </el-form-item>
        <div v-if="mode === 'authorization'" class="user-status-setting">
          <div><label for="user-enabled">账号状态</label><p>{{ form.enabled ? '允许使用此账号登录' : '停用后无法登录网页或访问内置仓库' }}</p></div>
          <el-switch id="user-enabled" v-model="form.enabled" :disabled="protectedAdmin" inline-prompt active-text="启用" inactive-text="停用" />
        </div>
        <p v-if="protectedAdmin" class="user-protection-note"><el-icon><Lock /></el-icon>这是最后一个可用管理员，无法停用或降级。</p>
      </template>
      <div v-if="mode !== 'create'" class="user-change-notice" :class="{ 'user-change-notice--self': isSelf }" role="note">
        <PortainerIcon name="info" />
        <p>保存后，该用户已有的网页登录、仓库登录令牌与自动化令牌将失效。<strong v-if="isSelf">你正在修改当前账号，保存后需重新登录。</strong><strong v-else-if="!form.enabled && mode === 'authorization'">账号停用期间无法重新登录。</strong></p>
      </div>
      <button type="submit" class="user-hidden-submit" tabindex="-1" aria-hidden="true" :disabled="saving || !canSave">提交</button>
    </el-form>
    <template #footer>
      <el-button :disabled="saving" @click="close(false)">取消</el-button>
      <el-button class="user-primary" :loading="saving" :disabled="!canSave" @click="save">{{ mode === 'create' ? '创建用户' : mode === 'password' ? '重设密码' : '保存更改' }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { usersApi } from '@/api/users'
import PortainerIcon from '@/components/PortainerIcon.vue'
import { authorizationChanged, buildUserRequest, emptyUserForm, isLastEnabledAdmin, registryAccessDescriptions, registryAccessLabels, userError, validatePassword, validateUsername, type ManagedUser, type UserEditorMode, type UserForm } from '@/utils/users'

const props = defineProps<{ modelValue: boolean; mode: UserEditorMode; user?: ManagedUser; users: ManagedUser[]; currentUserId?: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [user: ManagedUser, mode: UserEditorMode] }>()
const formRef = ref<FormInstance>()
const form = ref(emptyUserForm())
const saving = ref(false), submitError = ref('')
let disposed = false
const title = computed(() => ({ create: '创建用户', authorization: '修改授权', password: '重设密码' })[props.mode])
const description = computed(() => ({ create: '创建可用于网页登录和 docker login 的账号。', authorization: '管理用户角色、内置仓库权限和账号状态。', password: '设置新密码后，用户需要使用新密码重新登录。' })[props.mode])
const protectedAdmin = computed(() => Boolean(props.user && isLastEnabledAdmin(props.user, props.users)))
const isSelf = computed(() => Boolean(props.user && props.user.id === props.currentUserId))
const canSave = computed(() => props.mode === 'authorization' ? Boolean(props.user && authorizationChanged(form.value, props.user)) : Boolean((props.mode !== 'create' || form.value.username.trim()) && form.value.password && form.value.confirmPassword))
const rules: FormRules<UserForm> = {
  username: [{ validator: (_rule, value: string, callback) => { const error = validateUsername(value, props.users); callback(error ? new Error(error) : undefined) }, trigger: 'blur' }],
  password: [{ validator: (_rule, value: string, callback) => { const error = validatePassword(value); callback(error ? new Error(error) : undefined) }, trigger: ['blur', 'change'] }],
  confirmPassword: [{ validator: (_rule, value: string, callback) => callback(!value ? new Error('请再次输入密码') : value !== form.value.password ? new Error('两次输入的密码不一致') : undefined), trigger: ['blur', 'change'] }],
}
watch(() => props.modelValue, open => { if (open) { form.value = emptyUserForm(props.user); submitError.value = ''; formRef.value?.clearValidate() } })
function reset() { form.value = emptyUserForm(); submitError.value = ''; formRef.value?.clearValidate() }
function close(value: boolean) { if (!saving.value) emit('update:modelValue', value) }
function revalidateConfirmation() { if (form.value.confirmPassword) void formRef.value?.validateField('confirmPassword').catch(() => {}) }
async function save() {
  if (saving.value || !canSave.value || !formRef.value) return
  if (!await formRef.value.validate().catch(() => false) || disposed || saving.value) return
  saving.value = true; submitError.value = ''
  try {
    const payload = buildUserRequest(form.value, props.mode, props.users, props.user)
    const result = props.mode === 'create'
      ? await usersApi.create(payload as Parameters<typeof usersApi.create>[0])
      : await usersApi.update(props.user!.id, payload)
    if (disposed) return
    emit('update:modelValue', false)
    emit('saved', result, props.mode)
  } catch (error) { if (!disposed) submitError.value = userError(error) }
  finally { saving.value = false }
}
onBeforeUnmount(() => { disposed = true; reset() })
</script>
