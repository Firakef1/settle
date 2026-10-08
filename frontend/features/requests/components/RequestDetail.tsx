'use client';

import Link from 'next/link';
import { useRef, useState, type FormEvent, type ReactNode } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { Modal } from '../../../shared/components/Modal';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { REQUEST_RULES, requestAPI } from '../../../shared/services/requestAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { Comment, Request, Role } from '../../../shared/types';
import { apiErrorMessage, getApiError } from '../../../shared/utils/apiError';
import { ROLES } from '../../../shared/utils/constants';
import { daysPending, formatDateOnly, formatISO, formatMoney } from '../../../shared/utils/format';
import { fontHeading, initials } from '../../../shared/utils/fonts';
import { useAddComment, useInvalidateRequests, useRequestDetail } from '../hooks/useRequests';
import { StatusPill, TYPE, UrgencyPill } from './pills';

const card = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const label = 'text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#747878]';

// Extra content for the right column, e.g. the finance decision panel (#35).
export interface RequestDetailProps {
  id: string;
  aside?: (request: Request, currency: string) => ReactNode;
}

export function RequestDetail({ id, aside }: RequestDetailProps) {
  const detail = useRequestDetail(id);
  const { user, currentOrg } = useAuthStore();
  const currency = useOrganization(currentOrg?.org_id).data?.currency ?? 'USD';

  if (detail.isPending) {
    return (
      <div className="flex flex-col gap-5" aria-busy="true">
        <div className="h-44 animate-pulse rounded-2xl bg-[#efeeeb]" />
        <div className="grid gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
          <div className="h-96 animate-pulse rounded-2xl bg-[#efeeeb]" />
          <div className="h-64 animate-pulse rounded-2xl bg-[#efeeeb]" />
        </div>
      </div>
    );
  }

  if (detail.isError) {
    const notFound = getApiError(detail.error).status === 404;
    return (
      <div className={`${card} flex flex-col items-center gap-3 py-16 text-center`}>
        <Icon name={notFound ? 'search_off' : 'error'} size={30} className="text-[#747878]" />
        <h1 className={`${fontHeading} text-[18px] font-bold`}>{notFound ? `${id} wasn't found` : "Couldn't load this request"}</h1>
        <p className="max-w-md text-[13px] text-[#444748]">
          {notFound
            ? "It doesn't exist in this organization, or it belongs to someone else."
            : apiErrorMessage(detail.error, 'Check your connection and try again.')}
        </p>
        <div className="flex gap-2">
          {!notFound && (
            <button type="button" onClick={() => detail.refetch()} className="rounded-full bg-[#efeeeb] px-4 py-2 text-[13px] font-semibold">
              Try again
            </button>
          )}
          <Link href="/requests" className="rounded-full bg-black px-4 py-2 text-[13px] font-semibold text-white">
            Back to requests
          </Link>
        </div>
      </div>
    );
  }

  const request = detail.data;
  const isOwner = request.requester.id === user?.id;

  return (
    <div className="flex flex-col gap-6">
      <Link href="/requests" className="flex w-fit items-center gap-1 text-[13px] text-[#444748] hover:text-[#1b1c1a]">
        <Icon name="arrow_back" size={14} />
        Back to requests
      </Link>

      <Banner request={request} currency={currency} />

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-5">
          <Summary request={request} currency={currency} />
          <Receipts request={request} canUpload={isOwner && request.status === 'draft'} />
          <Timeline request={request} />
          <Discussion request={request} currentUserId={user?.id} currentUserName={user?.name} />
        </div>
        <div className="flex flex-col gap-5 lg:sticky lg:top-24">
          {isOwner && <OwnerActions request={request} currency={currency} />}
          {aside?.(request, currency)}
          <NextSteps request={request} />
        </div>
      </div>
    </div>
  );
}

function Banner({ request, currency }: { request: Request; currency: string }) {
  const age = daysPending(request.submitted_at ?? request.created_at);
  return (
    <section className="relative overflow-hidden rounded-2xl bg-[#1b1c1c] p-7 text-white shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1),0px_8px_10px_-6px_rgba(0,0,0,0.1)]">
      <div aria-hidden className="pointer-events-none absolute -right-16 -top-24 size-80 rounded-full bg-[rgba(199,239,57,0.12)] blur-[48px]" />
      <div className="relative flex flex-wrap items-end justify-between gap-6">
        <div className="flex min-w-0 max-w-[620px] flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-[rgba(227,226,223,0.12)] px-2.5 py-0.5 font-mono text-[12px] font-semibold text-[#c7ef39]">{request.id}</span>
            <StatusPill status={request.status} />
            <span className="flex items-center gap-1 text-[12px] text-[#a3a3a2]">
              <Icon name={TYPE[request.type].icon} size={14} />
              {TYPE[request.type].label}
            </span>
          </div>
          <h1 className={`${fontHeading} text-[26px] font-bold leading-8 tracking-[-0.6px]`}>{request.purpose}</h1>
          <p className="text-[13px] text-[#a3a3a2]">
            Raised by <span className="font-semibold text-white">{request.requester.name}</span> on {formatDateOnly(request.created_at)}
          </p>
        </div>
        <dl className="grid grid-cols-3 gap-3">
          <Metric label="Amount" value={formatMoney(request.amount, currency)} />
          <Metric label="Urgency" value={request.urgency[0].toUpperCase() + request.urgency.slice(1)} />
          <Metric label={request.status === 'pending' ? 'Pending' : 'Age'} value={`${age}d`} />
        </dl>
      </div>
    </section>
  );
}

function Metric({ label: l, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl bg-[rgba(227,226,223,0.08)] px-3.5 py-2.5">
      <dt className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#848483]">{l}</dt>
      <dd className={`${fontHeading} whitespace-nowrap text-[17px] font-bold`}>{value}</dd>
    </div>
  );
}

function SectionTitle({ n, title, right }: { n: number; title: string; right?: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <h2 className={`${fontHeading} flex items-center gap-2.5 text-[16px] font-semibold`}>
        <span className="flex size-6 items-center justify-center rounded-full bg-black text-[12px] font-bold text-white">{n}</span>
        {title}
      </h2>
      {right}
    </div>
  );
}

function Summary({ request, currency }: { request: Request; currency: string }) {
  const amount = formatMoney(request.amount, currency);
  const cut = amount.lastIndexOf('.');
  return (
    <section className={card}>
      <SectionTitle n={1} title="Request summary" />
      <p className={`${fontHeading} mt-5 text-[40px] font-bold leading-[48px] tracking-[-1.2px] tabular-nums`}>
        {cut > 0 ? amount.slice(0, cut) : amount}
        <span className="text-[20px] text-[#747878]">{cut > 0 ? amount.slice(cut) : ''}</span>
      </p>
      <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3">
        <Field label="Type" value={TYPE[request.type].label} />
        <Field label="Urgency" value={<UrgencyPill urgency={request.urgency} />} />
        <Field label="Status" value={<StatusPill status={request.status} />} />
        <Field label="Requester" value={request.requester.name} sub={request.requester.email} />
        <Field label="Created" value={formatISO(request.created_at)} />
        <Field label="Submitted" value={request.submitted_at ? formatISO(request.submitted_at) : 'Not yet'} />
      </dl>
      <div className="mt-5 rounded-xl bg-[#f4f3f0] p-4">
        <p className={label}>Business purpose</p>
        <p className="mt-1 whitespace-pre-wrap text-[14px] leading-6 text-[#1b1c1a]">{request.purpose}</p>
      </div>
    </section>
  );
}

function Field({ label: l, value, sub }: { label: string; value: ReactNode; sub?: string }) {
  return (
    <div className="min-w-0">
      <dt className={label}>{l}</dt>
      <dd className="mt-1 text-[14px] font-medium text-[#1b1c1a]">{value}</dd>
      {sub && <dd className="truncate text-[12px] text-[#747878]">{sub}</dd>}
    </div>
  );
}

function fileName(path: string, requestId: string) {
  const base = path.split('/').pop() ?? path;
  return base.startsWith(`${requestId}_`) ? base.slice(requestId.length + 1) : base;
}

function Receipts({ request, canUpload }: { request: Request; canUpload: boolean }) {
  const invalidate = useInvalidateRequests();
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const upload = async (files: FileList | null) => {
    if (!files?.length) return;
    setError('');
    setBusy(true);
    try {
      for (const file of Array.from(files)) {
        if (!(REQUEST_RULES.receiptTypes as readonly string[]).includes(file.type)) throw new Error(`${file.name}: only PDF, PNG and JPG files are accepted.`);
        if (file.size > REQUEST_RULES.maxReceiptBytes) throw new Error(`${file.name} is larger than 10 MB.`);
        await requestAPI.uploadReceipt(request.id, file);
      }
    } catch (err) {
      setError(err instanceof Error && !('response' in err) ? err.message : apiErrorMessage(err, 'Upload failed.'));
    } finally {
      await invalidate();
      setBusy(false);
    }
  };

  return (
    <section className={card}>
      <SectionTitle
        n={2}
        title="Receipts"
        right={
          canUpload && (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={() => input.current?.click()}
                className="flex items-center gap-1 rounded-full bg-[#c7ef39] px-3 py-1.5 text-[12px] font-semibold text-[#171e00] disabled:opacity-50"
              >
                <Icon name="upload_file" size={14} />
                {busy ? 'Uploading…' : 'Add receipt'}
              </button>
              <input
                ref={input}
                type="file"
                multiple
                accept="application/pdf,image/png,image/jpeg"
                className="sr-only"
                onChange={(e) => {
                  upload(e.target.files);
                  e.target.value = '';
                }}
              />
            </>
          )
        }
      />
      {error && (
        <p role="alert" className="mt-3 flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a]">
          <Icon name="error" size={14} />
          {error}
        </p>
      )}
      {request.receipts.length === 0 ? (
        <p className="mt-4 rounded-xl bg-[#f4f3f0] px-4 py-3 text-[13px] text-[#444748]">
          {request.type === 'reimbursement'
            ? canUpload
              ? 'No receipts yet. Add at least one before submitting.'
              : 'No receipts attached.'
            : `${TYPE[request.type].label}s don't need receipts.`}
        </p>
      ) : (
        <ul className="mt-4 flex flex-col gap-2">
          {request.receipts.map((r) => {
            const name = fileName(r.file_path, request.id);
            return (
              <li key={r.id} className="flex items-center justify-between gap-3 rounded-xl bg-[#f4f3f0] px-4 py-3">
                <span className="flex min-w-0 items-center gap-3">
                  <Icon name={name.toLowerCase().endsWith('.pdf') ? 'picture_as_pdf' : 'image'} size={20} className="text-[#ba1a1a]" />
                  <span className="min-w-0">
                    <span className="block truncate text-[13px] font-semibold">{name}</span>
                    <span className="block text-[12px] text-[#747878]">Uploaded {formatISO(r.created_at)}</span>
                  </span>
                </span>
                <span className="whitespace-nowrap rounded-full bg-[#e3e2df] px-2 py-0.5 text-[11px] font-semibold text-[#444748]">
                  OCR {r.ocr_status}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

interface Step {
  title: string;
  at?: string | null;
  state: 'done' | 'current' | 'upcoming' | 'stopped';
  note?: string;
}

function reviewAge(days: number) {
  return days === 0 ? 'In review since today' : `In review for ${days} ${days === 1 ? 'day' : 'days'}`;
}

// The API returns timeline: null, so build it from the dates and status we have.
function buildTimeline(r: Request): Step[] {
  const steps: Step[] = [{ title: r.type === 'reimbursement' ? 'Draft created' : 'Request created', at: r.created_at, state: 'done' }];
  const submitted: Step = { title: 'Submitted to finance', at: r.submitted_at, state: r.submitted_at ? 'done' : 'upcoming' };
  switch (r.status) {
    case 'draft':
      steps.push({ ...submitted, state: 'current', note: 'Waiting for you to submit' }, { title: 'Finance decision', state: 'upcoming' }, { title: 'Paid', state: 'upcoming' });
      break;
    case 'pending':
      steps.push(submitted, { title: 'Finance decision', state: 'current', note: reviewAge(daysPending(r.submitted_at ?? r.created_at)) }, { title: 'Paid', state: 'upcoming' });
      break;
    case 'approved':
      steps.push(submitted, { title: 'Approved', at: r.updated_at, state: 'done' }, { title: 'Paid', state: 'current', note: 'Waiting for finance to record the payment' });
      break;
    case 'paid':
      steps.push(submitted, { title: 'Approved', state: 'done' }, { title: 'Paid', at: r.updated_at, state: 'done' });
      break;
    case 'rejected':
      steps.push(submitted, { title: 'Rejected', at: r.updated_at, state: 'stopped', note: 'See the discussion for the reason. You can resubmit.' });
      break;
    case 'failed':
      steps.push(submitted, { title: 'Approved', state: 'done' }, { title: 'Payment failed', at: r.updated_at, state: 'stopped', note: 'You can resubmit with corrected details.' });
      break;
    case 'withdrawn':
      steps.push({ title: 'Withdrawn', at: r.updated_at, state: 'stopped' });
      break;
  }
  return steps;
}

function Timeline({ request }: { request: Request }) {
  const steps = buildTimeline(request);
  const dot = {
    done: 'bg-[#526600] text-white',
    current: 'bg-[#c7ef39] text-[#171e00] ring-4 ring-[rgba(199,239,57,0.3)]',
    upcoming: 'bg-[#efeeeb] text-[#747878]',
    stopped: 'bg-[#ba1a1a] text-white',
  };
  const icon = { done: 'check', current: 'more_horiz', upcoming: 'radio_button_unchecked', stopped: 'close' };
  return (
    <section className={card}>
      <SectionTitle n={3} title="Lifecycle" />
      <ol className="mt-5 flex flex-col">
        {steps.map((s, i) => (
          <li key={s.title} className="relative flex gap-4 pb-6 last:pb-0">
            {i < steps.length - 1 && <span aria-hidden className="absolute left-[13px] top-7 h-[calc(100%-28px)] w-0.5 bg-[#efeeeb]" />}
            <span className={`relative flex size-7 shrink-0 items-center justify-center rounded-full ${dot[s.state]}`}>
              <Icon name={icon[s.state]} size={15} />
            </span>
            <div className="pt-0.5">
              <p className={`text-[14px] font-semibold ${s.state === 'upcoming' ? 'text-[#747878]' : s.state === 'stopped' ? 'text-[#ba1a1a]' : 'text-[#1b1c1a]'}`}>{s.title}</p>
              {s.at && <p className="text-[12px] text-[#747878]">{formatISO(s.at)}</p>}
              {s.note && <p className="mt-0.5 text-[13px] text-[#444748]">{s.note}</p>}
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

function authorName(c: Comment, currentUserId?: string, currentUserName?: string) {
  if (c.author_id === currentUserId) return currentUserName ? `${currentUserName} (you)` : 'You';
  return c.author || c.author_name || 'Team member';
}

function Discussion({ request, currentUserId, currentUserName }: { request: Request; currentUserId?: string; currentUserName?: string }) {
  const add = useAddComment(request.id);
  const [text, setText] = useState('');

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const content = text.trim();
    if (!content) return;
    add.mutate(content, { onSuccess: () => setText('') });
  };

  return (
    <section className={card}>
      <SectionTitle n={4} title="Discussion" right={<span className={label}>{request.comments.length} {request.comments.length === 1 ? 'comment' : 'comments'}</span>} />
      {request.comments.length === 0 ? (
        <p className="mt-4 text-[13px] text-[#444748]">No comments yet. Ask a question or add context for the approver.</p>
      ) : (
        <ul className="mt-5 flex flex-col gap-4">
          {request.comments.map((c) => {
            const mine = c.author_id === currentUserId;
            const name = authorName(c, currentUserId, currentUserName);
            return (
              <li key={c.id} className="flex gap-3">
                <span className={`${fontHeading} flex size-8 shrink-0 items-center justify-center rounded-full text-[11px] font-bold ${mine ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-[#1b1c1a] text-white'}`}>
                  {initials(c.author_id === currentUserId ? currentUserName : name)}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="flex flex-wrap items-center gap-2 text-[13px]">
                    <span className="font-semibold">{name}</span>
                    {c.role && <span className="rounded-full bg-[#efeeeb] px-2 py-0.5 text-[11px] font-semibold text-[#444748]">{ROLES[c.role as Role] ?? c.role}</span>}
                    <span className="text-[12px] text-[#747878]">{formatISO(c.created_at)}</span>
                  </p>
                  <p className="mt-1 whitespace-pre-wrap rounded-xl rounded-tl-sm bg-[#f4f3f0] px-3.5 py-2.5 text-[14px] leading-5">{c.content || c.text}</p>
                </div>
              </li>
            );
          })}
        </ul>
      )}
      <form onSubmit={submit} className="mt-5 flex flex-col gap-2">
        <label htmlFor="comment" className="sr-only">
          Add a comment
        </label>
        <textarea
          id="comment"
          rows={2}
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Write a comment…"
          className="w-full resize-y rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
        />
        <div className="flex items-center justify-between gap-2">
          <span className="text-[12px] text-[#ba1a1a]" role="alert">
            {add.isError && apiErrorMessage(add.error, 'Could not post the comment.')}
          </span>
          <button
            type="submit"
            disabled={!text.trim() || add.isPending}
            className={`${fontHeading} rounded-full bg-black px-4 py-2 text-[13px] font-semibold text-white disabled:opacity-40`}
          >
            {add.isPending ? 'Posting…' : 'Post comment'}
          </button>
        </div>
      </form>
    </section>
  );
}

function OwnerActions({ request, currency }: { request: Request; currency: string }) {
  const invalidate = useInvalidateRequests();
  const [withdrawOpen, setWithdrawOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      await requestAPI.submit(request.id);
      await invalidate();
    } catch (err) {
      setError(apiErrorMessage(err, 'Could not submit.'));
    } finally {
      setBusy(false);
    }
  };

  const needsReceipt = request.type === 'reimbursement' && request.receipts.length === 0;
  const canWithdraw = request.status === 'draft' || request.status === 'pending';
  const canResubmit = request.status === 'rejected' || request.status === 'failed';

  return (
    <section className={card}>
      <p className={label}>Your actions</p>
      <div className="mt-4 flex flex-col gap-2">
        {request.status === 'draft' && (
          <>
            <button
              type="button"
              onClick={submit}
              disabled={busy || needsReceipt}
              className={`${fontHeading} flex items-center justify-center gap-1.5 rounded-full bg-black px-4 py-2.5 text-[14px] font-semibold text-white disabled:opacity-40`}
            >
              <Icon name="send" size={15} />
              {busy ? 'Submitting…' : 'Submit for review'}
            </button>
            {needsReceipt && <p className="text-center text-[12px] text-[#444748]">Add a receipt first.</p>}
          </>
        )}
        {canResubmit && (
          <Link
            href={`/requests/${request.id}/resubmit`}
            className={`${fontHeading} flex items-center justify-center gap-1.5 rounded-full bg-[#ba1a1a] px-4 py-2.5 text-[14px] font-semibold text-white hover:bg-[#93000a]`}
          >
            <Icon name="redo" size={15} />
            Resubmit as a new request
          </Link>
        )}
        {canWithdraw && (
          <button
            type="button"
            onClick={() => setWithdrawOpen(true)}
            className="flex items-center justify-center gap-1.5 rounded-full bg-[#efeeeb] px-4 py-2.5 text-[14px] font-semibold text-[#1b1c1a] hover:bg-[#e3e2df]"
          >
            <Icon name="undo" size={15} />
            Withdraw request
          </button>
        )}
        {!canWithdraw && !canResubmit && request.status !== 'draft' && (
          <p className="text-[13px] text-[#444748]">Nothing to do right now.</p>
        )}
        {error && (
          <p role="alert" className="text-[12px] font-medium text-[#ba1a1a]">
            {error}
          </p>
        )}
      </div>
      <WithdrawDialog open={withdrawOpen} onClose={() => setWithdrawOpen(false)} request={request} currency={currency} />
    </section>
  );
}

function WithdrawDialog({ open, onClose, request, currency }: { open: boolean; onClose: () => void; request: Request; currency: string }) {
  const invalidate = useInvalidateRequests();
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const confirm = async () => {
    setBusy(true);
    setError('');
    try {
      // The withdraw call has no body; keep the reason as a comment so finance can see it.
      if (reason.trim()) await requestAPI.addComment(request.id, `Withdrawn: ${reason.trim()}`);
      await requestAPI.withdraw(request.id);
      await invalidate();
      onClose();
    } catch (err) {
      setError(apiErrorMessage(err, 'Could not withdraw the request.'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      icon="undo"
      tone="danger"
      title="Withdraw this request?"
      description="It leaves the finance queue and can't be reopened. You can always raise a new request."
      footer={
        <>
          <button type="button" onClick={onClose} className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
            Keep request
          </button>
          <button
            type="button"
            onClick={confirm}
            disabled={busy}
            className={`${fontHeading} rounded-full bg-[#ba1a1a] px-4 py-2 text-[14px] font-semibold text-white hover:bg-[#93000a] disabled:opacity-50`}
          >
            {busy ? 'Withdrawing…' : 'Withdraw request'}
          </button>
        </>
      }
    >
      <div className="flex items-center justify-between gap-3 rounded-xl bg-[#f6f6f4] px-4 py-3">
        <div className="min-w-0">
          <p className="font-mono text-[12px] font-semibold text-[#444748]">{request.id}</p>
          <p className="truncate text-[14px] font-medium">{request.purpose}</p>
        </div>
        <p className={`${fontHeading} whitespace-nowrap text-[16px] font-bold`}>{formatMoney(request.amount, currency)}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <label htmlFor="withdraw-reason" className={label}>
          Reason (optional)
        </label>
        <textarea
          id="withdraw-reason"
          rows={3}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="e.g. Duplicate of another request"
          className="w-full resize-y rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
        />
        <p className="text-[12px] text-[#747878]">Posted as a comment so finance can see why.</p>
      </div>
      {error && (
        <p role="alert" className="text-[12px] font-medium text-[#ba1a1a]">
          {error}
        </p>
      )}
    </Modal>
  );
}

function NextSteps({ request }: { request: Request }) {
  const text: Record<Request['status'], string> = {
    draft: 'Drafts are only visible to you. Submit to send it to the finance queue.',
    pending: 'Finance reviews pending requests oldest first. Anything older than 7 days is flagged for attention.',
    approved: 'Approved. Finance records the payment once it has gone out.',
    paid: 'Paid and closed. Nothing else to do.',
    rejected: 'Rejected requests can be resubmitted with changes. Your receipts can be carried over.',
    failed: 'The payment did not go through. Resubmit with corrected details.',
    withdrawn: 'Withdrawn requests are closed and kept for the record.',
  };
  return (
    <section className="rounded-2xl bg-[#f4f3f0] p-5">
      <p className="flex items-center gap-1.5 text-[12px] font-semibold uppercase tracking-[0.55px] text-[#444748]">
        <Icon name="info" size={14} />
        What happens next
      </p>
      <p className="mt-2 text-[13px] leading-5 text-[#444748]">{text[request.status]}</p>
    </section>
  );
}
