'use client';

import { useParams } from 'next/navigation';
import { RequestDetail } from '../../../../features/requests/components/RequestDetail';

export default function RequestDetailPage() {
  const { id } = useParams<{ id: string }>();
  return <RequestDetail id={decodeURIComponent(id)} />;
}
