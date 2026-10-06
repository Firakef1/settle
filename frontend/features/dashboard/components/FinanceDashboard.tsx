'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import type { DashboardSummary, OrgStats, Role } from '../../../shared/types';
import { PLANS, ROLES } from '../../../shared/utils/constants';
import { formatMoney } from '../../../shared/utils/format';
import { useDashboardSummary, useOrgStats } from '../api';
import {
  DashboardSkeleton,
  Dot,
  ErrorState,
  Icon,
  Pill,
  PulseDot,
  SegmentBar,
  errorMessage,
  fontHeading,
  greeting,
  percent,
  updatedAgo,
  type Segment,
} from './ui';

const ESCALATED_ROWS = 5;
const cardShadow = 'shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';

interface FinanceDashboardProps {
  orgId: string;
  orgName: string;
  userName?: string;
  role: Role;
}

export function FinanceDashboard({ orgId, orgName, userName, role }: FinanceDashboardProps) {
  const summary = useDashboardSummary(orgId);
  const stats = useOrgStats(orgId);

  if (summary.isPending || stats.isPending) return <DashboardSkeleton />;
  if (summary.isError || stats.isError) {
    return (
      <ErrorState
        message={errorMessage(summary.error ?? stats.error)}
        onRetry={() => {
          summary.refetch();
          stats.refetch();
        }}
      />
    );
  }

  const s = summary.data;
  const st = stats.data;
  const currency = st.financials.currency;
  const escalatedVolume = s.escalated_items.reduce((sum, item) => sum + item.amount, 0);
  const atRisk = s.aging_breakdown['7_plus_days'] ?? 0;

  return (
    <div className="flex flex-col gap-7">
      <Greeting
        title={greeting(userName)}
        role={role}
        orgName={orgName}
        attentionCount={s.escalated_items.length}
        attentionVolume={formatMoney(escalatedVolume, currency)}
      />
      <KpiCards summary={s} stats={st} atRisk={atRisk} escalatedVolume={escalatedVolume} />
      <Lifecycle stats={st} />
      <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,7fr)_minmax(0,3fr)]">
        <EscalatedTable summary={s} currency={currency} escalatedVolume={escalatedVolume} />
        <QueueBreakdown summary={s} atRisk={atRisk} updatedAt={summary.dataUpdatedAt} />
      </div>
      <OrgOverview summary={s} stats={st} />
    </div>
  );
}

function Greeting({
  title,
  role,
  orgName,
  attentionCount,
  attentionVolume,
}: {
  title: string;
  role: Role;
  orgName: string;
  attentionCount: number;
  attentionVolume: string;
}) {
  return (
    <section className="flex flex-wrap items-center justify-between gap-4">
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px] text-[#1b1c1a]`}>{title}</h1>
          <Pill className="bg-[#c7ef39] font-semibold text-[#171e00]">
            <span aria-hidden className="size-1.5 rounded-full bg-[#171e00]" />
            {ROLES[role]}
          </Pill>
        </div>
        <p className="text-[14px] leading-5 tracking-[-0.07px] text-[#444748]">
          Approval queue, spend, and request aging across {orgName}.
        </p>
      </div>
      {attentionCount > 0 && (
        <Link
          href="/approvals"
          className="flex items-center gap-2 rounded-full bg-[#f4f3f0] px-3.5 py-2 text-[12px] leading-4 drop-shadow-[0px_1px_1px_rgba(0,0,0,0.05)] transition-colors hover:bg-[#efeeeb]"
        >
          <PulseDot className="bg-[#ba1a1a]" />
          <span className="font-semibold text-[#1b1c1a]">
            {attentionCount} {attentionCount === 1 ? 'request needs' : 'requests need'} attention
          </span>
          <span className="font-medium text-[#747878]">·</span>
          <span className="font-bold text-[#ba1a1a]">{attentionVolume} volume</span>
        </Link>
      )}
    </section>
  );
}

function KpiCard({
  label,
  icon,
  accent = 'bg-[#efeeeb]',
  danger = false,
  children,
  footer,
}: {
  label: string;
  icon: ReactNode;
  accent?: string;
  danger?: boolean;
  children: ReactNode;
  footer: ReactNode;
}) {
  return (
    <div className={`relative flex flex-col justify-between overflow-hidden rounded-xl bg-white p-5 ${cardShadow}`}>
      <div className={`absolute inset-x-0 top-0 h-1 ${accent}`} />
      <div className="flex items-center justify-between">
        <h2
          className={`text-[11px] uppercase leading-4 tracking-[0.55px] ${danger ? 'font-bold text-[#ba1a1a]' : 'font-semibold text-[#444748]'}`}
        >
          {label}
        </h2>
        {icon}
      </div>
      <div className="flex flex-wrap items-baseline gap-x-2 py-3.5">{children}</div>
      <div className="flex items-center justify-between gap-2 pt-1 text-[12px] leading-4">{footer}</div>
    </div>
  );
}

const bigNumber = `${fontHeading} text-[32px] font-bold leading-[38px] tracking-[-0.96px]`;

function KpiCards({
  summary,
  stats,
  atRisk,
  escalatedVolume,
}: {
  summary: DashboardSummary;
  stats: OrgStats;
  atRisk: number;
  escalatedVolume: number;
}) {
  const currency = stats.financials.currency;
  const { routine, urgent, critical } = summary.urgency_breakdown;
  const paidThisMonth = formatMoney(stats.financials.this_month_spent, currency);
  const decimalAt = paidThisMonth.lastIndexOf('.');
  const paidWhole = decimalAt > 0 ? paidThisMonth.slice(0, decimalAt) : paidThisMonth;
  const paidCents = decimalAt > 0 ? paidThisMonth.slice(decimalAt) : '';
  const hasRisk = atRisk > 0;

  return (
    <section aria-label="Key metrics" className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-4">
      <KpiCard
        label="Pending queue"
        icon={<Icon name="schedule" size={20} className="text-[#444748]" />}
        footer={
          <>
            <span className="font-medium text-[#444748]">
              {routine} routine · {urgent + critical} urgent/critical
            </span>
            {critical > 0 && <span className="font-bold text-[#ba1a1a]">{critical} critical</span>}
          </>
        }
      >
        <span className={`${bigNumber} text-[#1b1c1a]`}>{summary.pending_count}</span>
        <span className="text-[13px] font-medium leading-[18px] text-[#444748]">awaiting decision</span>
      </KpiCard>

      <KpiCard
        label="Approved"
        icon={<Icon name="check_circle" size={21} className="text-[#526600]" />}
        footer={
          <>
            <span className="font-medium text-[#444748]">Awaiting payment</span>
            <Link
              href="/approvals"
              className="whitespace-nowrap rounded-full bg-[#efeeeb] px-2 py-0.5 font-medium text-[#1b1c1a] transition-colors hover:bg-[#e3e2df]"
            >
              Record payouts
            </Link>
          </>
        }
      >
        <span className={`${bigNumber} text-[#1b1c1a]`}>{stats.request_stats.approved}</span>
        <span className="text-[13px] font-medium leading-[18px] text-[#444748]">ready to pay</span>
      </KpiCard>

      <KpiCard
        label="Paid this month"
        icon={<Icon name="account_balance" size={20} className="text-[#444748]" />}
        footer={
          <>
            <span className="font-medium text-[#444748]">{stats.request_stats.paid} settled requests</span>
            <span className="text-right font-bold text-[#526600]">
              {formatMoney(stats.financials.total_spent, currency)} all time
            </span>
          </>
        }
      >
        <span className={`${fontHeading} text-[32px] font-bold leading-[38px] tracking-[-0.8px] text-[#1b1c1a]`}>
          {paidWhole}
        </span>
        <span className={`${fontHeading} -ml-1 text-[16px] font-semibold leading-6 tracking-[-0.16px] text-[#444748]`}>
          {paidCents}
        </span>
      </KpiCard>

      <KpiCard
        label="At risk (7+ days)"
        danger={hasRisk}
        accent={hasRisk ? 'bg-[#ba1a1a]' : 'bg-[#efeeeb]'}
        icon={hasRisk ? <PulseDot className="bg-[#ba1a1a]" /> : <Dot className="bg-[#c4c7c7]" />}
        footer={
          hasRisk ? (
            <Link href="/approvals" className="group flex w-full items-center justify-between">
              <Pill className="bg-[#ffdad6] px-2 font-bold text-[#93000a]">Requires triage</Pill>
              <Icon name="arrow_forward" size={15} className="text-[#ba1a1a] transition-transform group-hover:translate-x-0.5" />
            </Link>
          ) : (
            <span className="font-medium text-[#444748]">Nothing pending past 7 days</span>
          )
        }
      >
        <span className={`${bigNumber} ${hasRisk ? 'text-[#ba1a1a]' : 'text-[#1b1c1a]'}`}>{atRisk}</span>
        {hasRisk && (
          <span className="text-[13px] font-medium leading-[18px] text-[#ba1a1a]">
            {formatMoney(escalatedVolume, stats.financials.currency)} at risk
          </span>
        )}
      </KpiCard>
    </section>
  );
}

function Lifecycle({ stats }: { stats: OrgStats }) {
  const r = stats.request_stats;
  const segments: Segment[] = [
    { key: 'pending', label: 'In review', value: r.pending, color: 'bg-black' },
    { key: 'approved', label: 'Approved', value: r.approved, color: 'bg-[#c7ef39]' },
    { key: 'paid', label: 'Paid', value: r.paid, color: 'bg-[#1b1c1a]' },
    { key: 'closed', label: 'Rejected / failed', value: r.rejected + r.failed, color: 'bg-[#c4c7c7]' },
    { key: 'withdrawn', label: 'Withdrawn', value: r.withdrawn, color: 'bg-[#dbdad7]' },
  ];

  return (
    <section
      aria-labelledby="lifecycle-heading"
      className="flex flex-col gap-3.5 rounded-xl bg-white p-5 drop-shadow-[0px_1px_1.5px_rgba(0,0,0,0.04)]"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h2 id="lifecycle-heading" className={`${fontHeading} text-[16px] font-bold leading-6 tracking-[-0.16px] text-[#1b1c1a]`}>
            Request Lifecycle
          </h2>
          <Pill className="bg-[#efeeeb] px-2 font-medium text-[#444748]">{r.total} submitted</Pill>
        </div>
        <p className="flex items-center gap-1 text-[11px] font-semibold leading-4 tracking-[0.55px] text-[#444748]">
          <Icon name="speed" size={13} />
          Members: <span className="text-[#1b1c1a]">{stats.member_counts.total}</span> · Plan:{' '}
          <span className="text-[#1b1c1a]">{PLANS[stats.plan_usage.current_plan] ?? stats.plan_usage.current_plan}</span>
        </p>
      </div>
      <div className="flex flex-col gap-1">
        <SegmentBar segments={segments} trackClass="bg-[#efeeeb]" gapClass="gap-1" />
        <ul className="grid grid-cols-2 gap-2 pt-1 sm:grid-cols-3 lg:grid-cols-5">
          {segments.map((seg) => (
            <li key={seg.key} className="flex items-center gap-1.5 text-[13px] leading-[18px] text-[#444748]">
              <span aria-hidden className={`size-2.5 shrink-0 rounded-full ${seg.color}`} />
              {seg.label} <span className="font-semibold text-[#1b1c1a]">({seg.value})</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function riskLevel(days: number) {
  return days >= 14
    ? { label: 'Critical (14+d)', pill: 'bg-[#ffdad6] font-bold text-[#93000a]', days: 'bg-[#ba1a1a] text-white', dot: 'bg-[#ba1a1a]' }
    : { label: 'High (7-13d)', pill: 'bg-[#efeeeb] font-semibold text-[#1b1c1a]', days: 'bg-[#e3e2df] text-[#1b1c1a]', dot: 'bg-[#526600]' };
}

const URGENCY_LABEL = { routine: 'Routine', urgent: 'Urgent', critical: 'Critical' } as const;

function EscalatedTable({
  summary,
  currency,
  escalatedVolume,
}: {
  summary: DashboardSummary;
  currency: string;
  escalatedVolume: number;
}) {
  const items = summary.escalated_items;
  const rows = items.slice(0, ESCALATED_ROWS);
  const th = 'px-3.5 py-2 text-[11px] font-bold leading-4 tracking-[0.55px] text-[#444748]';

  return (
    <section aria-labelledby="escalated-heading" className={`flex flex-col overflow-hidden rounded-xl bg-white ${cardShadow}`}>
      <div className="flex flex-wrap items-center justify-between gap-2 px-5 py-3.5">
        <div className="flex items-center gap-2">
          <Icon name="warning" size={15} className="text-[#ba1a1a]" />
          <h2 id="escalated-heading" className={`${fontHeading} text-[16px] font-bold leading-6 tracking-[-0.16px] text-[#1b1c1a]`}>
            Needs Attention (Aging &gt; 7 Days)
          </h2>
          {items.length > 0 && <Pill className="bg-[#ffdad6] px-2 font-bold text-[#93000a]">{items.length} flagged</Pill>}
        </div>
        <Link href="/approvals" className="flex items-center gap-1 text-[13px] font-medium leading-[18px] text-[#1b1c1a] hover:underline">
          View full approval queue
          <Icon name="arrow_forward" size={13} />
        </Link>
      </div>

      {rows.length === 0 ? (
        <div className="flex flex-col items-center gap-2 border-t border-[#efeeeb] px-5 py-12 text-center">
          <Icon name="task_alt" size={24} className="text-[#526600]" />
          <p className="text-[13px] font-semibold text-[#1b1c1a]">No requests pending longer than 7 days</p>
          <p className="text-[12px] text-[#444748]">Escalated requests will show up here, oldest first.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] border-collapse text-left">
            <thead className="bg-[#f4f3f0]">
              <tr>
                <th scope="col" className={`${th} pl-5`}>Request ID</th>
                <th scope="col" className={th}>Requester &amp; Urgency</th>
                <th scope="col" className={`${th} text-right`}>Amount</th>
                <th scope="col" className={`${th} text-center`}>Days<br />Pending</th>
                <th scope="col" className={th}>Risk Level</th>
                <th scope="col" className={`${th} pr-5 text-right`}>Quick<br />Action</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => {
                const risk = riskLevel(item.days_pending);
                return (
                  <tr key={item.id} className="border-t border-[#efeeeb]">
                    <td className="py-4 pl-5 pr-3.5">
                      <span className="flex items-center gap-2 whitespace-nowrap text-[13px] font-bold leading-[18px] tracking-[-0.325px] text-[#1b1c1a]">
                        <Dot className={risk.dot} />
                        {item.id}
                      </span>
                    </td>
                    <td className="max-w-[320px] px-3.5 py-4">
                      <p className="truncate text-[13px] font-semibold leading-[18px] text-[#1b1c1a]">{item.requester_name}</p>
                      <p className="truncate text-[13px] leading-[18px] text-[#444748]">{URGENCY_LABEL[item.urgency]} urgency</p>
                    </td>
                    <td className="whitespace-nowrap px-3.5 py-4 text-right font-mono text-[13px] font-bold leading-[18px] text-[#1b1c1a]">
                      {formatMoney(item.amount, currency)}
                    </td>
                    <td className="px-3.5 py-4 text-center">
                      <Pill className={`px-2.5 font-bold ${risk.days}`}>{item.days_pending}d</Pill>
                    </td>
                    <td className="px-3.5 py-4">
                      <Pill className={risk.pill}>{risk.label}</Pill>
                    </td>
                    <td className="py-4 pl-3.5 pr-5 text-right">
                      <Link
                        href={`/requests/${item.id}`}
                        className={`${fontHeading} inline-flex items-center rounded-full bg-black px-3.5 py-1.5 text-[13px] font-semibold leading-[18px] text-white drop-shadow-[0px_1px_1px_rgba(0,0,0,0.05)] transition-colors hover:bg-[#1c1b1b]`}
                      >
                        Review
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {items.length > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-2 bg-[#f4f3f0] px-5 py-2">
          <p className="text-[13px] leading-[18px] text-[#444748]">
            Showing {rows.length} of {items.length} escalated {items.length === 1 ? 'request' : 'requests'}
          </p>
          <p className="flex items-center gap-2">
            <span className="text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#444748]">
              Total at-risk exposure:
            </span>
            <span className="text-[13px] font-bold leading-[18px] text-[#ba1a1a]">{formatMoney(escalatedVolume, currency)}</span>
          </p>
        </div>
      )}
    </section>
  );
}

function QueueBreakdown({ summary, atRisk, updatedAt }: { summary: DashboardSummary; atRisk: number; updatedAt: number }) {
  const { routine, urgent, critical } = summary.urgency_breakdown;
  const rows = [
    { key: 'critical', count: critical, label: 'critical', detail: 'Highest urgency — review first', dot: 'bg-[#ba1a1a]' },
    { key: 'urgent', count: urgent, label: 'urgent', detail: 'Flagged urgent by the requester', dot: 'bg-[#c7ef39]' },
    { key: 'routine', count: routine, label: 'routine', detail: 'Standard review cycle', dot: 'bg-[#dbdad7]' },
  ];
  const withinWeek = summary.pending_count - atRisk;
  const onTimeRate = summary.pending_count > 0 ? percent(withinWeek, summary.pending_count) : 100;

  return (
    <section
      aria-labelledby="breakdown-heading"
      className="relative flex flex-col gap-4 overflow-hidden rounded-2xl bg-[#1b1c1c] p-5 shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1),0px_8px_10px_-6px_rgba(0,0,0,0.1)]"
    >
      <div aria-hidden className="pointer-events-none absolute -right-12 -top-12 size-40 rounded-full bg-[rgba(199,239,57,0.1)] blur-[32px]" />

      <div className="relative flex flex-col gap-1">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <Icon name="bolt" size={16} className="font-semibold text-[#c7ef39]" />
            <h2 id="breakdown-heading" className={`${fontHeading} text-[16px] font-bold leading-6 tracking-[-0.4px] text-white`}>
              Queue Breakdown
            </h2>
          </div>
          <Pill className="bg-[rgba(227,226,223,0.1)] px-2 font-medium text-[#848483]">{updatedAgo(updatedAt)}</Pill>
        </div>
        <p className={`${fontHeading} pt-2 text-[22px] font-bold leading-[27.5px] text-white`}>
          {atRisk > 0
            ? `${atRisk} ${atRisk === 1 ? 'request' : 'requests'} pending over 7 days`
            : `${summary.pending_count} ${summary.pending_count === 1 ? 'request' : 'requests'} awaiting decision`}
        </p>
      </div>

      <ul className="relative flex flex-col gap-2" aria-label="Pending requests by urgency">
        {rows.map((row) => (
          <li key={row.key} className="flex items-start gap-2 rounded-lg bg-[rgba(227,226,223,0.05)] p-2">
            <span aria-hidden className={`mt-1.5 size-2 shrink-0 rounded-full ${row.dot}`} />
            <div>
              <p className="text-[13px] font-semibold leading-[18px] text-white">
                {row.count} {row.label}
              </p>
              <p className="text-[12px] font-medium leading-4 text-[#848483]">{row.detail}</p>
            </div>
          </li>
        ))}
      </ul>

      <Link
        href="/approvals"
        className={`${fontHeading} relative flex w-full items-center justify-center gap-1 rounded-full bg-[#c7ef39] px-3.5 py-2 text-[16px] font-bold leading-6 tracking-[-0.16px] text-[#171e00] shadow-[0px_4px_6px_-1px_rgba(0,0,0,0.1),0px_2px_4px_-2px_rgba(0,0,0,0.1)] transition-colors hover:bg-[#b5f546]`}
      >
        Review queue
        <Icon name="arrow_forward" size={15} />
      </Link>

      <div className="relative flex items-center justify-between border-t border-[rgba(227,226,223,0.1)] pt-4">
        <div>
          <p className="text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#848483]">Within 7 days</p>
          <p className={`${fontHeading} pt-0.5 text-[20px] font-bold leading-7 tracking-[-0.4px] text-white`}>
            {onTimeRate.toFixed(1)}%
          </p>
          <p className={`text-[12px] font-semibold leading-4 ${atRisk > 0 ? 'text-[#ffb4ab]' : 'text-[#c7ef39]'}`}>
            {withinWeek} of {summary.pending_count} pending
          </p>
        </div>
        <Gauge value={onTimeRate} />
      </div>
    </section>
  );
}

// Ring sizes match the Figma arc gauge (64px box, ~6.2px stroke).
function Gauge({ value }: { value: number }) {
  const r = 28.3;
  const circumference = 2 * Math.PI * r;
  const offset = circumference * (1 - Math.min(Math.max(value, 0), 100) / 100);
  return (
    <div className="relative size-16 shrink-0">
      <svg width="64" height="64" viewBox="0 0 64 64" className="-rotate-90" aria-hidden>
        <circle cx="32" cy="32" r={r} fill="none" stroke="#E3E2DF" strokeOpacity="0.2" strokeWidth="6.22" />
        <circle
          cx="32"
          cy="32"
          r={r}
          fill="none"
          stroke="#C7EF39"
          strokeWidth="6.22"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
        />
      </svg>
      <Icon name="verified" size={14} className="absolute inset-0 m-auto text-[#c7ef39]" />
    </div>
  );
}

function BarRow({ label, value, max, color, suffix }: { label: string; value: number; max: number; color: string; suffix?: string }) {
  const width = max > 0 ? Math.min(percent(value, max), 100) : 0;
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-[12px] leading-4">
        <span className="font-medium text-[#444748]">{label}</span>
        <span className="font-semibold text-[#1b1c1a]">
          {value}
          {suffix}
        </span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-[#efeeeb]">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}

function OverviewCard({ title, icon, children }: { title: string; icon: string; children: ReactNode }) {
  return (
    <div className={`flex flex-col gap-4 rounded-xl bg-white p-5 ${cardShadow}`}>
      <div className="flex items-center justify-between">
        <h3 className="text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#444748]">{title}</h3>
        <Icon name={icon} size={20} className="text-[#444748]" />
      </div>
      {children}
    </div>
  );
}

function OrgOverview({ summary, stats }: { summary: DashboardSummary; stats: OrgStats }) {
  const aging = summary.aging_breakdown;
  const agingMax = Math.max(summary.pending_count, 1);
  const members = stats.member_counts;
  const usage = stats.plan_usage;
  const limitLabel = (limit: number) => (limit === 0 ? ' / unlimited' : ` / ${limit}`);
  const overLimit = (count: number, limit: number) => limit > 0 && count >= limit;

  return (
    <section aria-label="Organization overview" className="grid grid-cols-1 gap-5 md:grid-cols-3">
      <OverviewCard title="Pending by age" icon="hourglass_top">
        <BarRow label="0–3 days" value={aging['0_to_3_days'] ?? 0} max={agingMax} color="bg-[#c7ef39]" />
        <BarRow label="3–7 days" value={aging['3_to_7_days'] ?? 0} max={agingMax} color="bg-[#1b1c1a]" />
        <BarRow label="7+ days" value={aging['7_plus_days'] ?? 0} max={agingMax} color="bg-[#ba1a1a]" />
      </OverviewCard>

      <OverviewCard title="Members" icon="group">
        <p className={`${fontHeading} -mt-2 text-[24px] font-bold leading-8 text-[#1b1c1a]`}>{members.total}</p>
        <BarRow label="Admins" value={members.org_admin} max={members.total} color="bg-[#1b1c1a]" />
        <BarRow label="Finance" value={members.finance} max={members.total} color="bg-[#526600]" />
        <BarRow label="Staff" value={members.staff} max={members.total} color="bg-[#c7ef39]" />
      </OverviewCard>

      <OverviewCard title={`${PLANS[usage.current_plan] ?? usage.current_plan} plan usage`} icon="data_usage">
        <BarRow
          label="Requests this month"
          value={usage.requests_this_month}
          max={usage.request_limit || usage.requests_this_month}
          color={overLimit(usage.requests_this_month, usage.request_limit) ? 'bg-[#ba1a1a]' : 'bg-[#1b1c1a]'}
          suffix={limitLabel(usage.request_limit)}
        />
        <BarRow
          label="Users"
          value={usage.user_count}
          max={usage.user_limit || usage.user_count}
          color={overLimit(usage.user_count, usage.user_limit) ? 'bg-[#ba1a1a]' : 'bg-[#1b1c1a]'}
          suffix={limitLabel(usage.user_limit)}
        />
        {(overLimit(usage.requests_this_month, usage.request_limit) || overLimit(usage.user_count, usage.user_limit)) && (
          <p className="flex items-center gap-1 text-[12px] font-medium leading-4 text-[#ba1a1a]">
            <Icon name="error" size={14} />
            Plan limit reached — consider upgrading in Settings.
          </p>
        )}
      </OverviewCard>
    </section>
  );
}
