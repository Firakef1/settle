import * as React from 'react';

export const Button = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
    size?: 'sm' | 'md' | 'lg';
  }
>(({ className = '', variant = 'primary', size = 'md', ...props }, ref) => {
  const variants = {
    primary: 'bg-[#0E0E0E] text-white hover:bg-[#1C1B1B] shadow-sm',
    secondary: 'bg-[#EFEEEB] text-[#1B1C1A] hover:bg-[#E9E8E5]',
    danger: 'bg-[#93000A] text-white hover:bg-[#BA1A1A]',
    ghost: 'bg-transparent text-[#1B1C1A] hover:bg-[#EFEEEB]',
  };
  const sizes = {
    sm: 'px-3 py-1.5 text-xs rounded-full font-semibold',
    md: 'px-4 py-2 text-sm rounded-full font-semibold',
    lg: 'px-6 py-3 text-base rounded-full font-bold',
  };
  return (
    <button
      ref={ref}
      className={`inline-flex items-center justify-center gap-1.5 transition-all disabled:opacity-50 disabled:cursor-not-allowed ${variants[variant]} ${sizes[size]} ${className}`}
      {...props}
    />
  );
});
Button.displayName = 'Button';
