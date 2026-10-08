'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthHydrated } from '../../shared/hooks/useAuthHydrated';
import { useAuthStore } from '../../shared/stores/authStore';

// Signed-in visitors skip the marketing page and go to their dashboard.
export function RedirectIfSignedIn() {
  const hydrated = useAuthHydrated();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const router = useRouter();

  useEffect(() => {
    if (hydrated && isAuthenticated && localStorage.getItem('accessToken')) router.replace('/dashboard');
  }, [hydrated, isAuthenticated, router]);

  return null;
}
