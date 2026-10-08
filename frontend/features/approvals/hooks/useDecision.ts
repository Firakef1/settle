import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { approvalAPI } from '../../../shared/services/approvalAPI';
import { requestAPI } from '../../../shared/services/requestAPI';
import { toast } from '../../../shared/stores/notificationStore';
import type { Approval, PaymentMethod } from '../../../shared/types';
import { getApiError } from '../../../shared/utils/apiError';

export type DecisionKind = 'approve' | 'reject' | 'paid' | 'failed';

export interface DecisionInput {
  kind: DecisionKind;
  requestId: string;
  text?: string;
  method?: PaymentMethod;
}

const DONE: Record<DecisionKind, string> = {
  approve: 'approved',
  reject: 'rejected',
  paid: 'marked as paid',
  failed: 'marked as payment failed',
};

// The request detail doesn't include the approval, so keep the latest one we
// got back from a decision call (for this session).
export const approvalKey = (requestId: string) => ['approvals', 'latest', requestId] as const;

export function useLatestApproval(requestId: string) {
  return useQuery<Approval | null>({
    queryKey: approvalKey(requestId),
    queryFn: () => null,
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

export function useDecision() {
  const queryClient = useQueryClient();
  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['requests'] }),
      queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
    ]);

  return useMutation({
    mutationFn: async ({ kind, requestId, text, method }: DecisionInput) => {
      let approval: Approval;
      switch (kind) {
        case 'approve':
          approval = await approvalAPI.approve(requestId, text);
          break;
        case 'reject':
          approval = await approvalAPI.reject(requestId, text ?? '');
          break;
        case 'paid':
          approval = await approvalAPI.markPaid(requestId, method ?? 'bank_transfer');
          break;
        case 'failed':
          approval = await approvalAPI.paymentFailed(requestId, text ?? '');
          break;
      }
      // The request detail doesn't include the approval, so the requester would
      // never see why. Post the reason (or approval note) as a comment too.
      const note =
        kind === 'reject' ? `Rejected: ${text}` : kind === 'failed' ? `Payment failed: ${text}` : kind === 'approve' && text ? `Approved: ${text}` : '';
      if (note) await requestAPI.addComment(requestId, note).catch(() => undefined);
      return approval;
    },
    onSuccess: async (approval, { kind, requestId }) => {
      queryClient.setQueryData(approvalKey(requestId), approval);
      toast.success(`${requestId} ${DONE[kind]}.`);
      await refresh();
    },
    onError: async (error, { requestId }) => {
      const { status, message } = getApiError(error);
      if (status === 409) {
        toast.info(`${requestId} was already decided by someone else. Refreshing.`);
        await refresh();
      } else {
        toast.error(message);
      }
    },
  });
}
