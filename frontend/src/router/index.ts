import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import { authService } from '../services/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/dashboard'
    },
    {
      path: '/setup',
      name: 'setup',
      component: () => import('../views/SetupView.vue'),
      meta: { requiresGuest: true }
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { requiresGuest: true }
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: DashboardView,
      meta: { requiresAuth: true }
    },
    {
      path: '/proxies',
      name: 'proxies',
      component: () => import('../views/ProxiesView.vue'),
      meta: { requiresAuth: true }
    },
  ],
})

// Auth guard
router.beforeEach(async (to, from, next) => {
  try {
    // Check auth status
    const status = await authService.getStatus()
    
    // If auth is disabled, allow all routes
    if (!status.auth_enabled) {
      next()
      return
    }

    // If system needs setup, redirect to setup
    if (!status.is_setup) {
      if (to.name !== 'setup') {
        next('/setup')
        return
      }
      next()
      return
    }

    // Check if user is authenticated
    const isAuthenticated = authService.isAuthenticated()
    
    // If user needs to be authenticated but isn't
    if (to.meta.requiresAuth && !isAuthenticated) {
      // Try to verify the token
      if (authService.getToken()) {
        try {
          await authService.getCurrentUser()
          next()
          return
        } catch (error) {
          // Token is invalid, redirect to login
          next('/login')
          return
        }
      }
      next('/login')
      return
    }

    // If user is authenticated but trying to access guest-only pages
    if (to.meta.requiresGuest && isAuthenticated) {
      next('/dashboard')
      return
    }

    next()
  } catch (error) {
    console.error('Router guard error:', error)
    // On error, allow navigation but user might need to authenticate later
    next()
  }
})

export default router
