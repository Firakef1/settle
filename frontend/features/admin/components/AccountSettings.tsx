'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import { authAPI } from '../../../shared/services/authAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Role } from '../../../shared/types';
import { getApiError } from '../../../shared/utils/apiError';
import { ROLES } from '../../../shared/utils/constants';
import { formatDateOnly } from '../../../shared/utils/format';
import { fontHeading, initials } from '../../../shared/utils/fonts';

const card = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';

export function AccountSettings() {
  const router = useRouter();
  const { user, currentOrg, logout } = useAuthStore();
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const signOut = async () => {
    await logout();
    router.replace('/login');
  };

  const deleteAccount = async () => {
    if (!password) return;
    setBusy(true);
    setError('');
    try {
      await authAPI.deleteAccount(password);
      await logout();
      router.replace('/login');
    } catch (err) {
      const { status, message } = getApiError(err, 'Could not delete the account.');
      setError(status === 403 ? 'That password is incorrect.' : message);
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Account</h1>
        <p className="text-[14px] text-[#444748]">Your profile and sign-in.</p>
      </div>

      <section className={`${card} flex flex-wrap items-center gap-5`}>
        <span className={`${fontHeading} flex size-16 items-center justify-center rounded-full bg-[#c7ef39] text-[22px] font-bold text-[#171e00]`}>
          {initials(user?.name)}
        </span>
        <div className="min-w-0 flex-1">
          <p className={`${fontHeading} text-[20px] font-bold`}>{user?.name}</p>
          <p className="text-[14px] text-[#444748]">{user?.email}</p>
          <p className="mt-1 flex flex-wrap items-center gap-2 text-[12px] text-[#747878]">
            <span className="rounded-full bg-[#efeeeb] px-2 py-0.5 font-semibold text-[#1b1c1a]">{ROLES[(currentOrg?.role ?? 'staff') as Role]}</span>
            in {currentOrg?.org_name}
            {user?.created_at && <span>· member since {formatDateOnly(user.created_at)}</span>}
            {user?.email_verified && (
              <span className="flex items-center gap-0.5 text-[#526600]">
                <Icon name="verified" size={13} />
                verified
              </span>
            )}
          </p>
        </div>
        <p className="w-full text-[12px] text-[#747878]">Name and email can&apos;t be changed in the app yet.</p>
      </section>

      <section className={`${card} flex flex-wrap items-center justify-between gap-4`}>
        <div>
          <h2 className={`${fontHeading} text-[16px] font-semibold`}>Sign out</h2>
          <p className="text-[13px] text-[#444748]">End your session on this device.</p>
        </div>
        <button type="button" onClick={signOut} className="flex items-center gap-1.5 rounded-full bg-[#efeeeb] px-4 py-2 text-[14px] font-semibold hover:bg-[#e3e2df]">
          <Icon name="logout" size={16} />
          Log out
        </button>
      </section>

      <section className={`${card} flex flex-wrap items-center justify-between gap-4 border border-[#ffdad6]`}>
        <div>
          <h2 className={`${fontHeading} text-[16px] font-semibold text-[#93000a]`}>Delete account</h2>
          <p className="max-w-lg text-[13px] text-[#444748]">Permanently deletes your Settle account and signs you out. This can&apos;t be undone.</p>
        </div>
        <button
          type="button"
          onClick={() => setDeleting(true)}
          className="flex items-center gap-1.5 rounded-full bg-[#ba1a1a] px-4 py-2 text-[14px] font-semibold text-white hover:bg-[#93000a]"
        >
          <Icon name="delete_forever" size={16} />
          Delete account
        </button>
      </section>

      <Modal
        open={deleting}
        onClose={() => {
          setDeleting(false);
          setPassword('');
          setError('');
        }}
        icon="delete_forever"
        tone="danger"
        title="Delete your account?"
        description="Enter your password to confirm. You'll be signed out right away."
        footer={
          <>
            <button
              type="button"
              onClick={() => setDeleting(false)}
              className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={deleteAccount}
              disabled={!password || busy}
              className={`${fontHeading} rounded-full bg-[#ba1a1a] px-4 py-2 text-[14px] font-semibold text-white disabled:opacity-40`}
            >
              {busy ? 'Deleting…' : 'Delete my account'}
            </button>
          </>
        }
      >
        <div className="flex flex-col gap-1.5">
          <label htmlFor="delete-password" className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">
            Current password
          </label>
          <input
            id="delete-password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && deleteAccount()}
            className="h-11 rounded-xl bg-[#f4f3f0] px-4 text-[14px] outline-none focus:ring-2 focus:ring-[#ffdad6]"
          />
          {error && (
            <p role="alert" className="flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a]">
              <Icon name="error" size={14} />
              {error}
            </p>
          )}
        </div>
      </Modal>
    </div>
  );
}
