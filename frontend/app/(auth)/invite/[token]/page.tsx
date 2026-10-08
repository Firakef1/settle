'use client';

import { useParams } from 'next/navigation';
import { AcceptInvite } from '../../../../features/admin/components/AcceptInvite';

export default function InviteAcceptPage() {
  const { token } = useParams<{ token: string }>();
  return <AcceptInvite token={token} />;
}
