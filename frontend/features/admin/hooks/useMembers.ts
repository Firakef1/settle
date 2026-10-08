import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { orgAPI, type InviteRequest } from '../../../shared/services/orgAPI';
import { toast } from '../../../shared/stores/notificationStore';
import type { Invitation, Role } from '../../../shared/types';
import { getApiError } from '../../../shared/utils/apiError';

export const memberKeys = {
  list: (orgId: string) => ['members', orgId] as const,
  // There is no list-invitations route; remember the ones created this session.
  invites: (orgId: string) => ['invites', 'session', orgId] as const,
};

export function useMembers(orgId: string) {
  return useQuery({ queryKey: memberKeys.list(orgId), queryFn: () => orgAPI.listMembers(orgId), enabled: !!orgId });
}

export function useSessionInvites(orgId: string) {
  return useQuery<Invitation[]>({ queryKey: memberKeys.invites(orgId), queryFn: () => [], staleTime: Infinity, gcTime: Infinity });
}

function useRefresh(orgId: string) {
  const qc = useQueryClient();
  return () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: memberKeys.list(orgId) }),
      qc.invalidateQueries({ queryKey: ['dashboard'] }),
    ]);
}

export function useInvite(orgId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: InviteRequest) => orgAPI.invite(orgId, body),
    onSuccess: (invite) => {
      qc.setQueryData<Invitation[]>(memberKeys.invites(orgId), (prev = []) => [invite, ...prev.filter((i) => i.email !== invite.email)]);
    },
  });
}

export function useUpdateRole(orgId: string) {
  const refresh = useRefresh(orgId);
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: Exclude<Role, 'org_admin'>; name: string }) => orgAPI.updateRole(orgId, userId, role),
    onSuccess: async (_, { name, role }) => {
      toast.success(`${name} is now ${role === 'finance' ? 'Finance' : 'Staff'}.`);
      await refresh();
    },
    onError: (err) => {
      const { status, message } = getApiError(err);
      toast.error(status === 403 ? "You can't change the role of the last admin." : message);
    },
  });
}

export function useRemoveMember(orgId: string) {
  const refresh = useRefresh(orgId);
  return useMutation({
    mutationFn: ({ userId }: { userId: string; name: string }) => orgAPI.removeMember(orgId, userId),
    onSuccess: async (_, { name }) => {
      toast.success(`${name} was removed from the organization.`);
      await refresh();
    },
    onError: (err) => {
      const { status, message } = getApiError(err);
      toast.error(status === 403 ? "The last admin can't be removed." : message);
    },
  });
}
