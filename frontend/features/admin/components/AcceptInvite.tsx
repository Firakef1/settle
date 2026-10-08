'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { orgAPI } from '../../../shared/services/orgAPI';
import { getApiError } from '../../../shared/utils/apiError';
import { fontHeading } from '../../../shared/utils/fonts';

const input = 'h-11 w-full rounded-xl border border-[#e5e4e0] bg-[#f6f6f4] px-3.5 text-[14px] outline-none focus:border-[#0e0e0e] focus:bg-white';

function errorFor(status: number | undefined, message: string) {
  if (status === 404) return { title: 'Invite not found', body: 'This link is wrong or was never created. Ask your admin for a new one.' };
  if (status === 409) return { title: "You're already a member", body: 'This account is already in the organization. Just sign in.' };
  if (status === 400 && /password/i.test(message)) return null; // shown inline
  if (status === 400) return { title: 'This invite has expired or was used', body: 'Invite links last 7 days and work once. Ask your admin for a new one.' };
  return null;
}

export function AcceptInvite({ token }: { token: string }) {
  const router = useRouter();
  const [existing, setExisting] = useState(false);
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [inline, setInline] = useState('');
  const [fatal, setFatal] = useState<{ title: string; body: string } | null>(null);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setInline('');
    if (!existing && password.length < 6) {
      setInline('Choose a password of at least 6 characters.');
      return;
    }
    setBusy(true);
    try {
      const member = await orgAPI.acceptInvite(token, existing ? {} : { name: name.trim(), password });
      router.push(`/login?invited=1&email=${encodeURIComponent(member.email)}`);
    } catch (err) {
      const { status, message } = getApiError(err, 'Could not accept the invite.');
      const f = errorFor(status, message);
      if (f) setFatal(f);
      else setInline(message);
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col bg-[#faf9f6] text-[#1b1c1a]">
      <header className="border-b border-[#e5e4e0] bg-white">
        <div className="mx-auto flex h-16 max-w-5xl items-center gap-2.5 px-6">
          <Image src="/brand/settle-logo.jpg" alt="" width={32} height={32} className="size-8 rounded-lg" />
          <span className={`${fontHeading} text-lg font-bold`}>Settle</span>
        </div>
      </header>
      <main className="flex flex-1 items-start justify-center px-4 py-14">
        <div className="w-full max-w-md rounded-2xl border border-[#e5e4e0] bg-white p-8 shadow-sm">
          {fatal ? (
            <div className="flex flex-col items-center gap-3 text-center">
              <span className="flex size-12 items-center justify-center rounded-2xl bg-[#ffdad6] text-[#ba1a1a]">
                <Icon name="link_off" size={22} />
              </span>
              <h1 className={`${fontHeading} text-[22px] font-bold`}>{fatal.title}</h1>
              <p className="text-[14px] text-[#575a5a]">{fatal.body}</p>
              <Link href="/login" className={`${fontHeading} mt-2 rounded-full bg-[#0e0e0e] px-5 py-2.5 text-[14px] font-semibold text-white`}>
                Go to sign in
              </Link>
            </div>
          ) : (
            <form onSubmit={submit} className="flex flex-col gap-5" noValidate>
              <div className="flex flex-col items-center gap-3 text-center">
                <span className="flex size-12 items-center justify-center rounded-2xl bg-[#c7ef39] text-[#171e00]">
                  <Icon name="group_add" size={22} />
                </span>
                <h1 className={`${fontHeading} text-[24px] font-bold tracking-tight`}>You&apos;re invited to Settle</h1>
                <p className="text-[14px] text-[#575a5a]">Accept to join your team&apos;s organization and start submitting payout requests.</p>
              </div>

              <div className="grid grid-cols-2 gap-1 rounded-full bg-[#f4f3f0] p-1" role="tablist">
                {[
                  [false, "I'm new to Settle"],
                  [true, 'I have an account'],
                ].map(([value, text]) => (
                  <button
                    key={String(value)}
                    type="button"
                    role="tab"
                    aria-selected={existing === value}
                    onClick={() => {
                      setExisting(value as boolean);
                      setInline('');
                    }}
                    className={`rounded-full px-3 py-1.5 text-[13px] ${existing === value ? 'bg-white font-semibold shadow-sm' : 'text-[#444748]'}`}
                  >
                    {text as string}
                  </button>
                ))}
              </div>

              {existing ? (
                <p className="rounded-xl bg-[#f4f3f0] px-4 py-3 text-[13px] text-[#444748]">
                  Your existing account (the invited email) joins the organization. You&apos;ll sign in with your current password next.
                </p>
              ) : (
                <>
                  <div className="flex flex-col gap-1.5">
                    <label htmlFor="invite-name" className="text-[12px] font-semibold uppercase tracking-wide">
                      Full name
                    </label>
                    <input id="invite-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Grace Hopper" autoComplete="name" className={input} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label htmlFor="invite-password" className="text-[12px] font-semibold uppercase tracking-wide">
                      Password
                    </label>
                    <input
                      id="invite-password"
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="At least 6 characters"
                      autoComplete="new-password"
                      className={input}
                    />
                    <p className="text-[12px] text-[#747878]">After joining, we&apos;ll email a code to verify your address.</p>
                  </div>
                </>
              )}

              {inline && (
                <p role="alert" className="flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a]">
                  <Icon name="error" size={14} />
                  {inline}
                </p>
              )}

              <button
                type="submit"
                disabled={busy}
                className={`${fontHeading} flex h-12 items-center justify-center gap-2 rounded-full bg-[#0e0e0e] text-[15px] font-semibold text-white disabled:opacity-50`}
              >
                {busy ? 'Joining…' : 'Accept invite'}
                <Icon name="arrow_forward" size={16} className="text-[#b5f546]" />
              </button>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}
