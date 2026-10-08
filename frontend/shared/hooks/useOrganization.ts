import { useQuery } from '@tanstack/react-query';
import { orgAPI } from '../services/orgAPI';

// Any member can read the organization (name, currency, plan). Same key as the
// dashboard's query, so the cache is shared.
export function useOrganization(orgId: string | undefined) {
  return useQuery({
    queryKey: ['dashboard', 'org', orgId],
    queryFn: () => orgAPI.get(orgId as string),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });
}
