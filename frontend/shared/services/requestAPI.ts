import api from './api';
import type {
  Comment,
  DataResponse,
  ListResponse,
  MessageResponse,
  Receipt,
  Request,
  RequestListItem,
  RequestStatus,
  RequestType,
  Urgency,
} from '../types';

// Routes and shapes follow backend/docs/Api-contract.md, section 3.

export interface CreateRequestBody {
  type: RequestType;
  amount: number;
  purpose: string;
  urgency: Urgency;
}

export interface ListRequestsQuery {
  status?: RequestStatus;
  urgency?: Urgency;
  requester_id?: string;
  sort_by?: 'created_at';
  sort_order?: 'asc' | 'desc';
  limit?: number;
  offset?: number;
}

export interface ResubmitBody {
  amount?: number;
  purpose?: string;
  urgency?: Urgency;
  receipt_mode?: 'carry' | 'new';
}

// Validation rules from the contract, shared by the create and resubmit forms.
export const REQUEST_RULES = {
  minAmount: 0.01,
  maxAmount: 999999.99,
  minPurpose: 10,
  maxPurpose: 500,
  maxReceiptBytes: 10 * 1024 * 1024,
  receiptTypes: ['image/jpeg', 'image/png', 'application/pdf'],
} as const;

export const requestAPI = {
  // Reimbursement comes back as draft; advance and stipend as pending.
  async create(body: CreateRequestBody): Promise<Request> {
    const response = await api.post<DataResponse<Request>>('/requests', body);
    return response.data.data;
  },

  // Owner only, draft only. Multipart field name is `receipt`.
  async uploadReceipt(id: string, file: File): Promise<Receipt> {
    const form = new FormData();
    form.append('receipt', file);
    const response = await api.post<DataResponse<Receipt>>(`/requests/${id}/receipts`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data.data;
  },

  // Owner only, draft only. A reimbursement needs at least one receipt.
  async submit(id: string): Promise<Request> {
    const response = await api.post<DataResponse<Request>>(`/requests/${id}/submit`);
    return response.data.data;
  },

  // finance: every non-draft request in the org. staff and org_admin: their own.
  async list(query: ListRequestsQuery = {}): Promise<ListResponse<RequestListItem>> {
    const response = await api.get<ListResponse<RequestListItem>>('/requests', { params: query });
    return response.data;
  },

  async get(id: string): Promise<Request> {
    const response = await api.get<DataResponse<Request>>(`/requests/${id}`);
    return response.data.data;
  },

  // Owner only. Draft or pending.
  async withdraw(id: string): Promise<MessageResponse> {
    const response = await api.put<MessageResponse>(`/requests/${id}/withdraw`);
    return response.data;
  },

  // Owner only. Rejected or failed. Returns the NEW request.
  async resubmit(id: string, body: ResubmitBody): Promise<Request> {
    const response = await api.post<DataResponse<Request>>(`/requests/${id}/resubmit`, body);
    return response.data.data;
  },

  // Comments come back on the detail; reload it after posting.
  async addComment(id: string, content: string): Promise<Comment> {
    const response = await api.post<DataResponse<Comment>>(`/requests/${id}/comments`, { content });
    return response.data.data;
  },
};
