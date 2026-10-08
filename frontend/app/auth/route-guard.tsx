'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useAuthStore } from '@/shared/stores/authStore';

export function RouteGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.replace('/login');
    }
  }, [isAuthenticated, router]);

  if (!isAuthenticated) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[#FAF9F6] text-[#444748]">
        Redirecting to login...
      </div>
    );
  }

  return <>{children}</>;
}

export function RoleGuard({ allowedRoles, children }: { allowedRoles: string[]; children: React.ReactNode }) {
  const { currentOrg } = useAuthStore();
  const role = currentOrg?.role || 'staff';

  if (!allowedRoles.includes(role)) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3">
        <span className="material-symbols-outlined text-4xl text-[#C4C7C7]">lock_person</span>
        <h2 className="text-xl font-bold text-[#1B1C1A]">Access Denied</h2>
        <p className="text-[#444748]">Your role ({role}) does not have access to this page.</p>
      </div>
    );
  }
  return <>{children}</>;
}
