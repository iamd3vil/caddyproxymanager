<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiClient, type Proxy } from '@/services/api'

const proxies = ref<Proxy[]>([])
const showAddModal = ref(false)
const loading = ref(false)
const error = ref('')

// Form data
const formData = ref({
  domain: '',
  target_url: '',
  ssl_mode: 'auto',
  challenge_type: 'http',
  dns_provider: 'cloudflare',
  dns_credentials: {} as Record<string, string>
})

const loadProxies = async () => {
  loading.value = true
  error.value = ''
  
  const response = await apiClient.getProxies()
  if (response.error) {
    error.value = response.error
    proxies.value = []
  } else if (response.data) {
    proxies.value = response.data.proxies || []
  } else {
    proxies.value = []
  }
  
  loading.value = false
}

const addProxy = async () => {
  if (!formData.value.domain || !formData.value.target_url) {
    error.value = 'Domain and target URL are required'
    return
  }
  
  loading.value = true
  error.value = ''
  
  const response = await apiClient.createProxy(formData.value)
  if (response.error) {
    error.value = response.error
  } else {
    showAddModal.value = false
    formData.value = { 
      domain: '', 
      target_url: '', 
      ssl_mode: 'auto', 
      challenge_type: 'http', 
      dns_provider: 'cloudflare', 
      dns_credentials: {} as Record<string, string> 
    }
    await loadProxies()
  }
  
  loading.value = false
}

const deleteProxy = async (proxy: Proxy) => {
  const proxyName = getProxyName(proxy)
  
  // Check if proxy has a valid ID
  if (!proxy.id || proxy.id.trim() === '') {
    error.value = 'Cannot delete proxy: Missing proxy ID. This proxy may need to be recreated.'
    return
  }
  
  if (!confirm(`Are you sure you want to delete proxy for ${proxyName}?`)) {
    return
  }
  
  loading.value = true
  error.value = ''
  
  const response = await apiClient.deleteProxy(proxy.id)
  if (response.error) {
    error.value = response.error
  } else {
    await loadProxies()
  }
  
  loading.value = false
}

// Helper functions
const getProxyName = (proxy: Proxy): string => {
  if (proxy.domain && proxy.domain.trim()) {
    return proxy.domain
  }
  // Extract hostname from target_url as fallback
  try {
    const url = new URL(proxy.target_url)
    return `${url.hostname}:${url.port || (url.protocol === 'https:' ? '443' : '80')}`
  } catch {
    return proxy.target_url
  }
}

const getSSLMode = (proxy: Proxy): string => {
  if (proxy.ssl_mode && proxy.ssl_mode.trim()) {
    return proxy.ssl_mode
  }
  // Default based on target URL
  return proxy.target_url.startsWith('https:') ? 'auto' : 'none'
}


const getProxyType = (proxy: Proxy): string => {
  try {
    const url = new URL(proxy.target_url)
    const protocol = url.protocol.replace(':', '')
    return protocol.toUpperCase()
  } catch {
    return 'HTTP'
  }
}

// DNS challenge handling
const updateDNSCredentials = () => {
  const provider = formData.value.dns_provider
  formData.value.dns_credentials = {}
  
  // Set default credential structure based on provider
  if (provider === 'cloudflare') {
    formData.value.dns_credentials = { api_token: '', email: '' }
  } else if (provider === 'digitalocean') {
    formData.value.dns_credentials = { auth_token: '' }
  } else if (provider === 'duckdns') {
    formData.value.dns_credentials = { token: '' }
  }
}

// DNS provider configurations
const dnsProviders = [
  { 
    value: 'cloudflare', 
    label: 'Cloudflare',
    fields: [
      { key: 'api_token', label: 'API Token', type: 'password', required: true },
      { key: 'email', label: 'Email (optional)', type: 'email', required: false }
    ]
  },
  { 
    value: 'digitalocean', 
    label: 'DigitalOcean',
    fields: [
      { key: 'auth_token', label: 'Auth Token', type: 'password', required: true }
    ]
  },
  { 
    value: 'duckdns', 
    label: 'DuckDNS',
    fields: [
      { key: 'token', label: 'Token', type: 'password', required: true }
    ]
  }
]

onMounted(() => {
  loadProxies()
  updateDNSCredentials()
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

    <div class="mb-8 flex justify-between items-center">
      <div>
        <h1 class="text-3xl font-bold text-base-content">Proxy Management</h1>
        <p class="text-base-content/70 mt-2">Manage your Caddy proxy configurations</p>
      </div>
      <button class="btn btn-primary" @click="showAddModal = true" :disabled="loading">
        <span v-if="loading" class="loading loading-spinner loading-sm"></span>
        <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
        Add Proxy
      </button>
    </div>

    <!-- Loading state -->
    <div v-if="loading && (proxies?.length || 0) === 0" class="text-center py-16">
      <span class="loading loading-spinner loading-lg"></span>
      <p class="text-base-content/70 mt-4">Loading proxies...</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="(proxies?.length || 0) === 0" class="text-center py-16">
      <h3 class="text-xl font-semibold mt-6 text-base-content">No proxies configured</h3>
      <p class="text-base-content/70 mt-2">Get started by adding your first proxy configuration</p>
      <button class="btn btn-primary mt-4" @click="showAddModal = true">
        Add Your First Proxy
      </button>
    </div>

    <!-- Proxy list -->
    <div v-else class="grid gap-4">
      <div v-for="proxy in proxies" :key="proxy.id || proxy.target_url" class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <h2 class="card-title text-primary mb-3">
                {{ getProxyName(proxy) }}
              </h2>
              
              <!-- Proxy Route Visualization -->
              <div class="flex items-center gap-3 mb-3 p-3 bg-base-200 rounded-lg">
                <div class="text-sm">
                  <div class="font-semibold text-base-content">From:</div>
                  <div class="text-primary font-mono">{{ getProxyName(proxy) }}</div>
                </div>
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-base-content/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5-5 5M6 12h12" />
                </svg>
                <div class="text-sm">
                  <div class="font-semibold text-base-content">To:</div>
                  <div class="text-accent font-mono">{{ proxy.target_url }}</div>
                </div>
              </div>

              <div class="flex gap-2 flex-wrap">
                <div class="badge badge-secondary">{{ getSSLMode(proxy) }}</div>
                <div v-if="proxy.status" class="badge badge-success">{{ proxy.status }}</div>
                <div class="badge badge-outline">{{ getProxyType(proxy) }}</div>
              </div>
            </div>
            <div class="card-actions">
              <button 
                class="btn btn-error btn-sm" 
                @click="deleteProxy(proxy)" 
                :disabled="loading || !proxy.id || proxy.id.trim() === ''"
                :title="!proxy.id || proxy.id.trim() === '' ? 'Cannot delete: Missing proxy ID' : 'Delete proxy'">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Proxy Modal -->
    <div v-if="showAddModal" class="modal modal-open">
      <div class="modal-box">
        <h3 class="font-bold text-lg text-base-content">Add New Proxy</h3>
        <form @submit.prevent="addProxy" class="py-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text">Domain/Subdomain</span>
            </label>
            <input 
              v-model="formData.domain"
              type="text" 
              placeholder="example.com" 
              class="input input-bordered" 
              required
            />
          </div>
          
          <div class="form-control mt-4">
            <label class="label">
              <span class="label-text">Target URL</span>
            </label>
            <input 
              v-model="formData.target_url"
              type="text" 
              placeholder="http://localhost:3000" 
              class="input input-bordered" 
              required
            />
          </div>

          <div class="form-control mt-4">
            <label class="label">
              <span class="label-text">SSL Certificate</span>
            </label>
            <select v-model="formData.ssl_mode" class="select select-bordered">
              <option value="auto">Auto (Let's Encrypt)</option>
              <option value="custom">Custom Certificate</option>
              <option value="none">None (HTTP only)</option>
            </select>
          </div>

          <!-- ACME Challenge Configuration (only shown when SSL is auto) -->
          <div v-if="formData.ssl_mode === 'auto'" class="mt-6 p-4 bg-base-200 rounded-lg">
            <h4 class="font-semibold text-base-content mb-4">ACME Challenge Configuration</h4>
            
            <div class="form-control">
              <label class="label">
                <span class="label-text">Challenge Type</span>
              </label>
              <select v-model="formData.challenge_type" class="select select-bordered">
                <option value="http">HTTP-01 Challenge</option>
                <option value="dns">DNS-01 Challenge</option>
              </select>
              <div class="label">
                <span class="label-text-alt">
                  {{ formData.challenge_type === 'http' 
                      ? 'Uses HTTP validation (port 80 must be accessible)' 
                      : 'Uses DNS validation (works behind firewalls)' }}
                </span>
              </div>
            </div>

            <!-- DNS Challenge Configuration -->
            <div v-if="formData.challenge_type === 'dns'" class="mt-4">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">DNS Provider</span>
                </label>
                <select 
                  v-model="formData.dns_provider" 
                  @change="updateDNSCredentials()"
                  class="select select-bordered"
                >
                  <option v-for="provider in dnsProviders" :key="provider.value" :value="provider.value">
                    {{ provider.label }}
                  </option>
                </select>
              </div>

              <!-- DNS Provider Credentials -->
              <div class="mt-4">
                <h5 class="font-medium text-base-content mb-2">DNS Provider Credentials</h5>
                <div v-for="field in dnsProviders.find(p => p.value === formData.dns_provider)?.fields" :key="field.key" class="form-control mt-2">
                  <label class="label">
                    <span class="label-text">{{ field.label }}</span>
                  </label>
                  <input 
                    v-model="formData.dns_credentials[field.key]"
                    :type="field.type"
                    class="input input-bordered input-sm"
                    :required="field.required && formData.challenge_type === 'dns'"
                    :placeholder="field.type === 'password' ? '••••••••••••••••' : ''"
                  />
                </div>
                
                <!-- Help text for Cloudflare -->
                <div v-if="formData.dns_provider === 'cloudflare'" class="alert alert-info mt-2">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" class="stroke-current shrink-0 w-6 h-6">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                  </svg>
                  <div class="text-xs">
                    <p><strong>API Token:</strong> Create a token with Zone:DNS:Edit permissions</p>
                    <p><strong>Email:</strong> Only needed for legacy API key authentication</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </form>
        
        <div class="modal-action">
          <button class="btn" @click="showAddModal = false" :disabled="loading">Cancel</button>
          <button class="btn btn-primary" @click="addProxy" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            Add Proxy
          </button>
        </div>
      </div>
    </div>
  </div>
</template>