'use client';

import { useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../../shared/stores/authStore';

// The backend redirects here after "Sign in with Google/Microsoft" with the
// session in the URL fragment (#token=…&refresh_token=…&next=…).
export default function OAuthCallbackPage() {
  const router = useRouter();
  const signInWithTokens = useAuthStore((s) => s.signInWithTokens);
  const done = useRef(false);

  useEffect(() => {
    if (done.current) return;
    done.current = true;

    const params = new URLSearchParams(window.location.hash.slice(1));
    // Drop the tokens from the address bar and history right away.
    window.history.replaceState(null, '', window.location.pathname);

    const token = params.get('token');
    const refreshToken = params.get('refresh_token');
    const next = params.get('next');
    if (!token || !refreshToken) {
      router.replace('/login?oauth_error=failed');
      return;
    }

    signInWithTokens(token, refreshToken)
      .then(() => {
        const { orgs } = useAuthStore.getState();
        if (orgs.length === 0) router.replace('/select-organization');
        else router.replace(next && next.startsWith('/') && !next.startsWith('//') ? next : '/dashboard');
      })
      .catch(() => router.replace('/login?oauth_error=failed'));
  }, [router, signInWithTokens]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#FAF9F6] text-[14px] text-[#444748]">
      <span className="material-symbols-outlined mr-2 animate-spin text-[20px] text-[#B5F546]">progress_activity</span>
      Signing you in…
    </main>
  );
}
