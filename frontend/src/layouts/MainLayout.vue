<template>
  <div class="workspace" :class="{ collapsed }">
    <aside class="workspace-sidebar">
      <router-link
        class="brand"
        :to="auth.user?.is_admin ? '/overview' : '/builtin'"
        ><span class="brand-mark">D</span
        ><span v-if="!collapsed"
          ><strong>Dockerun</strong><small>轻量部署工作台</small></span
        ></router-link
      >
      <nav aria-label="主导航">
        <template v-for="group in groups" :key="group.title"
          ><p v-if="!collapsed" class="nav-group">{{ group.title }}</p>
          <router-link
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            :class="{ 'router-link-active': (item.path === '/images' && route.name === 'ImageDetail') || (item.path === '/containers' && route.path.startsWith('/containers/')) || (item.path === '/networks' && route.path.startsWith('/networks/')) || (item.path === '/volumes' && route.path.startsWith('/volumes/')) || (item.path === '/registry' && route.path.startsWith('/registry/')) }"
            :title="item.title"
            ><el-icon><PortainerIcon v-if="item.path === '/images'" name="list" /><PortainerIcon v-else-if="item.path === '/networks'" name="network" /><PortainerIcon v-else-if="item.path === '/volumes'" name="volume" /><PortainerIcon v-else-if="item.path === '/registry'" name="registry" /><component v-else :is="item.icon" /></el-icon
            ><span v-if="!collapsed">{{ item.title }}</span></router-link
          ></template
        >
      </nav>
      <div class="sidebar-bottom">
        <el-icon><Monitor /></el-icon
        ><span v-if="!collapsed">{{
          auth.user?.is_admin ? "本机 · 独立模式" : "镜像仓库工作区"
        }}</span>
      </div>
    </aside>
    <div class="workspace-body">
      <header class="workspace-header">
        <el-button
          text
          circle
          aria-label="折叠或展开导航"
          @click="collapsed = !collapsed"
          ><el-icon
            ><Fold v-if="!collapsed" /><Expand v-else /></el-icon></el-button
        >
        <nav class="workspace-breadcrumbs" aria-label="页面导航">
          <ol class="workspace-breadcrumb-list">
            <li v-for="(crumb, index) in breadcrumbs" :key="index">
              <router-link v-if="crumb.to" :to="crumb.to">{{ crumb.label }}</router-link>
              <span v-else :aria-current="index === breadcrumbs.length - 1 ? 'page' : undefined">{{ crumb.label }}</span>
            </li>
          </ol>
        </nav>
        <div class="header-account">
          <el-tag size="small" effect="plain">{{
            auth.user?.is_admin ? "管理员" : "普通用户"
          }}</el-tag
          ><router-link to="/account"
            ><el-icon><User /></el-icon> {{ auth.user?.username }}</router-link
          ><el-button text @click="logout">退出</el-button>
        </div>
      </header>
      <main class="workspace-main" :class="{ 'workspace-main--maintenance': route.name === 'Settings', 'workspace-main--registries': route.path === '/builtin' || route.path === '/registry' || route.path.startsWith('/registry/'), 'workspace-main--images': route.name === 'Images', 'workspace-main--containers': route.name === 'Containers', 'workspace-main--users': route.name === 'Users', 'workspace-main--management-list': ['Applications', 'ApplicationTemplates', 'Deployments', 'Bundles', 'Hosts', 'DeployScripts', 'WebhookConfigs'].includes(String(route.name)), 'workspace-main--editor': route.meta.workspaceEditor === true, 'workspace-main--networks': route.name === 'Networks', 'workspace-main--network-detail': route.path.startsWith('/networks/'), 'workspace-main--volumes': route.name === 'Volumes', 'workspace-main--volume-detail': route.path.startsWith('/volumes/'), 'workspace-main--image-detail': route.name === 'ImageDetail', 'workspace-main--container-detail': route.path.startsWith('/containers/') }"><router-view :key="route.path" /></main>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import PortainerIcon from "@/components/PortainerIcon.vue";
import { workspaceBreadcrumbs } from "@/utils/navigation";
const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const collapsed = ref(window.innerWidth <= 1100);
const breadcrumbs = computed(() => workspaceBreadcrumbs(route, Boolean(auth.user?.is_admin)));
const groups = computed(() =>
  auth.user?.is_admin
    ? [
        {
          title: "发布工作区",
          items: [
            { path: "/overview", title: "概览", icon: "Odometer" },
            { path: "/applications", title: "应用", icon: "Grid" },
            { path: "/application-templates", title: "应用模板", icon: "Collection" },
            { path: "/deployments", title: "部署记录", icon: "List" },
            { path: "/bundles", title: "发布包", icon: "Box" },
          ],
        },
        {
          title: "镜像与资源",
          items: [
            { path: "/builtin", title: "内置仓库", icon: "Collection" },
            { path: "/registry", title: "外部仓库", icon: "Connection" },
            { path: "/images", title: "本地镜像", icon: "List" },
            { path: "/containers", title: "容器", icon: "Box" },
            { path: "/networks", title: "网络", icon: "Share" },
            { path: "/volumes", title: "卷", icon: "FolderOpened" },
          ],
        },
        {
          title: "系统管理",
          items: [
            { path: "/users", title: "用户管理", icon: "User" },
            { path: "/hosts", title: "目标主机", icon: "Monitor" },
            { path: "/settings", title: "仓库与维护", icon: "Tools" },
            { path: "/webhooks", title: "外部回调", icon: "Link" },
            { path: "/system", title: "系统信息", icon: "Setting" },
            { path: "/deploy-scripts", title: "历史脚本", icon: "Document" },
            { path: "/account", title: "个人账号", icon: "UserFilled" },
          ],
        },
      ]
    : [
        {
          title: "镜像工作区",
          items: [
            { path: "/builtin", title: "内置仓库", icon: "Collection" },
            { path: "/account", title: "个人账号", icon: "User" },
          ],
        },
      ],
);
async function logout() {
  const failure = await router.push("/login");
  if (!failure) auth.logout();
}
</script>

<style scoped>
.workspace-breadcrumbs { flex: 1; min-width: 0; overflow-x: auto; padding-block: 4px; }
.workspace-breadcrumb-list { display: flex; align-items: center; width: max-content; margin: 0; padding: 0; list-style: none; font-size: 13px; line-height: 20px; white-space: nowrap; }
.workspace-breadcrumb-list li { display: flex; align-items: center; gap: 10px; }
.workspace-breadcrumb-list li + li { margin-left: 10px; }
.workspace-breadcrumb-list li + li::before { content: '/'; color: #747980; }
.workspace-breadcrumb-list a { color: #d6d6d6; }
.workspace-breadcrumb-list a:hover { color: #35acd7; }
.workspace-breadcrumb-list a:focus-visible { outline: 2px solid #35acd7; outline-offset: 2px; }
.workspace-breadcrumb-list [aria-current="page"] { color: #aaa; }
.header-account { flex-shrink: 0; }
@media (max-width: 700px) {
  .workspace-header { flex-wrap: wrap; height: auto; min-height: 64px; padding: 8px 12px; gap: 6px 10px; }
  .workspace-breadcrumbs { flex: 1 0 100%; order: 3; }
  .workspace-breadcrumb-list { font-size: 12px; }
}
.workspace-main { display: flex; flex-direction: column; flex: 1; min-height: 0; min-width: 0; overflow: auto; padding: 18px 20px; background: #151515; }
.workspace-main--management-list { overflow: hidden; }
@media(max-width:760px) { .workspace-main { padding: 12px; } }
.workspace-main--maintenance, .workspace-main--registries, .workspace-main--images, .workspace-main--users, .workspace-main--containers, .workspace-main--networks, .workspace-main--network-detail, .workspace-main--volumes, .workspace-main--volume-detail, .workspace-main--image-detail, .workspace-main--container-detail {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 18px 20px;
  background: #151515;
}
.workspace-main--maintenance, .workspace-main--image-detail, .workspace-main--container-detail, .workspace-main--network-detail, .workspace-main--volume-detail { overflow: auto; }
.workspace-main--editor { overflow: hidden; }
@media (max-width: 760px) {
  .workspace-main--maintenance, .workspace-main--registries, .workspace-main--images, .workspace-main--users, .workspace-main--containers, .workspace-main--networks, .workspace-main--network-detail, .workspace-main--volumes, .workspace-main--volume-detail, .workspace-main--image-detail, .workspace-main--container-detail { padding: 12px; }
}
</style>
