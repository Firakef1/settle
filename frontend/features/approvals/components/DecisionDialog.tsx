'use client';

import { useEffect, useState } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import type { PaymentMethod, RequestStatus, Urgency } from '../../../shared/types';
import { formatMoney } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useDecision, type DecisionKind } from '../hooks/useDecision';

export interface DecisionTarget {
  id: string;
  amount: number;
  purpose: string;
  status: RequestStatus;
  urgency: Urgency;
  requesterName: string;
}

const KINDS: Record<
  DecisionKind,
  { tab: string; title: string; icon: string; tone: 'success' | 'danger'; button: string; buttonClass: string; textLabel?: string; required?: boolean; placeholder?: string }
> = {
  approve: {
    tab: 'Approve',
    title: 'Approve request',
    icon: 'check_circle',
    tone: 'success',
    button: 'Approve',
    buttonClass: 'bg-[#c7ef39] text-[#171e00] hover:bg-[#b5f546]',
    textLabel: 'Note (optional)',
    placeholder: 'e.g. Approved for the conference budget.',
  },
  reject: {
    tab: 'Reject',
    title: 'Reject request',
    icon: 'cancel',
    tone: 'danger',
    button: 'Reject',
    buttonClass: 'bg-[#ba1a1a] text-white hover:bg-[#93000a]',
    textLabel: 'Reason for the requester',
    required: true,
    placeholder: 'e.g. Missing the itemised hotel invoice.',
  },
  paid: {
    tab: 'Mark paid',
    title: 'Record payment',
    icon: 'payments',
    tone: 'success',
    button: 'Mark as paid',
    buttonClass: 'bg-black text-white hover:bg-[#1c1b1b]',
  },
  failed: {
    tab: 'Payment failed',
    title: 'Payment failed',
    icon: 'error',
    tone: 'danger',
    button: 'Mark as failed',
    buttonClass: 'bg-[#ba1a1a] text-white hover:bg-[#93000a]',
    textLabel: 'What went wrong',
    required: true,
    placeholder: 'e.g. The bank rejected the account number.',
  },
};

const METHODS: Array<{ value: PaymentMethod; label: string; icon: string }> = [
  { value: 'bank_transfer', label: 'Bank transfer', icon: 'account_balance' },
  { value: 'check', label: 'Check', icon: 'edit_document' },
  { value: 'cash', label: 'Cash', icon: 'payments' },
  { value: 'other', label: 'Other', icon: 'more_horiz' },
];

// Only the decisions the API allows for the current status.
export function allowedDecisions(status: RequestStatus): DecisionKind[] {
  if (status === 'pending') return ['approve', 'reject'];
  if (status === 'approved') return ['paid', 'failed'];
  return [];
}

export function DecisionDialog({
  target,
  initial,
  currency,
  onClose,
}: {
  target: DecisionTarget | null;
  initial: DecisionKind;
  currency: string;
  onClose: () => void;
}) {
  const decision = useDecision();
  const [kind, setKind] = useState<DecisionKind>(initial);
  const [text, setText] = useState('');
  const [method, setMethod] = useState<PaymentMethod>('bank_transfer');
  const [touched, setTouched] = useState(false);

  useEffect(() => {
    setKind(initial);
    setText('');
    setTouched(false);
    setMethod('bank_transfer');
  }, [target?.id, initial]);

  if (!target) return null;
  const allowed = allowedDecisions(target.status);
  const cfg = KINDS[kind];
  const missing = !!cfg.required && !text.trim();

  const confirm = () => {
    setTouched(true);
    if (missing) return;
    decision.mutate(
      { kind, requestId: target.id, text: text.trim() || undefined, method },
      { onSuccess: onClose, onError: (err) => (err as { response?: { status?: number } })?.response?.status === 409 && onClose() },
    );
  };

  return (
    <Modal
      open
      onClose={onClose}
      icon={cfg.icon}
      tone={cfg.tone}
      title={cfg.title}
      description={`${target.requesterName} · ${target.urgency} urgency`}
      width="max-w-[520px]"
      footer={
        <>
          <button type="button" onClick={onClose} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
            Cancel
          </button>
          <button
            type="button"
            onClick={confirm}
            disabled={decision.isPending}
            className={`${fontHeading} rounded-full px-5 py-2 text-[14px] font-semibold disabled:opacity-50 ${cfg.buttonClass}`}
          >
            {decision.isPending ? 'Saving…' : cfg.button}
          </button>
        </>
      }
    >
      {allowed.length > 1 && (
        <div role="tablist" aria-label="Decision" className="grid grid-cols-2 gap-1 rounded-full bg-[#f4f3f0] p-1">
          {allowed.map((k) => (
            <button
              key={k}
              type="button"
              role="tab"
              aria-selected={kind === k}
              onClick={() => {
                setKind(k);
                setTouched(false);
              }}
              className={`rounded-full px-3 py-1.5 text-[13px] transition-colors ${kind === k ? 'bg-white font-semibold shadow-sm' : 'text-[#444748]'}`}
            >
              {KINDS[k].tab}
            </button>
          ))}
        </div>
      )}

      <div className="flex items-center justify-between gap-3 rounded-xl bg-[#f6f6f4] px-4 py-3">
        <div className="min-w-0">
          <p className="font-mono text-[12px] font-semibold text-[#444748]">{target.id}</p>
          <p className="truncate text-[14px] font-medium">{target.purpose}</p>
        </div>
        <p className={`${fontHeading} whitespace-nowrap text-[18px] font-bold`}>{formatMoney(target.amount, currency)}</p>
      </div>

      {kind === 'paid' ? (
        <fieldset>
          <legend className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">Payment method</legend>
          <div className="mt-2 grid grid-cols-2 gap-2">
            {METHODS.map((m) => (
              <label
                key={m.value}
                className={`flex cursor-pointer items-center gap-2 rounded-xl border px-3 py-2.5 text-[13px] ${
                  method === m.value ? 'border-[#1b1c1a] bg-white font-semibold' : 'border-[#e3e2df] bg-[#faf9f6]'
                }`}
              >
                <input type="radio" name="method" value={m.value} checked={method === m.value} onChange={() => setMethod(m.value)} className="sr-only" />
                <Icon name={m.icon} size={16} />
                {m.label}
              </label>
            ))}
          </div>
        </fieldset>
      ) : (
        <div className="flex flex-col gap-1.5">
          <label htmlFor="decision-text" className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">
            {cfg.textLabel}
          </label>
          <textarea
            id="decision-text"
            rows={4}
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={cfg.placeholder}
            aria-invalid={touched && missing}
            className="w-full resize-y rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
          />
          {touched && missing && (
            <p role="alert" className="flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a]">
              <Icon name="error" size={14} />A reason is required.
            </p>
          )}
          {kind === 'reject' && <p className="text-[12px] text-[#747878]">The requester sees this reason and can resubmit.</p>}
        </div>
      )}
    </Modal>
  );
}
