import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authAPI, LoginResponse } from '../services/authAPI';

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
  // State
  user: User | null;
  orgs: Organization[];
  currentOrg: Organization | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
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
      // Initial state
      user: null,
      orgs: [],
      currentOrg: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      // Login
      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authAPI.login({ email, password });

          // Store tokens
          localStorage.setItem('accessToken', response.token);
          localStorage.setItem('refreshToken', response.refresh_token);

          // Update state
          set({
            user: response.user,
            orgs: response.orgs,
            currentOrg: response.orgs[0] || null, // Select first org by default
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (error: any) {
          set({
            isLoading: false,
            error: error.response?.data?.message || 'Login failed. Please try again.',
          });
          throw error;
        }
      },

      // Signup
      signup: async (name: string, email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.signup({ name, email, password });
          set({ isLoading: false, error: null });
        } catch (error: any) {
          set({
            isLoading: false,
            error: error.response?.data?.message || 'Signup failed. Please try again.',
          });
          throw error;
        }
      },

      // Verify Email
      verifyEmail: async (email: string, verification_code: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authAPI.verifyEmail({ email, verification_code });

          // Store tokens
          localStorage.setItem('accessToken', response.token);
          localStorage.setItem('refreshToken', response.refresh_token);

          // Update state
          set({
            user: response.user,
            orgs: response.orgs,
            currentOrg: response.orgs[0] || null,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (error: any) {
          set({
            isLoading: false,
            error: error.response?.data?.message || 'Verification failed. Please try again.',
          });
          throw error;
        }
      },

      // Forgot Password
      forgotPassword: async (email: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.forgotPassword({ email });
          set({ isLoading: false, error: null });
        } catch (error: any) {
          set({
            isLoading: false,
            error: error.response?.data?.message || 'Failed to send reset email.',
          });
          throw error;
        }
      },

      // Reset Password
      resetPassword: async (email: string, otp: string, new_password: string) => {
        set({ isLoading: true, error: null });
        try {
          await authAPI.resetPassword({ email, otp, new_password });
          set({ isLoading: false, error: null });
        } catch (error: any) {
          set({
            isLoading: false,
            error: error.response?.data?.message || 'Password reset failed.',
          });
          throw error;
        }
      },

      // Logout
      logout: async () => {
        set({ isLoading: true });
        try {
          const refreshToken = localStorage.getItem('refreshToken');
          if (refreshToken) {
            await authAPI.logout(refreshToken);
          }
        } catch (error) {
          // Continue with logout even if API call fails
          console.warn('Logout API call failed:', error);
        } finally {
          // Clear everything
          localStorage.removeItem('accessToken');
          localStorage.removeItem('refreshToken');
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

      // Refresh Token
      refreshToken: async () => {
        const refreshToken = localStorage.getItem('refreshToken');
        if (!refreshToken) {
          throw new Error('No refresh token available');
        }

        try {
          const response = await authAPI.refreshToken({ refresh_token: refreshToken });

          // Store new tokens
          localStorage.setItem('accessToken', response.token);
          localStorage.setItem('refreshToken', response.refresh_token);

          // Update state
          set({
            user: response.user,
            orgs: response.orgs,
            isAuthenticated: true,
            error: null,
          });
        } catch (error: any) {
          // Refresh failed, clear auth state
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

      // Select Organization
      selectOrganization: (orgId: string) => {
        const { orgs } = get();
        const selectedOrg = orgs.find(org => org.org_id === orgId);
        if (selectedOrg) {
          set({ currentOrg: selectedOrg });
        }
      },

      // Clear Error
      clearError: () => set({ error: null }),

      // Set Loading
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
