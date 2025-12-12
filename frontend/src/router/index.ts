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
        {
          path: '',
          redirect: '/containers'
        },
        {
          path: 'containers',
          name: 'Containers',
          component: () => import('@/views/Containers.vue')
        },
        {
          path: 'images',
          name: 'Images',
          component: () => import('@/views/Images.vue')
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
          path: 'system',
          name: 'System',
          component: () => import('@/views/System.vue')
        }
      ]
    }
  ]
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})

export default router

