import { api } from './api'

export interface User {
  id: string
  username: string
  created: string
  updated: string
}

export interface AuthResponse {
  success: boolean
  message?: string
  token?: string
}

export interface StatusResponse {
  is_setup: boolean
  auth_enabled: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export interface SetupRequest {
  username: string
  password: string
}

export interface UserResponse {
  success: boolean
  user?: User
}

class AuthService {
  private token: string | null = null

  constructor() {
    // Load token from localStorage on initialization
    this.token = localStorage.getItem('auth_token')
  }

  private setAuthHeaders(): { [key: string]: string } {
    const headers: { [key: string]: string } = {
      'Content-Type': 'application/json'
    }
    
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }
    
    return headers
  }

  async getStatus(): Promise<StatusResponse> {
    const response = await fetch(`${api.baseUrl}/auth/status`)
    if (!response.ok) {
      throw new Error('Failed to get auth status')
    }
    return response.json()
  }

  async setup(data: SetupRequest): Promise<AuthResponse> {
    const response = await fetch(`${api.baseUrl}/auth/setup`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    })

    const result: AuthResponse = await response.json()
    
    if (result.success && result.token) {
      this.token = result.token
      localStorage.setItem('auth_token', result.token)
    }
    
    return result
  }

  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(`${api.baseUrl}/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    })

    const result: AuthResponse = await response.json()
    
    if (result.success && result.token) {
      this.token = result.token
      localStorage.setItem('auth_token', result.token)
    }
    
    return result
  }

  async logout(): Promise<AuthResponse> {
    const response = await fetch(`${api.baseUrl}/auth/logout`, {
      method: 'POST',
      headers: this.setAuthHeaders()
    })

    const result: AuthResponse = await response.json()
    
    // Clear token regardless of response
    this.token = null
    localStorage.removeItem('auth_token')
    
    return result
  }

  async getCurrentUser(): Promise<UserResponse> {
    if (!this.token) {
      throw new Error('Not authenticated')
    }

    const response = await fetch(`${api.baseUrl}/auth/me`, {
      headers: this.setAuthHeaders()
    })

    if (!response.ok) {
      if (response.status === 401) {
        // Token is invalid, clear it
        this.token = null
        localStorage.removeItem('auth_token')
      }
      throw new Error('Failed to get current user')
    }

    return response.json()
  }

  isAuthenticated(): boolean {
    return this.token !== null
  }

  getToken(): string | null {
    return this.token
  }

  // Helper method to add auth headers to any request
  async authenticatedFetch(url: string, options: RequestInit = {}): Promise<Response> {
    const headers = {
      ...this.setAuthHeaders(),
      ...(options.headers as Record<string, string> || {})
    }

    const response = await fetch(url, {
      ...options,
      headers
    })

    // If unauthorized, clear token
    if (response.status === 401) {
      this.token = null
      localStorage.removeItem('auth_token')
    }

    return response
  }
}

export const authService = new AuthService()