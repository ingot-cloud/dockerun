<template>
  <div class="workspace-page bundle-page">
    <WorkspaceHeading title="发布包" :loading="loading" :busy="busy" @refresh="load()" />
    <section class="management-card bundle-import" aria-label="导入交付材料">
      <div class="bundle-import-toolbar">
        <el-select v-model="kind" class="bundle-kind" :disabled="busy" aria-label="交付材料类型"><el-option label="完整发布包" value="bundle" /><el-option label="普通镜像归档" value="image" /></el-select>
        <div class="bundle-source-options">
          <el-radio-group v-model="importSource" size="small" :disabled="busy" aria-label="导入来源"><el-radio-button value="upload">上传文件</el-radio-button><el-radio-button value="server">服务器文件</el-radio-button></el-radio-group>
        </div>
      </div>
      <form class="bundle-import-form" :class="{ 'bundle-server-form': importSource === 'server' }" @submit.prevent="importSource === 'upload' ? upload() : importDirectory()">
        <template v-if="importSource === 'upload'">
          <label class="bundle-file" :class="{ disabled: busy }">选择文件<input type="file" accept=".tar,.tar.gz,.tgz" :aria-label="kind==='bundle'?'选择完整发布包':'选择镜像归档'" :disabled="busy" @change="select" /></label>
          <span class="bundle-file-name" :title="file?.name">{{ file?.name || '未选择文件' }}</span>
        </template>
        <el-input v-else v-model="serverFile" placeholder="服务器导入目录中的文件名，例如 application-release.tar.gz" :disabled="busy" aria-label="服务器导入目录中的文件" />
        <el-tooltip content="支持 .tar、.tar.gz、.tgz；上传文件按 SHA-256 校验，重新选择同一文件可从服务器记录的偏移继续上传。服务器文件从服务器导入目录读取。" placement="bottom"><el-button native-type="button" class="bundle-import-help" text circle aria-label="查看导入格式与断点续传说明"><PortainerIcon name="info" /></el-button></el-tooltip>
        <el-button native-type="submit" type="primary" :disabled="busy || (importSource === 'upload' ? !file : !serverFile.trim())" :loading="uploading || importing">{{ importSource === 'upload' ? '校验并导入' : '导入服务器文件' }}</el-button>
      </form>
      <div v-if="progress !== null || stage" class="bundle-progress"><el-progress v-if="progress !== null" :percentage="progress" /><p v-if="stage" role="status">{{ stage }}</p></div>
      <div v-if="operationError" class="management-error bundle-operation-error" role="alert">{{ operationError }}</div>
    </section>
    <ResourceList class="bundle-records" title="导入记录" icon="Files" :items="rows" :loading="loading" :error="error" :busy="busy" selection :selectable="row=>!bundleDeletionReason(row)" placeholder="搜索名称、编号、摘要或错误…" :search-keys="['name','sha256','id','error']" :sorts="[{key:'created_at',label:'导入时间'},{key:'name',label:'名称'}]" :filter-options="statuses" filter-key="status" empty-text="暂无导入记录" @refresh="load()" @selection-change="selected=$event"><template #actions><el-button type="danger" :disabled="!selected.length || busy || loading" :title="selected.length ? '删除选中的发布包' : '请先勾选要删除的发布包'" @click="openDeleteTargets(selected)">批量删除</el-button></template><el-table-column prop="id" label="编号" width="80" /><el-table-column prop="name" label="交付材料" min-width="190" show-overflow-tooltip /><el-table-column label="状态" width="150"><template #default="{row}"><StatusBadge :value="row.status" /></template></el-table-column><el-table-column label="本机镜像" min-width="190"><template #default="{row}"><span>{{row.local_loaded?'已加载':'由目标节点确认'}}</span><p v-if="row.local_load_note" class="management-description">{{row.local_load_note}}</p></template></el-table-column><el-table-column prop="sha256" label="校验摘要（SHA-256）" min-width="190" show-overflow-tooltip /><el-table-column prop="error" label="错误信息" min-width="180" show-overflow-tooltip /><el-table-column label="操作" min-width="310"><template #default="{row}"><div class="management-row-actions"><el-button link type="primary" :disabled="busy || row.status!=='ready' || row.kind==='image'" @click="createVersion(row)">创建待发布版本</el-button><el-button v-if="row.status==='failed'" link :disabled="busy" @click="retry(row)">重试导入</el-button><el-button link :disabled="busy" @click="downloadBundle(row)">下载</el-button><el-tooltip :content="bundleDeletionReason(row)" :disabled="!bundleDeletionReason(row)"><span><el-button link type="danger" :disabled="busy || !!bundleDeletionReason(row)" :aria-label="`删除发布包 ${row.name}`" @click="openDelete(row)">删除</el-button></span></el-tooltip></div></template></el-table-column></ResourceList>
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="发布包" endpoint="/bundles" warning="将删除导入记录、已上传的归档文件和解包材料，无法撤销。已加载到 Docker 的镜像及服务器导入目录中的原始文件会保留；正在导入或被发布版本引用的材料不能删除。" @removed="onRemoved" />
  </div>
</template>
<script setup lang="ts">
import { computed,ref,watch } from 'vue'
import { onBeforeRouteLeave,useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import CryptoJS from 'crypto-js'
import { api,download,errorText } from '@/api/workspace'
import request from '@/api/request'
import { useResource } from '@/composables/useResource'
import { statusLabel } from '@/utils/management'
import WorkspaceHeading from '@/components/WorkspaceHeading.vue'
import ResourceList from '@/components/ResourceList.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import PortainerIcon from '@/components/PortainerIcon.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import { bundleDeletionReason,workspaceDeleteTargets } from '@/utils/workspaceDeletion'
const router=useRouter(),{data:bundleRows,loading,error,load}=useResource<any[]>('/bundles',[],{interval:3000,active:rows=>rows.some(row=>['queued','validating','loading'].includes(row.status))})
const deleteVisible=ref(false),deleteTargets=ref<{id:number;name:string}[]>([]),removedIds=ref(new Set<number>()),selected=ref<{id:number;name:string;status?:string}[]>([])
const rows=computed(()=>bundleRows.value.filter(row=>!removedIds.value.has(row.id)))
const file=ref<File>(),uploading=ref(false),importing=ref(false),rowBusy=ref(false),progress=ref<number|null>(null),stage=ref(''),operationError=ref(''),serverFile=ref(''),kind=ref('bundle'),importSource=ref('upload')
const busy=computed(()=>uploading.value || importing.value || rowBusy.value || deleteVisible.value),statuses=['ready','validating','loading','failed'].map(value=>({value,label:statusLabel(value)}))
function openDeleteTargets(items:{id:number;name:string;status?:string}[]){if(busy.value)return;const targets=workspaceDeleteTargets(items.filter(row=>!bundleDeletionReason(row)),row=>row.id,row=>row.name);if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function openDelete(row:{id:number;name:string;status?:string}){openDeleteTargets([row])}
function onRemoved(ids:number[]){removedIds.value=new Set([...removedIds.value,...ids])}
onBeforeRouteLeave(()=>!busy.value)
watch(importSource,()=>{progress.value=null;stage.value='';operationError.value=''})
function select(event:Event){file.value=(event.target as HTMLInputElement).files?.[0];progress.value=null;operationError.value='';stage.value=''}
async function upload(){if(!file.value || busy.value)return;const selected=file.value,selectedKind=kind.value;uploading.value=true;operationError.value='';try{if(!selected.size)throw new Error('文件为空，请选择有效的交付材料');stage.value='正在计算文件校验值…';const hasher=CryptoJS.algo.SHA256.create(),chunkSize=8*1024*1024;for(let start=0;start<selected.size;start+=chunkSize){const bytes=new Uint8Array(await selected.slice(start,start+chunkSize).arrayBuffer());hasher.update(CryptoJS.lib.WordArray.create(bytes as any))}const hash=hasher.finalize().toString();let {offset}=await api('GET',`/bundles/uploads/${hash}`);if(!Number.isSafeInteger(offset) || offset<0 || offset>selected.size)throw new Error('服务器返回的上传偏移无效，请重试');progress.value=Math.floor(offset/selected.size*100);while(offset<selected.size){stage.value='正在上传材料，可断点续传…';const end=Math.min(offset+chunkSize,selected.size);const result:any=await request.put(`/bundles/uploads/${hash}`,selected.slice(offset,end),{headers:{'Content-Type':'application/octet-stream','Content-Range':`bytes ${offset}-${end-1}/${selected.size}`},timeout:120000,silentError:true});if(!Number.isSafeInteger(result.offset) || result.offset<=offset || result.offset>selected.size)throw new Error('上传进度未推进，请重新选择同一文件重试');offset=result.offset;progress.value=Math.floor(offset/selected.size*100)}await api('POST',`/bundles/uploads/${hash}/complete`,{kind:selectedKind,name:selected.name});stage.value='上传完成，服务器正在校验和加载镜像，请在下方查看导入结果。';ElMessage.success('材料已上传');await load()}catch(cause){operationError.value=errorText(cause)}finally{uploading.value=false}}
async function importDirectory(){if(busy.value || !serverFile.value.trim())return;importing.value=true;operationError.value='';try{await api('POST','/bundles/import-directory',{file:serverFile.value.trim(),kind:kind.value});ElMessage.success('服务器文件已提交导入');await load()}catch(cause){operationError.value=errorText(cause)}finally{importing.value=false}}
async function retry(row:any){if(busy.value)return;rowBusy.value=true;operationError.value='';try{await api('POST',`/bundles/${row.id}/retry`);ElMessage.success('已重新提交导入');await load()}catch(cause){operationError.value=errorText(cause)}finally{rowBusy.value=false}}
async function createVersion(row:any){if(busy.value)return;rowBusy.value=true;operationError.value='';try{const result=await api('POST',`/bundles/${row.id}/version`,{});rowBusy.value=false;await router.push(`/applications/${result.release.application_id}`)}catch(cause){operationError.value=errorText(cause)}finally{rowBusy.value=false}}
async function downloadBundle(row:any){try{await download(`/bundles/${row.id}/download`,row.name)}catch(cause){operationError.value=errorText(cause)}}
</script>
<style scoped>
.bundle-page { overflow: hidden; }
.bundle-import { display: flex; align-items: center; flex-wrap: wrap; flex-shrink: 0; padding: 12px 16px; gap: 8px 16px; }
.bundle-import-toolbar, .bundle-source-options, .bundle-import-form { display: flex; align-items: center; min-width: 0; gap: 8px; }
.bundle-import-toolbar { flex-wrap: wrap; gap: 8px; }
.bundle-kind { width: 140px; }
.bundle-import-form { flex: 1 1 214px; }
.bundle-source-options { flex-shrink: 0; }
.bundle-source-options :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) { background: #24343c; color: #b2d9ea; border-color: #35acd7; box-shadow: -1px 0 0 0 #35acd7; }
.bundle-import-help { width: 24px; height: 24px; padding: 4px; color: #999; }
.bundle-import-form > .el-button { flex-shrink: 0; margin: 0; }
.bundle-server-form > .el-input { flex: 1; min-width: 0; }
.bundle-file { position:relative; flex-shrink:0; border:1px solid #414141; border-radius:4px; background:#282828; padding:7px 12px; color:#ddd; font-size:12px; cursor:pointer; }.bundle-file:focus-within { outline:2px solid #33a7d5;outline-offset:2px; }.bundle-file input { position:absolute;inset:0;width:100%;height:100%;opacity:0;cursor:pointer; }
.bundle-file.disabled { opacity: .5; cursor: not-allowed; }.bundle-file.disabled input { cursor: not-allowed; }
.bundle-file-name { flex: 1; min-width:0; overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#999;font-size:12px; }
.bundle-progress { display: flex; flex: 1 1 100%; flex-direction: column; gap: 6px; }
.bundle-progress p { margin: 0; color: #999; font-size: 12px; line-height: 18px; overflow-wrap: anywhere; }
.bundle-operation-error { flex: 1 1 100%; padding: 6px 10px; border-radius: 4px; }
.bundle-records { flex: 1; min-height: 0; }
.bundle-records :deep(.resource-list-table) { min-height: 0; overflow: hidden; }
.bundle-records :deep(.resource-list-table > .el-table) { min-width: 0; }
.bundle-records :deep(.el-scrollbar__wrap) { overscroll-behavior: contain; }
@media (max-width: 760px) {
  .bundle-import { padding: 10px 12px; }
  .bundle-kind { width: 116px; }
  .bundle-records :deep(.management-toolbar) { flex-wrap: nowrap; padding: 10px 12px; gap: 10px; }
  .bundle-records :deep(.management-actions) { flex: 1; min-width: 0; }
  .bundle-records :deep(.management-search) { flex: 1; min-width: 0; width: 0; }
  .bundle-records :deep(.management-filters) { gap: 8px; padding: 0 12px 10px; }
  .bundle-records :deep(.management-filters > .el-select) { flex: 1 1 0; min-width: 0; width: 0; }
  .bundle-records :deep(.management-filters > .el-button) { flex-shrink: 0; margin: 0; }
  .bundle-records :deep(.management-pagination) { padding: 8px 12px; gap: 6px; }
  .bundle-records :deep(.el-pagination) { --el-pagination-button-width: 22px; --el-pagination-button-height: 28px; --el-pagination-font-size: 12px; --el-pagination-item-gap: 0; min-width: 0; }
  .bundle-records :deep(.management-pagination .el-select) { width: 90px; }
  .bundle-records :deep(.el-pagination__sizes) { margin-right: 6px; }
}
@media (max-height: 520px) and (min-width: 761px) {
  .bundle-page { gap: 8px; }
  .bundle-import { padding: 8px 12px; }
  .bundle-records { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-rows: auto auto minmax(0, 1fr) auto; grid-template-areas: 'toolbar filters' 'feedback feedback' 'table table' 'pagination pagination'; }
  .bundle-records :deep(.management-toolbar) { grid-area: toolbar; padding: 4px 12px; gap: 8px; }
  .bundle-records :deep(.management-actions) { flex: 1; }
  .bundle-records :deep(.management-search) { flex: 1; width: 0; }
  .bundle-records :deep(.management-filters) { grid-area: filters; flex-wrap: nowrap; gap: 6px; padding: 4px 12px 4px 0; }
  .bundle-records :deep(.management-filters > .el-select) { min-width: 0; width: 84px; }
  .bundle-records :deep(.management-summary) { display: none; }
  .bundle-records :deep(.management-error) { grid-area: feedback; }
  .bundle-records :deep(.resource-list-table) { grid-area: table; }
  .bundle-records :deep(.management-pagination) { grid-area: pagination; min-height: 44px; padding: 6px 12px; }
}
</style>
