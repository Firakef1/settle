import api from './api';
import type {
  AuditLogEntry,
  Currency,
  DataResponse,
  Invitation,
  Member,
  MessageResponse,
  Organization,
  OrgStats,
  PaginatedResponse,
  Plan,
  PlanId,
  Role,
} from '../types';

// Shapes and routes follow backend/docs/Api-contract.md, section 2 and 5.

export interface CreateOrganizationRequest {
  name: string;
  currency: Currency;
}

export interface UpdateOrganizationRequest {
  name?: string;
  currency?: Currency;
}

export interface InviteRequest {
  email: string;
  role: Exclude<Role, 'org_admin'>;
}

export interface AcceptInviteRequest {
  name?: string;
  password?: string;
}

export interface PlanChange {
  org_id: string;
  plan: PlanId;
  updated_at: string;
}

export interface AuditLogQuery {
  action?: string;
  actor_id?: string;
  start_date?: string;
  end_date?: string;
  page?: number;
  limit?: number;
}

export const orgAPI = {
  async create(data: CreateOrganizationRequest): Promise<Organization> {
    const response = await api.post<DataResponse<Organization>>('/organizations', data);
    return response.data.data;
  },

  async get(orgId: string): Promise<Organization> {
    const response = await api.get<DataResponse<Organization>>(`/organizations/${orgId}`);
    return response.data.data;
  },

  // org_admin only
  async update(orgId: string, data: UpdateOrganizationRequest): Promise<Organization> {
    const response = await api.put<DataResponse<Organization>>(`/organizations/${orgId}`, data);
    return response.data.data;
  },

  // org_admin or finance
  async getStats(orgId: string): Promise<OrgStats> {
    const response = await api.get<DataResponse<OrgStats>>(`/organizations/${orgId}/stats`);
    return response.data.data;
  },

  // org_admin or finance
  async listMembers(orgId: string): Promise<Member[]> {
    const response = await api.get<DataResponse<Member[]>>(`/organizations/${orgId}/members`);
    return response.data.data;
  },

  // org_admin only. No email is sent: build the link from `token`.
  async invite(orgId: string, data: InviteRequest): Promise<Invitation> {
    const response = await api.post<DataResponse<Invitation>>(`/organizations/${orgId}/invitations`, data);
    return response.data.data;
  },

  // Public. Does not sign the user in.
  async acceptInvite(token: string, data: AcceptInviteRequest): Promise<Member> {
    const response = await api.post<DataResponse<Member>>(`/invitations/${token}/accept`, data);
    return response.data.data;
  },

  // org_admin only. 403 when the target is the last admin.
  async updateRole(orgId: string, userId: string, role: Exclude<Role, 'org_admin'>): Promise<MessageResponse> {
    const response = await api.put<MessageResponse>(`/organizations/${orgId}/members/${userId}/role`, { role });
    return response.data;
  },

  // org_admin only. 403 when the target is the last admin.
  async removeMember(orgId: string, userId: string): Promise<MessageResponse> {
    const response = await api.delete<MessageResponse>(`/organizations/${orgId}/members/${userId}`);
    return response.data;
  },

  // Public
  async listPlans(): Promise<Plan[]> {
    const response = await api.get<DataResponse<Plan[]>>('/billing/plans');
    return response.data.data;
  },

  // org_admin only
  async changePlan(orgId: string, plan: PlanId): Promise<PlanChange> {
    const response = await api.put<DataResponse<PlanChange>>(`/organizations/${orgId}/plan`, { plan });
    return response.data.data;
  },

  // org_admin or finance
  async getAuditLog(orgId: string, query: AuditLogQuery = {}): Promise<PaginatedResponse<AuditLogEntry>> {
    const response = await api.get<PaginatedResponse<AuditLogEntry>>(`/organizations/${orgId}/audit-log`, {
      params: query,
    });
    return response.data;
  },
};
