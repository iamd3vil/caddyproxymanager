<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface AuditLogEntry {
  timestamp: string
  action: string
  details: string
  user_id?: string
  username?: string
  ip_address?: string
}

const auditLogs = ref<AuditLogEntry[]>([])
const loading = ref(false)
const error = ref('')

const loadAuditLogs = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const response = await fetch('/api/audit-log', {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('auth_token') || ''}`
      }
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    const data = await response.json()
    auditLogs.value = data.entries || []
  } catch (err) {
    error.value = 'Failed to load audit logs'
    console.error('Audit log load error:', err)
  } finally {
    loading.value = false
  }
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

const getActionBadgeClass = (action: string) => {
  if (action.includes('LOGIN')) return 'badge-success'
  if (action.includes('LOGOUT')) return 'badge-info'
  if (action.includes('SETUP')) return 'badge-warning'
  if (action.includes('CREATE')) return 'badge-primary'
  if (action.includes('UPDATE')) return 'badge-secondary'
  if (action.includes('DELETE')) return 'badge-error'
  return 'badge-ghost'
}

onMounted(() => {
  loadAuditLogs()
})
</script>

<template>
  <div>
    <div class="mb-8 flex justify-between items-center">
      <div>
        <h1 class="text-3xl font-bold text-base-content">Audit Log</h1>
        <p class="text-base-content/70 mt-2">View system activity and user actions</p>
      </div>
      <button class="btn btn-primary" @click="loadAuditLogs" :disabled="loading">
        <span v-if="loading" class="loading loading-spinner loading-sm"></span>
        Refresh
      </button>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="alert alert-error mb-4">
      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="error = ''">✕</button>
    </div>

    <!-- Loading state -->
    <div v-if="loading && auditLogs.length === 0" class="text-center py-16">
      <span class="loading loading-spinner loading-lg"></span>
      <p class="text-base-content/70 mt-4">Loading audit logs...</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="auditLogs.length === 0" class="text-center py-16">
      <h3 class="text-xl font-semibold mt-6 text-base-content">No audit logs found</h3>
      <p class="text-base-content/70 mt-2">There are no recorded actions in the system yet</p>
    </div>

    <!-- Audit log table -->
    <div v-else class="overflow-x-auto">
      <table class="table table-zebra">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Action</th>
            <th>Details</th>
            <th>User</th>
            <th>IP Address</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(log, index) in auditLogs" :key="index">
            <td>{{ formatDate(log.timestamp) }}</td>
            <td>
              <div class="badge" :class="getActionBadgeClass(log.action)">
                {{ log.action }}
              </div>
            </td>
            <td>{{ log.details }}</td>
            <td>{{ log.username || log.user_id || 'N/A' }}</td>
            <td>{{ log.ip_address || 'N/A' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>