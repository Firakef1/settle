# Implementation Plan - Account Management

## Scope

Implement two account management capabilities:

1. Password reset flow with:

- Forgot password request
- OTP generation and delivery via email
- OTP verification and password update

2. Account deletion flow where:

- Authenticated users can delete only their own account

---

## Goals and Non-Goals

### Goals

- Provide secure password reset using short-lived OTP sent by email.
- Prevent user enumeration and OTP abuse.
- Allow authenticated self-service account deletion with clear auditability.
- Keep implementation aligned with existing domain structure (`internal/auth`, `internal/shared`, validators, handlers, repositories, services).

### Non-Goals

- Admin-driven deletion of other users.
- Account recovery after deletion.
- Multi-channel OTP delivery (SMS, WhatsApp, etc.).

---

## Proposed Functional Design

## 1) Password Reset

### User Journey

1. User clicks "Forgot password".
2. User submits email.
3. System responds with generic success message regardless of email existence.
4. If email exists, system sends a 6-digit OTP to email.
5. User submits email + OTP + new password.
6. System validates OTP and password policy, then updates password.
7. OTP becomes single-use; all active refresh tokens for the user are revoked.

### API Endpoints (Auth domain)

- `POST /auth/forgot-password`
  - Request: `{ "email": "user@example.com" }`
  - Response: `200` with generic message.

- `POST /auth/reset-password`
  - Request: `{ "email": "user@example.com", "otp": "123456", "new_password": "..." }`
  - Response: `200` on success, `400` for invalid/expired OTP or invalid payload.

### Security Rules

- OTP length: 6 digits (numeric).
- OTP TTL: default 10-15 minutes (configurable).
- Store OTP as keyed hash (HMAC-SHA256) instead of plaintext.
- Constant-time hash comparison.
- Single active OTP per user per purpose (`password_reset`).
- Max retry attempts per OTP (for example 5), then invalidate OTP.
- Rate limit forgot-password endpoint (IP + email based).
- Generic response to prevent account enumeration.
- Revoke all user refresh tokens on successful reset.

---

## 2) Account Deletion (Self-Service)

### User Journey

1. Authenticated user opens account settings and chooses "Delete account".
2. User confirms intent (recommended: require current password in request).
3. System validates authentication and ownership.
4. System performs transactional delete/deactivation and token revocation.
5. User session is invalidated and subsequent requests require new auth.

### API Endpoint

- `DELETE /auth/me`
  - Auth: required
  - Request options:
    - Minimal: no body
    - Stronger security (recommended): `{ "current_password": "..." }`
  - Response: `200` with confirmation message

### Ownership and Authorization

- User identity is taken from JWT/session context only.
- No user ID in path/body for deletion endpoint.
- Handler must not permit deleting other accounts.

### Deletion Strategy

Soft delete (mark as deleted, disable login, retain references)

---

## Data Model and Migration Plan

## 1) Password Reset OTP Table

Add new migration, for example:

- `migrations/014_create_password_reset_otps.up.sql`
- `migrations/014_create_password_reset_otps.down.sql`

Suggested table:

- `password_reset_otps`
  - `user_id UUID NOT NULL` (FK -> users.id)
  - `otp_hash TEXT NOT NULL`
  - `expires_at TIMESTAMPTZ NOT NULL`
  - `attempt_count INT NOT NULL DEFAULT 0`
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - Unique active row per user (either PK on user_id or unique index)

Optional fields:

- `last_sent_at TIMESTAMPTZ` for resend cooldown

## 2) Account Deletion Fields

If soft delete:

- Add to `users`:
  - `deleted_at TIMESTAMPTZ NULL`
  - `status` update convention (e.g. `active` -> `deleted`)

Ensure all auth queries exclude deleted users.

---

## Backend Implementation Breakdown

## 1) Shared Email Capability

Reuse existing email transport at:

- `backend/internal/shared/service/email_transport.go`

Add/reset template helper for password reset OTP:

- subject: "Reset your password"
- body includes OTP, TTL, and security notice

## 2) Auth DTO / Validator / Handler / Service / Repository

Follow existing domain pattern under `backend/internal/auth/`:

- DTO
  - forgot-password request
  - reset-password request
  - delete-account request (if password confirmation required)

- Validator
  - email format validation
  - OTP format validation (6 numeric chars)
  - password policy validation

- Repository
  - create/update OTP row
  - fetch active OTP by user
  - increment attempt count
  - consume/delete OTP on success

- Service
  - forgot password use case
  - reset password use case
  - delete own account use case

- Handler
  - wire new endpoints to service calls
  - map domain errors to HTTP status codes consistently

## 3) Router

Register routes in router module:

- Public:
  - `POST /auth/forgot-password`
  - `POST /auth/reset-password`

- Protected:
  - `DELETE /auth/me`

## 4) Token and Session Invalidation

On successful password reset or account deletion:

- Revoke all refresh tokens for that user.
- For access tokens, rely on short TTL and token refresh revocation.

---

## Configuration and Environment

Add/update backend config keys:

- `PASSWORD_RESET_OTP_TTL` (e.g. `15m`)
- `PASSWORD_RESET_MAX_ATTEMPTS` (e.g. `5`)
- `PASSWORD_RESET_RESEND_COOLDOWN` (optional)
- Existing email settings must be enabled in all envs.

Add startup validation for required config.

---

## Error Handling and API Contract

Recommended status behavior:

- `POST /auth/forgot-password`
  - Always `200` with same response body.

- `POST /auth/reset-password`
  - `200` on success
  - `400` invalid payload / invalid OTP / expired OTP
  - `429` for throttling

- `DELETE /auth/me`
  - `200` on success
  - `401` if unauthenticated
  - `403` if policy requires current password and validation fails

Document request/response in Swagger and frontend API contract docs.

---

## Testing Plan

## Backend Tests

Add/extend tests in `backend/tests/` (auth-focused):

- forgot-password returns 200 for existing and non-existing email
- OTP generated and email delivery invoked for existing user
- reset-password success updates password hash
- reset-password rejects invalid/expired OTP
- OTP attempt limit enforced
- refresh tokens revoked after password reset
- authenticated user can delete own account
- unauthenticated deletion rejected
- deleted user cannot login anymore (soft delete path)

## Risks and Mitigations

- Risk: email delays or failures block resets.
  - Mitigation: retries, monitoring, and clear user guidance.

- Risk: OTP brute force attempts.
  - Mitigation: attempt caps, rate limits, short TTL.

- Risk: deleting user breaks relational integrity.
  - Mitigation: prefer soft delete initially, validate downstream query filters.

---

## Acceptance Criteria

1. Users can initiate forgot-password with email and always receive a generic success response.
2. Existing users receive OTP email and can reset password within TTL.
3. OTP is single-use, hashed in storage, and rate-limited.
4. Password reset revokes refresh tokens.
5. Authenticated users can delete only their own account.
6. Deleted users cannot continue authenticated activity.
7. API docs and tests are updated and passing.

---

## Suggested Task Sequencing

1. DB migrations for OTP and deletion fields.
2. Auth repository/service implementation.
3. Handlers, validators, and route wiring.
4. Email template and delivery integration.
5. Swagger/API contract updates.
6. Tests, hardening, and rollout verification.
