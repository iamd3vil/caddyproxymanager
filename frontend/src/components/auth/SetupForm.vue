<template>
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="card w-full max-w-sm bg-base-100 shadow-xl">
      <div class="card-body">
        <div class="text-center mb-6">
          <h1 class="text-2xl font-bold">Welcome to Caddy Proxy Manager</h1>
          <p class="text-base-content/60">Create your admin account</p>
        </div>

        <form @submit.prevent="handleSetup" class="space-y-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text">Username</span>
            </label>
            <input
              v-model="form.username"
              type="text"
              class="input input-bordered w-full"
              :class="{ 'input-error': errors.username }"
              placeholder="Choose a username"
              required
              :disabled="loading"
            />
            <div v-if="errors.username" class="label">
              <span class="label-text-alt text-error">{{ errors.username }}</span>
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Password</span>
            </label>
            <input
              v-model="form.password"
              type="password"
              class="input input-bordered w-full"
              :class="{ 'input-error': errors.password }"
              placeholder="Choose a secure password"
              required
              :disabled="loading"
            />
            <div v-if="errors.password" class="label">
              <span class="label-text-alt text-error">{{ errors.password }}</span>
            </div>
            <div v-else class="label">
              <span class="label-text-alt">At least 6 characters</span>
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Confirm Password</span>
            </label>
            <input
              v-model="confirmPassword"
              type="password"
              class="input input-bordered w-full"
              :class="{ 'input-error': errors.confirmPassword }"
              placeholder="Confirm your password"
              required
              :disabled="loading"
            />
            <div v-if="errors.confirmPassword" class="label">
              <span class="label-text-alt text-error">{{ errors.confirmPassword }}</span>
            </div>
          </div>

          <div v-if="error" class="alert alert-error">
            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{{ error }}</span>
          </div>

          <div class="form-control mt-6">
            <button 
              type="submit" 
              class="btn btn-primary w-full"
              :class="{ loading: loading }"
              :disabled="loading"
            >
              <span v-if="!loading">Create Account</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { authService, type SetupRequest } from '../../services/auth'

const router = useRouter()

const loading = ref(false)
const error = ref('')
const confirmPassword = ref('')

const form = reactive<SetupRequest>({
  username: '',
  password: ''
})

const errors = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const clearErrors = () => {
  errors.username = ''
  errors.password = ''
  errors.confirmPassword = ''
  error.value = ''
}

const validateForm = (): boolean => {
  clearErrors()
  let isValid = true

  if (!form.username.trim()) {
    errors.username = 'Username is required'
    isValid = false
  }

  if (!form.password.trim()) {
    errors.password = 'Password is required'
    isValid = false
  } else if (form.password.length < 6) {
    errors.password = 'Password must be at least 6 characters'
    isValid = false
  }

  if (!confirmPassword.value.trim()) {
    errors.confirmPassword = 'Please confirm your password'
    isValid = false
  } else if (form.password !== confirmPassword.value) {
    errors.confirmPassword = 'Passwords do not match'
    isValid = false
  }

  return isValid
}

const handleSetup = async () => {
  if (!validateForm()) return

  loading.value = true
  clearErrors()

  try {
    const response = await authService.setup(form)
    
    if (response.success) {
      // Navigate to dashboard on success
      router.push('/dashboard')
    } else {
      error.value = response.message || 'Setup failed'
    }
  } catch (err) {
    error.value = 'An error occurred during setup'
    console.error('Setup error:', err)
  } finally {
    loading.value = false
  }
}
</script>