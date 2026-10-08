'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { REQUEST_RULES, requestAPI } from '../../../shared/services/requestAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Request, Urgency } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { formatMoney } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useInvalidateRequests, useRequestDetail } from '../hooks/useRequests';
import { StatusPill, TYPE } from './pills';

const card = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const label = 'text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#444748]';
const input = 'w-full rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60';

export function ResubmitForm({ id }: { id: string }) {
  const detail = useRequestDetail(id);
  const { user } = useAuthStore();

  if (detail.isPending) return <div className="h-96 animate-pulse rounded-2xl bg-[#efeeeb]" aria-busy="true" />;
  if (detail.isError) {
    return (
      <div className={`${card} text-center`}>
        <p className="text-[14px]">{apiErrorMessage(detail.error, "Couldn't load this request.")}</p>
        <Link href="/requests" className="mt-3 inline-block text-[13px] font-semibold underline">
          Back to requests
        </Link>
      </div>
    );
  }

  const original = detail.data;
  const allowed = original.requester.id === user?.id && (original.status === 'rejected' || original.status === 'failed');
  if (!allowed) {
    return (
      <div className={`${card} flex flex-col items-center gap-3 py-12 text-center`}>
        <Icon name="block" size={28} className="text-[#747878]" />
        <p className="max-w-md text-[14px] text-[#444748]">
          Only the requester can resubmit, and only when a request was rejected or its payment failed. {original.id} is{' '}
          {original.status}.
        </p>
        <Link href={`/requests/${original.id}`} className="rounded-full bg-black px-4 py-2 text-[13px] font-semibold text-white">
          Back to {original.id}
        </Link>
      </div>
    );
  }
  return <Form original={original} />;
}

function Form({ original }: { original: Request }) {
  const router = useRouter();
  const invalidate = useInvalidateRequests();
  const { currentOrg } = useAuthStore();
  const currency = useOrganization(currentOrg?.org_id).data?.currency ?? 'USD';

  const [amount, setAmount] = useState(String(original.amount));
  const [urgency, setUrgency] = useState<Urgency>(original.urgency);
  const [purpose, setPurpose] = useState(original.purpose);
  const [receiptMode, setReceiptMode] = useState<'carry' | 'new'>('carry');
  const [notes, setNotes] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const amountValue = Number(amount);
  const purposeLength = purpose.trim().length;
  const invalid =
    !(amountValue >= REQUEST_RULES.minAmount && amountValue <= REQUEST_RULES.maxAmount) ||
    purposeLength < REQUEST_RULES.minPurpose ||
    purposeLength > REQUEST_RULES.maxPurpose;
  const isReimbursement = original.type === 'reimbursement';
  const becomesDraft = isReimbursement && (receiptMode === 'new' || original.receipts.length === 0);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (invalid) {
      setError(`Check the amount (max ${REQUEST_RULES.maxAmount.toLocaleString('en-US')}) and purpose (10–500 characters).`);
      return;
    }
    setBusy(true);
    setError('');
    try {
      const created = await requestAPI.resubmit(original.id, {
        amount: amountValue,
        purpose: purpose.trim(),
        urgency,
        ...(isReimbursement ? { receipt_mode: receiptMode } : {}),
      });
      if (notes.trim()) {
        await requestAPI.addComment(created.id, notes.trim()).catch(() => undefined);
      }
      await invalidate();
      router.push(`/requests/${created.id}`);
    } catch (err) {
      setError(apiErrorMessage(err, 'Could not resubmit.'));
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="flex flex-col gap-6" noValidate>
      <div className="flex flex-col gap-2">
        <Link href={`/requests/${original.id}`} className="flex w-fit items-center gap-1 text-[13px] text-[#444748] hover:text-[#1b1c1a]">
          <Icon name="arrow_back" size={14} />
          Back to {original.id}
        </Link>
        <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Resubmit request</h1>
      </div>

      <div className="relative overflow-hidden rounded-2xl bg-[#1b1c1c] p-5 text-white">
        <div aria-hidden className="pointer-events-none absolute -right-10 -top-16 size-40 rounded-full bg-[rgba(199,239,57,0.12)] blur-[32px]" />
        <p className="relative flex items-start gap-3 text-[13px] leading-5 text-[#d6d6d5]">
          <Icon name="link" size={18} className="mt-0.5 text-[#c7ef39]" />
          <span>
            This creates a <span className="font-semibold text-white">new request</span> linked to {original.id}. The original stays{' '}
            {original.status} for the record. Change what finance asked for below.
          </span>
        </p>
      </div>

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-5">
          <section className={card}>
            <h2 className={`${fontHeading} text-[16px] font-semibold`}>Amount & urgency</h2>
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="r-amount" className={label}>
                  Amount ({currency})
                </label>
                <input
                  id="r-amount"
                  type="number"
                  inputMode="decimal"
                  step="0.01"
                  min={REQUEST_RULES.minAmount}
                  max={REQUEST_RULES.maxAmount}
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  className={`${input} ${fontHeading} text-[20px] font-bold`}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label htmlFor="r-urgency" className={label}>
                  Urgency
                </label>
                <select id="r-urgency" value={urgency} onChange={(e) => setUrgency(e.target.value as Urgency)} className={`${input} h-[52px]`}>
                  <option value="routine">Routine</option>
                  <option value="urgent">Urgent</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
            </div>
          </section>

          <section className={card}>
            <div className="flex items-center justify-between">
              <label htmlFor="r-purpose" className={`${fontHeading} text-[16px] font-semibold`}>
                Purpose
              </label>
              <span className="font-mono text-[11px] text-[#747878]">
                {purposeLength} / {REQUEST_RULES.maxPurpose}
              </span>
            </div>
            <textarea id="r-purpose" rows={4} value={purpose} onChange={(e) => setPurpose(e.target.value)} className={`${input} mt-3 resize-y`} />
          </section>

          {isReimbursement && (
            <section className={card}>
              <h2 className={`${fontHeading} text-[16px] font-semibold`}>Receipts</h2>
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                {(
                  [
                    ['carry', 'Keep my receipts', `Copy the ${original.receipts.length} receipt(s) from ${original.id}.`],
                    ['new', 'Upload new receipts', 'Start the new request as a draft and add receipts there.'],
                  ] as const
                ).map(([value, title, body]) => (
                  <label
                    key={value}
                    className={`flex cursor-pointer flex-col gap-1 rounded-xl border p-4 ${receiptMode === value ? 'border-[#1b1c1a] bg-white' : 'border-[#e3e2df] bg-[#faf9f6]'}`}
                  >
                    <input type="radio" name="receipt-mode" value={value} checked={receiptMode === value} onChange={() => setReceiptMode(value)} className="sr-only" />
                    <span className="flex items-center gap-2 text-[14px] font-semibold">
                      <Icon name={receiptMode === value ? 'radio_button_checked' : 'radio_button_unchecked'} size={16} />
                      {title}
                    </span>
                    <span className="text-[12px] text-[#444748]">{body}</span>
                  </label>
                ))}
              </div>
            </section>
          )}

          <section className={card}>
            <label htmlFor="r-notes" className={`${fontHeading} text-[16px] font-semibold`}>
              Notes for the approver <span className="text-[12px] font-normal text-[#747878]">(optional)</span>
            </label>
            <textarea
              id="r-notes"
              rows={3}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="What changed since the last submission?"
              className={`${input} mt-3 resize-y`}
            />
          </section>

          {error && (
            <p role="alert" className="flex items-center gap-1 text-[13px] font-medium text-[#ba1a1a]">
              <Icon name="error" size={15} />
              {error}
            </p>
          )}
          <div className="flex items-center justify-between gap-3">
            <Link href={`/requests/${original.id}`} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
              Cancel
            </Link>
            <button
              type="submit"
              disabled={busy}
              className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-black px-5 py-2.5 text-[14px] font-semibold text-white disabled:opacity-50`}
            >
              {busy ? 'Resubmitting…' : becomesDraft ? 'Create new draft' : 'Resubmit for review'}
              <Icon name="arrow_forward" size={15} />
            </button>
          </div>
          {becomesDraft && <p className="-mt-3 text-right text-[12px] text-[#444748]">The new request starts as a draft so you can attach receipts.</p>}
        </div>

        <aside className={`${card} flex flex-col gap-3`}>
          <p className={label}>Original request</p>
          <div className="flex items-center gap-2">
            <span className="font-mono text-[13px] font-semibold">{original.id}</span>
            <StatusPill status={original.status} />
          </div>
          <p className={`${fontHeading} text-[22px] font-bold`}>{formatMoney(original.amount, currency)}</p>
          <p className="text-[13px] text-[#444748]">
            {TYPE[original.type].label} · {original.urgency}
          </p>
          <p className="line-clamp-4 text-[13px] text-[#444748]">{original.purpose}</p>
          {original.comments.length > 0 && (
            <div className="rounded-xl bg-[#f4f3f0] p-3">
              <p className={label}>Latest comment</p>
              <p className="mt-1 text-[13px]">
                {original.comments[original.comments.length - 1].content}
                {original.comments[original.comments.length - 1].author && (
                  <span className="text-[#747878]"> — {original.comments[original.comments.length - 1].author}</span>
                )}
              </p>
            </div>
          )}
        </aside>
      </div>
    </form>
  );
}
