'use client';

import { Suspense, useState, useEffect, useMemo } from 'react';
import { useSearchParams } from 'next/navigation';
import { useAuthStore } from '../../../../shared/stores/authStore';
import Link from 'next/link';

interface PasswordRequirement {
  label: string;
  test: (password: string) => boolean;
}

type PageState = 'form' | 'success' | 'expired';

const passwordRequirements: PasswordRequirement[] = [
  { label: 'At least 8 characters', test: (p) => p.length >= 8 },
  { label: 'One uppercase letter', test: (p) => /[A-Z]/.test(p) },
  { label: 'One lowercase letter', test: (p) => /[a-z]/.test(p) },
  { label: 'One number', test: (p) => /\d/.test(p) },
  { label: 'One special character', test: (p) => /[!@#$%^&*(),.?":{}|<>]/.test(p) },
];

function ResetPasswordContent({ params }: { params: { token: string } }) {
  const searchParams = useSearchParams();
  const { resetPassword, isLoading, error, clearError } = useAuthStore();

  const [pageState, setPageState] = useState<PageState>('form');
  const [formData, setFormData] = useState({
    password: '',
    confirmPassword: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [validationErrors, setValidationErrors] = useState<{[key: string]: string}>({});

  // Extract email from URL params
  const email = searchParams.get('email') || '';
  const token = params.token;

  // Validate token format on mount
  useEffect(() => {
    if (!token || token.length < 10) {
      setPageState('expired');
    }
  }, [token]);

  // Password strength calculation
  const passwordStrength = useMemo(() => {
    const password = formData.password;
    if (!password) return { score: 0, label: '', color: '' };

    const passedRequirements = passwordRequirements.filter(req => req.test(password)).length;
    const score = (passedRequirements / passwordRequirements.length) * 100;

    if (score < 40) return { score, label: 'Weak', color: '#BA1A1A' };
    if (score < 80) return { score, label: 'Fair', color: '#F57C00' };
    return { score, label: 'Strong', color: '#2E7D32' };
  }, [formData.password]);

  // Password match validation
  const passwordsMatch = formData.password && formData.confirmPassword && formData.password === formData.confirmPassword;

  const validateForm = () => {
    const errors: {[key: string]: string} = {};

    if (!formData.password) {
      errors.password = 'Password is required';
    } else if (!passwordRequirements.every(req => req.test(formData.password))) {
      errors.password = 'Password does not meet all requirements';
    }

    if (!formData.confirmPassword) {
      errors.confirmPassword = 'Please confirm your password';
    } else if (formData.password !== formData.confirmPassword) {
      errors.confirmPassword = 'Passwords do not match';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) return;

    clearError();

    try {
      // Use token as OTP - the API expects an otp parameter
      await resetPassword(email, token, formData.password);
      setPageState('success');
    } catch (err) {
      const status = (err as { response?: { status?: number } }).response?.status;
      if (status === 400 || status === 404) {
        setPageState('expired');
      }
    }
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (validationErrors[field]) {
      setValidationErrors(prev => ({ ...prev, [field]: '' }));
    }
    clearError();
  };

  // Success state
  if (pageState === 'success') {
    return (
      <div className="bg-[#F6F6F4] text-[#111413] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0E0E0E] font-[Inter]">
        <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-8 sm:py-12">
          <div className="w-full max-w-md mx-auto bg-white rounded-2xl border border-[#E5E7EB] shadow-xl shadow-black/[0.04] p-8 text-center">
            <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-[#B5F546]/15 flex items-center justify-center">
              <span className="material-symbols-outlined text-[32px] text-[#2E7D32]">check_circle</span>
            </div>
            <h2 className="font-[Plus_Jakarta_Sans] text-2xl text-[#111413] font-bold mb-4">Password Reset Successfully</h2>
            <p className="text-[#737775] mb-8">Your password has been updated. You can now sign in with your new password.</p>
            <Link
              href="/login"
              className="inline-flex items-center justify-center gap-2 h-12 px-6 rounded-full bg-[#0E0E0E] hover:bg-neutral-800 text-white font-semibold text-sm transition-all"
            >
              Continue to Sign In
              <span className="material-symbols-outlined text-[16px]">arrow_forward</span>
            </Link>
          </div>
        </main>
      </div>
    );
  }

  // Expired token state
  if (pageState === 'expired') {
    return (
      <div className="bg-[#F6F6F4] text-[#111413] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0E0E0E] font-[Inter]">
        <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-8 sm:py-12">
          <div className="w-full max-w-md mx-auto bg-white rounded-2xl border border-[#E5E7EB] shadow-xl shadow-black/[0.04] p-8 text-center">
            <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-[#FFDAD6] flex items-center justify-center">
              <span className="material-symbols-outlined text-[32px] text-[#BA1A1A]">schedule</span>
            </div>
            <h2 className="font-[Plus_Jakarta_Sans] text-2xl text-[#111413] font-bold mb-4">Reset Link Expired</h2>
            <p className="text-[#737775] mb-8">This password reset link has expired or is invalid. Please request a new one to reset your password.</p>
            <Link
              href="/forgot-password"
              className="inline-flex items-center justify-center gap-2 h-12 px-6 rounded-full bg-[#0E0E0E] hover:bg-neutral-800 text-white font-semibold text-sm transition-all"
            >
              Request New Link
              <span className="material-symbols-outlined text-[16px]">refresh</span>
            </Link>
          </div>
        </main>
      </div>
    );
  }

  // Main form state
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
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-8 sm:py-12">
        <div className="w-full max-w-md mx-auto bg-white rounded-2xl border border-[#E5E7EB] shadow-xl shadow-black/[0.04] p-8">
          {/* Header */}
          <div className="text-center mb-8">
            <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-[#B5F546]/15 flex items-center justify-center">
              <span className="material-symbols-outlined text-[32px] text-[#0E0E0E]">lock_reset</span>
            </div>
            <h1 className="font-[Plus_Jakarta_Sans] text-2xl text-[#111413] font-bold mb-2">Reset Password</h1>
            <p className="text-sm text-[#737775]">Create a strong new password for your account</p>
          </div>

          {/* Error Banner */}
          {error && (
            <div className="mb-6 p-4 rounded-xl bg-[#FFDAD6] text-[#93000A] border border-[#BA1A1A]/20 flex items-start justify-between gap-3">
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

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* New Password */}
            <div className="space-y-2">
              <label className="block text-xs font-semibold uppercase tracking-wider text-[#737775]" htmlFor="password">
                New Password
              </label>
              <div className="relative">
                <input
                  className="w-full h-12 pl-4 pr-12 rounded-xl bg-[#F4F4F1]/60 border border-[#E5E7EB] text-[#111413] placeholder:text-[#737775]/50 text-sm outline-none focus:bg-white focus:border-[#0E0E0E] focus:ring-2 focus:ring-[#B5F546]/50 transition-all font-medium font-mono"
                  id="password"
                  placeholder="Enter your new password"
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

              {/* Password Strength Meter */}
              {formData.password && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-[#737775]">Password strength</span>
                    <span className="font-semibold" style={{ color: passwordStrength.color }}>
                      {passwordStrength.label}
                    </span>
                  </div>
                  <div className="w-full h-2 bg-[#E5E7EB] rounded-full overflow-hidden">
                    <div
                      className="h-full transition-all duration-300 ease-out"
                      style={{
                        width: `${passwordStrength.score}%`,
                        backgroundColor: passwordStrength.color,
                      }}
                    />
                  </div>
                </div>
              )}

              {validationErrors.password && (
                <p className="text-xs text-[#BA1A1A] flex items-center gap-1">
                  <span className="material-symbols-outlined text-[14px]">error</span>
                  {validationErrors.password}
                </p>
              )}
            </div>

            {/* Password Requirements Checklist */}
            <div className="space-y-2">
              <div className="text-xs font-semibold uppercase tracking-wider text-[#737775]">Requirements</div>
              <div className="space-y-1.5">
                {passwordRequirements.map((req, index) => {
                  const met = req.test(formData.password);
                  return (
                    <div key={index} className="flex items-center gap-2 text-sm">
                      <span className={`material-symbols-outlined text-[16px] ${met ? 'text-[#2E7D32]' : 'text-[#737775]'}`}>
                        {met ? 'check_circle' : 'radio_button_unchecked'}
                      </span>
                      <span className={met ? 'text-[#2E7D32]' : 'text-[#737775]'}>
                        {req.label}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Confirm Password */}
            <div className="space-y-2">
              <label className="block text-xs font-semibold uppercase tracking-wider text-[#737775]" htmlFor="confirmPassword">
                Confirm Password
              </label>
              <div className="relative">
                <input
                  className="w-full h-12 pl-4 pr-12 rounded-xl bg-[#F4F4F1]/60 border border-[#E5E7EB] text-[#111413] placeholder:text-[#737775]/50 text-sm outline-none focus:bg-white focus:border-[#0E0E0E] focus:ring-2 focus:ring-[#B5F546]/50 transition-all font-medium font-mono"
                  id="confirmPassword"
                  placeholder="Confirm your new password"
                  type={showConfirmPassword ? "text" : "password"}
                  value={formData.confirmPassword}
                  onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
                />
                <button
                  aria-label="Toggle confirm password visibility"
                  className="absolute inset-y-0 right-0 px-3.5 flex items-center text-[#737775] hover:text-[#111413] transition-colors"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  type="button"
                >
                  <span className="material-symbols-outlined text-[20px]">
                    {showConfirmPassword ? 'visibility_off' : 'visibility'}
                  </span>
                </button>
              </div>

              {/* Password Match Indicator */}
              {formData.confirmPassword && (
                <div className="flex items-center gap-2 text-sm">
                  <span className={`material-symbols-outlined text-[16px] ${passwordsMatch ? 'text-[#2E7D32]' : 'text-[#BA1A1A]'}`}>
                    {passwordsMatch ? 'check_circle' : 'cancel'}
                  </span>
                  <span className={passwordsMatch ? 'text-[#2E7D32]' : 'text-[#BA1A1A]'}>
                    {passwordsMatch ? 'Passwords match' : 'Passwords do not match'}
                  </span>
                </div>
              )}

              {validationErrors.confirmPassword && (
                <p className="text-xs text-[#BA1A1A] flex items-center gap-1">
                  <span className="material-symbols-outlined text-[14px]">error</span>
                  {validationErrors.confirmPassword}
                </p>
              )}
            </div>

            {/* Submit Button */}
            <button
              className="w-full h-12 rounded-full bg-[#0E0E0E] hover:bg-neutral-800 active:scale-[0.99] text-white font-[Plus_Jakarta_Sans] text-sm font-semibold tracking-wide shadow-md shadow-[#0E0E0E]/20 transition-all flex items-center justify-center gap-2 group disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={isLoading || !passwordsMatch || !passwordRequirements.every(req => req.test(formData.password))}
              type="submit"
            >
              {isLoading ? (
                <div className="flex items-center gap-2">
                  <svg className="animate-spin h-5 w-5 text-[#B5F546]" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" fill="currentColor" />
                  </svg>
                  <span>Updating password...</span>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <span>Reset Password</span>
                  <span className="w-5 h-5 rounded-full bg-[#B5F546] text-[#0E0E0E] flex items-center justify-center group-hover:translate-x-0.5 transition-transform">
                    <span className="material-symbols-outlined text-[14px] font-bold">arrow_forward</span>
                  </span>
                </div>
              )}
            </button>
          </form>

          {/* Back to Login */}
          <div className="mt-6 pt-4 text-center text-sm text-[#737775]">
            Remember your password?
            <Link className="font-semibold text-[#0E0E0E] hover:underline underline-offset-4 ml-1" href="/login">
              Back to Sign In
            </Link>
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

// useSearchParams() needs a Suspense boundary so the page can be prerendered.
export default function ResetPasswordPage({ params }: { params: { token: string } }) {
  return (
    <Suspense fallback={null}>
      <ResetPasswordContent params={params} />
    </Suspense>
  );
}
