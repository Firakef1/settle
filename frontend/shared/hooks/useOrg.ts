import { useAuthStore } from '../stores/authStore';

export function useOrg() {
  const store = useAuthStore();
  // Token carries the first org (per backend contract: "The token is one organization")
  return store.currentOrg;
}
