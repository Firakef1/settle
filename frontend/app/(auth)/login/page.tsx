'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';
import { getApiError } from '../../../shared/utils/apiError';
import { authAPI } from '../../../shared/services/authAPI';
import { OAUTH_ERRORS, OAuthButtons } from '../../../features/auth/components/OAuthButtons';

export default function LoginPage() {
  const router = useRouter();
  const { login, isLoading, error, clearError } = useAuthStore();

  const [formData, setFormData] = useState({
    email: '',
    password: '',
  });
  const [rememberMe, setRememberMe] = useState(true);
  const [showPassword, setShowPassword] = useState(false);
  const [invited, setInvited] = useState(false);
  const [oauthError, setOauthError] = useState('');
  const [nextPath, setNextPath] = useState<string | undefined>();

  // Arriving from an accepted invite: /login?invited=1&email=…
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('invited') === '1') setInvited(true);
    const oauth = params.get('oauth_error');
    if (oauth) setOauthError(OAUTH_ERRORS[oauth] ?? OAUTH_ERRORS.failed);
    const next = params.get('next');
    if (next && next.startsWith('/') && !next.startsWith('//')) setNextPath(next);
    const email = params.get('email');
    if (email) setFormData((prev) => ({ ...prev, email }));
  }, []);
  const [validationErrors, setValidationErrors] = useState<{[key: string]: string}>({});

  const validateForm = () => {
    const errors: {[key: string]: string} = {};

    if (!formData.email) {
      errors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      errors.email = 'Please enter a valid email address';
    }

    if (!formData.password) {
      errors.password = 'Password is required';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    clearError();

    try {
      await login(formData.email, formData.password);

      // The token carries the first org; with none, send them to create or join one.
      const { orgs } = useAuthStore.getState();
      const next = new URLSearchParams(window.location.search).get('next');
      if (orgs.length === 0) {
        router.push('/select-organization');
      } else {
        router.push(next && next.startsWith('/') ? next : '/dashboard');
      }
    } catch (err) {
      // 403 + email_not_verified: the password was right but the email isn't verified yet.
      if (getApiError(err).code === 'email_not_verified') {
        clearError();
        // Accounts created from an invite were never sent a code, so send one now.
        await authAPI.resendVerification(formData.email).catch(() => undefined);
        router.push(`/verify-email?email=${encodeURIComponent(formData.email)}`);
      }
      // Other errors are shown from the store.
    }
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (validationErrors[field]) {
      setValidationErrors(prev => ({ ...prev, [field]: '' }));
    }
    clearError();
  };

  return (
    <div className="bg-[#F6F6F4] text-[#111413] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0E0E0E] font-[Inter]">
      {/* Header */}
      <header className="w-full bg-white/90 backdrop-blur-md border-b border-[#E5E7EB]">
        <div className="h-16 max-w-7xl mx-auto px-6 lg:px-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded bg-[#0E0E0E] flex items-center justify-center">
              <span className="text-[#B5F546] font-bold text-[10px]">S</span>
            </div>
            <span className="font-[Plus_Jakarta_Sans] text-xl text-[#0E0E0E] font-bold tracking-tight">Settle</span>
          </div>
          <div className="flex items-center gap-2 py-1.5 px-3.5 rounded-full bg-[#F4F4F1] border border-[#E5E7EB] text-[#737775]">
            <span className="material-symbols-outlined text-[16px] text-[#0E0E0E]">lock</span>
            <span className="text-xs font-medium tracking-wide">256-Bit SSL Encrypted</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-8 sm:py-12 bg-[#F6F6F4]">
        <div className="flex flex-col w-full max-w-7xl mx-auto">
          {/* Main Split Auth Card */}
          <div className="w-full bg-white rounded-2xl border border-[#E5E7EB] shadow-xl shadow-black/[0.04] overflow-hidden flex flex-col lg:flex-row">
            {/* Left Side: Dark Panel (60% width) */}
            <div className="relative w-full lg:w-[58%] p-8 sm:p-12 lg:p-16 flex flex-col justify-between overflow-hidden bg-[#0E0E0E] text-white min-h-[520px] lg:min-h-[660px]">
              {/* Ambient Subtle Dot Grid */}
              <div
                className="absolute inset-0 opacity-15 pointer-events-none"
                style={{
                  backgroundImage: 'radial-gradient(#B5F546 1px, transparent 1px)',
                  backgroundSize: '28px 28px'
                }}
              />

              {/* Subtle Glow Depth */}
              <div className="absolute -top-32 -left-32 w-96 h-96 bg-[#B5F546]/10 rounded-full blur-3xl pointer-events-none" />
              <div className="absolute -bottom-32 -right-32 w-96 h-96 bg-[#C8F03A]/15 rounded-full blur-3xl pointer-events-none" />

              {/* Top Lockup */}
              <div className="relative z-10 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-[#1B1E1D] border border-white/10 flex items-center justify-center p-1.5 shadow-inner">
                    <div className="w-5 h-5 rounded-sm bg-[#B5F546] flex items-center justify-center">
                      <span className="w-2.5 h-2.5 bg-[#0E0E0E] rounded-[1px]" />
                    </div>
                  </div>
                  <span className="font-[Plus_Jakarta_Sans] text-xl tracking-tight text-white font-bold">Settle</span>
                </div>
                <div className="hidden sm:flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-white/90">
                  <span className="w-2 h-2 rounded-full bg-[#B5F546] animate-pulse" />
                  <span className="text-xs font-mono font-medium tracking-wide">Continuous Engine v4.8</span>
                </div>
              </div>

              {/* Center Messaging & Stats */}
              <div className="relative z-10 my-10 lg:my-14">
                {/* Neon Lime Badge */}
                <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-[#B5F546] text-[#0E0E0E] mb-6 shadow-sm shadow-[#B5F546]/20 font-semibold text-xs uppercase tracking-wider">
                  <span className="material-symbols-outlined text-[16px]">bolt</span>
                  <span>Continuous Settlement</span>
                </div>

                <h1 className="font-[Plus_Jakarta_Sans] text-3xl sm:text-4xl lg:text-[42px] text-white font-extrabold tracking-tight mb-4 leading-[1.15]">
                  Simplify Expense<br className="hidden sm:block" /> Management
                </h1>

                <p className="text-white/70 max-w-lg mb-9 leading-relaxed text-sm sm:text-base">
                  High-velocity precision and institutional spend controls engineered for modern hyper-growth operations.
                </p>

                {/* Floating Metric Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 max-w-lg">
                  <div className="p-4 rounded-xl bg-[#1B1E1D]/90 border border-white/10 backdrop-blur-md shadow-lg flex items-center gap-3.5 group hover:border-[#B5F546]/40 transition-colors">
                    <div className="w-11 h-11 rounded-lg bg-[#B5F546]/15 border border-[#B5F546]/30 text-[#B5F546] flex items-center justify-center shrink-0">
                      <span className="material-symbols-outlined text-[22px]">speed</span>
                    </div>
                    <div className="min-w-0">
                      <div className="text-xs text-white/60 font-medium">Reconciliation Velocity</div>
                      <div className="font-[Plus_Jakarta_Sans] text-xl text-white font-bold tracking-tight flex items-baseline gap-1">
                        0.42 <span className="text-xs font-mono text-[#B5F546] font-normal">sec</span>
                      </div>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl bg-[#1B1E1D]/90 border border-white/10 backdrop-blur-md shadow-lg flex items-center gap-3.5 group hover:border-[#B5F546]/40 transition-colors">
                    <div className="w-11 h-11 rounded-lg bg-white/10 border border-white/15 text-[#C8F03A] flex items-center justify-center shrink-0">
                      <span className="material-symbols-outlined text-[22px]">verified_user</span>
                    </div>
                    <div className="min-w-0">
                      <div className="text-xs text-white/60 font-medium">Fraud Prevention Rate</div>
                      <div className="font-[Plus_Jakarta_Sans] text-xl text-white font-bold tracking-tight">99.98%</div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Bottom Status */}
              <div className="relative z-10 pt-4 border-t border-white/10 flex flex-wrap items-center justify-between gap-3 text-white/50 text-xs">
                <div className="flex items-center gap-2">
                  <span className="material-symbols-outlined text-[16px] text-[#B5F546]">verified</span>
                  <span className="font-mono">PCI-DSS Level 1 · SOC2 Type II</span>
                </div>
                <div className="hidden sm:block">
                  <span>Global Multi-Currency Hub</span>
                </div>
              </div>
            </div>

            {/* Right Side: Form Panel (42% width) */}
            <div className="w-full lg:w-[42%] p-8 sm:p-12 lg:p-14 flex flex-col justify-between bg-white">
              <div className="w-full">
                {/* Form Header */}
                <div className="mb-8">
                  <div className="mb-6 flex items-center justify-center">
                    <div className="w-10 h-10 rounded-lg bg-[#0E0E0E] flex items-center justify-center">
                      <span className="text-[#B5F546] font-bold text-sm">SETTLE</span>
                    </div>
                  </div>
                  <h2 className="font-[Plus_Jakarta_Sans] text-2xl sm:text-3xl text-[#111413] font-bold tracking-tight mb-2">
                    Welcome Back
                  </h2>
                  <p className="text-sm text-[#737775] font-normal">
                    Sign in to manage your institutional accounts
                  </p>
                </div>

                {oauthError && !error && (
                  <div role="alert" className="mb-6 flex items-start gap-2.5 rounded-xl bg-[#FFDAD6] p-4 text-[#93000A]">
                    <span className="material-symbols-outlined text-[20px] text-[#BA1A1A]">error</span>
                    <span className="text-sm font-medium">{oauthError}</span>
                  </div>
                )}

                {invited && !error && (
                  <div className="mb-6 flex items-center gap-2.5 rounded-xl bg-[#EEF9D6] p-4 text-[#2F3D00]">
                    <span className="material-symbols-outlined text-[20px] text-[#526600]">check_circle</span>
                    <span className="text-sm font-medium">You&apos;ve joined the organization. Sign in to continue.</span>
                  </div>
                )}

                {/* Error Banner */}
                {error && (
                  <div className="mb-6 p-4 rounded-xl bg-[#FFDAD6] text-[#93000A] border border-[#BA1A1A]/20 flex items-start justify-between gap-3 transition-all duration-300">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[#BA1A1A] text-[20px] shrink-0">error</span>
                      <span className="text-sm font-medium">{error}</span>
                    </div>
                    <button
                      aria-label="Dismiss error"
                      className="text-[#93000A]/70 hover:text-[#93000A] p-0.5 rounded transition-colors shrink-0"
                      onClick={clearError}
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[18px]">close</span>
                    </button>
                  </div>
                )}

                {/* Login Form */}
                <form className="space-y-4" onSubmit={handleSubmit}>
                  {/* Email Input */}
                  <div className="space-y-1.5">
                    <label className="block text-xs font-semibold uppercase tracking-wider text-[#737775]" htmlFor="email">
                      Email Address
                    </label>
                    <div className="relative">
                      <input
                        className="w-full h-12 px-4 rounded-xl bg-[#F4F4F1]/60 border border-[#E5E7EB] text-[#111413] placeholder:text-[#737775]/50 text-sm outline-none focus:bg-white focus:border-[#0E0E0E] focus:ring-2 focus:ring-[#B5F546]/50 transition-all font-medium"
                        id="email"
                        placeholder="you@company.com"
                        required
                        type="email"
                        value={formData.email}
                        onChange={(e) => handleInputChange('email', e.target.value)}
                      />
                    </div>
                    {validationErrors.email && (
                      <p className="text-xs text-[#BA1A1A] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.email}
                      </p>
                    )}
                  </div>

                  {/* Password Input */}
                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between">
                      <label className="block text-xs font-semibold uppercase tracking-wider text-[#737775]" htmlFor="password">
                        Password
                      </label>
                      <Link className="text-xs text-[#0E0E0E] font-semibold hover:text-[#A6E832] underline transition-all" href="/forgot-password">
                        Forgot password?
                      </Link>
                    </div>
                    <div className="relative">
                      <input
                        className="w-full h-12 pl-4 pr-12 rounded-xl bg-[#F4F4F1]/60 border border-[#E5E7EB] text-[#111413] placeholder:text-[#737775]/50 text-sm outline-none focus:bg-white focus:border-[#0E0E0E] focus:ring-2 focus:ring-[#B5F546]/50 transition-all font-medium font-mono"
                        id="password"
                        placeholder="••••••••"
                        required
                        type={showPassword ? "text" : "password"}
                        value={formData.password}
                        onChange={(e) => handleInputChange('password', e.target.value)}
                      />
                      <button
                        aria-label="Toggle password visibility"
                        className="absolute inset-y-0 right-0 px-3.5 flex items-center text-[#737775] hover:text-[#111413] transition-colors"
                        onClick={() => setShowPassword(!showPassword)}
                        type="button"
                      >
                        <span className="material-symbols-outlined text-[20px]">
                          {showPassword ? 'visibility_off' : 'visibility'}
                        </span>
                      </button>
                    </div>
                    {validationErrors.password && (
                      <p className="text-xs text-[#BA1A1A] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.password}
                      </p>
                    )}
                  </div>

                  {/* Remember Me Checkbox */}
                  <div className="pt-1 flex items-center justify-between">
                    <label className="flex items-center gap-2.5 cursor-pointer select-none group">
                      <input
                        checked={rememberMe}
                        className="w-4 h-4 rounded text-[#0E0E0E] bg-[#F4F4F1] border border-[#E5E7EB] focus:ring-0 cursor-pointer accent-[#0E0E0E]"
                        id="remember-me"
                        type="checkbox"
                        onChange={(e) => setRememberMe(e.target.checked)}
                      />
                      <span className="text-sm text-[#737775] group-hover:text-[#111413] transition-colors font-medium">
                        Remember me for 30 days
                      </span>
                    </label>
                  </div>

                  {/* Submit Button */}
                  <div className="pt-3">
                    <button
                      className="w-full h-12 rounded-full bg-[#0E0E0E] hover:bg-neutral-800 active:scale-[0.99] text-white font-[Plus_Jakarta_Sans] text-sm font-semibold tracking-wide shadow-md shadow-[#0E0E0E]/20 transition-all flex items-center justify-center gap-2 group disabled:opacity-50 disabled:cursor-not-allowed"
                      disabled={isLoading}
                      type="submit"
                    >
                      {isLoading ? (
                        <div className="flex items-center gap-2">
                          <svg className="animate-spin h-5 w-5 text-[#B5F546]" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                            <path className="opacity-75" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" fill="currentColor" />
                          </svg>
                          <span className="text-sm font-medium">Verifying credentials...</span>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <span>Sign In</span>
                          <span className="w-5 h-5 rounded-full bg-[#B5F546] text-[#0E0E0E] flex items-center justify-center group-hover:translate-x-0.5 transition-transform">
                            <span className="material-symbols-outlined text-[14px] font-bold">arrow_forward</span>
                          </span>
                        </div>
                      )}
                    </button>
                  </div>
                </form>

                <OAuthButtons next={nextPath} />
              </div>

              {/* Sign Up Link */}
              <div className="mt-8 pt-4 text-center text-sm text-[#737775]">
                Don&apos;t have an account?
                <Link className="font-semibold text-[#0E0E0E] hover:underline underline-offset-4 ml-1" href="/signup">
                  Sign up
                </Link>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-white border-t border-[#E5E7EB] py-5">
        <div className="max-w-7xl mx-auto px-6 lg:px-12 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-[#737775] font-medium">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[16px] text-[#0E0E0E]">verified_user</span>
            <span>© 2025 Settle Technologies Inc. All rights reserved.</span>
          </div>
          <div className="flex items-center gap-6">
            <Link className="hover:text-[#0E0E0E] transition-colors" href="#">Privacy Policy</Link>
            <Link className="hover:text-[#0E0E0E] transition-colors" href="#">Terms of Service</Link>
            <Link className="hover:text-[#0E0E0E] transition-colors" href="#">Security Operations</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
