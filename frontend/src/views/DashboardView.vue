<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiClient } from '@/services/api'

const status = ref<any>(null)
const proxies = ref<any[]>([])
const loading = ref(false)
const error = ref('')

const loadData = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const [statusResponse, proxiesResponse] = await Promise.all([
      apiClient.getStatus(),
      apiClient.getProxies()
    ])
    
    if (statusResponse.error) {
      error.value = statusResponse.error
    } else {
      status.value = statusResponse.data
    }
    
    if (proxiesResponse.error) {
      error.value = proxiesResponse.error
      proxies.value = []
    } else if (proxiesResponse.data) {
      proxies.value = proxiesResponse.data.proxies || []
    } else {
      proxies.value = []
    }
  } catch (err) {
    error.value = 'Failed to load dashboard data'
  }
  
  loading.value = false
}


onMounted(() => {
  loadData()
})
</script>

<template>
  <div>
    <!-- Error Alert -->
    <div v-if="error" class="alert alert-error mb-4">
      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="error = ''">✕</button>
    </div>

    <div class="mb-8">
      <h1 class="text-3xl font-bold text-base-content">Dashboard</h1>
      <p class="text-base-content/70 mt-2">Welcome to Caddy Proxy Manager</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div class="card bg-base-200 shadow-xl">
        <div class="card-body">
          <h2 class="card-title text-primary">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Active Proxies
          </h2>
          <p class="text-3xl font-bold text-base-content">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            <span v-else>{{ proxies?.length || 0 }}</span>
          </p>
          <p class="text-sm text-base-content/70">Currently running</p>
        </div>
      </div>

      <div class="card bg-base-200 shadow-xl">
        <div class="card-body">
          <h2 class="card-title text-warning">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            Health Issues
          </h2>
          <p class="text-3xl font-bold text-base-content">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            <span v-else>{{ error ? 1 : 0 }}</span>
          </p>
          <p class="text-sm text-base-content/70">Needs attention</p>
        </div>
      </div>

      <div class="card bg-base-200 shadow-xl">
        <div class="card-body">
          <h2 class="card-title text-info">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            Caddy Status
          </h2>
          <p v-if="loading" class="text-lg font-semibold">
            <span class="loading loading-spinner loading-sm"></span>
            Checking...
          </p>
          <p v-else-if="status?.caddy_reachable" class="text-lg font-semibold text-success">
            {{ status.caddy_status }}
          </p>
          <p v-else class="text-lg font-semibold text-error">
            Unreachable
          </p>
          <p class="text-sm text-base-content/70">
            Last checked: {{ status?.last_checked ? new Date(status.last_checked).toLocaleTimeString() : 'Never' }}
          </p>
        </div>
      </div>
    </div>

    <div class="mt-8">
      <div class="card bg-base-200 shadow-xl">
        <div class="card-body">
          <h2 class="card-title">Quick Actions</h2>
          <div class="card-actions justify-start mt-4">
            <RouterLink to="/proxies" class="btn btn-primary">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Add New Proxy
            </RouterLink>
            <button class="btn btn-outline" @click="loadData" :disabled="loading">
              <span v-if="loading" class="loading loading-spinner loading-sm"></span>
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Refresh Data
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>