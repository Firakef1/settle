// Types derived from backend/docs/Api-contract.md
// Modern TypeScript: strict, readonly arrays preferred, branded IDs where useful

// --- Enums (as const unions) ---
export type Role = 'staff' | 'finance' | 'org_admin';
export type RequestType = 'reimbursement' | 'advance' | 'stipend';
export type Urgency = 'routine' | 'urgent' | 'critical';
export type RequestStatus = 'draft' | 'pending' | 'approved' | 'paid' | 'rejected' | 'failed' | 'withdrawn';
export type Currency = 'USD' | 'EUR' | 'GBP';
export type PlanId = 'free' | 'starter' | 'pro';
export type PaymentMethod = 'bank_transfer' | 'check' | 'cash' | 'other';

// --- Account / Auth ---
export interface User {
  id: string;
  email: string;
  name: string;
  status: 'active' | string;
  email_verified: boolean;
  created_at: string; // ISO-8601
}

export interface AuthResponse {
  token: string;
  refresh_token: string;
  user: User;
  orgs: OrgMembership[];
}

// --- Organization ---
export interface Organization {
  id: string;
  name: string;
  slug: string;
  currency: Currency;
  plan: PlanId;
  created_at: string;
  updated_at: string;
}

export interface OrgMembership {
  org_id: string;
  org_name: string;
  org_slug: string;
  role: Role;
  department: string;
}

export interface Member {
  user_id: string;
  name: string;
  email: string;
  role: Role;
  joined_at: string;
}

export interface Invitation {
  id: string;
  org_id: string;
  email: string;
  role: 'staff' | 'finance';
  token: string;
  expires_at: string;
  created_at: string;
}

// --- Requests ---
export interface Request {
  id: string;
  type: RequestType;
  amount: number;
  purpose: string;
  urgency: Urgency;
  status: RequestStatus;
  requester: { id: string; name: string; email: string };
  receipts: Receipt[];
  comments: Comment[];
  timeline: unknown | null; // backend returns null; client builds
  submitted_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface RequestListItem {
  id: string;
  amount: number;
  purpose: string;
  urgency: Urgency;
  status: RequestStatus;
  requester: { id: string; name: string; email: string };
  days_pending: number;
  created_at: string;
}

export interface Receipt {
  id: string;
  file_path: string;
  file_url?: string;
  file_name?: string;
  file_size?: number;
  file_type?: string;
  ocr_status: 'pending' | 'processing' | 'success' | 'failed';
  extracted_amount?: number;
  extracted_merchant?: string;
  extracted_date?: string;
  created_at: string;
}

export interface Comment {
  id: string;
  author_id: string;
  // The API currently sends `author` as an empty string; match on author_id.
  author?: string;
  author_name?: string;
  author_role?: string;
  content: string;
  text?: string; // alias for content in some responses
  created_at: string;
}

export interface Approval {
  id: string;
  request_id: string;
  org_id: string;
  approver_id: string;
  decision: 'approved' | 'rejected' | 'pending_payment' | 'failed';
  decision_note?: string;
  payment_status?: string;
  payment_method?: PaymentMethod;
  payment_at?: string | null;
  failure_reason?: string;
  failure_at?: string | null;
  request_status: RequestStatus;
  created_at: string;
  updated_at: string;
}

// --- Dashboard / Stats ---
export interface DashboardSummary {
  pending_count: number;
  urgent_count: number;
  critical_count: number;
  urgency_breakdown: Record<Urgency, number>;
  aging_breakdown: Record<string, number>;
  escalated_items: Array<{
    id: string;
    requester_id: string;
    requester_name: string;
    amount: number;
    urgency: Urgency;
    days_pending: number;
    created_at: string;
  }>;
}

export interface OrgStats {
  org_id: string;
  member_counts: { total: number; org_admin: number; finance: number; staff: number };
  request_stats: {
    total: number;
    pending: number;
    approved: number;
    paid: number;
    rejected: number;
    failed: number;
    withdrawn: number;
  };
  financials: { total_spent: number; this_month_spent: number; currency: Currency };
  plan_usage: {
    current_plan: PlanId;
    requests_this_month: number;
    request_limit: number;
    user_count: number;
    user_limit: number;
  };
}

export interface AuditLogEntry {
  id: string;
  org_id: string;
  actor_id: string;
  actor_name: string;
  action: string;
  target_id: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface Plan {
  id: PlanId;
  name: string;
  price: number;
  request_limit: number;
  user_limit: number;
  features: string[];
}

// --- Response wrappers ---
export interface DataResponse<T> {
  data: T;
}

export interface ListResponse<T> {
  data: T[];
  meta: {
    total: number;
    limit: number;
    offset: number;
    has_more: boolean;
  };
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    total: number;
    page: number;
    limit: number;
    offset: number;
    has_more: boolean;
  };
}

export interface MessageResponse {
  message: string;
}

export interface ErrorResponse {
  error: string;
  code?: string;
}
