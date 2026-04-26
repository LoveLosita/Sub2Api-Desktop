<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListAPIKeys, CreateAPIKey, DeleteAPIKey, ListGroups, GetProxyAddr } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const apiKeys = ref<model.APIKey[]>([])
const groups = ref<model.Group[]>([])
const loading = ref(true)
const showModal = ref(false)
const form = ref(emptyForm())
const newlyCreatedKey = ref('')
const proxyAddr = ref('127.0.0.1:8787')

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

const baseUrl = ref('http://127.0.0.1:8787/v1')

onMounted(async () => {
  try {
    const addr = await GetProxyAddr()
    if (addr) {
      proxyAddr.value = addr
      baseUrl.value = `http://${addr}/v1`
    }
  } catch { /* ignore */ }
  load()
})

async function load() {
  loading.value = true
  try {
    const [k, g] = await Promise.all([ListAPIKeys(), ListGroups()])
    apiKeys.value = k ?? []
    groups.value = g ?? []
  } catch (e) {
    console.error(e)
    apiKeys.value = []
    groups.value = []
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
    alert('创建失败: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('确定删除此密钥？')) return
  try {
    await DeleteAPIKey(id)
    await load()
  } catch (e) {
    alert('删除失败: ' + e)
  }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

function copyKeyConfig(key: string) {
  const config = `Base URL: ${baseUrl.value}\nAPI Key: ${key}`
  navigator.clipboard.writeText(config)
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
      <h1 class="text-2xl font-bold text-white">API 密钥</h1>
      <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
        + 生成密钥
      </button>
    </div>

    <!-- Base URL 提示 -->
    <div class="mb-4 bg-dark-800 rounded-lg border border-dark-700 p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-dark-400 text-xs mb-1">代理端点 (Base URL)</div>
          <code class="text-primary-400 text-sm font-mono">{{ baseUrl }}</code>
        </div>
        <button @click="copyToClipboard(baseUrl)" class="text-dark-400 hover:text-white text-sm transition-colors">复制</button>
      </div>
      <div class="text-dark-500 text-xs mt-2">在客户端中将此地址设为 API Base URL，并使用下方生成的密钥进行认证。</div>
    </div>

    <!-- 新密钥通知 -->
    <div v-if="newlyCreatedKey" class="mb-4 bg-green-900/20 border border-green-700/50 rounded-lg p-4">
      <div class="text-green-400 text-sm font-medium mb-2">新 API 密钥已创建</div>
      <div class="space-y-1">
        <div class="flex items-center justify-between bg-dark-900/50 rounded px-3 py-1.5">
          <div>
            <span class="text-dark-400 text-xs mr-2">Base URL:</span>
            <code class="text-white text-sm font-mono">{{ baseUrl }}</code>
          </div>
          <button @click="copyToClipboard(baseUrl)" class="text-green-400 hover:text-green-300 text-xs">复制</button>
        </div>
        <div class="flex items-center justify-between bg-dark-900/50 rounded px-3 py-1.5">
          <div>
            <span class="text-dark-400 text-xs mr-2">API Key:</span>
            <code class="text-white text-sm font-mono">{{ newlyCreatedKey }}</code>
          </div>
          <button @click="copyToClipboard(newlyCreatedKey)" class="text-green-400 hover:text-green-300 text-xs">复制</button>
        </div>
      </div>
      <div class="mt-2 flex justify-end">
        <button @click="copyKeyConfig(newlyCreatedKey)" class="text-primary-400 hover:text-primary-300 text-xs mr-3">复制全部</button>
        <button @click="newlyCreatedKey = ''" class="text-dark-400 hover:text-white text-xs">关闭</button>
      </div>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <div v-else-if="apiKeys.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      暂无 API 密钥，点击"生成密钥"创建一个。
    </div>

    <div v-else class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="text-dark-400 text-sm border-b border-dark-700">
            <th class="text-left px-4 py-3">名称</th>
            <th class="text-left px-4 py-3">密钥</th>
            <th class="text-left px-4 py-3">分组</th>
            <th class="text-left px-4 py-3">状态</th>
            <th class="text-left px-4 py-3">最后使用</th>
            <th class="text-right px-4 py-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="k in apiKeys" :key="k.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
            <td class="px-4 py-3 text-white text-sm">{{ k.name }}</td>
            <td class="px-4 py-3 text-white text-sm font-mono">{{ (k.key || '').substring(0, 12) }}...</td>
            <td class="px-4 py-3 text-white text-sm">{{ groupName(k.group_id) }}</td>
            <td class="px-4 py-3">
              <span :class="k.status === 'active' ? 'text-green-400' : 'text-dark-400'" class="text-sm">{{ k.status === 'active' ? '正常' : '禁用' }}</span>
            </td>
            <td class="px-4 py-3 text-dark-400 text-sm">{{ k.last_used_at || '从未' }}</td>
            <td class="px-4 py-3 text-right space-x-2">
              <button @click="copyKeyConfig(k.key)" class="text-primary-400 hover:text-primary-300 text-sm" title="复制 Base URL + Key">复制配置</button>
              <button @click="copyToClipboard(k.key)" class="text-dark-400 hover:text-white text-sm">仅Key</button>
              <button @click="remove(k.id)" class="text-red-400 hover:text-red-300 text-sm">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-md p-6">
        <h2 class="text-lg font-semibold text-white mb-4">生成 API 密钥</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">名称</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="密钥名称" />
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">绑定分组（可选）</label>
            <select v-model="form.group_id" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
              <option :value="null">无分组</option>
              <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>

          <!-- Base URL 预览 -->
          <div class="bg-dark-900 rounded-lg p-3 border border-dark-700">
            <div class="text-dark-400 text-xs mb-1">生成后的连接信息</div>
            <div class="text-dark-300 text-xs">Base URL: <code class="text-primary-400">{{ baseUrl }}</code></div>
            <div class="text-dark-300 text-xs">API Key: <code class="text-dark-500">生成后显示</code></div>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button @click="showModal = false" class="px-4 py-2 text-dark-300 hover:text-white text-sm transition-colors">取消</button>
          <button @click="save" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
            生成
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
