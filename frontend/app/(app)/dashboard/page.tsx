'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Role } from '../../../shared/types';
import { useOrganization } from '../../../features/dashboard/api';
import { FinanceDashboard } from '../../../features/dashboard/components/FinanceDashboard';
import { StaffDashboard } from '../../../features/dashboard/components/StaffDashboard';
import { DashboardSkeleton, Icon, fontBody, fontHeading } from '../../../features/dashboard/components/ui';

// One route for every role: finance/org_admin get the queue overview,
// staff get their own requests (they receive 403 from summary and stats).
export default function DashboardPage() {
  const { user, currentOrg } = useAuthStore();
  // The auth store is persisted to localStorage, so wait for the client
  // before reading it to avoid a server/client hydration mismatch.
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  return (
    <main className={`${fontBody} min-h-screen bg-[#faf9f6] px-5 py-8 text-[#1b1c1a] sm:px-7`}>
      <div className="mx-auto w-full max-w-[1000px]">
        {!mounted ? (
          <DashboardSkeleton />
        ) : !currentOrg ? (
          <NoOrganization />
        ) : (
          <RoleDashboard
            orgId={currentOrg.org_id}
            orgName={currentOrg.org_name}
            role={currentOrg.role as Role}
            userName={user?.name}
          />
        )}
      </div>
    </main>
  );
}

function RoleDashboard({ orgId, orgName, role, userName }: { orgId: string; orgName: string; role: Role; userName?: string }) {
  if (role === 'finance' || role === 'org_admin') {
    return <FinanceDashboard orgId={orgId} orgName={orgName} role={role} userName={userName} />;
  }
  return <StaffHome orgId={orgId} orgName={orgName} role={role} userName={userName} />;
}

function StaffHome({ orgId, orgName, role, userName }: { orgId: string; orgName: string; role: Role; userName?: string }) {
  // Currency comes from the organization; fall back to USD while it loads or if it fails.
  const org = useOrganization(orgId);
  return (
    <StaffDashboard orgId={orgId} orgName={orgName} role={role} userName={userName} currency={org.data?.currency ?? 'USD'} />
  );
}

function NoOrganization() {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl bg-white px-6 py-16 text-center shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
      <Icon name="domain_add" size={28} className="text-[#444748]" />
      <h1 className={`${fontHeading} text-[18px] font-bold text-[#1b1c1a]`}>No organization selected</h1>
      <p className="max-w-md text-[13px] text-[#444748]">
        Choose or create an organization to see your dashboard.
      </p>
      <Link
        href="/select-organization"
        className={`${fontHeading} rounded-full bg-[#0e0e0e] px-4 py-2 text-[13px] font-semibold text-white hover:bg-[#1c1b1b]`}
      >
        Select organization
      </Link>
    </div>
  );
}
