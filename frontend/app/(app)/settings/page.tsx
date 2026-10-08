import { OrgSettings } from '../../../features/admin/components/OrgSettings';

// Everyone can see the organization; only org_admin can edit it.
export default function SettingsPage() {
  return <OrgSettings />;
}
