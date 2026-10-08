import { useSyncExternalStore } from 'react';
import { useAuthStore } from '../stores/authStore';

// The auth store is persisted to localStorage. On the server and during the
// first client render this returns false, so markup matches; it flips to true
// once the saved session has been read.
export function useAuthHydrated(): boolean {
  return useSyncExternalStore(
    (onChange) => useAuthStore.persist.onFinishHydration(onChange),
    () => useAuthStore.persist.hasHydrated(),
    () => false,
  );
}
