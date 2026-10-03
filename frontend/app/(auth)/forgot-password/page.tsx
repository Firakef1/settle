'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';

export default function ForgotPasswordPage() {
  const router = useRouter();
  const { forgotPassword, isLoading, error, clearError } = useAuthStore();

  const [email, setEmail] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  const [validationError, setValidationError] = useState('');

  const validateEmail = (email: string) => {
    return /\S+@\S+\.\S+/.test(email);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError('');

    if (!email) {
      setValidationError('Please enter your work email address');
      return;
    }

    if (!validateEmail(email)) {
      setValidationError('Please enter a valid email address');
      return;
    }

    clearError();

    try {
      await forgotPassword(email);
      setShowSuccess(true);
    } catch (err) {
      // Error handled by store
    }
  };

  const handleResend = () => {
    // In real app, call resendVerification
    console.log('Resending reset link to:', email);
  };

  const resetToForm = () => {
    setShowSuccess(false);
    setEmail('');
    clearError();
  };

  return (
    <div className="bg-[#FAF9F6] text-[#1C1B1B] font-[Inter] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0E0E0E]">
      {/* Header */}
      <header className="w-full bg-white/90 backdrop-blur-md border-b border-[#E8E7E3]/70 sticky top-0 z-50">
        <div className="h-16 max-w-7xl mx-auto px-6 lg:px-12 flex items-center justify-between">
          <Link className="flex items-center gap-2.5 transition-opacity hover:opacity-85" href="/login">
            <div className="w-8 h-8">
              <div className="w-full h-full bg-gradient-to-br from-[#B5F546] to-[#AFD519] rounded-lg flex items-center justify-center text-[#0E0E0E] font-bold text-xs">
                S
              </div>
            </div>
            <span className="font-[Plus_Jakarta_Sans] text-xl text-[#1C1B1B] font-bold tracking-tight">Settle</span>
          </Link>
          <div className="flex items-center gap-1.5 py-1.5 px-3.5 rounded-full bg-[#F0EFEA] border border-[#E8E7E3] text-xs font-medium text-[#1C1B1B]">
            <span className="w-2 h-2 rounded-full bg-[#B5F546] inline-block"></span>
            <span className="material-symbols-outlined text-[15px] text-[#1C1B1B]">lock</span>
            <span>256-Bit SSL Encrypted</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-[1.75rem] bg-[#FAF9F6] relative overflow-hidden">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-[#F2F1EC]/60 rounded-full blur-3xl pointer-events-none -z-10"></div>

        <div className="w-full max-w-[440px] mx-auto">
          {/* Header */}
          <div className="flex items-center justify-between px-2 pb-3.5">
            <div className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-[#525252]">
              <span className="material-symbols-outlined text-[15px] text-[#1C1B1B]">verified_user</span>
              <span>Directory Recovery</span>
            </div>
            <span className="text-xs font-mono text-[#787878] bg-[#F0EFEA] px-2 py-0.5 rounded-md border border-[#E8E7E3]">v2.4 Sec-Ops</span>
          </div>

          {/* Elevated Card Container */}
          <div className="relative w-full bg-white rounded-2xl border border-[#E8E7E3] shadow-[0_8px_30px_rgba(0,0,0,0.04)] p-7 sm:p-9 transition-all duration-300">
            {/* State Toggle */}
            <div className="mb-7 p-1 bg-[#F6F6F4] border border-[#E8E7E3]/80 rounded-xl">
              <button
                className="w-full flex items-center justify-center gap-2 py-1.5 px-3 rounded-lg text-xs font-semibold text-[#1C1B1B] hover:bg-white hover:shadow-xs transition-all cursor-pointer"
                onClick={() => setShowSuccess(!showSuccess)}
                type="button"
              >
                <span className="material-symbols-outlined text-[15px] text-[#1C1B1B]">{showSuccess ? 'arrow_back' : 'sync_alt'}</span>
                <span>{showSuccess ? 'Back to Form' : 'Preview Success'}</span>
              </button>
            </div>

            {/* Form State */}
            {!showSuccess ? (
              <div className="space-y-6 transition-opacity duration-200">
                {/* Icon & Heading */}
                <div className="flex flex-col items-center text-center space-y-2.5">
                  <div className="relative flex items-center justify-center w-16 h-16 rounded-full bg-[#F6F6F4] border border-[#E8E7E3] text-[#1C1B1B] mb-1">
                    <span className="material-symbols-outlined text-[30px]">lock_reset</span>
                    <div className="absolute -bottom-0.5 -right-0.5 w-6 h-6 rounded-full bg-[#1C1B1B] flex items-center justify-center text-[#B5F546] border-2 border-white shadow-sm">
                      <span className="material-symbols-outlined text-[13px] font-bold">vpn_key</span>
                    </div>
                  </div>
                  <h1 className="font-[Plus_Jakarta_Sans] text-[26px] font-bold text-[#1C1B1B] tracking-tight">Forgot Password?</h1>
                  <p className="font-[Inter] text-[15px] text-[#525252] leading-relaxed max-w-[320px]">
                    Enter your verified work email address and we'll send you instructions to reset your password.
                  </p>
                </div>

                {/* Form */}
                <form className="space-y-4" onSubmit={handleSubmit}>
                  <div className="space-y-1.5 text-left">
                    <label className="block text-xs font-semibold uppercase tracking-wider text-[#1C1B1B]" htmlFor="work-email">
                      Work Email Address
                    </label>
                    <div className="relative">
                      <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-neutral-400">
                        <span className="material-symbols-outlined text-[18px]">alternate_email</span>
                      </div>
                      <input
                        className="w-full h-12 pl-10 pr-4 bg-[#F6F6F4] hover:bg-[#F2F1ED] text-[#1C1B1B] placeholder:text-neutral-400 rounded-xl text-sm border border-[#E8E7E3] focus:bg-white focus:outline-none focus:ring-2 focus:ring-[#1C1B1B] focus:border-transparent transition-all"
                        id="work-email"
                        placeholder="name@company.com"
                        type="email"
                        value={email}
                        onChange={(e) => {
                          setEmail(e.target.value);
                          setValidationError('');
                        }}
                      />
                    </div>
                    {validationError && (
                      <p className="text-[#BA1A1A] text-xs flex items-center gap-1 pt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationError}
                      </p>
                    )}
                  </div>

                  <button
                    className="w-full h-12 bg-[#1C1B1B] hover:bg-[#222222] text-white font-medium text-sm rounded-full shadow-[0_4px_14px_rgba(0,0,0,0.12)] hover:shadow-[0_6px_20px_rgba(0,0,0,0.18)] transition-all flex items-center justify-center gap-2 group active:scale-[0.99] disabled:opacity-50 disabled:cursor-not-allowed"
                    disabled={isLoading}
                    type="submit"
                  >
                    <span>Send Reset Link</span>
                    <span className="material-symbols-outlined text-[18px] text-[#B5F546] group-hover:translate-x-1 transition-transform">arrow_forward</span>
                  </button>
                </form>

                {/* Enterprise Callout */}
                <div className="p-3.5 bg-[#F6F6F4]/80 rounded-xl border border-[#E8E7E3]/80 flex items-start gap-2.5 text-left">
                  <span className="material-symbols-outlined text-[18px] text-[#1C1B1B] shrink-0 mt-0.5">policy</span>
                  <p className="font-[Inter] text-[13px] text-[#525252] leading-relaxed">
                    Enterprise SSO enabled? Authenticate directly through your organization's identity portal or reach out to internal IT.
                  </p>
                </div>

                {/* Return to Sign In */}
                <div className="text-center pt-1 border-t border-[#E8E7E3]/60">
                  <p className="font-[Inter] text-xs text-[#525252] pt-4">
                    Remember your password?{' '}
                    <Link
                      className="font-semibold text-[#1C1B1B] hover:text-black inline-flex items-center gap-0.5 ml-1 group underline decoration-[#B5F546] underline-offset-4 decoration-2"
                      href="/login"
                    >
                      Sign in
                      <span className="material-symbols-outlined text-[14px] group-hover:translate-x-0.5 transition-transform">chevron_right</span>
                    </Link>
                  </p>
                </div>
              </div>
            ) : (
              /* Success State */
              <div className="space-y-6 text-center transition-opacity duration-200">
                {/* Success Graphic */}
                <div className="flex flex-col items-center space-y-2.5">
                  <div className="relative flex items-center justify-center w-16 h-16 rounded-full bg-[#B5F546] text-[#1C1B1B] mb-1 shadow-sm">
                    <span className="material-symbols-outlined text-[32px] font-bold">check</span>
                    <div className="absolute -bottom-0.5 -right-0.5 w-6 h-6 rounded-full bg-[#1C1B1B] flex items-center justify-center text-[#B5F546] border-2 border-white">
                      <span className="material-symbols-outlined text-[13px]">mail</span>
                    </div>
                  </div>
                  <h2 className="font-[Plus_Jakarta_Sans] text-[26px] font-bold text-[#1C1B1B] tracking-tight">Check Your Email</h2>
                  <div className="space-y-2 max-w-[320px]">
                    <p className="font-[Inter] text-[15px] text-[#525252]">
                      We've sent a one-time password reset link to
                    </p>
                    <div className="inline-flex items-center gap-1.5 bg-[#F6F6F4] border border-[#E8E7E3] px-3.5 py-1.5 rounded-full text-xs font-mono font-medium text-[#1C1B1B] break-all">
                      {email || 'sarah.chen@acmelabs.io'}
                    </div>
                    <p className="font-[Inter] text-[13px] text-neutral-500 flex items-center justify-center gap-1 pt-1">
                      <span className="material-symbols-outlined text-[15px] text-[#1C1B1B]">schedule</span>
                      The security link expires in 24 hours.
                    </p>
                  </div>
                </div>

                {/* Security Checklist */}
                <div className="bg-[#F6F6F4] rounded-xl border border-[#E8E7E3] p-3.5 text-left space-y-2.5">
                  <div className="flex items-center gap-2 font-[Inter] text-[13px] font-medium text-[#1C1B1B]">
                    <span className="w-4 h-4 rounded-full bg-[#B5F546] text-[#1C1B1B] flex items-center justify-center text-[10px] font-bold shrink-0">✓</span>
                    <span>Domain match verified &amp; secure token issued</span>
                  </div>
                  <div className="flex items-center gap-2 font-[Inter] text-[13px] font-medium text-[#1C1B1B]">
                    <span className="w-4 h-4 rounded-full bg-[#B5F546] text-[#1C1B1B] flex items-center justify-center text-[10px] font-bold shrink-0">✓</span>
                    <span>Single-use encrypted authentication dispatch</span>
                  </div>
                </div>

                {/* Actions */}
                <div className="space-y-3">
                  <button
                    className="w-full h-12 bg-[#1C1B1B] hover:bg-[#222222] text-white font-medium text-sm rounded-full shadow-[0_4px_14px_rgba(0,0,0,0.12)] transition-all flex items-center justify-center gap-2 group cursor-pointer"
                    onClick={resetToForm}
                    type="button"
                  >
                    <span className="material-symbols-outlined text-[18px] text-[#B5F546] group-hover:-translate-x-0.5 transition-transform">arrow_back</span>
                    <span>Back to Sign In</span>
                  </button>
                  <div className="pt-1">
                    <p className="font-[Inter] text-xs text-[#525252]">
                      Didn't receive email? Check spam folder or{' '}
                      <button
                        className="font-semibold text-[#1C1B1B] underline decoration-[#B5F546] underline-offset-4 decoration-2 hover:opacity-80 ml-1 cursor-pointer"
                        onClick={handleResend}
                        type="button"
                      >
                        resend link
                      </button>
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Enterprise Footnote */}
          <div className="mt-6 flex items-center justify-center gap-4 font-[Inter] text-xs font-medium text-[#525252]">
            <div className="flex items-center gap-1.5">
              <span className="material-symbols-outlined text-[16px] text-[#1C1B1B]">key</span>
              <span>Hardware Key Support</span>
            </div>
            <span className="text-neutral-300">•</span>
            <div className="flex items-center gap-1.5">
              <span className="material-symbols-outlined text-[16px] text-[#1C1B1B]">verified</span>
              <span>SOC2 Type II Certified</span>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-[#FAF9F6] border-t border-[#E8E7E3]/70 py-6">
        <div className="max-w-7xl mx-auto px-6 lg:px-12 flex flex-col sm:flex-row items-center justify-between gap-4 text-[#525252]">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]"></span>
            <span>© 2025 Settle Technologies Inc. Kinetic Capital Operations.</span>
          </div>
          <div className="flex items-center gap-6">
            <Link className="hover:text-[#1C1B1B] transition-colors" href="#">Privacy Policy</Link>
            <Link className="hover:text-[#1C1B1B] transition-colors" href="#">Terms of Service</Link>
            <Link className="hover:text-[#1C1B1B] transition-colors" href="#">Security Operations</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}