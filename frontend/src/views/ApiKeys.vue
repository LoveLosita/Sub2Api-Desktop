<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListAPIKeys, CreateAPIKey, DeleteAPIKey, ListGroups } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const apiKeys = ref<model.APIKey[]>([])
const groups = ref<model.Group[]>([])
const loading = ref(true)
const showModal = ref(false)
const form = ref(emptyForm())
const newlyCreatedKey = ref('')

function emptyForm() {
  return {
    name: '',
    key: '',
    group_id: null as number | null,
    status: 'active',
    ip_whitelist: [] as string[],
    ip_blacklist: [] as string[],
  }
}

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [k, g] = await Promise.all([ListAPIKeys(), ListGroups()])
    apiKeys.value = k
    groups.value = g
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = emptyForm()
  newlyCreatedKey.value = ''
  showModal.value = true
}

async function save() {
  const obj: any = { ...form.value }
  if (!obj.group_id) obj.group_id = null
  try {
    await CreateAPIKey(obj)
    showModal.value = false
    await load()
    if (apiKeys.value.length > 0) {
      const last = apiKeys.value[apiKeys.value.length - 1]
      newlyCreatedKey.value = last.key
    }
  } catch (e) {
    alert('Create failed: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('Delete this API key?')) return
  try {
    await DeleteAPIKey(id)
    await load()
  } catch (e) {
    alert('Delete failed: ' + e)
  }
}

function copyKey(key: string) {
  navigator.clipboard.writeText(key)
}

function groupName(gid: number | undefined): string {
  if (!gid) return '—'
  const found = groups.value.find((gr: model.Group) => gr.id === gid)
  return found?.name ?? `#${gid}`
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">API Keys</h1>
      <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
        + Generate Key
      </button>
    </div>

    <!-- New key notification -->
    <div v-if="newlyCreatedKey" class="mb-4 bg-green-900/20 border border-green-700/50 rounded-lg p-4 flex items-center justify-between">
      <div>
        <div class="text-green-400 text-sm font-medium">New API Key Created</div>
        <code class="text-white text-sm font-mono mt-1 block">{{ newlyCreatedKey }}</code>
      </div>
      <div class="flex gap-2">
        <button @click="copyKey(newlyCreatedKey)" class="text-green-400 hover:text-green-300 text-sm">Copy</button>
        <button @click="newlyCreatedKey = ''" class="text-dark-400 hover:text-white text-sm">Dismiss</button>
      </div>
    </div>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <div v-else-if="apiKeys.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      No API keys yet. Click "Generate Key" to create one.
    </div>

    <div v-else class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="text-dark-400 text-sm border-b border-dark-700">
            <th class="text-left px-4 py-3">Name</th>
            <th class="text-left px-4 py-3">Key</th>
            <th class="text-left px-4 py-3">Group</th>
            <th class="text-left px-4 py-3">Status</th>
            <th class="text-left px-4 py-3">Last Used</th>
            <th class="text-right px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="k in apiKeys" :key="k.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
            <td class="px-4 py-3 text-white text-sm">{{ k.name }}</td>
            <td class="px-4 py-3 text-white text-sm font-mono">{{ k.key.substring(0, 12) }}...</td>
            <td class="px-4 py-3 text-white text-sm">{{ groupName(k.group_id) }}</td>
            <td class="px-4 py-3">
              <span :class="k.status === 'active' ? 'text-green-400' : 'text-dark-400'" class="text-sm">{{ k.status }}</span>
            </td>
            <td class="px-4 py-3 text-dark-400 text-sm">{{ k.last_used_at || 'Never' }}</td>
            <td class="px-4 py-3 text-right space-x-2">
              <button @click="copyKey(k.key)" class="text-primary-400 hover:text-primary-300 text-sm">Copy</button>
              <button @click="remove(k.id)" class="text-red-400 hover:text-red-300 text-sm">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-md p-6">
        <h2 class="text-lg font-semibold text-white mb-4">Generate API Key</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">Name</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="Key name" />
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">Assign to Group (optional)</label>
            <select v-model="form.group_id" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
              <option :value="null">No group</option>
              <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button @click="showModal = false" class="px-4 py-2 text-dark-300 hover:text-white text-sm transition-colors">Cancel</button>
          <button @click="save" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
            Generate
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
