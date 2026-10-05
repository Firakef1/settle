import * as React from 'react';

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className = '', ...props }, ref) => (
    <input
      ref={ref}
      className={`w-full rounded-xl bg-[#F4F3F0] px-3.5 py-2.5 text-[14px] text-[#1B1C1A] placeholder:text-[#444748]/70 outline-none focus:ring-2 focus:ring-[#B5F546]/40 transition-all shadow-sm ${className}`}
      {...props}
    />
  )
);
Input.displayName = 'Input';
