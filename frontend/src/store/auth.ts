import { create } from 'zustand';
import { api } from '@/lib/api';
import { clearAuthTokens, getAccessToken, getRefreshToken, setAuthTokens } from '@/lib/auth-cookies';

interface User {
  id: string;
  username: string;
  email: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (token: string, refreshToken: string, userData: User) => void;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
  initialize: () => Promise<void>;
  checkAuth: () => Promise<void>;
}

type ApiUser = {
  id?: string;
  user_id?: string;
  username?: string;
  email?: string;
};

type ApiUserPayload = {
  user?: ApiUser;
} & ApiUser;

function normalizeUser(payload: unknown): User | null {
  if (!payload || typeof payload !== 'object') {
    return null;
  }

  const src = payload as ApiUserPayload;
  const candidate = src.user && typeof src.user === 'object' ? src.user : src;

  const id = candidate.id ?? candidate.user_id;
  const username = candidate.username;
  const email = candidate.email;

  if (!id || !username || !email) {
    return null;
  }

  return { id, username, email };
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  login: (token, refreshToken, userData) => {
    if (typeof window !== 'undefined') {
      setAuthTokens(token, refreshToken);
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
    }
    const normalized = normalizeUser(userData);
    set({
      user: normalized,
      isAuthenticated: !!normalized,
      isLoading: false,
    });
  },
  logout: async () => {
    if (typeof window !== 'undefined') {
      const activeRefreshToken = getRefreshToken() || localStorage.getItem('refresh_token');
      if (activeRefreshToken) {
        try {
          await api.post('/auth/logout', { refresh_token: activeRefreshToken });
        } catch (e) {
          console.error("Logout failed on server", e);
        }
      }
      clearAuthTokens();
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
    }
    set({ user: null, isAuthenticated: false, isLoading: false });
  },
  logoutAll: async () => {
    try {
      await api.post('/auth/logout-all');
    } catch (e) {
      console.error('Logout-all failed on server', e);
    } finally {
      if (typeof window !== 'undefined') {
        clearAuthTokens();
        localStorage.removeItem('token');
        localStorage.removeItem('refresh_token');
      }
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
  initialize: async () => {
    if (typeof window === 'undefined') return;
    const tokenFromCookie = getAccessToken();
    const tokenFromLocalStorage = localStorage.getItem('token');
    const token = tokenFromCookie || tokenFromLocalStorage;

    if (!tokenFromCookie && tokenFromLocalStorage) {
      const refreshFromLocalStorage = localStorage.getItem('refresh_token') || '';
      if (refreshFromLocalStorage) {
        setAuthTokens(tokenFromLocalStorage, refreshFromLocalStorage);
      }
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
    }

    if (!token) {
      set({ user: null, isAuthenticated: false, isLoading: false });
      return;
    }

    try {
      const response = await api.get('/auth/me');
      const normalized = normalizeUser(response.data);
      if (!normalized) {
        throw new Error('Invalid user payload from /auth/me');
      }
      set({ user: normalized, isAuthenticated: true, isLoading: false });
    } catch {
      clearAuthTokens();
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
  checkAuth: async () => {
    await useAuthStore.getState().initialize();
  },
}));
