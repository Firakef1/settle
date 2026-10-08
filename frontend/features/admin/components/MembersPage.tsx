'use client';

import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import { orgAPI } from '../../../shared/services/orgAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Member, Role } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { ROLES } from '../../../shared/utils/constants';
import { formatDateOnly } from '../../../shared/utils/format';
import { fontHeading, initials } from '../../../shared/utils/fonts';
import { useMembers, useRemoveMember, useSessionInvites, useUpdateRole } from '../hooks/useMembers';
import { CopyLink, InviteDialog } from './InviteDialog';

type Filter = 'all' | Role;
const ROLE_BADGE: Record<Role, string> = {
  org_admin: 'bg-black text-white',
  finance: 'bg-[#c7ef39] text-[#171e00]',
  staff: 'bg-[#efeeeb] text-[#1b1c1a]',
};

export function MembersPage() {
  const router = useRouter();
  const { currentOrg, user, logout } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const isAdmin = currentOrg?.role === 'org_admin';
  const members = useMembers(orgId);
  const invites = useSessionInvites(orgId).data ?? [];
  const stats = useQuery({ queryKey: ['dashboard', 'stats', orgId], queryFn: () => orgAPI.getStats(orgId), enabled: !!orgId });
  const updateRole = useUpdateRole(orgId);
  const remove = useRemoveMember(orgId);

  const [filter, setFilter] = useState<Filter>('all');
  const [search, setSearch] = useState('');
  const [inviteOpen, setInviteOpen] = useState(false);
  const [removing, setRemoving] = useState<Member | null>(null);

  const all = members.data ?? [];
  const counts = { all: all.length, org_admin: 0, finance: 0, staff: 0 } as Record<Filter, number>;
  for (const m of all) counts[m.role] += 1;
  const rows = all.filter((m) => {
    const q = search.trim().toLowerCase();
    return (filter === 'all' || m.role === filter) && (!q || m.name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q));
  });
  const joinedEmails = new Set(all.map((m) => m.email.toLowerCase()));
  const pendingInvites = invites.filter((i) => !joinedEmails.has(i.email.toLowerCase()));
  const usage = stats.data?.plan_usage;

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1">
          <p className="text-[12px] font-medium text-[#747878]">Settings / Members</p>
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Members & Access</h1>
          <p className="text-[14px] text-[#444748]">
            {isAdmin ? 'Invite people, change roles, and remove access.' : 'Everyone in the organization. Only admins can make changes.'}
          </p>
        </div>
        {isAdmin && (
          <button
            type="button"
            onClick={() => setInviteOpen(true)}
            className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-black px-5 py-2.5 text-[14px] font-semibold text-white hover:bg-[#1c1b1b]`}
          >
            <Icon name="person_add" size={16} />
            Invite member
          </button>
        )}
      </section>

      {pendingInvites.length > 0 && (
        <section className="rounded-2xl bg-white p-5 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
          <h2 className={`${fontHeading} text-[15px] font-semibold`}>Invites created this session</h2>
          <p className="text-[12px] text-[#747878]">The API doesn&apos;t list invitations, so these disappear on reload. Links stay valid until they expire.</p>
          <ul className="mt-3 flex flex-col gap-3">
            {pendingInvites.map((i) => (
              <li key={i.id} className="flex flex-col gap-1.5">
                <p className="flex flex-wrap items-center gap-2 text-[13px]">
                  <span className="font-semibold">{i.email}</span>
                  <span className={`rounded-full px-2 py-0.5 text-[11px] font-semibold ${ROLE_BADGE[i.role]}`}>{ROLES[i.role]}</span>
                  <span className="text-[12px] text-[#747878]">expires {formatDateOnly(i.expires_at)}</span>
                </p>
                <CopyLink token={i.token} />
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="flex flex-col overflow-hidden rounded-2xl bg-white shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[#efeeeb] px-5 py-4">
          <div role="tablist" aria-label="Filter by role" className="flex rounded-full bg-[#f4f3f0] p-1">
            {(['all', 'org_admin', 'finance', 'staff'] as Filter[]).map((f) => (
              <button
                key={f}
                type="button"
                role="tab"
                aria-selected={filter === f}
                onClick={() => setFilter(f)}
                className={`rounded-full px-3.5 py-1 text-[13px] ${filter === f ? 'bg-white font-semibold shadow-sm' : 'text-[#444748]'}`}
              >
                {f === 'all' ? 'All' : ROLES[f]} <span className="text-[#747878]">{counts[f]}</span>
              </button>
            ))}
          </div>
          <div className="relative w-full sm:w-64">
            <Icon name="search" size={15} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[#444748]" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search name or email…"
              aria-label="Search members"
              className="h-9 w-full rounded-full bg-[#f4f3f0] pl-9 pr-3 text-[13px] outline-none focus:ring-2 focus:ring-[#c7ef39]/60"
            />
          </div>
        </div>

        {members.isPending ? (
          <div className="flex flex-col gap-2 p-5" aria-busy="true">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-14 animate-pulse rounded-xl bg-[#f4f3f0]" />
            ))}
          </div>
        ) : members.isError ? (
          <p className="px-5 py-12 text-center text-[13px] text-[#ba1a1a]">{apiErrorMessage(members.error, "Couldn't load members.")}</p>
        ) : rows.length === 0 ? (
          <p className="px-5 py-12 text-center text-[13px] text-[#444748]">No members match.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px] border-collapse text-left">
              <thead className="bg-[rgba(244,243,240,0.7)]">
                <tr className="text-[11px] font-semibold tracking-[0.55px] text-[#444748]">
                  <th scope="col" className="py-3 pl-5 pr-3">Member</th>
                  <th scope="col" className="px-3 py-3">Role</th>
                  <th scope="col" className="px-3 py-3">Joined</th>
                  {isAdmin && <th scope="col" className="py-3 pl-3 pr-5 text-right">Actions</th>}
                </tr>
              </thead>
              <tbody>
                {rows.map((m) => {
                  const me = m.user_id === user?.id;
                  return (
                    <tr key={m.user_id} className="border-t border-[#efeeeb]">
                      <td className="py-3.5 pl-5 pr-3">
                        <div className="flex items-center gap-3">
                          <span className={`${fontHeading} flex size-9 items-center justify-center rounded-full text-[12px] font-bold ${me ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-[#1b1c1a] text-white'}`}>
                            {initials(m.name)}
                          </span>
                          <div className="min-w-0">
                            <p className="truncate text-[14px] font-semibold">
                              {m.name} {me && <span className="text-[12px] font-normal text-[#747878]">(you)</span>}
                            </p>
                            <p className="truncate text-[12px] text-[#747878]">{m.email}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-3.5">
                        {isAdmin && m.role !== 'org_admin' ? (
                          <select
                            aria-label={`Role for ${m.name}`}
                            value={m.role}
                            disabled={updateRole.isPending}
                            onChange={(e) => updateRole.mutate({ userId: m.user_id, role: e.target.value as 'staff' | 'finance', name: m.name })}
                            className="h-8 rounded-full border border-[#e3e2df] bg-white px-3 text-[13px] outline-none focus:ring-2 focus:ring-[#c7ef39]/60"
                          >
                            <option value="staff">Staff</option>
                            <option value="finance">Finance</option>
                          </select>
                        ) : (
                          <span className={`rounded-full px-2.5 py-1 text-[12px] font-semibold ${ROLE_BADGE[m.role]}`}>{ROLES[m.role]}</span>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-3 py-3.5 text-[13px] text-[#444748]">{formatDateOnly(m.joined_at)}</td>
                      {isAdmin && (
                        <td className="py-3.5 pl-3 pr-5 text-right">
                          <button
                            type="button"
                            onClick={() => setRemoving(m)}
                            className="rounded-full px-3 py-1.5 text-[12px] font-semibold text-[#93000a] hover:bg-[#ffdad6]"
                          >
                            {me ? 'Leave' : 'Remove'}
                          </button>
                        </td>
                      )}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#efeeeb] bg-[rgba(244,243,240,0.4)] px-5 py-3 text-[13px] text-[#444748]">
          <p>
            {counts.all} {counts.all === 1 ? 'member' : 'members'} · {counts.org_admin} admin, {counts.finance} finance, {counts.staff} staff
          </p>
          {usage && (
            <p className={usage.user_limit > 0 && usage.user_count >= usage.user_limit ? 'font-semibold text-[#ba1a1a]' : ''}>
              Seats: {usage.user_count} / {usage.user_limit === 0 ? 'unlimited' : usage.user_limit} on the {usage.current_plan} plan
            </p>
          )}
        </div>
      </section>

      <InviteDialog open={inviteOpen} orgId={orgId} onClose={() => setInviteOpen(false)} />

      <Modal
        open={!!removing}
        onClose={() => setRemoving(null)}
        icon="person_remove"
        tone="danger"
        title={removing?.user_id === user?.id ? 'Leave this organization?' : `Remove ${removing?.name}?`}
        description={
          removing?.user_id === user?.id
            ? "You'll lose access right away. The last admin can't leave."
            : 'They lose access right away. Their past requests stay in the records.'
        }
        footer={
          <>
            <button type="button" onClick={() => setRemoving(null)} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
              Cancel
            </button>
            <button
              type="button"
              disabled={remove.isPending}
              onClick={() =>
                removing &&
                remove.mutate(
                  { userId: removing.user_id, name: removing.name },
                  {
                    onSettled: () => setRemoving(null),
                    // Leaving yourself ends access to this org: sign out.
                    onSuccess: async () => {
                      if (removing.user_id === user?.id) {
                        await logout();
                        router.replace('/login');
                      }
                    },
                  },
                )
              }
              className={`${fontHeading} rounded-full bg-[#ba1a1a] px-4 py-2 text-[14px] font-semibold text-white disabled:opacity-50`}
            >
              {remove.isPending ? 'Removing…' : removing?.user_id === user?.id ? 'Leave' : 'Remove'}
            </button>
          </>
        }
      />
    </div>
  );
}
