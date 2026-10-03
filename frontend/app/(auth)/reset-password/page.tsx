'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';

export default function ResetPasswordPage() {
  const router = useRouter();
  const { resetPassword, isLoading, error, clearError } = useAuthStore();

  const [view, setView] = useState<'form' | 'success' | 'expired'>('form');
  const [tokenValid, setTokenValid] = useState(true);
  const [email, setEmail] = useState('elena.armas@meridianfin.com');

  const [formData, setFormData] = useState({
    newPassword: '',
    confirmPassword: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [validationError, setValidationError] = useState('');

  // Password strength validation
  const getPasswordStrength = (password: string) => {
    const checks = {
      length: password.length >= 8,
      upper: /[A-Z]/.test(password),
      number: /[0-9]/.test(password),
      symbol: /[^A-Za-z0-9]/.test(password),
    };
    const score = Object.values(checks).filter(Boolean).length;
    return { checks, score };
  };

  const passwordsMatch = formData.newPassword && formData.confirmPassword &&
                        formData.newPassword === formData.confirmPassword;

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    clearError();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError('');

    if (!formData.newPassword) {
      setValidationError('Please enter a new password');
      return;
    }

    const { checks } = getPasswordStrength(formData.newPassword);
    if (!checks.length || !checks.upper || !checks.number) {
      setValidationError('Password must contain 8+ chars, uppercase, and number');
      return;
    }

    if (!passwordsMatch) {
      setValidationError('Passwords do not match');
      return;
    }

    try {
      await resetPassword(email, 'tk_sec_9938a9e', formData.newPassword);
      setView('success');
    } catch (err) {
      // Error handled by store
    }
  };

  // Reset token when component mounts (simulated)
  useEffect(() => {
    const checkToken = async () => {
      // In real app, verify token with API
      setTokenValid(true);
    };
    checkToken();
  }, []);

  const { checks, score } = getPasswordStrength(formData.newPassword);

  const renderBar = (index: number) => {
    const colors = ['bg-[#E5E4E0]', 'bg-[#E5E4E0]', 'bg-[#E5E4E0]', 'bg-[#E5E4E0]'];

    if (formData.newPassword.length === 0) {
      return colors[0];
    }

    if (score <= 1) {
      colors[0] = 'bg-[#1C1B1B]';
      return colors[0];
    }

    if (score === 2) {
      colors[0] = colors[1] = 'bg-[#B5F546]';
      return colors[index];
    }

    if (score === 3) {
      colors[0] = colors[1] = colors[2] = 'bg-[#B5F546]';
      return colors[index];
    }

    return 'bg-[#B5F546]';
  };

  const renderRequirement = (id: string, isValid: boolean, label: string) => (
    <li id={id} className={`flex items-center gap-2 font-[Inter] text-xs transition-colors duration-200 ${isValid ? 'text-[#1C1B1B] font-medium' : 'text-[#525252]'}`}>
      <span className={`material-symbols-outlined text-[15px] ${isValid ? 'text-[#1C1B1B]' : 'text-[#898B88]'}`} style={{fontVariationSettings: isValid ? "'FILL' 1" : "'FILL' 0"}}>
        {isValid ? 'check_circle' : 'radio_button_unchecked'}
      </span>
      {label}
    </li>
  );

  return (
    <div className="bg-[#FAF9F6] text-[#141514] font-[Inter] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0E0E0E]">
      {/* Header */}
      <header className="w-full bg-white/80 backdrop-blur-md border-b border-[#E5E4E0]">
        <div className="h-16 max-w-7xl mx-auto px-6 lg:px-12 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7">
              <div className="w-full h-full bg-gradient-to-br from-[#B5F546] to-[#AFD519] rounded-lg flex items-center justify-center text-[#0E0E0E] font-bold text-xs">
                S
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 py-1 px-3 rounded-full bg-[#F6F6F4] border border-[#E5E4E0] text-[#525252] text-xs font-medium">
            <span className="material-symbols-outlined text-[15px] text-[#0E0E0E]">lock</span>
            <span>256-Bit SSL Encrypted</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-[1.75rem] bg-[#FAF9F6]">
        {/* Ambient Glow */}
        <div className="absolute -top-12 w-80 h-80 bg-[#B5F546]/15 rounded-full blur-3xl pointer-events-none"></div>

        {/* State Switcher */}
        <aside aria-label="Demo State Controls" className="mb-6 bg-[#F6F6F4] border border-[#E5E4E0] p-1 rounded-full shadow-sm flex items-center gap-1 z-10">
          <button
            className={`px-3.5 py-1.5 rounded-full text-xs font-semibold transition-all duration-200 ${view === 'form' ? 'bg-white text-[#0E0E0E] border border-[#E5E4E0] shadow-sm' : 'text-[#525252]'}`}
            onClick={() => setView('form')}
            type="button"
          >
            Reset Form
          </button>
          <button
            className={`px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-200 ${view === 'success' ? 'bg-white text-[#0E0E0E] border border-[#E5E4E0] shadow-sm' : 'text-[#525252]'}`}
            onClick={() => setView('success')}
            type="button"
          >
            Success State
          </button>
          <button
            className={`px-3.5 py-1.5 rounded-full text-xs font-medium transition-all duration-200 ${view === 'expired' ? 'bg-white text-[#0E0E0E] border border-[#E5E4E0] shadow-sm' : 'text-[#525252]'}`}
            onClick={() => setView('expired')}
            type="button"
          >
            Expired Token
          </button>
        </aside>

        <div className="w-full max-w-[440px] relative z-10">
          {view === 'form' && (
            <div className="bg-white rounded-2xl border border-[#E5E4E0] shadow-[0_4px_24px_rgba(0,0,0,0.04)] p-6 sm:p-8 flex flex-col gap-5 transition-all duration-300">
              {/* Card Header */}
              <div className="flex flex-col items-center text-center gap-2">
                <div className="w-12 h-12 rounded-2xl bg-[#F6F6F4] border border-[#E5E4E0] flex items-center justify-center text-[#0E0E0E] shadow-sm mb-1">
                  <span className="material-symbols-outlined text-[24px]">lock_reset</span>
                </div>
                <div className="inline-flex items-center gap-1.5 bg-[#B5F546] text-[#0E0E0E] text-[11px] font-bold uppercase tracking-wider px-2.5 py-0.5 rounded-full">
                  <span className="material-symbols-outlined text-[13px]">shield</span>
                  End-to-End Encrypted
                </div>
                <h1 className="font-[Plus_Jakarta_Sans] text-[26px] text-[#0E0E0E] font-bold tracking-tight mt-1">Reset Your Password</h1>
                <p className="font-[Inter] text-[15px] text-[#525252] max-w-xs leading-relaxed">
                  Enter your new password below for your Settle corporate account.
                </p>
              </div>

              {/* Token Context */}
              <div className="bg-[#F6F6F4] border border-[#E5E4E0] p-3 rounded-xl flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-[#0E0E0E] text-[#B5F546] flex items-center justify-center font-semibold text-xs shrink-0">
                  EA
                </div>
                <div className="flex flex-col flex-1 min-w-0">
                  <span className="font-[Inter] text-xs font-medium text-[#0E0E0E] truncate">{email}</span>
                  <span className="font-[Inter] text-[11px] text-[#525252] font-mono">Token auth: tk_sec_9938a9e</span>
                </div>
                <span className="px-2 py-0.5 rounded-full bg-[#B5F546] text-[#0E0E0E] text-[10px] font-bold uppercase tracking-wide shrink-0">Active</span>
              </div>

              {/* Password Form */}
              <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
                {/* New Password */}
                <div className="flex flex-col gap-1.5">
                  <label className="font-[Inter] text-xs font-semibold text-[#0E0E0E] uppercase tracking-wide" htmlFor="new-password">New Password</label>
                  <div className="relative flex items-center">
                    <input
                      className="w-full h-11 px-3.5 pr-10 rounded-xl bg-[#F6F6F4] border border-[#E5E4E0] text-[#0E0E0E] text-sm placeholder:text-[#898B88]/70 focus:bg-white focus:outline-none focus:ring-2 focus:ring-[#B5F546] focus:border-[#0E0E0E] transition-all shadow-none"
                      id="new-password"
                      placeholder="••••••••"
                      required
                      type={showPassword ? "text" : "password"}
                      value={formData.newPassword}
                      onChange={(e) => handleInputChange('newPassword', e.target.value)}
                    />
                    <button
                      aria-label="Toggle password visibility"
                      className="absolute right-3 text-[#525252] hover:text-[#0E0E0E] flex items-center justify-center focus:outline-none"
                      onClick={() => setShowPassword(!showPassword)}
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[19px]">visibility</span>
                    </button>
                  </div>
                </div>

                {/* Strength Gauge */}
                <div className="flex flex-col gap-1.5 pt-0.5">
                  <div className="flex justify-between items-center text-xs">
                    <span className="text-[#525252]">Complexity Level</span>
                    <span className="font-semibold text-[#898B88] text-[11px]">
                      {formData.newPassword.length === 0 ? 'Enter password' :
                       score <= 1 ? 'Weak' :
                       score === 2 ? 'Fair' :
                       score === 3 ? 'Strong' : 'Institutional Grade'}
                    </span>
                  </div>
                  <div className="grid grid-cols-4 gap-1.5 w-full h-1.5">
                    {[0, 1, 2, 3].map(i => (
                      <div key={i} className={`h-full rounded-full transition-colors duration-300 ${renderBar(i)}`}></div>
                    ))}
                  </div>
                </div>

                {/* Confirm Password */}
                <div className="flex flex-col gap-1.5 pt-0.5">
                  <div className="flex justify-between items-center">
                    <label className="font-[Inter] text-xs font-semibold text-[#0E0E0E] uppercase tracking-wide" htmlFor="confirm-password">Confirm Password</label>
                    {passwordsMatch && (
                      <span className="hidden items-center gap-1 font-[Inter] text-[11px] font-semibold text-[#0E0E0E]">
                        <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]"></span> Passwords match
                      </span>
                    )}
                  </div>
                  <div className="relative flex items-center">
                    <input
                      className="w-full h-11 px-3.5 pr-10 rounded-xl bg-[#F6F6F4] border border-[#E5E4E0] text-[#0E0E0E] text-sm placeholder:text-[#898B88]/70 focus:bg-white focus:outline-none focus:ring-2 focus:ring-[#B5F546] focus:border-[#0E0E0E] transition-all shadow-none"
                      id="confirm-password"
                      placeholder="••••••••"
                      required
                      type={showConfirmPassword ? "text" : "password"}
                      value={formData.confirmPassword}
                      onChange={(e) => handleInputChange('confirmPassword', e.target.value)}
                    />
                    <button
                      aria-label="Toggle confirm password visibility"
                      className="absolute right-3 text-[#525252] hover:text-[#0E0E0E] flex items-center justify-center focus:outline-none"
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[19px]">visibility</span>
                    </button>
                  </div>
                </div>

                {/* Requirements Checklist */}
                <div className="bg-[#F6F6F4] border border-[#E5E4E0] p-3.5 rounded-xl flex flex-col gap-2 mt-0.5">
                  <p className="font-[Inter] text-xs font-semibold text-[#0E0E0E]">Password Requirements</p>
                  <ul className="flex flex-col gap-1.5">
                    {renderRequirement('req-len', checks.length, 'At least 8 characters')}
                    {renderRequirement('req-upper', checks.upper, 'Contains uppercase letter')}
                    {renderRequirement('req-num', checks.number, 'Contains number')}
                    {renderRequirement('req-sym', checks.symbol, 'Contains special symbol (!@#$%^&*)')}
                  </ul>
                </div>

                {/* Submit Button */}
                <button
                  className="w-full h-12 mt-1 rounded-full bg-[#0E0E0E] text-white text-sm font-semibold flex items-center justify-center gap-2 hover:bg-[#222222] transition-all duration-200 shadow-md active:scale-[0.99] disabled:opacity-50 disabled:pointer-events-none"
                  disabled={isLoading || formData.newPassword.length === 0 || !passwordsMatch}
                  type="submit"
                >
                  {isLoading ? (
                    <div className="flex items-center gap-2">
                      <span className="material-symbols-outlined text-[18px] animate-spin text-[#B5F546]">progress_activity</span>
                      <span>Securing Credentials...</span>
                    </div>
                  ) : (
                    <>
                      <span>Reset Password</span>
                      <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
                    </>
                  )}
                </button>
              </form>

              {/* Cancel Link */}
              <div className="flex items-center justify-center pt-1">
                <Link
                  className="font-[Inter] text-xs font-medium text-[#525252] hover:text-[#0E0E0E] transition-colors flex items-center gap-1.5"
                  href="/login"
                >
                  <span className="material-symbols-outlined text-[15px]">arrow_back</span>
                  Cancel and return to sign in
                </Link>
              </div>
            </div>
          )}

          {view === 'success' && (
            <div className="bg-white rounded-2xl border border-[#E5E4E0] shadow-[0_4px_24px_rgba(0,0,0,0.04)] p-6 sm:p-8 flex-col items-center text-center gap-5 transition-all duration-300">
              <div className="relative flex items-center justify-center my-2">
                <div className="w-16 h-16 rounded-full bg-[#B5F546]/20 border border-[#B5F546] flex items-center justify-center text-[#0E0E0E]">
                  <span className="material-symbols-outlined text-[36px]" style={{fontVariationSettings: "'FILL' 1"}}>check_circle</span>
                </div>
                <div className="absolute -bottom-1 -right-1 w-6 h-6 rounded-full bg-[#0E0E0E] flex items-center justify-center text-[#B5F546] shadow-sm">
                  <span className="material-symbols-outlined text-[14px]">lock_reset</span>
                </div>
              </div>

              <div className="flex flex-col gap-1 max-w-sm">
                <span className="inline-block mx-auto px-2.5 py-0.5 rounded-full bg-[#B5F546] text-[#0E0E0E] text-[11px] font-bold uppercase tracking-wider mb-1">Credential Security Updated</span>
                <h2 className="font-[Plus_Jakarta_Sans] text-[26px] text-[#0E0E0E] font-bold tracking-tight">Password Reset Successfully</h2>
                <p className="font-[Inter] text-[15px] text-[#525252] leading-relaxed">
                  Your password has been changed. All previous sessions have been invalidated for security. You can now sign in with your new credentials.
                </p>
              </div>

              {/* Device Session Confirmation */}
              <div className="w-full bg-[#F6F6F4] border border-[#E5E4E0] p-3 rounded-xl flex items-center justify-between text-left">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-lg bg-white border border-[#E5E4E0] flex items-center justify-center text-[#0E0E0E]">
                    <span className="material-symbols-outlined text-[20px]">laptop_mac</span>
                  </div>
                  <div>
                    <div className="font-[Inter] text-xs font-semibold text-[#0E0E0E]">Active Audit Record Logged</div>
                    <div className="font-[Inter] text-[11px] text-[#525252] font-mono">Timestamp: Just now • IP: 198.51.100.24</div>
                  </div>
                </div>
                <span className="w-2.5 h-2.5 rounded-full bg-[#B5F546]"></span>
              </div>

              <button
                className="w-full h-12 rounded-full bg-[#0E0E0E] text-white text-sm font-semibold flex items-center justify-center gap-2 hover:bg-[#222222] transition-all duration-200 shadow-md active:scale-[0.99]"
                onClick={() => setView('form')}
                type="button"
              >
                <span>Sign In</span>
                <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
              </button>

              <div className="flex items-center gap-1.5 font-[Inter] text-xs text-[#525252]">
                <span className="material-symbols-outlined text-[15px] text-[#0E0E0E]">verified_user</span>
                Settle 2FA &amp; Hardware Key protection enabled
              </div>
            </div>
          )}

          {view === 'expired' && (
            <div className="bg-white rounded-2xl border border-[#E5E4E0] shadow-[0_4px_24px_rgba(0,0,0,0.04)] p-6 sm:p-8 flex-col items-center text-center gap-5 transition-all duration-300">
              <div className="w-14 h-14 rounded-2xl bg-[#F6F6F4] border border-[#E5E4E0] flex items-center justify-center text-[#0E0E0E] shadow-sm my-1">
                <span className="material-symbols-outlined text-[28px]">timer_off</span>
              </div>

              <div className="flex flex-col gap-1 max-w-sm">
                <span className="inline-block mx-auto px-2.5 py-0.5 rounded-full bg-[#F6F6F4] border border-[#E5E4E0] text-[#0E0E0E] text-[11px] font-bold uppercase tracking-wider mb-1">Link Expired</span>
                <h2 className="font-[Plus_Jakarta_Sans] text-[26px] text-[#0E0E0E] font-bold tracking-tight">Reset Link No Longer Valid</h2>
                <p className="font-[Inter] text-[15px] text-[#525252] leading-relaxed">
                  This reset link has expired or has already been used. For your security, password reset tokens are valid for 15 minutes only.
                </p>
              </div>

              {/* Security Guidance */}
              <div className="w-full bg-[#F6F6F4] border border-[#E5E4E0] p-3.5 rounded-xl flex items-start gap-3 text-left">
                <span className="material-symbols-outlined text-[#0E0E0E] text-[18px] mt-0.5">info</span>
                <p className="font-[Inter] text-xs text-[#525252] leading-relaxed">
                  If you did not request this reset or believe this is an error, please reach out directly to your organization's Settle administrator.
                </p>
              </div>

              {/* Action Stack */}
              <div className="w-full flex flex-col gap-2.5">
                <button
                  className="w-full h-12 rounded-full bg-[#0E0E0E] text-white text-sm font-semibold flex items-center justify-center gap-2 hover:bg-[#222222] transition-all duration-200 shadow-md"
                  onClick={() => setView('form')}
                  type="button"
                >
                  <span className="material-symbols-outlined text-[17px]">mail</span>
                  <span>Request New Reset Link</span>
                </button>
                <Link
                  className="w-full h-11 rounded-full bg-[#F6F6F4] border border-[#E5E4E0] text-[#0E0E0E] font-[Inter] text-xs font-semibold flex items-center justify-center gap-1.5 hover:bg-[#E5E4E0]/50 transition-colors"
                  href="/login"
                >
                  Return to Sign In
                </Link>
              </div>

              <div className="flex items-center gap-1.5 font-[Inter] text-xs text-[#525252]">
                <span>Need emergency access?</span>
                <Link className="text-[#0E0E0E] font-semibold hover:underline" href="#">Contact SecOps</Link>
              </div>
            </div>
          )}
        </div>

        {/* Regulatory Badge */}
        <div className="mt-6 flex items-center gap-3 font-[Inter] text-xs text-[#525252]">
          <div className="flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]"></span>
            <span className="font-medium text-[#0E0E0E]">FIPS 140-2 Validated</span>
          </div>
          <span className="text-[#E5E4E0]">•</span>
          <div className="flex items-center gap-1.5">
            <span className="material-symbols-outlined text-[14px] text-[#0E0E0E]">key</span>
            <span className="font-medium text-[#0E0E0E]">PBKDF2 Hashed</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-white/80 backdrop-blur-md border-t border-[#E5E4E0] py-5">
        <div className="max-w-7xl mx-auto px-6 lg:px-12 flex flex-col sm:flex-row items-center justify-between gap-3 text-[#525252]">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]"></span>
            <span>© 2025 Settle Technologies Inc. SOC2 Type II Certified.</span>
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
