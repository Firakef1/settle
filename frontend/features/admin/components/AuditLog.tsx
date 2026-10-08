'use client';

import Link from 'next/link';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Fragment, useState, type ReactNode } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { orgAPI } from '../../../shared/services/orgAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { AuditLogEntry } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { formatISO } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';
import { useMembers } from '../hooks/useMembers';

// Actions the backend writes today (see backend/docs/Api-contract.md and #44).
const ACTIONS: Record<string, { label: string; icon: string; tone: string }> = {
  request_created: { label: 'Request created', icon: 'add_circle', tone: 'bg-[#efeeeb] text-[#1b1c1a]' },
  receipt_uploaded: { label: 'Receipt uploaded', icon: 'upload_file', tone: 'bg-[#efeeeb] text-[#1b1c1a]' },
  comment_added: { label: 'Comment added', icon: 'chat', tone: 'bg-[#efeeeb] text-[#1b1c1a]' },
  request_withdrawn: { label: 'Request withdrawn', icon: 'undo', tone: 'bg-[#efeeeb] text-[#747878]' },
  request_resubmitted: { label: 'Request resubmitted', icon: 'redo', tone: 'bg-[#efeeeb] text-[#1b1c1a]' },
  approved: { label: 'Approved', icon: 'check_circle', tone: 'bg-[rgba(199,239,57,0.25)] text-[#3d4c00]' },
  rejected: { label: 'Rejected', icon: 'cancel', tone: 'bg-[#ffdad6] text-[#93000a]' },
  paid: { label: 'Paid', icon: 'payments', tone: 'bg-[#1b1c1a] text-white' },
  failed: { label: 'Payment failed', icon: 'error', tone: 'bg-[#ffdad6] text-[#93000a]' },
  invite_created: { label: 'Invite created', icon: 'person_add', tone: 'bg-[#e3e2df] text-[#1b1c1a]' },
  invite_accepted: { label: 'Invite accepted', icon: 'how_to_reg', tone: 'bg-[#e3e2df] text-[#1b1c1a]' },
  role_updated: { label: 'Role changed', icon: 'manage_accounts', tone: 'bg-[#e3e2df] text-[#1b1c1a]' },
  member_removed: { label: 'Member removed', icon: 'person_remove', tone: 'bg-[#ffdad6] text-[#93000a]' },
  org_created: { label: 'Organization created', icon: 'domain_add', tone: 'bg-[#e3e2df] text-[#1b1c1a]' },
  org_updated: { label: 'Organization updated', icon: 'edit', tone: 'bg-[#e3e2df] text-[#1b1c1a]' },
  plan_updated: { label: 'Plan changed', icon: 'workspace_premium', tone: 'bg-[#c7ef39] text-[#171e00]' },
};

const LIMITS = [20, 50, 100];
const select =
  'h-9 rounded-full border border-[#e3e2df] bg-white px-3.5 text-[13px] text-[#1b1c1a] outline-none focus:ring-2 focus:ring-[#c7ef39]/60';

function str(v: unknown) {
  return typeof v === 'string' || typeof v === 'number' ? String(v) : '';
}

// One readable line per entry, using whatever metadata the action carries.
function describe(e: AuditLogEntry): ReactNode {
  const m = e.metadata ?? {};
  switch (e.action) {
    case 'invite_created':
      return `Invited ${str(m.email)} as ${str(m.role)}`;
    case 'invite_accepted':
      return `${str(m.email) || e.actor_name} joined as ${str(m.role)}`;
    case 'role_updated':
      return m.new_role ? `Role changed${m.old_role ? ` from ${str(m.old_role)}` : ''} to ${str(m.new_role)}` : 'Member role changed';
    case 'plan_updated':
      return `Plan ${str(m.old_plan)} → ${str(m.new_plan)}`;
    case 'org_updated':
      return Object.keys(m).length ? `Updated ${Object.keys(m).join(', ')}` : 'Organization details updated';
    case 'paid':
      return m.payment_method ? `Paid by ${str(m.payment_method).replace('_', ' ')}` : null;
    default: {
      const note = str(m.reason) || str(m.note) || str(m.failure_reason);
      return note || null;
    }
  }
}

export function AuditLog() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const { currentOrg } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const members = useMembers(orgId);
  const [open, setOpen] = useState<string | null>(null);

  const action = params.get('action') ?? '';
  const actor = params.get('actor') ?? '';
  const from = params.get('from') ?? '';
  const to = params.get('to') ?? '';
  const page = Math.max(1, Number(params.get('page')) || 1);
  const limit = LIMITS.includes(Number(params.get('limit'))) ? Number(params.get('limit')) : 20;

  // Date inputs are local days; send the day's start/end as ISO-8601.
  const query = {
    action: action || undefined,
    actor_id: actor || undefined,
    start_date: from ? new Date(`${from}T00:00:00`).toISOString() : undefined,
    end_date: to ? new Date(`${to}T23:59:59.999`).toISOString() : undefined,
    page,
    limit,
  };
  const log = useQuery({
    queryKey: ['audit-log', orgId, query],
    queryFn: () => orgAPI.getAuditLog(orgId, query),
    enabled: !!orgId,
    placeholderData: keepPreviousData,
  });

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

  const filtered = !!(action || actor || from || to);
  const p = log.data?.pagination;
  const rows = log.data?.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Audit log</h1>
          <p className="text-[14px] text-[#444748]">Who did what in {currentOrg?.org_name}. Filters are kept in the link, so you can share a view.</p>
        </div>
        {p && (
          <span className="rounded-full bg-white px-3 py-1.5 text-[12px] font-semibold text-[#444748] shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
            {p.total.toLocaleString('en-US')}{filtered ? ' matching' : ''} {p.total === 1 ? 'entry' : 'entries'}
          </span>
        )}
      </section>

      <section className="flex flex-col overflow-hidden rounded-2xl bg-white shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
        <div className="flex flex-wrap items-end gap-2 border-b border-[#efeeeb] px-5 py-4">
          <select aria-label="Action" value={action} onChange={(e) => setParam({ action: e.target.value || undefined })} className={select}>
            <option value="">All actions</option>
            {Object.entries(ACTIONS).map(([value, a]) => (
              <option key={value} value={value}>
                {a.label}
              </option>
            ))}
          </select>
          <select aria-label="Actor" value={actor} onChange={(e) => setParam({ actor: e.target.value || undefined })} className={select}>
            <option value="">Everyone</option>
            {(members.data ?? []).map((m) => (
              <option key={m.user_id} value={m.user_id}>
                {m.name}
              </option>
            ))}
          </select>
          <label className="flex items-center gap-1.5 text-[12px] text-[#444748]">
            From
            <input type="date" aria-label="From date" value={from} max={to || undefined} onChange={(e) => setParam({ from: e.target.value || undefined })} className={select} />
          </label>
          <label className="flex items-center gap-1.5 text-[12px] text-[#444748]">
            To
            <input type="date" aria-label="To date" value={to} min={from || undefined} onChange={(e) => setParam({ to: e.target.value || undefined })} className={select} />
          </label>
          {filtered && (
            <button
              type="button"
              onClick={() => setParam({ action: undefined, actor: undefined, from: undefined, to: undefined })}
              className="h-9 rounded-full px-3 text-[13px] font-medium text-[#444748] hover:bg-[#efeeeb]"
            >
              Clear filters
            </button>
          )}
        </div>

        {log.isPending ? (
          <div className="flex flex-col gap-2 p-5" aria-busy="true">
            {[0, 1, 2, 3, 4].map((i) => (
              <div key={i} className="h-11 animate-pulse rounded-xl bg-[#f4f3f0]" />
            ))}
          </div>
        ) : log.isError ? (
          <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <Icon name="error" size={26} className="text-[#ba1a1a]" />
            <p className="text-[13px] text-[#444748]">{apiErrorMessage(log.error, "Couldn't load the audit log.")}</p>
            <button type="button" onClick={() => log.refetch()} className="rounded-full bg-[#efeeeb] px-4 py-2 text-[13px] font-semibold">
              Try again
            </button>
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
            <Icon name="manage_search" size={28} className="text-[#747878]" />
            <p className="text-[14px] font-semibold">{filtered ? 'No entries match these filters' : 'Nothing logged yet'}</p>
          </div>
        ) : (
          <div className={`overflow-x-auto ${log.isFetching ? 'opacity-70' : ''}`}>
            <table className="w-full min-w-[820px] border-collapse text-left">
              <thead className="bg-[rgba(244,243,240,0.7)]">
                <tr className="text-[11px] font-semibold tracking-[0.55px] text-[#444748]">
                  <th scope="col" className="py-3 pl-5 pr-3">Time</th>
                  <th scope="col" className="px-3 py-3">Actor</th>
                  <th scope="col" className="px-3 py-3">Action</th>
                  <th scope="col" className="px-3 py-3">Target</th>
                  <th scope="col" className="py-3 pl-3 pr-5">
                    <span className="sr-only">Details</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((e) => {
                  const a = ACTIONS[e.action] ?? { label: e.action, icon: 'bolt', tone: 'bg-[#efeeeb] text-[#1b1c1a]' };
                  const summary = describe(e);
                  const expanded = open === e.id;
                  const hasMeta = e.metadata && Object.keys(e.metadata).length > 0;
                  return (
                    <Fragment key={e.id}>
                      <tr className="border-t border-[#efeeeb] align-top">
                        <td className="whitespace-nowrap py-3 pl-5 pr-3 text-[13px] text-[#444748]">{formatISO(e.created_at)}</td>
                        <td className="px-3 py-3 text-[13px] font-semibold">
                          <button type="button" onClick={() => setParam({ actor: e.actor_id })} className="hover:underline" title="Show only this person">
                            {e.actor_name || 'Unknown user'}
                          </button>
                        </td>
                        <td className="px-3 py-3">
                          <span className={`inline-flex items-center gap-1 whitespace-nowrap rounded-full px-2.5 py-0.5 text-[12px] font-semibold ${a.tone}`}>
                            <Icon name={a.icon} size={13} />
                            {a.label}
                          </span>
                          {summary && <p className="mt-1 text-[12px] text-[#444748]">{summary}</p>}
                        </td>
                        <td className="px-3 py-3 text-[13px]">
                          {e.target_id?.startsWith('REQ-') ? (
                            <Link href={`/requests/${e.target_id}`} className="font-mono font-semibold hover:underline">
                              {e.target_id}
                            </Link>
                          ) : (
                            <span className="font-mono text-[12px] text-[#747878]" title={e.target_id}>
                              {e.target_id ? `${e.target_id.slice(0, 8)}…` : '—'}
                            </span>
                          )}
                        </td>
                        <td className="py-3 pl-3 pr-5 text-right">
                          {hasMeta && (
                            <button
                              type="button"
                              onClick={() => setOpen(expanded ? null : e.id)}
                              aria-expanded={expanded}
                              className="flex items-center gap-0.5 rounded-full px-2 py-1 text-[12px] font-medium text-[#444748] hover:bg-[#efeeeb]"
                            >
                              Details
                              <Icon name={expanded ? 'expand_less' : 'expand_more'} size={16} />
                            </button>
                          )}
                        </td>
                      </tr>
                      {expanded && (
                        <tr className="bg-[#faf9f6]">
                          <td colSpan={5} className="px-5 pb-4">
                            <pre className="overflow-x-auto rounded-xl bg-[#1b1c1c] p-4 font-mono text-[12px] leading-5 text-[#e3e2df]">
                              {JSON.stringify(e.metadata, null, 2)}
                            </pre>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#efeeeb] bg-[rgba(244,243,240,0.4)] px-5 py-3 text-[13px] text-[#444748]">
          <label className="flex items-center gap-2">
            Rows
            <select value={limit} onChange={(e) => setParam({ limit: e.target.value === '20' ? undefined : e.target.value })} className="h-8 rounded-full border border-[#e3e2df] bg-white px-2 text-[12px]">
              {LIMITS.map((l) => (
                <option key={l} value={l}>
                  {l}
                </option>
              ))}
            </select>
          </label>
          <div className="flex items-center gap-3">
            {p && p.total > 0 && (
              <span>
                {p.offset + 1}–{p.offset + rows.length} of {p.total}
              </span>
            )}
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
              <span className="px-1 text-[12px] font-semibold text-[#1b1c1a]">Page {page}</span>
              <button
                type="button"
                aria-label="Next page"
                disabled={!p?.has_more}
                onClick={() => setParam({ page: String(page + 1) })}
                className="flex size-8 items-center justify-center rounded-full hover:bg-white disabled:opacity-30"
              >
                <Icon name="chevron_right" size={18} />
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
