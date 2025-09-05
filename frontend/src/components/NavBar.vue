<script setup lang="ts">
import { RouterLink, useRouter } from "vue-router";
import { ref, onMounted, watch } from "vue";
import { authService } from "../services/auth";
import {
  Menu,
  Sun,
  Moon,
  ChevronDown,
  User,
  LogOut,
  Server,
  Globe,
  Zap,
  Box,
} from "lucide-vue-next";

const router = useRouter();
const isDark = ref(false);
const isAuthenticated = ref(false);
const isAuthEnabled = ref(true);
const currentUser = ref<{ username: string } | null>(null);

// Check auth status function
const checkAuthStatus = async () => {
  try {
    const status = await authService.getStatus();
    isAuthEnabled.value = status.auth_enabled;
    isAuthenticated.value = authService.isAuthenticated();

    // Fetch current user if authenticated and auth is enabled
    if (isAuthEnabled.value && isAuthenticated.value) {
      try {
        const userResponse = await authService.getCurrentUser();
        if (userResponse.success && userResponse.user) {
          currentUser.value = userResponse.user;
        }
      } catch (error) {
        console.error("Failed to get current user:", error);
        // If user fetch fails, user might not be authenticated
        isAuthenticated.value = false;
      }
    }
  } catch (error) {
    console.error("Failed to check auth status:", error);
  }
};

// Initialize theme and check auth status
onMounted(async () => {
  let theme = localStorage.getItem("theme");
  if (!theme) {
    // No saved theme, check system preference
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    theme = prefersDark ? "dark" : "light";
  }
  isDark.value = theme === "dark";
  applyTheme(theme);

  // Check auth status
  await checkAuthStatus();
});

// Watch for route changes to update auth status
watch(
  () => router.currentRoute.value,
  async () => {
    await checkAuthStatus();
  },
);

// Apply theme function
const applyTheme = (theme: string) => {
  document.documentElement.setAttribute("data-theme", theme);
  document.body.setAttribute("data-theme", theme);
};

// Watch for theme changes
watch(isDark, (newValue) => {
  const theme = newValue ? "dark" : "light";
  applyTheme(theme);
  localStorage.setItem("theme", theme);
});

// Handle logout
const handleLogout = async () => {
  try {
    await authService.logout();
    isAuthenticated.value = false;
    currentUser.value = null;
    router.push("/login");
  } catch (error) {
    console.error("Logout error:", error);
    // Force logout even if API call fails
    isAuthenticated.value = false;
    currentUser.value = null;
    router.push("/login");
  }
};
</script>

<template>
  <div class="navbar bg-base-100 shadow-sm border-b">
    <div class="navbar-start">
      <div class="dropdown">
        <div tabindex="0" role="button" class="btn btn-ghost lg:hidden">
          <Menu class="h-5 w-5" />
        </div>
        <ul
          tabindex="0"
          class="menu menu-sm dropdown-content bg-base-100 rounded-box z-[1] mt-3 w-52 p-2 shadow"
        >
          <li><RouterLink to="/">Dashboard</RouterLink></li>
          <li><RouterLink to="/proxies">Proxies</RouterLink></li>
          <li><RouterLink to="/redirects">Redirects</RouterLink></li>
          <li><RouterLink to="/audit-log">Audit Log</RouterLink></li>
        </ul>
      </div>
      <RouterLink to="/" class="btn btn-ghost text-xl">
        <Zap class="h-6 w-6" />
        Caddy PM
      </RouterLink>
    </div>
    <div class="navbar-center hidden lg:flex">
      <ul class="menu menu-horizontal px-1">
        <li><RouterLink to="/" class="btn btn-ghost">Dashboard</RouterLink></li>
        <li>
          <RouterLink to="/proxies" class="btn btn-ghost">Proxies</RouterLink>
        </li>
        <li>
          <RouterLink to="/redirects" class="btn btn-ghost">Redirects</RouterLink>
        </li>
        <li>
          <RouterLink to="/audit-log" class="btn btn-ghost">Audit Log</RouterLink>
        </li>
      </ul>
    </div>
    <div class="navbar-end gap-2">
      <!-- Theme toggle -->
      <label class="flex cursor-pointer gap-2">
        <Sun :size="20" />
        <input type="checkbox" v-model="isDark" class="toggle" />
        <Moon :size="20" />
      </label>

      <!-- User dropdown (only show if auth is enabled and user is authenticated) -->
      <div v-if="isAuthEnabled && isAuthenticated" class="dropdown dropdown-end">
        <div tabindex="0" role="button" class="btn btn-ghost gap-2">
          <!-- User avatar -->
          <div class="avatar placeholder">
            <div class="bg-neutral text-neutral-content w-8 h-8 rounded-full">
              <span class="text-xs">{{
                currentUser?.username?.charAt(0).toUpperCase() || "U"
              }}</span>
            </div>
          </div>
          <!-- Username -->
          <span class="hidden sm:inline text-sm">{{ currentUser?.username || "User" }}</span>
          <!-- Dropdown arrow -->
          <ChevronDown class="h-4 w-4" />
        </div>
        <ul
          tabindex="0"
          class="dropdown-content menu bg-base-100 rounded-box z-[1] w-52 p-2 shadow"
        >
          <li class="menu-title">
            <span>Signed in as</span>
            <span class="font-semibold">{{ currentUser?.username || "User" }}</span>
          </li>
          <div class="divider my-0"></div>
          <li>
            <button @click="handleLogout" class="text-error">
              <LogOut class="h-4 w-4" />
              Sign Out
            </button>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
