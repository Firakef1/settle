import { useAuthStore } from '../stores/authStore';

export function useAuth() {
  const store = useAuthStore();
  return {
    user: store.user,
    token: null,
    currentOrg: store.currentOrg,
    orgs: store.orgs,
    isAuthenticated: store.isAuthenticated,
    role: store.currentOrg?.role ?? null,
  };
}
