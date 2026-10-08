'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { orgAPI } from '../../../shared/services/orgAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import { toast } from '../../../shared/stores/notificationStore';
import type { Plan } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { fontHeading } from '../../../shared/utils/fonts';

const card = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const limitText = (n: number) => (n === 0 ? 'Unlimited' : n.toLocaleString('en-US'));

export function BillingPage() {
  const queryClient = useQueryClient();
  const { currentOrg } = useAuthStore();
  const orgId = currentOrg?.org_id ?? '';
  const isAdmin = currentOrg?.role === 'org_admin';
  const org = useOrganization(orgId);
  const stats = useQuery({ queryKey: ['dashboard', 'stats', orgId], queryFn: () => orgAPI.getStats(orgId), enabled: !!orgId });
  const plans = useQuery({ queryKey: ['billing', 'plans'], queryFn: orgAPI.listPlans, staleTime: 10 * 60 * 1000 });
  const [target, setTarget] = useState<Plan | null>(null);

  const change = useMutation({
    mutationFn: (plan: Plan) => orgAPI.changePlan(orgId, plan.id),
    onSuccess: async (_, plan) => {
      toast.success(`You're now on the ${plan.name} plan.`);
      setTarget(null);
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });

  const current = org.data?.plan ?? stats.data?.plan_usage.current_plan;
  const usage = stats.data?.plan_usage;
  const currentPlan = plans.data?.find((p) => p.id === current);
  const reqPct = usage && usage.request_limit > 0 ? (usage.requests_this_month / usage.request_limit) * 100 : 0;
  const userPct = usage && usage.user_limit > 0 ? (usage.user_count / usage.user_limit) * 100 : 0;
  const near = Math.max(reqPct, userPct) >= 80;

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>Billing & plan</h1>
          <p className="text-[14px] text-[#444748]">
            {isAdmin ? 'See this month’s usage and change your plan.' : 'Your organization’s plan and usage. Only admins can change the plan.'}
          </p>
        </div>
        {currentPlan && (
          <span className="flex items-center gap-1.5 rounded-full bg-[#c7ef39] px-3 py-1.5 text-[12px] font-bold text-[#171e00]">
            <Icon name="workspace_premium" size={14} />
            {currentPlan.name} · ${currentPlan.price}/mo
          </span>
        )}
      </section>

      {near && (
        <p className="flex items-start gap-2 rounded-xl bg-[#fff8e1] px-4 py-3 text-[13px] text-[#6b4e00]">
          <Icon name="warning" size={16} className="mt-0.5" />
          You&apos;re at {Math.round(Math.max(reqPct, userPct))}% of a plan limit. Requests aren&apos;t blocked over the limit, but consider upgrading.
        </p>
      )}

      <section className={`${card} grid gap-8 md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]`}>
        <div className="flex flex-col gap-5">
          <h2 className={`${fontHeading} text-[18px] font-semibold`}>This month&apos;s usage</h2>
          {usage ? (
            <>
              <UsageBar label="Payout requests" used={usage.requests_this_month} limit={usage.request_limit} pct={reqPct} />
              <UsageBar label="Team members" used={usage.user_count} limit={usage.user_limit} pct={userPct} />
            </>
          ) : stats.isError ? (
            <p className="text-[13px] text-[#ba1a1a]">{apiErrorMessage(stats.error, "Couldn't load usage.")}</p>
          ) : (
            <div className="h-24 animate-pulse rounded-xl bg-[#f4f3f0]" />
          )}
        </div>
        <div className="flex flex-col items-center justify-center gap-2">
          <SemiGauge pct={usage && usage.request_limit > 0 ? reqPct : null} />
          <p className="text-center text-[12px] text-[#444748]">
            {usage && usage.request_limit > 0 ? 'of monthly request allowance used' : 'Unlimited requests on this plan'}
          </p>
        </div>
      </section>

      <section className="flex flex-col gap-4">
        <h2 className={`${fontHeading} text-[18px] font-semibold`}>Plans</h2>
        {plans.isPending ? (
          <div className="grid gap-4 md:grid-cols-3">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-96 animate-pulse rounded-2xl bg-[#efeeeb]" />
            ))}
          </div>
        ) : plans.isError ? (
          <p className="text-[13px] text-[#ba1a1a]">{apiErrorMessage(plans.error, "Couldn't load plans.")}</p>
        ) : (
          <div className="grid items-stretch gap-4 md:grid-cols-3">
            {plans.data.map((p) => {
              const isCurrent = p.id === current;
              const featured = p.id === 'starter';
              return (
                <div
                  key={p.id}
                  className={`relative flex flex-col gap-4 rounded-2xl p-6 ${
                    featured ? 'bg-[#1b1c1c] text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1)]' : 'bg-white shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]'
                  } ${isCurrent ? 'ring-2 ring-[#c7ef39]' : ''}`}
                >
                  <div className="flex items-center justify-between">
                    <h3 className={`${fontHeading} text-[20px] font-bold`}>{p.name}</h3>
                    {isCurrent ? (
                      <span className="rounded-full bg-[#c7ef39] px-2.5 py-0.5 text-[11px] font-bold text-[#171e00]">Current plan</span>
                    ) : (
                      featured && <span className="rounded-full bg-[rgba(199,239,57,0.15)] px-2.5 py-0.5 text-[11px] font-bold text-[#c7ef39]">Most popular</span>
                    )}
                  </div>
                  <p className={`${fontHeading} text-[36px] font-bold leading-none tracking-[-1px]`}>
                    ${p.price}
                    <span className={`text-[14px] font-medium ${featured ? 'text-[#a3a3a2]' : 'text-[#747878]'}`}> / month</span>
                  </p>
                  <div className={`grid grid-cols-2 gap-2 rounded-xl p-3 text-[12px] ${featured ? 'bg-[rgba(227,226,223,0.08)]' : 'bg-[#f4f3f0]'}`}>
                    <div>
                      <p className={featured ? 'text-[#a3a3a2]' : 'text-[#747878]'}>Requests / mo</p>
                      <p className="font-bold">{limitText(p.request_limit)}</p>
                    </div>
                    <div>
                      <p className={featured ? 'text-[#a3a3a2]' : 'text-[#747878]'}>Members</p>
                      <p className="font-bold">{limitText(p.user_limit)}</p>
                    </div>
                  </div>
                  <ul className="flex flex-1 flex-col gap-2">
                    {p.features.map((f) => (
                      <li key={f} className="flex items-start gap-2 text-[13px]">
                        <Icon name="check" size={15} className={featured ? 'mt-0.5 text-[#c7ef39]' : 'mt-0.5 text-[#526600]'} />
                        {f}
                      </li>
                    ))}
                  </ul>
                  {isAdmin && !isCurrent && (
                    <button
                      type="button"
                      onClick={() => setTarget(p)}
                      className={`${fontHeading} rounded-full px-4 py-2.5 text-[14px] font-semibold ${
                        featured ? 'bg-[#c7ef39] text-[#171e00] hover:bg-[#b5f546]' : 'bg-black text-white hover:bg-[#1c1b1b]'
                      }`}
                    >
                      Switch to {p.name}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </section>

      <Modal
        open={!!target}
        onClose={() => setTarget(null)}
        icon="workspace_premium"
        tone="success"
        title={`Switch to ${target?.name}?`}
        description={
          target &&
          `$${target.price}/month · ${limitText(target.request_limit)} requests per month · ${limitText(target.user_limit)} members. The change applies right away.`
        }
        footer={
          <>
            <button type="button" onClick={() => setTarget(null)} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
              Cancel
            </button>
            <button
              type="button"
              disabled={change.isPending}
              onClick={() => target && change.mutate(target)}
              className={`${fontHeading} rounded-full bg-black px-5 py-2 text-[14px] font-semibold text-white disabled:opacity-50`}
            >
              {change.isPending ? 'Switching…' : 'Confirm switch'}
            </button>
          </>
        }
      >
        {usage && target && target.user_limit > 0 && usage.user_count > target.user_limit && (
          <p className="flex items-start gap-2 rounded-xl bg-[#fff8e1] px-4 py-3 text-[13px] text-[#6b4e00]">
            <Icon name="warning" size={16} className="mt-0.5" />
            You have {usage.user_count} members; {target.name} allows {target.user_limit}.
          </p>
        )}
        {change.isError && <p className="text-[12px] font-medium text-[#ba1a1a]">{apiErrorMessage(change.error, 'Could not change the plan.')}</p>}
      </Modal>
    </div>
  );
}

function UsageBar({ label, used, limit, pct }: { label: string; used: number; limit: number; pct: number }) {
  const over = limit > 0 && used >= limit;
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline justify-between">
        <span className="text-[13px] font-medium text-[#444748]">{label}</span>
        <span className={`${fontHeading} text-[15px] font-bold ${over ? 'text-[#ba1a1a]' : ''}`}>
          {used.toLocaleString('en-US')} <span className="text-[12px] font-medium text-[#747878]">/ {limitText(limit)}</span>
        </span>
      </div>
      <div className="h-3 overflow-hidden rounded-full bg-[#efeeeb] p-0.5">
        <div
          className={`h-full rounded-full ${over ? 'bg-[#ba1a1a]' : pct >= 80 ? 'bg-[#f59e0b]' : 'bg-[#1b1c1a]'}`}
          style={{ width: `${limit === 0 ? 4 : Math.min(Math.max(pct, 2), 100)}%` }}
        />
      </div>
    </div>
  );
}

// Half-circle gauge (Figma "semi-gauge"); null = unlimited.
function SemiGauge({ pct }: { pct: number | null }) {
  const r = 80;
  const length = Math.PI * r;
  const value = pct === null ? 0 : Math.min(pct, 100);
  return (
    <div className="relative h-[112px] w-[192px]">
      <svg width="192" height="112" viewBox="0 0 192 112" aria-hidden>
        <path d="M16 96 A80 80 0 0 1 176 96" fill="none" stroke="#efeeeb" strokeWidth="16" strokeLinecap="round" />
        {pct !== null && (
          <path
            d="M16 96 A80 80 0 0 1 176 96"
            fill="none"
            stroke={value >= 100 ? '#ba1a1a' : value >= 80 ? '#f59e0b' : '#c7ef39'}
            strokeWidth="16"
            strokeLinecap="round"
            strokeDasharray={length}
            strokeDashoffset={length * (1 - value / 100)}
          />
        )}
      </svg>
      <p className={`${fontHeading} absolute inset-x-0 bottom-1 text-center text-[28px] font-bold`}>{pct === null ? '∞' : `${Math.round(value)}%`}</p>
    </div>
  );
}
