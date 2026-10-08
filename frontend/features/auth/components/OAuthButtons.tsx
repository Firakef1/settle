'use client';

import { useQuery } from '@tanstack/react-query';
import { authAPI } from '../../../shared/services/authAPI';

const PROVIDERS: Record<string, { label: string; logo: React.ReactNode }> = {
  google: {
    label: 'Google',
    logo: (
      <svg width="16" height="16" viewBox="0 0 48 48" aria-hidden>
        <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.7 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.2-.1-2.3-.4-3.5z" />
        <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.7 15.1 19 12 24 12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16.3 4 9.7 8.3 6.3 14.7z" />
        <path fill="#4CAF50" d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.1 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.5 39.6 16.2 44 24 44z" />
        <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.2-2.2 4.2-4.1 5.6l6.2 5.2C37 39.2 44 34 44 24c0-1.2-.1-2.3-.4-3.5z" />
      </svg>
    ),
  },
  microsoft: {
    label: 'Microsoft',
    logo: (
      <svg width="15" height="15" viewBox="0 0 23 23" aria-hidden>
        <path fill="#f25022" d="M1 1h10v10H1z" />
        <path fill="#7fba00" d="M12 1h10v10H12z" />
        <path fill="#00a4ef" d="M1 12h10v10H1z" />
        <path fill="#ffb900" d="M12 12h10v10H12z" />
      </svg>
    ),
  },
};

// "Continue with Google / Microsoft". Renders nothing unless the server has
// at least one provider configured, so there are never dead buttons.
export function OAuthButtons({ next, label = 'Or continue with' }: { next?: string; label?: string }) {
  const providers = useQuery({ queryKey: ['auth', 'oauth-providers'], queryFn: authAPI.oauthProviders, staleTime: 5 * 60 * 1000, retry: false });
  const enabled = (providers.data ?? []).filter((p) => PROVIDERS[p]);
  if (enabled.length === 0) return null;

  return (
    <div className="mt-6 border-t border-[#E5E7EB] pt-5">
      <p className="-mt-8 mb-4 flex justify-center">
        <span className="bg-white px-3 text-xs font-semibold uppercase tracking-wider text-[#737775]">{label}</span>
      </p>
      <div className={`grid gap-3 ${enabled.length > 1 ? 'grid-cols-2' : 'grid-cols-1'}`}>
        {enabled.map((p) => (
          <a
            key={p}
            href={authAPI.oauthStartUrl(p, next)}
            className="flex h-10 items-center justify-center gap-2 rounded-full border border-[#E5E7EB] bg-[#F4F4F1] px-3 text-xs font-semibold text-[#111413] shadow-sm transition-all hover:bg-[#E8E7E2]"
          >
            {PROVIDERS[p].logo}
            <span>Continue with {PROVIDERS[p].label}</span>
          </a>
        ))}
      </div>
    </div>
  );
}

export const OAUTH_ERRORS: Record<string, string> = {
  cancelled: 'Sign-in was cancelled.',
  state: 'That sign-in link expired or was already used. Please try again.',
  unavailable: "That sign-in option isn't available right now.",
  no_email: "Your account didn't share an email address, so we couldn't sign you in.",
  account_exists:
    'An account with this email already exists. Sign in with your password once; the provider can only be used for accounts it verifies.',
  failed: "We couldn't complete sign-in with that provider. Please try again.",
};
