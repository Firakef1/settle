'use client';

import { useState, type FormEvent } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import type { Invitation } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { formatDateOnly } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useInvite } from '../hooks/useMembers';

export function inviteLink(token: string) {
  return `${typeof window !== 'undefined' ? window.location.origin : ''}/invite/${token}`;
}

export function CopyLink({ token }: { token: string }) {
  const [copied, setCopied] = useState(false);
  const link = inviteLink(token);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard blocked: the link is selectable in the input.
    }
  };
  return (
    <div className="flex items-center gap-2 rounded-xl bg-[#f4f3f0] p-1.5 pl-3">
      <input readOnly value={link} aria-label="Invite link" onFocus={(e) => e.target.select()} className="min-w-0 flex-1 bg-transparent font-mono text-[12px] outline-none" />
      <button
        type="button"
        onClick={copy}
        className={`flex shrink-0 items-center gap-1 rounded-lg px-3 py-1.5 text-[12px] font-semibold ${copied ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-black text-white'}`}
      >
        <Icon name={copied ? 'check' : 'content_copy'} size={14} />
        {copied ? 'Copied' : 'Copy'}
      </button>
    </div>
  );
}

export function InviteDialog({ open, orgId, onClose }: { open: boolean; orgId: string; onClose: () => void }) {
  const invite = useInvite(orgId);
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<'staff' | 'finance'>('staff');
  const [created, setCreated] = useState<Invitation | null>(null);
  const validEmail = /\S+@\S+\.\S+/.test(email.trim());

  const close = () => {
    setEmail('');
    setRole('staff');
    setCreated(null);
    invite.reset();
    onClose();
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!validEmail) return;
    invite.mutate({ email: email.trim(), role }, { onSuccess: setCreated });
  };

  return (
    <Modal
      open={open}
      onClose={close}
      icon={created ? 'mark_email_read' : 'person_add'}
      tone={created ? 'success' : 'default'}
      title={created ? 'Invite link ready' : 'Invite a teammate'}
      description={
        created
          ? `Settle doesn't send emails yet — share this link with ${created.email}. It expires ${formatDateOnly(created.expires_at)}.`
          : 'They get a link to join this organization. Admins can only invite Staff or Finance members.'
      }
    >
      {created ? (
        <div className="flex flex-col gap-4">
          <CopyLink token={created.token} />
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => {
                setCreated(null);
                setEmail('');
              }}
              className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]"
            >
              Invite another
            </button>
            <button type="button" onClick={close} className={`${fontHeading} rounded-full bg-black px-5 py-2 text-[14px] font-semibold text-white`}>
              Done
            </button>
          </div>
        </div>
      ) : (
        <form onSubmit={submit} className="flex flex-col gap-4" noValidate>
          <div className="flex flex-col gap-1.5">
            <label htmlFor="invite-email" className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">
              Work email
            </label>
            <input
              id="invite-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="name@company.com"
              className="h-11 rounded-xl bg-[#f4f3f0] px-4 text-[14px] outline-none focus:ring-2 focus:ring-[#c7ef39]/60"
            />
          </div>
          <fieldset>
            <legend className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">Role</legend>
            <div className="mt-1.5 grid grid-cols-2 gap-2">
              {(
                [
                  ['staff', 'Staff', 'Raises their own requests'],
                  ['finance', 'Finance', 'Reviews and pays requests'],
                ] as const
              ).map(([value, title, body]) => (
                <label
                  key={value}
                  className={`flex cursor-pointer flex-col rounded-xl border px-3.5 py-2.5 ${role === value ? 'border-[#1b1c1a] bg-white' : 'border-[#e3e2df] bg-[#faf9f6]'}`}
                >
                  <input type="radio" name="invite-role" value={value} checked={role === value} onChange={() => setRole(value)} className="sr-only" />
                  <span className="text-[14px] font-semibold">{title}</span>
                  <span className="text-[12px] text-[#444748]">{body}</span>
                </label>
              ))}
            </div>
          </fieldset>
          {invite.isError && (
            <p role="alert" className="flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a]">
              <Icon name="error" size={14} />
              {apiErrorMessage(invite.error, 'Could not create the invite.')}
            </p>
          )}
          <div className="flex justify-end gap-2 border-t border-[#efeeeb] pt-4">
            <button type="button" onClick={close} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
              Cancel
            </button>
            <button
              type="submit"
              disabled={!validEmail || invite.isPending}
              className={`${fontHeading} rounded-full bg-black px-5 py-2 text-[14px] font-semibold text-white disabled:opacity-40`}
            >
              {invite.isPending ? 'Creating…' : 'Create invite link'}
            </button>
          </div>
        </form>
      )}
    </Modal>
  );
}
