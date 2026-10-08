'use client';

import { Suspense } from 'react';
import { RoleGuard } from '../../auth/route-guard';
import { ApprovalsQueue } from '../../../features/approvals/components/ApprovalsQueue';

export default function ApprovalsPage() {
  return (
    <RoleGuard allowedRoles={['finance', 'org_admin']}>
      {/* useSearchParams() (tabs and filters in the URL) needs a Suspense boundary. */}
      <Suspense fallback={null}>
        <ApprovalsQueue />
      </Suspense>
    </RoleGuard>
  );
}
