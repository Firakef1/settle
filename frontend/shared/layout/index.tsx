import React from 'react';
import Link from 'next/link';

export function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-white">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0">
        <Header />
        <div className="flex-1 px-6 py-6">{children}</div>
      </main>
    </div>
  );
}

function Header() {
  return (
    <header className="flex items-center justify-between px-6 py-3 border-b border-[#EFEEEB] bg-white/80 backdrop-blur-sm sticky top-0 z-30">
      <div className="flex items-center gap-3">
        <span className="material-symbols-outlined text-2xl text-[#B5F546]">receipt_long</span>
        <span className="font-extrabold text-[18px] text-[#1B1C1A]">Settle</span>
      </div>
      <div className="flex items-center gap-3 text-[13px] font-medium text-[#444748]">
        <Link href="/requests" className="hover:text-[#1B1C1A]">Requests</Link>
        <Link href="/dashboard" className="hover:text-[#1B1C1A]">Dashboard</Link>
      </div>
    </header>
  );
}

function Sidebar() {
  return (
    <aside className="w-64 shrink-0 border-r border-[#EFEEEB] bg-[#F4F3F0] p-4 hidden md:block">
      <nav className="flex flex-col gap-1 text-[14px] font-medium text-[#1B1C1A]">
        <Link href="/dashboard" className="px-3 py-2 rounded-full hover:bg-[#EFEEEB] transition-colors">Dashboard</Link>
        <Link href="/requests" className="px-3 py-2 rounded-full hover:bg-[#EFEEEB] transition-colors">Requests</Link>
        <div className="mt-4 text-[11px] font-bold uppercase tracking-wider text-[#444748]">Settings</div>
        <Link href="/settings" className="px-3 py-2 rounded-full hover:bg-[#EFEEEB] transition-colors">Account</Link>
      </nav>
    </aside>
  );
}
