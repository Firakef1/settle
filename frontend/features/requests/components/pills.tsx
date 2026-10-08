import { Icon } from '../../../shared/components/Icon';
import type { RequestStatus, RequestType, Urgency } from '../../../shared/types';

const pill = 'inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-[12px] leading-4 tracking-[-0.07px]';

const URGENCY: Record<Urgency, { label: string; className: string; dot?: string }> = {
  routine: { label: 'Routine', className: 'bg-[#efeeeb] font-medium text-[#444748]' },
  urgent: { label: 'Urgent', className: 'bg-[#e3e2df] font-medium text-[#1b1c1a]', dot: 'bg-[#f59e0b]' },
  critical: { label: 'Critical', className: 'bg-[#ffdad6] font-semibold text-[#ba1a1a]', dot: 'bg-[#ba1a1a]' },
};

export const STATUS: Record<RequestStatus, { label: string; icon: string; className: string }> = {
  draft: { label: 'Draft', icon: 'edit_note', className: 'bg-[#efeeeb] font-medium text-[#444748]' },
  pending: { label: 'Pending', icon: 'hourglass_top', className: 'bg-[#efeeeb] font-medium text-[#444748]' },
  approved: { label: 'Approved', icon: 'check_circle', className: 'bg-[rgba(199,239,57,0.2)] font-semibold text-[#526600]' },
  paid: { label: 'Paid', icon: 'payments', className: 'bg-[#1b1c1a] font-semibold text-white' },
  rejected: { label: 'Rejected', icon: 'cancel', className: 'bg-[rgba(186,26,26,0.1)] font-semibold text-[#ba1a1a]' },
  failed: { label: 'Payment failed', icon: 'error', className: 'bg-[rgba(186,26,26,0.1)] font-semibold text-[#ba1a1a]' },
  withdrawn: { label: 'Withdrawn', icon: 'undo', className: 'bg-[#efeeeb] font-medium text-[#747878]' },
};

export const TYPE: Record<RequestType, { label: string; icon: string; description: string }> = {
  reimbursement: { label: 'Reimbursement', icon: 'receipt_long', description: 'I paid out of pocket and need the money back' },
  advance: { label: 'Advance', icon: 'payments', description: 'Money up front before a purchase or trip' },
  stipend: { label: 'Stipend', icon: 'event_repeat', description: 'A fixed allowance or grant disbursement' },
};

export function UrgencyPill({ urgency }: { urgency: Urgency }) {
  const u = URGENCY[urgency];
  return (
    <span className={`${pill} ${u.className}`}>
      {u.dot && <span aria-hidden className={`size-1.5 rounded-full ${u.dot}`} />}
      {u.label}
    </span>
  );
}

export function StatusPill({ status }: { status: RequestStatus }) {
  const s = STATUS[status];
  return (
    <span className={`${pill} ${s.className}`}>
      <Icon name={s.icon} size={13} />
      {s.label}
    </span>
  );
}
