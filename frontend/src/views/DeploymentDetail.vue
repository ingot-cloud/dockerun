<template>
  <div class="workspace-page">
    <WorkspaceHeading :title="`部署 #${route.params.id}`" :loading="loading" @refresh="load()"><router-link :to="{path:'/deployments',query:route.query}">返回部署记录</router-link><StatusBadge v-if="loaded" :value="task.status" /></WorkspaceHeading>
    <div v-if="error" class="management-error" role="alert"><span>部署详情加载失败：{{error}}</span><el-button link @click="load()">重试</el-button></div>
    <div v-loading="loading" class="stats-grid"><section class="stat-card"><span class="management-muted">当前阶段</span><strong style="font-size:18px">{{stageLabel(task.stage)}}</strong></section><section class="stat-card"><span class="management-muted">发布版本</span><strong style="font-size:18px">{{task.release_id?`#${task.release_id}`:'—'}}</strong></section><section class="stat-card"><span class="management-muted">恢复结果</span><div style="margin-top:12px"><StatusBadge :value="task.recovery_status || 'unknown'" /></div></section><section class="stat-card"><span class="management-muted">触发身份</span><strong style="font-size:18px">{{task.user_id?`用户 #${task.user_id}`:'—'}}</strong></section></div>
    <section class="management-card"><header class="management-toolbar"><h2>执行结果</h2></header><div class="management-card-body" style="padding-top:0"><p>{{task.result || (loaded?'任务正在等待或执行':'等待加载任务结果')}}</p><pre v-if="task.actual_images" class="management-code" style="margin-top:16px">{{task.actual_images}}</pre></div></section>
    <section v-if="nodes.length>1" class="management-card"><header class="management-toolbar"><h2>逐机发布结果</h2></header><el-table :data="nodes" empty-text="暂无主机结果"><el-table-column prop="target_id" label="主机 ID" width="110" /><el-table-column label="任务" width="120"><template #default="{row}"><router-link :to="{path:`/deployments/${row.id}`,query:route.query}">#{{row.id}}</router-link></template></el-table-column><el-table-column label="状态" width="150"><template #default="{row}"><StatusBadge :value="row.status" /></template></el-table-column><el-table-column prop="result" label="执行结果" min-width="220" show-overflow-tooltip /></el-table></section>
    <section class="management-card"><header class="management-toolbar"><h2>执行日志</h2><span class="management-description">{{active?'任务执行中，日志自动更新':'持久化保存，可随时回看'}}</span></header><div class="management-card-body" style="padding-top:0"><div class="management-code task-logs" tabindex="0" aria-label="部署执行日志" role="log"><div v-for="log in logs" :key="log.id" class="task-line"><small>{{formatDate(log.created_at)}} · {{stageLabel(log.stage)}}</small><span>{{log.message}}</span></div><p v-if="!logs.length" class="management-muted">{{loaded?'暂无日志输出':'正在加载日志…'}}</p></div></div></section>
  </div>
</template>
<script setup lang="ts">
import { computed,onBeforeUnmount,onMounted,ref } from 'vue'
import { useRoute } from 'vue-router'
import { api,errorText } from '@/api/workspace'
import { formatDate,stageLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import StatusBadge from '@/components/StatusBadge.vue'
const route=useRoute(),task=ref<any>({}),logs=ref<any[]>([]),nodes=ref<any[]>([]),loading=ref(false),loaded=ref(false),error=ref('')
const active=computed(()=>[task.value,...nodes.value].some(item=>['queued','running','blocked'].includes(item.status)))
const controller=new AbortController();let disposed=false,fetching=false,timer:ReturnType<typeof setTimeout>|undefined
async function load(){if(disposed || fetching)return;clearTimeout(timer);fetching=true;if(!loaded.value)loading.value=true;error.value='';try{const result=await api('GET',`/deployments/${route.params.id}?after=${logs.value[logs.value.length - 1]?.id || 0}`,undefined,controller.signal);if(disposed)return;task.value=result.deployment;nodes.value=result.nodes || [];const seen=new Set(logs.value.map(log=>log.id));logs.value.push(...(result.logs || []).filter((log:any)=>!seen.has(log.id)));loaded.value=true}catch(cause){if(!disposed)error.value=errorText(cause)}finally{fetching=false;if(!disposed){loading.value=false;if(active.value || error.value)timer=setTimeout(load,error.value?5000:1500)}}}
onMounted(load);onBeforeUnmount(()=>{disposed=true;clearTimeout(timer);controller.abort()})
</script>
