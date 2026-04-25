<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { GetDashboardStats } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const stats = ref<model.DashboardStats | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    stats.value = await GetDashboardStats()
  } catch (e) {
    console.error('Failed to load dashboard stats:', e)
  } finally {
    loading.value = false
  }
})

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}

function formatCost(n: number): string {
  return '$' + n.toFixed(4)
}
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-white mb-6">Dashboard</h1>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <template v-else-if="stats">
      <!-- Account Status -->
      <div class="grid grid-cols-4 gap-4 mb-6">
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Total Accounts</div>
          <div class="text-2xl font-bold text-white mt-1">{{ stats.total_accounts }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Active</div>
          <div class="text-2xl font-bold text-green-400 mt-1">{{ stats.active_accounts }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Error</div>
          <div class="text-2xl font-bold text-red-400 mt-1">{{ stats.error_accounts }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Rate Limited</div>
          <div class="text-2xl font-bold text-yellow-400 mt-1">{{ stats.rate_limit_accounts }}</div>
        </div>
      </div>

      <!-- Request Stats -->
      <div class="grid grid-cols-4 gap-4 mb-6">
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Total Requests</div>
          <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(stats.total_requests) }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Today Requests</div>
          <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(stats.today_requests) }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Today Tokens</div>
          <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(stats.today_tokens) }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Today Cost</div>
          <div class="text-2xl font-bold text-primary-400 mt-1">{{ formatCost(stats.today_cost) }}</div>
        </div>
      </div>

      <!-- Cost Overview -->
      <div class="grid grid-cols-2 gap-4 mb-6">
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Total Tokens</div>
          <div class="text-2xl font-bold text-white mt-1">{{ formatNumber(stats.total_tokens) }}</div>
        </div>
        <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
          <div class="text-dark-400 text-sm">Total Cost</div>
          <div class="text-2xl font-bold text-primary-400 mt-1">{{ formatCost(stats.total_cost) }}</div>
        </div>
      </div>

      <!-- Model Cost Table -->
      <div class="bg-dark-800 rounded-lg border border-dark-700">
        <div class="px-4 py-3 border-b border-dark-700">
          <h2 class="text-white font-semibold">Cost by Model</h2>
        </div>
        <table v-if="stats.by_model && stats.by_model.length" class="w-full">
          <thead>
            <tr class="text-dark-400 text-sm border-b border-dark-700">
              <th class="text-left px-4 py-2">Model</th>
              <th class="text-right px-4 py-2">Requests</th>
              <th class="text-right px-4 py-2">Tokens</th>
              <th class="text-right px-4 py-2">Cost</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in stats.by_model" :key="entry.model" class="border-b border-dark-700/50 hover:bg-dark-700/30">
              <td class="px-4 py-2 text-white text-sm font-mono">{{ entry.model }}</td>
              <td class="px-4 py-2 text-white text-sm text-right">{{ formatNumber(entry.requests) }}</td>
              <td class="px-4 py-2 text-white text-sm text-right">{{ formatNumber(entry.tokens) }}</td>
              <td class="px-4 py-2 text-primary-400 text-sm text-right">{{ formatCost(entry.cost) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-else class="p-6 text-dark-500 text-sm text-center">No usage data yet</div>
      </div>
    </template>
  </div>
</template>
