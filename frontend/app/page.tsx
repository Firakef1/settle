'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/shared/stores/authStore';
import { useAuthHydrated } from '@/shared/hooks/useAuthHydrated';

export default function HomePage() {
  const hydrated = useAuthHydrated();
  const { isAuthenticated } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (!hydrated) return;
    router.replace(isAuthenticated ? '/dashboard' : '/login');
  }, [hydrated, isAuthenticated, router]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#FAF9F6] text-[#444748]">
      <div className="flex flex-col items-center gap-3">
        <span className="material-symbols-outlined animate-spin text-3xl text-[#B5F546]">progress_activity</span>
        <span>Redirecting…</span>
      </div>
    </main>
  );
}
