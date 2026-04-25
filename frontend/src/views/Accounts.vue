<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { ListAccounts, CreateAccount, UpdateAccount, DeleteAccount } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const accounts = ref<model.Account[]>([])
const loading = ref(true)
const showModal = ref(false)
const editing = ref(false)
const form = ref(emptyForm())

function emptyForm() {
  return {
    id: 0,
    name: '',
    platform: 'claude',
    type: 'api_key',
    credentials: {} as Record<string, any>,
    extra: {} as Record<string, any>,
    proxy_id: null as number | null,
    concurrency: 3,
    priority: 50,
    status: 'active',
    error_message: '',
    schedulable: true,
    group_ids: [] as number[],
  }
}

const platforms = ['claude', 'openai', 'gemini']
const authTypes = ['api_key', 'oauth', 'cookie']

onMounted(load)

async function load() {
  loading.value = true
  try {
    accounts.value = await ListAccounts()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  form.value = emptyForm()
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
    concurrency: a.concurrency,
    priority: a.priority,
    status: a.status,
    error_message: a.error_message ?? '',
    schedulable: a.schedulable,
    group_ids: [...(a.group_ids ?? [])],
  }
  showModal.value = true
}

async function save() {
  const obj: any = { ...form.value }
  if (!editing.value) delete obj.id
  try {
    if (editing.value) {
      await UpdateAccount(obj)
    } else {
      await CreateAccount(obj)
    }
    showModal.value = false
    await load()
  } catch (e) {
    alert('Save failed: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('Delete this account?')) return
  try {
    await DeleteAccount(id)
    await load()
  } catch (e) {
    alert('Delete failed: ' + e)
  }
}

function statusClass(s: string) {
  return s === 'active' ? 'text-green-400' : s === 'error' ? 'text-red-400' : 'text-dark-400'
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
      <h1 class="text-2xl font-bold text-white">Accounts</h1>
      <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
        + Add Account
      </button>
    </div>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <div v-else-if="accounts.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      No accounts yet. Click "Add Account" to get started.
    </div>

    <div v-else class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="text-dark-400 text-sm border-b border-dark-700">
            <th class="text-left px-4 py-3">Name</th>
            <th class="text-left px-4 py-3">Platform</th>
            <th class="text-left px-4 py-3">Type</th>
            <th class="text-center px-4 py-3">Priority</th>
            <th class="text-center px-4 py-3">Concurrency</th>
            <th class="text-left px-4 py-3">Status</th>
            <th class="text-right px-4 py-3">Actions</th>
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
            <td class="px-4 py-3">
              <span :class="statusClass(a.status)" class="text-sm">{{ a.status }}</span>
              <div v-if="a.error_message" class="text-red-400 text-xs mt-0.5 truncate max-w-48" :title="a.error_message">{{ a.error_message }}</div>
            </td>
            <td class="px-4 py-3 text-right space-x-2">
              <button @click="openEdit(a)" class="text-primary-400 hover:text-primary-300 text-sm">Edit</button>
              <button @click="remove(a.id)" class="text-red-400 hover:text-red-300 text-sm">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-lg max-h-[80vh] overflow-y-auto p-6">
        <h2 class="text-lg font-semibold text-white mb-4">{{ editing ? 'Edit Account' : 'Add Account' }}</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">Name</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="My API Account" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">Platform</label>
              <select v-model="form.platform" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option v-for="p in platforms" :key="p" :value="p">{{ p }}</option>
              </select>
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">Auth Type</label>
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

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">Priority</label>
              <input v-model.number="form.priority" type="number" min="0" max="100" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">Concurrency</label>
              <input v-model.number="form.concurrency" type="number" min="1" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">Status</label>
              <select v-model="form.status" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </div>
          </div>

          <label class="flex items-center gap-2 text-dark-300 text-sm cursor-pointer">
            <input v-model="form.schedulable" type="checkbox" class="rounded" />
            Schedulable
          </label>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button @click="showModal = false" class="px-4 py-2 text-dark-300 hover:text-white text-sm transition-colors">Cancel</button>
          <button @click="save" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
            {{ editing ? 'Update' : 'Create' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
