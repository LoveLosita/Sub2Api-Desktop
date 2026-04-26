import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
    { path: '/accounts', name: 'accounts', component: () => import('../views/Accounts.vue') },
    { path: '/groups', name: 'groups', component: () => import('../views/Groups.vue') },
    { path: '/proxies', name: 'proxies', component: () => import('../views/Proxies.vue') },
    { path: '/apikeys', name: 'apikeys', component: () => import('../views/ApiKeys.vue') },
    { path: '/usage', name: 'usage', component: () => import('../views/Usage.vue') },
    { path: '/pricing', name: 'pricing', component: () => import('../views/Pricing.vue') },
    { path: '/settings', name: 'settings', component: () => import('../views/Settings.vue') },
  ],
})

export default router
