'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useAuthStore } from '@/shared/stores/authStore';
import { useAuthHydrated } from '@/shared/hooks/useAuthHydrated';
import { ROLES } from '@/shared/utils/constants';
import type { Role } from '@/shared/types';

function FullPageMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#FAF9F6] text-[13px] text-[#444748]">
      <span className="material-symbols-outlined mr-2 animate-spin text-[18px] text-[#B5F546]">progress_activity</span>
      {children}
    </div>
  );
}

// Protects every (app) route: no session → /login, session without an
// organization → /select-organization (where they can create one).
export function RouteGuard({ children }: { children: React.ReactNode }) {
  const hydrated = useAuthHydrated();
  const { isAuthenticated, currentOrg } = useAuthStore();
  const router = useRouter();
  const pathname = usePathname();

  const hasToken = typeof window !== 'undefined' && !!localStorage.getItem('accessToken');
  const signedIn = isAuthenticated && hasToken;

  useEffect(() => {
    if (!hydrated) return;
    if (!signedIn) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    } else if (!currentOrg) {
      router.replace('/select-organization');
    }
  }, [hydrated, signedIn, currentOrg, router, pathname]);

  if (!hydrated) return <FullPageMessage>Loading…</FullPageMessage>;
  if (!signedIn) return <FullPageMessage>Redirecting to login…</FullPageMessage>;
  if (!currentOrg) return <FullPageMessage>Choose an organization…</FullPageMessage>;
  return <>{children}</>;
}

// Shows a 403 state instead of rendering a page the role can't use.
export function RoleGuard({ allowedRoles, children }: { allowedRoles: Role[]; children: React.ReactNode }) {
  const { currentOrg } = useAuthStore();
  const role = (currentOrg?.role ?? 'staff') as Role;

  if (!allowedRoles.includes(role)) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-3 text-center">
        <span className="material-symbols-outlined text-4xl text-[#C4C7C7]">lock_person</span>
        <h2 className="text-xl font-bold text-[#1B1C1A]">Access denied</h2>
        <p className="max-w-sm text-[14px] text-[#444748]">
          This page is for {allowedRoles.map((r) => ROLES[r]).join(' or ')} roles. You&apos;re signed in as{' '}
          {ROLES[role] ?? role}.
        </p>
      </div>
    );
  }
  return <>{children}</>;
}
