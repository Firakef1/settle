'use client';

import { useState } from 'react';
import { Icon } from '../../../shared/components/Icon';
import type { Request } from '../../../shared/types';
import { formatISO } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useLatestApproval, type DecisionKind } from '../hooks/useDecision';
import { DecisionDialog, allowedDecisions } from './DecisionDialog';

const METHOD_LABEL: Record<string, string> = { bank_transfer: 'Bank transfer', check: 'Check', cash: 'Cash', other: 'Other' };

const BUTTONS: Record<DecisionKind, { label: string; icon: string; className: string }> = {
  approve: { label: 'Approve', icon: 'check_circle', className: 'bg-[#c7ef39] text-[#171e00] hover:bg-[#b5f546]' },
  reject: { label: 'Reject', icon: 'cancel', className: 'bg-[rgba(255,218,214,0.5)] text-[#93000a] hover:bg-[#ffdad6]' },
  paid: { label: 'Mark as paid', icon: 'payments', className: 'bg-[#c7ef39] text-[#171e00] hover:bg-[#b5f546]' },
  failed: { label: 'Payment failed', icon: 'error', className: 'bg-[rgba(255,218,214,0.5)] text-[#93000a] hover:bg-[#ffdad6]' },
};

// Right-column panel on the request detail for finance and org admins.
export function FinancePanel({ request, currency }: { request: Request; currency: string }) {
  const [open, setOpen] = useState<DecisionKind | null>(null);
  const latest = useLatestApproval(request.id).data;
  const actions = allowedDecisions(request.status);

  return (
    <section className="relative overflow-hidden rounded-2xl bg-[#1b1c1c] p-5 text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1),0px_8px_10px_-6px_rgba(0,0,0,0.1)]">
      <div aria-hidden className="pointer-events-none absolute -right-12 -top-12 size-40 rounded-full bg-[rgba(199,239,57,0.1)] blur-[32px]" />
      <p className="relative flex items-center gap-2 text-[12px] font-semibold uppercase tracking-[0.55px] text-[#848483]">
        <Icon name="bolt" size={15} className="text-[#c7ef39]" />
        Finance decision
      </p>
      <p className={`${fontHeading} relative mt-2 text-[18px] font-bold leading-6`}>
        {request.status === 'pending'
          ? 'Approve or reject this request'
          : request.status === 'approved'
            ? 'Record the payment outcome'
            : latest
              ? 'Decision recorded'
              : 'No decision needed'}
      </p>
      {actions.length > 0 && (
        <div className="relative mt-4 flex flex-col gap-2">
          {actions.map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setOpen(k)}
              className={`${fontHeading} flex items-center justify-center gap-1.5 rounded-full px-4 py-2.5 text-[14px] font-semibold transition-colors ${BUTTONS[k].className}`}
            >
              <Icon name={BUTTONS[k].icon} size={16} />
              {BUTTONS[k].label}
            </button>
          ))}
        </div>
      )}

      {latest && (
        <dl className="relative mt-4 flex flex-col gap-2 border-t border-[rgba(227,226,223,0.1)] pt-4 text-[13px]">
          <div className="flex justify-between gap-3">
            <dt className="text-[#848483]">Decision</dt>
            <dd className="font-semibold capitalize">{latest.decision.replace('_', ' ')}</dd>
          </div>
          {latest.decision_note && (
            <div className="flex justify-between gap-3">
              <dt className="text-[#848483]">Note</dt>
              <dd className="text-right">{latest.decision_note}</dd>
            </div>
          )}
          {latest.payment_status && (
            <div className="flex justify-between gap-3">
              <dt className="text-[#848483]">Payment</dt>
              <dd className="capitalize">{latest.payment_status.replace('_', ' ')}</dd>
            </div>
          )}
          {latest.payment_method && (
            <div className="flex justify-between gap-3">
              <dt className="text-[#848483]">Method</dt>
              <dd>{METHOD_LABEL[latest.payment_method] ?? latest.payment_method}</dd>
            </div>
          )}
          {latest.payment_at && (
            <div className="flex justify-between gap-3">
              <dt className="text-[#848483]">Paid at</dt>
              <dd>{formatISO(latest.payment_at)}</dd>
            </div>
          )}
          {latest.failure_reason && (
            <div className="flex justify-between gap-3">
              <dt className="text-[#848483]">Failure</dt>
              <dd className="text-right">{latest.failure_reason}</dd>
            </div>
          )}
          <div className="flex justify-between gap-3">
            <dt className="text-[#848483]">Updated</dt>
            <dd>{formatISO(latest.updated_at)}</dd>
          </div>
        </dl>
      )}

      <DecisionDialog
        target={
          open
            ? {
                id: request.id,
                amount: request.amount,
                purpose: request.purpose,
                status: request.status,
                urgency: request.urgency,
                requesterName: request.requester.name,
              }
            : null
        }
        initial={open ?? 'approve'}
        currency={currency}
        onClose={() => setOpen(null)}
      />
    </section>
  );
}
