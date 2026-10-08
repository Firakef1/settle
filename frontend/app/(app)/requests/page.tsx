'use client';

import { Suspense } from 'react';
import { RequestsList } from '../../../features/requests/components/RequestsList';

// useSearchParams() (filters in the URL) needs a Suspense boundary.
export default function RequestsPage() {
  return (
    <Suspense fallback={null}>
      <RequestsList />
    </Suspense>
  );
}
