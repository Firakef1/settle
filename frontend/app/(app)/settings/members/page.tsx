'use client';

import { RoleGuard } from '../../../auth/route-guard';
import { MembersPage } from '../../../../features/admin/components/MembersPage';

// org_admin manages members; finance can view.
export default function SettingsMembersPage() {
  return (
    <RoleGuard allowedRoles={['finance', 'org_admin']}>
      <MembersPage />
    </RoleGuard>
  );
}
