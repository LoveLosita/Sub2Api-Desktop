<script lang="ts" setup>
import { ref, onMounted } from 'vue'

const version = ref('0.1.0')

// eslint-disable-next-line @typescript-eslint/no-explicit-any
declare const window: any
onMounted(async () => {
  try {
    if (window['go']?.main?.App?.GetAppVersion) {
      version.value = await window.go.main.App.GetAppVersion()
    }
  } catch { /* ignore */ }
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-white mb-6">Dashboard</h1>
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
        <div class="text-dark-400 text-sm">Accounts</div>
        <div class="text-2xl font-bold text-white mt-1">--</div>
      </div>
      <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
        <div class="text-dark-400 text-sm">Today Requests</div>
        <div class="text-2xl font-bold text-white mt-1">--</div>
      </div>
      <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
        <div class="text-dark-400 text-sm">Today Tokens</div>
        <div class="text-2xl font-bold text-white mt-1">--</div>
      </div>
      <div class="bg-dark-800 rounded-lg p-4 border border-dark-700">
        <div class="text-dark-400 text-sm">Today Cost</div>
        <div class="text-2xl font-bold text-primary-400 mt-1">$0.00</div>
      </div>
    </div>
    <div class="bg-dark-800 rounded-lg p-6 border border-dark-700">
      <p class="text-dark-400">Dashboard charts will be added in Phase 7.</p>
    </div>
    <p class="text-dark-500 text-xs mt-4">Desktop Proxy v{{ version }}</p>
  </div>
</template>
