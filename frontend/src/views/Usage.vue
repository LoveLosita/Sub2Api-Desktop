<script lang="ts" setup>
import { ref, onMounted, watch } from 'vue'
import { ListUsage } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const logs = ref<model.UsageLog[]>([])
const total = ref(0)
const loading = ref(true)
const page = ref(1)
const pageSize = 20
const modelFilter = ref('')
const startDate = ref('')
const endDate = ref('')

onMounted(load)

async function load() {
  loading.value = true
  try {
    const result: any = await ListUsage(pageSize, (page.value - 1) * pageSize, modelFilter.value, startDate.value, endDate.value)
    logs.value = (Array.isArray(result) ? result[0] : result?.logs) ?? []
    total.value = Array.isArray(result) ? result[1] : (result?.total ?? 0)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function totalPages(): number {
  return Math.max(1, Math.ceil(total.value / pageSize))
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}

function formatCost(n: number): string {
  return '$' + n.toFixed(6)
}

function formatDuration(ms: number | undefined): string {
  if (!ms) return '—'
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}

watch([modelFilter, startDate, endDate], () => {
  page.value = 1
  load()
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-white mb-6">Usage</h1>

    <!-- Filters -->
    <div class="flex gap-3 mb-4">
      <input
        v-model="modelFilter"
        class="bg-dark-800 border border-dark-700 rounded-lg px-3 py-2 text-white text-sm w-48 focus:border-primary-500 outline-none"
        placeholder="Filter by model..."
      />
      <input
        v-model="startDate"
        type="date"
        class="bg-dark-800 border border-dark-700 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none"
      />
      <input
        v-model="endDate"
        type="date"
        class="bg-dark-800 border border-dark-700 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none"
      />
      <div class="ml-auto text-dark-400 text-sm self-center">{{ total }} records</div>
    </div>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <div v-else-if="logs.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      No usage records yet.
    </div>

    <template v-else>
      <div class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="text-dark-400 text-xs border-b border-dark-700">
                <th class="text-left px-3 py-2">Model</th>
                <th class="text-right px-3 py-2">Input</th>
                <th class="text-right px-3 py-2">Output</th>
                <th class="text-right px-3 py-2">Cache</th>
                <th class="text-right px-3 py-2">Cost</th>
                <th class="text-center px-3 py-2">Stream</th>
                <th class="text-right px-3 py-2">Duration</th>
                <th class="text-center px-3 py-2">Status</th>
                <th class="text-left px-3 py-2">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="l in logs" :key="l.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
                <td class="px-3 py-2 text-white text-xs font-mono">{{ l.model }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens(l.input_tokens) }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens(l.output_tokens) }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens(l.cache_creation_tokens + l.cache_read_tokens) }}</td>
                <td class="px-3 py-2 text-primary-400 text-xs text-right">{{ formatCost(l.total_cost) }}</td>
                <td class="px-3 py-2 text-center">
                  <span v-if="l.stream" class="text-green-400 text-xs">Yes</span>
                  <span v-else class="text-dark-500 text-xs">No</span>
                </td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatDuration(l.duration_ms) }}</td>
                <td class="px-3 py-2 text-center">
                  <span v-if="l.status_code && l.status_code < 400" class="text-green-400 text-xs">{{ l.status_code }}</span>
                  <span v-else-if="l.status_code" class="text-red-400 text-xs">{{ l.status_code }}</span>
                  <span v-else class="text-dark-500 text-xs">—</span>
                </td>
                <td class="px-3 py-2 text-dark-400 text-xs">{{ l.created_at }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Pagination -->
      <div class="flex items-center justify-between mt-4">
        <button
          @click="page--; load()"
          :disabled="page <= 1"
          class="px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-dark-300 text-sm hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          Previous
        </button>
        <span class="text-dark-400 text-sm">Page {{ page }} of {{ totalPages() }}</span>
        <button
          @click="page++; load()"
          :disabled="page >= totalPages()"
          class="px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-dark-300 text-sm hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          Next
        </button>
      </div>
    </template>
  </div>
</template>
