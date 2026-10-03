import api from './api';

// Organization Types
export interface Organization {
  id: string;
  name: string;
  slug: string;
  description?: string;
  logo_url?: string;
  created_at: string;
  updated_at: string;
  member_count: number;
  settings: OrganizationSettings;
}

export interface OrganizationSettings {
  require_approval_for_expenses: boolean;
  default_currency: string;
  expense_categories: string[];
  approval_workflow: 'single' | 'multi' | 'department';
  auto_approve_limit?: number;
  notification_preferences: NotificationSettings;
}

export interface NotificationSettings {
  email_notifications: boolean;
  slack_integration: boolean;
  webhook_url?: string;
}

export interface CreateOrganizationRequest {
  name: string;
  slug: string;
  description?: string;
  settings?: Partial<OrganizationSettings>;
}

export interface UpdateOrganizationRequest {
  name?: string;
  description?: string;
  logo_url?: string;
  settings?: Partial<OrganizationSettings>;
}

export interface OrganizationMember {
  id: string;
  user_id: string;
  org_id: string;
  email: string;
  name: string;
  role: 'admin' | 'manager' | 'member';
  department?: string;
  status: 'active' | 'pending' | 'inactive';
  joined_at: string;
  last_active?: string;
}

export interface InviteRequest {
  email: string;
  role: 'admin' | 'manager' | 'member';
  department?: string;
}

export interface InviteResponse {
  id: string;
  email: string;
  role: string;
  department?: string;
  invited_by: string;
  expires_at: string;
  created_at: string;
}

export interface AcceptInviteRequest {
  invite_id: string;
  name: string;
  password: string;
}

export interface UpdateMemberRequest {
  role?: 'admin' | 'manager' | 'member';
  department?: string;
  status?: 'active' | 'inactive';
}

export interface ApiError {
  message: string;
  code?: string;
}

// Organization API functions
export const orgAPI = {
  // Create organization
  async create(data: CreateOrganizationRequest): Promise<Organization> {
    const response = await api.post('/organizations', data);
    return response.data;
  },

  // Get organization details
  async getById(orgId: string): Promise<Organization> {
    const response = await api.get(`/organizations/${orgId}`);
    return response.data;
  },

  // Get user's organizations
  async getUserOrgs(): Promise<Organization[]> {
    const response = await api.get('/organizations');
    return response.data;
  },

  // Update organization
  async update(orgId: string, data: UpdateOrganizationRequest): Promise<Organization> {
    const response = await api.patch(`/organizations/${orgId}`, data);
    return response.data;
  },

  // Delete organization
  async delete(orgId: string): Promise<{ message: string }> {
    const response = await api.delete(`/organizations/${orgId}`);
    return response.data;
  },

  // Invite Management
  async createInvite(orgId: string, data: InviteRequest): Promise<InviteResponse> {
    const response = await api.post(`/organizations/${orgId}/invites`, data);
    return response.data;
  },

  async getInvites(orgId: string): Promise<InviteResponse[]> {
    const response = await api.get(`/organizations/${orgId}/invites`);
    return response.data;
  },

  async cancelInvite(orgId: string, inviteId: string): Promise<{ message: string }> {
    const response = await api.delete(`/organizations/${orgId}/invites/${inviteId}`);
    return response.data;
  },

  async acceptInvite(data: AcceptInviteRequest): Promise<{ message: string; organization: Organization }> {
    const response = await api.post('/invites/accept', data);
    return response.data;
  },

  async getInviteDetails(inviteId: string): Promise<InviteResponse & { org_name: string }> {
    const response = await api.get(`/invites/${inviteId}`);
    return response.data;
  },

  // Member Management
  async getMembers(orgId: string): Promise<OrganizationMember[]> {
    const response = await api.get(`/organizations/${orgId}/members`);
    return response.data;
  },

  async getMember(orgId: string, memberId: string): Promise<OrganizationMember> {
    const response = await api.get(`/organizations/${orgId}/members/${memberId}`);
    return response.data;
  },

  async updateMember(orgId: string, memberId: string, data: UpdateMemberRequest): Promise<OrganizationMember> {
    const response = await api.patch(`/organizations/${orgId}/members/${memberId}`, data);
    return response.data;
  },

  async removeMember(orgId: string, memberId: string): Promise<{ message: string }> {
    const response = await api.delete(`/organizations/${orgId}/members/${memberId}`);
    return response.data;
  },

  // Settings Management
  async updateSettings(orgId: string, settings: Partial<OrganizationSettings>): Promise<Organization> {
    const response = await api.patch(`/organizations/${orgId}/settings`, settings);
    return response.data;
  },

  async getSettings(orgId: string): Promise<OrganizationSettings> {
    const response = await api.get(`/organizations/${orgId}/settings`);
    return response.data;
  }
};
