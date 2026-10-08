import Image from 'next/image';
import Link from 'next/link';
import type { ReactNode } from 'react';
import { Icon } from '../../shared/components/Icon';
import { fontBody, fontHeading } from '../../shared/utils/fonts';
import { Pricing } from './Pricing';

const STEPS = [
  { icon: 'receipt_long', title: 'Submit', body: 'Staff raise a reimbursement, advance or stipend and drop in the receipt.' },
  { icon: 'document_scanner', title: 'Read', body: 'OCR pulls the merchant, date and total from the receipt so finance can check it fast.' },
  { icon: 'check_circle', title: 'Approve', body: 'Finance works one queue, oldest first, with anything over 7 days flagged.' },
  { icon: 'payments', title: 'Pay & record', body: 'Mark it paid with the method used. Every step lands in the audit log.' },
];

const ROLES: Array<{ icon: string; role: string; points: string[]; dark?: boolean }> = [
  {
    icon: 'badge',
    role: 'Staff',
    points: ['Raise requests in a minute', 'Track every request from draft to paid', 'Resubmit with fixes when something is rejected'],
  },
  {
    icon: 'query_stats',
    role: 'Finance',
    points: ['One approval queue with aging and urgency', 'Approve, reject, mark paid or failed', 'Dashboard of spend, risk and backlog'],
    dark: true,
  },
  {
    icon: 'shield_person',
    role: 'Admin',
    points: ['Invite people with a link', 'Set roles and manage the plan', 'Searchable audit log of every decision'],
  },
];

export function Landing() {
  return (
    <div className={`${fontBody} min-h-screen bg-[#faf9f6] text-[#1b1c1a]`}>
      <header className="sticky top-0 z-30 border-b border-[#efeeeb] bg-[rgba(250,249,246,0.9)] backdrop-blur-[12px]">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-5 sm:px-8">
          <Link href="/" className="flex items-center gap-2.5">
            <Image src="/brand/settle-logo.jpg" alt="" width={32} height={32} className="size-8 rounded-lg" />
            <span className={`${fontHeading} text-[18px] font-bold tracking-[-0.4px]`}>Settle</span>
          </Link>
          <nav aria-label="Sections" className="hidden items-center gap-7 text-[14px] text-[#444748] md:flex">
            <a href="#how" className="hover:text-[#1b1c1a]">How it works</a>
            <a href="#roles" className="hover:text-[#1b1c1a]">For your team</a>
            <a href="#pricing" className="hover:text-[#1b1c1a]">Pricing</a>
          </nav>
          <div className="flex items-center gap-2">
            <Link href="/login" className="rounded-full px-4 py-2 text-[14px] font-medium hover:bg-[#efeeeb]">
              Log in
            </Link>
            <Link href="/signup" className={`${fontHeading} rounded-full bg-black px-4 py-2 text-[14px] font-semibold text-white hover:bg-[#1c1b1b]`}>
              Get started
            </Link>
          </div>
        </div>
      </header>

      <main>
        <section className="mx-auto grid max-w-6xl items-center gap-12 px-5 pb-20 pt-16 sm:px-8 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)] lg:pt-24">
          <div className="flex flex-col gap-6">
            <span className="flex w-fit items-center gap-2 rounded-full bg-[#c7ef39] px-3 py-1 text-[12px] font-semibold text-[#171e00]">
              <span className="size-1.5 rounded-full bg-[#171e00]" />
              Payouts & expense requests
            </span>
            <h1 className={`${fontHeading} isolate text-[44px] font-bold leading-[1.05] tracking-[-1.6px] sm:text-[56px]`}>
              Every payout,
              <br />
              <span className="relative inline-block">
                settled
                <span aria-hidden className="absolute inset-x-0 bottom-1 -z-10 h-4 rounded-sm bg-[#c7ef39]" />
              </span>{' '}
              on time.
            </h1>
            <p className="max-w-[520px] text-[17px] leading-7 text-[#444748]">
              Settle replaces expense spreadsheets and email threads. Staff submit with receipts, finance approves from one queue, and every decision is
              on the record.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link
                href="/signup"
                className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-black px-6 py-3 text-[16px] font-semibold text-white hover:bg-[#1c1b1b]`}
              >
                Create your workspace
                <Icon name="arrow_forward" size={17} className="text-[#c7ef39]" />
              </Link>
              <Link href="/login" className="rounded-full bg-white px-6 py-3 text-[16px] font-medium shadow-[0px_1px_3px_0px_rgba(0,0,0,0.06)] hover:bg-[#f4f3f0]">
                I have an account
              </Link>
            </div>
            <p className="flex items-center gap-1.5 text-[13px] text-[#747878]">
              <Icon name="check" size={15} className="text-[#526600]" />
              Free for teams up to 5. No card needed.
            </p>
          </div>
          <HeroPreview />
        </section>

        <section id="how" className="border-y border-[#efeeeb] bg-white">
          <div className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
            <SectionHeading eyebrow="How it works" title="From receipt to paid in four steps" />
            <ol className="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
              {STEPS.map((s, i) => (
                <li key={s.title} className="relative flex flex-col gap-3 rounded-2xl bg-[#faf9f6] p-6">
                  <span className="flex items-center justify-between">
                    <span className="flex size-10 items-center justify-center rounded-xl bg-black text-white">
                      <Icon name={s.icon} size={19} />
                    </span>
                    <span className={`${fontHeading} text-[32px] font-bold text-[#e3e2df]`}>0{i + 1}</span>
                  </span>
                  <h3 className={`${fontHeading} text-[18px] font-bold`}>{s.title}</h3>
                  <p className="text-[14px] leading-6 text-[#444748]">{s.body}</p>
                </li>
              ))}
            </ol>
          </div>
        </section>

        <section id="roles" className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
          <SectionHeading eyebrow="For your team" title="One workspace, three roles" />
          <div className="mt-12 grid gap-5 md:grid-cols-3">
            {ROLES.map((r) => (
              <div
                key={r.role}
                className={`relative flex flex-col gap-4 overflow-hidden rounded-2xl p-7 ${
                  r.dark ? 'bg-[#1b1c1c] text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1)]' : 'bg-white shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]'
                }`}
              >
                {r.dark && <div aria-hidden className="pointer-events-none absolute -right-12 -top-12 size-40 rounded-full bg-[rgba(199,239,57,0.12)] blur-[32px]" />}
                <span className={`relative flex size-11 items-center justify-center rounded-xl ${r.dark ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-[#efeeeb]'}`}>
                  <Icon name={r.icon} size={21} />
                </span>
                <h3 className={`${fontHeading} relative text-[22px] font-bold`}>{r.role}</h3>
                <ul className="relative flex flex-col gap-2.5">
                  {r.points.map((p) => (
                    <li key={p} className={`flex items-start gap-2 text-[14px] ${r.dark ? 'text-[#d6d6d5]' : 'text-[#444748]'}`}>
                      <Icon name="check" size={16} className={r.dark ? 'mt-0.5 text-[#c7ef39]' : 'mt-0.5 text-[#526600]'} />
                      {p}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </section>

        <section id="pricing" className="border-t border-[#efeeeb] bg-white">
          <div className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
            <SectionHeading eyebrow="Pricing" title="Start free, upgrade when you grow" />
            <Pricing />
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
          <div className="relative overflow-hidden rounded-3xl bg-[#1b1c1c] px-8 py-14 text-center text-white sm:px-14">
            <div aria-hidden className="pointer-events-none absolute -left-16 -top-24 size-72 rounded-full bg-[rgba(199,239,57,0.14)] blur-[48px]" />
            <h2 className={`${fontHeading} relative text-[34px] font-bold tracking-[-1px]`}>Ready to clear the queue?</h2>
            <p className="relative mx-auto mt-3 max-w-lg text-[16px] text-[#a3a3a2]">Set up your organization and invite your finance team in a couple of minutes.</p>
            <div className="relative mt-7 flex flex-wrap justify-center gap-3">
              <Link href="/signup" className={`${fontHeading} rounded-full bg-[#c7ef39] px-6 py-3 text-[16px] font-semibold text-[#171e00] hover:bg-[#b5f546]`}>
                Get started free
              </Link>
              <Link href="/login" className="rounded-full bg-[rgba(227,226,223,0.1)] px-6 py-3 text-[16px] font-medium hover:bg-[rgba(227,226,223,0.18)]">
                Log in
              </Link>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-[#efeeeb]">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-5 py-6 text-[13px] text-[#747878] sm:px-8">
          <span className="flex items-center gap-2">
            <span className="size-1.5 rounded-full bg-[#c7ef39]" />© {new Date().getFullYear()} Settle
          </span>
          <span>Payout & expense request management</span>
        </div>
      </footer>
    </div>
  );
}

function SectionHeading({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="flex flex-col items-center gap-3 text-center">
      <span className="text-[12px] font-semibold uppercase tracking-[1px] text-[#526600]">{eyebrow}</span>
      <h2 className={`${fontHeading} max-w-2xl text-[34px] font-bold leading-tight tracking-[-1px]`}>{title}</h2>
    </div>
  );
}

// A static illustration of the product in the app's own visual language.
function HeroPreview() {
  return (
    <div aria-hidden className="relative">
      <div className="absolute -right-6 -top-6 size-64 rounded-full bg-[rgba(199,239,57,0.25)] blur-[60px]" />
      <div className="relative flex flex-col gap-4">
        <div className="grid grid-cols-3 gap-3">
          <MiniKpi label="Pending" value="23" foot="9 urgent" />
          <MiniKpi label="Paid this month" value="$118k" foot="38 settled" />
          <MiniKpi label="At risk" value="7" foot="7+ days" danger />
        </div>
        <div className="rounded-2xl bg-white p-5 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.06)]">
          <div className="flex items-center justify-between">
            <span className={`${fontHeading} text-[15px] font-bold`}>Approvals queue</span>
            <span className="rounded-full bg-[#c7ef39] px-2 py-0.5 text-[11px] font-bold text-[#171e00]">23</span>
          </div>
          <div className="mt-3 flex flex-col">
            <QueueRow id="REQ-2796" who="Elena Rostova" amount="$1,890.00" days="16d" critical />
            <QueueRow id="REQ-2803" who="Carlos Mendez" amount="$1,340.00" days="14d" critical />
            <QueueRow id="REQ-2811" who="Marcus Lee" amount="$2,450.00" days="11d" />
          </div>
        </div>
        <div className="relative ml-auto w-[78%] overflow-hidden rounded-2xl bg-[#1b1c1c] p-5 text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.12)]">
          <p className="flex items-center gap-1.5 text-[12px] font-semibold uppercase tracking-[0.55px] text-[#848483]">
            <Icon name="bolt" size={14} className="text-[#c7ef39]" />
            Decision
          </p>
          <p className={`${fontHeading} mt-1 text-[18px] font-bold`}>REQ-2811 approved</p>
          <p className="mt-1 text-[12px] text-[#a3a3a2]">Logged to the audit trail · 2s ago</p>
        </div>
      </div>
    </div>
  );
}

function MiniKpi({ label, value, foot, danger }: { label: string; value: string; foot: string; danger?: boolean }) {
  return (
    <div className="relative overflow-hidden rounded-xl bg-white p-3.5 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.06)]">
      <div className={`absolute inset-x-0 top-0 h-1 ${danger ? 'bg-[#ba1a1a]' : 'bg-[#efeeeb]'}`} />
      <p className={`text-[10px] font-semibold uppercase tracking-[0.5px] ${danger ? 'text-[#ba1a1a]' : 'text-[#444748]'}`}>{label}</p>
      <p className={`${fontHeading} mt-1 text-[24px] font-bold ${danger ? 'text-[#ba1a1a]' : ''}`}>{value}</p>
      <p className="text-[11px] text-[#747878]">{foot}</p>
    </div>
  );
}

function QueueRow({ id, who, amount, days, critical }: { id: string; who: string; amount: string; days: string; critical?: boolean }): ReactNode {
  return (
    <div className="flex items-center justify-between gap-3 border-t border-[#efeeeb] py-2.5 first:border-t-0">
      <span className="flex items-center gap-2">
        <span className={`size-2 rounded-full ${critical ? 'bg-[#ba1a1a]' : 'bg-[#526600]'}`} />
        <span className="text-[13px] font-bold">{id}</span>
        <span className="hidden text-[13px] text-[#444748] sm:inline">{who}</span>
      </span>
      <span className="flex items-center gap-2">
        <span className="font-mono text-[13px] font-bold">{amount}</span>
        <span className={`rounded-full px-2 py-0.5 text-[11px] font-bold ${critical ? 'bg-[#ba1a1a] text-white' : 'bg-[#e3e2df]'}`}>{days}</span>
      </span>
    </div>
  );
}
