<template>
  <div class="workspace-page">
    <WorkspaceHeading title="部署概览" :loading="loading" @refresh="load()"><el-button type="primary" @click="$router.push('/applications')">发布应用</el-button></WorkspaceHeading>
    <div v-if="error" class="management-error" role="alert"><span>概览加载失败：{{error}}</span><el-button link @click="load()">重试</el-button></div>
    <div v-loading="loading" class="stats-grid"><router-link v-for="card in cards" :key="card.key" class="stat-card" :to="card.to" style="color:inherit;text-decoration:none"><span class="management-muted">{{card.title}}</span><strong>{{data.counts?.[card.key] ?? '—'}}</strong></router-link></div>
    <section class="management-card"><header class="management-toolbar"><h2><span class="management-section-icon"><el-icon><Grid /></el-icon></span>应用环境</h2></header><div v-loading="loading" class="management-card-body" style="padding-top:0"><el-table :data="data.environments || []" empty-text="暂无应用环境，先创建应用并配置环境"><el-table-column label="应用" width="120"><template #default="{row}"><router-link :to="`/applications/${row.application_id}`">应用 #{{row.application_id}}</router-link></template></el-table-column><el-table-column prop="name" label="部署环境" min-width="150" /><el-table-column label="当前版本" width="120"><template #default="{row}">{{row.current_release_id?`#${row.current_release_id}`:'尚未发布'}}</template></el-table-column><el-table-column label="目标主机" width="130"><template #default="{row}">{{row.target_id>1?`主机 #${row.target_id}`:'本机'}}</template></el-table-column><el-table-column label="健康状态" min-width="160"><template #default="{row}"><StatusBadge :value="row.health" /></template></el-table-column></el-table></div></section>
    <section class="management-card"><header class="management-toolbar"><h2>从镜像到运行</h2></header><p class="management-card-body management-description" style="padding-top:0">选择仓库镜像、本机镜像或完整发布包，确认环境参数和差异后执行。生产环境默认手动发布，镜像导入与应用部署分别进行。</p></section>
  </div>
</template>
<script setup lang="ts">
import { useResource } from '@/composables/useResource'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import StatusBadge from '@/components/StatusBadge.vue'
const {data,loading,error,load}=useResource<any>('/overview',{})
const cards=[{key:'applications',title:'应用',to:'/applications'},{key:'deployments',title:'部署记录',to:'/deployments'},{key:'bundles',title:'离线发布包',to:'/bundles'},{key:'image_events',title:'镜像提交',to:'/builtin'}]
</script>
