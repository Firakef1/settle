'use client';

import Link from 'next/link';
import { useMemo, useState, type ReactNode } from 'react';
import type { RequestListItem, RequestStatus, Role, Urgency } from '../../../shared/types';
import { REQUEST_STATUSES, ROLES, URGENCIES } from '../../../shared/utils/constants';
import { formatDateOnly, formatMoney } from '../../../shared/utils/format';
import { useMyRequests } from '../api';
import {
  DashboardSkeleton,
  ErrorState,
  Icon,
  Pill,
  PulseDot,
  SegmentBar,
  errorMessage,
  fontHeading,
  greeting,
  type Segment,
} from './ui';

const RECENT_ROWS = 5;
const NEEDS_ATTENTION: RequestStatus[] = ['rejected', 'failed'];
const cardShadow = 'shadow-[0px_1px_2px_0px_rgba(0,0,0,0.05)]';

type Tab = 'all' | 'attention' | 'pending' | 'approved';

interface StaffDashboardProps {
  orgId: string;
  orgName: string;
  userName?: string;
  role: Role;
  currency: string;
}

export function StaffDashboard({ orgId, orgName, userName, role, currency }: StaffDashboardProps) {
  const requests = useMyRequests(orgId);
  const [tab, setTab] = useState<Tab>('all');

  const items = useMemo(() => requests.data?.data ?? [], [requests.data]);
  const counts = useMemo(() => countByStatus(items), [items]);

  if (requests.isPending) return <DashboardSkeleton />;
  if (requests.isError) {
    return <ErrorState message={errorMessage(requests.error)} onRetry={() => requests.refetch()} />;
  }

  const total = requests.data.meta.total;
  const approvedVolume = sumAmount(items, 'approved');
  const paidVolume = sumAmount(items, 'paid');
  const pending = items.filter((r) => r.status === 'pending');
  const avgPending = pending.length ? pending.reduce((sum, r) => sum + r.days_pending, 0) / pending.length : null;

  return (
    <div className="flex flex-col gap-7">
      <Greeting title={greeting(userName)} role={role} orgName={orgName} avgPending={avgPending} />
      <KpiCards counts={counts} approvedVolume={formatMoney(approvedVolume, currency)} />
      <Pipeline counts={counts} total={total} paidVolume={formatMoney(paidVolume, currency)} />
      <RecentRequests
        items={items}
        counts={counts}
        total={total}
        tab={tab}
        onTabChange={setTab}
        currency={currency}
        paidVolume={formatMoney(paidVolume, currency)}
      />
    </div>
  );
}

function countByStatus(items: RequestListItem[]): Record<RequestStatus, number> {
  const counts: Record<RequestStatus, number> = {
    draft: 0,
    pending: 0,
    approved: 0,
    paid: 0,
    rejected: 0,
    failed: 0,
    withdrawn: 0,
  };
  for (const item of items) counts[item.status] += 1;
  return counts;
}

function sumAmount(items: RequestListItem[], status: RequestStatus): number {
  return items.filter((r) => r.status === status).reduce((sum, r) => sum + r.amount, 0);
}

function plural(n: number, one: string, many = `${one}s`) {
  return `${n} ${n === 1 ? one : many}`;
}

function Greeting({
  title,
  role,
  orgName,
  avgPending,
}: {
  title: string;
  role: Role;
  orgName: string;
  avgPending: number | null;
}) {
  return (
    <section className="flex flex-wrap items-end justify-between gap-4">
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <Pill className="bg-[#c7ef39] text-[11px] font-semibold tracking-[0.66px] text-[#556a00]">
            <span aria-hidden className="size-1.5 rounded-full bg-[#526600]" />
            {ROLES[role]}
          </Pill>
          <span className="text-[11px] font-semibold leading-4 tracking-[0.66px] text-[#444748]">{orgName}</span>
        </div>
        <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px] text-[#1b1c1a]`}>{title}</h1>
        <p className="max-w-[672px] text-[14px] leading-5 tracking-[-0.07px] text-[#444748]">
          Track your pending reimbursements, advances, and recent payouts across {orgName}.
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {avgPending !== null && (
          <div className="flex items-center gap-2 rounded-xl bg-[#f4f3f0] px-3.5 py-2 drop-shadow-[0px_1px_1px_rgba(0,0,0,0.05)]">
            <Icon name="bolt" size={18} className="text-[#c7ef39]" />
            <div>
              <p className="text-[10px] leading-4 tracking-[-0.07px] text-[#444748]">Avg. time in review</p>
              <p className={`${fontHeading} text-[13px] font-bold leading-[18px] tracking-[-0.07px] text-[#1b1c1a]`}>
                {avgPending.toFixed(1)} days
              </p>
            </div>
          </div>
        )}
        <Link
          href="/requests/new"
          className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-[#c7ef39] px-4 py-2.5 text-[13px] font-semibold leading-[18px] tracking-[-0.07px] text-[#556a00] drop-shadow-[0px_1px_1px_rgba(0,0,0,0.05)] transition-colors hover:bg-[#b5f546]`}
        >
          <Icon name="add" size={16} />
          Submit Request
        </Link>
      </div>
    </section>
  );
}

function KpiCard({
  label,
  labelClass = 'text-[#444748]',
  accent,
  icon,
  value,
  footer,
  footerHref,
  footerClass = 'bg-[rgba(244,243,240,0.5)]',
  arrowClass = 'text-[#444748]',
}: {
  label: ReactNode;
  labelClass?: string;
  accent: string;
  icon: ReactNode;
  value: ReactNode;
  footer: ReactNode;
  footerHref: string;
  footerClass?: string;
  arrowClass?: string;
}) {
  return (
    <div className={`relative flex flex-col overflow-hidden rounded-2xl bg-white ${cardShadow}`}>
      <div className={`absolute inset-x-0 top-0 h-1 ${accent}`} />
      <div className="flex flex-col gap-4 px-6 pb-4 pt-6">
        <div className="flex items-center justify-between">
          <h2 className={`flex items-center gap-2 text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] ${labelClass}`}>
            {label}
          </h2>
          {icon}
        </div>
        <div className="flex items-baseline gap-2">{value}</div>
      </div>
      <Link
        href={footerHref}
        className={`group mt-auto flex items-center justify-between gap-2 px-6 py-3 text-[13px] leading-[18px] tracking-[-0.07px] ${footerClass}`}
      >
        <span className="flex min-w-0 items-center gap-1.5 truncate">{footer}</span>
        <Icon name="arrow_forward" size={14} className={`transition-transform group-hover:translate-x-0.5 ${arrowClass}`} />
      </Link>
    </div>
  );
}

const bigNumber = `${fontHeading} text-[32px] font-bold leading-[38px] tracking-[-0.8px]`;

function KpiCards({ counts, approvedVolume }: { counts: Record<RequestStatus, number>; approvedVolume: string }) {
  const active = counts.draft + counts.pending;
  const attention = counts.rejected + counts.failed;

  return (
    <section aria-label="Key metrics" className="grid grid-cols-1 gap-3.5 md:grid-cols-3">
      <KpiCard
        label={
          <>
            Active requests
            <PulseDot className="bg-[#526600]" />
          </>
        }
        accent="bg-[#e3e2df]"
        icon={
          <span className="flex size-8 items-center justify-center rounded-full bg-[#efeeeb]">
            <Icon name="pending_actions" size={16} className="text-[#1b1c1a]" />
          </span>
        }
        value={
          <>
            <span className={`${bigNumber} text-[#1b1c1a]`}>{active}</span>
            <span className="text-[13px] font-medium leading-[18px] tracking-[-0.07px] text-[#444748]">in flight</span>
          </>
        }
        footerHref="/requests?status=pending"
        footer={
          <span className="text-[#444748]">
            {counts.pending} in finance review • {plural(counts.draft, 'draft')}
          </span>
        }
      />

      <KpiCard
        label="Approved for payout"
        accent="bg-[#c7ef39]"
        icon={
          <span className="flex size-8 items-center justify-center rounded-full bg-[rgba(199,239,57,0.2)]">
            <Icon name="verified" size={17} className="text-[#526600]" />
          </span>
        }
        value={
          <span className="flex items-baseline gap-2 text-[#556a00]">
            <span className={bigNumber}>{counts.approved}</span>
            <span className="text-[13px] font-semibold leading-[18px] tracking-[-0.07px]">{approvedVolume}</span>
          </span>
        }
        footerHref="/requests?status=approved"
        footer={
          <>
            <Icon name="calendar_today" size={13} className="text-[#444748]" />
            <span className="text-[#444748]">Awaiting payment from finance</span>
          </>
        }
      />

      <KpiCard
        label="Action needed"
        labelClass={attention > 0 ? 'text-[#ba1a1a]' : 'text-[#444748]'}
        accent={attention > 0 ? 'bg-[#ba1a1a]' : 'bg-[#e3e2df]'}
        icon={
          <span className={`flex size-8 items-center justify-center rounded-full ${attention > 0 ? 'bg-[rgba(255,218,214,0.3)]' : 'bg-[#efeeeb]'}`}>
            <Icon name="warning" size={17} className={`${attention > 0 ? 'text-[#ba1a1a]' : 'text-[#444748]'}`} />
          </span>
        }
        value={
          <span className={`flex items-baseline gap-2 ${attention > 0 ? 'text-[#ba1a1a]' : 'text-[#1b1c1a]'}`}>
            <span className={bigNumber}>{attention}</span>
            <span className="text-[13px] font-medium leading-[18px] tracking-[-0.07px]">
              {attention > 0 ? 'attention required' : 'all clear'}
            </span>
          </span>
        }
        footerHref="/requests?status=rejected"
        footerClass={attention > 0 ? 'bg-[rgba(255,218,214,0.1)]' : 'bg-[rgba(244,243,240,0.5)]'}
        arrowClass={attention > 0 ? 'text-[#93000a]' : 'text-[#444748]'}
        footer={
          attention > 0 ? (
            <span className="truncate font-medium text-[#93000a]">
              {plural(attention, 'request')} rejected or failed — review and resubmit
            </span>
          ) : (
            <span className="text-[#444748]">No rejected or failed requests</span>
          )
        }
      />
    </section>
  );
}

const PIPELINE: Array<Segment & { unit: string; highlight?: boolean }> = [
  { key: 'draft', label: 'Draft', value: 0, color: 'bg-[rgba(68,71,72,0.4)]', unit: 'not submitted' },
  { key: 'pending', label: 'In Review', value: 0, color: 'bg-black', unit: 'with finance' },
  { key: 'approved', label: 'Approved', value: 0, color: 'bg-[#c7ef39]', unit: 'queued', highlight: true },
  { key: 'paid', label: 'Paid', value: 0, color: 'bg-[#526600]', unit: 'disbursed' },
  { key: 'closed', label: 'Closed', value: 0, color: 'bg-[#e3e2df]', unit: 'rejected / withdrawn' },
];

function Pipeline({ counts, total, paidVolume }: { counts: Record<RequestStatus, number>; total: number; paidVolume: string }) {
  const values: Record<string, number> = {
    draft: counts.draft,
    pending: counts.pending,
    approved: counts.approved,
    paid: counts.paid,
    closed: counts.rejected + counts.failed + counts.withdrawn,
  };
  const segments = PIPELINE.map((s) => ({ ...s, value: values[s.key] }));

  return (
    <section
      aria-labelledby="pipeline-heading"
      className="flex flex-col gap-6 rounded-2xl bg-white p-6 drop-shadow-[0px_1px_1px_rgba(0,0,0,0.05)]"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span className="flex size-7 items-center justify-center rounded-lg bg-black">
            <Icon name="conversion_path" size={16} className="text-white" />
          </span>
          <div>
            <h2 id="pipeline-heading" className={`${fontHeading} text-[16px] font-bold leading-6 tracking-[-0.16px] text-[#1b1c1a]`}>
              My Submission Pipeline
            </h2>
            <p className="text-[13px] leading-[18px] tracking-[-0.07px] text-[#444748]">
              Where your {plural(total, 'request')} stand right now
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-full bg-[#f4f3f0] px-3 py-1.5">
          <Icon name="payments" size={14} className="text-[#444748]" />
          <span className="text-[11px] font-semibold leading-4 tracking-[0.66px] text-[#444748]">Total paid:</span>
          <span className="text-[13px] font-semibold leading-[18px] tracking-[-0.07px] text-[#1b1c1a]">{paidVolume}</span>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <SegmentBar segments={segments} trackClass="bg-[#e9e8e5]" gapClass="gap-0.5" />
        <ul className="grid grid-cols-2 gap-3 pt-2 sm:grid-cols-3 lg:grid-cols-5">
          {segments.map((seg) => (
            <li
              key={seg.key}
              className={`flex items-start gap-2.5 rounded-xl p-2 ${seg.highlight ? 'bg-[rgba(199,239,57,0.15)]' : 'bg-[rgba(244,243,240,0.4)]'}`}
            >
              <span aria-hidden className={`mt-1 size-2.5 shrink-0 rounded-full ${seg.color}`} />
              <div className={seg.highlight ? 'text-[#556a00]' : 'text-[#444748]'}>
                <p className="text-[11px] font-semibold leading-4 tracking-[0.66px]">{seg.label}</p>
                <p className="flex items-baseline gap-1.5">
                  <span className={`${fontHeading} text-[15px] font-bold leading-[22px] tracking-[-0.15px] ${seg.highlight ? '' : 'text-[#1b1c1a]'}`}>
                    {seg.value}
                  </span>
                  <span className="text-[11px] leading-5 tracking-[-0.07px]">{seg.unit}</span>
                </p>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

const URGENCY_STYLE: Record<Urgency, { pill: string; dot: string }> = {
  routine: { pill: 'bg-[#efeeeb] font-medium text-[#444748]', dot: '' },
  urgent: { pill: 'bg-[#e3e2df] font-medium text-[#1b1c1a]', dot: 'bg-[#f59e0b]' },
  critical: { pill: 'bg-[#ffdad6] font-semibold text-[#ba1a1a]', dot: 'bg-[#ba1a1a]' },
};

const STATUS_STYLE: Record<RequestStatus, { pill: string; icon: string; label: string }> = {
  draft: { pill: 'bg-[#efeeeb] font-medium text-[#444748]', icon: 'edit_note', label: 'Draft' },
  pending: { pill: 'bg-[#efeeeb] font-medium text-[#444748]', icon: 'hourglass_top', label: 'Pending' },
  approved: { pill: 'bg-[rgba(199,239,57,0.2)] font-semibold text-[#526600]', icon: 'check_circle', label: 'Approved' },
  paid: { pill: 'bg-[#efeeeb] font-semibold text-[#1b1c1a]', icon: 'payments', label: 'Paid' },
  rejected: { pill: 'bg-[rgba(186,26,26,0.1)] font-semibold text-[#ba1a1a]', icon: 'cancel', label: 'Rejected' },
  failed: { pill: 'bg-[rgba(186,26,26,0.1)] font-semibold text-[#ba1a1a]', icon: 'error', label: 'Payment failed' },
  withdrawn: { pill: 'bg-[#efeeeb] font-medium text-[#747878]', icon: 'undo', label: 'Withdrawn' },
};

function AgingNote({ item }: { item: RequestListItem }) {
  switch (item.status) {
    case 'pending':
      return item.days_pending >= 7 ? (
        <Pill className="bg-[rgba(255,218,214,0.2)] px-2.5 font-medium text-[#b45309]">
          <Icon name="schedule" size={12} />
          {plural(item.days_pending, 'day')} pending
        </Pill>
      ) : (
        <span className="text-[13px] text-[#444748]">{plural(item.days_pending, 'day')} in review</span>
      );
    case 'rejected':
    case 'failed':
      return (
        <span className="flex items-center gap-1.5 text-[13px] font-medium text-[#ba1a1a]">
          <Icon name="error" size={14} />
          {item.status === 'rejected' ? 'Rejected — see comments' : 'Payment failed'}
        </span>
      );
    case 'draft':
      return <span className="text-[13px] text-[#444748]">Not submitted yet</span>;
    case 'approved':
      return <span className="text-[13px] text-[#444748]">Awaiting payment</span>;
    case 'paid':
      return <span className="text-[13px] text-[#444748]">Settled</span>;
    default:
      return <span className="text-[13px] text-[#747878]">{REQUEST_STATUSES[item.status]}</span>;
  }
}

function RowAction({ item }: { item: RequestListItem }) {
  const href = `/requests/${item.id}`;
  const base = `${fontHeading} inline-flex items-center gap-1 whitespace-nowrap rounded-full px-3 py-1.5 text-[13px] leading-[18px] tracking-[-0.07px] transition-colors`;
  if (item.status === 'rejected' || item.status === 'failed') {
    return (
      <Link href={href} className={`${base} bg-[#ba1a1a] text-white hover:bg-[#93000a]`}>
        Resubmit
        <Icon name="redo" size={13} />
      </Link>
    );
  }
  return (
    <Link href={href} className={`${base} bg-[#efeeeb] text-[#1b1c1a] hover:bg-[#e3e2df]`}>
      {item.status === 'draft' ? 'Continue' : 'View'}
      <Icon name="arrow_forward" size={12} />
    </Link>
  );
}

function RecentRequests({
  items,
  counts,
  total,
  tab,
  onTabChange,
  currency,
  paidVolume,
}: {
  items: RequestListItem[];
  counts: Record<RequestStatus, number>;
  total: number;
  tab: Tab;
  onTabChange: (tab: Tab) => void;
  currency: string;
  paidVolume: string;
}) {
  const attention = counts.rejected + counts.failed;
  const tabs: Array<{ id: Tab; label: string; badge?: number }> = [
    { id: 'all', label: 'All' },
    { id: 'attention', label: 'Needs Attention', badge: attention },
    { id: 'pending', label: `Pending (${counts.pending})` },
    { id: 'approved', label: `Approved (${counts.approved})` },
  ];
  const filtered = items.filter((r) => {
    if (tab === 'attention') return NEEDS_ATTENTION.includes(r.status);
    if (tab === 'pending') return r.status === 'pending';
    if (tab === 'approved') return r.status === 'approved';
    return true;
  });
  const rows = filtered.slice(0, RECENT_ROWS);
  const th = 'px-3 py-3 text-[11px] font-semibold leading-4 tracking-[0.55px] text-[#444748]';

  return (
    <section aria-labelledby="recent-heading" className={`flex flex-col overflow-hidden rounded-2xl bg-white ${cardShadow}`}>
      <div className="flex flex-wrap items-center justify-between gap-3 px-6 pb-4 pt-6">
        <div className="flex items-center gap-3">
          <h2 id="recent-heading" className={`${fontHeading} text-[20px] font-bold leading-7 tracking-[-0.5px] text-[#1b1c1a]`}>
            Recent Requests
          </h2>
          <Pill className="bg-[#efeeeb] font-semibold tracking-[-0.07px] text-[#444748]">{total} total</Pill>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div role="tablist" aria-label="Filter requests" className="flex items-center rounded-full bg-[#f4f3f0] p-1">
            {tabs.map((t) => {
              const selected = tab === t.id;
              return (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  aria-selected={selected}
                  onClick={() => onTabChange(t.id)}
                  className={`${fontHeading} flex items-center gap-1.5 rounded-full px-3.5 py-1 text-[13px] leading-[18px] tracking-[-0.07px] transition-colors ${
                    selected ? 'bg-white text-[#1b1c1a] shadow-sm' : 'text-[#444748] hover:text-[#1b1c1a]'
                  }`}
                >
                  {t.label}
                  {!!t.badge && (
                    <span className="flex size-4 items-center justify-center rounded-full bg-[#ba1a1a] text-[10px] text-white">
                      {t.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
          <Link href="/requests" className={`${fontHeading} flex items-center gap-1 pl-2 text-[13px] leading-[18px] tracking-[-0.07px] text-[#1b1c1a] hover:underline`}>
            View all {plural(total, 'request')}
            <Icon name="arrow_forward" size={13} />
          </Link>
        </div>
      </div>

      {rows.length === 0 ? (
        <div className="flex flex-col items-center gap-2 border-t border-[#efeeeb] px-6 py-14 text-center">
          <Icon name={tab === 'all' ? 'receipt_long' : 'filter_alt_off'} size={26} className="text-[#747878]" />
          <p className="text-[14px] font-semibold text-[#1b1c1a]">
            {tab === 'all' ? 'No requests yet' : 'Nothing in this view'}
          </p>
          <p className="max-w-sm text-[13px] text-[#444748]">
            {tab === 'all'
              ? 'Submit a reimbursement, advance, or stipend and it will show up here.'
              : 'Try another filter, or view all of your requests.'}
          </p>
          {tab === 'all' && (
            <Link
              href="/requests/new"
              className={`${fontHeading} mt-2 flex items-center gap-1.5 rounded-full bg-[#0e0e0e] px-4 py-2 text-[13px] font-semibold text-white hover:bg-[#1c1b1b]`}
            >
              <Icon name="add" size={16} />
              New request
            </Link>
          )}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px] border-collapse text-left">
            <thead className="bg-[rgba(244,243,240,0.7)]">
              <tr>
                <th scope="col" className={`${th} pl-6`}>Request</th>
                <th scope="col" className={th}>Date</th>
                <th scope="col" className={th}>Purpose</th>
                <th scope="col" className={`${th} text-right`}>Amount</th>
                <th scope="col" className={`${th} text-center`}>Urgency</th>
                <th scope="col" className={th}>Status</th>
                <th scope="col" className={th}>Aging &amp; Notes</th>
                <th scope="col" className={`${th} pr-6 text-right`}>Action</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => {
                const flagged = NEEDS_ATTENTION.includes(item.status);
                const urgency = URGENCY_STYLE[item.urgency];
                const status = STATUS_STYLE[item.status];
                return (
                  <tr key={item.id} className={`border-t border-[#efeeeb] ${flagged ? 'bg-[rgba(255,218,214,0.05)]' : ''}`}>
                    <td className="py-4 pl-6 pr-3">
                      <span className={`flex items-center gap-2 whitespace-nowrap text-[13px] font-semibold leading-[18px] ${flagged ? 'text-[#ba1a1a]' : 'text-[#1b1c1a]'}`}>
                        <Icon name={flagged ? 'priority_high' : 'receipt_long'} size={15} />
                        {item.id}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-[13px] font-medium leading-[18px] text-[#444748]">
                      {formatDateOnly(item.created_at)}
                    </td>
                    <td className="w-[190px] min-w-[150px] max-w-[190px] px-3 py-4">
                      <p className={`line-clamp-2 text-[13px] leading-[18px] tracking-[-0.07px] ${flagged ? 'font-medium text-[#ba1a1a]' : 'text-[#1b1c1a]'}`} title={item.purpose}>
                        {item.purpose}
                      </p>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-right text-[13px] font-semibold leading-[18px] text-[#1b1c1a]">
                      {formatMoney(item.amount, currency)}
                    </td>
                    <td className="px-3 py-4 text-center">
                      <Pill className={`py-1 tracking-[-0.07px] ${urgency.pill}`}>
                        {urgency.dot && <span aria-hidden className={`size-1.5 rounded-full ${urgency.dot}`} />}
                        {URGENCIES[item.urgency]}
                      </Pill>
                    </td>
                    <td className="px-3 py-4">
                      <Pill className={`py-1 tracking-[-0.07px] ${status.pill}`}>
                        <Icon name={status.icon} size={13} />
                        {status.label}
                      </Pill>
                    </td>
                    <td className="px-3 py-4">
                      <AgingNote item={item} />
                    </td>
                    <td className="py-4 pl-3 pr-6 text-right">
                      <RowAction item={item} />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2 bg-[rgba(244,243,240,0.4)] px-6 py-4 text-[13px] leading-[18px] tracking-[-0.07px] text-[#444748]">
        <p>
          Showing {rows.length} of {filtered.length} {tab === 'all' ? 'recent' : 'matching'} {filtered.length === 1 ? 'request' : 'requests'}
        </p>
        <p className="flex items-center gap-2">
          <span aria-hidden className="size-2 rounded-full bg-[#c7ef39]" />
          Paid out to you so far: {paidVolume}
        </p>
      </div>
    </section>
  );
}
