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
    <h1 class="text-2xl font-bold text-white mb-6">Settings</h1>

    <div v-if="loading" class="text-dark-400">Loading...</div>

    <template v-else-if="cfg">
      <div class="space-y-6 max-w-2xl">
        <!-- Server -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">Server</h2>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-400 text-sm mb-1">Host</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Server?.Host || '127.0.0.1' }}</div>
            </div>
            <div>
              <label class="block text-dark-400 text-sm mb-1">Port</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Server?.Port || 8787 }}</div>
            </div>
          </div>
          <p class="text-dark-500 text-xs mt-2">Edit config.yaml to change server settings. Restart required.</p>
        </div>

        <!-- Gateway -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">Gateway</h2>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-dark-400 text-sm mb-1">Max Body Size (bytes)</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Gateway?.MaxBodySize || 10485760 }}</div>
            </div>
            <div>
              <label class="block text-dark-400 text-sm mb-1">Max Account Retries</label>
              <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Gateway?.MaxAccountRetries || 3 }}</div>
            </div>
          </div>
        </div>

        <!-- Database -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">Database</h2>
          <div>
            <label class="block text-dark-400 text-sm mb-1">Database Path</label>
            <div class="text-white text-sm bg-dark-900 rounded px-3 py-2 border border-dark-700">{{ cfg.Database?.Path || 'data.db' }}</div>
          </div>
        </div>

        <!-- Proxy API Endpoint -->
        <div class="bg-dark-800 rounded-lg border border-dark-700 p-4">
          <h2 class="text-white font-semibold mb-3">API Endpoint</h2>
          <div class="bg-dark-900 rounded px-3 py-2 border border-dark-700">
            <code class="text-primary-400 text-sm">http://{{ cfg.Server?.Host || '127.0.0.1' }}:{{ cfg.Server?.Port || 8787 }}</code>
          </div>
          <p class="text-dark-500 text-xs mt-2">
            Use this endpoint with your generated API keys to proxy requests to Claude/OpenAI/Gemini.
          </p>
          <div class="mt-3 space-y-1 text-dark-400 text-xs font-mono">
            <div>POST /v1/messages - Claude Messages API</div>
            <div>POST /v1/chat/completions - OpenAI Chat Completions</div>
            <div>POST /v1/responses - OpenAI Responses API</div>
            <div>POST /v1beta/models/* - Gemini Native API</div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
