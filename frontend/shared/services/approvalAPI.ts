import api from './api';
import type { Approval, DataResponse, PaymentMethod } from '../types';

// Finance decisions, backend/docs/Api-contract.md section 4.
// finance or org_admin only. 409 when the request was already decided.
export const approvalAPI = {
  // Status must be pending.
  async approve(id: string, note?: string): Promise<Approval> {
    const response = await api.put<DataResponse<Approval>>(`/requests/${id}/approve`, note ? { note } : {});
    return response.data.data;
  },

  // Status must be pending. Reason is required.
  async reject(id: string, reason: string): Promise<Approval> {
    const response = await api.put<DataResponse<Approval>>(`/requests/${id}/reject`, { reason });
    return response.data.data;
  },

  // Status must be approved with the payment still open.
  async markPaid(id: string, paymentMethod: PaymentMethod): Promise<Approval> {
    const response = await api.put<DataResponse<Approval>>(`/requests/${id}/mark-paid`, { payment_method: paymentMethod });
    return response.data.data;
  },

  // Status must be approved with the payment still open. Reason is required.
  async paymentFailed(id: string, failureReason: string): Promise<Approval> {
    const response = await api.put<DataResponse<Approval>>(`/requests/${id}/payment-failed`, { failure_reason: failureReason });
    return response.data.data;
  },
};
