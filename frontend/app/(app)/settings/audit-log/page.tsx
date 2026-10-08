'use client';

import { Suspense } from 'react';
import { RoleGuard } from '../../../auth/route-guard';
import { AuditLog } from '../../../../features/admin/components/AuditLog';

export default function AuditLogPage() {
  return (
    <RoleGuard allowedRoles={['finance', 'org_admin']}>
      {/* Filters live in the URL (useSearchParams) so views can be shared. */}
      <Suspense fallback={null}>
        <AuditLog />
      </Suspense>
    </RoleGuard>
  );
}
