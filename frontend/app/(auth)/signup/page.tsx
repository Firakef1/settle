'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';

export default function SignupPage() {
  const router = useRouter();
  const { signup, isLoading, error, clearError } = useAuthStore();

  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    confirmPassword: '',
  });
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [validationErrors, setValidationErrors] = useState<{[key: string]: string}>({});

  // Password strength validation
  const getPasswordStrength = (password: string) => {
    const checks = {
      length: password.length >= 8,
      upper: /[A-Z]/.test(password),
      number: /[0-9]/.test(password),
    };
    const score = Object.values(checks).filter(Boolean).length;
    return { checks, score };
  };

  const passwordsMatch = formData.password && formData.confirmPassword &&
                        formData.password === formData.confirmPassword;

  const validateForm = () => {
    const errors: {[key: string]: string} = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (!formData.email) {
      errors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      errors.email = 'Please enter a valid email address';
    }

    if (!formData.password) {
      errors.password = 'Password is required';
    } else {
      const { checks } = getPasswordStrength(formData.password);
      if (!checks.length || !checks.upper || !checks.number) {
        errors.password = 'Password must meet all requirements';
      }
    }

    if (!formData.confirmPassword) {
      errors.confirmPassword = 'Please confirm your password';
    } else if (formData.password !== formData.confirmPassword) {
      errors.confirmPassword = 'Passwords do not match';
    }

    if (!acceptedTerms) {
      errors.terms = 'You must agree to the Terms of Service and Privacy Policy';
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
      await signup(formData.name, formData.email, formData.password);

      // Redirect to email verification
      router.push(`/verify-email?email=${encodeURIComponent(formData.email)}`);
    } catch {
      // Error is handled by the store
    }
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (validationErrors[field]) {
      setValidationErrors(prev => ({ ...prev, [field]: '' }));
    }
    clearError();
  };

  const { checks } = getPasswordStrength(formData.password);

  return (
    <div className="bg-[#faf9f6] text-[#121514] font-[Inter] antialiased min-h-screen flex flex-col justify-between selection:bg-[#caf23c] selection:text-[#0e0e0e]">
      {/* Header */}
      <header className="w-full bg-white/80 backdrop-blur-md border-b border-[#e2e2df]">
        <div className="h-16 max-w-7xl mx-auto px-6 lg:px-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-[#0e0e0e] flex items-center justify-center p-1.5 shadow-sm border border-neutral-800">
                <div className="w-full h-full grid grid-cols-2 gap-0.5">
                  <div className="bg-[#caf23c] rounded-xs"></div>
                  <div className="bg-neutral-400 rounded-xs"></div>
                  <div className="bg-neutral-600 rounded-xs"></div>
                  <div className="bg-[#caf23c] rounded-xs"></div>
                </div>
              </div>
              <span className="font-[Plus_Jakarta_Sans] text-xl text-[#0e0e0e] font-bold tracking-tight">Settle</span>
            </div>
          </div>
          <div className="flex items-center gap-2 py-1.5 px-3.5 rounded-full bg-[#efeeeb] border border-[#e2e2df]/80 text-[#5f6368] text-xs font-medium">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]" style={{fontVariationSettings: "'FILL' 1"}}>verified_user</span>
            <span>256-Bit SSL Encrypted</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-8 lg:py-12 bg-[#faf9f6]">
        <div className="w-full max-w-7xl mx-auto">
          <div className="grid grid-cols-1 lg:grid-cols-12 rounded-2xl overflow-hidden border border-[#e2e2df]/70 shadow-xl bg-white">
            {/* Left Marketing Panel */}
            <section className="lg:col-span-7 bg-[#0e0e0e] text-white p-8 sm:p-12 lg:p-14 flex flex-col justify-between relative overflow-hidden">
              {/* Background Effects */}
              <div className="absolute -top-32 -left-32 w-80 h-80 rounded-full bg-[#caf23c] opacity-10 blur-3xl pointer-events-none"></div>
              <div className="absolute -bottom-24 -right-24 w-96 h-96 rounded-full bg-[#1e2808] opacity-50 blur-3xl pointer-events-none"></div>
              <div className="absolute top-1/2 left-1/3 w-64 h-64 rounded-full bg-[#caf23c] opacity-[0.04] blur-2xl pointer-events-none"></div>
              <div className="absolute inset-0 bg-[radial-gradient(#262a26_1px,transparent_1px)] [background-size:24px_24px] opacity-40 pointer-events-none"></div>

              <div className="relative z-10 flex flex-col gap-6">
                {/* Header Badge */}
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="inline-flex items-center gap-2 bg-[#1c201d] border border-[#2b3524] px-3.5 py-1.5 rounded-full">
                    <span className="w-2 h-2 rounded-full bg-[#caf23c] animate-pulse"></span>
                    <span className="font-[Inter] text-xs font-semibold text-[#caf23c] tracking-wide uppercase">Enterprise Tier Security</span>
                  </div>
                  <span className="font-[Inter] text-xs text-neutral-400 font-mono tracking-wider">SOC2 TYPE II • ISO 27001</span>
                </div>

                <div className="space-y-3 mt-4">
                  <h1 className="font-[Plus_Jakarta_Sans] text-3xl sm:text-4xl lg:text-[40px] leading-[1.15] text-white font-bold tracking-tight max-w-xl">
                    Precision spend management built for global scale.
                  </h1>
                  <p className="font-[Inter] text-neutral-400 max-w-lg text-[15px] leading-relaxed">
                    Empower controllers and finance executives with autonomous reconciliations, programmable virtual cards, and sub-second compliance routing.
                  </p>
                </div>

                {/* Metric Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-4">
                  <div className="bg-[#161817] border border-[#262827] hover:border-[#caf23c]/40 transition-colors p-5 rounded-xl">
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-[Plus_Jakarta_Sans] text-3xl font-extrabold text-white tracking-tight">$140M+</span>
                      <span className="text-[11px] font-semibold uppercase tracking-wider text-[#caf23c] bg-[#1a2310] border border-[#2c3f15] px-2 py-0.5 rounded">Volume</span>
                    </div>
                    <p className="font-[Inter] text-xs text-neutral-400 mt-2 leading-relaxed">Managed & settled monthly across 42 jurisdictions</p>
                  </div>

                  <div className="bg-[#161817] border border-[#262827] hover:border-[#caf23c]/40 transition-colors p-5 rounded-xl">
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-[Plus_Jakarta_Sans] text-3xl font-extrabold text-[#caf23c] tracking-tight">99.98%</span>
                      <span className="text-[11px] font-semibold uppercase tracking-wider text-[#caf23c] bg-[#1a2310] border border-[#2c3f15] px-2 py-0.5 rounded">Autonomous</span>
                    </div>
                    <p className="font-[Inter] text-xs text-neutral-400 mt-2 leading-relaxed">Automated approval routes with zero-touch reconciliation</p>
                  </div>
                </div>
              </div>

              {/* Social Proof */}
              <div className="relative z-10 mt-10 pt-4">
                <div className="bg-[#151716]/90 border border-neutral-800 backdrop-blur-md p-5 rounded-xl text-neutral-200 flex flex-col gap-3 shadow-2xl">
                  <div className="flex items-center gap-1 text-[#caf23c]">
                    {Array.from({length: 5}).map((_, i) => (
                      <span key={i} className="material-symbols-outlined text-[17px]" style={{fontVariationSettings: "'FILL' 1"}}>star</span>
                    ))}
                  </div>
                  <blockquote className="font-[Inter] text-sm text-neutral-300 italic leading-relaxed">
                    &ldquo;Settle unified our cross-border liquidity and multi-subsidiary auditing in under two weeks. Month-end close went from 9 days down to 4 hours.&rdquo;
                  </blockquote>
                  <div className="flex items-center justify-between pt-2 border-t border-neutral-800/80 mt-1">
                    <div className="flex items-center gap-3">
                      <div className="w-9 h-9 rounded-full overflow-hidden bg-neutral-800 flex-shrink-0 ring-1 ring-[#caf23c]/40">
                        <div className="w-full h-full bg-gradient-to-br from-[#caf23c] to-[#afd519] flex items-center justify-center text-[#0e0e0e] font-bold text-sm">ER</div>
                      </div>
                      <div>
                        <div className="font-[Inter] text-xs font-semibold text-white">Elena Rostova</div>
                        <div className="font-[Inter] text-[11px] text-neutral-400">VP of Finance, Northwind Capital</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1.5 text-neutral-400 text-xs font-mono">
                      <span className="material-symbols-outlined text-[16px] text-[#caf23c]">bolt</span>
                      <span>Real-time sync</span>
                    </div>
                  </div>
                </div>

                {/* Footer assurance */}
                <div className="flex flex-wrap items-center justify-between text-neutral-400 font-[Inter] text-xs mt-4 px-1 gap-2">
                  <span>Trusted by controllers at Stripe, Brex, and 2,400+ orgs</span>
                  <span className="flex items-center gap-1.5 font-medium text-neutral-300">
                    <span className="w-2 h-2 rounded-full bg-[#caf23c] inline-block"></span>
                    Systems Operational
                  </span>
                </div>
              </div>
            </section>

            {/* Right Signup Form Panel */}
            <section className="lg:col-span-5 p-8 sm:p-12 flex flex-col justify-center bg-white">
              <div className="w-full max-w-md mx-auto space-y-6">
                {/* Header */}
                <div className="space-y-1.5">
                  <h2 className="font-[Plus_Jakarta_Sans] text-2xl sm:text-3xl text-[#0e0e0e] tracking-tight font-bold">
                    Create Your Account
                  </h2>
                  <p className="font-[Inter] text-sm text-[#5f6368]">
                    Join thousands of teams using Settle to streamline corporate finance.
                  </p>
                </div>

                {/* Error Banner */}
                {error && (
                  <div className="p-4 rounded-xl bg-[#ffdad6] text-[#93000a] border border-[#ba1a1a]/20 flex items-start justify-between gap-3 transition-all duration-300">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[#ba1a1a] text-[20px] shrink-0">error</span>
                      <span className="text-sm font-medium">{error}</span>
                    </div>
                    <button
                      aria-label="Dismiss error"
                      className="text-[#93000a]/70 hover:text-[#93000a] p-0.5 rounded transition-colors shrink-0"
                      onClick={clearError}
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[18px]">close</span>
                    </button>
                  </div>
                )}

                {/* Form */}
                <form className="space-y-4" onSubmit={handleSubmit}>
                  {/* Full Name */}
                  <div className="space-y-1.5">
                    <label className="font-[Inter] text-xs font-semibold text-[#0e0e0e] block uppercase tracking-wider" htmlFor="fullName">
                      Full Name
                    </label>
                    <div className="relative flex items-center">
                      <span className="material-symbols-outlined absolute left-3.5 text-neutral-400 text-[18px] pointer-events-none">person</span>
                      <input
                        className="w-full h-11 pl-10 pr-4 rounded-xl bg-[#f4f3f0] border border-[#e2e2df] text-[#0e0e0e] font-[Inter] text-sm placeholder:text-neutral-400 focus:outline-none focus:border-[#0e0e0e] focus:bg-white transition-all shadow-xs"
                        id="fullName"
                        name="fullName"
                        placeholder="John Doe"
                        required
                        type="text"
                        value={formData.name}
                        onChange={(e) => handleInputChange('name', e.target.value)}
                      />
                    </div>
                    {validationErrors.name && (
                      <p className="text-xs text-[#ba1a1a] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.name}
                      </p>
                    )}
                  </div>

                  {/* Email */}
                  <div className="space-y-1.5">
                    <label className="font-[Inter] text-xs font-semibold text-[#0e0e0e] block uppercase tracking-wider" htmlFor="email">
                      Email Address
                    </label>
                    <div className="relative flex items-center">
                      <span className="material-symbols-outlined absolute left-3.5 text-neutral-400 text-[18px] pointer-events-none">mail</span>
                      <input
                        className="w-full h-11 pl-10 pr-4 rounded-xl bg-[#f4f3f0] border border-[#e2e2df] text-[#0e0e0e] font-[Inter] text-sm placeholder:text-neutral-400 focus:outline-none focus:border-[#0e0e0e] focus:bg-white transition-all shadow-xs"
                        id="email"
                        name="email"
                        placeholder="you@company.com"
                        required
                        type="email"
                        value={formData.email}
                        onChange={(e) => handleInputChange('email', e.target.value)}
                      />
                    </div>
                    {validationErrors.email && (
                      <p className="text-xs text-[#ba1a1a] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.email}
                      </p>
                    )}
                  </div>

                  {/* Password */}
                  <div className="space-y-1.5">
                    <label className="font-[Inter] text-xs font-semibold text-[#0e0e0e] block uppercase tracking-wider" htmlFor="password">
                      Password
                    </label>
                    <div className="relative flex items-center">
                      <span className="material-symbols-outlined absolute left-3.5 text-neutral-400 text-[18px] pointer-events-none">lock</span>
                      <input
                        className="w-full h-11 pl-10 pr-10 rounded-xl bg-[#f4f3f0] border border-[#e2e2df] text-[#0e0e0e] font-[Inter] text-sm placeholder:text-neutral-400 focus:outline-none focus:border-[#0e0e0e] focus:bg-white transition-all shadow-xs"
                        id="password"
                        name="password"
                        placeholder="••••••••"
                        required
                        type={showPassword ? "text" : "password"}
                        value={formData.password}
                        onChange={(e) => handleInputChange('password', e.target.value)}
                      />
                      <button
                        aria-label="Toggle password visibility"
                        className="absolute right-3 text-neutral-400 hover:text-[#0e0e0e] focus:outline-none transition-colors"
                        onClick={() => setShowPassword(!showPassword)}
                        type="button"
                      >
                        <span className="material-symbols-outlined text-[18px]">
                          {showPassword ? 'visibility_off' : 'visibility'}
                        </span>
                      </button>
                    </div>

                    {/* Password Requirements */}
                    <div className="pt-2 pb-1 space-y-1.5 bg-[#f4f3f0]/80 border border-[#e2e2df]/60 p-3 rounded-xl mt-2">
                      <div className={`flex items-center gap-2 font-[Inter] text-xs transition-colors ${checks.length ? 'text-[#0e0e0e] font-medium' : 'text-[#5f6368]'}`}>
                        <span className={`material-symbols-outlined text-[16px] ${checks.length ? 'text-[#526600]' : 'text-neutral-400'}`} style={{fontVariationSettings: checks.length ? "'FILL' 1" : "'FILL' 0"}}>
                          check_circle
                        </span>
                        <span>At least 8 characters</span>
                      </div>
                      <div className={`flex items-center gap-2 font-[Inter] text-xs transition-colors ${checks.upper ? 'text-[#0e0e0e] font-medium' : 'text-[#5f6368]'}`}>
                        <span className={`material-symbols-outlined text-[16px] ${checks.upper ? 'text-[#526600]' : 'text-neutral-400'}`} style={{fontVariationSettings: checks.upper ? "'FILL' 1" : "'FILL' 0"}}>
                          check_circle
                        </span>
                        <span>Contains uppercase letter</span>
                      </div>
                      <div className={`flex items-center gap-2 font-[Inter] text-xs transition-colors ${checks.number ? 'text-[#0e0e0e] font-medium' : 'text-[#5f6368]'}`}>
                        <span className={`material-symbols-outlined text-[16px] ${checks.number ? 'text-[#526600]' : 'text-neutral-400'}`} style={{fontVariationSettings: checks.number ? "'FILL' 1" : "'FILL' 0"}}>
                          check_circle
                        </span>
                        <span>Contains number</span>
                      </div>
                    </div>
                    {validationErrors.password && (
                      <p className="text-xs text-[#ba1a1a] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.password}
                      </p>
                    )}
                  </div>

                  {/* Confirm Password */}
                  <div className="space-y-1.5">
                    <label className="font-[Inter] text-xs font-semibold text-[#0e0e0e] block uppercase tracking-wider" htmlFor="confirmPassword">
                      Confirm Password
                    </label>
                    <div className="relative flex items-center">
                      <span className="material-symbols-outlined absolute left-3.5 text-neutral-400 text-[18px] pointer-events-none">lock_reset</span>
                      <input
                        className="w-full h-11 pl-10 pr-10 rounded-xl bg-[#f4f3f0] border border-[#e2e2df] text-[#0e0e0e] font-[Inter] text-sm placeholder:text-neutral-400 focus:outline-none focus:border-[#0e0e0e] focus:bg-white transition-all shadow-xs"
                        id="confirmPassword"
                        name="confirmPassword"
                        placeholder="••••••••"
                        required
                        type={showConfirmPassword ? "text" : "password"}
                        value={formData.confirmPassword}
                        onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
                      />
                      <div className="absolute right-3 flex items-center gap-1.5">
                        {passwordsMatch && (
                          <span className="material-symbols-outlined text-[18px] text-[#0e0e0e]" style={{fontVariationSettings: "'FILL' 1"}}>
                            check_circle
                          </span>
                        )}
                        <button
                          aria-label="Toggle confirm password visibility"
                          className="text-neutral-400 hover:text-[#0e0e0e] focus:outline-none transition-colors"
                          onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                          type="button"
                        >
                          <span className="material-symbols-outlined text-[18px]">
                            {showConfirmPassword ? 'visibility_off' : 'visibility'}
                          </span>
                        </button>
                      </div>
                    </div>
                    {passwordsMatch && (
                      <p className="font-[Inter] text-xs text-neutral-600 font-medium flex items-center gap-1">
                        <span className="w-1.5 h-1.5 rounded-full bg-[#0e0e0e] inline-block"></span>
                        Passwords match
                      </p>
                    )}
                    {validationErrors.confirmPassword && (
                      <p className="text-xs text-[#ba1a1a] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.confirmPassword}
                      </p>
                    )}
                  </div>

                  {/* Terms Checkbox */}
                  <div className="pt-1">
                    <label className="flex items-start gap-2.5 cursor-pointer select-none">
                      <input
                        className="mt-0.5 w-4 h-4 rounded text-[#0e0e0e] focus:ring-0 cursor-pointer accent-[#0e0e0e]"
                        id="terms"
                        required
                        type="checkbox"
                        checked={acceptedTerms}
                        onChange={(e) => setAcceptedTerms(e.target.checked)}
                      />
                      <span className="font-[Inter] text-xs text-[#5f6368] leading-relaxed">
                        I agree to Settle&apos;s{' '}
                        <Link className="font-medium text-[#0e0e0e] hover:underline underline-offset-2" href="#">
                          Terms of Service
                        </Link>
                        {' '}and{' '}
                        <Link className="font-medium text-[#0e0e0e] hover:underline underline-offset-2" href="#">
                          Privacy Policy
                        </Link>
                        .
                      </span>
                    </label>
                    {validationErrors.terms && (
                      <p className="text-xs text-[#ba1a1a] flex items-center gap-1 mt-1">
                        <span className="material-symbols-outlined text-[14px]">error</span>
                        {validationErrors.terms}
                      </p>
                    )}
                  </div>

                  {/* Submit Button */}
                  <div className="pt-2">
                    <button
                      className="w-full h-12 rounded-full bg-[#0e0e0e] hover:bg-neutral-800 active:bg-black text-white font-[Inter] text-sm font-semibold flex items-center justify-center gap-2 shadow-md hover:shadow-lg transition-all cursor-pointer group disabled:opacity-50 disabled:cursor-not-allowed"
                      disabled={isLoading}
                      type="submit"
                    >
                      {isLoading ? (
                        <div className="flex items-center gap-2">
                          <span className="material-symbols-outlined text-[18px] animate-spin text-[#caf23c]">
                            progress_activity
                          </span>
                          <span>Provisioning Workspace...</span>
                        </div>
                      ) : (
                        <>
                          <span>Create Account</span>
                          <span className="material-symbols-outlined text-[18px] text-[#caf23c] group-hover:translate-x-0.5 transition-transform">
                            arrow_forward
                          </span>
                        </>
                      )}
                    </button>
                  </div>
                </form>

                {/* Sign In Link */}
                <div className="text-center pt-1">
                  <p className="font-[Inter] text-xs text-[#5f6368]">
                    Already have an account?{' '}
                    <Link className="font-semibold text-[#0e0e0e] hover:underline underline-offset-2 ml-1" href="/login">
                      Sign in
                    </Link>
                  </p>
                </div>

                {/* Bottom Assurance */}
                <div className="pt-2 flex items-center justify-center gap-1.5 text-neutral-400 text-xs">
                  <span className="material-symbols-outlined text-[15px]">shield</span>
                  <span className="font-[Inter]">Enterprise SAML / SSO supported upon team invite</span>
                </div>
              </div>
            </section>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-white border-t border-[#e2e2df]/60 py-6">
        <div className="max-w-7xl mx-auto px-6 lg:px-12 flex flex-col sm:flex-row items-center justify-between gap-4 text-[#5f6368] text-xs">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]">verified_user</span>
            <span>© 2025 Settle Technologies Inc. SOC2 Type II Certified.</span>
          </div>
          <div className="flex items-center gap-6">
            <Link className="hover:text-[#0e0e0e] transition-colors" href="#">Privacy Policy</Link>
            <Link className="hover:text-[#0e0e0e] transition-colors" href="#">Terms of Service</Link>
            <Link className="hover:text-[#0e0e0e] transition-colors" href="#">Security Operations</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
