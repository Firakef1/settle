'use client';

import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useState, type ReactNode } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { RequestListItem, RequestStatus, Role, Urgency } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { formatDateOnly, formatMoney } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useRequestCount, useRequestList } from '../hooks/useRequests';
import { STATUS, StatusPill, UrgencyPill } from './pills';

const PAGE_SIZE = 20;
const STATUSES = Object.keys(STATUS) as RequestStatus[];
const URGENCIES: Urgency[] = ['routine', 'urgent', 'critical'];
const cardShadow = 'shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const selectClass =
  'h-9 rounded-full border border-[#e3e2df] bg-white px-3.5 text-[13px] text-[#1b1c1a] outline-none focus:ring-2 focus:ring-[#c7ef39]/60';

export function RequestsList() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const { currentOrg } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const role = (currentOrg?.role ?? 'staff') as Role;
  const isFinance = role === 'finance';
  const currency = useOrganization(orgId).data?.currency ?? 'USD';

  const status = (STATUSES.includes(params.get('status') as RequestStatus) ? params.get('status') : undefined) as
    | RequestStatus
    | undefined;
  const urgency = (URGENCIES.includes(params.get('urgency') as Urgency) ? params.get('urgency') : undefined) as
    | Urgency
    | undefined;
  const order = params.get('order') === 'asc' ? 'asc' : 'desc';
  const page = Math.max(1, Number(params.get('page')) || 1);
  const [search, setSearch] = useState('');

  const list = useRequestList(orgId, {
    status,
    urgency,
    sort_by: 'created_at',
    sort_order: order,
    limit: PAGE_SIZE,
    offset: (page - 1) * PAGE_SIZE,
  });
  const pendingCount = useRequestCount(orgId, 'pending');
  const approvedCount = useRequestCount(orgId, 'approved');
  const paidCount = useRequestCount(orgId, 'paid');

  const setParam = (updates: Record<string, string | undefined>) => {
    const next = new URLSearchParams(params.toString());
    for (const [key, value] of Object.entries(updates)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    if (!('page' in updates)) next.delete('page');
    const qs = next.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  };

  const rows = (list.data?.data ?? []).filter((r) => {
    const q = search.trim().toLowerCase();
    return !q || r.id.toLowerCase().includes(q) || r.purpose.toLowerCase().includes(q);
  });
  const total = list.data?.meta.total ?? 0;
  const from = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1;
  const to = Math.min(page * PAGE_SIZE, total);
  const filtered = !!(status || urgency);

  return (
    <div className="flex flex-col gap-7">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>
            {isFinance ? 'All Requests' : 'My Requests'}
          </h1>
          <p className="text-[14px] leading-5 text-[#444748]">
            {isFinance
              ? `Every submitted payout request in ${currentOrg?.org_name}.`
              : 'Reimbursements, advances and stipends you have raised.'}
          </p>
        </div>
        <Link
          href="/requests/new"
          className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-[#c7ef39] px-4 py-2.5 text-[13px] font-semibold leading-[18px] text-[#171e00] transition-colors hover:bg-[#b5f546]`}
        >
          <Icon name="add" size={16} />
          New request
        </Link>
      </section>

      <section aria-label="Totals" className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard
          label="Pending review"
          icon="hourglass_top"
          value={pendingCount.data}
          hint="Waiting for a finance decision"
          href="/requests?status=pending"
        />
        <StatCard
          label="Approved"
          icon="check_circle"
          value={approvedCount.data}
          hint="Approved, awaiting payment"
          href="/requests?status=approved"
          accent
        />
        <StatCard label="Paid" icon="payments" value={paidCount.data} hint="Settled payouts" href="/requests?status=paid" dark />
      </section>

      <section aria-label="Requests" className={`flex flex-col overflow-hidden rounded-2xl bg-white ${cardShadow}`}>
        <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
          <div className="relative min-w-[220px] flex-1 sm:max-w-sm">
            <Icon name="search" size={15} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[#444748]" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search this page by ID or purpose…"
              aria-label="Search by ID or purpose"
              className="h-9 w-full rounded-full bg-[#f4f3f0] pl-9 pr-3 text-[13px] outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              aria-label="Status"
              value={status ?? ''}
              onChange={(e) => setParam({ status: e.target.value || undefined })}
              className={selectClass}
            >
              <option value="">All statuses</option>
              {STATUSES.filter((s) => !(isFinance && s === 'draft')).map((s) => (
                <option key={s} value={s}>
                  {STATUS[s].label}
                </option>
              ))}
            </select>
            <select
              aria-label="Urgency"
              value={urgency ?? ''}
              onChange={(e) => setParam({ urgency: e.target.value || undefined })}
              className={selectClass}
            >
              <option value="">All urgencies</option>
              {URGENCIES.map((u) => (
                <option key={u} value={u}>
                  {u[0].toUpperCase() + u.slice(1)}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={() => setParam({ order: order === 'desc' ? 'asc' : undefined })}
              className={`${selectClass} flex items-center gap-1`}
              aria-label={`Sorted ${order === 'desc' ? 'newest' : 'oldest'} first`}
            >
              <Icon name={order === 'desc' ? 'arrow_downward' : 'arrow_upward'} size={14} />
              {order === 'desc' ? 'Newest first' : 'Oldest first'}
            </button>
            {filtered && (
              <button
                type="button"
                onClick={() => setParam({ status: undefined, urgency: undefined })}
                className="h-9 rounded-full px-3 text-[13px] font-medium text-[#444748] hover:bg-[#efeeeb]"
              >
                Clear
              </button>
            )}
          </div>
        </div>

        {list.isPending ? (
          <div className="flex flex-col gap-2 border-t border-[#efeeeb] p-5" aria-busy="true">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="h-12 animate-pulse rounded-xl bg-[#f4f3f0]" />
            ))}
          </div>
        ) : list.isError ? (
          <Empty
            icon="error"
            title="Couldn't load requests"
            body={apiErrorMessage(list.error, 'Check your connection and try again.')}
            action={
              <button type="button" onClick={() => list.refetch()} className="rounded-full bg-[#efeeeb] px-4 py-2 text-[13px] font-semibold">
                Try again
              </button>
            }
          />
        ) : rows.length === 0 ? (
          <Empty
            icon={filtered || search ? 'filter_alt_off' : 'receipt_long'}
            title={filtered || search ? 'No requests match' : 'No requests yet'}
            body={
              filtered || search
                ? 'Try a different filter or search.'
                : 'Raise a reimbursement, advance or stipend and it will show up here.'
            }
            action={
              !filtered && !search ? (
                <Link href="/requests/new" className={`${fontHeading} rounded-full bg-[#0e0e0e] px-4 py-2 text-[13px] font-semibold text-white`}>
                  New request
                </Link>
              ) : undefined
            }
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] border-collapse text-left">
              <thead className="bg-[rgba(244,243,240,0.7)]">
                <tr className="text-[11px] font-semibold leading-4 tracking-[0.55px] text-[#444748]">
                  <th scope="col" className="py-3 pl-5 pr-3">Request</th>
                  {isFinance && <th scope="col" className="px-3 py-3">Requester</th>}
                  <th scope="col" className="px-3 py-3">Created</th>
                  <th scope="col" className="px-3 py-3 text-right">Amount</th>
                  <th scope="col" className="px-3 py-3">Urgency</th>
                  <th scope="col" className="px-3 py-3">Status</th>
                  <th scope="col" className="px-3 py-3">Aging</th>
                  <th scope="col" className="py-3 pl-3 pr-5 text-right">
                    <span className="sr-only">Open</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <Row key={r.id} item={r} currency={currency} showRequester={isFinance} onOpen={() => router.push(`/requests/${r.id}`)} />
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#efeeeb] bg-[rgba(244,243,240,0.4)] px-5 py-3 text-[13px] text-[#444748]">
          <p>
            {total > 0 ? `Showing ${from}–${to} of ${total}` : 'No results'}
            {search && rows.length !== (list.data?.data.length ?? 0) && ` · ${rows.length} match your search on this page`}
          </p>
          <div className="flex items-center gap-1">
            <PageButton disabled={page <= 1} onClick={() => setParam({ page: String(page - 1) })} label="Previous page" icon="chevron_left" />
            <span className="px-2 text-[12px] font-semibold text-[#1b1c1a]">Page {page}</span>
            <PageButton
              disabled={!list.data?.meta.has_more}
              onClick={() => setParam({ page: String(page + 1) })}
              label="Next page"
              icon="chevron_right"
            />
          </div>
        </div>
      </section>
    </div>
  );
}

function StatCard({
  label,
  icon,
  value,
  hint,
  href,
  accent,
  dark,
}: {
  label: string;
  icon: string;
  value: number | undefined;
  hint: string;
  href: string;
  accent?: boolean;
  dark?: boolean;
}) {
  return (
    <Link
      href={href}
      className={`group relative flex flex-col gap-3 overflow-hidden rounded-2xl p-5 transition-transform hover:-translate-y-0.5 ${
        dark ? 'bg-[#1b1c1c] text-white' : `bg-white ${cardShadow}`
      }`}
    >
      {!dark && <div className={`absolute inset-x-0 top-0 h-1 ${accent ? 'bg-[#c7ef39]' : 'bg-[#e3e2df]'}`} />}
      <div className="flex items-center justify-between">
        <span className={`text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] ${dark ? 'text-[#848483]' : 'text-[#444748]'}`}>
          {label}
        </span>
        <span className={`flex size-8 items-center justify-center rounded-full ${dark ? 'bg-[#c7ef39] text-[#171e00]' : accent ? 'bg-[rgba(199,239,57,0.2)] text-[#526600]' : 'bg-[#efeeeb] text-[#1b1c1a]'}`}>
          <Icon name={icon} size={16} />
        </span>
      </div>
      <p className={`${fontHeading} text-[32px] font-bold leading-[38px] tracking-[-0.8px]`}>
        {value ?? <span className="inline-block h-8 w-10 animate-pulse rounded bg-current opacity-10" />}
      </p>
      <p className={`flex items-center justify-between text-[12px] font-medium ${dark ? 'text-[#c7ef39]' : 'text-[#444748]'}`}>
        {hint}
        <Icon name="arrow_forward" size={14} className="transition-transform group-hover:translate-x-0.5" />
      </p>
    </Link>
  );
}

function Row({
  item,
  currency,
  showRequester,
  onOpen,
}: {
  item: RequestListItem;
  currency: string;
  showRequester: boolean;
  onOpen: () => void;
}) {
  const flagged = item.status === 'rejected' || item.status === 'failed';
  const aging =
    item.status === 'pending' ? (
      <span className={`text-[13px] ${item.days_pending >= 7 ? 'font-semibold text-[#b45309]' : 'text-[#444748]'}`}>
        {item.days_pending === 0 ? 'Pending since today' : `${item.days_pending} ${item.days_pending === 1 ? 'day' : 'days'} pending`}
      </span>
    ) : flagged ? (
      <span className="text-[13px] font-medium text-[#ba1a1a]">Needs your action</span>
    ) : (
      <span className="text-[13px] text-[#747878]">—</span>
    );

  return (
    <tr
      onClick={onOpen}
      className={`cursor-pointer border-t border-[#efeeeb] transition-colors hover:bg-[#faf9f6] ${flagged ? 'bg-[rgba(255,218,214,0.06)]' : ''}`}
    >
      <td className="max-w-[280px] py-3.5 pl-5 pr-3">
        <Link href={`/requests/${item.id}`} onClick={(e) => e.stopPropagation()} className="block">
          <span className={`block text-[13px] font-semibold leading-[18px] ${flagged ? 'text-[#ba1a1a]' : 'text-[#1b1c1a]'}`}>{item.id}</span>
          <span className="block truncate text-[13px] leading-[18px] text-[#444748]" title={item.purpose}>
            {item.purpose}
          </span>
        </Link>
      </td>
      {showRequester && (
        <td className="px-3 py-3.5 text-[13px] font-medium text-[#1b1c1a]">
          <span className="block">{item.requester.name}</span>
          <span className="block text-[12px] font-normal text-[#747878]">{item.requester.email}</span>
        </td>
      )}
      <td className="whitespace-nowrap px-3 py-3.5 text-[13px] text-[#444748]">{formatDateOnly(item.created_at)}</td>
      <td className="whitespace-nowrap px-3 py-3.5 text-right text-[13px] font-semibold tabular-nums">{formatMoney(item.amount, currency)}</td>
      <td className="px-3 py-3.5">
        <UrgencyPill urgency={item.urgency} />
      </td>
      <td className="px-3 py-3.5">
        <StatusPill status={item.status} />
      </td>
      <td className="px-3 py-3.5">{aging}</td>
      <td className="py-3.5 pl-3 pr-5 text-right">
        <Icon name="chevron_right" size={18} className="text-[#747878]" />
      </td>
    </tr>
  );
}

function PageButton({ disabled, onClick, label, icon }: { disabled: boolean; onClick: () => void; label: string; icon: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      className="flex size-8 items-center justify-center rounded-full text-[#1b1c1a] hover:bg-[#efeeeb] disabled:opacity-30 disabled:hover:bg-transparent"
    >
      <Icon name={icon} size={18} />
    </button>
  );
}

function Empty({ icon, title, body, action }: { icon: string; title: string; body: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 border-t border-[#efeeeb] px-6 py-14 text-center">
      <Icon name={icon} size={26} className="text-[#747878]" />
      <p className="text-[14px] font-semibold text-[#1b1c1a]">{title}</p>
      <p className="max-w-sm text-[13px] text-[#444748]">{body}</p>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
