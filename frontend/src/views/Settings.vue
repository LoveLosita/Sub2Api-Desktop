<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { GetConfig } from '../../wailsjs/go/main/App'
import { config as configNS } from '../../wailsjs/go/models'

const cfg = ref<configNS.Config | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    cfg.value = await GetConfig()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-white mb-6">系统设置</h1>

    <div v-if="loading" class="text-dark-400">加载中...</div>

    <template v-else-if="cfg">
      <div class="space-y-6 max-w-2xl">
        <!-- 服务器 -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">服务器</h2>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-400 text-sm mb-1">主机地址</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Server?.Host || '127.0.0.1' }}</div>
            </div>
            <div>
              <label class="block text-dark-400 text-sm mb-1">端口</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Server?.Port || 8787 }}</div>
            </div>
          </div>
          <p class="text-dark-500 text-xs mt-2">修改 config.yaml 更改服务器设置，需要重启。</p>
        </div>

        <!-- 网关 -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">网关</h2>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-400 text-sm mb-1">最大请求体 (字节)</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Gateway?.MaxBodySize || 10485760 }}</div>
            </div>
            <div>
              <label class="block text-dark-400 text-sm mb-1">最大账号重试</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Gateway?.MaxAccountRetries || 3 }}</div>
            </div>
          </div>
        </div>

        <!-- 数据库 -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">数据库</h2>
          <div>
            <label class="block text-dark-400 text-sm mb-1">数据库路径</label>
            <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Database?.Path || 'data.db' }}</div>
          </div>
        </div>

        <!-- API 端点 -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">API 端点</h2>
          <div class="bg-dark-900 rounded px-3 py-2 border border-dark-700">
            <code class="text-primary-400 text-sm">http://{{ cfg.Server?.Host || '127.0.0.1' }}:{{ cfg.Server?.Port || 8787 }}</code>
          </div>
          <p class="text-dark-500 text-xs mt-2">
            使用此端点和生成的 API 密钥代理请求到 Claude/OpenAI/Gemini。
          </p>
          <div class="mt-3 space-y-1 text-dark-400 text-xs font-mono">
            <div>POST /v1/messages - Claude Messages API</div>
            <div>POST /v1/chat/completions - OpenAI Chat Completions</div>
            <div>POST /v1/responses - OpenAI Responses API</div>
            <div>POST /v1beta/models/* - Gemini 原生 API</div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
