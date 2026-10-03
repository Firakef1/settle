'use client';

import { useState, useEffect, useRef } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';

export default function VerifyEmailPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { verifyEmail, isLoading, error, clearError } = useAuthStore();

  const email = searchParams?.get('email') || 'sarah.chen@acmelabs.io';

  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const [resendTimer, setResendTimer] = useState(45);
  const [showAlert, setShowAlert] = useState(false);
  const [alertTitle, setAlertTitle] = useState('');
  const [alertDescription, setAlertDescription] = useState('');

  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Auto-focus first input
  useEffect(() => {
    inputRefs.current[0]?.focus();
  }, []);

  // Countdown timer for resend
  useEffect(() => {
    if (resendTimer > 0) {
      const timer = setInterval(() => setResendTimer(prev => prev - 1), 1000);
      return () => clearInterval(timer);
    }
  }, [resendTimer]);

  const handleOtpChange = (index: number, value: string) => {
    if (value && value.length > 1) return;

    const newOtp = [...otp];
    newOtp[index] = value;
    setOtp(newOtp);
    clearError();

    if (value && index < 5) {
      inputRefs.current[index + 1]?.focus();
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !otp[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    }
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData.getData('text').replace(/[^0-9]/g, '').slice(0, 6);

    if (pastedData) {
      const newOtp = [...otp];
      for (let i = 0; i < pastedData.length; i++) {
        newOtp[i] = pastedData[i];
      }
      setOtp(newOtp);

      const nextIndex = Math.min(pastedData.length, 5);
      inputRefs.current[nextIndex]?.focus();
    }
  };

  const handleResend = () => {
    clearError();
    // In a real app, call authAPI.resendVerification(email)
    setResendTimer(45);
    setOtp(['', '', '', '', '', '']);
    inputRefs.current[0]?.focus();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const code = otp.join('');
    if (code.length < 6) {
      setAlertTitle('Incomplete code');
      setAlertDescription('Please fill all 6 verification digits before continuing.');
      setShowAlert(true);
      return;
    }

    clearError();

    try {
      await verifyEmail(email, code);
      router.push('/dashboard');
    } catch (err) {
      // Error handled by store
    }
  };

  const dismissAlert = () => setShowAlert(false);

  return (
    <div className="bg-[#FAF9F6] text-[#0e0e0e] font-[Inter] antialiased min-h-screen flex flex-col justify-between selection:bg-[#B5F546] selection:text-[#0e0e0e]">
      {/* Header */}
      <header className="w-full bg-white/90 backdrop-blur-xl border-b border-[#e5e7eb]">
        <div className="h-16 max-w-7xl mx-auto px-6 lg:px-12 flex items-center justify-between">
          <div className="flex items-center gap-3.5">
            <div className="w-7 h-7">
              <div className="w-full h-full bg-gradient-to-br from-[#B5F546] to-[#AFD519] rounded-lg flex items-center justify-center text-[#0e0e0e] font-bold text-xs">
                S
              </div>
            </div>
            <span className="font-[Plus_Jakarta_Sans] text-xl text-[#0e0e0e] font-bold tracking-tight">Settle</span>
          </div>
          <div className="flex items-center gap-2 py-1.5 px-3.5 rounded-full bg-[#F0EFEA] border border-[#e5e7eb] text-[#525252] text-xs font-medium">
            <span className="material-symbols-outlined text-[15px] text-[#0e0e0e]">lock</span>
            <span>256-Bit SSL Encrypted</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="w-full flex-1 flex flex-col items-center justify-center px-4 py-[1.75rem] bg-[#FAF9F6]">
        <div className="w-full max-w-[440px] bg-white rounded-2xl border border-[#e5e7eb] shadow-[0_12px_32px_-4px_rgba(0,0,0,0.05),0_2px_6px_rgba(0,0,0,0.02)] p-8 sm:p-10 flex flex-col items-center">

          {/* Icon */}
          <div className="relative mb-6">
            <div className="w-16 h-16 rounded-full bg-[#B5F546] flex items-center justify-center text-[#0e0e0e] shadow-sm">
              <span className="material-symbols-outlined text-[30px]" style={{fontVariationSettings: "'FILL' 1"}}>mark_email_unread</span>
            </div>
            <div className="absolute -bottom-1 -right-1 w-6 h-6 rounded-full bg-[#0e0e0e] text-[#ffffff] flex items-center justify-center shadow">
              <span className="material-symbols-outlined text-[13px]">shield</span>
            </div>
          </div>

          {/* Header Text */}
          <div className="text-center w-full mb-6">
            <h1 className="font-[Plus_Jakarta_Sans] text-[28px] text-[#0e0e0e] mb-2 font-bold tracking-tight">Verify Your Email</h1>
            <p className="font-[Inter] text-[#525252] leading-relaxed mb-3">
              We've sent a 6-digit verification code to
            </p>
            <div className="inline-flex items-center gap-1.5 px-3.5 py-1 bg-[#F0EFEA] border border-[#e5e7eb] rounded-full max-w-full">
              <span className="material-symbols-outlined text-[14px] text-[#525252]">mail</span>
              <span className="font-[Inter] text-[13px] text-[#0e0e0e] font-semibold truncate">{email}</span>
            </div>
          </div>

          {/* Alert Banner */}
          {showAlert && (
            <div className="w-full mb-4 p-3 rounded-xl bg-[#FFDAD6] text-[#93000A] flex items-start gap-2.5 transition-all duration-200">
              <span className="material-symbols-outlined text-[18px] text-[#BA1A1A] flex-shrink-0 mt-0.5">error</span>
              <div className="flex-1">
                <p className="font-[Inter] text-[13px] font-semibold text-[#BA1A1A]">{alertTitle}</p>
                <p className="font-[Inter] text-[12px] text-[#525252] mt-0.5">{alertDescription}</p>
              </div>
              <button className="text-[#525252] hover:text-[#BA1A1A] transition-colors" onClick={dismissAlert}>
                <span className="material-symbols-outlined text-[16px]">close</span>
              </button>
            </div>
          )}

          {/* OTP Form */}
          <form className="w-full mb-6" onSubmit={handleSubmit}>
            <div className="flex items-center justify-between mb-3">
              <label className="font-[Inter] text-[11px] tracking-wider text-[#525252] uppercase font-bold">
                Verification Code
              </label>
              <button
                type="button"
                onClick={handlePaste}
                className="font-[Inter] text-[13px] text-[#0e0e0e] hover:text-[#526600] transition-colors flex items-center gap-1 font-semibold"
              >
                <span className="material-symbols-outlined text-[15px]">content_paste</span>
                Paste
              </button>
            </div>

            <div className="grid grid-cols-6 gap-2 sm:gap-2.5 w-full mb-6">
              {otp.map((digit, index) => (
                <input
                  key={index}
                  ref={el => inputRefs.current[index] = el}
                  className="otp-cell w-[52px] h-[52px] mx-auto text-center font-[Plus_Jakarta_Sans] text-[22px] font-bold rounded-lg border border-[#e5e7eb] bg-[#F0EFEA] text-[#0e0e0e] focus:bg-white focus:border-[#0e0e0e] focus:ring-2 focus:ring-[#B5F546] transition-all outline-none"
                  data-index={index}
                  inputMode="numeric"
                  maxLength={1}
                  pattern="[0-9]*"
                  type="text"
                  value={digit}
                  onChange={(e) => handleOtpChange(index, e.target.value)}
                  onKeyDown={(e) => handleKeyDown(index, e)}
                  onPaste={handlePaste}
                />
              ))}
            </div>

            <div className="flex items-center justify-between py-2.5 px-3.5 rounded-xl bg-[#F0EFEA] border border-[#e5e7eb] mb-6">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-[17px] text-[#525252]">schedule</span>
                <span className="font-[Inter] text-[13px] text-[#525252]">
                  Didn't receive the code?
                </span>
              </div>
              <div>
                {resendTimer > 0 ? (
                  <span className="font-[Inter] text-[13px] text-[#525252] font-medium">
                    Resend in <span className="font-bold text-[#0e0e0e]">{resendTimer}</span>s
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={handleResend}
                    className="font-[Inter] text-[13px] text-[#0e0e0e] hover:text-[#526600] font-bold transition-colors underline underline-offset-2"
                  >
                    Resend now
                  </button>
                )}
              </div>
            </div>

            <button
              className="w-full h-12 bg-[#0e0e0e] hover:bg-[#222222] text-[#ffffff] rounded-full font-[Plus_Jakarta_Sans] text-[15px] font-semibold flex items-center justify-center gap-2 shadow-sm active:scale-[0.99] transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={isLoading || otp.join('').length < 6}
              type="submit"
            >
              {isLoading ? (
                <>
                  <span className="material-symbols-outlined text-[18px] animate-spin">progress_activity</span>
                  <span>Verifying identity...</span>
                </>
              ) : (
                <>
                  <span>Verify Email</span>
                  <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
                </>
              )}
            </button>
          </form>

          {/* Change Email Link */}
          <div className="w-full flex items-center justify-center mt-6 pt-5 border-t border-[#f0f0ed]">
            <Link
              href="/signup"
              className="inline-flex items-center gap-1.5 font-[Inter] text-[13px] text-[#525252] hover:text-[#0e0e0e] transition-colors"
            >
              <span className="material-symbols-outlined text-[16px]">edit_square</span>
              <span>Wrong email? Change email address</span>
            </Link>
          </div>

          {/* State Simulator */}
          <div className="w-full mt-6 p-3 rounded-xl bg-[#F0EFEA] border border-[#e5e7eb] flex flex-col gap-2">
            <div className="flex items-center justify-between text-[#525252]">
              <span className="font-[Inter] text-[12px] font-semibold flex items-center gap-1 text-[#0e0e0e]">
                <span className="material-symbols-outlined text-[14px]">tune</span> State Simulator
              </span>
              <span className="font-[Inter] text-[11px]">Preview Edge Cases</span>
            </div>
            <div className="grid grid-cols-3 gap-1.5">
              <button
                type="button"
                onClick={() => { setShowAlert(false); const codes = ['7','4','2','8','9','1']; codes.forEach((c,i) => setOtp(p => { const n = [...p]; n[i] = c; return n; })); }}
                className="px-2 py-1.5 rounded-lg bg-white hover:bg-[#efeeeb] text-[#0e0e0e] border border-[#e5e7eb] font-[Inter] text-[12px] font-medium transition-colors text-center"
              >
                Active
              </button>
              <button
                type="button"
                onClick={() => { setAlertTitle('Invalid verification code'); setAlertDescription('The numbers entered do not match our records. Please try again.'); setShowAlert(true); }}
                className="px-2 py-1.5 rounded-lg bg-white hover:bg-[#FFDAD6] text-[#93000A] border border-[#e5e7eb] font-[Inter] text-[12px] font-medium transition-colors text-center"
              >
                Invalid Code
              </button>
              <button
                type="button"
                onClick={() => { setAlertTitle('Code expired'); setAlertDescription('This 6-digit authorization token has expired. Request a new one to continue.'); setShowAlert(true); }}
                className="px-2 py-1.5 rounded-lg bg-white hover:bg-[#FFDAD6] text-[#93000A] border border-[#e5e7eb] font-[Inter] text-[12px] font-medium transition-colors text-center"
              >
                Expired Code
              </button>
            </div>
          </div>
        </div>

        {/* Footer Indicators */}
        <div className="mt-8 flex items-center gap-6 text-[#525252]">
          <div className="flex items-center gap-1.5">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]">lock</span>
            <span className="font-[Inter] text-[13px]">TLS 1.3 Strict</span>
          </div>
          <div className="w-1 h-1 rounded-full bg-[#e5e7eb]"></div>
          <div className="flex items-center gap-1.5">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]">fingerprint</span>
            <span className="font-[Inter] text-[13px]">Hardware Token MFA</span>
          </div>
          <div className="w-1 h-1 rounded-full bg-[#e5e7eb]"></div>
          <div className="flex items-center gap-1.5">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]">verified</span>
            <span className="font-[Inter] text-[13px]">Zero Knowledge Vault</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-white/90 backdrop-blur-xl border-t border-[#e5e7eb] py-6">
        <div className="max-w-7xl mx-auto px-6 lg:px-12 flex flex-col sm:flex-row items-center justify-between gap-4 text-[#525252]">
          <div className="flex items-center gap-3.5">
            <span className="material-symbols-outlined text-[16px] text-[#0e0e0e]">verified_user</span>
            <span className="font-[Inter] text-[13px]">© 2025 Settle Technologies Inc. SOC2 Type II Certified.</span>
          </div>
          <div className="flex items-center gap-6">
            <Link className="font-[Inter] text-[13px] hover:text-[#0e0e0e] transition-colors" href="#">Privacy Policy</Link>
            <Link className="font-[Inter] text-[13px] hover:text-[#0e0e0e] transition-colors" href="#">Terms of Service</Link>
            <Link className="font-[Inter] text-[13px] hover:text-[#0e0e0e] transition-colors" href="#">Security Operations</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
