<template>
  <div class="workspace-page">
    <WorkspaceHeading title="系统信息" :loading="loading" @refresh="load()" />
    <div v-if="error" class="management-error" role="alert"><span>系统信息加载失败：{{error}}</span><el-button link :disabled="loading" @click="load()">重试</el-button></div>
    <div v-loading="loading" class="management-grid">
      <section v-for="section in sections" :key="section.title" class="management-card"><header class="management-toolbar"><h2><span class="management-section-icon"><el-icon><component :is="section.icon" /></el-icon></span>{{section.title}}</h2></header><dl class="management-details"><div v-for="item in section.items" :key="item.label"><dt>{{item.label}}</dt><dd>{{value(item.keys)}}</dd></div></dl></section>
    </div>
    <div v-if="!loaded && !loading && error" class="management-empty"><p>暂时无法加载系统信息</p><span>请确认 Docker 服务正常运行后重试。</span></div>
  </div>
</template>
<script setup lang="ts">
import { useResource } from '@/composables/useResource'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
const {data,loading,error,loaded,load}=useResource<Record<string,unknown>>('/system/info',{})
const sections=[{title:'Docker 环境',icon:'Box',items:[{label:'客户端版本',keys:['docker_version']},{label:'服务端版本',keys:['server_version']},{label:'接口版本',keys:['docker_api_version']},{label:'应用栈工具版本',keys:['compose_version']}]},{title:'服务器信息',icon:'Monitor',items:[{label:'操作系统',keys:['operating_system']},{label:'内核版本',keys:['kernel_version']},{label:'处理器架构',keys:['server_arch','arch']},{label:'宿主系统',keys:['server_os','os']}]},{title:'资源容量',icon:'DataAnalysis',items:[{label:'总内存',keys:['total_memory']},{label:'处理器数量',keys:['cpus']}]}]
function value(keys:string[]){const result=keys.map(key=>data.value[key]).find(item=>item!==null && item!==undefined && item!=='');if(result==='Not Installed')return '未安装';if(result==='N/A')return '暂不可用';return result ?? '—'}
</script>
