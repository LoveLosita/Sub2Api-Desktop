<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListGroups, CreateGroup, UpdateGroup, DeleteGroup, ListAccounts } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const groups = ref<model.Group[]>([])
const accounts = ref<model.Account[]>([])
const loading = ref(true)
const showModal = ref(false)
const editing = ref(false)
const form = ref(emptyForm())

function emptyForm() {
  return {
    id: 0,
    name: '',
    description: '',
    platform: 'claude',
    rate_multiplier: 1.0,
    is_exclusive: false,
    status: 'active',
    model_routing: {} as Record<string, number[]>,
    model_routing_enabled: false,
    account_ids: [] as number[],
  }
}

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [g, a] = await Promise.all([ListGroups(), ListAccounts()])
    groups.value = g
    accounts.value = a
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

function openEdit(g: model.Group) {
  editing.value = true
  form.value = {
    id: g.id,
    name: g.name,
    description: g.description ?? '',
    platform: g.platform,
    rate_multiplier: g.rate_multiplier,
    is_exclusive: g.is_exclusive,
    status: g.status,
    model_routing: { ...(g.model_routing ?? {}) },
    model_routing_enabled: g.model_routing_enabled,
    account_ids: [...(g.account_ids ?? [])],
  }
  showModal.value = true
}

async function save() {
  const obj: any = { ...form.value }
  if (!editing.value) delete obj.id
  try {
    if (editing.value) {
      await UpdateGroup(obj)
    } else {
      await CreateGroup(obj)
    }
    showModal.value = false
    await load()
  } catch (e) {
    alert('Save failed: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('Delete this group?')) return
  try {
    await DeleteGroup(id)
    await load()
  } catch (e) {
    alert('Delete failed: ' + e)
  }
}

function toggleAccount(aid: number) {
  const idx = form.value.account_ids.indexOf(aid)
  if (idx >= 0) form.value.account_ids.splice(idx, 1)
  else form.value.account_ids.push(aid)
}

function accountName(id: number): string {
  return accounts.value.find(a => a.id === id)?.name ?? `#${id}`
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">Groups</h1>
      <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
        + Add Group
      </button>
    </div>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <div v-else-if="groups.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      No groups yet. Click "Add Group" to create one.
    </div>

    <div v-else class="space-y-4">
      <div v-for="g in groups" :key="g.id" class="bg-dark-800 rounded-lg border border-dark-700 p-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-white font-semibold">{{ g.name }}</h3>
            <div class="flex items-center gap-3 mt-1">
              <span class="text-xs px-2 py-0.5 rounded bg-dark-700 text-dark-300">{{ g.platform }}</span>
              <span class="text-xs" :class="g.status === 'active' ? 'text-green-400' : 'text-dark-400'">{{ g.status }}</span>
              <span class="text-dark-500 text-xs">Multiplier: {{ g.rate_multiplier }}x</span>
              <span v-if="g.is_exclusive" class="text-xs px-2 py-0.5 rounded bg-yellow-900/30 text-yellow-400">exclusive</span>
            </div>
            <div v-if="g.description" class="text-dark-400 text-sm mt-1">{{ g.description }}</div>
          </div>
          <div class="flex gap-2">
            <button @click="openEdit(g)" class="text-primary-400 hover:text-primary-300 text-sm">Edit</button>
            <button @click="remove(g.id)" class="text-red-400 hover:text-red-300 text-sm">Delete</button>
          </div>
        </div>
        <div v-if="g.account_ids && g.account_ids.length" class="mt-3 flex flex-wrap gap-1">
          <span v-for="aid in g.account_ids" :key="aid" class="text-xs px-2 py-0.5 rounded bg-dark-700 text-dark-300">
            {{ accountName(aid) }}
          </span>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-lg max-h-[80vh] overflow-y-auto p-6">
        <h2 class="text-lg font-semibold text-white mb-4">{{ editing ? 'Edit Group' : 'Add Group' }}</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">Name</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="Group name" />
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">Description</label>
            <input v-model="form.description" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="Optional description" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">Platform</label>
              <select v-model="form.platform" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option value="claude">claude</option>
                <option value="openai">openai</option>
                <option value="gemini">gemini</option>
              </select>
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">Rate Multiplier</label>
              <input v-model.number="form.rate_multiplier" type="number" step="0.1" min="0" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-300 text-sm mb-1">Status</label>
              <select v-model="form.status" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </div>
            <label class="flex items-end gap-2 pb-1 text-dark-300 text-sm cursor-pointer">
              <input v-model="form.is_exclusive" type="checkbox" class="rounded" />
              Exclusive
            </label>
          </div>

          <!-- Account Selection -->
          <div>
            <label class="block text-dark-300 text-sm mb-2">Accounts</label>
            <div class="bg-dark-900 border border-dark-600 rounded-lg p-2 max-h-40 overflow-y-auto space-y-1">
              <label v-for="acc in accounts" :key="acc.id" class="flex items-center gap-2 px-2 py-1 hover:bg-dark-700 rounded cursor-pointer">
                <input type="checkbox" :checked="form.account_ids.includes(acc.id)" @change="toggleAccount(acc.id)" class="rounded" />
                <span class="text-white text-sm">{{ acc.name }}</span>
                <span class="text-dark-500 text-xs">{{ acc.platform }}</span>
              </label>
              <div v-if="accounts.length === 0" class="text-dark-500 text-sm px-2">No accounts available</div>
            </div>
          </div>
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
