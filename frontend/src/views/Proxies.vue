<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListProxies, CreateProxy, UpdateProxy, DeleteProxy } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const proxies = ref<model.Proxy[]>([])
const loading = ref(true)
const showModal = ref(false)
const editing = ref(false)
const form = ref(emptyForm())

function emptyForm() {
  return {
    id: 0,
    name: '',
    protocol: 'http',
    host: '',
    port: 7890,
    username: '',
    password: '',
    status: 'active',
  }
}

const protocols = ['http', 'https', 'socks5']

onMounted(load)

async function load() {
  loading.value = true
  try {
    proxies.value = await ListProxies() ?? []
  } catch (e) {
    console.error(e)
    proxies.value = []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  form.value = emptyForm()
  showModal.value = true
}

function openEdit(p: model.Proxy) {
  editing.value = true
  form.value = {
    id: p.id,
    name: p.name,
    protocol: p.protocol,
    host: p.host,
    port: p.port,
    username: p.username ?? '',
    password: p.password ?? '',
    status: p.status,
  }
  showModal.value = true
}

async function save() {
  const obj: any = { ...form.value }
  if (!obj.username) obj.username = null
  if (!obj.password) obj.password = null
  if (!editing.value) delete obj.id
  try {
    if (editing.value) {
      await UpdateProxy(obj)
    } else {
      await CreateProxy(obj)
    }
    showModal.value = false
    await load()
  } catch (e) {
    alert('保存失败: ' + e)
  }
}

async function remove(id: number) {
  if (!confirm('确定删除此代理？')) return
  try {
    await DeleteProxy(id)
    await load()
  } catch (e) {
    alert('删除失败: ' + e)
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-white">代理管理</h1>
      <button @click="openCreate" class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors">
        + 添加代理
      </button>
    </div>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <div v-else-if="proxies.length === 0" class="bg-dark-800 rounded-lg p-6 border border-dark-700 text-dark-400 text-center">
      暂无代理配置，点击"添加代理"添加一个。
    </div>

    <div v-else class="bg-dark-800 rounded-lg border border-dark-700 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="text-dark-400 text-sm border-b border-dark-700">
            <th class="text-left px-4 py-3">名称</th>
            <th class="text-left px-4 py-3">协议</th>
            <th class="text-left px-4 py-3">主机</th>
            <th class="text-center px-4 py-3">端口</th>
            <th class="text-left px-4 py-3">认证</th>
            <th class="text-left px-4 py-3">状态</th>
            <th class="text-right px-4 py-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in proxies" :key="p.id" class="border-b border-dark-700/50 hover:bg-dark-700/30">
            <td class="px-4 py-3 text-white text-sm">{{ p.name }}</td>
            <td class="px-4 py-3">
              <span class="text-xs px-2 py-0.5 rounded bg-dark-700 text-dark-300">{{ p.protocol }}</span>
            </td>
            <td class="px-4 py-3 text-white text-sm font-mono">{{ p.host }}</td>
            <td class="px-4 py-3 text-white text-sm text-center">{{ p.port }}</td>
            <td class="px-4 py-3 text-dark-400 text-sm">{{ p.username ? '是' : '否' }}</td>
            <td class="px-4 py-3">
              <span :class="p.status === 'active' ? 'text-green-400' : 'text-dark-400'" class="text-sm">{{ p.status === 'active' ? '正常' : '禁用' }}</span>
            </td>
            <td class="px-4 py-3 text-right space-x-2">
              <button @click="openEdit(p)" class="text-primary-400 hover:text-primary-300 text-sm">编辑</button>
              <button @click="remove(p.id)" class="text-red-400 hover:text-red-300 text-sm">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-dark-800 rounded-xl border border-dark-700 w-full max-w-md p-6">
        <h2 class="text-lg font-semibold text-white mb-4">{{ editing ? '编辑代理' : '添加代理' }}</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-dark-300 text-sm mb-1">名称</label>
            <input v-model="form.name" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="代理名称" />
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">协议</label>
            <select v-model="form.protocol" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
              <option v-for="p in protocols" :key="p" :value="p">{{ p }}</option>
            </select>
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="block text-dark-300 text-sm mb-1">主机</label>
              <input v-model="form.host" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm font-mono focus:border-primary-500 outline-none" placeholder="127.0.0.1" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">端口</label>
              <input v-model.number="form.port" type="number" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-dark-300 text-sm mb-1">用户名</label>
              <input v-model="form.username" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="可选" />
            </div>
            <div>
              <label class="block text-dark-300 text-sm mb-1">密码</label>
              <input v-model="form.password" type="password" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none" placeholder="可选" />
            </div>
          </div>

          <div>
            <label class="block text-dark-300 text-sm mb-1">状态</label>
            <select v-model="form.status" class="w-full bg-dark-900 border border-dark-600 rounded-lg px-3 py-2 text-white text-sm focus:border-primary-500 outline-none">
              <option value="active">正常</option>
              <option value="disabled">禁用</option>
            </select>
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
