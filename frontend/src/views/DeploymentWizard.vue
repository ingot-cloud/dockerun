<template>
  <WorkspaceEditor :title="`发布 ${data.application?.name || '应用'}`" :loading="loading" :busy="busy" :reset-key="step" @refresh="loadOptions">
    <template #actions>
      <el-button :disabled="loading || step === 0 || busy" @click="step--">上一步</el-button>
      <el-button v-if="step < 2" type="primary" :disabled="loading || busy || !data.application || (step === 1 && !environment)" @click="step++">下一步</el-button>
      <el-button v-if="step === 2" type="primary" :disabled="loading" :loading="busy" @click="preflight">准备版本并预检查</el-button>
      <el-button v-if="step === 3" type="primary" :disabled="loading" :loading="busy" @click="deploy">确认并部署</el-button>
    </template>
    <template #navigation><el-steps :active="step" finish-status="success"><el-step title="选择来源" /><el-step title="应用与环境" /><el-step title="参数配置" /><el-step title="预检查与差异" /></el-steps></template>
    <template v-if="error" #feedback><el-alert :title="error" type="error" class="inline-error" :closable="false" /></template>
  <div v-loading="loading" class="workspace-page workspace-editor-page form-page">
    <p class="management-description">选择交付材料，确认部署环境和版本差异后再执行部署。</p>
    <div v-if="!loading && !data.environments?.length" class="management-warning" role="status"><span>首次发布需要先添加环境，配置目标主机、环境参数和脚本。</span><router-link :to="{path:`/applications/${id}`,query:route.query}">前往配置环境</router-link></div>
    <p v-else-if="!selectedEnv?.current_release_id" class="management-form-help">首次发布请完整配置镜像、端口、挂载及配置文件。首次成功后，可回到环境配置开启自动发布。</p>
    <template v-if="step === 0"
      ><div class="source-grid">
        <button
          v-for="s in sources"
          :key="s.value"
          class="source-card" :disabled="busy || loading"
          :aria-pressed="form.source === s.value"
          @click="selectSource(s.value)"
        >
          <strong>{{ s.title }}</strong
          ><span class="muted">{{ s.description }}</span>
        </button>
      </div>
      <section class="panel">
        <h2>使用已验证版本</h2>
        <el-select
          v-model="existing"
          placeholder="选择已有版本进行晋升或恢复"
          clearable
          style="width: 100%"
          ><el-option
            v-for="r in data.releases || []"
            :key="r.id"
            :value="r.id"
            :label="`${r.version} · ${sourceLabel(r.source)}`"
        /></el-select></section
    ></template>
    <section v-if="step === 1" class="panel">
      <el-form label-position="top" :disabled="busy || loading"
        ><el-form-item label="部署环境"
          ><el-select v-model="environment" style="width: 100%"
            ><el-option
              v-for="e in data.environments || []"
              :key="e.id"
              :value="e.id"
              :label="`${e.name} · ${e.target_id > 1 ? '主机 #' + e.target_id : '本机'}`" /></el-select></el-form-item
        ><el-alert
          v-if="!data.environments?.length"
          title="请先在应用详情中添加环境"
          type="warning" /><template v-if="!existing"
          ><el-form-item v-if="form.source === 'registry'" label="仓库"
            ><el-select v-model="form.registry_id"
              ><el-option
                v-for="r in registries"
                :key="r.id"
                :value="r.id"
                :label="r.name" /></el-select><p class="management-form-help">{{ selectedRegistry?.kind==='builtin' ? '默认使用内置仓库，镜像推送事件由系统直接接收，无需配置 Webhook。' : '外部仓库自动发布需要接入推送回调；手动发布只需镜像可拉取。' }}</p></el-form-item
          ><el-form-item v-if="form.source === 'bundle'" label="完整发布包"
            ><el-select v-model="bundleId"
              ><el-option
                v-for="b in bundles.filter((b) => b.status === 'ready' && b.kind !== 'image')"
                :key="b.id"
                :value="b.id"
                :label="`${b.name} · #${b.id}`" /></el-select
            ><router-link to="/bundles" style="margin-left: 12px"
              >导入发布包</router-link
            ></el-form-item
          ><el-form-item v-else label="版本名称"
            ><el-input
              v-model="form.version"
              placeholder="v1.0.0" /></el-form-item></template
      ></el-form>
    </section>
    <section v-if="step === 2" class="panel">
      <template v-if="existing || form.source === 'bundle'"
        ><h2>
          {{ restoreConfiguration ? "恢复历史成功配置" : "保留目标环境参数" }}
        </h2>
        <el-checkbox v-if="existing" v-model="restoreConfiguration"
          >恢复此版本在所选环境上次成功时的参数</el-checkbox
        >
        <p class="muted">
          镜像版本保持原有身份；目标环境的端口、地址及参数不会被包内默认值覆盖。
        </p>
        <pre class="code-block">{{
          maskTemplateParameters(selectedEnv?.parameters || {}, data.application?.definition?.parameters || [])
        }}</pre></template
      ><template v-else
        ><el-form label-position="top" :disabled="busy || loading"
          ><el-form-item label="定义方式"
            ><el-radio-group v-model="definitionMode"
              ><el-radio-button value="simple">简单容器</el-radio-button
              ><el-radio-button value="compose"
                >应用栈（Compose）</el-radio-button
              ></el-radio-group
            ></el-form-item
          ><template v-if="definitionMode === 'simple'"
            ><el-form-item label="镜像引用"
              ><el-input
                v-model="simple.image"
                :placeholder="`${registryHost(selectedRegistry) || 'registry.example.com'}/team/app:v1`" /><p class="management-form-help">填写包含仓库地址、镜像名称和标签的完整引用。请先将镜像推送到所选仓库。</p></el-form-item
            ><el-form-item label="端口映射（可选：宿主端口:容器端口）"
              ><el-input
                v-model="simple.port"
                placeholder="8081:80" /></el-form-item
            ><el-form-item label="容器环境变量（JSON，可引用目标环境参数）"
              ><el-input
                v-model="simple.environment"
                type="textarea"
                :rows="3" /></el-form-item></template
          ><template v-if="definitionMode === 'compose'"
            ><el-form-item label="Compose 定义"
              ><CodeEditor
                v-model="form.compose"
                label="Compose 定义"
                language="yaml"
                height="360px" /></el-form-item></template
          ><el-form-item label="目标平台（可选）"
            ><el-select v-model="form.platform" clearable
              ><el-option label="Linux amd64" value="linux/amd64" /><el-option
                label="Linux arm64"
                value="linux/arm64" /></el-select></el-form-item
          ><el-form-item label="非敏感配置文件（JSON：configs/文件名 → 内容）"
            ><el-input
              v-model="form.configs"
              type="textarea"
              :rows="3" /></el-form-item></el-form
        ><el-alert
          :title="
            form.source === 'local'
              ? '仅使用目标 Docker 已有镜像；缺失直接失败。'
              : '创建版本时锁定仓库镜像摘要，部署时拉取同一身份的镜像。'
          "
          type="info"
          :closable="false"
      /></template>
    </section>
    <section v-if="step === 2" class="panel"><h2>本次部署脚本</h2><p class="management-description">{{selectedEnv?.script_mode==='override'?'使用当前环境的独立脚本。':selectedEnv?.script_mode==='disabled'?'此环境已停用脚本，由系统启动 Compose。':'沿用发布版本保存的应用脚本。新版本使用当前应用脚本。'}} 脚本在确认部署后执行，可在应用详情或环境配置中修改。</p><el-collapse v-if="hasScripts(plannedScripts)"><el-collapse-item title="查看部署前、部署与部署后脚本" name="scripts"><DeploymentScriptEditor :model-value="plannedScripts" disabled /></el-collapse-item></el-collapse><p v-else class="management-form-help">未配置脚本，由系统准备镜像、检查目标主机并启动 Compose。</p></section>
    <section v-if="step === 3" class="panel">
      <h2>预检查结果</h2>
      <el-alert
        v-if="preview"
        :title="preview.message"
        type="success"
        :closable="false"
      />
      <p class="muted" style="margin: 14px 0">
        环境 {{ preview?.environment }} · 目标
        {{
          preview?.target_id > 1 ? "#" + preview?.target_id : "本机"
        }}。普通替换允许短暂停机。
      </p>
      <h2>新版本镜像</h2>
      <pre class="code-block">{{ preview?.images }}</pre>
      <el-collapse
        ><el-collapse-item title="新应用栈定义" name="new">
          <pre class="code-block">{{ preview?.compose }}</pre></el-collapse-item
        ><el-collapse-item title="当前应用栈定义" name="old">
          <pre class="code-block">{{
            preview?.previous_compose || "首次部署"
          }}</pre>
        </el-collapse-item></el-collapse
      >
      <el-collapse v-if="hasScripts(preview?.scripts)"><el-collapse-item title="确认本次执行的脚本快照" name="scripts"><DeploymentScriptEditor :model-value="copyScripts(preview?.scripts)" disabled /></el-collapse-item></el-collapse>
    </section>
  </div>
  </WorkspaceEditor>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, defineAsyncComponent } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, errorText } from "@/api/workspace";
import WorkspaceEditor from '@/components/WorkspaceEditor.vue';
import { parseObject, sourceLabel } from '@/utils/management';
import DeploymentScriptEditor from '@/components/DeploymentScriptEditor.vue';
import { copyScripts, defaultReleaseRegistry, emptyScripts, hasScripts, registryHost } from '@/utils/releases';
import { maskTemplateParameters } from '@/utils/applicationTemplates';
import { useUnsavedChanges } from '@/composables/useUnsavedChanges';
const loading = ref(false);
let preparedSignature = '';
let executionKey = '';
const optionsController=new AbortController();
let disposed=false;
const defaultsLoaded=ref(false);
const CodeEditor = defineAsyncComponent(
  () => import("@/components/CodeEditor.vue"),
);
const route = useRoute(),
  router = useRouter(),
  id = route.params.id,
  step = ref(0),
  error = ref(""),
  busy = ref(false),
  data = ref<any>({}),
  environment = ref<number>(),
  registries = ref<any[]>([]),
  bundles = ref<any[]>([]),
  bundleId = ref<number>(),
  existing = ref<number | undefined>(Number(route.query.release) || undefined),
  preview = ref<any>(null),
  releaseId = ref<number>(0);
const form = ref({
  source: "registry",
  registry_id: 0,
  version: "",
  platform: "",
  compose:
    'services:\n  app:\n    image: your-registry/team/app:v1\n    ports:\n      - "8081:80"\n',
  configs: "{}",
});
const definitionMode = ref("simple");
const simple = ref({ image: "", port: "", environment: "{}" });
function selectSource(source: string) {
  form.value.source = source;
  existing.value = undefined;
}
const restoreConfiguration = ref(false);
const { markSaved } = useUnsavedChanges(() => ({ form: form.value, environment: environment.value, bundle: bundleId.value, existing: existing.value, definitionMode: definitionMode.value, simple: simple.value, restore: restoreConfiguration.value }), { enabled: () => defaultsLoaded.value, busy: () => busy.value });
const sources = [
  {
    value: "registry",
    title: "仓库镜像",
    description: "内置或外部镜像仓库，在线交付。",
  },
  {
    value: "local",
    title: "本机镜像",
    description: "使用已通过 docker load 导入的镜像。",
  },
  {
    value: "bundle",
    title: "完整发布包",
    description: "镜像、定义和配置齐备，支持断网部署。",
  },
];
const selectedEnv = computed(() =>
  data.value.environments?.find((e: any) => e.id === environment.value),
);
const selectedRegistry = computed(() => registries.value.find(item => item.id === form.value.registry_id));
const plannedScripts = computed(() => {
  if(selectedEnv.value?.script_mode === 'disabled') return emptyScripts();
  if(selectedEnv.value?.script_mode === 'override') return copyScripts(selectedEnv.value.scripts);
  const version = data.value.releases?.find((item:any) => item.id === existing.value);
  return copyScripts(existing.value ? version?.scripts : data.value.application?.scripts);
});
async function preflight() {
  if (busy.value || loading.value) return;
  busy.value = true;
  error.value = "";
  try {
    if (!environment.value) throw new Error('请选择部署环境');
    if(!existing.value && form.value.source!=='bundle')parseObject(form.value.configs, '非敏感配置文件');
    const signature = JSON.stringify({existing:existing.value,bundle:bundleId.value,environment:environment.value,parameters:selectedEnv.value?.parameters,form:form.value,definitionMode:definitionMode.value,simple:simple.value,scripts:plannedScripts.value});
    if (preparedSignature === signature && releaseId.value) {
      // 预检查失败重试复用已准备版本，避免重复创建不可变版本。
    } else if (existing.value) {
      releaseId.value = existing.value;
    } else if (form.value.source === "bundle") {
      if (!bundleId.value) throw new Error("请选择发布包");
      const r = await api("POST", `/bundles/${bundleId.value}/version`, {
        application_id: Number(id),
      });
      releaseId.value = r.release.id;
    } else {
      let definition = form.value.compose;
      if (definitionMode.value === "simple") {
        if (!simple.value.image) throw new Error("请填写镜像引用");
        const service: any = {
          image: simple.value.image,
          restart: "unless-stopped",
          environment: parseObject(simple.value.environment, '容器环境变量'),
        };
        if (simple.value.port) service.ports = [simple.value.port];
        definition = JSON.stringify({ services: { app: service } }, null, 2);
      }
      const r = await api(
        "POST",
        `/applications/${id}/releases?environment_id=${environment.value}`,
        { ...form.value, compose: definition,save_definition:true },
      );
      releaseId.value = r.id;
    }
    preparedSignature = signature;
    preview.value = await api("POST", "/deployments/preflight", {
      environment_id: environment.value,
      release_id: releaseId.value,
      restore_configuration: restoreConfiguration.value,
    });
    executionKey = crypto.randomUUID();
    step.value = 3;
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
async function deploy() {
  if (busy.value || !preview.value || !releaseId.value) return;
  busy.value = true;
  error.value = "";
  try {
    const d = await api("POST", "/deployments", {
      environment_id: environment.value,
      release_id: releaseId.value,
      restore_configuration: restoreConfiguration.value,
      idempotency_key: executionKey,
    });
    markSaved();
    busy.value = false;
    await router.replace(`/deployments/${d.id}`);
  } catch (e) {
    error.value = errorText(e);
  } finally {
    busy.value = false;
  }
}
async function loadOptions() {
  if (loading.value || busy.value) return;
  loading.value = true; error.value = '';
  try {
    const results = await Promise.all([
      api('GET', `/applications/${id}`,undefined,optionsController.signal), api<any[]>('GET','/registries',undefined,optionsController.signal), api<any[]>('GET','/bundles',undefined,optionsController.signal),
    ]);
    if(disposed)return;
    data.value = results[0]; registries.value = results[1] || []; bundles.value = results[2] || [];
    if (!data.value.environments?.some((item:any)=>item.id===environment.value)) environment.value=data.value.environments?.find((item:any)=>item.id===Number(route.query.environment))?.id || data.value.environments?.[0]?.id;
    if (!form.value.registry_id) form.value.registry_id=defaultReleaseRegistry(registries.value);
    if(!defaultsLoaded.value){const definition=data.value.application?.definition;if(definition?.compose && !existing.value){form.value.compose=definition.compose;form.value.configs=JSON.stringify(definition.configs || {},null,2);form.value.registry_id=selectedEnv.value?.registry_id || defaultReleaseRegistry(registries.value);definitionMode.value='compose';if(route.query.prepare==='template')step.value=2}defaultsLoaded.value=true;markSaved()}
  } catch(cause) { if(!disposed)error.value=errorText(cause) }
  finally { loading.value=false }
}
onMounted(loadOptions);
onBeforeUnmount(()=>{disposed=true;optionsController.abort()});
</script>
