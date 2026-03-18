import axios from 'axios';

export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as { _retry?: boolean; headers: Record<string, string> };
    const status = error?.response?.status;

    if (status === 401 && originalRequest && !originalRequest._retry) {
      const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null;
      if (!refreshToken) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;
      try {
        const refreshResponse = await api.post('/auth/refresh', { refresh_token: refreshToken });
        const { access_token, refresh_token } = refreshResponse.data.tokens || {};

        if (typeof window !== 'undefined') {
          if (access_token) {
            localStorage.setItem('token', access_token);
          }
          if (refresh_token) {
            localStorage.setItem('refresh_token', refresh_token);
          }
        }

        if (access_token) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
        }
        return api(originalRequest);
      } catch (refreshError) {
        if (typeof window !== 'undefined') {
          localStorage.removeItem('token');
          localStorage.removeItem('refresh_token');
        }
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);
