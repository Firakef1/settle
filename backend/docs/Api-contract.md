# Settle API contract

What the backend actually returns today. Hosama (organizations, members, invites, approvals, dashboard, audit, billing) and Menweyelet (auth, requests, receipts, comments).

Use this to build and to check a screen. If a call is not in this file, it is not implemented.

**Base URL:** `http://localhost:8080/api/v1`

The same routes also exist under `/v1`. Call `/api/v1`.

**Auth header** (every route below that says “token”):

```http
Authorization: Bearer <token>
Content-Type: application/json
```

**Errors** are almost always:

```json
{ "error": "human readable reason" }
```

Login, when the password is right but the email is not verified:

```json
{ "error": "email address has not been verified", "code": "email_not_verified" }
```

**Success shapes**

| Kind | Body |
|---|---|
| Login, verify, refresh | The object itself. No `data` wrapper. |
| One record | `{ "data": { } }` |
| Request list | `{ "data": [ ], "meta": { } }` |
| Audit log | `{ "data": [ ], "pagination": { } }` |
| A confirmation | `{ "message": "..." }` |

Times are ISO-8601. Money is a number in the organization’s currency.

---

## Words you can hard-code

| Word | Allowed values |
|---|---|
| Role | `staff`, `finance`, `org_admin` |
| Request type | `reimbursement`, `advance`, `stipend` |
| Urgency | `routine`, `urgent`, `critical` |
| Request status | `draft`, `pending`, `approved`, `rejected`, `paid`, `failed`, `withdrawn` |
| Currency | `USD`, `EUR`, `GBP` |
| Plan | `free`, `starter`, `pro` |
| Payment method | `bank_transfer`, `check`, `cash`, `other` |
| Invite role | `staff`, `finance` |
| Receipt mode on resubmit | `carry`, `new` |

A limit of `0` on a plan means unlimited.

---

## The token is one organization

Login and refresh put the user’s **first** membership into the token (`org_id` and `role`). There is no switch-organization call.

After creating an organization, call `POST /api/v1/auth/refresh` before any other screen. Until that refresh, the token has no org and no role. Request calls then have an empty org. Finance and admin screens answer `403`.

Request routes read the org only from the token.

Approvals and `GET /api/v1/dashboard/summary` also accept `?org_id=` or the header `X-Organization-Id` when the token has no org. The role still has to be inside the token.

---

## How a payout moves

`advance` and `stipend` are created as `pending`.

`reimbursement` is created as `draft`. Upload at least one receipt, then submit. A receipt can be attached only while the request is `draft`.

```text
reimbursement:   draft → submit → pending → approved → paid
                                          ↘ rejected
                                          ↘ failed

advance/stipend:              pending → approved → paid
                                      ↘ rejected
                                      ↘ failed

draft or pending → withdrawn     (owner)
rejected or failed → resubmit    (owner, new request id)
```

Request ids look like `REQ-a1b2c3`.

Approve and reject only while status is `pending`. Mark paid and payment failed only while status is `approved` and the payment is still open. Doing it again returns `409`.

---

## 1. Account

### Sign up

`POST /api/v1/auth/signup` → `201`

No token. Does not sign the user in. A code is emailed.

```json
{
  "email": "ada@lab.org",
  "password": "at least 8 characters",
  "name": "Ada Lovelace"
}
```

```json
{
  "message": "verification code sent to your email",
  "user": {
    "id": "uuid",
    "email": "ada@lab.org",
    "name": "Ada Lovelace",
    "status": "active",
    "email_verified": false,
    "created_at": "2026-10-03T12:00:00Z"
  }
}
```

| Status | When |
|---|---|
| 400 | Bad email, missing name, or password shorter than 8 |
| 409 | Email already registered |

### Verify email

`POST /api/v1/auth/verify-email` → `200`

This is the sign-in. Save both tokens. `orgs` is empty until they create or join an organization.

```json
{ "email": "ada@lab.org", "verification_code": "123456" }
```

```json
{
  "message": "email verified",
  "token": "access-jwt",
  "refresh_token": "refresh-token",
  "user": {
    "id": "uuid",
    "email": "ada@lab.org",
    "name": "Ada Lovelace",
    "status": "active",
    "email_verified": true,
    "created_at": "2026-10-03T12:00:00Z"
  },
  "orgs": []
}
```

`400` — invalid or expired code. Body: `{ "error": "invalid or expired verification code" }`.

### Resend code

`POST /api/v1/auth/resend-verification` → `200`

```json
{ "email": "ada@lab.org" }
```

```json
{ "message": "if this account exists and is unverified, a verification code has been sent" }
```

The message is the same when the email is unknown.

### Log in

`POST /api/v1/auth/login` → `200`

```json
{ "email": "ada@lab.org", "password": "secret" }
```

```json
{
  "token": "access-jwt",
  "refresh_token": "refresh-token",
  "user": {
    "id": "uuid",
    "email": "ada@lab.org",
    "name": "Ada Lovelace",
    "status": "active",
    "email_verified": true,
    "created_at": "2026-10-03T12:00:00Z"
  },
  "orgs": [
    {
      "org_id": "uuid",
      "org_name": "Acme Research Lab",
      "org_slug": "acme-research-lab",
      "role": "org_admin",
      "department": ""
    }
  ]
}
```

The access token lasts 24 hours. It carries the first org in `orgs`.

| Status | When |
|---|---|
| 400 | Missing or badly shaped email or password |
| 401 | Wrong email or password |
| 403 | Password is correct and email is not verified. `code` is `email_not_verified`. Send them to verify. |

### Refresh

`POST /api/v1/auth/refresh` → `200`

Same body as login. The old refresh token is dead. Save the new pair.

```json
{ "refresh_token": "the current refresh token" }
```

| Status | When |
|---|---|
| 401 | Missing, expired, or already used. A reused token revokes every refresh token for that user. Send them to login. |
| 403 | Email is not verified. Same `code` as login. |

### Forgot password

`POST /api/v1/auth/forgot-password` → `200`

```json
{ "email": "ada@lab.org" }
```

```json
{ "message": "If that email is registered, a password reset OTP has been sent." }
```

Same message when the email is unknown. `400` when the email format is bad.

### Reset password

`POST /api/v1/auth/reset-password` → `200`

```json
{
  "email": "ada@lab.org",
  "otp": "123456",
  "new_password": "at least 8 characters"
}
```

```json
{ "message": "password reset successfully" }
```

`otp` is exactly 6 digits. A bad or used OTP is `400`.

### Log out

`POST /api/v1/auth/logout` → `200`

No token required. Always returns success. Clear both tokens on the device anyway.

```json
{ "refresh_token": "optional" }
```

```json
{ "message": "logged out successfully" }
```

### Delete account

`DELETE /api/v1/auth/me` → `200`

Token required.

```json
{ "current_password": "their current password" }
```

```json
{ "message": "account deleted successfully" }
```

| Status | When |
|---|---|
| 400 | `current_password` missing |
| 401 | No token |
| 403 | Password does not match |

### Sign in with Google / Microsoft

Browser redirects, not JSON calls. A provider works only when its client ID and secret are set on the server.

`GET /api/v1/auth/oauth/providers` → `200`

```json
{ "data": ["google", "microsoft"] }
```

Empty when nothing is configured. Show a button only for the providers listed.

`GET /api/v1/auth/oauth/:provider/start?next=/requests` → `302` to the provider

Navigate the browser here (a link, not `fetch`). `next` is an optional path to open afterwards. Sets a short-lived HttpOnly cookie.

`GET /api/v1/auth/oauth/:provider/callback` → `302`

The provider sends the browser here. On success it redirects to:

```text
<FRONTEND_URL>/oauth/callback#token=<access>&refresh_token=<refresh>&next=/requests
```

The tokens are in the fragment. Save them, then call `POST /auth/refresh` to load `user` and `orgs`.

On failure it redirects to `<FRONTEND_URL>/login?oauth_error=<code>`:

| Code | When |
|---|---|
| `cancelled` | The user declined at the provider |
| `state` | The sign-in link expired, was reused, or was tampered with |
| `unavailable` | Unknown or unconfigured provider |
| `no_email` | The provider didn't share an email address |
| `account_exists` | An account already uses that email and the provider didn't verify it (Microsoft never marks emails verified) |
| `failed` | Anything else (bad code, provider down) |

Accounts: a provider account is linked to one user. A new email creates a verified user with no usable password (use forgot password to set one). An existing email is linked only when the provider verified it.

---

## 2. Organization, people, plan

Token required, except the plan list and accepting an invite.

### Create organization

`POST /api/v1/organizations` → `201`

The caller becomes `org_admin`. Plan starts as `free`. Slug is generated from the name. Then refresh the token.

```json
{ "name": "Acme Research Lab", "currency": "USD" }
```

```json
{
  "data": {
    "id": "uuid",
    "name": "Acme Research Lab",
    "slug": "acme-research-lab",
    "currency": "USD",
    "plan": "free",
    "created_at": "2026-10-03T12:00:00Z",
    "updated_at": "2026-10-03T12:00:00Z"
  }
}
```

`400` — empty name, name longer than 255 characters, or currency outside `USD`, `EUR`, `GBP`.

### Get organization

`GET /api/v1/organizations/:id` → `200`

Any member. Same `data` object as create.

| Status | When |
|---|---|
| 403 | Caller is not a member |
| 404 | Unknown id |

### Update organization

`PUT /api/v1/organizations/:id` → `200`

`org_admin` only. Send `name`, `currency`, or both. Same `data` object as create.

```json
{ "name": "Acme Lab", "currency": "EUR" }
```

### List members

`GET /api/v1/organizations/:id/members` → `200`

`org_admin` or `finance`.

```json
{
  "data": [
    {
      "user_id": "uuid",
      "name": "Ada Lovelace",
      "email": "ada@lab.org",
      "role": "org_admin",
      "joined_at": "2026-10-03T12:00:00Z"
    }
  ]
}
```

### Invite

`POST /api/v1/organizations/:id/invitations` → `201`

`org_admin` only. No email is sent. Build the link yourself from `token`. It expires in 7 days. You cannot invite someone as `org_admin`.

```json
{ "email": "grace@lab.org", "role": "staff" }
```

```json
{
  "data": {
    "id": "uuid",
    "org_id": "uuid",
    "email": "grace@lab.org",
    "role": "staff",
    "token": "64-character-hex",
    "expires_at": "2026-10-10T12:00:00Z",
    "created_at": "2026-10-03T12:00:00Z"
  }
}
```

`400` — bad email, or role other than `staff` or `finance`.

Suggested link: `/invite/<token>`.

### Accept invite

`POST /api/v1/invitations/:token/accept` → `201`

Public. Does not return a token. Send them to login afterward.

If that email already has an account, `name` and `password` are ignored and that account joins. If the email is new, password must be at least 6 characters. An empty name becomes the part before `@`.

```json
{ "name": "Grace Hopper", "password": "at least 6 characters" }
```

```json
{
  "data": {
    "user_id": "uuid",
    "name": "Grace Hopper",
    "email": "grace@lab.org",
    "role": "staff",
    "joined_at": "2026-10-03T12:00:00Z"
  }
}
```

| Status | When |
|---|---|
| 400 | Token expired, token already used, or new-user password shorter than 6 |
| 404 | Token not found |
| 409 | That user is already in the organization |

A brand-new account created here is not email-verified. Login then returns `403` with `email_not_verified`. An email that was already verified can log in after accepting.

### Change role

`PUT /api/v1/organizations/:id/members/:uid/role` → `200`

`org_admin` only. The new role can be `staff` or `finance`.

```json
{ "role": "finance" }
```

```json
{ "message": "role updated successfully" }
```

| Status | When |
|---|---|
| 400 | Role is not `staff` or `finance`. `org_admin` is rejected. |
| 403 | That person is the last org admin |
| 404 | Member not in this organization |

### Remove member

`DELETE /api/v1/organizations/:id/members/:uid` → `200`

`org_admin` only. The person is deleted from the member list.

```json
{ "message": "member removed successfully" }
```

| Status | When |
|---|---|
| 403 | That person is the last org admin |
| 404 | Member not in this organization |

An admin who is not the last admin can remove their own membership.

### List plans

`GET /api/v1/billing/plans` → `200`

Public. No token.

```json
{
  "data": [
    {
      "id": "free",
      "name": "Free",
      "price": 0,
      "request_limit": 100,
      "user_limit": 5,
      "features": [
        "Up to 5 team members",
        "100 payout requests / month",
        "Standard audit logs",
        "Basic email support"
      ]
    },
    {
      "id": "starter",
      "name": "Starter",
      "price": 49,
      "request_limit": 1000,
      "user_limit": 25,
      "features": [
        "Up to 25 team members",
        "1,000 payout requests / month",
        "Advanced audit log search & export",
        "Priority email support",
        "Automated receipt OCR extraction"
      ]
    },
    {
      "id": "pro",
      "name": "Pro",
      "price": 199,
      "request_limit": 0,
      "user_limit": 0,
      "features": [
        "Unlimited team members",
        "Unlimited payout requests",
        "Full compliance audit exports",
        "Custom approval workflows",
        "24/7 dedicated support"
      ]
    }
  ]
}
```

`price` is a monthly number. `0` on a limit means unlimited. Creating a request does not stop when the monthly limit is passed. Show usage from org stats. Do not expect `403` at the limit.

### Change plan

`PUT /api/v1/organizations/:id/plan` → `200`

`org_admin` only.

```json
{ "plan": "starter" }
```

```json
{
  "data": {
    "org_id": "uuid",
    "plan": "starter",
    "updated_at": "2026-10-03T12:00:00Z"
  }
}
```

`400` — plan is not `free`, `starter`, or `pro`. `404` — unknown organization.

---

## 3. Payout requests

Token required. Org comes from the token.

### Create

`POST /api/v1/requests` → `201`

Any logged-in role can call this. Reimbursement comes back as `draft`. Advance and stipend come back as `pending`.

```json
{
  "type": "reimbursement",
  "amount": 120.5,
  "purpose": "Conference travel, at least 10 characters",
  "urgency": "routine"
}
```

Response is `{ "data": Request }`. Shape is in “Request object” below.

| Rule | |
|---|---|
| Amount | Greater than 0, at most `999999.99` |
| Purpose | 10 to 500 characters |
| Type | `reimbursement`, `advance`, `stipend` |
| Urgency | `routine`, `urgent`, `critical` |

A broken rule is `400` with that rule’s message.

### Upload a receipt

`POST /api/v1/requests/:id/receipts` → `201`

Multipart. Field name is `receipt`. Owner only. Request must be `draft`.

The server stores the filename as `/uploads/receipts/<filename>`. The file bytes are not kept. `ocr_status` stays `pending`. `ocr_results` is absent.

```json
{
  "data": {
    "id": "uuid",
    "file_path": "/uploads/receipts/hotel.jpg",
    "ocr_status": "pending",
    "created_at": "2026-10-03T12:00:00Z"
  }
}
```

| Status | When |
|---|---|
| 400 | No file, or the request is not a draft (`"receipts can only be attached to drafts"`) |
| 403 | Caller does not own the request |
| 404 | Request not in this org |

### Submit

`POST /api/v1/requests/:id/submit` → `200`

Owner only. Draft only. No body. `{ "data": Request }` with status `pending`.

| Status | When |
|---|---|
| 400 | Not a draft, or a reimbursement with no receipt |
| 403 | Caller does not own the request |
| 404 | Request not in this org |

### List

`GET /api/v1/requests` → `200`

Query: `status`, `urgency`, `requester_id`, `sort_by`, `sort_order`, `limit`, `offset`.

`limit` defaults to 20. `offset` defaults to 0. The only sort that is honored is `sort_by=created_at` with `sort_order=asc` or `desc`. Otherwise the list is newest first.

`finance` sees every request in the org except drafts. `staff` sees only their own, and `requester_id` is ignored. `org_admin` is treated like `staff` on this list: only their own requests.

```json
{
  "data": [
    {
      "id": "REQ-a1b2c3",
      "amount": 120.5,
      "purpose": "Conference travel",
      "urgency": "routine",
      "status": "pending",
      "requester": { "id": "uuid", "name": "Ada Lovelace", "email": "ada@lab.org" },
      "days_pending": 2,
      "created_at": "2026-10-03T12:00:00Z"
    }
  ],
  "meta": { "total": 1, "limit": 20, "offset": 0, "has_more": false }
}
```

The row has no `type`. Open the detail for that.

### Detail

`GET /api/v1/requests/:id` → `200`

`finance` can open any non-draft request in the org. `staff` and `org_admin` can open only their own. Someone else’s request, including another person’s draft, is `404`.

```json
{
  "data": {
    "id": "REQ-a1b2c3",
    "type": "reimbursement",
    "amount": 120.5,
    "purpose": "Conference travel",
    "urgency": "urgent",
    "status": "pending",
    "requester": { "id": "uuid", "name": "Ada Lovelace", "email": "ada@lab.org" },
    "receipts": [
      {
        "id": "uuid",
        "file_path": "/uploads/receipts/hotel.jpg",
        "ocr_status": "pending",
        "created_at": "2026-10-03T12:00:00Z"
      }
    ],
    "comments": [
      {
        "id": "uuid",
        "author_id": "uuid",
        "author": "",
        "content": "Please use the new invoice.",
        "created_at": "2026-10-03T12:00:00Z"
      }
    ],
    "timeline": null,
    "submitted_at": "2026-10-03T12:00:00Z",
    "created_at": "2026-10-03T12:00:00Z",
    "updated_at": "2026-10-03T12:00:00Z"
  }
}
```

`approval` is not in this body. `timeline` is `null`. `comments[].author` is an empty string. Use `author_id`. Show the finance decision from the approve, reject, mark-paid, or payment-failed response.

### Withdraw

`PUT /api/v1/requests/:id/withdraw` → `200`

Owner only. Status must be `draft` or `pending`. No body.

```json
{ "message": "request withdrawn" }
```

| Status | When |
|---|---|
| 400 | Status is not draft or pending |
| 403 | Caller does not own the request |
| 404 | Request not in this org |

### Resubmit

`POST /api/v1/requests/:id/resubmit` → `201`

Owner only. Status must be `rejected` or `failed`. Every field is optional. `{ "data": Request }` is the **new** request.

```json
{
  "amount": 90,
  "purpose": "Updated purpose, still 10 to 500 characters",
  "urgency": "urgent",
  "receipt_mode": "carry"
}
```

`receipt_mode` defaults to `carry` (receipts are copied). `new` starts without them. A reimbursement that is `new`, or that has no receipts, comes back as `draft`. Otherwise the new request is `pending`.

| Status | When |
|---|---|
| 400 | Status is not rejected or failed, or an override breaks the amount, purpose, urgency, or receipt_mode rules |
| 403 | Caller does not own the request |
| 404 | Request not in this org |

### Add a comment

`POST /api/v1/requests/:id/comments` → `201`

There is no list-comments route. Comments come back on the detail. Reload the detail after posting.

```json
{ "content": "Please use the new invoice." }
```

```json
{
  "data": {
    "id": "uuid",
    "author_id": "uuid",
    "author": "",
    "content": "Please use the new invoice.",
    "created_at": "2026-10-03T12:00:00Z"
  }
}
```

Empty content is `400`. The request must be in the token’s org (`404` otherwise).

### Request object

Returned by create, submit, resubmit, and detail, inside `data`.

```json
{
  "id": "REQ-a1b2c3",
  "type": "reimbursement",
  "amount": 120.5,
  "purpose": "Conference travel",
  "urgency": "routine",
  "status": "draft",
  "requester": { "id": "uuid", "name": "Ada Lovelace", "email": "ada@lab.org" },
  "receipts": [],
  "comments": [],
  "timeline": null,
  "submitted_at": null,
  "created_at": "2026-10-03T12:00:00Z",
  "updated_at": "2026-10-03T12:00:00Z"
}
```

`submitted_at` is set once the request is `pending`. `approval` is omitted.

---

## 4. Finance decisions

Token required. Role in the token must be `finance` or `org_admin`. Staff is `403`.

All four return `200` and `{ "data": Approval }`.

Optional org when the token has none: `?org_id=<uuid>` or `X-Organization-Id: <uuid>`.

### Approve

`PUT /api/v1/requests/:id/approve`

Body is optional.

```json
{ "note": "Approved for the conference." }
```

Status must be `pending`. Request becomes `approved`. `payment_status` is `pending_payment`. `decision` is `approved`.

### Reject

`PUT /api/v1/requests/:id/reject`

```json
{ "reason": "Missing the hotel invoice." }
```

`reason` is required. Status must be `pending`. Request becomes `rejected`. `decision` is `rejected`. The reason is in `decision_note`.

### Mark paid

`PUT /api/v1/requests/:id/mark-paid`

```json
{ "payment_method": "bank_transfer" }
```

Status must be `approved` and payment must still be open. Request becomes `paid`. `payment_method` is one of `bank_transfer`, `check`, `cash`, `other`.

### Payment failed

`PUT /api/v1/requests/:id/payment-failed`

```json
{ "failure_reason": "Account number was rejected." }
```

`failure_reason` is required. Status must be `approved` and payment must still be open. Request becomes `failed`.

### Approval object

```json
{
  "id": "uuid",
  "request_id": "REQ-a1b2c3",
  "org_id": "uuid",
  "approver_id": "uuid",
  "decision": "approved",
  "decision_note": "Approved for the conference.",
  "payment_status": "pending_payment",
  "payment_method": "",
  "payment_at": null,
  "failure_reason": "",
  "failure_at": null,
  "request_status": "approved",
  "created_at": "2026-10-03T12:00:00Z",
  "updated_at": "2026-10-03T12:00:00Z"
}
```

After mark paid, `payment_status` is `paid`, `payment_method` and `payment_at` are set, and `request_status` is `paid`.

After payment failed, `payment_status` is `failed`, `failure_reason` and `failure_at` are set, and `request_status` is `failed`.

| Status | When |
|---|---|
| 400 | Missing org, missing reason, or payment method outside the four values |
| 401 | No token |
| 403 | Role is not finance or org_admin |
| 404 | Request is not in that org, or there is no approval to update |
| 409 | Approve or reject when status is not `pending`. Mark paid or payment failed when status is not `approved`, or the payment was already settled. |

---

## 5. Dashboard and audit

Token required. Role must be `org_admin` or `finance`.

### Dashboard summary

`GET /api/v1/dashboard/summary` → `200`

Org from the token, or `?org_id=`, or `X-Organization-Id`.

Counts are `pending` requests only. Escalated items are pending and older than 7 days, oldest first, at most 50.

```json
{
  "data": {
    "pending_count": 4,
    "urgent_count": 1,
    "critical_count": 1,
    "urgency_breakdown": { "routine": 2, "urgent": 1, "critical": 1 },
    "aging_breakdown": { "0_to_3_days": 2, "3_to_7_days": 1, "7_plus_days": 1 },
    "escalated_items": [
      {
        "id": "REQ-a1b2c3",
        "requester_id": "uuid",
        "requester_name": "Ada Lovelace",
        "amount": 120.5,
        "urgency": "critical",
        "days_pending": 9,
        "created_at": "2026-10-03T12:00:00Z"
      }
    ]
  }
}
```

`400` when no org can be found. `403` when the caller is not a member, or the token role is not allowed.

### Organization stats

`GET /api/v1/organizations/:id/stats` → `200`

Spent money is the sum of requests with status `paid`.

```json
{
  "data": {
    "org_id": "uuid",
    "member_counts": { "total": 3, "org_admin": 1, "finance": 1, "staff": 1 },
    "request_stats": {
      "total": 10,
      "pending": 2,
      "approved": 1,
      "paid": 4,
      "rejected": 1,
      "failed": 1,
      "withdrawn": 1
    },
    "financials": {
      "total_spent": 800,
      "this_month_spent": 120,
      "currency": "USD"
    },
    "plan_usage": {
      "current_plan": "free",
      "requests_this_month": 6,
      "request_limit": 100,
      "user_count": 3,
      "user_limit": 5
    }
  }
}
```

`403` if the caller is not a member. `404` if the organization does not exist.

### Audit log

`GET /api/v1/organizations/:id/audit-log` → `200`

Query:

| Param | |
|---|---|
| `action` | Exact action string |
| `actor_id` | User id |
| `start_date` | ISO-8601 |
| `end_date` | ISO-8601 |
| `page` | Default 1 |
| `limit` | Default 20, max 100 |
| `offset` | Used when you pass it. Otherwise taken from `page`. |

```json
{
  "data": [
    {
      "id": "uuid",
      "org_id": "uuid",
      "actor_id": "uuid",
      "actor_name": "Ada Lovelace",
      "action": "approved",
      "target_id": "REQ-a1b2c3",
      "metadata": {},
      "created_at": "2026-10-03T12:00:00Z"
    }
  ],
  "pagination": { "total": 1, "page": 1, "limit": 20, "offset": 0, "has_more": false }
}
```

Actions written today: organization create and update, `invite_created`, `invite_accepted`, `member_removed`, `role_updated`, plan change, `approved`, `rejected`, `paid`, `failed`, `request_created`, `request_withdrawn`, `request_resubmitted`, `receipt_uploaded`, and `comment_added`.

Request activity stores the subject in `metadata.target_type` (`request`, `receipt`, or `comment`). `target_id` is that row's id. Submitting a draft does not add its own audit row.

---

## Screen checklist

Copy a row, call it, compare the JSON above.

| Screen | Call |
|---|---|
| Sign up | `POST /auth/signup`, then `POST /auth/verify-email` |
| Resend code | `POST /auth/resend-verification` |
| Login | `POST /auth/login`. On `403` + `email_not_verified`, open verify. |
| Session | `POST /auth/refresh` after create-org, and when the access token expires |
| Forgot / reset | `POST /auth/forgot-password`, then `POST /auth/reset-password` |
| Logout / delete | `POST /auth/logout`, `DELETE /auth/me` |
| Create org | `POST /organizations`, then refresh |
| Org settings | `GET /organizations/:id`, `PUT /organizations/:id` |
| People | `GET /organizations/:id/members`, invite, role, remove |
| Invite link | `POST /organizations/:id/invitations`, then public `POST /invitations/:token/accept`, then login |
| Plans | `GET /billing/plans`, `PUT /organizations/:id/plan` |
| Staff list | `GET /requests` |
| New payout | `POST /requests`. If reimbursement: upload receipt, then `POST /requests/:id/submit` |
| Request page | `GET /requests/:id`, comment, withdraw or resubmit |
| Finance queue | `GET /requests` with a `finance` token |
| Decision | approve, reject, mark-paid, or payment-failed. Keep that response. Detail will not include it. |
| Dashboard | `GET /dashboard/summary` and `GET /organizations/:id/stats` |
| Audit | `GET /organizations/:id/audit-log` |

Paths above are under `/api/v1`.
