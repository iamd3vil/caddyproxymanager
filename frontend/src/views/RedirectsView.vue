<script setup lang="ts">
import { ref, onMounted } from "vue";
import { apiClient, type Redirect } from "../services/api";

const redirects = ref<Redirect[]>([]);
const loading = ref(false);
const error = ref("");
const showCreateModal = ref(false);
const showEditModal = ref(false);
const showDeleteModal = ref(false);
const editingRedirect = ref<Redirect | null>(null);

// Form data
const form = ref({
  source_domains: "",
  destination_url: "",
  redirect_code: 301,
  preserve_path: false,
});

// Validation
const formErrors = ref({
  source_domains: "",
  destination_url: "",
});

const validateForm = () => {
  formErrors.value = {
    source_domains: "",
    destination_url: "",
  };

  let isValid = true;

  if (!form.value.source_domains.trim()) {
    formErrors.value.source_domains = "Source domains are required";
    isValid = false;
  }

  if (!form.value.destination_url.trim()) {
    formErrors.value.destination_url = "Destination URL is required";
    isValid = false;
  } else if (!form.value.destination_url.match(/^https?:\/\/.+/)) {
    formErrors.value.destination_url = "Destination URL must start with http:// or https://";
    isValid = false;
  }

  return isValid;
};

const resetForm = () => {
  form.value = {
    source_domains: "",
    destination_url: "",
    redirect_code: 301,
    preserve_path: false,
  };
  formErrors.value = {
    source_domains: "",
    destination_url: "",
  };
};

const loadRedirects = async () => {
  loading.value = true;
  error.value = "";

  try {
    const response = await apiClient.getRedirects();
    if (response.error) {
      error.value = response.error;
    } else if (response.data) {
      redirects.value = response.data.redirects || [];
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load redirects";
  } finally {
    loading.value = false;
  }
};

const openCreateModal = () => {
  resetForm();
  showCreateModal.value = true;
};

const openEditModal = (redirect: Redirect) => {
  editingRedirect.value = redirect;
  form.value = {
    source_domains: redirect.source_domains.join(", "),
    destination_url: redirect.destination_url,
    redirect_code: redirect.redirect_code,
    preserve_path: redirect.preserve_path,
  };
  formErrors.value = {
    source_domains: "",
    destination_url: "",
  };
  showEditModal.value = true;
};

const openDeleteModal = (redirect: Redirect) => {
  editingRedirect.value = redirect;
  showDeleteModal.value = true;
};

const closeModals = () => {
  showCreateModal.value = false;
  showEditModal.value = false;
  showDeleteModal.value = false;
  editingRedirect.value = null;
  resetForm();
};

const createRedirect = async () => {
  if (!validateForm()) return;

  loading.value = true;
  try {
    const sourceDomains = form.value.source_domains
      .split(",")
      .map((domain) => domain.trim())
      .filter((domain) => domain.length > 0);

    const response = await apiClient.createRedirect({
      source_domains: sourceDomains,
      destination_url: form.value.destination_url,
      redirect_code: form.value.redirect_code,
      preserve_path: form.value.preserve_path,
    });

    if (response.error) {
      error.value = response.error;
    } else {
      closeModals();
      await loadRedirects();
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to create redirect";
  } finally {
    loading.value = false;
  }
};

const updateRedirect = async () => {
  if (!validateForm() || !editingRedirect.value) return;

  loading.value = true;
  try {
    const sourceDomains = form.value.source_domains
      .split(",")
      .map((domain) => domain.trim())
      .filter((domain) => domain.length > 0);

    const response = await apiClient.updateRedirect(editingRedirect.value.id, {
      source_domains: sourceDomains,
      destination_url: form.value.destination_url,
      redirect_code: form.value.redirect_code,
      preserve_path: form.value.preserve_path,
    });

    if (response.error) {
      error.value = response.error;
    } else {
      closeModals();
      await loadRedirects();
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to update redirect";
  } finally {
    loading.value = false;
  }
};

const deleteRedirect = async () => {
  if (!editingRedirect.value) return;

  loading.value = true;
  try {
    const response = await apiClient.deleteRedirect(editingRedirect.value.id);
    if (response.error) {
      error.value = response.error;
    } else {
      closeModals();
      await loadRedirects();
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to delete redirect";
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadRedirects();
});
</script>

<template>
  <div class="container mx-auto p-6">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-3xl font-bold text-base-content">Redirects</h1>
        <p class="text-base-content/60">Manage HTTP redirects</p>
      </div>
      <button @click="openCreateModal" class="btn btn-primary">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4v16m8-8H4"
          />
        </svg>
        Create Redirect
      </button>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="alert alert-error mb-4">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-6 w-6"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.502 0L4.268 15.5c-.77.833.192 2.5 1.732 2.5z"
        />
      </svg>
      <span>{{ error }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center items-center py-12">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <!-- Redirects Table -->
    <div v-else class="overflow-x-auto">
      <table class="table table-zebra w-full">
        <thead>
          <tr>
            <th>Source Domains</th>
            <th>Destination URL</th>
            <th>Type</th>
            <th>Preserve Path</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!redirects || redirects.length === 0">
            <td colspan="6" class="text-center py-8 text-base-content/60">
              No redirects configured yet
            </td>
          </tr>
          <tr v-for="redirect in redirects || []" :key="redirect.id">
            <td>
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="domain in redirect.source_domains"
                  :key="domain"
                  class="badge badge-outline"
                >
                  {{ domain }}
                </span>
              </div>
            </td>
            <td>
              <a
                :href="redirect.destination_url"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary hover:underline"
              >
                {{ redirect.destination_url }}
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 inline ml-1"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                  />
                </svg>
              </a>
            </td>
            <td>
              <span
                :class="redirect.redirect_code === 301 ? 'badge-success' : 'badge-warning'"
                class="badge"
              >
                {{ redirect.redirect_code }}
                {{ redirect.redirect_code === 301 ? "Permanent" : "Temporary" }}
              </span>
            </td>
            <td>
              <span :class="redirect.preserve_path ? 'badge-success' : 'badge-ghost'" class="badge">
                {{ redirect.preserve_path ? "Yes" : "No" }}
              </span>
            </td>
            <td>
              <span
                :class="redirect.status === 'active' ? 'badge-success' : 'badge-error'"
                class="badge"
              >
                {{ redirect.status || "active" }}
              </span>
            </td>
            <td>
              <div class="flex gap-2">
                <button @click="openEditModal(redirect)" class="btn btn-sm btn-ghost">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>
                <button @click="openDeleteModal(redirect)" class="btn btn-sm btn-ghost text-error">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="modal modal-open">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Create Redirect</h3>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Source Domains</span>
          </label>
          <input
            v-model="form.source_domains"
            type="text"
            placeholder="example.com, www.example.com"
            class="input input-bordered"
            :class="{ 'input-error': formErrors.source_domains }"
          />
          <label class="label">
            <span class="label-text-alt text-base-content/60">
              Comma-separated list of source domains
            </span>
          </label>
          <label v-if="formErrors.source_domains" class="label">
            <span class="label-text-alt text-error">{{ formErrors.source_domains }}</span>
          </label>
        </div>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Destination URL</span>
          </label>
          <input
            v-model="form.destination_url"
            type="url"
            placeholder="https://newsite.com"
            class="input input-bordered"
            :class="{ 'input-error': formErrors.destination_url }"
          />
          <label v-if="formErrors.destination_url" class="label">
            <span class="label-text-alt text-error">{{ formErrors.destination_url }}</span>
          </label>
        </div>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Redirect Type</span>
          </label>
          <select v-model="form.redirect_code" class="select select-bordered">
            <option :value="301">301 - Permanent Redirect</option>
            <option :value="302">302 - Temporary Redirect</option>
          </select>
        </div>

        <div class="form-control mb-6">
          <label class="label cursor-pointer justify-start gap-3">
            <input v-model="form.preserve_path" type="checkbox" class="checkbox" />
            <div>
              <span class="label-text">Preserve Path</span>
              <div class="text-xs text-base-content/60">
                Include the original path in the redirect URL
              </div>
            </div>
          </label>
        </div>

        <div class="modal-action">
          <button @click="closeModals" class="btn">Cancel</button>
          <button @click="createRedirect" class="btn btn-primary" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            Create
          </button>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showEditModal" class="modal modal-open">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">Edit Redirect</h3>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Source Domains</span>
          </label>
          <input
            v-model="form.source_domains"
            type="text"
            placeholder="example.com, www.example.com"
            class="input input-bordered"
            :class="{ 'input-error': formErrors.source_domains }"
          />
          <label class="label">
            <span class="label-text-alt text-base-content/60">
              Comma-separated list of source domains
            </span>
          </label>
          <label v-if="formErrors.source_domains" class="label">
            <span class="label-text-alt text-error">{{ formErrors.source_domains }}</span>
          </label>
        </div>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Destination URL</span>
          </label>
          <input
            v-model="form.destination_url"
            type="url"
            placeholder="https://newsite.com"
            class="input input-bordered"
            :class="{ 'input-error': formErrors.destination_url }"
          />
          <label v-if="formErrors.destination_url" class="label">
            <span class="label-text-alt text-error">{{ formErrors.destination_url }}</span>
          </label>
        </div>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text">Redirect Type</span>
          </label>
          <select v-model="form.redirect_code" class="select select-bordered">
            <option :value="301">301 - Permanent Redirect</option>
            <option :value="302">302 - Temporary Redirect</option>
          </select>
        </div>

        <div class="form-control mb-6">
          <label class="label cursor-pointer justify-start gap-3">
            <input v-model="form.preserve_path" type="checkbox" class="checkbox" />
            <div>
              <span class="label-text">Preserve Path</span>
              <div class="text-xs text-base-content/60">
                Include the original path in the redirect URL
              </div>
            </div>
          </label>
        </div>

        <div class="modal-action">
          <button @click="closeModals" class="btn">Cancel</button>
          <button @click="updateRedirect" class="btn btn-primary" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            Update
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Modal -->
    <div v-if="showDeleteModal" class="modal modal-open">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4 text-error">Delete Redirect</h3>
        <p class="mb-6">
          Are you sure you want to delete this redirect? This action cannot be undone.
        </p>

        <div v-if="editingRedirect" class="bg-base-200 p-4 rounded mb-6">
          <div class="text-sm text-base-content/80 mb-2">
            <strong>Source Domains:</strong> {{ editingRedirect.source_domains.join(", ") }}
          </div>
          <div class="text-sm text-base-content/80">
            <strong>Destination:</strong> {{ editingRedirect.destination_url }}
          </div>
        </div>

        <div class="modal-action">
          <button @click="closeModals" class="btn">Cancel</button>
          <button @click="deleteRedirect" class="btn btn-error" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            Delete
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
