'use client';

import { useParams } from 'next/navigation';
import { ResubmitForm } from '../../../../../features/requests/components/ResubmitForm';

export default function ResubmitPage() {
  const { id } = useParams<{ id: string }>();
  return <ResubmitForm id={decodeURIComponent(id)} />;
}
