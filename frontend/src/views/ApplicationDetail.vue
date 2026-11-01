<template>
  <WorkspaceEditor class="application-detail-editor" :class="{'application-detail-editor--list':activeTab!=='configuration'}" :title="data.application?.name || '应用详情'" :loading="loading" :busy="busy" :reset-key="activeTab" @refresh="load()">
    <template #actions><el-button :disabled="loading || !data.application || busy || !environmentRows.length" @click="editDeployment()">编辑部署配置</el-button><el-button type="primary" :disabled="loading || !data.application || busy || !environmentRows.length" :title="environmentRows.length?'编辑部署内容并发布应用':'请先添加部署环境'" @click="prepareDeployment()">发布应用</el-button></template>
    <template #navigation>
      <div v-if="data.application && setupStep<3" class="application-setup-guide" aria-label="首次发布指引">
        <ApplicationSetupSteps :step="setupStep" />
        <div class="application-setup-next"><span>{{setupStep===1?'下一步：选择主机，配置首个环境。':'下一步：编辑部署命令或 Compose，预览后首次发布。'}}</span><el-button link type="primary" :disabled="busy || loading || (setupStep===1 && (optionsLoading || !!optionsError))" @click="setupStep===1?edit():prepareDeployment()">{{setupStep===1?'配置首个环境':'准备首次发布'}}<span aria-hidden="true"> →</span></el-button></div>
      </div>
      <nav class="application-detail-tabs" aria-label="应用详情视图">
        <button v-for="tab in tabs" :key="tab.value" type="button" :aria-current="activeTab===tab.value?'page':undefined" :disabled="busy" @click="activeTab=tab.value">{{tab.label}}<span v-if="tab.value!=='configuration'">{{tab.value==='environments'?environmentRows.length:versions.length}}</span></button>
      </nav>
    </template>
    <template v-if="error || actionError || optionsError" #feedback>
      <div v-if="error || actionError" class="management-error" role="alert"><span>{{error || actionError}}</span><el-button link :disabled="loading || busy" @click="actionError='';load()">重试</el-button></div>
      <div v-if="optionsError" class="management-warning" role="alert"><span>{{optionsError}}</span><el-button link :disabled="optionsLoading || busy" @click="loadOptions">重试</el-button></div>
    </template>
    <div class="application-detail-page workspace-editor-page">
      <ResourceList v-if="activeTab==='environments'" key="environments" class="application-environment-list" selection :selectable="row=>!environmentDeletionReason(row)" @selection-change="selectedEnvironments=$event" title="部署环境" icon="Connection" :items="environmentRows" :loading="loading" :busy="busy" :search-keys="['name','target_name','current_version']" query-prefix="env_" placeholder="搜索环境、主机或当前版本…" :sorts="[{key:'name',label:'环境名称'}]" empty-text="暂无部署环境" @refresh="load()">
        <template #actions><el-button type="danger" :disabled="busy || loading || !selectedEnvironments.length" :title="selectedEnvironments.length?'删除选中的部署环境':'请先勾选没有活动任务的部署环境'" @click="openDeleteEnvironments(selectedEnvironments)">批量删除</el-button><el-button type="primary" :disabled="busy || loading || optionsLoading || !!optionsError || !data.application" @click="edit()">添加环境</el-button></template>
        <el-table-column prop="name" label="环境" min-width="160" show-overflow-tooltip />
        <el-table-column prop="target_name" label="部署目标" min-width="150" show-overflow-tooltip />
        <el-table-column prop="current_version" label="当前版本" min-width="150" show-overflow-tooltip />
        <el-table-column label="健康状态" min-width="180"><template #default="{row}"><StatusBadge class="application-health-state" :value="row.health" /></template></el-table-column>
        <el-table-column label="运行镜像" min-width="300"><template #default="{row}">
          <div v-for="container in runtimeImages.filter(item=>item.environment_id===row.id && item.state==='running')" :key="`${container.target_id}:${container.id}`" class="application-runtime-image">
            <router-link v-if="runtimeImageLocation(container)" :to="runtimeImageLocation(container)!" :title="container.image" :aria-label="`查看镜像详情：${container.image || container.image_id}`">{{container.name}} · {{container.image || '镜像引用未知'}}</router-link>
            <span v-else :title="container.target_id>1?'镜像位于远程主机，暂不支持在此查看详情':container.image">{{container.name}} · {{container.image || '镜像引用未知'}}</span>
            <code :title="container.image_id">{{container.image_id?.replace('sha256:','').slice(0,12) || 'Image ID 未知'}}</code>
          </div>
          <span v-if="!runtimeImages.some(item=>item.environment_id===row.id && item.state==='running')" class="management-muted">{{!commandApplication || row.container_names?.length?'尚未核对到运行容器':'未关联容器'}}</span>
        </template></el-table-column>
        <el-table-column label="发布方式" min-width="170"><template #default="{row}">{{row.auto_deploy?'自动（受推送人限制）':'手动'}}</template></el-table-column>
        <el-table-column label="操作" width="300" :fixed="narrow?false:'right'"><template #default="{row}"><div class="management-row-actions"><el-button link type="primary" :disabled="busy || !!optionsError || optionsLoading" @click="edit(row)">配置</el-button><el-button link type="primary" :disabled="busy || !!optionsError || optionsLoading" @click="edit(row,true)">自动发布</el-button><el-button link :disabled="busy || !!optionsError || optionsLoading" @click="clone(row)">复制</el-button><router-link :to="{path:'/deployments',query:{environment_id:row.id}}">记录</router-link><el-tooltip :content="environmentDeletionReason(row)" :disabled="!environmentDeletionReason(row)"><span><el-button link type="danger" :disabled="busy || loading || !!environmentDeletionReason(row)" :aria-label="`删除部署环境 ${row.name} #${row.id}`" @click="openDeleteEnvironments([row])">删除</el-button></span></el-tooltip></div></template></el-table-column>
        <template #empty><span>添加首个环境，选择目标主机并填写部署参数。</span></template>
      </ResourceList>
      <div v-else-if="activeTab==='releases'" class="application-version-section"><p class="management-description">「部署此版本」可以将版本部署到其他环境，也可以重新部署历史版本。</p>
      <ResourceList key="releases" title="发布版本" icon="Collection" :items="versions" :loading="loading" :busy="busy" selection :selectable="row=>!releaseDeletionReason(row)" :search-keys="['version','platform','id']" query-prefix="release_" placeholder="搜索版本、平台或编号…" :sorts="[{key:'created_at',label:'创建时间'},{key:'version',label:'版本名称'}]" empty-text="暂无发布版本" @refresh="load()" @selection-change="selectedVersions=$event">
        <template #actions><el-button type="danger" :disabled="busy || loading || !selectedVersions.length" :title="selectedVersions.length?'删除选中的发布版本':'请先勾选未被使用的发布版本'" @click="openDeleteVersions(selectedVersions)">批量删除</el-button></template>
        <el-table-column prop="version" label="版本" min-width="190" show-overflow-tooltip />
        <el-table-column label="来源" width="130"><template #default="{row}">{{sourceLabel(row.source)}}</template></el-table-column>
        <el-table-column label="平台" width="130"><template #default="{row}">{{row.platform || '—'}}</template></el-table-column>
        <el-table-column label="创建时间" width="180"><template #default="{row}">{{formatDate(row.created_at)}}</template></el-table-column>
        <el-table-column label="操作" width="420" :fixed="narrow?false:'right'"><template #default="{row}"><div class="management-row-actions"><el-button link type="primary" :disabled="busy || !environmentRows.length" :title="'可以部署到其他环境，也可以重新部署历史版本'" @click="prepareDeployment(row.id)">部署此版本</el-button><el-button link type="primary" :disabled="busy || !environmentRows.length" @click="editDeployment(row.id)">基于此版本编辑</el-button><el-button link :disabled="busy" @click="inspect=row">查看版本</el-button><el-button v-if="row.source==='bundle'" link type="primary" :loading="publishing===row.id" :disabled="busy" @click="publishBundle(row)">镜像入库</el-button><el-tooltip :content="releaseDeletionReason(row)" :disabled="!releaseDeletionReason(row)"><span><el-button link type="danger" :disabled="busy || loading || !!releaseDeletionReason(row)" :aria-label="`删除发布版本 ${row.version} #${row.id}`" @click="openDeleteVersions([row])">删除</el-button></span></el-tooltip></div></template></el-table-column>
        <template #empty><span>通过「发布应用」编辑命令或 Compose，创建首个版本。</span></template>
      </ResourceList>
      </div>
      <div v-else class="workspace-page application-configuration">
    <section class="management-card"><header class="management-toolbar"><h2>部署配置</h2><el-button :disabled="busy || !environmentRows.length" @click="editDeployment()">编辑部署配置</el-button></header><div class="management-card-body"><p class="management-description">编辑部署命令、Compose、配置文件与前后置脚本，保存后供后续手动和自动发布使用。历史版本与已提交任务保留原快照。</p><div v-for="env in environments" :key="env.id" class="management-row-actions"><span>{{env.name}} · {{env.auto_deploy?'自动发布已开启':'手动发布'}}</span><el-button link type="primary" :disabled="busy || optionsLoading || !!optionsError" @click="edit(env,true)">配置自动发布</el-button></div></div></section>
    <p class="management-description">{{data.application?.description || '独立管理各环境的版本、健康与发布授权。'}}</p>
    <section v-if="data.application?.definition?.compose" class="management-card"><header class="management-toolbar"><h2>{{templateCategory(data.application.category)}}配置</h2><el-button :disabled="busy" @click="openSaveTemplate">另存为应用模板</el-button></header><div class="management-card-body" style="padding-top:0"><p class="management-description">{{data.application.template_id?`创建来源：${data.application.template_id} · ${data.application.template_version}。`:'此应用已保存可复用定义。'}} 应用持有独立配置，模板更新不会修改此应用。</p><p v-if="data.application.definition.storage" class="management-form-help">持久化：{{data.application.definition.storage}}</p><p v-if="data.application.definition.initialization" class="management-form-help">首次初始化：{{data.application.definition.initialization}}</p><p class="management-form-help">{{data.application.definition.instructions}}</p><p v-if="data.application.definition.stateful" class="management-form-help">升级方式：先在环境配置中调整镜像版本，再点击「发布应用」预检查并手动部署。业务数据需单独备份。</p></div></section>
    <section v-if="data.application?.definition?.endpoints?.length && environments.some((env:any)=>env.current_release_id)" class="management-card"><header class="management-toolbar"><h2>环境访问配置</h2></header><div class="management-card-body" style="padding-top:0"><template v-for="env in environments.filter((env:any)=>env.current_release_id)" :key="env.id"><p class="management-description">{{env.name}}{{env.target_id>1?'（远程主机请替换为实际访问地址）':''}}</p><div v-for="endpoint in environmentEndpoints(env)" :key="endpoint.name" class="management-row-actions application-endpoint"><span>{{endpoint.name}}</span><code>{{endpoint.address}}</code><el-button link @click="copyEndpoint(endpoint.address)">复制地址</el-button></div></template><p class="management-form-help">地址根据环境参数生成，发布后请确认网络可达。访问凭据在环境配置中管理。</p></div></section>
    <section class="management-card"><header class="management-toolbar"><h2>运行容器与镜像</h2></header><div class="management-card-body"><p class="management-form-help">以下为目标主机最近一次核对的实际容器信息；远程主机在部署完成后上报。</p><el-table v-if="runtimeImages.length" :data="runtimeImages" size="small"><el-table-column prop="environment_name" label="环境" min-width="100" /><el-table-column prop="target_name" label="主机" min-width="100" /><el-table-column prop="name" label="容器" min-width="150" show-overflow-tooltip /><el-table-column label="镜像引用" min-width="230" show-overflow-tooltip><template #default="{row}"><router-link v-if="runtimeImageLocation(row)" class="application-runtime-image-link" :to="runtimeImageLocation(row)!" :aria-label="`查看镜像详情：${row.image || row.image_id}`">{{row.image || row.image_id}}</router-link><span v-else :title="row.target_id>1?'镜像位于远程主机，暂不支持在此查看详情':undefined">{{row.image || '—'}}</span></template></el-table-column><el-table-column prop="image_id" label="Image ID" min-width="260" show-overflow-tooltip /><el-table-column label="状态" width="100"><template #default="{row}"><StatusBadge :value="row.state" :label="row.state==='missing'?'容器不存在':containerStateLabel(row.state)" /></template></el-table-column><el-table-column label="核对时间" min-width="165"><template #default="{row}">{{formatDate(row.checked_at)}}</template></el-table-column></el-table><p v-else class="management-description">暂无实际镜像信息。命令部署请先关联容器，执行部署后核对；Compose 自动记录本应用项目的容器。</p></div></section>
    <section class="management-card"><header class="management-toolbar"><h2>应用附加脚本</h2><el-button :disabled="busy || !data.application" @click="openScripts">配置附加脚本</el-button></header><div class="management-card-body" style="padding-top:0"><p class="management-description">{{ hasScripts(data.application?.scripts) ? '已配置应用脚本，手动和自动创建的新版本都会保存脚本快照。' : '尚未配置脚本，可在发布应用时按需添加前后置脚本，也可在此设置应用默认脚本。' }} 环境可以单独覆盖或停用脚本，已有版本保留原脚本。</p></div></section>
        <section class="management-card"><header class="management-toolbar"><h2>发布操作指引</h2></header><div class="management-card-body application-instructions"><p>先添加环境并选择主机，再通过「发布应用」编辑命令或 Compose；运行参数、前后置脚本和高级配置分组管理，预览后确认部署。</p><p>保存部署配置后，可通过环境行的「自动发布」直接配置仓库、镜像名称、标签规则和允许触发的用户。命令部署通过 DOCKERUN_IMAGE 使用本次推送的镜像。推送到内置仓库无需配置 Webhook。</p></div></section>
      </div>
    <FormDialog v-model="visible" :draft="{form,parameterPairs:parameterPairs.map(({key,value})=>({key,value})),targetIds,publishers}" :title="automaticOnly?`自动发布 · ${form.name}`:'环境配置'" wide :loading="saving" :confirm-text="automaticOnly?'保存自动发布':'保存环境'" @confirm="automaticOnly?saveAutomatic():save()" @cancel="visible=false">
      <div v-if="formError" class="management-error management-feedback" role="alert">{{formError}}</div>
      <el-form :model="form" class="application-environment-form" label-position="top" :disabled="saving" @submit.prevent="automaticOnly?saveAutomatic():save()">
        <template v-if="!automaticOnly"><el-form-item label="环境名称" required>
          <el-input v-model="form.name" placeholder="例如：测试环境、生产环境" maxlength="100" />
        </el-form-item>
        <el-form-item label="部署目标" required>
          <el-select v-model="targetSelection" :multiple="!singleHost" placeholder="选择目标主机">
            <el-option v-for="host in hosts" :key="host.id" :value="host.id" :label="host.name" :disabled="host.id!==1 && !host.enabled" />
          </el-select>
          <p class="management-form-help">{{singleHost?'此应用整套服务仅支持单台主机。':'可选择一台或多台主机；多台按选择顺序逐台部署，失败后暂停后续任务。'}}</p>
        </el-form-item>
        <section class="environment-config-section" aria-labelledby="environment-parameters-heading">
          <h3 id="environment-parameters-heading">运行参数</h3>
          <EnvironmentParameterPairs v-model="parameterPairs" :schema="parameterSchema" :disabled="saving" />
        </section>
        <ContainerBindingEditor v-if="commandApplication" v-model:names="form.container_names" v-model:replace="form.replace_containers" :command="commandApplication" :disabled="saving" />
        </template>
        <el-collapse v-model="environmentSections" class="environment-config-groups">
          <el-collapse-item name="automatic" :disabled="saving">
            <template #title><span class="environment-config-title">自动发布<span class="environment-config-summary">{{form.auto_deploy?'已开启':'未开启'}}</span></span></template>
            <el-form-item label="镜像仓库" required>
              <el-select v-model="form.registry_id" @change="changeEnvironmentRegistry" placeholder="默认使用内置仓库">
                <el-option v-for="registry in registries" :key="registry.id" :value="registry.id" :label="registry.name" />
              </el-select>
              <p class="management-form-help">{{selectedRegistry?.kind==='builtin'?'默认使用内置仓库，无需配置 Webhook。':'使用外部仓库；开启自动发布时需要接入外部回调。'}}</p>
            </el-form-item>
            <el-switch v-model="form.auto_deploy" active-text="允许镜像提交后自动发布" aria-label="允许镜像提交后自动发布" />
            <p class="management-form-help">匹配推送按 tag 和摘要判断更新，同 tag 且同摘要时跳过。<el-button link type="primary" :disabled="saving" @click="openDeploymentFromEnvironment">编辑部署配置</el-button></p>
            <template v-if="form.auto_deploy">
              <div class="management-warning management-feedback">默认仅管理员推送可触发；普通用户需具有推送权限并在此明确授权。</div>
              <el-form-item label="镜像名称（不含仓库域名和标签）" required><el-input v-model="form.repository" placeholder="例如：team/gateway" /><p class="management-form-help">例如 nginx 或 team/gateway。Compose 配置需包含此镜像；命令部署在脚本中使用 $DOCKERUN_IMAGE。</p></el-form-item>
              <el-form-item label="标签规则"><el-input v-model="form.tag_pattern" placeholder="例如：latest、* 或 test-*" /></el-form-item>
              <el-form-item label="允许触发的普通用户"><el-select v-model="publishers" multiple placeholder="默认仅管理员"><el-option v-for="user in users.filter(user=>!user.is_admin && user.enabled)" :key="user.id" :value="user.id" :label="user.username" /></el-select></el-form-item>
              <p v-if="automaticIssue" class="management-warning management-feedback" role="status">{{automaticIssue}}</p>
              <p v-else class="management-form-help">保存后，需要更新的镜像推送将使用最新保存的部署配置与环境参数创建任务。不同 tag 即使内容相同也会部署，同 tag 内容更新时也会部署。命令部署会先准备镜像，并通过 DOCKERUN_IMAGE 传入固定摘要引用。发布包镜像入库不会触发自动发布。</p>
            </template>
            <ContainerBindingEditor v-if="automaticOnly && commandApplication" v-model:names="form.container_names" v-model:replace="form.replace_containers" :command="commandApplication" :disabled="saving" />
          </el-collapse-item>
          <el-collapse-item v-if="!automaticOnly" title="高级配置" name="advanced" :disabled="saving">
            <section class="environment-config-section" aria-labelledby="environment-scripts-heading">
              <h3 id="environment-scripts-heading">附加脚本</h3>
              <el-form-item label="附加脚本来源"><el-select v-model="form.script_mode"><el-option value="inherit" label="使用版本中的应用附加脚本" /><el-option value="override" label="使用此环境的独立附加脚本" /><el-option value="disabled" label="不执行附加脚本" /></el-select><p class="management-form-help">控制部署前后置脚本及兼容的旧启动脚本，不改变主部署命令和部署检查。新任务保存执行快照，之后编辑不会改变已提交任务。</p></el-form-item>
              <DeploymentScriptEditor v-if="form.script_mode==='override'" v-model="form.scripts" :disabled="saving" />
              <p v-else class="management-form-help">{{form.script_mode==='disabled'?'跳过附加脚本，主部署命令与部署检查仍然执行；普通 Compose 由系统启动。':'手动与自动创建的新版本使用保存后的应用附加脚本；部署已有版本时沿用所选版本的附加脚本。'}}</p>
            </section>
            <section class="environment-config-section" aria-labelledby="environment-recovery-heading">
              <h3 id="environment-recovery-heading">恢复策略</h3>
              <div class="environment-recovery-options"><el-checkbox v-model="form.stateless" :disabled="data.application?.definition?.stateful">无状态应用</el-checkbox><el-checkbox v-model="form.auto_rollback" :disabled="!form.stateless">失败自动恢复</el-checkbox></div>
              <p class="management-form-help">{{data.application?.definition?.stateful?'此应用包含持久数据，不能启用无状态自动恢复。':'仅无状态应用可启用自动恢复；恢复上一成功版本的服务与配置，不回滚业务数据。'}}</p>
            </section>
          </el-collapse-item>
        </el-collapse>
        <button type="submit" hidden>保存环境</button>
      </el-form>
    </FormDialog>
    <FormDialog :model-value="!!inspect" title="不可变发布版本" extra-wide :show-footer="false" @update:model-value="(value:boolean)=>{if(!value)inspect=null}"><pre class="management-code">{{JSON.stringify(inspect,null,2)}}</pre></FormDialog>
    <FormDialog v-model="scriptsVisible" editor :draft="scriptDraft" title="应用附加脚本" extra-wide :loading="scriptsSaving" confirm-text="保存应用脚本" @confirm="saveScripts" @cancel="scriptsVisible=false"><div v-if="scriptsError" class="management-error management-feedback" role="alert">{{scriptsError}}</div><p class="management-description">用于之后手动或自动创建的新版本。环境选择独立脚本或停用脚本时，优先使用环境配置。</p><DeploymentScriptEditor v-model="scriptDraft" :disabled="scriptsSaving" fill /></FormDialog>
    <FormDialog v-model="saveTemplateVisible" :draft="templateForm" title="另存为应用模板" :loading="templateSaving" confirm-text="保存模板" @confirm="saveAsTemplate" @cancel="saveTemplateVisible=false"><div v-if="templateError" class="management-error management-feedback" role="alert">{{templateError}}</div><p class="management-description">保存应用定义、非敏感配置文件和应用脚本；环境参数值、主机和业务数据不会复制。请确认配置文件及脚本没有固定凭据，已有模板版本不会被覆盖。</p><el-form label-position="top" :disabled="templateSaving"><el-form-item label="模板名称" required><el-input v-model="templateForm.name" maxlength="100" /></el-form-item><el-form-item label="模板版本" required><el-input v-model="templateForm.version" placeholder="例如：1.0.0" maxlength="100" /></el-form-item><el-form-item label="用途说明"><el-input v-model="templateForm.description" type="textarea" :rows="2" /></el-form-item></el-form></FormDialog>
    <ManagementDeleteDialog v-model="environmentDeleteVisible" :items="environmentDeleteTargets" kind="部署环境" :endpoint="`/applications/${id}/environments`" warning="将删除所选环境及其部署状态，无法撤销。应用、发布版本、部署历史和日志会保留，已部署的容器、网络、卷和镜像会保留；删除后无法再通过此环境管理或回滚。存在排队、执行或恢复中的任务时不能删除。" @removed="onEnvironmentsRemoved" />
    <ManagementDeleteDialog v-model="deleteVisible" :items="deleteTargets" kind="发布版本" endpoint="/releases" warning="将删除所选版本定义，无法撤销，删除后不能再选择该版本进行部署。部署历史、执行快照、已部署资源、镜像及原始发布包会保留。正在使用或被排队、执行、恢复中的任务引用的版本不能删除。" @removed="onVersionsRemoved" />
  </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { computed,onBeforeUnmount,onMounted,ref,watch } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useRoute,useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/api/request'
import { api,errorText } from '@/api/workspace'
import { useResource } from '@/composables/useResource'
import { formatDate,parseArray,sourceLabel } from '@/utils/management'
import { containerStateLabel } from '@/utils/containers'
import WorkspaceEditor from '@/components/WorkspaceEditor.vue'
import ResourceList from '@/components/ResourceList.vue'
import FormDialog from '@/components/FormDialog.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import DeploymentScriptEditor from '@/components/DeploymentScriptEditor.vue'
import ApplicationSetupSteps from '@/components/ApplicationSetupSteps.vue'
import ManagementDeleteDialog from '@/components/ManagementDeleteDialog.vue'
import { workspaceDeleteTargets } from '@/utils/workspaceDeletion'
import { runtimeImageLocation,applicationRuntimeImages,applicationDetailTab,applicationSetupStep,environmentDeletionReason,environmentTargetConfiguration,environmentParameterPairs,environmentParameterValues,type ApplicationDetailTab,type EnvironmentParameterPair } from '@/utils/applicationDetail'
import { automationIssue, copyScripts, defaultReleaseRegistry, emptyScripts, hasScripts, imageRepository, releaseImages, releaseDeletionReason } from '@/utils/releases'
import EnvironmentParameterPairs from '@/components/EnvironmentParameterPairs.vue'
import ContainerBindingEditor from '@/components/ContainerBindingEditor.vue'
import { applicationTemplatesApi,type ApplicationTemplate,type TemplateParameter } from '@/api/applicationTemplates'
import { templateCategory,templateValues,validateTemplateValues,retargetTemplateParameterPairs,templateEndpoints } from '@/utils/applicationTemplates'
const route=useRoute(),router=useRouter(),id=route.params.id
const narrow=useMediaQuery('(max-width: 760px)')
const {data,loading,error,load}=useResource<any>(`/applications/${id}`,{})
const tabs:{value:ApplicationDetailTab;label:string}[]=[{value:'environments',label:'部署环境'},{value:'releases',label:'发布版本'},{value:'configuration',label:'应用配置'}]
const activeTab=computed({get:()=>applicationDetailTab(route.query.app_view),set:(value:ApplicationDetailTab)=>{if(!busy.value)void router.replace({query:{...route.query,app_view:value}})}})
const deleteVisible=ref(false),deleteTargets=ref<{id:number;name:string}[]>([]),removedVersionIds=ref(new Set<number>()),selectedVersions=ref<any[]>([])
const environmentDeleteVisible=ref(false),environmentDeleteTargets=ref<{id:number;name:string}[]>([]),removedEnvironmentIds=ref(new Set<number>()),selectedEnvironments=ref<any[]>([])
const versions=computed(()=>(data.value.releases || []).filter((row:any)=>!removedVersionIds.value.has(row.id)))
const environments=computed(()=>(data.value.environments || []).filter((row:any)=>!removedEnvironmentIds.value.has(row.id)))
const environmentRows=computed(()=>environments.value.map((row:any)=>({...row,target_name:environmentTarget(row),current_version:row.current_release_id?(versions.value.find((version:any)=>version.id===row.current_release_id)?.version || `版本 #${row.current_release_id}`):'尚未发布'})))
const runtimeImages=computed(()=>applicationRuntimeImages(environments.value,data.value.targets || [],hosts.value))
const commandApplication=computed(()=>data.value.application?.definition?.execution?.mode==='command' || data.value.application?.kind==='command')
const setupStep=computed(()=>applicationSetupStep(environments.value,(data.value.targets || []).filter((row:any)=>!removedEnvironmentIds.value.has(row.environment_id))))
const visible=ref(false),saving=ref(false),publishing=ref(0),actionError=ref(''),formError=ref(''),optionsError=ref(''),optionsLoading=ref(false)
const users=ref<any[]>([]),registries=ref<any[]>([]),hosts=ref<any[]>([]),form=ref<any>({}),parameterPairs=ref<EnvironmentParameterPair[]>([]),targetIds=ref<number[]>([]),publishers=ref<number[]>([]),inspect=ref<any>(null)
const environmentSections=ref<string[]>([]),automaticOnly=ref(false)
const singleHost=computed(()=>data.value.application?.definition?.topology==='single_host')
const targetSelection=computed<number | number[] | undefined>({get:()=>singleHost.value?targetIds.value[0]:targetIds.value,set:value=>{targetIds.value=Array.isArray(value)?value:typeof value==='number'?[value]:[]}})
const scriptsVisible=ref(false),scriptsSaving=ref(false),scriptsError=ref(''),scriptDraft=ref(emptyScripts())
const parameterSchema=computed<TemplateParameter[]>(()=>data.value.application?.definition?.parameters || [])
const saveTemplateVisible=ref(false),templateSaving=ref(false),templateError=ref(''),templateForm=ref({name:'',version:'1.0.0',description:''});let previousEnvironmentRegistry=0
const busy=computed(()=>saving.value || scriptsSaving.value || templateSaving.value || !!publishing.value || deleteVisible.value || environmentDeleteVisible.value),controller=new AbortController();let disposed=false,setupOpened=false
watch(activeTab,()=>{selectedVersions.value=[];selectedEnvironments.value=[]})
watch([()=>data.value.application?.id,()=>optionsLoading.value,()=>optionsError.value,()=>route.query.setup],()=>{if(!setupOpened && route.query.setup==='environment' && data.value.application && !loading.value && !optionsLoading.value && !optionsError.value && !environmentRows.value.length){setupOpened=true;edit()}},{flush:'post'})
function environmentTarget(row:any){const ids=parseArray(row.target_ids);return (ids.length?ids:[row.target_id || 1]).map(id=>hosts.value.find(host=>host.id===id)?.name || (id===1?'本机':`主机 #${id}`)).join('、')}
function editDeployment(version?:number){if(busy.value || loading.value || !environmentRows.value.length)return;const query={...route.query};delete query.release;delete query.setup;delete query.edit_release;query.mode='edit';if(version)query.edit_release=String(version);void router.push({path:`/applications/${id}/deploy`,query})}
function prepareDeployment(version?:number){if(busy.value || loading.value || !environmentRows.value.length)return;const query={...route.query};delete query.release;delete query.setup;delete query.edit_release;delete query.mode;if(version)query.release=String(version);void router.push({path:`/applications/${id}/deploy`,query})}
function openDeleteVersions(items:any[]){if(busy.value || loading.value)return;const targets=workspaceDeleteTargets(items.filter(row=>!releaseDeletionReason(row)),row=>row.id,row=>row.version || `版本 #${row.id}`);if(!targets.length)return;deleteTargets.value=targets;deleteVisible.value=true}
function onVersionsRemoved(ids:number[]){removedVersionIds.value=new Set([...removedVersionIds.value,...ids]);void load(true)}
function openDeleteEnvironments(items:any[]){if(busy.value || loading.value)return;const targets=workspaceDeleteTargets(items.filter(row=>!environmentDeletionReason(row)),row=>row.id,row=>row.name || `环境 #${row.id}`);if(!targets.length)return;environmentDeleteTargets.value=targets;environmentDeleteVisible.value=true}
function onEnvironmentsRemoved(ids:number[]){removedEnvironmentIds.value=new Set([...removedEnvironmentIds.value,...ids]);void load(true)}
const selectedRegistry=computed(()=>registries.value.find(reg=>reg.id===form.value.registry_id))
const automaticIssue=computed(()=>automationIssue(form.value,data.value.releases || [],data.value.application))
async function loadOptions(){if(optionsLoading.value)return;optionsLoading.value=true;optionsError.value='';const results=await Promise.allSettled(['/users','/registries','/targets'].map(path=>api<any[]>('GET',path,undefined,controller.signal)));if(disposed)return;const targets=[users,registries,hosts];results.forEach((result,index)=>{if(result.status==='fulfilled')targets[index].value=result.value || [];else optionsError.value='部分配置选项加载失败，请重试后配置环境。'});optionsLoading.value=false}
function edit(row?:any,onlyAutomatic=false){
  automaticOnly.value=onlyAutomatic
  const base=data.value.releases?.find((r:any)=>r.id===row?.current_release_id)
  form.value={name:'',target_id:1,stateless:false,auto_rollback:false,auto_deploy:false,repository:'',tag_pattern:'*',...row,registry_id:row?.registry_id || defaultReleaseRegistry(registries.value),script_mode:row?.script_mode || 'inherit',scripts:copyScripts(row?.scripts),container_names:[...(row?.container_names || [])],replace_containers:Boolean(row?.replace_containers)}
  if(!form.value.repository && base?.source==='registry' && base.registry_id===form.value.registry_id)form.value.repository=imageRepository(releaseImages(base)[0]?.reference || '')
  const targets=parseArray(row?.target_ids)
  targetIds.value=targets.length?targets:[row?.target_id || 1]
  publishers.value=parseArray(row?.allowed_publishers)
  const values=row?(row.parameters || {}):parameterSchema.value.length?templateValues({definition:data.value.application.definition} as ApplicationTemplate,registries.value.find(item=>item.id===form.value.registry_id)):{}
  parameterPairs.value=environmentParameterPairs(values,parameterSchema.value)
  formError.value=''
  previousEnvironmentRegistry=form.value.registry_id
  environmentSections.value=onlyAutomatic || row?.auto_deploy?['automatic']:[]
  visible.value=true
}
async function openDeploymentFromEnvironment(){const environmentId=form.value.id;await router.push({path:`/applications/${id}/deploy`,query:{mode:'edit',environment:environmentId}})}
function clone(row:any){edit({...row,id:0,current_release_id:0,name:`${row.name}-副本`,auto_deploy:false,container_names:[],replace_containers:false})}
function changeEnvironmentRegistry(){if(automaticOnly.value)return;parameterPairs.value=retargetTemplateParameterPairs(parameterSchema.value,parameterPairs.value,registries.value.find(item=>item.id===previousEnvironmentRegistry),selectedRegistry.value);previousEnvironmentRegistry=form.value.registry_id}
function environmentParameters(){const values=environmentParameterValues(parameterPairs.value,parameterSchema.value);if(parameterSchema.value.length)validateTemplateValues(parameterSchema.value,values);return values}
async function saveAutomatic(){
  if(saving.value)return
  formError.value=''
  try{
    if(form.value.auto_deploy && automaticIssue.value)throw new Error(automaticIssue.value)
    saving.value=true
    await api('PATCH',`/applications/${id}/environments/${form.value.id}/automation`,{auto_deploy:Boolean(form.value.auto_deploy),registry_id:form.value.registry_id,repository:form.value.repository?.trim() || '',tag_pattern:form.value.tag_pattern || '*',allowed_publishers:JSON.stringify(publishers.value),container_names:form.value.container_names || [],replace_containers:Boolean(form.value.replace_containers)})
    visible.value=false;ElMessage.success('自动发布配置已保存');await load()
  }catch(cause){formError.value=errorText(cause)}finally{saving.value=false}
}
async function save(){if(saving.value)return;formError.value='';try{if(!form.value.name?.trim())throw new Error('请输入环境名称');const targets=environmentTargetConfiguration(targetIds.value,singleHost.value);if(form.value.auto_deploy && automaticIssue.value)throw new Error(automaticIssue.value);const payload={id:form.value.id || 0,name:form.value.name.trim(),...targets,stateless:Boolean(form.value.stateless),auto_deploy:Boolean(form.value.auto_deploy),registry_id:form.value.registry_id || defaultReleaseRegistry(registries.value),repository:form.value.repository?.trim() || '',tag_pattern:form.value.tag_pattern || '*',parameters:environmentParameters(),allowed_publishers:JSON.stringify(publishers.value),auto_rollback:Boolean(form.value.stateless && form.value.auto_rollback),script_mode:form.value.script_mode,scripts:form.value.script_mode==='override'?copyScripts(form.value.scripts):emptyScripts(),container_names:form.value.container_names || [],replace_containers:Boolean(form.value.replace_containers)};saving.value=true;await api('PUT',`/applications/${id}/environments`,payload);visible.value=false;ElMessage.success('环境配置已保存');await load();if(route.query.setup){const query={...route.query};delete query.setup;await router.replace({query})}}catch(cause){formError.value=errorText(cause)}finally{saving.value=false}}
function openScripts(){scriptDraft.value=copyScripts(data.value.application?.scripts);scriptsError.value='';scriptsVisible.value=true}
async function saveScripts(){if(scriptsSaving.value)return;scriptsSaving.value=true;scriptsError.value='';try{await api('PUT',`/applications/${id}/scripts`,{scripts:scriptDraft.value});scriptsVisible.value=false;ElMessage.success('应用脚本已保存，新版本将使用此配置');await load()}catch(cause){scriptsError.value=errorText(cause)}finally{scriptsSaving.value=false}}
function environmentEndpoints(env:any){return templateEndpoints(data.value.application.definition,env.parameters || {},env.target_id>1?'<目标主机地址>':window.location.hostname)}
async function copyEndpoint(value:string){try{await navigator.clipboard.writeText(value);ElMessage.success('地址已复制')}catch{actionError.value='地址复制失败，请手动复制'}}
function openSaveTemplate(){templateForm.value={name:data.value.application.name,version:'1.0.0',description:data.value.application.description || ''};templateError.value='';saveTemplateVisible.value=true}
async function saveAsTemplate(){if(templateSaving.value)return;templateError.value='';if(!templateForm.value.name.trim() || !templateForm.value.version.trim()){templateError.value='请填写模板名称和版本';return}templateSaving.value=true;try{await applicationTemplatesApi.save(String(id),templateForm.value);saveTemplateVisible.value=false;ElMessage.success('应用模板已保存，可在应用模板列表中复用')}catch(cause){templateError.value=errorText(cause)}finally{templateSaving.value=false}}
async function publishBundle(row:any){if(busy.value)return;publishing.value=row.id;actionError.value='';try{await request.post('/bundles/publish',{release_id:row.id},{timeout:0,silentError:true});ElMessage.success('版本镜像已入库');await load()}catch(cause){actionError.value=errorText(cause)}finally{publishing.value=0}}
onMounted(loadOptions);onBeforeUnmount(()=>{disposed=true;controller.abort()})
</script>
<style scoped>
.application-runtime-image{display:flex;flex-direction:column;gap:4px;min-width:0;font-size:12px}.application-runtime-image>span,.application-runtime-image>a{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.application-runtime-image>code{font-size:11px;color:#999}
.application-runtime-image>a,.application-runtime-image-link{color:#35acd7;text-decoration:none}.application-runtime-image>a:hover,.application-runtime-image-link:hover{text-decoration:underline}.application-runtime-image>a:focus-visible,.application-runtime-image-link:focus-visible{outline:2px solid #35acd7;outline-offset:2px}
.application-health-state{box-sizing:border-box;max-width:100%;white-space:normal;overflow-wrap:anywhere;vertical-align:middle;line-height:18px;padding-block:2px}.application-health-state::before{flex-shrink:0}
.application-version-section{display:flex;flex-direction:column;flex:1;min-height:0;gap:12px;overflow:hidden}.application-version-section>.management-description{margin:0;flex-shrink:0}
.application-detail-editor :deep(.workspace-editor-page) { max-width:none; }
.application-detail-editor :deep(.workspace-editor-heading) { padding-bottom:10px; }
.application-detail-editor :deep(.workspace-editor-header-content) { gap:10px; }
.application-detail-editor :deep(.workspace-editor-navigation) { padding-bottom:0; }
.application-detail-editor--list :deep(.workspace-editor-scroll) { display:flex; flex-direction:column; overflow:hidden; }
.application-detail-page { display:flex; flex-direction:column; min-height:0; min-width:0; }
.application-detail-editor--list .application-detail-page { flex:1; overflow:hidden; }
.application-detail-page > .resource-list { flex:1; min-height:0; overflow:hidden; }
.application-detail-page :deep(.resource-list-table) { min-height:0; overflow:hidden; }
.application-detail-tabs { display:flex; gap:20px; min-width:0; border-bottom:1px solid #303030; }
.application-detail-tabs button { display:inline-flex; align-items:center; gap:6px; padding:8px 0; border:0; border-bottom:2px solid transparent; background:none; color:#aaa; font:inherit; font-size:13px; cursor:pointer; white-space:nowrap; }
.application-detail-tabs button[aria-current=page] { color:#35acd7; border-bottom-color:#35acd7; }
.application-detail-tabs button:disabled { cursor:default; opacity:.5; }
.application-detail-tabs button:focus-visible { outline:2px solid #35acd7; outline-offset:2px; }
.application-detail-tabs button span { font-size:11px; color:#888; }
.application-setup-guide { display:flex; align-items:center; justify-content:space-between; gap:8px 20px; flex-wrap:wrap; padding-bottom:6px; }
.application-setup-next { display:flex; align-items:center; flex-wrap:wrap; gap:4px 12px; color:#999; font-size:12px; min-width:0; }
.application-setup-next .el-button { margin:0; font-size:12px; }
.application-configuration { max-width:1100px; width:100%; margin-inline:auto; }
.application-configuration > .management-description { margin:0; overflow-wrap:anywhere; }
.application-endpoint { margin:10px 0; font-size:12px; overflow-wrap:anywhere; }.application-endpoint code { min-width:0; overflow-wrap:anywhere; }
.application-instructions p { margin:0; font-size:12px; color:#aaa; line-height:1.8; }.application-instructions p + p { margin-top:10px; }
.application-environment-form .environment-config-section { min-width:0; margin:0 0 24px; }
.application-environment-form .environment-config-section:last-child { margin-bottom:0; }
.application-environment-form h3 { margin:0 0 16px; font-size:14px; font-weight:500; }
.environment-config-groups { --el-collapse-header-text-color:#d6d6d6; --el-collapse-content-text-color:#d6d6d6; }
.environment-config-groups :deep(.el-collapse-item__header) { min-height:48px; height:auto; padding:10px 0; font-size:13px; font-weight:500; line-height:20px; }
.environment-config-groups :deep(.el-collapse-item__content) { padding:8px 0 20px; }
.environment-config-title { display:flex; align-items:center; flex-wrap:wrap; gap:8px; text-align:left; }
.environment-config-summary { color:#999; font-size:12px; font-weight:400; }
.environment-recovery-options { display:flex; flex-wrap:wrap; gap:4px 20px; }
.environment-recovery-options .el-checkbox { margin:0; }
.application-environment-form :deep(.el-form-item__content) { display:block; }
.application-environment-form :deep(.el-switch) { max-width:100%; height:auto; min-height:32px; }
.application-environment-form :deep(.el-switch__label) { height:auto; white-space:normal; }
@media(max-width:760px) {
  .application-detail-tabs { gap:18px; }
  .application-detail-editor :deep(.workspace-editor-heading) { flex-wrap:nowrap; gap:8px; }
  .application-detail-editor :deep(.workspace-editor-heading .workspace-heading-title) { flex:1 1 0; gap:6px; }
  .application-detail-editor :deep(.workspace-heading-title h1) { font-size:16px; line-height:24px; }
  .application-detail-editor :deep(.workspace-editor-heading .management-heading-actions) { flex-wrap:nowrap; flex-shrink:0; width:auto; margin-left:auto; gap:4px; }
  .application-detail-page :deep(.management-toolbar) { display:grid; grid-template-columns:minmax(0,1fr) auto; gap:8px; padding:10px 12px; }
  .application-detail-page :deep(.management-toolbar .management-actions) { display:contents; }
  .application-detail-page :deep(.management-toolbar .management-search) { grid-column:1/-1; grid-row:2; width:100%; }
  .application-detail-page :deep(.management-toolbar .management-actions > .el-button) { grid-column:2; grid-row:1; }
  .application-detail-page :deep(.application-environment-list .management-toolbar) { grid-template-columns:minmax(0,1fr) auto auto; column-gap:6px; }
  .application-detail-page :deep(.application-environment-list .management-actions > .el-button:last-child) { grid-column:3; }
  .application-detail-page :deep(.management-filters) { padding:0 12px 12px; }
  .application-detail-page :deep(.management-pagination) { gap:6px; padding:8px 12px; }
  .application-detail-page :deep(.management-pagination > span) { width:auto; flex:1 1 auto; }
  .application-detail-page :deep(.el-pagination) { --el-pagination-button-width:22px; --el-pagination-button-height:28px; --el-pagination-font-size:12px; --el-pagination-item-gap:0; min-width:0; }
  .application-detail-page :deep(.management-pagination .el-select) { width:96px; }
  .application-detail-page :deep(.el-pagination__sizes) { margin-right:6px; }
}
</style>
