'use client';

import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Icon } from '../../../shared/components/Icon';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { orgAPI } from '../../../shared/services/orgAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { RequestListItem, RequestStatus, Urgency } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { formatDateOnly, formatMoney } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useRequestCount, useRequestList } from '../../requests/hooks/useRequests';
import { StatusPill, UrgencyPill } from '../../requests/components/pills';
import type { DecisionKind } from '../hooks/useDecision';
import { DecisionDialog, allowedDecisions, type DecisionTarget } from './DecisionDialog';

const PAGE_SIZE = 20;
const ATTENTION_DAYS = 7;
type Tab = 'pending' | 'attention' | 'approved' | 'paid' | 'rejected' | 'failed';

const TABS: Array<{ id: Tab; label: string; status: RequestStatus }> = [
  { id: 'pending', label: 'Pending', status: 'pending' },
  { id: 'attention', label: 'Needs attention', status: 'pending' },
  { id: 'approved', label: 'Awaiting payment', status: 'approved' },
  { id: 'paid', label: 'Paid', status: 'paid' },
  { id: 'rejected', label: 'Rejected', status: 'rejected' },
  { id: 'failed', label: 'Failed', status: 'failed' },
];
const select =
  'h-9 rounded-full border border-[#e3e2df] bg-white px-3.5 text-[13px] text-[#1b1c1a] outline-none focus:ring-2 focus:ring-[#c7ef39]/60';

export function ApprovalsQueue() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const { currentOrg } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const isAdmin = currentOrg?.role === 'org_admin';
  const currency = useOrganization(orgId).data?.currency ?? 'USD';

  const tab: Tab = params.get('view') === 'attention' ? 'attention' : ((TABS.find((t) => t.id === params.get('tab'))?.id ?? 'pending') as Tab);
  const status = TABS.find((t) => t.id === tab)!.status;
  const urgency = (['routine', 'urgent', 'critical'] as Urgency[]).find((u) => u === params.get('urgency'));
  const requester = params.get('requester') ?? undefined;
  // Queues work oldest first by default.
  const order = params.get('order') === 'desc' ? 'desc' : 'asc';
  const page = Math.max(1, Number(params.get('page')) || 1);
  const [decide, setDecide] = useState<{ target: DecisionTarget; kind: DecisionKind } | null>(null);

  // "Needs attention" is pending + 7 days or older: pull one large page and filter.
  const attention = tab === 'attention';
  const list = useRequestList(orgId, {
    status,
    urgency,
    requester_id: requester,
    sort_by: 'created_at',
    sort_order: attention ? 'asc' : order,
    limit: attention ? 100 : PAGE_SIZE,
    offset: attention ? 0 : (page - 1) * PAGE_SIZE,
  });
  const members = useQuery({ queryKey: ['members', orgId], queryFn: () => orgAPI.listMembers(orgId), enabled: !!orgId, staleTime: 60_000 });

  const counts = {
    pending: useRequestCount(orgId, 'pending').data,
    approved: useRequestCount(orgId, 'approved').data,
    paid: useRequestCount(orgId, 'paid').data,
    rejected: useRequestCount(orgId, 'rejected').data,
    failed: useRequestCount(orgId, 'failed').data,
  };

  const setParam = (updates: Record<string, string | undefined>) => {
    const next = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(updates)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    if (!('page' in updates)) next.delete('page');
    const qs = next.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  };

  const rows = (list.data?.data ?? []).filter((r) => !attention || r.days_pending >= ATTENTION_DAYS);
  const total = attention ? rows.length : list.data?.meta.total ?? 0;
  const from = total === 0 ? 0 : attention ? 1 : (page - 1) * PAGE_SIZE + 1;
  const to = attention ? rows.length : Math.min(page * PAGE_SIZE, total);

  const open = (item: RequestListItem, kind: DecisionKind) =>
    setDecide({
      kind,
      target: { id: item.id, amount: item.amount, purpose: item.purpose, status: item.status, urgency: item.urgency, requesterName: item.requester.name },
    });

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Approvals Queue</h1>
          <p className="text-[14px] leading-5 text-[#444748]">
            Review submitted requests, then record each payment. Oldest first; anything pending {ATTENTION_DAYS}+ days is flagged.
          </p>
        </div>
        <div className="flex items-center gap-2 rounded-full bg-white px-4 py-2 text-[12px] font-semibold text-[#444748] shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
          <span className="size-2 rounded-full bg-[#c7ef39]" />
          {counts.pending ?? '…'} pending · {counts.approved ?? '…'} awaiting payment
        </div>
      </section>

      {isAdmin && (
        <p className="flex items-start gap-2 rounded-xl bg-[#fff8e1] px-4 py-3 text-[13px] text-[#6b4e00]">
          <Icon name="info" size={16} className="mt-0.5" />
          The API returns only an admin&apos;s own requests from this list, so you may not see everyone&apos;s. The Overview page still shows the
          org-wide counts and escalations. A finance member sees the full queue.
        </p>
      )}

      <section className="flex flex-col overflow-hidden rounded-2xl bg-white shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
        <div className="flex flex-col gap-3 border-b border-[#efeeeb] px-5 py-4">
          <div role="tablist" aria-label="Queue" className="flex flex-wrap gap-1">
            {TABS.map((t) => {
              const selected = tab === t.id;
              const count = t.id === 'attention' ? undefined : counts[t.id as keyof typeof counts];
              return (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  aria-selected={selected}
                  onClick={() => setParam(t.id === 'attention' ? { view: 'attention', tab: undefined } : { tab: t.id === 'pending' ? undefined : t.id, view: undefined })}
                  className={`flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-[13px] transition-colors ${
                    selected ? 'bg-[#1b1c1a] font-semibold text-white' : 'text-[#444748] hover:bg-[#f4f3f0]'
                  }`}
                >
                  {t.id === 'attention' && <Icon name="warning" size={14} className={selected ? 'text-[#ffb4ab]' : 'text-[#ba1a1a]'} />}
                  {t.label}
                  {count !== undefined && (
                    <span className={`rounded-full px-1.5 text-[11px] font-bold ${selected ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-[#efeeeb]'}`}>{count}</span>
                  )}
                </button>
              );
            })}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select aria-label="Urgency" value={urgency ?? ''} onChange={(e) => setParam({ urgency: e.target.value || undefined })} className={select}>
              <option value="">All urgencies</option>
              <option value="critical">Critical</option>
              <option value="urgent">Urgent</option>
              <option value="routine">Routine</option>
            </select>
            <select aria-label="Requester" value={requester ?? ''} onChange={(e) => setParam({ requester: e.target.value || undefined })} className={select}>
              <option value="">All requesters</option>
              {(members.data ?? []).map((m) => (
                <option key={m.user_id} value={m.user_id}>
                  {m.name}
                </option>
              ))}
            </select>
            {!attention && (
              <button type="button" onClick={() => setParam({ order: order === 'asc' ? 'desc' : undefined })} className={`${select} flex items-center gap-1`}>
                <Icon name={order === 'asc' ? 'arrow_upward' : 'arrow_downward'} size={14} />
                {order === 'asc' ? 'Oldest first' : 'Newest first'}
              </button>
            )}
            {(urgency || requester) && (
              <button type="button" onClick={() => setParam({ urgency: undefined, requester: undefined })} className="h-9 rounded-full px-3 text-[13px] font-medium text-[#444748] hover:bg-[#efeeeb]">
                Clear filters
              </button>
            )}
          </div>
        </div>

        {list.isPending ? (
          <div className="flex flex-col gap-2 p-5" aria-busy="true">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="h-14 animate-pulse rounded-xl bg-[#f4f3f0]" />
            ))}
          </div>
        ) : list.isError ? (
          <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <Icon name="error" size={26} className="text-[#ba1a1a]" />
            <p className="text-[14px] font-semibold">Couldn&apos;t load the queue</p>
            <p className="text-[13px] text-[#444748]">{apiErrorMessage(list.error)}</p>
            <button type="button" onClick={() => list.refetch()} className="mt-2 rounded-full bg-[#efeeeb] px-4 py-2 text-[13px] font-semibold">
              Try again
            </button>
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <Icon name="task_alt" size={28} className="text-[#526600]" />
            <p className="text-[14px] font-semibold">{attention ? `Nothing pending ${ATTENTION_DAYS}+ days` : 'Queue is clear'}</p>
            <p className="text-[13px] text-[#444748]">
              {urgency || requester ? 'No requests match these filters.' : 'New submissions will show up here.'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] border-collapse text-left">
              <thead className="bg-[#f4f3f0]">
                <tr className="text-[11px] font-bold leading-4 tracking-[0.55px] text-[#444748]">
                  <th scope="col" className="py-3 pl-5 pr-3">Request</th>
                  <th scope="col" className="px-3 py-3">Requester</th>
                  <th scope="col" className="px-3 py-3 text-right">Amount</th>
                  <th scope="col" className="px-3 py-3">Urgency</th>
                  <th scope="col" className="px-3 py-3 text-center">Days</th>
                  <th scope="col" className="px-3 py-3">Status</th>
                  <th scope="col" className="py-3 pl-3 pr-5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((item) => {
                  const old = item.status === 'pending' && item.days_pending >= ATTENTION_DAYS;
                  const edge = item.urgency === 'critical' ? 'border-l-[#ba1a1a]' : item.urgency === 'urgent' ? 'border-l-[#f59e0b]' : 'border-l-transparent';
                  return (
                    <tr
                      key={item.id}
                      onClick={() => router.push(`/requests/${item.id}`)}
                      className={`cursor-pointer border-l-4 border-t border-t-[#efeeeb] ${edge} transition-colors hover:bg-[#faf9f6] ${old ? 'bg-[rgba(255,218,214,0.12)]' : ''}`}
                    >
                      <td className="max-w-[260px] py-3.5 pl-4 pr-3">
                        <Link href={`/requests/${item.id}`} onClick={(e) => e.stopPropagation()} className="block">
                          <span className="block text-[13px] font-bold tracking-[-0.3px]">{item.id}</span>
                          <span className="block truncate text-[13px] text-[#444748]" title={item.purpose}>
                            {item.purpose}
                          </span>
                        </Link>
                      </td>
                      <td className="px-3 py-3.5 text-[13px]">
                        <span className="block font-semibold">{item.requester.name}</span>
                        <span className="block text-[12px] text-[#747878]">{formatDateOnly(item.created_at)}</span>
                      </td>
                      <td className="whitespace-nowrap px-3 py-3.5 text-right font-mono text-[13px] font-bold">{formatMoney(item.amount, currency)}</td>
                      <td className="px-3 py-3.5">
                        <UrgencyPill urgency={item.urgency} />
                      </td>
                      <td className="px-3 py-3.5 text-center">
                        <span
                          className={`rounded-full px-2.5 py-0.5 text-[12px] font-bold ${
                            item.days_pending >= 14 ? 'bg-[#ba1a1a] text-white' : old ? 'bg-[#ffdad6] text-[#93000a]' : 'bg-[#efeeeb] text-[#1b1c1a]'
                          }`}
                        >
                          {item.days_pending}d
                        </span>
                      </td>
                      <td className="px-3 py-3.5">
                        <StatusPill status={item.status} />
                      </td>
                      <td className="py-3.5 pl-3 pr-5">
                        <div className="flex justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                          {allowedDecisions(item.status).map((k) => (
                            <button
                              key={k}
                              type="button"
                              onClick={() => open(item, k)}
                              className={`${fontHeading} whitespace-nowrap rounded-full px-3 py-1.5 text-[12px] font-semibold transition-colors ${
                                k === 'approve' || k === 'paid' ? 'bg-black text-white hover:bg-[#1c1b1b]' : 'bg-[#efeeeb] text-[#93000a] hover:bg-[#ffdad6]'
                              }`}
                            >
                              {k === 'approve' ? 'Approve' : k === 'reject' ? 'Reject' : k === 'paid' ? 'Mark paid' : 'Failed'}
                            </button>
                          ))}
                          {allowedDecisions(item.status).length === 0 && <Icon name="chevron_right" size={18} className="text-[#747878]" />}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#efeeeb] bg-[#f4f3f0] px-5 py-3 text-[13px] text-[#444748]">
          <p>{total > 0 ? `Showing ${from}–${to} of ${total}` : 'No results'}</p>
          {!attention && (
            <div className="flex items-center gap-1">
              <button
                type="button"
                aria-label="Previous page"
                disabled={page <= 1}
                onClick={() => setParam({ page: String(page - 1) })}
                className="flex size-8 items-center justify-center rounded-full hover:bg-white disabled:opacity-30"
              >
                <Icon name="chevron_left" size={18} />
              </button>
              <span className="px-2 text-[12px] font-semibold text-[#1b1c1a]">Page {page}</span>
              <button
                type="button"
                aria-label="Next page"
                disabled={!list.data?.meta.has_more}
                onClick={() => setParam({ page: String(page + 1) })}
                className="flex size-8 items-center justify-center rounded-full hover:bg-white disabled:opacity-30"
              >
                <Icon name="chevron_right" size={18} />
              </button>
            </div>
          )}
        </div>
      </section>

      <DecisionDialog target={decide?.target ?? null} initial={decide?.kind ?? 'approve'} currency={currency} onClose={() => setDecide(null)} />
    </div>
  );
}
