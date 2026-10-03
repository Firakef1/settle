import api from './api';

// Auth API Types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface SignupRequest {
  name: string;
  email: string;
  password: string;
}

export interface VerifyEmailRequest {
  email: string;
  verification_code: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  email: string;
  otp: string;
  new_password: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: {
    id: string;
    email: string;
    name: string;
    status: string;
    email_verified: boolean;
    created_at: string;
  };
  orgs: Array<{
    org_id: string;
    org_name: string;
    org_slug: string;
    role: string;
    department?: string;
  }>;
}

export interface ApiError {
  message: string;
  code?: string;
}

// Auth API functions
export const authAPI = {
  // Login
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await api.post('/auth/login', credentials);
    return response.data;
  },

  // Signup
  async signup(data: SignupRequest): Promise<{ message: string }> {
    const response = await api.post('/auth/signup', data);
    return response.data;
  },

  // Verify email
  async verifyEmail(data: VerifyEmailRequest): Promise<LoginResponse> {
    const response = await api.post('/auth/verify-email', data);
    return response.data;
  },

  // Forgot password
  async forgotPassword(data: ForgotPasswordRequest): Promise<{ message: string }> {
    const response = await api.post('/auth/forgot-password', data);
    return response.data;
  },

  // Reset password
  async resetPassword(data: ResetPasswordRequest): Promise<{ message: string }> {
    const response = await api.post('/auth/reset-password', data);
    return response.data;
  },

  // Refresh token
  async refreshToken(data: RefreshRequest): Promise<LoginResponse> {
    const response = await api.post('/auth/refresh', data);
    return response.data;
  },

  // Logout
  async logout(refreshToken: string): Promise<{ message: string }> {
    const response = await api.post('/auth/logout', { refresh_token: refreshToken });
    return response.data;
  },

  // Resend verification code
  async resendVerification(email: string): Promise<{ message: string }> {
    const response = await api.post('/auth/resend-verification', { email });
    return response.data;
  }
};