'use client';

import { RoleGuard } from '../../../auth/route-guard';
import { BillingPage } from '../../../../features/admin/components/BillingPage';

// org_admin changes the plan; finance can view usage (stats need finance or admin).
export default function SettingsBillingPage() {
  return (
    <RoleGuard allowedRoles={['finance', 'org_admin']}>
      <BillingPage />
    </RoleGuard>
  );
}
