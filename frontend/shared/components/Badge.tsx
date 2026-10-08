import * as React from 'react';

export const Badge = React.forwardRef<
  HTMLSpanElement,
  React.HTMLAttributes<HTMLSpanElement> & {
    color?: 'default' | 'success' | 'warning' | 'danger' | 'info';
  }
>(({ className = '', color = 'default', children, ...props }, ref) => {
  const colors: Record<string, string> = {
    default: 'bg-[#EFEEEB] text-[#1B1C1A]',
    success: 'bg-emerald-50 text-emerald-800',
    warning: 'bg-amber-50 text-amber-800',
    danger: 'bg-[#FFDAD6] text-[#93000A]',
    info: 'bg-sky-50 text-sky-800',
  };
  return (
    <span
      ref={ref}
      className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[12px] font-semibold ${colors[color]} ${className}`}
      {...props}
    >
      {children}
    </span>
  );
});
Badge.displayName = 'Badge';
