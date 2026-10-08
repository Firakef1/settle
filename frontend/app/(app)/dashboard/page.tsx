'use client';

import { useAuthStore } from '../../../shared/stores/authStore';
import type { Role } from '../../../shared/types';
import { useOrganization } from '../../../features/dashboard/api';
import { FinanceDashboard } from '../../../features/dashboard/components/FinanceDashboard';
import { StaffDashboard } from '../../../features/dashboard/components/StaffDashboard';

// One route for every role: finance/org_admin get the queue overview,
// staff get their own requests (they receive 403 from summary and stats).
// RouteGuard guarantees a hydrated session with a current org here.
export default function DashboardPage() {
  const { user, currentOrg } = useAuthStore();
  if (!currentOrg) return null;

  const role = currentOrg.role as Role;
  const props = { orgId: currentOrg.org_id, orgName: currentOrg.org_name, role, userName: user?.name };

  if (role === 'finance' || role === 'org_admin') return <FinanceDashboard {...props} />;
  return <StaffHome {...props} />;
}

function StaffHome(props: { orgId: string; orgName: string; role: Role; userName?: string }) {
  // Currency comes from the organization; fall back to USD while it loads or if it fails.
  const org = useOrganization(props.orgId);
  return <StaffDashboard {...props} currency={org.data?.currency ?? 'USD'} />;
}
