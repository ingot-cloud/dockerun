<template>
  <div class="workspace-page">
    <WorkspaceHeading title="个人账号" :loading="loading" :busy="busy" @refresh="load()"><span class="management-description">{{auth.user?.username}} · {{auth.user?.is_admin?'管理员':'普通用户'}}</span></WorkspaceHeading>
    <div class="management-grid">
      <section class="management-card"><header class="management-toolbar"><h2>修改密码</h2></header><div class="management-card-body"><p class="management-description">修改后，已有登录及访问令牌失效，需要重新登录。</p><div v-if="passwordError" class="management-error management-feedback" role="alert">{{passwordError}}</div><el-form ref="passwordForm" :model="passwords" :rules="passwordRules" label-position="top" class="management-form" :disabled="busy" @submit.prevent="changePassword"><el-form-item label="原密码" prop="old"><el-input v-model="passwords.old" type="password" show-password autocomplete="current-password" /></el-form-item><el-form-item label="新密码" prop="next"><el-input v-model="passwords.next" type="password" show-password autocomplete="new-password" placeholder="输入 6–72 字节密码" /></el-form-item><el-form-item label="确认新密码" prop="confirm"><el-input v-model="passwords.confirm" type="password" show-password autocomplete="new-password" /></el-form-item><el-button native-type="submit" type="primary" :loading="passwordSaving">更新并重新登录</el-button></el-form></div></section>
      <section class="management-card"><header class="management-toolbar"><h2>创建自动化令牌</h2></header><div class="management-card-body"><p class="management-description">权限不能超过当前账号。令牌有效期 30 天，仅在创建时展示一次。</p><div v-if="tokenError" class="management-error management-feedback" role="alert">{{tokenError}}</div><el-form ref="tokenForm" :model="token" :rules="tokenRules" label-position="top" class="management-form" :disabled="busy" @submit.prevent="createToken"><el-form-item label="令牌用途" prop="name"><el-input v-model="token.name" placeholder="例如：构建服务拉取镜像" maxlength="100" /></el-form-item><el-form-item label="授权范围" prop="scopes"><el-checkbox-group v-model="token.scopes"><el-checkbox value="pull" :disabled="!canPull">拉取镜像</el-checkbox><el-checkbox value="push" :disabled="!canPush">推送镜像</el-checkbox><el-checkbox v-if="auth.user?.is_admin" value="deploy">发布应用</el-checkbox></el-checkbox-group></el-form-item><p v-if="!canPull" class="management-form-help">当前账号没有仓库权限，请联系管理员授权。</p><el-button type="primary" native-type="submit" :loading="tokenSaving" :disabled="!token.scopes.length">创建 30 天令牌</el-button></el-form><div v-if="secret" class="management-secret"><p>令牌仅展示一次，请立即保存。关闭后无法再次查看。</p><pre class="management-code">{{secret}}</pre><el-button @click="copySecret">复制令牌</el-button><el-button @click="secret=''">已保存，隐藏</el-button></div></div></section>
    </div>
    <ResourceList title="已签发令牌" icon="Key" :items="tokens" :loading="loading" :error="error" :busy="busy" :search-keys="['name','id']" :sorts="[{key:'created_at',label:'签发时间'},{key:'expires_at',label:'到期时间'},{key:'name',label:'名称'}]" empty-text="暂无自动化令牌" style="flex:none;height:460px" @refresh="load()"><el-table-column prop="name" label="用途" min-width="170" show-overflow-tooltip /><el-table-column label="权限" min-width="180"><template #default="{row}">{{scopeText(row.scopes)}}</template></el-table-column><el-table-column label="到期时间" width="180"><template #default="{row}">{{formatDate(row.expires_at)}}</template></el-table-column><el-table-column label="状态" width="130"><template #default="{row}"><StatusBadge v-if="tokenState(row)!=='ready'" :value="tokenState(row)" /><span v-else class="state state-success">有效期内</span></template></el-table-column><el-table-column label="操作" width="110"><template #default="{row}"><el-button link type="danger" :disabled="row.revoked || busy" @click="revoke(row)">撤销令牌</el-button></template></el-table-column></ResourceList>
    <p class="management-description">列表状态反映撤销标记和有效期；账号授权变更、密码修改或停用也会使已有令牌失效。</p>
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage,ElMessageBox,type FormInstance,type FormRules } from 'element-plus'
import { useClipboard } from '@vueuse/core'
import { api,errorText } from '@/api/workspace'
import { authApi } from '@/api/auth'
import { encryptPassword } from '@/utils/crypto'
import { validatePassword } from '@/utils/users'
import { formatDate,tokenState } from '@/utils/management'
import { useAuthStore } from '@/stores/auth'
import { useResource } from '@/composables/useResource'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import StatusBadge from '@/components/StatusBadge.vue'
const auth=useAuthStore(),router=useRouter(),{copy}=useClipboard({legacy:true})
const {data:tokens,loading,error,load}=useResource<any[]>('/auth/tokens',[])
const passwordForm=ref<FormInstance>(),tokenForm=ref<FormInstance>(),passwordSaving=ref(false),tokenSaving=ref(false),revoking=ref(false),passwordError=ref(''),tokenError=ref(''),secret=ref('')
const canPull=computed(()=>Boolean(auth.user?.is_admin || ['read_only','read_write'].includes(auth.user?.registry_access))),canPush=computed(()=>Boolean(auth.user?.is_admin || auth.user?.registry_access==='read_write'))
const passwords=ref({old:'',next:'',confirm:''}),token=ref({name:'',scopes:canPull.value?['pull']:[] as string[]})
const busy=computed(()=>passwordSaving.value || tokenSaving.value || revoking.value)
const passwordRules:FormRules={old:[{required:true,message:'请输入原密码',trigger:'blur'}],next:[{validator:(_r,value,done)=>{const message=validatePassword(value);done(message?new Error(message):undefined)},trigger:['blur','change']}],confirm:[{validator:(_r,value,done)=>done(value===passwords.value.next && value?undefined:new Error('两次输入的新密码不一致')),trigger:['blur','change']}]}
const tokenRules={name:[{required:true,whitespace:true,message:'请输入令牌用途',trigger:'blur'}],scopes:[{type:'array' as const,required:true,min:1,message:'请选择至少一项权限',trigger:'change'}]}
function scopeText(value:string){return (value || '').split(' ').map(scope=>({pull:'拉取镜像',push:'推送镜像',deploy:'发布应用'}[scope] || '未知权限')).join('、') || '无权限'}
async function changePassword(){if(busy.value || !await passwordForm.value?.validate().catch(()=>false) || busy.value)return;passwordSaving.value=true;passwordError.value='';try{await authApi.changePassword({old_password:encryptPassword(passwords.value.old),new_password:encryptPassword(passwords.value.next)});auth.logout();ElMessage.success('密码已更新，请重新登录');await router.replace('/login')}catch(cause){passwordError.value=errorText(cause)}finally{passwordSaving.value=false}}
async function createToken(){if(busy.value || !await tokenForm.value?.validate().catch(()=>false) || busy.value)return;tokenSaving.value=true;tokenError.value='';try{const result=await api('POST','/auth/tokens',{name:token.value.name.trim(),scopes:token.value.scopes,days:30});secret.value=result.token;token.value.name='';await load();ElMessage.success('自动化令牌已创建')}catch(cause){tokenError.value=errorText(cause)}finally{tokenSaving.value=false}}
async function copySecret(){try{await copy(secret.value);ElMessage.success('令牌已复制')}catch{ElMessage.error('复制失败，请重试')}}
async function revoke(row:any){if(busy.value)return;revoking.value=true;try{await ElMessageBox.confirm(`确定撤销「${row.name}」？使用此令牌的自动化任务将无法继续认证。`,'撤销自动化令牌',{confirmButtonText:'确认撤销',cancelButtonText:'取消',type:'warning'})}catch{revoking.value=false;return}try{await api('DELETE',`/auth/tokens/${row.id}`);ElMessage.success('令牌已撤销');await load()}catch(cause){error.value=errorText(cause)}finally{revoking.value=false}}
</script>
