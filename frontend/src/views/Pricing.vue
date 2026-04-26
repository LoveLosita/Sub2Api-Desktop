<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { ListPricing, UpdatePricing, ResetPricing, FetchRemotePricing } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

type Pricing = model.ModelPricing

const allPricing = ref<Pricing[]>([])
const loading = ref(true)
const fetching = ref(false)
const editingId = ref<number | null>(null)
const editForm = ref({ input: 0, output: 0, cacheCreation: 0, cacheRead: 0 })
const activeTab = ref('claude')

onMounted(load)

async function load() {
  loading.value = true
  try {
    allPricing.value = await ListPricing() ?? []
  } catch (e) {
    console.error(e)
    allPricing.value = []
  } finally {
    loading.value = false
  }
}

function platformOf(m: string): string {
  if (m.startsWith('claude-') || m.startsWith('claude_')) return 'claude'
  if (m.startsWith('gpt-') || m.startsWith('o1') || m.startsWith('o3') || m.startsWith('o4')) return 'openai'
  if (m.startsWith('gemini-') || m.startsWith('gemini_')) return 'gemini'
  return 'other'
}

const tabs = [
  { key: 'claude', label: 'Claude' },
  { key: 'openai', label: 'OpenAI' },
  { key: 'gemini', label: 'Gemini' },
]

const filtered = computed(() => allPricing.value.filter(p => platformOf(p.model) === activeTab.value))

function fmt(n: number | undefined): string {
  if (n == null) return '—'
  return '$' + n.toFixed(2)
}

function startEdit(p: Pricing) {
  editingId.value = p.id
  editForm.value = {
    input: p.input_price ?? 0,
    output: p.output_price ?? 0,
    cacheCreation: p.cache_creation_price ?? 0,
    cacheRead: p.cache_read_price ?? 0,
  }
}

async function saveEdit() {
  if (editingId.value == null) return
  try {
    await UpdatePricing(editingId.value, editForm.value.input, editForm.value.output, editForm.value.cacheCreation, editForm.value.cacheRead)
    editingId.value = null
    await load()
  } catch (e) {
    alert('保存失败: ' + e)
  }
}

function cancelEdit() {
  editingId.value = null
}

async function resetPricing() {
  if (!confirm('确定重置为默认价格？')) return
  try {
    await ResetPricing()
    await load()
  } catch (e) {
    alert('重置失败: ' + e)
  }
}

async function fetchRemote() {
  fetching.value = true
  try {
    const count = await FetchRemotePricing()
    await load()
    alert(`已从远程更新 ${count} 个模型价格`)
  } catch (e) {
    alert('拉取失败: ' + e)
  } finally {
    fetching.value = false
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">模型定价</h1>
      <div class="flex items-center gap-3">
        <button @click="resetPricing" class="px-3 py-1.5 text-dark-300 hover:text-white text-xs border border-dark-600 rounded-lg transition-colors">
          重置默认
        </button>
        <button @click="fetchRemote" :disabled="fetching" class="px-3 py-1.5 bg-primary-600 hover:bg-primary-500 disabled:opacity-50 text-white text-xs rounded-lg transition-colors">
          {{ fetching ? '拉取中...' : '从远程拉取最新价格' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <template v-else>
      <!-- Tabs -->
      <div class="flex gap-1 mb-4">
        <button v-for="tab in tabs" :key="tab.key" @click="activeTab = tab.key"
          class="px-4 py-2 text-sm rounded-lg transition-colors"
          :class="activeTab === tab.key ? 'bg-primary-600 text-white' : 'bg-dark-800 text-dark-400 hover:text-white'">
          {{ tab.label }}
        </button>
      </div>

      <!-- Table -->
      <div class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
        <table class="w-full">
          <thead>
            <tr class="text-dark-400 text-sm border-b border-dark-700">
              <th class="text-left px-4 py-3">模型</th>
              <th class="text-right px-4 py-3">输入 ($/M)</th>
              <th class="text-right px-4 py-3">输出 ($/M)</th>
              <th class="text-right px-4 py-3">缓存写入 ($/M)</th>
              <th class="text-right px-4 py-3">缓存读取 ($/M)</th>
              <th class="text-right px-4 py-3">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in filtered" :key="p.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
              <template v-if="editingId === p.id">
                <td class="px-4 py-2 text-white text-sm font-mono">{{ p.model }}</td>
                <td class="px-4 py-2 text-right"><input v-model.number="editForm.input" type="number" step="0.01" class="w-20 bg-dark-900 border border-dark-600 rounded px-2 py-1 text-white text-sm text-right" /></td>
                <td class="px-4 py-2 text-right"><input v-model.number="editForm.output" type="number" step="0.01" class="w-20 bg-dark-900 border border-dark-600 rounded px-2 py-1 text-white text-sm text-right" /></td>
                <td class="px-4 py-2 text-right"><input v-model.number="editForm.cacheCreation" type="number" step="0.01" class="w-20 bg-dark-900 border border-dark-600 rounded px-2 py-1 text-white text-sm text-right" /></td>
                <td class="px-4 py-2 text-right"><input v-model.number="editForm.cacheRead" type="number" step="0.01" class="w-20 bg-dark-900 border border-dark-600 rounded px-2 py-1 text-white text-sm text-right" /></td>
                <td class="px-4 py-2 text-right space-x-2">
                  <button @click="saveEdit" class="text-green-400 hover:text-green-300 text-sm">保存</button>
                  <button @click="cancelEdit" class="text-dark-400 hover:text-white text-sm">取消</button>
                </td>
              </template>
              <template v-else>
                <td class="px-4 py-2 text-white text-sm font-mono">{{ p.model }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ fmt(p.input_price) }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ fmt(p.output_price) }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ fmt(p.cache_creation_price) }}</td>
                <td class="px-4 py-2 text-white text-sm text-right">{{ fmt(p.cache_read_price) }}</td>
                <td class="px-4 py-2 text-right">
                  <button @click="startEdit(p)" class="text-primary-400 hover:text-primary-300 text-sm">编辑</button>
                </td>
              </template>
            </tr>
          </tbody>
        </table>
        <div v-if="filtered.length === 0" class="p-6 text-dark-500 text-sm text-center">暂无定价数据</div>
      </div>

      <div class="mt-4 text-dark-500 text-xs">
        价格单位: $/M tokens（每百万 token 美元）。点击"编辑"修改价格，点击"从远程拉取最新价格"从 LiteLLM 获取官方最新定价。
      </div>
    </template>
  </div>
</template>
