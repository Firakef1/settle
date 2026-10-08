'use client';

import { useNotificationStore } from '../stores/notificationStore';
import { Icon } from './Icon';

const TONE = {
  success: { icon: 'check_circle', className: 'bg-[#1b1c1a] text-white', iconClass: 'text-[#c7ef39]' },
  error: { icon: 'error', className: 'bg-[#93000a] text-white', iconClass: 'text-[#ffdad6]' },
  info: { icon: 'info', className: 'bg-[#1b1c1a] text-white', iconClass: 'text-[#c7ef39]' },
};

export function Toaster() {
  const { toasts, dismiss } = useNotificationStore();
  return (
    <div aria-live="polite" className="pointer-events-none fixed bottom-5 right-5 z-[70] flex w-[min(380px,calc(100vw-2.5rem))] flex-col gap-2">
      {toasts.map((t) => {
        const tone = TONE[t.tone];
        return (
          <div key={t.id} role="status" className={`pointer-events-auto flex items-start gap-3 rounded-xl px-4 py-3 text-[13px] shadow-lg ${tone.className}`}>
            <Icon name={tone.icon} size={18} className={tone.iconClass} />
            <p className="flex-1 leading-5">{t.message}</p>
            <button type="button" onClick={() => dismiss(t.id)} aria-label="Dismiss" className="opacity-70 hover:opacity-100">
              <Icon name="close" size={16} />
            </button>
          </div>
        );
      })}
    </div>
  );
}
