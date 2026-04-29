<script lang="ts" setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { GetDashboardStats } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const stats = ref<model.DashboardStats | null>(null)
const loading = ref(true)
const errorMsg = ref('')
const since = ref('')
const viewMode = ref<'total' | 'today'>(localStorage.getItem('dashboard_viewMode') === 'today' ? 'today' : 'total')
let debounceTimer: ReturnType<typeof setTimeout> | null = null

function onUsageLogged() {
  if (debounceTimer) return
  load()
  debounceTimer = setTimeout(() => { debounceTimer = null }, 500)
}

onMounted(() => {
  since.value = localStorage.getItem('dashboard_since') || ''
  load()
  EventsOn('usage:logged', onUsageLogged)
})

onUnmounted(() => {
  EventsOff('usage:logged')
  if (debounceTimer) { clearTimeout(debounceTimer); debounceTimer = null }
})

const apiSince = computed(() => {
  if (viewMode.value === 'total') return ''
  return since.value
})

async function load() {
  errorMsg.value = ''
  try {
    const result = await GetDashboardStats(apiSince.value)
    stats.value = result
    loading.value = false
  } catch (e: any) {
    console.error('加载看板失败:', e)
    errorMsg.value = String(e)
    loading.value = false
  }
}

watch(apiSince, () => load())
watch(viewMode, (v) => localStorage.setItem('dashboard_viewMode', v))

function resetDashboard() {
  since.value = toUtcString(new Date())
  localStorage.setItem('dashboard_since', since.value)
  load()
}

function clearReset() {
  since.value = ''
  localStorage.removeItem('dashboard_since')
  load()
}

function toShanghaiString(d: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function toUtcString(d: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`
}

const sinceDisplay = computed(() => {
  if (!since.value) return ''
  const d = new Date(since.value.replace(' ', 'T') + 'Z')
  return toShanghaiString(d)
})

const isReset = computed(() => since.value !== '')

const displayStats = computed(() => {
  if (!stats.value) return null
  const s = stats.value
  if (viewMode.value === 'total') {
    return {
      requests: s.total_requests,
      tokens: s.total_tokens,
      cost: s.total_cost,
      byModel: s.by_model,
      requestLabel: '总请求数',
      tokenLabel: '总 Token',
      costLabel: '总花费',
    }
  }
  if (isReset.value) {
    return {
      requests: s.total_requests,
      tokens: s.total_tokens,
      cost: s.total_cost,
      byModel: s.by_model,
      requestLabel: '区间请求数',
      tokenLabel: '区间 Token',
      costLabel: '区间花费',
    }
  }
  return {
    requests: s.today_requests,
    tokens: s.today_tokens,
    cost: s.today_cost,
    byModel: s.today_by_model,
    requestLabel: '今日请求数',
    tokenLabel: '今日 Token',
    costLabel: '今日花费',
  }
})

function formatNumber(n: number | undefined): string {
  if (n == null) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}

function formatCost(n: number | undefined): string {
  if (n == null) return '$0.00'
  return '$' + n.toFixed(4)
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">仪表盘</h1>
      <div class="flex items-center gap-3">
        <span v-if="isReset && viewMode === 'today'" class="text-dark-400 text-xs">
          统计起点: {{ sinceDisplay }}
        </span>
        <button v-if="isReset && viewMode === 'today'" @click="clearReset"
          class="px-3 py-1.5 text-dark-300 hover:text-white text-xs border border-dark-600 rounded-lg transition-colors">
          清除重置
        </button>
        <button v-if="viewMode === 'today'" @click="resetDashboard"
          class="px-3 py-1.5 bg-dark-700 hover:bg-dark-600 text-dark-200 text-xs rounded-lg transition-colors border border-dark-600">
          重置看板
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <div v-else-if="errorMsg"
      class="bg-red-900/20 border border-red-700/50 rounded-lg p-4 text-red-400 text-sm">
      {{ errorMsg }}
    </div>

    <template v-else-if="stats">
      <!-- 账号状态 (always visible, independent of view mode) -->
      <div class="grid grid-cols-4 gap-4 mb-6">
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">总账号数</div>
          <div class="text-2xl font-bold text-white mt-1">{{ stats.total_accounts ?? 0 }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">正常</div>
          <div class="text-2xl font-bold text-green-400 mt-1">{{ stats.active_accounts ?? 0 }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">异常</div>
          <div class="text-2xl font-bold text-red-400 mt-1">{{ stats.error_accounts ?? 0 }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">限速</div>
          <div class="text-2xl font-bold text-yellow-400 mt-1">{{ stats.rate_limit_accounts ?? 0 }}</div>
        </div>
      </div>

      <!-- View Mode Toggle -->
      <div class="flex items-center justify-center mb-6">
        <div class="inline-flex bg-dark-800 rounded-lg border border-dark-700 p-1">
          <button @click="viewMode = 'today'"
            class="px-8 py-2 text-sm rounded-md transition-all duration-200"
            :class="viewMode === 'today' ? 'bg-primary-600 text-white shadow-lg' : 'text-dark-400 hover:text-dark-200'">
            今日数据
          </button>
          <button @click="viewMode = 'total'"
            class="px-8 py-2 text-sm rounded-md transition-all duration-200"
            :class="viewMode === 'total' ? 'bg-primary-600 text-white shadow-lg' : 'text-dark-400 hover:text-dark-200'">
            总数据
          </button>
        </div>
      </div>

      <!-- Data Section -->
      <template v-if="displayStats">
        <div class="grid grid-cols-3 gap-4 mb-6">
          <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
            <div class="text-dark-400 text-sm">{{ displayStats.requestLabel }}</div>
            <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(displayStats.requests) }}</div>
          </div>
          <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
            <div class="text-dark-400 text-sm">{{ displayStats.tokenLabel }}</div>
            <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(displayStats.tokens) }}</div>
          </div>
          <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
            <div class="text-dark-400 text-sm">{{ displayStats.costLabel }}</div>
            <div class="text-2xl font-bold text-primary-400 mt-1">{{ formatCost(displayStats.cost) }}</div>
          </div>
        </div>

        <!-- 按模型统计费用 -->
        <div class="bg-dark-800 rounded-lg border border-dark-700">
          <div class="px-4 py-3 border-b border-dark-700">
            <h2 class="text-white font-semibold">按模型统计费用</h2>
          </div>
          <table v-if="displayStats.byModel && displayStats.byModel.length" class="w-full">
            <thead>
              <tr class="text-dark-400 text-sm border-b border-dark-700">
                <th class="text-left px-4 py-2">模型</th>
                <th class="text-right px-4 py-2">请求数</th>
                <th class="text-right px-4 py-2">Token 数</th>
                <th class="text-right px-4 py-2">费用</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in displayStats.byModel" :key="entry.model"
                class="border-b border-dark-700/50 hover:bg-dark-700/30">
                <td class="px-4 py-2 text-white text-sm font-mono">{{ entry.model }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ formatNumber(entry.requests) }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ formatNumber(entry.tokens) }}</td>
                <td class="px-4 py-2 text-primary-400 text-sm text-right">{{ formatCost(entry.cost) }}</td>
              </tr>
            </tbody>
          </table>
          <div v-else class="p-6 text-dark-500 text-sm text-center">暂无使用数据</div>
        </div>
      </template>
    </template>
  </div>
</template>
