import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authAPI } from '../services/authAPI';
import { apiErrorMessage } from '../utils/apiError';

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
  signInWithTokens: (token: string, refreshToken: string) => Promise<void>;
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
            orgs: response.orgs ?? [],
            currentOrg: response.orgs?.[0] ?? null,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (error: unknown) {
          set({
            isLoading: false,
            error: apiErrorMessage(error, 'Login failed. Please try again.'),
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
          set({
            isLoading: false,
            error: apiErrorMessage(error, 'Signup failed. Please try again.'),
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
            orgs: response.orgs ?? [],
            currentOrg: response.orgs?.[0] ?? null,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
          // verify-email omits `orgs`. Someone who joined through an invite already
          // has one, so refresh once to load it (refresh returns the memberships).
          if (!response.orgs) {
            await get().refreshToken().catch(() => undefined);
          }
        } catch (error: unknown) {
          set({
            isLoading: false,
            error: apiErrorMessage(error, 'Verification failed. Please try again.'),
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
          set({
            isLoading: false,
            error: apiErrorMessage(error, 'Failed to send reset email.'),
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
          set({
            isLoading: false,
            error: apiErrorMessage(error, 'Password reset failed.'),
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
          // The new token carries the first org (e.g. right after creating one),
          // so keep the current org in step with it.
          const { currentOrg } = get();
          set({
            user: response.user,
            orgs: response.orgs ?? [],
            currentOrg: response.orgs?.find((o) => o.org_id === currentOrg?.org_id) ?? response.orgs?.[0] ?? null,
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

      // After "Sign in with Google/Microsoft": the backend hands over a token
      // pair; refresh once to load the user and their organizations.
      signInWithTokens: async (token: string, refreshToken: string) => {
        localStorage.setItem('accessToken', token);
        localStorage.setItem('refreshToken', refreshToken);
        set({ currentOrg: null });
        await get().refreshToken();
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
