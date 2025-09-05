<script setup lang="ts">
import { ref, onMounted } from "vue";
import { apiClient } from "@/services/api";
import { CheckCircle, AlertTriangle, Zap, Plus, RefreshCw, X } from "lucide-vue-next";

const status = ref<any>(null);
const proxies = ref<any[]>([]);
const loading = ref(false);
const error = ref("");

const loadData = async () => {
  loading.value = true;
  error.value = "";

  try {
    const [statusResponse, proxiesResponse] = await Promise.all([
      apiClient.getStatus(),
      apiClient.getProxies(),
    ]);

    if (statusResponse.error) {
      error.value = statusResponse.error;
    } else {
      status.value = statusResponse.data;
    }

    if (proxiesResponse.error) {
      error.value = proxiesResponse.error;
      proxies.value = [];
    } else if (proxiesResponse.data) {
      proxies.value = proxiesResponse.data.proxies || [];
    } else {
      proxies.value = [];
    }
  } catch (err) {
    error.value = "Failed to load dashboard data";
  }

  loading.value = false;
};

onMounted(() => {
  loadData();
});
</script>

<template>
  <div>
    <!-- Error Alert -->
    <div v-if="error" class="alert alert-error mb-4">
      <AlertTriangle class="stroke-current shrink-0 h-6 w-6" />
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="error = ''">
        <X class="h-4 w-4" />
      </button>
    </div>

    <div class="mb-8">
      <h1 class="text-3xl font-bold text-base-content">Dashboard</h1>
      <p class="text-base-content/70 mt-2">Welcome to Caddy Proxy Manager</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div class="card bg-base-200 shadow-xl">
        <div class="card-body">
          <h2 class="card-title text-primary">
            <CheckCircle class="h-6 w-6" />
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
            <AlertTriangle class="h-6 w-6" />
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
            <Zap class="h-6 w-6" />
            Caddy Status
          </h2>
          <p v-if="loading" class="text-lg font-semibold">
            <span class="loading loading-spinner loading-sm"></span>
            Checking...
          </p>
          <p v-else-if="status?.caddy_reachable" class="text-lg font-semibold text-success">
            {{ status.caddy_status }}
          </p>
          <p v-else class="text-lg font-semibold text-error">Unreachable</p>
          <p class="text-sm text-base-content/70">
            Last checked:
            {{
              status?.last_checked ? new Date(status.last_checked).toLocaleTimeString() : "Never"
            }}
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
              <Plus class="h-5 w-5" />
              Add New Proxy
            </RouterLink>
            <button class="btn btn-outline" @click="loadData" :disabled="loading">
              <span v-if="loading" class="loading loading-spinner loading-sm"></span>
              <RefreshCw v-else class="h-5 w-5" />
              Refresh Data
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
