export interface Proxy {
  id: string;
  domain: string;
  target_url: string;
  ssl_mode: string;
  challenge_type?: string;
  dns_provider?: string;
  dns_credentials?: Record<string, string>;
  custom_headers?: Record<string, string>;
  basic_auth?: { enabled: boolean; username: string; password: string } | null;
  health_check_enabled?: boolean;
  health_check_interval?: string;
  health_check_path?: string;
  health_check_expected_status?: number;
  allowed_ips?: string[];
  blocked_ips?: string[];
  status?: string;
  created_at: string;
  updated_at: string;
}

export interface Redirect {
  id: string;
  source_domains: string[];
  destination_url: string;
  redirect_code: number;
  preserve_path: boolean;
  status?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
}

export interface ProxiesResponse {
  proxies: Proxy[];
  count: number;
}

export interface RedirectsResponse {
  redirects: Redirect[];
  count: number;
}

export interface StatusResponse {
  caddy_status: string;
  caddy_reachable: boolean;
  upstreams?: any;
  error?: string;
  last_checked: string;
}

class ApiClient {
  public baseUrl: string;

  constructor(baseUrl?: string) {
    if (baseUrl) {
      this.baseUrl = baseUrl;
    } else {
      // Use relative URLs to leverage Vite proxy in dev and current origin in production
      this.baseUrl = "";
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    useAuth = true,
  ): Promise<ApiResponse<T>> {
    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...((options.headers as Record<string, string>) || {}),
      };

      // Add auth headers if needed
      if (useAuth) {
        const token = localStorage.getItem("auth_token");
        if (token) {
          headers["Authorization"] = `Bearer ${token}`;
        }
      }

      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        ...options,
        headers,
      });

      if (!response.ok) {
        const errorText = await response.text();
        try {
          const errorJson = JSON.parse(errorText);
          return { error: errorJson.error || errorText };
        } catch {
          return { error: errorText };
        }
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return { error: error instanceof Error ? error.message : "Unknown error" };
    }
  }

  async health(): Promise<ApiResponse<{ status: string; timestamp: string }>> {
    return this.request("/api/health");
  }

  async getProxies(): Promise<ApiResponse<ProxiesResponse>> {
    return this.request("/api/proxies");
  }

  async createProxy(proxy: {
    domain: string;
    target_url: string;
    ssl_mode?: string;
    challenge_type?: string;
    dns_provider?: string;
    dns_credentials?: Record<string, string>;
    custom_headers?: Record<string, string>;
    basic_auth?: { enabled: boolean; username: string; password: string } | null;
    health_check_enabled?: boolean;
    health_check_interval?: string;
    health_check_path?: string;
    health_check_expected_status?: number;
    allowed_ips?: string[];
    blocked_ips?: string[];
  }): Promise<ApiResponse<Proxy>> {
    return this.request("/api/proxies", {
      method: "POST",
      body: JSON.stringify(proxy),
    });
  }

  async updateProxy(
    id: string,
    proxy: {
      domain: string;
      target_url: string;
      ssl_mode?: string;
      challenge_type?: string;
      dns_provider?: string;
      dns_credentials?: Record<string, string>;
      custom_headers?: Record<string, string>;
      basic_auth?: { enabled: boolean; username: string; password: string } | null;
      health_check_enabled?: boolean;
      health_check_interval?: string;
      health_check_path?: string;
      health_check_expected_status?: number;
      allowed_ips?: string[];
      blocked_ips?: string[];
    },
  ): Promise<ApiResponse<Proxy>> {
    return this.request(`/api/proxies/${id}`, {
      method: "PUT",
      body: JSON.stringify(proxy),
    });
  }

  async deleteProxy(id: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/api/proxies/${id}`, {
      method: "DELETE",
    });
  }

  async getProxyStatus(
    id: string,
  ): Promise<ApiResponse<{ status: string; last_checked: string; message: string }>> {
    return this.request(`/api/proxies/${id}/status`);
  }

  async getStatus(): Promise<ApiResponse<StatusResponse>> {
    return this.request("/api/status");
  }

  async reload(): Promise<ApiResponse<{ message: string }>> {
    return this.request("/api/reload", {
      method: "POST",
    });
  }

  async getRedirects(): Promise<ApiResponse<RedirectsResponse>> {
    return this.request("/api/redirects");
  }

  async createRedirect(redirect: {
    source_domains: string[];
    destination_url: string;
    redirect_code?: number;
    preserve_path?: boolean;
  }): Promise<ApiResponse<Redirect>> {
    return this.request("/api/redirects", {
      method: "POST",
      body: JSON.stringify(redirect),
    });
  }

  async updateRedirect(
    id: string,
    redirect: {
      source_domains: string[];
      destination_url: string;
      redirect_code?: number;
      preserve_path?: boolean;
    },
  ): Promise<ApiResponse<Redirect>> {
    return this.request(`/api/redirects/${id}`, {
      method: "PUT",
      body: JSON.stringify(redirect),
    });
  }

  async deleteRedirect(id: string): Promise<ApiResponse<{ message: string }>> {
    return this.request(`/api/redirects/${id}`, {
      method: "DELETE",
    });
  }
}

export const apiClient = new ApiClient();

// Export api object for use in auth service
export const api = {
  baseUrl: apiClient.baseUrl + "/api",
};
