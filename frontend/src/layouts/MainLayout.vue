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
            :class="{ 'router-link-active': (item.path === '/images' && route.name === 'ImageDetail') || (item.path === '/containers' && route.path.startsWith('/containers/')) }"
            :title="item.title"
            ><el-icon><PortainerIcon v-if="item.path === '/images'" name="list" /><component v-else :is="item.icon" /></el-icon
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
        ><el-breadcrumb separator="/"
          ><el-breadcrumb-item>Dockerun</el-breadcrumb-item
          ><el-breadcrumb-item v-if="route.name === 'ImageDetail'" :to="{ name: 'Images', query: route.query }">本地镜像</el-breadcrumb-item
          ><el-breadcrumb-item v-if="route.path.startsWith('/containers/')" :to="{ name: 'Containers', query: route.query }">容器</el-breadcrumb-item
          ><el-breadcrumb-item>{{
            route.meta.title || route.name
          }}</el-breadcrumb-item></el-breadcrumb
        >
        <div class="header-account">
          <el-tag size="small" effect="plain">{{
            auth.user?.is_admin ? "管理员" : "普通用户"
          }}</el-tag
          ><router-link to="/account"
            ><el-icon><User /></el-icon> {{ auth.user?.username }}</router-link
          ><el-button text @click="logout">退出</el-button>
        </div>
      </header>
      <main class="workspace-main" :class="{ 'workspace-main--images': route.name === 'Images', 'workspace-main--containers': route.name === 'Containers', 'workspace-main--image-detail': route.name === 'ImageDetail', 'workspace-main--container-detail': route.path.startsWith('/containers/') }"><router-view /></main>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import PortainerIcon from "@/components/PortainerIcon.vue";
const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const collapsed = ref(window.innerWidth <= 1100);
const groups = computed(() =>
  auth.user?.is_admin
    ? [
        {
          title: "发布工作区",
          items: [
            { path: "/overview", title: "概览", icon: "Odometer" },
            { path: "/applications", title: "应用", icon: "Grid" },
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
            { path: "/webhooks", title: "外部 Webhook", icon: "Link" },
            { path: "/system", title: "系统设置", icon: "Setting" },
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
function logout() {
  auth.logout();
  router.push("/login");
}
</script>

<style scoped>
.workspace-main--images, .workspace-main--containers, .workspace-main--image-detail, .workspace-main--container-detail {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 18px 20px;
  background: #151515;
}
.workspace-main--image-detail, .workspace-main--container-detail { overflow: auto; }
@media (max-width: 760px) {
  .workspace-main--images, .workspace-main--containers, .workspace-main--image-detail, .workspace-main--container-detail { padding: 12px; }
}
</style>
