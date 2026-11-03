<template>
  <div class="compose-preview">
    <div class="compose-preview-toolbar"><div class="compose-preview-title"><strong>{{title}}</strong><span v-if="parsed.services.length" class="management-muted">{{parsed.services.length}} 个服务</span><DeploymentHelp v-if="description" :label="`查看${title}说明`" :title="title"><p>{{description}}</p><p>镜像摘要和 Image ID 用于核对版本身份；展示服务概览不会修改部署配置。</p></DeploymentHelp></div><el-radio-group v-model="view" aria-label="部署内容展示方式"><el-radio-button value="summary">服务概览</el-radio-button><el-radio-button value="yaml">Compose 详情</el-radio-button></el-radio-group></div>
    <p v-if="!content" class="management-description">{{emptyText}}</p>
    <div v-else-if="view==='summary'" class="compose-preview-services" tabindex="0" aria-label="部署服务概览滚动区域">
      <div v-if="parsed.error" class="management-error" role="alert">{{parsed.error}}<el-button link @click="view='yaml'">查看原始配置</el-button></div>
      <section v-for="service in parsed.services" :key="service.name" class="compose-preview-service">
        <h3>{{service.name}}</h3><dl>
          <dt>容器名称</dt><dd>{{service.containerName}}</dd>
          <dt>镜像</dt><dd>{{service.image}}</dd>
          <template v-if="service.digest"><dt>锁定摘要</dt><dd class="compose-preview-identity">{{service.digest}}</dd></template>
          <template v-if="service.imageID"><dt>Image ID</dt><dd class="compose-preview-identity">{{service.imageID}}</dd></template>
          <dt>端口映射</dt><dd><div v-for="port in service.ports" :key="port">{{port}}</div><span v-if="!service.ports.length" class="management-muted">未发布端口</span></dd>
          <dt>网络</dt><dd><div v-for="network in service.networks" :key="network">{{network}}</div></dd>
          <dt>数据挂载</dt><dd><div v-for="mount in service.mounts" :key="mount">{{mount}}</div><span v-if="!service.mounts.length" class="management-muted">未配置</span></dd>
          <template v-if="service.configs.length"><dt>配置文件</dt><dd><div v-for="file in service.configs" :key="file">{{file}}</div></dd></template>
          <dt>重启策略</dt><dd>{{service.restart}}</dd><dt>健康检查</dt><dd>{{service.healthcheck}}</dd>
          <dt>容器变量</dt><dd>{{service.environmentCount}} 项</dd>
        </dl>
      </section>
    </div>
    <CodeEditor v-else :model-value="parsed.yaml" language="yaml" :document-key="title" :label="`${title} · Compose`" fill readonly />
  </div>
</template>
<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import DeploymentHelp from '@/components/DeploymentHelp.vue'
import { deploymentComposePreview, type DeploymentPreviewImage } from '@/utils/deploymentPreview'
const CodeEditor=defineAsyncComponent(()=>import('@/components/CodeEditor.vue'))
const props=withDefaults(defineProps<{content:string;images?:DeploymentPreviewImage[]|null;title?:string;description?:string;emptyText?:string}>(),{title:'部署内容',emptyText:'暂无部署配置'})
const view=ref('summary')
const parsed=computed(()=>{if(!props.content)return {yaml:'',services:[],error:''};try{return {...deploymentComposePreview(props.content,props.images),error:''}}catch(cause){return {yaml:props.content,services:[],error:cause instanceof Error?cause.message:'无法生成服务概览'}}})
</script>
<style scoped>
.compose-preview{display:flex;flex-direction:column;gap:12px;flex:1;min-height:0;min-width:0;overflow:hidden}.compose-preview-toolbar,.compose-preview-title{display:flex;align-items:center;gap:8px;flex-wrap:wrap;min-width:0}.compose-preview-toolbar{justify-content:space-between;flex-shrink:0}.compose-preview-title{font-size:13px}.compose-preview-title .management-muted{font-size:12px}.compose-preview-services{flex:1;min-height:0;overflow:auto;scrollbar-width:thin}.compose-preview-service{border:1px solid #383838;border-radius:8px;background:#1b1b1b;padding:14px;margin-bottom:12px}.compose-preview-service:last-child{margin-bottom:0}.compose-preview-service h3{font-size:15px;font-weight:600;margin:0 0 12px;overflow-wrap:anywhere}.compose-preview-service dl{display:grid;grid-template-columns:110px minmax(0,1fr);gap:8px 16px;font-size:13px;margin:0}.compose-preview-service dt{color:#999}.compose-preview-service dd{margin:0;min-width:0;overflow-wrap:anywhere;white-space:pre-wrap}.compose-preview-identity{color:#aaa;font-size:12px}.compose-preview>.code-editor-wrapper{flex:1;min-height:0}.compose-preview .management-error{margin-bottom:12px}
@media(max-width:760px){.compose-preview-service{padding:12px}.compose-preview-service dl{grid-template-columns:84px minmax(0,1fr);gap:8px 12px}.compose-preview-toolbar{align-items:flex-start}}
</style>
