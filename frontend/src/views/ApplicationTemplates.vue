<template>
  <div class="workspace-page">
    <WorkspaceHeading title="应用模板" :loading="loading" @refresh="load()" />
    <p class="management-description">选择模板，填写镜像和环境参数后创建应用并发布。中间件默认手动更新，模板更新不会修改已有应用。</p>
    <ResourceList title="应用模板列表" icon="Collection" :items="items" :loading="loading" :error="error" :search-keys="['name','description','version']" :sorts="[{key:'name',label:'名称'},{key:'version',label:'版本'}]" :filter-options="[{value:'business',label:'业务应用'},{value:'middleware',label:'中间件'}]" filter-key="category" filter-label="全部类型" empty-text="暂无应用模板" @refresh="load()">
      <template #default="{query}"><el-table-column label="模板名称" min-width="220"><template #default="{row}"><router-link :to="{path:`/application-templates/${row.id}/create`,query}">{{row.name}}</router-link><div class="management-muted" style="font-size:11px">{{row.source==='custom'?'自定义模板':'内置模板'}} · {{row.version}}</div></template></el-table-column><el-table-column label="类型" width="120"><template #default="{row}">{{templateCategory(row.category)}}</template></el-table-column><el-table-column prop="description" label="用途" min-width="300" show-overflow-tooltip /><el-table-column label="部署方式" width="160"><template #default="{row}">{{row.definition.topology==='single_host'?'单台目标主机':'单机或固定主机组'}}</template></el-table-column><el-table-column label="操作" width="140"><template #default="{row}"><router-link :to="{path:`/application-templates/${row.id}/create`,query}">创建应用</router-link></template></el-table-column></template>
    </ResourceList>
  </div>
</template>
<script setup lang="ts">
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import { useResource } from '@/composables/useResource'
import { templateCategory } from '@/utils/applicationTemplates'
import type { ApplicationTemplate } from '@/api/applicationTemplates'
const {data:items,loading,error,load}=useResource<ApplicationTemplate[]>('/managed-templates',[])
</script>
