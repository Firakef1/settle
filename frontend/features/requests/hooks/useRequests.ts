import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { requestAPI, type ListRequestsQuery } from '../../../shared/services/requestAPI';
import type { RequestStatus } from '../../../shared/types';

export const requestKeys = {
  all: ['requests'] as const,
  list: (orgId: string, query: ListRequestsQuery) => ['requests', 'list', orgId, query] as const,
  count: (orgId: string, status: RequestStatus) => ['requests', 'count', orgId, status] as const,
  detail: (id: string) => ['requests', 'detail', id] as const,
};

export function useRequestList(orgId: string, query: ListRequestsQuery) {
  return useQuery({
    queryKey: requestKeys.list(orgId, query),
    queryFn: () => requestAPI.list(query),
    placeholderData: keepPreviousData,
  });
}

// Total for one status, using meta.total from a 1-row page.
export function useRequestCount(orgId: string, status: RequestStatus) {
  return useQuery({
    queryKey: requestKeys.count(orgId, status),
    queryFn: async () => (await requestAPI.list({ status, limit: 1 })).meta.total,
  });
}

export function useRequestDetail(id: string) {
  return useQuery({
    queryKey: requestKeys.detail(id),
    queryFn: () => requestAPI.get(id),
    retry: (count, error) => (error as { response?: { status?: number } })?.response?.status !== 404 && count < 2,
  });
}

// Refresh every list, count, detail and dashboard after a change.
export function useInvalidateRequests() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: requestKeys.all }),
      queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
    ]);
}

export function useAddComment(id: string) {
  const invalidate = useInvalidateRequests();
  return useMutation({
    mutationFn: (content: string) => requestAPI.addComment(id, content),
    onSuccess: invalidate,
  });
}
