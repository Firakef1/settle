import { useQuery } from '@tanstack/react-query';
import api from '../../shared/services/api';
import type {
  DashboardSummary,
  DataResponse,
  ListResponse,
  Organization,
  OrgStats,
  RequestListItem,
} from '../../shared/types';

// Query keys live under 'dashboard' so approval mutations (#35) can
// invalidate everything on this page with queryKey: ['dashboard'].
export const dashboardKeys = {
  all: ['dashboard'] as const,
  summary: (orgId: string) => ['dashboard', 'summary', orgId] as const,
  stats: (orgId: string) => ['dashboard', 'stats', orgId] as const,
  org: (orgId: string) => ['dashboard', 'org', orgId] as const,
  myRequests: (orgId: string) => ['dashboard', 'my-requests', orgId] as const,
};

// The staff home counts statuses client-side, so pull a large page in one call.
const MY_REQUESTS_LIMIT = 100;

export const dashboardAPI = {
  async getSummary(): Promise<DashboardSummary> {
    const response = await api.get<DataResponse<DashboardSummary>>('/dashboard/summary');
    return response.data.data;
  },

  async getOrgStats(orgId: string): Promise<OrgStats> {
    const response = await api.get<DataResponse<OrgStats>>(`/organizations/${orgId}/stats`);
    return response.data.data;
  },

  async getOrganization(orgId: string): Promise<Organization> {
    const response = await api.get<DataResponse<Organization>>(`/organizations/${orgId}`);
    return response.data.data;
  },

  async getMyRequests(): Promise<ListResponse<RequestListItem>> {
    const response = await api.get<ListResponse<RequestListItem>>('/requests', {
      params: { limit: MY_REQUESTS_LIMIT, sort_by: 'created_at', sort_order: 'desc' },
    });
    return response.data;
  },
};

// finance / org_admin only — staff get 403 from both endpoints.
export function useDashboardSummary(orgId: string) {
  return useQuery({
    queryKey: dashboardKeys.summary(orgId),
    queryFn: dashboardAPI.getSummary,
  });
}

export function useOrgStats(orgId: string) {
  return useQuery({
    queryKey: dashboardKeys.stats(orgId),
    queryFn: () => dashboardAPI.getOrgStats(orgId),
  });
}

// Any member can read the organization; used for the currency.
export function useOrganization(orgId: string) {
  return useQuery({
    queryKey: dashboardKeys.org(orgId),
    queryFn: () => dashboardAPI.getOrganization(orgId),
    staleTime: 5 * 60 * 1000,
  });
}

export function useMyRequests(orgId: string) {
  return useQuery({
    queryKey: dashboardKeys.myRequests(orgId),
    queryFn: dashboardAPI.getMyRequests,
  });
}
