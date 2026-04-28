<script lang="ts" setup>
import { ref, onMounted, watch } from 'vue'
import { ListUsage, ListUsageModels } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const logs = ref<model.UsageLog[]>([])
const total = ref(0)
const loading = ref(true)
const page = ref(1)
const pageSize = 20
const modelFilter = ref('')
const modelOptions = ref<string[]>([])
const startDate = ref('')
const endDate = ref('')

onMounted(() => {
  loadModels()
  load()
})

async function loadModels() {
  try {
    modelOptions.value = (await ListUsageModels()) ?? []
  } catch (e) {
    console.error(e)
    modelOptions.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const result: any = await ListUsage(pageSize, (page.value - 1) * pageSize, modelFilter.value, startDate.value, endDate.value)
    logs.value = result?.logs ?? []
    total.value = result?.total ?? 0
  } catch (e) {
    console.error(e)
    logs.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function totalPages(): number {
  return Math.max(1, Math.ceil(total.value / pageSize))
}

function formatTokens(n: number | undefined): string {
  if (n == null) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}

function formatCost(n: number | undefined): string {
  if (n == null) return '$0'
  return '$' + n.toFixed(6)
}

function formatDuration(ms: number | undefined | null): string {
  if (!ms) return '—'
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}

function toShanghai(utcStr: string | undefined | null): string {
  if (!utcStr) return '—'
  let str = utcStr.includes('T') ? utcStr : utcStr.replace(' ', 'T')
  let d = new Date(str)
  if (isNaN(d.getTime())) d = new Date(str + 'Z')
  if (isNaN(d.getTime())) return utcStr
  const bj = new Date(d.getTime() + 8 * 3600_000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${bj.getUTCFullYear()}-${pad(bj.getUTCMonth() + 1)}-${pad(bj.getUTCDate())} ${pad(bj.getUTCHours())}:${pad(bj.getUTCMinutes())}:${pad(bj.getUTCSeconds())}`
}

watch([modelFilter, startDate, endDate], () => {
  page.value = 1
  load()
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-white mb-6">使用记录</h1>

    <!-- 筛选 -->
    <div class="flex gap-3 mb-4">
      <select
        v-model="modelFilter"
        class="bg-dark-800 border border-dark-700 rounded-lg px-3 py-2 text-white text-sm w-52 focus:border-primary-500 outline-none"
      >
        <option value="">全部模型</option>
        <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
      </select>
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
      <div class="ml-auto text-dark-400 text-sm self-center">{{ total }} 条记录</div>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <div v-else-if="logs.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      暂无使用记录。
    </div>

    <template v-else>
      <div class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="text-dark-400 text-xs border-b border-dark-700">
                <th class="text-left px-3 py-2">模型</th>
                <th class="text-left px-3 py-2">账号</th>
                <th class="text-right px-3 py-2">输入</th>
                <th class="text-right px-3 py-2">输出</th>
                <th class="text-right px-3 py-2">缓存</th>
                <th class="text-right px-3 py-2">费用</th>
                <th class="text-center px-3 py-2">流式</th>
                <th class="text-right px-3 py-2">耗时</th>
                <th class="text-center px-3 py-2">状态</th>
                <th class="text-left px-3 py-2">时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="l in logs" :key="l.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
                <td class="px-3 py-2 text-white text-xs font-mono">{{ l.model }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs">{{ l.account_name || ('#' + l.account_id) }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens(l.input_tokens) }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens(l.output_tokens) }}</td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatTokens((l.cache_creation_tokens || 0) + (l.cache_read_tokens || 0)) }}</td>
                <td class="px-3 py-2 text-primary-400 text-xs text-right">{{ formatCost(l.total_cost) }}</td>
                <td class="px-3 py-2 text-center">
                  <span v-if="l.stream" class="text-green-400 text-xs">是</span>
                  <span v-else class="text-dark-500 text-xs">否</span>
                </td>
                <td class="px-3 py-2 text-dark-300 text-xs text-right">{{ formatDuration(l.duration_ms) }}</td>
                <td class="px-3 py-2 text-center">
                  <span v-if="l.status_code && l.status_code < 400" class="text-green-400 text-xs">{{ l.status_code }}</span>
                  <span v-else-if="l.status_code" class="text-red-400 text-xs">{{ l.status_code }}</span>
                  <span v-else class="text-dark-500 text-xs">—</span>
                </td>
                <td class="px-3 py-2 text-dark-400 text-xs">{{ toShanghai(l.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 分页 -->
      <div class="flex items-center justify-between mt-4">
        <button
          @click="page--; load()"
          :disabled="page <= 1"
          class="px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-dark-300 text-sm hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          上一页
        </button>
        <span class="text-dark-400 text-sm">第 {{ page }} 页 / 共 {{ totalPages() }} 页</span>
        <button
          @click="page++; load()"
          :disabled="page >= totalPages()"
          class="px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-dark-300 text-sm hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          下一页
        </button>
      </div>
    </template>
  </div>
</template>
