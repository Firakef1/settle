import { useAuthStore } from '../stores/authStore';

export function useRole() {
  const { user, currentOrg } = useAuthStore();
  return user?.id ? (currentOrg?.role ?? 'staff') : null;
}
