import { RouteGuard } from '../auth/route-guard';
import { AppLayout } from '@/shared/layout';

// Every signed-in page: auth/org guard, then the sidebar + header shell.
export default function AppShellLayout({ children }: { children: React.ReactNode }) {
  return (
    <RouteGuard>
      <AppLayout>{children}</AppLayout>
    </RouteGuard>
  );
}
