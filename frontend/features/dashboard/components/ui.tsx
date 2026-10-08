import type { ReactNode } from 'react';
import { Button } from '../../../shared/components/Button';
import { Icon } from '../../../shared/components/Icon';
import { apiErrorMessage } from '../../../shared/utils/apiError';
import { fontHeading } from '../../../shared/utils/fonts';

export { Icon } from '../../../shared/components/Icon';
export { fontBody, fontHeading } from '../../../shared/utils/fonts';

export function Dot({ className }: { className: string }) {
  return <span aria-hidden className={`inline-block size-2 shrink-0 rounded-full ${className}`} />;
}

export function PulseDot({ className }: { className: string }) {
  return (
    <span aria-hidden className="relative inline-flex size-2 shrink-0">
      <span className={`absolute inset-0 animate-ping rounded-full opacity-75 ${className}`} />
      <span className={`relative inline-flex size-2 rounded-full ${className}`} />
    </span>
  );
}

export function Pill({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-[12px] leading-4 ${className}`}
    >
      {children}
    </span>
  );
}

export function greeting(name: string | undefined, now = new Date()): string {
  const hour = now.getHours();
  const part = hour < 12 ? 'morning' : hour < 18 ? 'afternoon' : 'evening';
  const first = name?.trim().split(/\s+/)[0];
  return first ? `Good ${part}, ${first}` : `Good ${part}`;
}

export function percent(part: number, whole: number): number {
  return whole > 0 ? (part / whole) * 100 : 0;
}

export interface Segment {
  key: string;
  label: string;
  value: number;
  color: string;
}

// Stacked pill bar used by both pipeline sections. Empty segments are skipped
// so the rounded ends always land on the first/last visible segment.
export function SegmentBar({ segments, trackClass, gapClass }: { segments: Segment[]; trackClass: string; gapClass: string }) {
  const total = segments.reduce((sum, s) => sum + s.value, 0);
  const visible = segments.filter((s) => s.value > 0);
  return (
    <div
      role="img"
      aria-label={segments.map((s) => `${s.label}: ${s.value}`).join(', ')}
      className={`flex h-3 w-full overflow-hidden rounded-full p-0.5 ${trackClass} ${gapClass}`}
    >
      {visible.map((s, i) => (
        <div
          key={s.key}
          className={`h-full ${s.color} ${i === 0 ? 'rounded-l-full' : ''} ${i === visible.length - 1 ? 'rounded-r-full' : ''}`}
          style={{ width: `${percent(s.value, total)}%` }}
        />
      ))}
    </div>
  );
}

export function SkeletonBlock({ className }: { className: string }) {
  return <div className={`animate-pulse rounded-xl bg-[#efeeeb] ${className}`} />;
}

export function DashboardSkeleton() {
  return (
    <div className="flex flex-col gap-7" aria-busy="true" aria-label="Loading dashboard">
      <SkeletonBlock className="h-20 w-full max-w-lg" />
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <SkeletonBlock key={i} className="h-40" />
        ))}
      </div>
      <SkeletonBlock className="h-28" />
      <SkeletonBlock className="h-96" />
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl bg-white px-6 py-16 text-center shadow-[0px_1px_3px_0px_rgba(0,0,0,0.04)]">
      <Icon name="error" size={28} className="text-[#ba1a1a]" />
      <p className={`${fontHeading} text-[16px] font-bold text-[#1b1c1a]`}>We couldn&apos;t load the dashboard</p>
      <p className="max-w-md text-[13px] text-[#444748]">{message}</p>
      <Button variant="secondary" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function errorMessage(error: unknown): string {
  return apiErrorMessage(error, 'Check your connection and try again.');
}

export function updatedAgo(timestamp: number, now = Date.now()): string {
  if (!timestamp) return 'Not loaded';
  const minutes = Math.floor((now - timestamp) / 60000);
  if (minutes < 1) return 'Updated just now';
  if (minutes < 60) return `Updated ${minutes}m ago`;
  return `Updated ${Math.floor(minutes / 60)}h ago`;
}
