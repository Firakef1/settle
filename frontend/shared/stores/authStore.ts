import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authAPI } from '../services/authAPI';
import { AxiosError } from 'axios';

interface User {
  id: string;
  email: string;
  name: string;
  status: string;
  email_verified: boolean;
  created_at: string;
}

interface Organization {
  org_id: string;
  org_name: string;
  org_slug: string;
  role: string;
  department?: string;
}

interface AuthState {
  user: User | null;
  orgs: Organization[];
  currentOrg: Organization | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  login: (email: string, password: string) => Promise<void>;
  signup: (name: string, email: string, password: string) => Promise<void>;
  verifyEmail: (email: string, code: string) => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (email: string, otp: string, newPassword: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  selectOrganization: (orgId: string) => void;
  clearError: () => void;
  setLoading: (loading: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      orgs: [],
      currentOrg: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authAPI.login({ email, password });
          if (typeof window !== 'undefined') {
            localStorage.setItem('accessToken', response.token);
            localStorage.setItem('refreshToken', response.refresh_token);
          }
          set({
            user: response.user,
            orgs: response.orgs,
            currentOrg: response.orgs[0] || null,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (error: unknown) {
          const axiosError = error as AxiosError<{ message?: string }>;
          set({
            isLoading: false,
            error: axiosError.response?.data?.message || 'Login failed. Please try again.',
          });
          throw error;
        }
      },

      signup: async (name: string, email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.signup({ name, email, password });
          set({ isLoading: false, error: null });
        } catch (error: unknown) {
          const axiosError = error as AxiosError<{ message?: string }>;
          set({
            isLoading: false,
            error: axiosError.response?.data?.message || 'Signup failed. Please try again.',
          });
          throw error;
        }
      },

      verifyEmail: async (email: string, verification_code: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authAPI.verifyEmail({ email, verification_code });
          if (typeof window !== 'undefined') {
            localStorage.setItem('accessToken', response.token);
            localStorage.setItem('refreshToken', response.refresh_token);
          }
          set({
            user: response.user,
            orgs: response.orgs,
            currentOrg: response.orgs[0] || null,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (error: unknown) {
          const axiosError = error as AxiosError<{ message?: string }>;
          set({
            isLoading: false,
            error: axiosError.response?.data?.message || 'Verification failed. Please try again.',
          });
          throw error;
        }
      },

      forgotPassword: async (email: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.forgotPassword({ email });
          set({ isLoading: false, error: null });
        } catch (error: unknown) {
          const axiosError = error as AxiosError<{ message?: string }>;
          set({
            isLoading: false,
            error: axiosError.response?.data?.message || 'Failed to send reset email.',
          });
          throw error;
        }
      },

      resetPassword: async (email: string, otp: string, new_password: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.resetPassword({ email, otp, new_password });
          set({ isLoading: false, error: null });
        } catch (error: unknown) {
          const axiosError = error as AxiosError<{ message?: string }>;
          set({
            isLoading: false,
            error: axiosError.response?.data?.message || 'Password reset failed.',
          });
          throw error;
        }
      },

      logout: async () => {
        set({ isLoading: true });
        try {
          if (typeof window !== 'undefined') {
            const refreshToken = localStorage.getItem('refreshToken');
            if (refreshToken) {
              await authAPI.logout(refreshToken);
            }
          }
        } catch (error) {
          console.warn('Logout API call failed:', error);
        } finally {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('accessToken');
            localStorage.removeItem('refreshToken');
          }
          set({
            user: null,
            orgs: [],
            currentOrg: null,
            isAuthenticated: false,
            isLoading: false,
            error: null,
          });
        }
      },

      refreshToken: async () => {
        if (typeof window === 'undefined') throw new Error('Cannot refresh token on server side');

        const refreshToken = localStorage.getItem('refreshToken');
        if (!refreshToken) throw new Error('No refresh token available');
        try {
          const response = await authAPI.refreshToken({ refresh_token: refreshToken });
          localStorage.setItem('accessToken', response.token);
          localStorage.setItem('refreshToken', response.refresh_token);
          set({
            user: response.user,
            orgs: response.orgs,
            isAuthenticated: true,
            error: null,
          });
        } catch (error: unknown) {
          localStorage.removeItem('accessToken');
          localStorage.removeItem('refreshToken');
          set({
            user: null,
            orgs: [],
            currentOrg: null,
            isAuthenticated: false,
            error: 'Session expired. Please sign in again.',
          });
          throw error;
        }
      },

      selectOrganization: (orgId: string) => {
        const { orgs } = get();
        const selectedOrg = orgs.find(org => org.org_id === orgId);
        if (selectedOrg) {
          set({ currentOrg: selectedOrg });
        }
      },

      clearError: () => set({ error: null }),
      setLoading: (loading: boolean) => set({ isLoading: loading }),
    }),
    {
      name: 'settle-auth-storage',
      partialize: (state) => ({
        user: state.user,
        orgs: state.orgs,
        currentOrg: state.currentOrg,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);
