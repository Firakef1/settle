'use client';

import { useEffect, useId, useRef, type ReactNode } from 'react';
import { Icon } from './Icon';
import { fontHeading } from '../utils/fonts';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: ReactNode;
  icon?: string;
  tone?: 'default' | 'danger' | 'success';
  children?: ReactNode;
  footer?: ReactNode;
  width?: string;
}

const TONES = {
  default: 'bg-[#efeeeb] text-[#1b1c1a]',
  danger: 'bg-[#ffdad6] text-[#ba1a1a]',
  success: 'bg-[#c7ef39] text-[#171e00]',
};

// Accessible dialog: Escape and backdrop click close it, focus moves inside,
// and the page behind doesn't scroll.
export function Modal({ open, onClose, title, description, icon, tone = 'default', children, footer, width = 'max-w-[480px]' }: ModalProps) {
  const titleId = useId();
  const panel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose();
    document.addEventListener('keydown', onKey);
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const first = panel.current?.querySelector<HTMLElement>('textarea, input, select, button:not([data-close])');
    (first ?? panel.current)?.focus();
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = overflow;
      previous?.focus();
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <button type="button" aria-label="Close dialog" tabIndex={-1} className="absolute inset-0 bg-[rgba(14,14,14,0.55)] backdrop-blur-[2px]" onClick={onClose} />
      <div
        ref={panel}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={`relative flex max-h-[90vh] w-full ${width} flex-col gap-5 overflow-y-auto rounded-2xl bg-white p-7 shadow-[0px_20px_25px_-5px_rgba(0,0,0,0.1),0px_8px_10px_-6px_rgba(0,0,0,0.1)] outline-none`}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-3">
            {icon && (
              <span className={`flex size-12 items-center justify-center rounded-2xl ${TONES[tone]}`}>
                <Icon name={icon} size={22} />
              </span>
            )}
            <div>
              <h2 id={titleId} className={`${fontHeading} text-[20px] font-bold leading-7 tracking-[-0.4px] text-[#1b1c1a]`}>
                {title}
              </h2>
              {description && <div className="mt-1 text-[14px] leading-5 text-[#444748]">{description}</div>}
            </div>
          </div>
          <button
            type="button"
            data-close
            onClick={onClose}
            aria-label="Close"
            className="rounded-full p-1 text-[#444748] hover:bg-[#efeeeb]"
          >
            <Icon name="close" size={20} />
          </button>
        </div>
        {children}
        {footer && <div className="flex items-center justify-end gap-2 border-t border-[#efeeeb] pt-4">{footer}</div>}
      </div>
    </div>
  );
}
