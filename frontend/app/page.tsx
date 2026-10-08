'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/shared/stores/authStore';

export default function HomePage() {
  const { isAuthenticated } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.replace('/login');
    } else {
      router.replace('/requests');
    }
  }, [isAuthenticated, router]);

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#FAF9F6] text-[#444748]">
      <div className="flex flex-col items-center gap-3">
        <span className="material-symbols-outlined text-3xl animate-spin text-[#B5F546]">progress_activity</span>
        <span>Redirecting...</span>
      </div>
    </main>
  );
}
