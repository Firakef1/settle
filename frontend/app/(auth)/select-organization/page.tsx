'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '../../../shared/stores/authStore';
import Link from 'next/link';

interface Organization {
  id: string;
  name: string;
  slug: string;
  role: string;
  memberCount: number;
  lastActivity: string;
  icon: string;
}

export default function SelectOrganizationPage() {
  const router = useRouter();
  const { selectOrganization, user } = useAuthStore();

  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [creatingOrg, setCreatingOrg] = useState(false);
  const [orgName, setOrgName] = useState('');
  const [orgSlug, setOrgSlug] = useState('');

  const organizations: Organization[] = [
    {
      id: 'org-1',
      name: 'Acme Research Lab',
      slug: 'acme-research-lab',
      role: 'org_admin',
      memberCount: 8,
      lastActivity: '2 hours ago',
      icon: 'science',
    },
    {
      id: 'org-2',
      name: 'Northwind Capital',
      slug: 'northwind-capital',
      role: 'finance',
      memberCount: 24,
      lastActivity: 'Yesterday',
      icon: 'account_balance',
    },
    {
      id: 'org-3',
      name: 'BioSynth Therapeutics',
      slug: 'biosynth-tx',
      role: 'staff',
      memberCount: 14,
      lastActivity: '5 days ago',
      icon: 'biotech',
    },
  ];

  const filteredOrgs = organizations.filter(
    org =>
      org.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      org.slug.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleSelectOrg = (orgId: string) => {
    selectOrganization(orgId);
    router.push('/dashboard');
  };

  const handleCreateOrg = (e: React.FormEvent) => {
    e.preventDefault();
    setCreatingOrg(true);
    setTimeout(() => {
      setCreatingOrg(false);
      setShowCreateModal(false);
      router.push('/dashboard');
    }, 1500);
  };

  const roleBadge = (role: string) => {
    if (role === 'org_admin') return { label: 'Admin', bg: 'bg-[#0E0E0E]', text: 'text-white', icon: 'shield_person' };
    if (role === 'finance') return { label: 'Finance', bg: 'bg-[#B5F546]', text: 'text-[#171E00]', icon: 'query_stats' };
    return { label: 'Staff', bg: 'bg-[#F0EFEB]', text: 'text-[#1B1C1A]', icon: 'badge' };
  };

  return (
    <div className="bg-[#FAF9F6] text-[#1B1C1A] antialiased min-h-screen flex flex-col">
      {/* Header */}
      <header className="w-full bg-white border-b border-[#E5E4E0] sticky top-0 z-40">
        <div className="h-16 max-w-5xl mx-auto px-6 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-[#0E0E0E] flex items-center justify-center">
              <div className="w-4 h-4 rounded-sm bg-[#B5F546] flex items-center justify-center">
                <span className="w-2 h-2 bg-[#0E0E0E] rounded-[1px] block" />
              </div>
            </div>
            <span className="text-[#0E0E0E] font-bold text-lg tracking-tight" style={{fontFamily: 'var(--font-plus-jakarta-sans)'}}>Settle</span>
          </div>
          <div className="flex items-center gap-2 py-1.5 px-3 rounded-full bg-[#F6F6F4] border border-[#E5E4E0] text-[#575A5A] text-xs font-medium">
            <span className="material-symbols-outlined text-[14px] text-[#0E0E0E]">lock</span>
            <span>256-Bit SSL</span>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="flex-1 flex flex-col items-center px-4 py-10 sm:py-14">
        <div className="w-full max-w-3xl mx-auto">

          {/* User Pill */}
          {user?.email && (
            <div className="flex items-center justify-center mb-8">
              <div className="flex items-center gap-2.5 bg-white border border-[#E5E4E0] px-4 py-2 rounded-full shadow-sm">
                <div className="w-6 h-6 rounded-full bg-[#0E0E0E] flex items-center justify-center text-white text-[11px] font-bold">
                  {user.email.charAt(0).toUpperCase()}
                </div>
                <span className="text-[13px] text-[#575A5A]">
                  Signed in as <span className="font-semibold text-[#1B1C1A]">{user.email}</span>
                </span>
                <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]" />
              </div>
            </div>
          )}

          {/* Title */}
          <div className="text-center mb-8">
            <h1 className="text-3xl sm:text-4xl font-bold text-[#1B1C1A] tracking-tight mb-2" style={{fontFamily: 'var(--font-plus-jakarta-sans)'}}>
              Choose a Workspace
            </h1>
            <p className="text-[15px] text-[#575A5A]">
              Select which organization you want to work in today.
            </p>
          </div>

          {/* Search */}
          <div className="relative mb-6">
            <span className="material-symbols-outlined absolute left-3.5 top-1/2 -translate-y-1/2 text-[#575A5A] text-[18px]">search</span>
            <input
              className="w-full h-11 pl-10 pr-4 bg-white border border-[#E5E4E0] rounded-xl text-[#1B1C1A] text-sm placeholder:text-[#575A5A]/60 focus:outline-none focus:border-[#0E0E0E] focus:ring-1 focus:ring-[#0E0E0E] transition-all"
              placeholder="Search organizations..."
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {/* Org Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
            {filteredOrgs.map((org) => {
              const badge = roleBadge(org.role);
              return (
                <button
                  key={org.id}
                  className="text-left flex flex-col justify-between p-5 rounded-2xl bg-white border border-[#E5E4E0] hover:border-[#0E0E0E] hover:shadow-md transition-all duration-200 hover:-translate-y-0.5 group"
                  onClick={() => handleSelectOrg(org.id)}
                  type="button"
                >
                  {/* Top row */}
                  <div className="flex items-start justify-between gap-3 mb-4">
                    <div className="w-11 h-11 rounded-xl bg-[#F6F6F4] border border-[#E5E4E0] flex items-center justify-center text-[#0E0E0E] group-hover:bg-[#0E0E0E] group-hover:text-[#B5F546] transition-colors">
                      <span className="material-symbols-outlined text-[22px]">{org.icon}</span>
                    </div>
                    <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full ${badge.bg} ${badge.text} text-[11px] font-semibold uppercase tracking-wide`}>
                      <span className="material-symbols-outlined text-[12px]">{badge.icon}</span>
                      {badge.label}
                    </span>
                  </div>

                  {/* Name & slug */}
                  <div className="mb-4">
                    <h2 className="text-[17px] font-bold text-[#1B1C1A] mb-0.5" style={{fontFamily: 'var(--font-plus-jakarta-sans)'}}>
                      {org.name}
                    </h2>
                    <p className="text-[12px] text-[#575A5A] font-mono truncate">{org.slug}</p>
                  </div>

                  {/* Meta */}
                  <div className="flex items-center justify-between text-[13px] text-[#575A5A] pt-3 border-t border-[#E5E4E0]">
                    <span className="flex items-center gap-1.5">
                      <span className="material-symbols-outlined text-[15px]">group</span>
                      {org.memberCount} members
                    </span>
                    <span className="flex items-center gap-1.5">
                      <span className="material-symbols-outlined text-[15px]">schedule</span>
                      {org.lastActivity}
                    </span>
                  </div>
                </button>
              );
            })}

            {/* Create new org card */}
            <button
              className="text-left flex flex-col items-center justify-center p-5 rounded-2xl border-2 border-dashed border-[#E5E4E0] hover:border-[#0E0E0E] bg-white/50 hover:bg-white transition-all duration-200 group min-h-[180px]"
              onClick={() => setShowCreateModal(true)}
              type="button"
            >
              <div className="w-11 h-11 rounded-xl bg-[#F6F6F4] border border-[#E5E4E0] flex items-center justify-center text-[#0E0E0E] mb-3 group-hover:bg-[#0E0E0E] group-hover:text-[#B5F546] transition-colors">
                <span className="material-symbols-outlined text-[22px]">add</span>
              </div>
              <h3 className="text-[16px] font-bold text-[#1B1C1A] mb-1" style={{fontFamily: 'var(--font-plus-jakarta-sans)'}}>
                New Organization
              </h3>
              <p className="text-[13px] text-[#575A5A] text-center leading-relaxed">
                Create a new workspace for your team
              </p>
            </button>
          </div>

          {/* No results */}
          {filteredOrgs.length === 0 && (
            <div className="w-full py-12 text-center bg-white border border-[#E5E4E0] rounded-2xl mb-6">
              <span className="material-symbols-outlined text-[40px] text-[#E5E4E0] block mb-3">search_off</span>
              <h3 className="text-[16px] font-bold text-[#1B1C1A] mb-1">No results found</h3>
              <p className="text-[13px] text-[#575A5A] mb-4">No organization matches &ldquo;{searchQuery}&rdquo;</p>
              <button
                className="px-4 py-2 bg-[#F6F6F4] border border-[#E5E4E0] text-[#1B1C1A] text-[13px] font-medium rounded-full hover:bg-[#EEEDEA] transition-colors"
                onClick={() => setSearchQuery('')}
              >
                Clear search
              </button>
            </div>
          )}

          {/* Footer links */}
          <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-2 text-[13px] text-[#575A5A]">
            <Link
              className="flex items-center gap-1.5 hover:text-[#0E0E0E] transition-colors"
              href="/login"
            >
              <span className="material-symbols-outlined text-[16px]">logout</span>
              Sign in with a different account
            </Link>
            <Link
              className="flex items-center gap-1.5 hover:text-[#0E0E0E] transition-colors"
              href="#"
            >
              <span className="material-symbols-outlined text-[16px]">help</span>
              Need help?
            </Link>
          </div>
        </div>
      </main>

      {/* Create Org Modal */}
      {showCreateModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm p-4"
          onClick={() => setShowCreateModal(false)}
        >
          <div
            className="w-full max-w-md bg-white border border-[#E5E4E0] rounded-2xl shadow-2xl p-6 flex flex-col gap-5"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal header */}
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-[18px] font-bold text-[#1B1C1A]" style={{fontFamily: 'var(--font-plus-jakarta-sans)'}}>
                  Create Organization
                </h3>
                <p className="text-[13px] text-[#575A5A] mt-0.5">Set up a new workspace for your team</p>
              </div>
              <button
                className="w-8 h-8 rounded-lg flex items-center justify-center text-[#575A5A] hover:bg-[#F6F6F4] transition-colors"
                onClick={() => setShowCreateModal(false)}
                type="button"
              >
                <span className="material-symbols-outlined text-[20px]">close</span>
              </button>
            </div>

            <form className="flex flex-col gap-4" onSubmit={handleCreateOrg}>
              <div>
                <label className="text-[12px] font-semibold text-[#1B1C1A] uppercase tracking-wide block mb-1.5">
                  Organization Name
                </label>
                <input
                  className="w-full h-11 px-3.5 bg-[#F6F6F4] border border-[#E5E4E0] rounded-xl text-[#1B1C1A] text-sm focus:outline-none focus:border-[#0E0E0E] focus:bg-white transition-all"
                  placeholder="e.g. Acme Inc."
                  required
                  type="text"
                  value={orgName}
                  onChange={(e) => {
                    setOrgName(e.target.value);
                    setOrgSlug(e.target.value.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, ''));
                  }}
                />
              </div>

              <div>
                <label className="text-[12px] font-semibold text-[#1B1C1A] uppercase tracking-wide block mb-1.5">
                  URL Slug
                </label>
                <div className="flex items-center h-11 bg-[#F6F6F4] border border-[#E5E4E0] rounded-xl overflow-hidden focus-within:border-[#0E0E0E] focus-within:bg-white transition-all">
                  <span className="pl-3.5 pr-1 text-[#575A5A] text-sm whitespace-nowrap">settle.app/</span>
                  <input
                    className="flex-1 pr-3.5 bg-transparent text-[#1B1C1A] text-sm focus:outline-none"
                    placeholder="acme-inc"
                    type="text"
                    value={orgSlug}
                    onChange={(e) => setOrgSlug(e.target.value)}
                  />
                </div>
              </div>

              <div className="flex items-center gap-3 pt-1">
                <button
                  className="flex-1 h-11 rounded-full border border-[#E5E4E0] text-[#1B1C1A] text-sm font-semibold hover:bg-[#F6F6F4] transition-colors"
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                >
                  Cancel
                </button>
                <button
                  className="flex-1 h-11 rounded-full bg-[#0E0E0E] text-white text-sm font-semibold hover:bg-neutral-800 transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
                  type="submit"
                  disabled={creatingOrg || !orgName.trim()}
                >
                  {creatingOrg ? (
                    <>
                      <span className="material-symbols-outlined text-[16px] animate-spin">progress_activity</span>
                      Creating...
                    </>
                  ) : 'Create Organization'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="w-full bg-white border-t border-[#E5E4E0] py-5">
        <div className="max-w-5xl mx-auto px-6 flex flex-col sm:flex-row items-center justify-between gap-3 text-[12px] text-[#575A5A]">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-[#B5F546]" />
            <span>© 2025 Settle Technologies Inc.</span>
          </div>
          <div className="flex items-center gap-5">
            <Link className="hover:text-[#1B1C1A] transition-colors" href="#">Privacy</Link>
            <Link className="hover:text-[#1B1C1A] transition-colors" href="#">Terms</Link>
            <Link className="hover:text-[#1B1C1A] transition-colors" href="#">Security</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
