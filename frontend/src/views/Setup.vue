<template>
  <main class="auth-page"><section class="auth-card auth-card--setup"><header class="auth-header"><div class="auth-brand"><span>D</span>Dockerun</div><h1>初始化工作台</h1><p>为管理员账号 admin 设置密码，并确认系统运行参数。</p></header><div v-if="error" class="management-error" role="alert">{{error}}</div><el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="loading" @submit.prevent="handleSetup"><section class="auth-section"><h2>管理员密码</h2><el-form-item label="初始密码" prop="password"><el-input v-model="form.password" type="password" autocomplete="new-password" show-password placeholder="输入 6–72 字节密码" /><p class="auth-help">中文通常占 3 字节。</p></el-form-item><el-form-item label="确认密码" prop="confirmPassword"><el-input v-model="form.confirmPassword" type="password" autocomplete="new-password" show-password placeholder="再次输入密码" /></el-form-item></section><section class="auth-section"><h2>系统运行参数</h2><el-form-item label="部署工作目录" prop="deployWorkDir"><el-input v-model="form.deployWorkDir" placeholder="例如：/ingot-data/dockerun" /><p class="auth-help">用于保存部署脚本及相关文件。</p></el-form-item><el-form-item label="脚本最大并发数" prop="maxScriptExecutors"><el-input-number v-model="form.maxScriptExecutors" :min="1" :max="20" aria-label="脚本最大并发数" /></el-form-item><el-form-item label="脚本超时时间（秒）" prop="scriptTimeout"><el-input-number v-model="form.scriptTimeout" :min="60" :max="7200" :step="60" aria-label="脚本超时时间" /></el-form-item></section><el-button type="primary" native-type="submit" :loading="loading">{{loading?'正在初始化…':'完成初始化'}}</el-button></el-form></section></main>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage,type FormInstance,type FormRules } from 'element-plus'
import { authApi } from '@/api/auth'
import { errorText } from '@/api/workspace'
import { encryptPassword } from '@/utils/crypto'
import { validatePassword } from '@/utils/users'
import '@/styles/auth.css'
const router=useRouter(),formRef=ref<FormInstance>(),loading=ref(false),error=ref('')
const form=ref({password:'',confirmPassword:'',deployWorkDir:'/ingot-data/dockerun',maxScriptExecutors:5,scriptTimeout:3600})
const rules:FormRules={password:[{validator:(_r,value,done)=>{const message=validatePassword(value);done(message?new Error(message):undefined)},trigger:['blur','change']}],confirmPassword:[{validator:(_r,value,done)=>done(value && value===form.value.password?undefined:new Error('两次输入的密码不一致')),trigger:['blur','change']}],deployWorkDir:[{required:true,whitespace:true,message:'请输入部署工作目录',trigger:'blur'}],maxScriptExecutors:[{required:true,message:'请设置脚本最大并发数',trigger:'change'}],scriptTimeout:[{required:true,message:'请设置脚本超时时间',trigger:'change'}]}
async function handleSetup(){if(loading.value || !await formRef.value?.validate().catch(()=>false) || loading.value)return;loading.value=true;error.value='';try{await authApi.setInitialPassword({new_password:encryptPassword(form.value.password),deploy_work_dir:form.value.deployWorkDir.trim(),max_script_executors:form.value.maxScriptExecutors,script_timeout:form.value.scriptTimeout});ElMessage.success('初始化完成，请使用管理员账号登录');await router.replace('/login')}catch(cause){error.value=errorText(cause)}finally{loading.value=false}}
</script>
