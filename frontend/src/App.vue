<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const proxyAddr = ref('')
const initError = ref('')

declare const window: any
onMounted(async () => {
  try {
    if (window['go']?.main?.App?.GetProxyAddr) {
      proxyAddr.value = await window.go.main.App.GetProxyAddr()
    }
  } catch { /* ignore */ }
})

const navItems = [
  { path: '/dashboard', label: '仪表盘', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1' },
  { path: '/accounts', label: '账号管理', icon: 'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z' },
  { path: '/groups', label: '分组管理', icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10' },
  { path: '/proxies', label: '代理管理', icon: 'M13 10V3L4 14h7v7l9-11h-7z' },
  { path: '/apikeys', label: 'API 密钥', icon: 'M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z' },
  { path: '/usage', label: '使用记录', icon: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z' },
  { path: '/pricing', label: '模型定价', icon: 'M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
  { path: '/settings', label: '系统设置', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
]

function isActive(path: string) {
  return route.path === path
}

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <aside
      class="flex-shrink-0 bg-dark-900 border-r border-dark-700 flex flex-col transition-all duration-200"
      :class="collapsed ? 'w-16' : 'w-52'"
    >
      <!-- Logo -->
      <div class="h-14 flex items-center px-4 border-b border-dark-700">
        <div class="w-7 h-7 rounded-md bg-primary-500 flex items-center justify-center text-white font-bold text-sm flex-shrink-0">
          P
        </div>
        <span v-if="!collapsed" class="ml-3 text-white font-semibold text-sm">Desktop Proxy</span>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 py-3 space-y-1 px-2">
        <button
          v-for="item in navItems"
          :key="item.path"
          @click="navigate(item.path)"
          class="w-full flex items-center px-3 py-2 rounded-lg text-sm transition-colors"
          :class="isActive(item.path)
            ? 'bg-primary-600/20 text-primary-400'
            : 'text-dark-400 hover:text-white hover:bg-dark-800'"
          :title="item.label"
        >
          <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" :d="item.icon" />
          </svg>
          <span v-if="!collapsed" class="ml-3">{{ item.label }}</span>
        </button>
      </nav>

      <!-- Proxy status -->
      <div v-if="!collapsed && proxyAddr" class="px-3 py-2 border-t border-dark-700">
        <div class="text-dark-500 text-xs mb-0.5">代理</div>
        <div class="flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-green-500"></span>
          <code class="text-dark-300 text-xs">{{ proxyAddr }}</code>
        </div>
      </div>

      <!-- Collapse toggle -->
      <div class="p-2 border-t border-dark-700">
        <button
          @click="collapsed = !collapsed"
          class="w-full flex items-center justify-center py-2 rounded-lg text-dark-500 hover:text-white hover:bg-dark-800 transition-colors"
        >
          <svg class="w-4 h-4 transition-transform" :class="collapsed ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
          </svg>
        </button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-auto bg-dark-950 p-6">
      <router-view />
    </main>
  </div>
</template>
