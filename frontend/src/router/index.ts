import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue')
    },
    {
      path: '/setup',
      name: 'Setup',
      component: () => import('@/views/Setup.vue')
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {path:'overview',name:'Overview',component:()=>import('@/views/Overview.vue'),meta:{title:'概览',admin:true}},
        {path:'applications',name:'Applications',component:()=>import('@/views/Applications.vue'),meta:{title:'应用',admin:true}},
        {path:'applications/:id',name:'ApplicationDetail',component:()=>import('@/views/ApplicationDetail.vue'),meta:{title:'应用详情',admin:true}},
        {path:'applications/:id/deploy',name:'DeploymentWizard',component:()=>import('@/views/DeploymentWizard.vue'),meta:{title:'发布应用',admin:true}},
        {path:'deployments',name:'Deployments',component:()=>import('@/views/Deployments.vue'),meta:{title:'部署记录',admin:true}},
        {path:'deployments/:id',name:'DeploymentDetail',component:()=>import('@/views/DeploymentDetail.vue'),meta:{title:'部署详情',admin:true}},
        {path:'bundles',name:'Bundles',component:()=>import('@/views/Bundles.vue'),meta:{title:'发布包',admin:true}},
        {path:'builtin',name:'Builtin',component:()=>import('@/views/Builtin.vue'),meta:{title:'内置仓库',admin:false}},
        {path:'users',name:'Users',component:()=>import('@/views/Users.vue'),meta:{title:'用户管理',admin:true}},
        {path:'account',name:'Account',component:()=>import('@/views/Account.vue'),meta:{title:'个人账号',admin:false}},
        {path:'hosts',name:'Hosts',component:()=>import('@/views/Hosts.vue'),meta:{title:'目标主机',admin:true}},
        {path:'settings',name:'Settings',component:()=>import('@/views/Settings.vue'),meta:{title:'仓库与维护',admin:true}},
        {
          path: '',
          redirect: '/overview'
        },
        {
          path: 'containers',
          name: 'Containers',
          component: () => import('@/views/Containers.vue')
        },
        {
          path: 'images',
          name: 'Images',
          component: () => import('@/views/Images.vue'),
          meta: { title: '本地镜像', admin: true }
        },
        {
          path: 'images/:id',
          name: 'ImageDetail',
          component: () => import('@/views/ImageDetail.vue'),
          meta: { title: '镜像详情', admin: true }
        },
        {
          path: 'networks',
          name: 'Networks',
          component: () => import('@/views/Networks.vue')
        },
        {
          path: 'volumes',
          name: 'Volumes',
          component: () => import('@/views/Volumes.vue')
        },
        {
          path: 'deploy-scripts',
          name: 'DeployScripts',
          component: () => import('@/views/DeployScripts.vue')
        },
        {
          path: 'registry',
          name: 'Registry',
          component: () => import('@/views/Registry.vue')
        },
        {
          path: 'registry/:id/images',
          name: 'RegistryImages',
          component: () => import('@/views/RegistryImages.vue')
        },
        {
          path: 'webhooks',
          name: 'WebhookConfigs',
          component: () => import('@/views/WebhookConfigs.vue')
        },
        {
          path: 'system',
          name: 'System',
          component: () => import('@/views/System.vue')
        }
      ]
    }
  ]
})

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta.requiresAuth && authStore.isAuthenticated && !authStore.user) await authStore.fetchCurrentUser()
  const adminRoute = to.meta.requiresAuth && to.meta.admin !== false
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (adminRoute && !authStore.user?.is_admin) {
    next("/builtin")
  } else {
    next()
  }
})

export default router

