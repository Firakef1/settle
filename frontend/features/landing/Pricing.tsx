'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { Icon } from '../../shared/components/Icon';
import { orgAPI } from '../../shared/services/orgAPI';
import { fontHeading } from '../../shared/utils/fonts';

const limit = (n: number) => (n === 0 ? 'Unlimited' : n.toLocaleString('en-US'));

// Plans come from the public GET /billing/plans, so the page always matches billing.
export function Pricing() {
  const plans = useQuery({ queryKey: ['billing', 'plans'], queryFn: orgAPI.listPlans, staleTime: 10 * 60 * 1000 });

  if (plans.isPending) {
    return (
      <div className="mt-12 grid gap-5 md:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-96 animate-pulse rounded-2xl bg-[#f4f3f0]" />
        ))}
      </div>
    );
  }
  if (plans.isError) {
    return (
      <p className="mt-10 text-center text-[14px] text-[#444748]">
        Plans couldn&apos;t load right now.{' '}
        <Link href="/signup" className="font-semibold underline">
          Start free
        </Link>{' '}
        and upgrade any time.
      </p>
    );
  }

  return (
    <div className="mt-12 grid items-stretch gap-5 md:grid-cols-3">
      {plans.data.map((p) => {
        const featured = p.id === 'starter';
        return (
          <div
            key={p.id}
            className={`flex flex-col gap-5 rounded-2xl p-7 ${featured ? 'bg-[#1b1c1c] text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.12)]' : 'bg-[#faf9f6]'}`}
          >
            <div className="flex items-center justify-between">
              <h3 className={`${fontHeading} text-[20px] font-bold`}>{p.name}</h3>
              {featured && <span className="rounded-full bg-[#c7ef39] px-2.5 py-0.5 text-[11px] font-bold text-[#171e00]">Most popular</span>}
            </div>
            <p className={`${fontHeading} text-[40px] font-bold leading-none tracking-[-1px]`}>
              ${p.price}
              <span className={`text-[14px] font-medium ${featured ? 'text-[#a3a3a2]' : 'text-[#747878]'}`}> / month</span>
            </p>
            <p className={`text-[13px] ${featured ? 'text-[#d6d6d5]' : 'text-[#444748]'}`}>
              {limit(p.request_limit)} requests / month · {limit(p.user_limit)} members
            </p>
            <ul className="flex flex-1 flex-col gap-2.5">
              {p.features.map((f) => (
                <li key={f} className="flex items-start gap-2 text-[14px]">
                  <Icon name="check" size={16} className={featured ? 'mt-0.5 text-[#c7ef39]' : 'mt-0.5 text-[#526600]'} />
                  {f}
                </li>
              ))}
            </ul>
            <Link
              href="/signup"
              className={`${fontHeading} rounded-full px-5 py-3 text-center text-[15px] font-semibold ${
                featured ? 'bg-[#c7ef39] text-[#171e00] hover:bg-[#b5f546]' : 'bg-black text-white hover:bg-[#1c1b1b]'
              }`}
            >
              {p.price === 0 ? 'Start free' : `Choose ${p.name}`}
            </Link>
          </div>
        );
      })}
    </div>
  );
}
