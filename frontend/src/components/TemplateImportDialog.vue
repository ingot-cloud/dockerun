<template>
  <FormDialog ref="dialog" :model-value="modelValue" :draft="draft" title="导入应用模板" :loading="busy" extra-wide :editor="!!preview" :show-footer="false" @update:model-value="emit('update:modelValue',$event)" @cancel="emit('update:modelValue',false)">
    <div v-if="operationError" class="management-error management-feedback" role="alert">{{ operationError }}</div>
    <div v-if="!preview" class="template-package-upload" @dragover.prevent @drop.prevent="drop">
      <input ref="input" class="template-package-file-input" type="file" accept=".zip,application/zip" aria-label="选择 ZIP 模板包" :disabled="busy" @change="select" />
      <p class="management-description">选择或拖入标准模板 ZIP，可包含一个或多个模板。每个模板应包含 template.json 及引用的 Compose、配置文件和附加脚本。</p>
      <div class="template-package-file"><el-button :disabled="busy" @click="input?.click()">选择 ZIP 模板包</el-button><span>{{ file ? `${file.name} · ${formatSize(file.size)}` : '尚未选择文件' }}</span></div>
      <p class="management-form-help">ZIP 最大 32 MiB。预览和导入只保存模板，不执行脚本、不拉取镜像、不创建应用。</p>
      <div v-if="uploading" role="status"><el-progress :percentage="progress" :show-text="false" /><p class="management-form-help">{{ progress<100 ? `正在上传 ${progress}%` : '上传完成，正在校验模板…' }}</p></div>
    </div>
    <template v-else>
      <div class="template-package-summary"><span>共 {{ preview.entries.length }} 个模板，待导入 {{ pendingCount }} 个</span><span class="management-muted">{{ file?.name }}</span><el-button link :disabled="busy" @click="chooseAgain">重新选择</el-button></div>
      <div class="template-package-preview-table">
        <el-table :data="preview.entries" row-key="index" height="100%" highlight-current-row @current-change="viewEntry">
          <el-table-column label="模板" min-width="180"><template #default="{row}"><strong>{{row.name}}</strong><div class="management-muted">{{row.key}} · {{row.version}}</div></template></el-table-column>
          <el-table-column label="校验结果" min-width="180"><template #default="{row}"><span :class="{'template-package-conflict':row.status==='conflict'}">{{templateImportStatus(row.status)}}</span><div class="management-muted">{{row.message}}</div></template></el-table-column>
          <el-table-column label="处理方式" width="170"><template #default="{row}"><el-select v-model="choices[row.index].action" :aria-label="`${row.name}的导入方式`" :disabled="busy"><el-option v-if="!['identical','conflict'].includes(row.status)" label="导入" value="import" /><el-option label="跳过" value="skip" /><el-option label="另存为新模板" value="rename" /></el-select></template></el-table-column>
          <el-table-column label="检查内容" width="90"><template #default="{row}"><el-button link :disabled="busy" @click="viewEntry(row)">查看文件</el-button></template></el-table-column>
        </el-table>
      </div>
      <div v-if="renamed.length" class="template-package-rename">
        <div v-for="choice in renamed" :key="choice.index" class="template-package-rename-row"><span>{{preview.entries[choice.index].name}}</span><el-input v-model="choice.key" :aria-label="`${preview.entries[choice.index].name}的新模板标识`" placeholder="新模板标识，如 team-redis" maxlength="100" :disabled="busy" /><el-input v-model="choice.name" :aria-label="`${preview.entries[choice.index].name}的新模板名称`" placeholder="新模板名称" maxlength="100" :disabled="busy" /></div>
      </div>
      <div v-if="activeEntry" class="template-package-files">
        <div class="template-package-file-heading"><span>{{ activeEntry.name }} · {{ activeEntry.parameter_count }} 个参数 · {{activeEntry.config_count}} 个配置文件</span><span v-if="activeEntry.script_stages.length" class="management-muted">包含{{activeEntry.script_stages.join('、')}}脚本</span></div>
        <el-select v-model="activeFile" aria-label="查看模板文件" :disabled="busy"><el-option v-for="name in fileNames" :key="name" :value="name" :label="name" /></el-select>
        <CodeEditor v-if="activeFile" :key="`${preview.id}/${activeEntry.index}/${activeFile}`" :model-value="activeEntry.files[activeFile]" :filename="activeFile" :label="activeFile" readonly fill />
      </div>
      <p class="management-form-help template-package-preview-help">检查后确认导入。相同内容可跳过；同版本不同内容只能另存，已有版本不会被覆盖。附加脚本仅在后续确认发布时执行。</p>
    </template>
    <template #footer><el-button :disabled="busy" @click="dialog?.cancel()">取消</el-button><el-button v-if="!preview" type="primary" :loading="uploading" :disabled="!file" @click="upload">预览模板</el-button><el-button v-else type="primary" :loading="importing" :disabled="!pendingCount" @click="confirm">确认导入</el-button></template>
  </FormDialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import FormDialog from '@/components/FormDialog.vue'
import CodeEditor from '@/components/CodeEditor.vue'
import { applicationTemplatesApi, type TemplateImportChoice, type TemplateImportEntry, type TemplateImportPreview, type TemplateImportResult } from '@/api/applicationTemplates'
import { errorText } from '@/api/workspace'
import { defaultTemplateImportChoices, templateImportStatus, validateTemplateArchive, validateTemplateImportChoices } from '@/utils/templatePackages'
const props=defineProps<{modelValue:boolean}>()
const emit=defineEmits<{'update:modelValue':[value:boolean];imported:[value:TemplateImportResult]}>()
const dialog=ref<InstanceType<typeof FormDialog>>()
const file=ref<File>(),input=ref<HTMLInputElement>(),preview=ref<TemplateImportPreview>(),choices=ref<TemplateImportChoice[]>([]),activeEntry=ref<TemplateImportEntry>(),activeFile=ref(''),uploading=ref(false),importing=ref(false),progress=ref(0),operationError=ref('')
const busy=computed(()=>uploading.value||importing.value)
const draft=computed(()=>({file:file.value?`${file.value.name}:${file.value.size}:${file.value.lastModified}`:'',choices:choices.value}))
const renamed=computed(()=>choices.value.filter(choice=>choice.action==='rename'))
const pendingCount=computed(()=>choices.value.filter(choice=>choice.action==='rename'||choice.action==='import'&&preview.value?.entries[choice.index]?.status!=='identical').length)
const fileNames=computed(()=>Object.keys(activeEntry.value?.files || {}).sort((a,b)=>a==='template.json'?-1:b==='template.json'?1:a.localeCompare(b)))
let stagedID=''
watch(()=>props.modelValue,open=>{if(open){file.value=undefined;preview.value=undefined;choices.value=[];activeEntry.value=undefined;activeFile.value='';operationError.value='';progress.value=0}else{void releasePreview()}},{flush:'sync'})
onBeforeUnmount(()=>void releasePreview())
async function releasePreview(){const id=stagedID;stagedID='';if(id){try{await applicationTemplatesApi.cancel(id)}catch{ /* 预览另有服务端过期清理。 */ }}}
function formatSize(size:number){return size>=1024*1024?`${(size/1024/1024).toFixed(1)} MiB`:`${Math.ceil(size/1024)} KiB`}
function setFile(value?:File){operationError.value='';if(!value)return;try{validateTemplateArchive(value);file.value=value;progress.value=0}catch(cause){operationError.value=errorText(cause);file.value=undefined}}
function select(event:Event){setFile((event.target as HTMLInputElement).files?.[0])}
function drop(event:DragEvent){if(busy.value||preview.value)return;if(event.dataTransfer?.files.length!==1){operationError.value='请一次选择一个 ZIP 模板包';return}setFile(event.dataTransfer.files[0])}
function viewEntry(entry?:TemplateImportEntry){if(!entry)return;activeEntry.value=entry;activeFile.value=Object.prototype.hasOwnProperty.call(entry.files,'template.json')?'template.json':Object.keys(entry.files)[0] || ''}
async function chooseAgain(){
  if(busy.value)return
  if(renamed.value.length){try{await ElMessageBox.confirm('重新选择将清除当前另存设置。','重新选择模板包',{confirmButtonText:'重新选择',cancelButtonText:'继续检查',type:'warning'})}catch{return}}
  await releasePreview();preview.value=undefined;choices.value=[];activeEntry.value=undefined;activeFile.value='';operationError.value='';progress.value=0
}
async function upload(){if(!file.value||busy.value)return;operationError.value='';uploading.value=true;try{const result=await applicationTemplatesApi.preview(file.value,value=>progress.value=value);stagedID=result.id;preview.value=result;choices.value=defaultTemplateImportChoices(result.entries);viewEntry(result.entries[0])}catch(cause){operationError.value=errorText(cause)}finally{uploading.value=false}}
async function confirm(){if(!preview.value||busy.value)return;operationError.value='';try{validateTemplateImportChoices(preview.value.entries,choices.value)}catch(cause){operationError.value=errorText(cause);return}importing.value=true;try{const result=await applicationTemplatesApi.confirm(preview.value.id,choices.value.map(choice=>({...choice})));emit('imported',result);emit('update:modelValue',false)}catch(cause){operationError.value=errorText(cause)}finally{importing.value=false}}
</script>
<style scoped>
.template-package-file-input { display:none; }
.template-package-upload { display:flex;flex-direction:column;gap:12px;padding:16px;border:1px dashed #484848;border-radius:8px;background:#171717; }
.template-package-file,.template-package-summary,.template-package-file-heading { display:flex;align-items:center;gap:10px;flex-wrap:wrap;min-width:0; }
.template-package-file span,.template-package-summary .management-muted { overflow-wrap:anywhere; }
.template-package-summary,.template-package-preview-help { flex-shrink:0; }
.template-package-preview-table { flex:0 0 194px;min-height:120px;overflow:hidden; }
.template-package-preview-table .management-muted { font-size:11px;line-height:1.5;overflow-wrap:anywhere; }
.template-package-preview-table strong { font-size:13px;font-weight:500; }
.template-package-conflict { color:#e6a990; }
.template-package-rename { max-height:130px;overflow:auto;flex-shrink:0;display:flex;flex-direction:column;gap:8px; }
.template-package-rename-row { display:grid;grid-template-columns:minmax(100px,1fr) minmax(150px,2fr) minmax(140px,2fr);align-items:center;gap:8px; }
.template-package-rename-row>span { overflow-wrap:anywhere;font-size:12px; }
.template-package-files { display:flex;flex-direction:column;gap:8px;flex:1;min-height:0;min-width:0; }
.template-package-file-heading { font-size:12px;flex-shrink:0; }
.template-package-files>.el-select { flex-shrink:0; }
.template-package-files>.code-editor-wrapper { flex:1;min-height:0; }
.template-package-preview-help { margin:0; }
@media(max-width:760px){.template-package-preview-table{flex-basis:168px}.template-package-rename-row{grid-template-columns:1fr}.template-package-rename{max-height:180px}.template-package-upload{padding:12px}.template-package-file-heading{font-size:11px}}
</style>
