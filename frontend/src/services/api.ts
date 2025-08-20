export interface Proxy {
  id: string
  domain: string
  target_url: string
  ssl_mode: string
  challenge_type?: string
  dns_provider?: string
  dns_credentials?: Record<string, string>
  status?: string
  created_at: string
  updated_at: string
}

export interface ApiResponse<T> {
  data?: T
  error?: string
}

export interface ProxiesResponse {
  proxies: Proxy[]
  count: number
}

export interface StatusResponse {
  caddy_status: string
  caddy_reachable: boolean
  upstreams?: any
  error?: string
  last_checked: string
}

class ApiClient {
  private baseUrl: string

  constructor(baseUrl?: string) {
    if (baseUrl) {
      this.baseUrl = baseUrl
    } else {
      // Use relative URLs to leverage Vite proxy in dev and current origin in production
      this.baseUrl = ''
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      })

      if (!response.ok) {
        const errorText = await response.text()
        try {
          const errorJson = JSON.parse(errorText)
          return { error: errorJson.error || errorText }
        } catch {
          return { error: errorText }
        }
      }

      const data = await response.json()
      return { data }
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Unknown error' }
    }
  }

  async health(): Promise<ApiResponse<{ status: string; timestamp: string }>> {
    return this.request('/api/health')
  }

  async getProxies(): Promise<ApiResponse<ProxiesResponse>> {
    return this.request('/api/proxies')
  }

  async createProxy(proxy: {
    domain: string
    target_url: string
    ssl_mode?: string
    challenge_type?: string
    dns_provider?: string
    dns_credentials?: Record<string, string>
  }): Promise<ApiResponse<Proxy>> {
    return this.request('/api/proxies', {
      method: 'POST',
      body: JSON.stringify(proxy),
    })
  }

  async updateProxy(
    id: string,
    proxy: {
      domain: string
      target_url: string
      ssl_mode?: string
      challenge_type?: string
      dns_provider?: string
      dns_credentials?: Record<string, string>
    }
  ): Promise<ApiResponse<Proxy>> {
    return this.request(`/api/proxies/${id}`, {
      method: 'PUT',
      body: JSON.stringify(proxy),
    })
  }

  async deleteProxy(id: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/api/proxies/${id}`, {
      method: 'DELETE',
    })
  }

  async getStatus(): Promise<ApiResponse<StatusResponse>> {
    return this.request('/api/status')
  }

  async reload(): Promise<ApiResponse<{ message: string }>> {
    return this.request('/api/reload', {
      method: 'POST',
    })
  }
}

export const apiClient = new ApiClient()