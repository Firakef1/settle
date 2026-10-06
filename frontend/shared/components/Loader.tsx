import * as React from 'react';

export const Loader = ({ size = 'md', label }: { size?: 'sm' | 'md' | 'lg'; label?: string }) => {
  const sizes = { sm: 'w-4 h-4', md: 'w-6 h-6', lg: 'w-8 h-8' };
  return (
    <div className="inline-flex items-center gap-2 text-[#444748]">
      <span className={`material-symbols-outlined animate-spin ${sizes[size]} text-[#B5F546]`}>progress_activity</span>
      {label && <span className="text-sm">{label}</span>}
    </div>
  );
};
