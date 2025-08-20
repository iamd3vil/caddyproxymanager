<template>
  <div class="min-h-screen flex items-center justify-center bg-base-200">
    <div class="card w-full max-w-sm bg-base-100 shadow-xl">
      <div class="card-body">
        <div class="text-center mb-6">
          <h1 class="text-2xl font-bold">Caddy Proxy Manager</h1>
          <p class="text-base-content/60">Sign in to continue</p>
        </div>

        <form @submit.prevent="handleLogin" class="space-y-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text">Username</span>
            </label>
            <input
              v-model="form.username"
              type="text"
              class="input input-bordered w-full"
              :class="{ 'input-error': errors.username }"
              placeholder="Enter your username"
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
              placeholder="Enter your password"
              required
              :disabled="loading"
            />
            <div v-if="errors.password" class="label">
              <span class="label-text-alt text-error">{{ errors.password }}</span>
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
              <span v-if="!loading">Sign In</span>
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
import { authService, type LoginRequest } from '../../services/auth'

const router = useRouter()

const loading = ref(false)
const error = ref('')

const form = reactive<LoginRequest>({
  username: '',
  password: ''
})

const errors = reactive({
  username: '',
  password: ''
})

const clearErrors = () => {
  errors.username = ''
  errors.password = ''
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
  }

  return isValid
}

const handleLogin = async () => {
  if (!validateForm()) return

  loading.value = true
  clearErrors()

  try {
    const response = await authService.login(form)
    
    if (response.success) {
      // Navigate to dashboard on success
      router.push('/dashboard')
    } else {
      error.value = response.message || 'Login failed'
    }
  } catch (err) {
    error.value = 'An error occurred during login'
    console.error('Login error:', err)
  } finally {
    loading.value = false
  }
}
</script>