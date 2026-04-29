<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { ListAccounts, ListGroups, CreateAccount, UpdateAccount, DeleteAccount, HealthCheckAccount, HealthCheckAllAccounts } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const accounts = ref<model.Account[]>([])
const groups = ref<model.Group[]>([])
const loading = ref(true)
const showModal = ref(false)
const editing = ref(false)
const form = ref(emptyForm())

// Health check state
const checkModel = ref('')
const checkingAll = ref(false)
const checkingIds = ref<Set<number>>(new Set())
const checkResults = ref<Map<number, { healthy: boolean; latency: number; error: string }>>(new Map())

function emptyForm() {
  return {
    id: 0,
    name: '',
    platform: 'claude',
    type: 'api_key',
    credentials: {} as Record<string, any>,
    extra: {} as Record<string, any>,
    proxy_id: null as number | null,
    base_url: '' as string,
    concurrency: 3,
    priority: 50,
    multiplier: 1,
    status: 'active',
    error_message: '',
    schedulable: true,
    group_ids: [] as number[],
  }
}

const platforms = ['claude', 'openai', 'gemini']
const authTypes = ['api_key', 'oauth', 'cookie']

const modelOptions = [
  { label: '默认（按平台自动选择）', value: '' },
  { label: 'Claude 3.5 Haiku', value: 'claude-3-5-haiku-20241022' },
  { label: 'Claude Haiku 4.5', value: 'claude-haiku-4-5-20251001' },
  { label: 'GPT-5.4', value: 'gpt-5.4' },
  { label: 'GPT-4o', value: 'gpt-4o' },
  { label: 'Gemini 2.0 Flash', value: 'gemini-2.0-flash' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [accs, grps] = await Promise.all([ListAccounts(), ListGroups()])
    accounts.value = accs ?? []
    groups.value = grps ?? []
  } catch (e) {
    console.error(e)
    accounts.value = []
    groups.value = []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  form.value = emptyForm()
  loadGroups()
  showModal.value = true
}

function openEdit(a: model.Account) {
  editing.value = true
  form.value = {
    id: a.id,
    name: a.name,
    platform: a.platform,
    type: a.type,
    credentials: { ...a.credentials },
    extra: { ...a.extra },
    proxy_id: a.proxy_id ?? null,
    base_url: a.base_url ?? '',
    concurrency: a.concurrency,
    priority: a.priority,
    multiplier: a.multiplier ?? 1,
    status: a.status,
    error_message: a.error_message ?? '',
    schedulable: a.schedulable,
    group_ids: [...(a.group_ids ?? [])],
  }
  loadGroups()
  showModal.value = true
}

async function loadGroups() {
  try {
    groups.value = await ListGroups() ?? []
  } catch (e) {
    console.error(e)
    groups.value = []
  }
}

function toggleGroup(gid: number) {
  const idx = form.value.group_ids.indexOf(gid)
  if (idx >= 0) form.value.group_ids.splice(idx, 1)
  else form.value.group_ids.push(gid)
}

const filteredGroups = computed(() => {
  return groups.value
})

async function save() {
  const obj: any = { ...form.value }
  if (!editing.value) delete obj.id
  if (!obj.base_url) obj.base_url = null
  try {
    if (editing.value) {
      await UpdateAccount(obj)
    } else {
      await CreateAccount(obj)
    }
    showModal.value = false
    await load()
  } catch (e) {
    alert('保存失败: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('确定删除此账号？')) return
  try {
    await DeleteAccount(id)
    await load()
  } catch (e) {
    alert('删除失败: ' + e)
  }
}

function statusClass(s: string) {
  return s === 'active' ? 'text-green-400' : s === 'error' ? 'text-red-400' : 'text-dark-400'
}

async function checkSingle(id: number) {
  checkingIds.value = new Set([...checkingIds.value, id])
  checkResults.value.delete(id)
  try {
    const result = await HealthCheckAccount(id, checkModel.value)
    checkResults.value = new Map(checkResults.value.set(id, {
      healthy: result.healthy,
      latency: result.latency_ms,
      error: result.error ?? '',
    }))
    await load()
  } catch (e: any) {
    checkResults.value = new Map(checkResults.value.set(id, {
      healthy: false,
      latency: 0,
      error: String(e),
    }))
  } finally {
    checkingIds.value = new Set([...checkingIds.value].filter(i => i !== id))
  }
}

async function checkAll() {
  checkingAll.value = true
  checkResults.value = new Map()
  try {
    const results = await HealthCheckAllAccounts(checkModel.value) ?? []
    const map = new Map<number, { healthy: boolean; latency: number; error: string }>()
    for (const r of results) {
      map.set(r.account_id, {
        healthy: r.healthy,
        latency: r.latency_ms,
        error: r.error ?? '',
      })
    }
    checkResults.value = map
    await load()
  } catch (e) {
    alert('健康检查失败: ' + e)
  } finally {
    checkingAll.value = false
  }
}

const credFields = computed(() => {
  const p = form.value.platform
  const t = form.value.type
  if (p === 'claude' && t === 'api_key') return ['api_key']
  if (p === 'claude' && t === 'oauth') return ['access_token', 'refresh_token']
  if (p === 'claude' && t === 'cookie') return ['session_key']
  if (p === 'openai' && t === 'api_key') return ['api_key', 'organization_id']
  if (p === 'openai' && t === 'oauth') return ['access_token', 'refresh_token']
  if (p === 'gemini' && t === 'api_key') return ['api_key']
  if (p === 'gemini' && t === 'oauth') return ['access_token', 'refresh_token']
  return ['api_key']
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">账号管理</h1>
      <div class="flex items-center gap-3">
        <select v-model="checkModel" class="bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
          <option v-for="opt in modelOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button @click="checkAll" :disabled="checkingAll || accounts.length === 0"
          class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:bg-dark-600 disabled:text-dark-400 text-white rounded-lg text-sm transition-colors flex items-center gap-2">
          <svg v-if="checkingAll" class="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
          {{ checkingAll ? '检查中...' : '检查全部' }}
        </button>
        <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
          + 添加账号
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <div v-else-if="accounts.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      暂无账号，点击"添加账号"开始配置。
    </div>

    <div v-else class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="text-dark-400 text-sm border-b border-dark-700">
            <th class="text-left px-4 py-3">名称</th>
            <th class="text-left px-4 py-3">平台</th>
            <th class="text-left px-4 py-3">认证类型</th>
            <th class="text-center px-4 py-3">优先级</th>
            <th class="text-center px-4 py-3">并发数</th>
            <th class="text-center px-4 py-3">倍率</th>
            <th class="text-left px-4 py-3">状态</th>
            <th class="text-left px-4 py-3">健康检查</th>
            <th class="text-right px-4 py-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in accounts" :key="a.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
            <td class="px-4 py-3 text-white text-sm">{{ a.name }}</td>
            <td class="px-4 py-3">
              <span class="text-xs px-2 py-0.5 rounded bg-dark-700 text-dark-300">{{ a.platform }}</span>
            </td>
            <td class="px-4 py-3">
              <span class="text-xs px-2 py-0.5 rounded bg-dark-700 text-dark-300">{{ a.type }}</span>
            </td>
            <td class="px-4 py-3 text-white text-sm text-center">{{ a.priority }}</td>
            <td class="px-4 py-3 text-white text-sm text-center">{{ a.concurrency }}</td>
            <td class="px-4 py-3 text-center">
              <span :class="a.multiplier !== 1 ? 'text-yellow-400' : 'text-dark-400'" class="text-sm">{{ a.multiplier ?? 1 }}x</span>
            </td>
            <td class="px-4 py-3">
              <span :class="statusClass(a.status)" class="text-sm">{{ a.status }}</span>
              <div v-if="a.error_message" class="text-red-400 text-xs mt-0.5 truncate max-w-48" :title="a.error_message">{{ a.error_message }}</div>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <button @click="checkSingle(a.id)" :disabled="checkingIds.has(a.id)"
                  class="text-xs px-2 py-1 rounded border transition-colors"
                  :class="checkingIds.has(a.id) ? 'border-dark-600 text-dark-500 cursor-wait' : 'border-dark-600 text-dark-300 hover:text-white hover:border-dark-400'">
                  <svg v-if="checkingIds.has(a.id)" class="animate-spin h-3 w-3 inline mr-1" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                  {{ checkingIds.has(a.id) ? '检查中' : '检查' }}
                </button>
                <template v-if="checkResults.has(a.id)">
                  <template v-if="checkResults.get(a.id)!.healthy">
                    <span class="text-green-400 text-xs">{{ checkResults.get(a.id)!.latency }}ms</span>
                  </template>
                  <template v-else>
                    <span class="text-red-400 text-xs truncate max-w-32 block" :title="checkResults.get(a.id)!.error">失败</span>
                  </template>
                </template>
              </div>
            </td>
            <td class="px-4 py-3 text-right space-x-2">
              <button @click="openEdit(a)" class="text-primary-400 hover:text-primary-300 text-sm">编辑</button>
              <button @click="remove(a.id)" class="text-red-400 hover:text-red-300 text-sm">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-lg max-h-[80vh] overflow-y-auto p-6">
        <h2 class="text-lg font-semibold text-white mb-4">{{ editing ? '编辑账号' : '添加账号' }}</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">名称</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="我的 API 账号" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">平台</label>
              <select v-model="form.platform" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option v-for="p in platforms" :key="p" :value="p">{{ p }}</option>
              </select>
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">认证类型</label>
              <select v-model="form.type" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option v-for="t in authTypes" :key="t" :value="t">{{ t }}</option>
              </select>
            </div>
          </div>

          <div v-for="field in credFields" :key="field">
            <label class="block text-dark-300 text-sm mb-1">{{ field }}</label>
            <input
              v-model="form.credentials[field]"
              class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm font-mono focus:border-primary-500 outline-none"
              :placeholder="field"
            />
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">Base URL <span class="text-dark-500">(可选，留空使用默认地址)</span></label>
            <input
              v-model="form.base_url"
              class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm font-mono focus:border-primary-500 outline-none"
              :placeholder="form.platform === 'claude' ? 'https://api.anthropic.com/v1' : form.platform === 'openai' ? 'https://api.openai.com/v1' : 'https://generativelanguage.googleapis.com/v1beta'"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">优先级</label>
              <input v-model.number="form.priority" type="number" min="0" max="100" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">并发数</label>
              <input v-model.number="form.concurrency" type="number" min="1" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">费用倍率</label>
              <input v-model.number="form.multiplier" type="number" min="0" step="0.1" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">状态</label>
              <select v-model="form.status" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option value="active">正常</option>
                <option value="disabled">禁用</option>
              </select>
            </div>
          </div>

          <label class="flex items-center gap-2 text-dark-300 text-sm cursor-pointer">
            <input v-model="form.schedulable" type="checkbox" class="rounded" />
            可调度
          </label>

          <div v-if="filteredGroups.length > 0">
            <label class="block text-dark-300 text-sm mb-2">所属分组</label>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="g in filteredGroups" :key="g.id"
                @click="toggleGroup(g.id)"
                class="px-3 py-1.5 rounded-lg text-sm border transition-colors"
                :class="form.group_ids.includes(g.id)
                  ? 'bg-primary-600/20 border-primary-500 text-primary-300'
                  : 'bg-dark-900 border-dark-600 text-dark-400 hover:border-dark-400'"
              >{{ g.name }}</button>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button @click="showModal = false" class="px-4 py-2 text-dark-300 hover:text-white text-sm transition-colors">取消</button>
          <button @click="save" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
            {{ editing ? '更新' : '创建' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
