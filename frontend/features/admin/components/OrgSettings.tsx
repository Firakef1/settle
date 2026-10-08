'use client';

import Link from 'next/link';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { orgAPI } from '../../../shared/services/orgAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import { toast } from '../../../shared/stores/notificationStore';
import type { Currency, Organization } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { CURRENCIES, PLANS } from '../../../shared/utils/constants';
import { formatDateOnly } from '../../../shared/utils/format';
import { fontHeading } from '../../../shared/utils/fonts';

const card = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const label = 'text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#444748]';
const input =
  'h-11 w-full rounded-xl bg-[#f4f3f0] px-4 text-[14px] outline-none focus:ring-2 focus:ring-[#c7ef39]/60 disabled:cursor-not-allowed disabled:text-[#444748]';

export function OrgSettings() {
  const { currentOrg } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const role = currentOrg?.role;
  const isAdmin = role === 'org_admin';
  const org = useOrganization(orgId);
  const stats = useQuery({
    queryKey: ['dashboard', 'stats', orgId],
    queryFn: () => orgAPI.getStats(orgId),
    enabled: !!orgId && (role === 'finance' || role === 'org_admin'),
  });

  if (org.isPending) return <div className="h-96 animate-pulse rounded-2xl bg-[#efeeeb]" aria-busy="true" />;
  if (org.isError) {
    return (
      <div className={`${card} text-center text-[14px] text-[#ba1a1a]`}>
        {apiErrorMessage(org.error, "Couldn't load the organization.")}{' '}
        <button type="button" onClick={() => org.refetch()} className="font-semibold underline">
          Try again
        </button>
      </div>
    );
  }

  const data = org.data;
  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Organization settings</h1>
          <p className="text-[14px] text-[#444748]">
            {isAdmin ? 'Name and currency for your workspace.' : 'Only admins can change these settings.'}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Chip icon="workspace_premium">{PLANS[data.plan] ?? data.plan} plan</Chip>
          <Chip icon="payments">{data.currency}</Chip>
          {stats.data && <Chip icon="group">{stats.data.member_counts.total} members</Chip>}
        </div>
      </section>

      <OrgForm org={data} editable={isAdmin} />

      <section className="grid gap-4 md:grid-cols-3">
        <InfoCard icon="workspace_premium" title="Plan" value={`${PLANS[data.plan] ?? data.plan}`} href={role === 'staff' ? undefined : '/settings/billing'} link="Billing & plan">
          {stats.data &&
            `${stats.data.plan_usage.requests_this_month} requests this month of ${stats.data.plan_usage.request_limit || 'unlimited'}`}
        </InfoCard>
        <InfoCard icon="group" title="Members" value={stats.data ? String(stats.data.member_counts.total) : '—'} href={role === 'staff' ? undefined : '/settings/members'} link="Manage members">
          {stats.data && `${stats.data.member_counts.org_admin} admin · ${stats.data.member_counts.finance} finance · ${stats.data.member_counts.staff} staff`}
        </InfoCard>
        <div className="relative overflow-hidden rounded-2xl bg-[#1b1c1c] p-5 text-white">
          <div aria-hidden className="pointer-events-none absolute -right-10 -top-10 size-28 rounded-full bg-[rgba(199,239,57,0.12)] blur-[24px]" />
          <p className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-[0.55px] text-[#848483]">
            <Icon name="history" size={14} className="text-[#c7ef39]" />
            Workspace
          </p>
          <p className={`${fontHeading} mt-2 text-[18px] font-bold`}>Since {formatDateOnly(data.created_at)}</p>
          <p className="mt-1 text-[12px] text-[#a3a3a2]">Last changed {formatDateOnly(data.updated_at)}</p>
        </div>
      </section>
    </div>
  );
}

function OrgForm({ org, editable }: { org: Organization; editable: boolean }) {
  const queryClient = useQueryClient();
  const [name, setName] = useState(org.name);
  const [currency, setCurrency] = useState<Currency>(org.currency);
  useEffect(() => {
    setName(org.name);
    setCurrency(org.currency);
  }, [org.name, org.currency]);

  const save = useMutation({
    mutationFn: () => orgAPI.update(org.id, { name: name.trim(), currency }),
    onSuccess: async (updated) => {
      queryClient.setQueryData(['dashboard', 'org', org.id], updated);
      // Keep the sidebar and greeting in step with the new name.
      useAuthStore.setState((s) => ({
        currentOrg: s.currentOrg?.org_id === updated.id ? { ...s.currentOrg, org_name: updated.name } : s.currentOrg,
        orgs: s.orgs.map((o) => (o.org_id === updated.id ? { ...o, org_name: updated.name } : o)),
      }));
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      toast.success('Organization settings saved.');
    },
  });

  const trimmed = name.trim();
  const nameError = !trimmed ? 'A name is required.' : trimmed.length > 255 ? 'Keep it under 255 characters.' : '';
  const dirty = trimmed !== org.name || currency !== org.currency;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!nameError && dirty) save.mutate();
  };

  return (
    <form onSubmit={submit} className={card}>
      <h2 className={`${fontHeading} text-[18px] font-semibold`}>General</h2>
      <div className="mt-5 grid gap-5 md:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <label htmlFor="org-name" className={label}>
            Organization name
          </label>
          <input id="org-name" value={name} maxLength={255} disabled={!editable} onChange={(e) => setName(e.target.value)} className={input} />
          {editable && nameError && <p className="text-[12px] font-medium text-[#ba1a1a]">{nameError}</p>}
        </div>
        <div className="flex flex-col gap-1.5">
          <label htmlFor="org-slug" className={label}>
            Workspace slug
          </label>
          <div className="relative">
            <input id="org-slug" value={org.slug} disabled className={`${input} pr-10 font-mono text-[13px]`} />
            <Icon name="lock" size={15} className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[#747878]" />
          </div>
          <p className="text-[12px] text-[#747878]">Generated from the name when the organization was created.</p>
        </div>
        <div className="flex flex-col gap-1.5">
          <label htmlFor="org-currency-setting" className={label}>
            Base currency
          </label>
          <select
            id="org-currency-setting"
            value={currency}
            disabled={!editable}
            onChange={(e) => setCurrency(e.target.value as Currency)}
            className={input}
          >
            {Object.keys(CURRENCIES).map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          {editable && currency !== org.currency && (
            <p className="flex items-start gap-1 text-[12px] text-[#6b4e00]">
              <Icon name="warning" size={14} className="mt-px" />
              Existing amounts aren&apos;t converted; they&apos;ll just be shown in {currency}.
            </p>
          )}
        </div>
      </div>
      {editable && (
        <div className="mt-6 flex flex-wrap items-center justify-end gap-3 border-t border-[#efeeeb] pt-5">
          {save.isError && <p className="mr-auto text-[12px] font-medium text-[#ba1a1a]">{apiErrorMessage(save.error, 'Could not save.')}</p>}
          <button
            type="button"
            disabled={!dirty || save.isPending}
            onClick={() => {
              setName(org.name);
              setCurrency(org.currency);
            }}
            className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb] disabled:opacity-40"
          >
            Discard
          </button>
          <button
            type="submit"
            disabled={!dirty || !!nameError || save.isPending}
            className={`${fontHeading} rounded-full bg-black px-5 py-2 text-[14px] font-semibold text-white disabled:opacity-40`}
          >
            {save.isPending ? 'Saving…' : 'Save changes'}
          </button>
        </div>
      )}
    </form>
  );
}

function Chip({ icon, children }: { icon: string; children: ReactNode }) {
  return (
    <span className="flex items-center gap-1.5 rounded-full bg-white px-3 py-1.5 text-[12px] font-semibold text-[#444748] shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
      <Icon name={icon} size={14} />
      {children}
    </span>
  );
}

function InfoCard({ icon, title, value, href, link, children }: { icon: string; title: string; value: string; href?: string; link: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1 rounded-2xl bg-white p-5 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
      <p className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-[0.55px] text-[#444748]">
        <Icon name={icon} size={14} />
        {title}
      </p>
      <p className={`${fontHeading} text-[22px] font-bold`}>{value}</p>
      {children && <p className="text-[12px] text-[#444748]">{children}</p>}
      {href && (
        <Link href={href} className="mt-2 flex items-center gap-1 text-[13px] font-semibold hover:underline">
          {link}
          <Icon name="arrow_forward" size={14} />
        </Link>
      )}
    </div>
  );
}
