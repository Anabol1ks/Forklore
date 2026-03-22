import axios from 'axios';
import { clearAuthTokens, getAccessToken, getRefreshToken, setAuthTokens } from '@/lib/auth-cookies';

export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = getAccessToken() || (typeof window !== 'undefined' ? localStorage.getItem('token') : null);
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as { _retry?: boolean; url?: string; headers: Record<string, string> };
    const status = error?.response?.status;
    const requestURL = originalRequest?.url || '';
    const isRefreshCall = requestURL.includes('/auth/refresh');

    if (status === 401 && originalRequest && !originalRequest._retry && !isRefreshCall) {
      const refreshToken = getRefreshToken() || (typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null);
      if (!refreshToken) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;
      try {
        const refreshResponse = await api.post('/auth/refresh', { refresh_token: refreshToken });
        const { access_token, refresh_token } = refreshResponse.data.tokens || {};

        if (typeof window !== 'undefined') {
          if (access_token && refresh_token) {
            setAuthTokens(access_token, refresh_token);
            localStorage.removeItem('token');
            localStorage.removeItem('refresh_token');
          }
        }

        if (access_token) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
        }
        return api(originalRequest);
      } catch {
        if (typeof window !== 'undefined') {
          clearAuthTokens();
          localStorage.removeItem('token');
          localStorage.removeItem('refresh_token');
        }
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);
