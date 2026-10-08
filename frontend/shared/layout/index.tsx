'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '../services/api';
import { useAuthStore } from '../stores/authStore';
import type { DashboardSummary, DataResponse, Role } from '../types';
import { ROLES } from '../utils/constants';
import { fontBody, fontHeading, initials } from '../utils/fonts';
import { Icon } from '../components/Icon';
import { Toaster } from '../components/Toaster';

type BadgeKind = 'pending' | 'atRisk';

interface NavItem {
  href: string;
  label: string;
  icon: string;
  roles: Role[];
  badge?: BadgeKind;
  isActive: (pathname: string, view: string | null) => boolean;
}

const ALL: Role[] = ['staff', 'finance', 'org_admin'];
const FINANCE: Role[] = ['finance', 'org_admin'];

const NAV: NavItem[] = [
  { href: '/dashboard', label: 'Overview', icon: 'grid_view', roles: ALL, isActive: (p) => p === '/dashboard' },
  { href: '/requests', label: 'My Requests', icon: 'receipt_long', roles: ALL, isActive: (p) => p.startsWith('/requests') },
  {
    href: '/approvals',
    label: 'Approvals',
    icon: 'check_box',
    roles: FINANCE,
    badge: 'pending',
    isActive: (p, view) => p.startsWith('/approvals') && view !== 'attention',
  },
  {
    href: '/approvals?view=attention',
    label: 'Needs Attention',
    icon: 'warning',
    roles: FINANCE,
    badge: 'atRisk',
    isActive: (p, view) => p.startsWith('/approvals') && view === 'attention',
  },
  { href: '/settings/members', label: 'Members', icon: 'group', roles: FINANCE, isActive: (p) => p.startsWith('/settings/members') },
  { href: '/settings/audit-log', label: 'Audit Log', icon: 'description', roles: FINANCE, isActive: (p) => p.startsWith('/settings/audit-log') },
  {
    href: '/settings',
    label: 'Settings',
    icon: 'settings',
    roles: ALL,
    isActive: (p) => p === '/settings' || p.startsWith('/settings/billing'),
  },
];

// Same key and data as the dashboard's summary query, so the cache is shared.
function useQueueCounts(orgId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ['dashboard', 'summary', orgId],
    queryFn: async () => (await api.get<DataResponse<DashboardSummary>>('/dashboard/summary')).data.data,
    enabled: enabled && !!orgId,
    staleTime: 60 * 1000,
  });
}

export function AppLayout({ children }: { children: ReactNode }) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const pathname = usePathname();

  // Close the mobile drawer after navigating.
  useEffect(() => setDrawerOpen(false), [pathname]);

  return (
    <div className={`${fontBody} flex min-h-screen bg-[#faf9f6] text-[#1b1c1a]`}>
      <div className="sticky top-0 hidden h-screen w-72 shrink-0 lg:block">
        <Suspense fallback={<div className="h-full bg-black" />}>
          <Sidebar />
        </Suspense>
      </div>

      {drawerOpen && (
        <div className="fixed inset-0 z-50 flex lg:hidden" role="dialog" aria-modal="true" aria-label="Navigation">
          <button
            type="button"
            aria-label="Close navigation"
            className="absolute inset-0 bg-black/40"
            onClick={() => setDrawerOpen(false)}
          />
          <div className="relative h-full w-72 max-w-[85vw]">
            <Suspense fallback={<div className="h-full bg-black" />}>
              <Sidebar />
            </Suspense>
          </div>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <Header onOpenMenu={() => setDrawerOpen(true)} />
        <main className="flex-1 px-5 py-8 sm:px-7">
          <div className="mx-auto w-full max-w-[1000px]">{children}</div>
        </main>
      </div>
      <Toaster />
    </div>
  );
}

function Sidebar() {
  const router = useRouter();
  const pathname = usePathname();
  const view = useSearchParams().get('view');
  const { user, orgs, currentOrg, logout } = useAuthStore();
  const role = (currentOrg?.role ?? 'staff') as Role;
  const isFinance = FINANCE.includes(role);
  const counts = useQueueCounts(currentOrg?.org_id, isFinance);

  const badgeValue = (kind?: BadgeKind) => {
    if (!kind || !counts.data) return 0;
    return kind === 'pending' ? counts.data.pending_count : counts.data.aging_breakdown['7_plus_days'] ?? 0;
  };

  const handleLogout = async () => {
    await logout();
    router.replace('/login');
  };

  return (
    <aside className="flex h-full flex-col justify-between overflow-y-auto bg-black p-3.5 drop-shadow-[0px_1px_4px_rgba(0,0,0,0.12)]">
      <div className="flex flex-col gap-5">
        <Link href="/dashboard" className="flex items-center gap-2 px-2 py-1">
          <Image src="/brand/settle-logo.jpg" alt="" width={32} height={32} className="size-8" />
          <div>
            <p className={`${fontHeading} text-[16px] font-bold leading-6 tracking-[-0.4px] text-white`}>Settle</p>
            <p className="text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#858383]">
              {isFinance ? 'Finance Ops' : 'Payouts'}
            </p>
          </div>
        </Link>

        <Link
          href="/select-organization"
          className="flex items-center justify-between gap-2 rounded-xl bg-[#1c1b1b] p-2 transition-colors hover:bg-[#262525]"
          title="Organization"
        >
          <span className="flex min-w-0 items-center gap-2">
            <span className={`${fontHeading} flex size-7 shrink-0 items-center justify-center rounded-lg bg-[#efeeeb] text-[16px] font-bold text-black`}>
              {(currentOrg?.org_name ?? '?').charAt(0).toUpperCase()}
            </span>
            <span className="min-w-0">
              <span className="block truncate text-[13px] font-medium leading-[18px] text-white">{currentOrg?.org_name}</span>
              <span className="block truncate text-[12px] font-medium leading-4 text-[#858383]">
                {orgs.length > 1 ? `${orgs.length} organizations` : ROLES[role]}
              </span>
            </span>
          </span>
          <Icon name="unfold_more" size={15} className="text-[#858383]" />
        </Link>

        <nav aria-label="Main" className="flex flex-col gap-1">
          {NAV.filter((item) => item.roles.includes(role)).map((item) => {
            const active = item.isActive(pathname, view);
            const badge = badgeValue(item.badge);
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? 'page' : undefined}
                className={`flex items-center justify-between rounded-xl px-3.5 py-2 text-[14px] leading-5 tracking-[-0.07px] transition-colors ${
                  active ? 'bg-[#1c1b1b] text-white' : 'text-[#858383] hover:bg-[#141414] hover:text-white'
                }`}
              >
                <span className="flex items-center gap-3.5">
                  <Icon name={item.icon} size={15} />
                  {item.label}
                </span>
                {badge > 0 && (
                  <span
                    className={`rounded-full px-1 py-0.5 text-[12px] font-bold leading-4 ${
                      item.badge === 'atRisk' ? 'bg-[#ffdad6] text-[#93000a]' : 'bg-[#c7ef39] text-[#171e00]'
                    }`}
                  >
                    {badge}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>
      </div>

      <div className="mt-6 flex items-center justify-between gap-2 rounded-xl bg-[#1c1b1b] p-2">
        <Link href="/settings" className="flex min-w-0 items-center gap-2">
          <span className={`${fontHeading} flex size-8 shrink-0 items-center justify-center rounded-full bg-[#c7ef39] text-[12px] font-bold text-[#171e00]`}>
            {initials(user?.name)}
          </span>
          <span className="min-w-0">
            <span className="block truncate text-[13px] font-medium leading-[18px] text-white">{user?.name}</span>
            <span className="block truncate text-[12px] font-medium leading-4 text-[#858383]">{ROLES[role]}</span>
          </span>
        </Link>
        <button
          type="button"
          onClick={handleLogout}
          aria-label="Log out"
          title="Log out"
          className="rounded-lg p-1 text-[#858383] transition-colors hover:bg-[#262525] hover:text-white"
        >
          <Icon name="logout" size={15} />
        </button>
      </div>
    </aside>
  );
}

function Header({ onOpenMenu }: { onOpenMenu: () => void }) {
  const router = useRouter();
  const { user } = useAuthStore();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState('');

  // ⌘K / Ctrl+K focuses the request jump box.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const jumpToRequest = (e: FormEvent) => {
    e.preventDefault();
    const raw = query.trim();
    if (!raw) return;
    const id = /^req-/i.test(raw) ? `REQ-${raw.slice(4)}` : `REQ-${raw}`;
    setQuery('');
    router.push(`/requests/${encodeURIComponent(id)}`);
  };

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between gap-4 bg-[rgba(250,249,246,0.9)] px-5 shadow-[0px_1px_8px_0px_rgba(0,0,0,0.04)] backdrop-blur-[12px] sm:px-7">
      <button
        type="button"
        onClick={onOpenMenu}
        aria-label="Open navigation"
        className="rounded-full p-1.5 text-[#444748] hover:bg-[#efeeeb] lg:hidden"
      >
        <Icon name="menu" size={22} />
      </button>

      <form onSubmit={jumpToRequest} role="search" className="relative hidden min-w-0 flex-1 sm:block">
        <Icon name="search" size={15} className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-[#444748]" />
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label="Go to request by ID"
          placeholder="Go to request ID, e.g. REQ-a1b2c3"
          className="w-full rounded-full bg-[#f4f3f0] py-2 pl-10 pr-12 text-[13px] text-[#1b1c1a] outline-none placeholder:text-[#444748] focus:ring-2 focus:ring-[#c7ef39]/60"
        />
        <kbd className="pointer-events-none absolute right-3.5 top-1/2 -translate-y-1/2 rounded bg-[#efeeeb] px-1 py-0.5 font-mono text-[12px] leading-4 text-[#444748]">
          ⌘K
        </kbd>
      </form>

      <div className="flex items-center gap-5">
        <Link
          href="/requests/new"
          className={`${fontHeading} flex items-center gap-1 rounded-full bg-black px-5 py-2 text-[16px] font-semibold leading-6 tracking-[-0.16px] text-white drop-shadow-[0px_1px_4px_rgba(0,0,0,0.08)] transition-colors hover:bg-[#1c1b1b]`}
        >
          <Icon name="add" size={14} />
          New request
        </Link>
        <Link
          href="/settings"
          aria-label="Account settings"
          className={`${fontHeading} flex size-8 items-center justify-center rounded-full bg-[#1b1c1a] text-[12px] font-bold text-white`}
        >
          {initials(user?.name)}
        </Link>
      </div>
    </header>
  );
}
