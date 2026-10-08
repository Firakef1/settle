export const ROLES = {
  staff: 'Staff',
  finance: 'Finance',
  org_admin: 'Admin',
} as const;

export const REQUEST_STATUSES = {
  draft: 'Draft',
  pending: 'Pending Approval',
  approved: 'Approved',
  paid: 'Paid',
  rejected: 'Rejected',
  failed: 'Failed',
  withdrawn: 'Withdrawn',
} as const;

export const URGENCIES = {
  routine: 'Routine',
  urgent: 'Urgent',
  critical: 'Critical',
} as const;

export const REQUEST_TYPES = {
  reimbursement: 'Reimbursement',
  advance: 'Advance',
  stipend: 'Stipend',
} as const;

export const CURRENCIES = {
  USD: 'USD',
  EUR: 'EUR',
  GBP: 'GBP',
} as const;

export const PLANS = {
  free: 'Free',
  starter: 'Starter',
  pro: 'Pro',
} as const;
