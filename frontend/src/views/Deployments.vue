<template>
  <div class="workspace-page">
    <WorkspaceHeading title="部署记录" :loading="loading" :busy="deleteVisible" @refresh="load()"><router-link v-if="route.query.environment_id" to="/deployments">查看全部环境</router-link></WorkspaceHeading>
    <ResourceList title="部署任务" icon="List" :items="rows" :loading="loading" :error="error" :busy="deleteVisible" selection :selectable="row=>!deploymentDeletionReason(row)" :search-keys="['name','application_name','environment_name','release_version','legacy_script_name','id','environment_id','release_id','result']" placeholder="搜索应用、环境、版本或结果…" :sorts="[{key:'created_at',label:'创建时间'},{key:'id',label:'任务编号'}]" :filter-options="statuses" filter-key="status" empty-text="暂无部署记录" @refresh="load()" @selection-change="selected=$event">
      <template #actions><el-button type="danger" :disabled="!selected.length || deleteVisible || loading" :title="selected.length ? '删除选中的部署记录' : '请先勾选已结束的部署记录'" @click="openDeleteTargets(selected)">批量删除</el-button></template>
      <el-table-column label="任务" min-width="280"><template #default="{row}"><router-link :to="{path:`/deployments/${row.id}`,query:route.query}">{{deploymentName(row)}}</router-link><div class="management-muted deployment-task-meta">任务编号 #{{row.id}} · {{deploymentTarget(row)}}</div></template></el-table-column><el-table-column label="环境" min-width="160" show-overflow-tooltip><template #default="{row}">{{deploymentEnvironment(row)}}</template></el-table-column><el-table-column label="发布版本" min-width="160" show-overflow-tooltip><template #default="{row}">{{deploymentVersion(row)}}</template></el-table-column><el-table-column label="状态" width="145"><template #default="{row}"><StatusBadge :value="row.status" /></template></el-table-column><el-table-column label="当前阶段" width="150"><template #default="{row}">{{stageLabel(row.stage)}}</template></el-table-column><el-table-column label="触发方式" width="180"><template #default="{row}">{{row.automatic?'镜像提交':'手动'}} · {{deploymentUser(row)}}</template></el-table-column><el-table-column prop="result" label="结果" min-width="230" show-overflow-tooltip /><el-table-column label="创建时间" width="180"><template #default="{row}">{{formatDate(row.created_at)}}</template></el-table-column>
      <el-table-column label="操作" width="90" fixed="right"><template #default="{row}"><el-tooltip :content="deploymentDeletionReason(row)" :disabled="!deploymentDeletionReason(row)"><span><el-button link type="danger" :disabled="deleteVisible || loading || !!deploymentDeletionReason(row)" :aria-label="`删除部署记录 #${row.id} ${deploymentName(row)}`" @click="openDeleteTargets([row])">删除</el-button></span></el-tooltip></template></el-table-column>
      <template #empty><span>从应用中发起发布后，可在这里查看执行记录。</span></template>
    </ResourceList><p class="management-description">任务状态自动更新；排队、执行或恢复中的任务不能删除。</p>
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="部署记录" endpoint="/deployments" warning="将删除所选部署记录及其日志，无法撤销。已部署资源会保留，运行检查和回滚所需的内部配置快照会保留。排队、执行或恢复中的任务不能删除。" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed,ref } from 'vue'
import { useRoute } from 'vue-router'
import { useResource } from '@/composables/useResource'
import { formatDate,stageLabel,statusLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import { workspaceDeleteTargets } from '@/utils/workspaceDeletion'
import { deploymentName, deploymentEnvironment, deploymentVersion, deploymentTarget, deploymentUser, deploymentDeletionReason } from '@/utils/deployments'
const route=useRoute()
const {data:deploymentRows,loading,error,load}=useResource<any[]>(()=>'/deployments'+(route.query.environment_id?`?environment_id=${encodeURIComponent(String(route.query.environment_id))}`:''),[],{interval:5000,active:rows=>rows.some(row=>['queued','running','blocked'].includes(row.status) || row.recovery_status==='running')})
const deleteVisible=ref(false),deleteTargets=ref<{id:number;name:string}[]>([]),removedIds=ref(new Set<number>()),selected=ref<any[]>([])
const rows=computed(()=>deploymentRows.value.filter(row=>!removedIds.value.has(row.id)))
function openDeleteTargets(items:any[]){if(deleteVisible.value || loading.value)return;const targets=workspaceDeleteTargets(items.filter(row=>!deploymentDeletionReason(row)),row=>row.id,row=>deploymentName(row));if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function onRemoved(ids:number[]){removedIds.value=new Set([...removedIds.value,...ids])}
const statuses=['queued','running','success','failed','needs_attention','cancelled','blocked','paused'].map(value=>({value,label:statusLabel(value)}))
</script>
<style scoped>
.deployment-task-meta { font-size: 11px; overflow-wrap: anywhere; }
</style>
