'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useRef, useState, type DragEvent, type FormEvent, type ReactNode } from 'react';
import { Icon } from '../../../shared/components/Icon';
import { useOrganization } from '../../../shared/hooks/useOrganization';
import { REQUEST_RULES, requestAPI } from '../../../shared/services/requestAPI';
import { useAuthStore } from '../../../shared/stores/authStore';
import type { RequestType, Urgency } from '../../../shared/types';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { fontHeading } from '../../../shared/utils/fonts';
import { useInvalidateRequests } from '../hooks/useRequests';
import { TYPE } from './pills';

const TYPES: RequestType[] = ['reimbursement', 'advance', 'stipend'];
const URGENCY_OPTIONS: Array<{ value: Urgency; label: string; hint: string }> = [
  { value: 'routine', label: 'Routine', hint: 'Normal review cycle' },
  { value: 'urgent', label: 'Urgent', hint: 'Needed within days' },
  { value: 'critical', label: 'Critical', hint: 'Blocking work now' },
];
const cardClass = 'rounded-2xl bg-white p-6 shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]';
const labelClass = 'text-[11px] font-semibold uppercase leading-4 tracking-[0.55px] text-[#444748]';

interface PickedFile {
  key: string;
  file: File;
}

function formatBytes(bytes: number) {
  return bytes >= 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export function NewRequestForm() {
  const router = useRouter();
  const invalidate = useInvalidateRequests();
  const { currentOrg } = useAuthStore();
  const currency = useOrganization(currentOrg?.org_id).data?.currency ?? 'USD';
  const fileInput = useRef<HTMLInputElement>(null);

  const [type, setType] = useState<RequestType>('reimbursement');
  const [amount, setAmount] = useState('');
  const [urgency, setUrgency] = useState<Urgency>('routine');
  const [purpose, setPurpose] = useState('');
  const [notes, setNotes] = useState('');
  const [files, setFiles] = useState<PickedFile[]>([]);
  const [fileError, setFileError] = useState('');
  const [dragging, setDragging] = useState(false);
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState<'draft' | 'submit' | null>(null);
  const [error, setError] = useState('');
  // Survive a partial failure: reuse the created draft and skip uploaded files on retry.
  const [draftId, setDraftId] = useState<string | null>(null);
  const [uploaded, setUploaded] = useState<Set<string>>(new Set());

  const isReimbursement = type === 'reimbursement';
  const amountValue = Number(amount);
  const amountError =
    !amount || !Number.isFinite(amountValue) || amountValue < REQUEST_RULES.minAmount
      ? 'Enter an amount greater than 0.'
      : amountValue > REQUEST_RULES.maxAmount
        ? `The maximum is ${REQUEST_RULES.maxAmount.toLocaleString('en-US')}.`
        : '';
  const purposeLength = purpose.trim().length;
  const purposeError =
    purposeLength < REQUEST_RULES.minPurpose
      ? `Describe the purpose in at least ${REQUEST_RULES.minPurpose} characters.`
      : purposeLength > REQUEST_RULES.maxPurpose
        ? `Keep it under ${REQUEST_RULES.maxPurpose} characters.`
        : '';
  const receiptError = isReimbursement && files.length === 0 ? 'Attach at least one receipt to submit a reimbursement.' : '';

  const addFiles = (list: FileList | null) => {
    if (!list) return;
    setFileError('');
    const next: PickedFile[] = [];
    for (const file of Array.from(list)) {
      if (!(REQUEST_RULES.receiptTypes as readonly string[]).includes(file.type)) {
        setFileError(`${file.name}: only PDF, PNG and JPG files are accepted.`);
        continue;
      }
      if (file.size > REQUEST_RULES.maxReceiptBytes) {
        setFileError(`${file.name} is larger than 10 MB.`);
        continue;
      }
      next.push({ key: `${file.name}-${file.size}-${file.lastModified}`, file });
    }
    setFiles((prev) => [...prev, ...next.filter((n) => !prev.some((p) => p.key === n.key))]);
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
    addFiles(e.dataTransfer.files);
  };

  const run = async (mode: 'draft' | 'submit') => {
    setTouched(true);
    setError('');
    if (amountError || purposeError || (mode === 'submit' && receiptError)) return;

    setBusy(mode);
    try {
      let id = draftId;
      if (!id) {
        const created = await requestAPI.create({ type, amount: amountValue, purpose: purpose.trim(), urgency });
        id = created.id;
        setDraftId(id);
      }
      if (isReimbursement) {
        const done = new Set(uploaded);
        for (const { key, file } of files) {
          if (done.has(key)) continue;
          await requestAPI.uploadReceipt(id, file);
          done.add(key);
          setUploaded(new Set(done));
        }
        if (mode === 'submit') await requestAPI.submit(id);
      }
      if (notes.trim()) {
        try {
          await requestAPI.addComment(id, notes.trim());
        } catch {
          // The request exists; a failed note shouldn't block. It can be re-added on the detail page.
        }
      }
      await invalidate();
      router.push(`/requests/${id}`);
    } catch (err) {
      setError(
        draftId || uploaded.size
          ? `${apiErrorMessage(err, 'Something went wrong.')} Your draft was saved — try again to finish.`
          : apiErrorMessage(err, 'Could not create the request.'),
      );
      setBusy(null);
    }
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    run('submit');
  };

  return (
    <form onSubmit={onSubmit} className="mx-auto flex w-full max-w-[720px] flex-col gap-6 pb-28" noValidate>
      <div className="flex flex-col gap-2">
        <Link href="/requests" className="flex w-fit items-center gap-1 text-[13px] text-[#444748] hover:text-[#1b1c1a]">
          <Icon name="arrow_back" size={14} />
          Back to My Requests
        </Link>
        <h1 className={`${fontHeading} text-[28px] font-bold leading-9 tracking-[-0.7px]`}>New Payout Request</h1>
        <p className="text-[14px] leading-5 text-[#444748]">
          {isReimbursement
            ? 'Reimbursements need at least one receipt. You can save a draft and add receipts later.'
            : 'Advances and stipends go straight to the finance queue when you submit.'}
        </p>
      </div>

      <section className={cardClass} aria-labelledby="details-heading">
        <SectionHeading n={1} id="details-heading" title="Request type & amount" note="Required" />

        <fieldset className="mt-5">
          <legend className={labelClass}>Request type</legend>
          <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-3">
            {TYPES.map((t) => {
              const selected = type === t;
              return (
                <label
                  key={t}
                  className={`relative flex cursor-pointer flex-col gap-2 rounded-xl border p-3.5 transition-colors ${
                    selected ? 'border-[#1b1c1a] bg-white shadow-sm' : 'border-[#e3e2df] bg-[#faf9f6] hover:border-[#c4c7c7]'
                  } ${draftId ? 'pointer-events-none opacity-60' : ''}`}
                >
                  <input
                    type="radio"
                    name="type"
                    value={t}
                    checked={selected}
                    onChange={() => setType(t)}
                    disabled={!!draftId}
                    className="sr-only"
                  />
                  <span className="flex items-center justify-between">
                    <span className="flex size-8 items-center justify-center rounded-lg bg-[#efeeeb]">
                      <Icon name={TYPE[t].icon} size={17} />
                    </span>
                    <span
                      className={`flex size-5 items-center justify-center rounded-full ${selected ? 'bg-[#c7ef39] text-[#171e00]' : 'bg-[#efeeeb]'}`}
                    >
                      {selected && <Icon name="check" size={13} />}
                    </span>
                  </span>
                  <span className={`${fontHeading} text-[15px] font-semibold`}>{TYPE[t].label}</span>
                  <span className="text-[12px] leading-4 text-[#444748]">{TYPE[t].description}</span>
                </label>
              );
            })}
          </div>
        </fieldset>

        <div className="mt-6 flex flex-col gap-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="amount" className={labelClass}>
              Amount ({currency}) *
            </label>
            <span className="text-[12px] text-[#747878]">Max {REQUEST_RULES.maxAmount.toLocaleString('en-US')}</span>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-[#f4f3f0] px-4 py-3 focus-within:ring-2 focus-within:ring-[#c7ef39]/60">
            <input
              id="amount"
              inputMode="decimal"
              type="number"
              min={REQUEST_RULES.minAmount}
              max={REQUEST_RULES.maxAmount}
              step="0.01"
              placeholder="0.00"
              value={amount}
              disabled={!!draftId}
              onChange={(e) => setAmount(e.target.value)}
              aria-invalid={touched && !!amountError}
              className={`${fontHeading} w-full bg-transparent text-[28px] font-bold tracking-[-0.7px] outline-none [appearance:textfield] placeholder:text-[#c4c7c7] [&::-webkit-inner-spin-button]:appearance-none`}
            />
            <span className="rounded-full bg-[#e3e2df] px-2 py-0.5 text-[11px] font-semibold text-[#444748]">{currency}</span>
          </div>
          {touched && amountError && <FieldError>{amountError}</FieldError>}
        </div>

        <fieldset className="mt-6">
          <legend className={labelClass}>Urgency</legend>
          <div className="mt-2 grid grid-cols-3 gap-1 rounded-full bg-[#f4f3f0] p-1">
            {URGENCY_OPTIONS.map((u) => (
              <label
                key={u.value}
                title={u.hint}
                className={`cursor-pointer rounded-full px-3 py-1.5 text-center text-[13px] transition-colors ${
                  urgency === u.value ? 'bg-white font-semibold text-[#1b1c1a] shadow-sm' : 'text-[#444748] hover:text-[#1b1c1a]'
                } ${draftId ? 'pointer-events-none opacity-60' : ''}`}
              >
                <input
                  type="radio"
                  name="urgency"
                  value={u.value}
                  checked={urgency === u.value}
                  onChange={() => setUrgency(u.value)}
                  disabled={!!draftId}
                  className="sr-only"
                />
                {u.label}
              </label>
            ))}
          </div>
        </fieldset>

        <div className="mt-6 flex flex-col gap-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="purpose" className={labelClass}>
              Business purpose *
            </label>
            <span className={`font-mono text-[11px] ${purposeLength > REQUEST_RULES.maxPurpose ? 'text-[#ba1a1a]' : 'text-[#747878]'}`}>
              {purposeLength} / {REQUEST_RULES.maxPurpose} characters
            </span>
          </div>
          <textarea
            id="purpose"
            rows={3}
            value={purpose}
            disabled={!!draftId}
            onChange={(e) => setPurpose(e.target.value)}
            placeholder="What is this for? e.g. Conference travel to ACS Fall 2026 — registration and hotel."
            aria-invalid={touched && !!purposeError}
            className="w-full resize-y rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] leading-5 outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
          />
          {touched && purposeError && <FieldError>{purposeError}</FieldError>}
        </div>
      </section>

      <section className={cardClass} aria-labelledby="receipts-heading">
        <SectionHeading n={2} id="receipts-heading" title="Receipts" note={isReimbursement ? 'Required to submit' : 'Not needed'} />
        {isReimbursement ? (
          <>
            <div
              onDragOver={(e) => {
                e.preventDefault();
                setDragging(true);
              }}
              onDragLeave={() => setDragging(false)}
              onDrop={onDrop}
              className={`mt-5 flex flex-col items-center gap-2 rounded-xl border-2 border-dashed px-6 py-8 text-center transition-colors ${
                dragging ? 'border-[#526600] bg-[rgba(199,239,57,0.12)]' : 'border-[#e3e2df] bg-[#faf9f6]'
              }`}
            >
              <span className="flex size-10 items-center justify-center rounded-full bg-[#c7ef39] text-[#171e00]">
                <Icon name="upload_file" size={20} />
              </span>
              <p className={`${fontHeading} text-[15px] font-semibold`}>
                Drop receipts here, or{' '}
                <button type="button" onClick={() => fileInput.current?.click()} className="underline underline-offset-2">
                  browse files
                </button>
              </p>
              <p className="text-[12px] text-[#444748]">PDF, PNG or JPG, up to 10 MB each</p>
              <input
                ref={fileInput}
                type="file"
                multiple
                accept="application/pdf,image/png,image/jpeg"
                className="sr-only"
                onChange={(e) => {
                  addFiles(e.target.files);
                  e.target.value = '';
                }}
              />
            </div>
            {fileError && <FieldError className="mt-2">{fileError}</FieldError>}
            {files.length > 0 && (
              <ul className="mt-4 flex flex-col gap-2">
                {files.map(({ key, file }) => (
                  <li key={key} className="flex items-center justify-between gap-3 rounded-xl bg-[#f4f3f0] px-4 py-3">
                    <span className="flex min-w-0 items-center gap-3">
                      <Icon name={file.type === 'application/pdf' ? 'picture_as_pdf' : 'image'} size={20} className="text-[#ba1a1a]" />
                      <span className="min-w-0">
                        <span className="block truncate text-[13px] font-semibold">{file.name}</span>
                        <span className="block text-[12px] text-[#444748]">
                          {formatBytes(file.size)}
                          {uploaded.has(key) && ' · uploaded'}
                        </span>
                      </span>
                    </span>
                    {!uploaded.has(key) && (
                      <button
                        type="button"
                        onClick={() => setFiles((prev) => prev.filter((f) => f.key !== key))}
                        className="flex items-center gap-1 text-[12px] font-semibold text-[#ba1a1a] hover:underline"
                      >
                        <Icon name="delete" size={14} />
                        Remove
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            )}
            {touched && receiptError && !fileError && <FieldError className="mt-2">{receiptError}</FieldError>}
          </>
        ) : (
          <p className="mt-4 flex items-center gap-2 rounded-xl bg-[#f4f3f0] px-4 py-3 text-[13px] text-[#444748]">
            <Icon name="info" size={16} />
            {TYPE[type].label}s don&apos;t need receipts. They go straight to finance review.
          </p>
        )}
      </section>

      <section className={cardClass} aria-labelledby="notes-heading">
        <SectionHeading n={3} id="notes-heading" title="Notes for finance" note="Optional" />
        <textarea
          aria-labelledby="notes-heading"
          rows={2}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="Anything the approver should know. Posted as the first comment on the request."
          className="mt-4 w-full resize-y rounded-xl bg-[#f4f3f0] px-4 py-3 text-[14px] leading-5 outline-none placeholder:text-[#747878] focus:ring-2 focus:ring-[#c7ef39]/60"
        />
      </section>

      <div className="fixed inset-x-0 bottom-0 z-20 border-t border-[#efeeeb] bg-[rgba(250,249,246,0.95)] backdrop-blur lg:left-72">
        <div className="mx-auto flex max-w-[720px] flex-col gap-2 px-5 py-3">
          {error && <FieldError>{error}</FieldError>}
          <div className="flex items-center justify-between gap-3">
            <Link href="/requests" className="rounded-full px-4 py-2 text-[14px] font-medium text-[#444748] hover:bg-[#efeeeb]">
              Cancel
            </Link>
            <div className="flex items-center gap-2">
              {isReimbursement && (
                <button
                  type="button"
                  onClick={() => run('draft')}
                  disabled={!!busy}
                  className={`${fontHeading} rounded-full bg-[#efeeeb] px-4 py-2 text-[14px] font-semibold hover:bg-[#e3e2df] disabled:opacity-50`}
                >
                  {busy === 'draft' ? 'Saving…' : 'Save as draft'}
                </button>
              )}
              <button
                type="submit"
                disabled={!!busy}
                className={`${fontHeading} flex items-center gap-1.5 rounded-full bg-black px-5 py-2 text-[14px] font-semibold text-white hover:bg-[#1c1b1b] disabled:opacity-50`}
              >
                {busy === 'submit' ? 'Submitting…' : 'Submit request'}
                <Icon name="arrow_forward" size={15} />
              </button>
            </div>
          </div>
        </div>
      </div>
    </form>
  );
}

function SectionHeading({ n, id, title, note }: { n: number; id: string; title: string; note: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <h2 id={id} className={`${fontHeading} flex items-center gap-2.5 text-[16px] font-semibold`}>
        <span className="flex size-6 items-center justify-center rounded-full bg-black text-[12px] font-bold text-white">{n}</span>
        {title}
      </h2>
      <span className="text-[11px] font-semibold uppercase tracking-[0.55px] text-[#747878]">{note}</span>
    </div>
  );
}

function FieldError({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <p role="alert" className={`flex items-center gap-1 text-[12px] font-medium text-[#ba1a1a] ${className}`}>
      <Icon name="error" size={14} />
      {children}
    </p>
  );
}
