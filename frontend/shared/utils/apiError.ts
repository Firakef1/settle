// The backend returns errors as { error, code? } (backend/docs/Api-contract.md).
// A few older handlers use { message }, so fall back to that.
export interface ApiErrorInfo {
  status?: number;
  code?: string;
  message: string;
}

export function getApiError(error: unknown, fallback = 'Something went wrong. Please try again.'): ApiErrorInfo {
  const response = (error as { response?: { status?: number; data?: { error?: string; message?: string; code?: string } } })
    ?.response;
  return {
    status: response?.status,
    code: response?.data?.code,
    message: response?.data?.error || response?.data?.message || fallback,
  };
}

export function apiErrorMessage(error: unknown, fallback?: string): string {
  return getApiError(error, fallback).message;
}
