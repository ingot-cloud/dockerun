<template>
  <div class="workspace-page">
    <WorkspaceHeading title="部署记录" :loading="loading" @refresh="load()"><router-link v-if="route.query.environment_id" to="/deployments">查看全部环境</router-link></WorkspaceHeading>
    <ResourceList title="部署任务" icon="List" :items="rows" :loading="loading" :error="error" :search-keys="['name','application_name','environment_name','release_version','legacy_script_name','id','environment_id','release_id','result']" placeholder="搜索应用、环境、版本或结果…" :sorts="[{key:'created_at',label:'创建时间'},{key:'id',label:'任务编号'}]" :filter-options="statuses" filter-key="status" empty-text="暂无部署记录" @refresh="load()">
      <el-table-column label="任务" min-width="280"><template #default="{row}"><router-link :to="{path:`/deployments/${row.id}`,query:route.query}">{{deploymentName(row)}}</router-link><div class="management-muted deployment-task-meta">任务编号 #{{row.id}} · {{deploymentTarget(row)}}</div></template></el-table-column><el-table-column label="环境" min-width="160" show-overflow-tooltip><template #default="{row}">{{deploymentEnvironment(row)}}</template></el-table-column><el-table-column label="发布版本" min-width="160" show-overflow-tooltip><template #default="{row}">{{deploymentVersion(row)}}</template></el-table-column><el-table-column label="状态" width="145"><template #default="{row}"><StatusBadge :value="row.status" /></template></el-table-column><el-table-column label="当前阶段" width="150"><template #default="{row}">{{stageLabel(row.stage)}}</template></el-table-column><el-table-column label="触发方式" width="180"><template #default="{row}">{{row.automatic?'镜像提交':'手动'}} · {{deploymentUser(row)}}</template></el-table-column><el-table-column prop="result" label="结果" min-width="230" show-overflow-tooltip /><el-table-column label="创建时间" width="180"><template #default="{row}">{{formatDate(row.created_at)}}</template></el-table-column>
      <template #empty><span>从应用中发起发布后，可在这里查看执行记录。</span></template>
    </ResourceList><p class="management-description">任务独立于浏览器运行，执行中的任务会自动更新状态。</p>
  </div>
</template>
<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useResource } from '@/composables/useResource'
import { formatDate,stageLabel,statusLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { deploymentName, deploymentEnvironment, deploymentVersion, deploymentTarget, deploymentUser } from '@/utils/deployments'
const route=useRoute()
const {data:rows,loading,error,load}=useResource<any[]>(()=>'/deployments'+(route.query.environment_id?`?environment_id=${encodeURIComponent(String(route.query.environment_id))}`:''),[],{interval:5000,active:rows=>rows.some(row=>['queued','running','blocked'].includes(row.status))})
const statuses=['queued','running','success','failed','needs_attention','cancelled','blocked','paused'].map(value=>({value,label:statusLabel(value)}))
</script>
<style scoped>
.deployment-task-meta { font-size: 11px; overflow-wrap: anywhere; }
</style>
