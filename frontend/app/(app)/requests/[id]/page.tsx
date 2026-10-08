'use client';

import { useParams } from 'next/navigation';
import { FinancePanel } from '../../../../features/approvals/components/FinancePanel';
import { RequestDetail } from '../../../../features/requests/components/RequestDetail';
import { useAuthStore } from '../../../../shared/stores/authStore';

export default function RequestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const role = useAuthStore((s) => s.currentOrg?.role);
  const canDecide = role === 'finance' || role === 'org_admin';

  return (
    <RequestDetail
      id={decodeURIComponent(id)}
      aside={(request, currency) => (canDecide && request.status !== 'draft' ? <FinancePanel request={request} currency={currency} /> : null)}
    />
  );
}
