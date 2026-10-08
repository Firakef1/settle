'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Icon } from '../../../shared/components/Icon';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Role } from '../../../shared/types';

const TABS: Array<{ href: string; label: string; icon: string; roles: Role[] }> = [
  { href: '/settings', label: 'Organization', icon: 'domain', roles: ['staff', 'finance', 'org_admin'] },
  { href: '/settings/members', label: 'Members', icon: 'group', roles: ['finance', 'org_admin'] },
  { href: '/settings/billing', label: 'Billing & plan', icon: 'credit_card', roles: ['finance', 'org_admin'] },
  { href: '/settings/audit-log', label: 'Audit log', icon: 'description', roles: ['finance', 'org_admin'] },
  { href: '/settings/account', label: 'Account', icon: 'person', roles: ['staff', 'finance', 'org_admin'] },
];

// Tabs shown above every /settings page, filtered by role.
export function SettingsTabs() {
  const pathname = usePathname();
  const role = (useAuthStore((s) => s.currentOrg?.role) ?? 'staff') as Role;
  return (
    <nav aria-label="Settings" className="-mx-1 mb-6 flex gap-1 overflow-x-auto border-b border-[#e3e2df] px-1">
      {TABS.filter((t) => t.roles.includes(role)).map((t) => {
        const active = t.href === '/settings' ? pathname === '/settings' : pathname.startsWith(t.href);
        return (
          <Link
            key={t.href}
            href={t.href}
            aria-current={active ? 'page' : undefined}
            className={`-mb-px flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-2.5 text-[13px] transition-colors ${
              active ? 'border-[#1b1c1a] font-semibold text-[#1b1c1a]' : 'border-transparent text-[#444748] hover:text-[#1b1c1a]'
            }`}
          >
            <Icon name={t.icon} size={15} />
            {t.label}
          </Link>
        );
      })}
    </nav>
  );
}
