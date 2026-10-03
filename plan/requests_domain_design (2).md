# Requests Domain — Implementation Design

## 1. Domain Overview

The **Requests domain** owns everything directly related to a staff payout request.

This domain combines:

- Payout request creation, submission, and retrieval
- Request listing and filtering
- Request withdrawal
- Request resubmission
- Receipt upload (multiple per request) and listing
- Receipt OCR status/results
- Request comments

Approval and Audit are separate domains owned by other developers. The Requests domain may interact with them through their public interfaces, but does **not** define or implement their internal business logic.

### Domain boundary

```text
Requests Domain
│
├── Request
│   ├── Create (draft or pending)
│   ├── Submit (draft → pending)
│   ├── List
│   ├── Get detail
│   ├── Get requester history
│   ├── Withdraw
│   └── Resubmit (carry or replace receipts)
│
├── Receipt (child entity of Request)
│   ├── Upload multiple receipts
│   ├── List request receipts
│   └── OCR status/results
│
└── Comment
    ├── Add
    └── List
```

---

# 2. API Routes

All resources are exposed under the Requests domain.

## Requests

| Method | Route | Description |
|---|---|---|
| `POST` | `/requests` | Create a new payout request (`draft` for reimbursement, `pending` otherwise) |
| `POST` | `/requests/:id/submit` | Submit a draft request for approval (`draft` → `pending`) |
| `GET` | `/requests` | List requests |
| `GET` | `/requests/:id` | Get full request details |
| `GET` | `/requests/:id/previous` | Get requester's previous requests |
| `PUT` | `/requests/:id/withdraw` | Withdraw a draft or pending request |
| `POST` | `/requests/:id/resubmit` | Resubmit a rejected/failed request as a new request |

## Receipts

| Method | Route | Description |
|---|---|---|
| `POST` | `/requests/:id/receipts` | Upload a receipt to a draft request |
| `GET` | `/requests/:id/receipts` | List all receipts attached to a request |

## Comments

| Method | Route | Description |
|---|---|---|
| `POST` | `/requests/:id/comments` | Add a comment |
| `GET` | `/requests/:id/comments` | List request comments |

---

# 3. Directory Structure

Recommended implementation structure:

```text
internal/requests/
├── handler/
│   └── request_handler.go
│
├── service/
│   └── request_service.go
│
├── repository/
│   └── request_repo.go
│
├── dto/
│   ├── create_request.go
│   ├── resubmit_request.go
│   ├── request_response.go
│   ├── request_list_response.go
│   ├── comment_request.go
│   └── receipt_response.go
│
├── validator/
│   └── request_validator.go
│
└── model/
    └── request.go
```

### Responsibility of each layer

```text
Handler
  ↓
Parse HTTP request
Validate basic input
Extract auth context
Call service
Build HTTP response

Service
  ↓
Business rules
Authorization rules
Request lifecycle
Transactions
Coordinates repository/file service

Repository
  ↓
Database access only

DTO
  ↓
HTTP input/output structures

Validator
  ↓
Input validation rules

Model
  ↓
Domain/database representation
```

---

# 4. Handler

Use one handler for the entire Requests domain.

```go
type RequestHandler struct {
    service *service.RequestService
}
```

Methods:

```go
func (h *RequestHandler) Create(c *gin.Context)
func (h *RequestHandler) List(c *gin.Context)
func (h *RequestHandler) GetDetail(c *gin.Context)
func (h *RequestHandler) GetPrevious(c *gin.Context)

func (h *RequestHandler) Submit(c *gin.Context)
func (h *RequestHandler) Withdraw(c *gin.Context)
func (h *RequestHandler) Resubmit(c *gin.Context)

func (h *RequestHandler) UploadReceipt(c *gin.Context)
func (h *RequestHandler) ListReceipts(c *gin.Context)

func (h *RequestHandler) AddComment(c *gin.Context)
func (h *RequestHandler) ListComments(c *gin.Context)
```

The handler should **not** contain business logic.

For example:

```go
func (h *RequestHandler) Withdraw(c *gin.Context) {
    requestID := c.Param("id")

    userID := getUserID(c)
    orgID := getOrgID(c)

    err := h.service.Withdraw(
        c.Request.Context(),
        orgID,
        requestID,
        userID,
    )

    if err != nil {
        // map service error to HTTP response
        return
    }

    // return success
}
```

The handler should not decide whether the request is actually withdrawable. That belongs to the service.

---

# 5. Service

Use one service for the domain:

```go
type RequestService struct {
    repo        RequestRepository
    fileService FileService
    // external/domain interfaces may be added here
}
```

The service exposes three logical groups of operations.

## Request operations

```go
Create(...)
List(...)
GetDetail(...)
GetPrevious(...)
Submit(...)
Withdraw(...)
Resubmit(...)
```

## Receipt operations

Receipts are child entities of a Request. A request may have zero or many receipts.

```go
UploadReceipt(...)
ListReceipts(...)
```

## Comment operations

```go
AddComment(...)
ListComments(...)
```

The methods remain part of one service because receipts and comments exist specifically around Requests.

---

# 6. Request Creation

## Input

```go
type CreateRequestDTO struct {
    Type    string  `json:"type"`
    Amount  float64 `json:"amount"`
    Purpose string  `json:"purpose"`
    Urgency string  `json:"urgency"`
}
```

## Business rules

When creating a request:

1. Amount must be greater than `0`.
2. Amount must be less than or equal to `999999.99`.
3. Purpose must contain `10-500` characters.
4. Type must be one of:
   - `reimbursement`
   - `advance`
   - `stipend`
5. Urgency must be one of:
   - `routine`
   - `urgent`
   - `critical`
6. A reimbursement cannot be submitted (`draft` → `pending`) until it has at least one receipt. Receipts are attached through the receipt endpoint while the request is still a draft (see §6A).
7. The request belongs to the authenticated user.
8. The request belongs to the authenticated user's organization.
9. A new request ID must be generated.
10. Initial status depends on the type: `draft` for `reimbursement`, `pending` for `advance` and `stipend`.

## Service flow

```text
Create
 │
 ├── Validate input
 │
 ├── Validate type/urgency
 │
 ├── Determine initial status (draft for reimbursement, else pending)
 │
 ├── Generate request ID
 │
 ├── INSERT request
 │
 └── Return created request
```

---

# 6A. Request Status Lifecycle and Submit

## Statuses owned by this domain

| Status | Meaning | Set by |
|---|---|---|
| `draft` | Created but not yet submitted. Receipts can be attached. Visible only to the owner. | Create (reimbursement), Resubmit (when receipts are missing) |
| `pending` | Submitted and waiting for approval. Receipts are locked. | Submit, Create (advance/stipend), Resubmit |
| `withdrawn` | Cancelled by the owner. | Withdraw |

Other statuses (`approved`, `rejected`, `failed`, `paid`, ...) belong to the Approval/payment flow and are only read by this domain.

```text
Create (reimbursement) ──► draft ──── Submit ────► pending ──► (Approval domain)
Create (advance/stipend) ───────────────────────► pending
draft ──── Withdraw ────► withdrawn
pending ── Withdraw ────► withdrawn
```

## Submit

Endpoint:

```text
POST /requests/:id/submit
```

Only the request owner can submit.

Only a `draft` request can be submitted.

A reimbursement must have at least one receipt.

### Service flow

```text
Submit
   │
   ├── Find request (lock row)
   ├── Verify organization
   ├── Verify requester == authenticated user
   ├── Verify status == draft
   ├── If type == reimbursement: verify receipt count >= 1
   ├── UPDATE status = pending, submitted_at = now
   ├── Emit audit event (via Audit interface)
   └── Return request
```

Errors:

```text
not draft             → ErrCannotSubmit     (409)
no receipt            → ErrReceiptRequired  (400)
not owner             → ErrUnauthorizedRequest (403)
```

Once submitted, the request is `pending` and its receipts are locked.

## Draft visibility

Drafts are private working copies:

- The owner sees their own drafts in list and detail.
- Finance and org admin never see drafts. List queries exclude them, and detail/previous return `404` for a draft they do not own.

---

# 7. Request ID Generation

Request IDs use:

```text
REQ-XXXXXX
```

Example:

```text
REQ-000001
REQ-000002
REQ-000042
REQ-001024
```

Prefer a PostgreSQL sequence.

Example:

```sql
CREATE SEQUENCE request_id_seq;
```

Then:

```sql
SELECT nextval('request_id_seq');
```

Go:

```go
func formatRequestID(sequence int64) string {
    return fmt.Sprintf("REQ-%06d", sequence)
}
```

Avoid `COUNT(*) + 1` in the final implementation because concurrent requests can generate duplicate IDs.

---

# 8. Request List

## Access rules

### Staff

A staff user can only see their own requests:

```sql
requester_id = userID
```

### Finance

Finance can see all non-draft requests belonging to their organization.

### Org Admin

Org admin can see all non-draft requests belonging to their organization.

Drafts are visible only to their owner. For finance/admin the service sets `ExcludeDrafts = true` before building repository filters.

The service should enforce these rules before building repository filters.

---

# 9. List Filters

Recommended filters:

```go
type ListFilters struct {
    Status      string
    Urgency     string
    RequesterID string

    ExcludeDrafts bool // set by the service for finance/admin, never by the client

    SortBy      string
    SortOrder   string

    Limit       int
    Offset      int
}
```

The service should ensure that a staff user cannot use `RequesterID` to access another user's requests.

For example:

```text
staff
  ↓
force requester_id = authenticated user ID

finance/admin
  ↓
requester_id filter may be supplied
```

---

# 10. Request List Response

```go
type RequestListResponse struct {
    Data []RequestListItem `json:"data"`
    Meta PaginationMeta    `json:"meta"`
}
```

```go
type PaginationMeta struct {
    Total   int  `json:"total"`
    Limit   int  `json:"limit"`
    Offset  int  `json:"offset"`
    HasMore bool `json:"has_more"`
}
```

Example:

```json
{
  "data": [
    {
      "id": "REQ-000042",
      "amount": 2500,
      "purpose": "Travel reimbursement for...",
      "urgency": "urgent",
      "status": "pending",
      "requester": {
        "id": "user-123",
        "name": "John Doe",
        "email": "john@example.com"
      },
      "days_pending": 3,
      "created_at": "2026-10-01T10:00:00Z"
    }
  ],
  "meta": {
    "total": 12,
    "limit": 20,
    "offset": 0,
    "has_more": false
  }
}
```

---

# 11. Request List Query

The repository should support organization, ownership, status, urgency, requester, sorting, pagination, and total count.

Conceptually:

```sql
SELECT
    r.id,
    r.amount,
    r.purpose,
    r.urgency,
    r.status,
    r.created_at,
    u.id AS requester_id,
    u.name AS requester_name,
    u.email AS requester_email,
    EXTRACT(
        DAY FROM NOW() - COALESCE(r.submitted_at, r.created_at)
    )::int AS days_pending
FROM requests r
JOIN users u ON u.id = r.requester_id
WHERE r.org_id = $1
  AND ($2 = '' OR r.status = $2)
  AND ($3 = '' OR r.urgency = $3)
  AND ($4 = '' OR r.requester_id = $4)
ORDER BY
    CASE r.urgency
        WHEN 'critical' THEN 1
        WHEN 'urgent' THEN 2
        WHEN 'routine' THEN 3
    END ASC,
    r.created_at ASC
LIMIT $5
OFFSET $6;
```

The exact SQL should be adapted to the project's actual schema.

---

# 12. Get Request Detail

Endpoint:

```text
GET /requests/:id
```

The detail response should contain the request and its related information.

Recommended shape:

```go
type RequestResponse struct {
    ID        string  `json:"id"`
    Type      string  `json:"type"`
    Amount    float64 `json:"amount"`
    Purpose   string  `json:"purpose"`
    Urgency   string  `json:"urgency"`
    Status    string  `json:"status"`

    Requester RequesterResponse `json:"requester"`

    Receipts []ReceiptResponse `json:"receipts"`
    Approval *ApprovalResponse `json:"approval,omitempty"`

    Comments []CommentResponse `json:"comments"`
    Timeline []TimelineEvent   `json:"timeline"`

    SubmittedAt *time.Time `json:"submitted_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

`ApprovalResponse` is only the response representation needed by this domain. Approval business logic remains outside this domain.

---

# 13. Detail Access Control

Before returning detail:

```text
Does request exist?
       ↓
Does request belong to user's organization?
       ↓
       YES
       ↓
Is user staff?
   /          \
 YES           NO
  ↓             ↓
owner?       finance/admin?
  ↓             ↓
 YES           YES
  ↓             ↓
ALLOW        ALLOW
```

Staff must not be able to retrieve another staff member's request by changing the URL ID.

Draft requests are only visible to their owner. Finance/admin receive `404` for a draft they do not own, so its existence is not revealed.

---

# 14. Previous Requests

Endpoint:

```text
GET /requests/:id/previous
```

Purpose:

> Give finance users context about the requester's previous payout requests.

Flow:

```text
Current Request
      ↓
Find requester_id
      ↓
Query other requests
      ↓
Exclude current request
      ↓
Return requester history
```

Recommended response:

```go
type PreviousRequestsResponse struct {
    Data []RequestListItem `json:"data"`
}
```

Access should follow the same organization and role rules as request detail. Draft requests are excluded from the history.

---

# 15. Withdraw

Endpoint:

```text
PUT /requests/:id/withdraw
```

Only the request owner can withdraw.

Only a `draft` or `pending` request can be withdrawn.

## Service flow

```text
Withdraw
   │
   ├── Find request
   │
   ├── Verify organization
   │
   ├── Verify requester == authenticated user
   │
   ├── Verify status is draft or pending
   │
   └── UPDATE status = withdrawn
```

Use a transaction if the status change must be coordinated with other request-domain writes.

The Requests domain should not implement Audit internals. If an audit event is required, call the Audit domain/interface provided by the other developer.

---

# 16. Resubmit

Endpoint:

```text
POST /requests/:id/resubmit
```

Resubmission creates a **new request**.

The old request remains in its original state.

Example:

```text
REQ-000041
status = rejected
        │
        │ resubmit
        ▼
REQ-000042
status = pending (or draft, see below)
```

The old request gets:

```text
resubmitted_as = REQ-000042
```

## Allowed source statuses

```text
rejected
failed
```

## Service flow

```text
Resubmit
   │
   ├── Get old request
   ├── Verify organization
   ├── Verify requester == userID
   ├── Verify status is rejected/failed
   ├── Generate new request ID
   ├── Copy original fields
   ├── Apply optional overrides
   ├── Validate the resulting request
   ├── Determine resulting status (see below)
   ├── Create new request
   ├── Copy receipt records if receipt_mode = carry
   ├── Link old request → new request
   └── Return new request
```

This must be transactional.

## Receipts on resubmission

The client chooses how receipts are handled with `receipt_mode`:

| Mode | Behavior |
|---|---|
| `carry` (default) | Receipt records of the old request are copied to the new request. They reference the same stored file and keep their OCR results. |
| `new` | The new request starts with no receipts. The user attaches new ones while it is a draft. |

The old request keeps its own receipts unchanged in both modes.

## Resulting status

```text
type != reimbursement                          → pending
reimbursement + carry (old had >= 1 receipt)   → pending
reimbursement + new                            → draft  (attach receipts, then Submit)
reimbursement + carry (old had 0 receipts)     → draft
```

A resubmission is itself an explicit action by the owner, so no separate Submit call is needed when the receipt requirement is already satisfied.

---

# 17. Resubmit DTO

```go
type ResubmitRequestDTO struct {
    Amount    *float64 `json:"amount,omitempty"`
    Purpose   *string  `json:"purpose,omitempty"`
    Urgency   *string  `json:"urgency,omitempty"`
    ReceiptMode *string `json:"receipt_mode,omitempty"` // "carry" (default) or "new"
}
```

Pointers allow the client to override only specific fields.

For example:

```json
{
  "amount": 3200,
  "urgency": "urgent"
}
```

Everything else comes from the previous request.

The resulting request must still pass all normal creation validation.

---

# 18. Receipts

Receipts are **child entities of a Request**, not a standalone domain.

A request can have zero or many receipts:

```text
Request 1 ──────── N Receipts
```

The receipt table may be separate for persistence and querying, but its business lifecycle is owned by the Requests domain.

## Upload

```text
POST /requests/:id/receipts
```

The service should:

1. Verify request exists.
2. Verify request belongs to the user's organization.
3. Verify the user is the request owner.
4. Verify the request status is `draft`. Once the request is submitted (`pending` or later) receipts are locked and the service returns `ErrReceiptsLocked`.
5. Validate file type.
6. Validate file size.
7. Generate a unique storage key.
8. Upload using shared `FileService`.
9. Create the receipt database record with `request_id`.
10. Trigger OCR processing.
11. Return the created receipt.

## List

```text
GET /requests/:id/receipts
```

This returns every receipt attached to the request.

A separate public `GET /receipts/:id` endpoint is not required because receipts are accessed through their parent request.

---

# 19. Receipt Validation

Allowed types:

```text
image/jpeg
image/png
application/pdf
```

Maximum size:

```text
10 MB
```

Do not trust only the filename extension.

Use the uploaded content type and, where appropriate, file-content detection.

---

# 20. Receipt Storage

The Requests domain should not directly manage S3/MinIO client details.

Use a shared abstraction:

```go
type FileService interface {
    Upload(
        ctx context.Context,
        key string,
        file io.Reader,
    ) (string, error)

    GetURL(key string) string
}
```

Storage key:

```text
receipts/{orgID}/{uuid}.{ext}
```

Example:

```text
receipts/org-123/550e8400-e29b-41d4-a716-446655440000.pdf
```

---

# 21. Receipt Model

Recommended response:

```go
type ReceiptResponse struct {
    ID              string     `json:"id"`
    RequestID       string     `json:"request_id"`
    FileURL         string     `json:"file_url"`
    FileName        string     `json:"file_name"`
    FileSize        int64      `json:"file_size"`
    FileType        string     `json:"file_type"`

    OCRStatus       string     `json:"ocr_status"`
    ExtractedAmount *float64   `json:"extracted_amount,omitempty"`
    Merchant        *string    `json:"merchant,omitempty"`
    ExtractedDate   *time.Time `json:"extracted_date,omitempty"`

    CreatedAt       time.Time  `json:"created_at"`
}
```

OCR statuses:

```text
pending
processing
success
failed
```

---

# 22. OCR Processing

For the hackathon implementation, OCR can be mocked.

The important part is keeping OCR behind a service boundary.

Example:

```go
type OCRService interface {
    Process(
        ctx context.Context,
        fileURL string,
    ) (*OCRResult, error)
}
```

The receipt service logic becomes:

```text
Upload receipt
      ↓
Store file
      ↓
Create receipt
      ↓
Set OCR status = processing
      ↓
Trigger OCR
      ↓
┌───────────────┐
│               │
▼               ▼
success        failure
│               │
▼               ▼
save data      failed
```

Do not put the OCR provider implementation directly into the request handler.

---

# 23. Comments

Comments are also part of the Requests domain.

## Add comment

```text
POST /requests/:id/comments
```

DTO:

```go
type CreateCommentDTO struct {
    Text string `json:"text"`
}
```

Validation:

```text
1 <= len(text) <= 1000
```

## Access

Staff:

```text
Can comment only on their own request.
```

Finance/Admin:

```text
Can comment on any request in their organization.
```

## Flow

```text
AddComment
    │
    ├── Validate text
    ├── Find request
    ├── Verify organization
    ├── Verify role/access
    ├── Insert comment
    └── Return comment
```

---

# 24. Comment Response

```go
type CommentResponse struct {
    ID        string    `json:"id"`
    Text      string    `json:"text"`
    Author    AuthorResponse `json:"author"`
    CreatedAt time.Time `json:"created_at"`
}
```

```go
type AuthorResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Role string `json:"role"`
}
```

List comments chronologically:

```sql
ORDER BY c.created_at ASC
```

---

# 25. Repository Interface

Keep the repository focused on persistence.

Example:

```go
type RequestRepository interface {
    // Requests
    Create(ctx context.Context, request *Request) error
    FindByID(ctx context.Context, orgID, requestID string) (*Request, error)
    List(ctx context.Context, filter ListFilter) ([]RequestListItem, error)
    Count(ctx context.Context, filter ListFilter) (int, error)

    UpdateStatus(
        ctx context.Context,
        requestID string,
        status string,
    ) error

    SetResubmittedAs(
        ctx context.Context,
        requestID string,
        newRequestID string,
    ) error

    // Marks a draft as pending and sets submitted_at.
    MarkSubmitted(ctx context.Context, orgID, requestID string) error

    // Locks the request row for the duration of a transaction.
    FindByIDForUpdate(ctx context.Context, orgID, requestID string) (*Request, error)

    // Receipts
    CreateReceipt(ctx context.Context, receipt *Receipt) error
    CountReceipts(ctx context.Context, orgID, requestID string) (int, error)
    CopyReceipts(ctx context.Context, orgID, fromRequestID, toRequestID string) error
    ListReceipts(
        ctx context.Context,
        orgID, requestID string,
    ) ([]Receipt, error)

    // Comments
    CreateComment(ctx context.Context, comment *Comment) error
    ListComments(
        ctx context.Context,
        orgID, requestID string,
    ) ([]Comment, error)
}
```

The exact interface should be adjusted to your existing repository conventions.

---

# 26. Transactions

Transactions are required when multiple database changes must succeed together.

## Resubmission

```text
BEGIN
   │
   ├── INSERT new request
   ├── COPY receipt records (receipt_mode = carry)
   ├── UPDATE old request
   └── COMMIT
```

If either operation fails:

```text
ROLLBACK
```

## Submit and receipt upload

Submit and UploadReceipt both lock the request row (`SELECT ... FOR UPDATE`) inside a transaction. This prevents a receipt from being attached while the status is flipping from `draft` to `pending`, and prevents two concurrent submits.

## Withdrawal

If withdrawal only updates the request, a transaction may not be necessary.

If the operation also requires another Requests-domain database write, use a transaction.

---

# 27. Models

The model layer should represent the domain data rather than HTTP input.

Example:

```go
type Request struct {
    ID           string
    OrgID        string
    RequesterID  string

    Type         string
    Amount       float64
    Purpose      string
    Urgency      string
    Status       string

    ResubmittedAs *string

    SubmittedAt  *time.Time

    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

Receipt:

```go
type Receipt struct {
    ID              string
    RequestID       string
    OrgID           string

    FileURL         string
    FileName        string
    FileSize        int64
    FileType        string

    OCRStatus       string
    ExtractedAmount *float64
    Merchant        *string
    ExtractedDate   *time.Time

    CreatedAt       time.Time
}
```

Comment:

```go
type Comment struct {
    ID        string
    RequestID string
    AuthorID  string
    Text      string
    CreatedAt time.Time
}
```

---

# 28. Validation

Keep request validation centralized.

```go
type RequestValidator struct{}
```

Responsibilities:

```go
ValidateCreate(dto CreateRequestDTO) error
ValidateResubmit(dto ResubmitRequestDTO) error
ValidateComment(dto CreateCommentDTO) error
ValidateListFilters(filters ListFilters) error
```

Validation rules:

| Field | Rule |
|---|---|
| Amount | `> 0` and `<= 999999.99` |
| Purpose | `10-500` characters |
| Type | reimbursement, advance, stipend |
| Urgency | routine, urgent, critical |
| Comment | `1-1000` characters |
| Receipt | at least one required to submit a reimbursement (checked by the service, not the validator) |
| Receipt mode | `carry` or `new` (resubmit only) |
| File | JPEG, PNG, PDF |
| File size | maximum 10 MB |

Business authorization rules remain in the service, not the validator.

---

# 29. Error Cases

Use domain-specific errors where useful.

Examples:

```go
var (
    ErrRequestNotFound       = errors.New("request not found")
    ErrUnauthorizedRequest   = errors.New("unauthorized request access")
    ErrInvalidRequestStatus  = errors.New("invalid request status")
    ErrCannotWithdraw        = errors.New("request cannot be withdrawn")
    ErrCannotResubmit        = errors.New("request cannot be resubmitted")
    ErrReceiptRequired       = errors.New("receipt is required")
    ErrReceiptNotFound       = errors.New("receipt not found")
    ErrReceiptsLocked        = errors.New("receipts can only be changed while the request is a draft")
    ErrCannotSubmit          = errors.New("only draft requests can be submitted")
    ErrCommentNotAllowed     = errors.New("comment not allowed")
)
```

The handler maps these to appropriate HTTP responses.

For example:

```text
not found       → 404
unauthorized    → 403
validation      → 400
conflict/state  → 409
internal error  → 500
```

Use the project's existing error-response convention if one already exists.

---

# 30. Timeline

The request detail may expose a timeline.

Example:

```go
type TimelineEvent struct {
    EventType string    `json:"event_type"`
    Label     string    `json:"label"`
    Timestamp time.Time `json:"timestamp"`
}
```

Example response:

```json
[
  {
    "event_type": "submitted",
    "label": "Request submitted",
    "timestamp": "2026-10-01T08:00:00Z"
  },
  {
    "event_type": "approved",
    "label": "Request approved",
    "timestamp": "2026-10-02T10:00:00Z"
  },
  {
    "event_type": "paid",
    "label": "Request paid",
    "timestamp": "2026-10-03T09:00:00Z"
  }
]
```

The Requests domain should build this from information available through its repository and other domain interfaces. It should **not implement approval or audit business logic**.

The `submitted` event uses `submitted_at`, not `created_at`. Drafts have no `submitted` event.

---

# 31. Dependency Direction

Keep dependencies flowing inward:

```text
HTTP
 │
 ▼
Handler
 │
 ▼
RequestService
 │
 ├──────────────► RequestRepository
 │
 ├──────────────► FileService
 │
 ├──────────────► OCRService
 │
 └──────────────► Other domain interfaces
```

The repository should not know about HTTP.

The handler should not know about SQL.

The file service should not know about Requests business rules.

---

# 32. Suggested Implementation Order

Implement in three phases. Each phase is standalone and depends only on the phases before it.

## Phase 1 — Foundations

Goal: everything the endpoints sit on. Routes are registered but return `501`.

- [ ] Create `internal/requests/`
- [ ] Migrations: `requests` (with `submitted_at`, `resubmitted_as`), `receipts` (non-null `request_id` FK + index), `comments`, `request_id_seq`
- [ ] Models: Request, Receipt, Comment
- [ ] DTOs and response types
- [ ] Domain errors and error-to-HTTP mapping
- [ ] Validator (create, resubmit incl. `receipt_mode`, comment, list filters, file)
- [ ] Repository interface and Postgres implementation, including transaction helper and row locking
- [ ] `formatRequestID`
- [ ] `FileService` and `OCRService` interfaces, mock OCR
- [ ] Service skeleton with access-control helpers (org check, owner vs finance/admin, draft visibility)
- [ ] Handler skeleton with auth-context helpers
- [ ] Register routes (`501`)
- [ ] Tests: validator, ID generation, repository integration, access helpers

## Phase 2 — Endpoints

Goal: all routes working. Build in four mergeable slices.

### 2A — Core requests
- [ ] `POST /requests` (draft for reimbursement, pending otherwise)
- [ ] `GET /requests` (role scoping, drafts hidden from finance/admin, filters, pagination)
- [ ] `GET /requests/:id`
- [ ] `GET /requests/:id/previous`

### 2B — Lifecycle
- [ ] `POST /requests/:id/submit`
- [ ] `PUT /requests/:id/withdraw` (draft or pending)
- [ ] `POST /requests/:id/resubmit` (transactional, `carry` / `new`)

### 2C — Receipts
- [ ] `POST /requests/:id/receipts` (owner only, draft only, multiple allowed)
- [ ] `GET /requests/:id/receipts`
- [ ] File validation, `FileService` integration, OCR trigger with mock OCR
- [ ] Detail endpoint returns `receipts`

### 2D — Comments
- [ ] `POST /requests/:id/comments`
- [ ] `GET /requests/:id/comments`

## Phase 3 — Finishing touches

Goal: integrate with other domains and harden.

- [ ] Approval integration (`ApprovalResponse` in detail)
- [ ] Audit integration (create, submit, withdraw, resubmit)
- [ ] Full request timeline
- [ ] Transaction failure tests (resubmit rollback)
- [ ] Concurrency tests (double submit, upload during submit, double resubmit)
- [ ] Role-based access matrix (staff / finance / admin / wrong org, including drafts)
- [ ] OCR failure handling
- [ ] Pagination defaults and maximum limit
- [ ] API docs and cleanup

---

# 33. Testing Strategy

## Request creation

Test:

- valid request
- amount <= 0
- amount > 999999.99
- purpose too short
- purpose too long
- invalid type
- invalid urgency
- reimbursement is created as draft
- advance is created as pending
- stipend is created as pending

## Request listing

Test:

- staff sees only own requests
- finance sees organization requests
- admin sees organization requests
- staff sees their own drafts
- finance/admin do not see drafts
- status filter
- urgency filter
- requester filter
- pagination
- total count
- days pending

## Request detail

Test:

- existing request
- nonexistent request
- wrong organization
- staff accessing another user's request
- finance accessing organization request
- finance accessing a draft returns 404

## Submit

Test:

- owner submits a draft reimbursement with one receipt
- owner submits with multiple receipts
- reimbursement without receipt cannot be submitted
- non-owner cannot submit
- pending request cannot be submitted again
- nonexistent request
- concurrent double submit
- receipt upload racing with submit

## Withdrawal

Test:

- owner can withdraw
- non-owner cannot withdraw
- pending can withdraw
- draft can withdraw
- rejected cannot withdraw
- approved cannot withdraw
- nonexistent request

## Resubmission

Test:

- rejected can resubmit
- failed can resubmit
- pending cannot resubmit
- draft cannot resubmit
- carry mode copies receipts to the new request
- new mode starts with no receipts
- reimbursement + carry resubmits as pending
- reimbursement + new resubmits as draft
- advance/stipend resubmit as pending
- old request keeps its receipts
- owner can resubmit
- non-owner cannot resubmit
- new ID generated
- old request linked to new request
- transaction rollback

## Receipts

Test:

- valid JPEG
- valid PNG
- valid PDF
- invalid file type
- >10 MB
- unauthorized upload
- upload allowed on draft
- upload rejected after submit (locked)
- non-owner cannot upload
- multiple receipts attached to one request
- list all receipts for a request
- receipt from another request cannot be accessed
- OCR success
- OCR failure

## Comments

Test:

- valid comment
- empty comment
- >1000 characters
- staff commenting on own request
- staff commenting on another request
- finance commenting on organization request
- wrong organization

---

# 34. Final Architecture

The final domain should look like:

```text
                         REQUESTS DOMAIN
                              │
             ┌────────────────┼────────────────┐
             │                │                │
             ▼                ▼                ▼
          REQUEST          RECEIPTS         COMMENTS
             │                │                │
             │                │                │
             │          child of Request      │
             │                │                │
             │           1 ───┴─── N           │
             │                                 │
             └──────────────┬──────────────────┘
                            │
                      RequestService
                            │
        ┌───────────────────┼────────────────────┐
        │                   │                    │
        ▼                   ▼                    ▼
 RequestRepository     FileService          OCRService
        │                   │                    │
        ▼                   ▼                    ▼
    PostgreSQL          S3/MinIO            OCR Provider
```

### Important domain rule

**Receipt is not a standalone domain.**

It is a child entity owned by the Requests domain:

```text
Request
  ├── Receipt 1
  ├── Receipt 2
  ├── Receipt 3
  ├── Comment 1
  └── Comment 2
```

The database can still have a separate `receipts` table because a request can have multiple receipts.

The public API reflects this relationship:

```text
POST /requests/:id/receipts
GET  /requests/:id/receipts
GET  /requests/:id
```

There is no standalone:

```text
POST /receipts/upload
GET  /receipts/:id
```

unless a future requirement explicitly needs it.

### Request creation and receipts

Request creation does not require a `receipt_id`:

```go
type CreateRequestDTO struct {
    Type    string  `json:"type"`
    Amount  float64 `json:"amount"`
    Purpose string  `json:"purpose"`
    Urgency string  `json:"urgency"`
}
```

Receipts are attached after the request exists:

```text
POST /requests
       ↓
REQ-000042
       ↓
POST /requests/REQ-000042/receipts
       ↓
Receipt 1
       ↓
POST /requests/REQ-000042/receipts
       ↓
Receipt 2
```

For a reimbursement, the lifecycle is:

```text
POST /requests                    → REQ-000042   status = draft
POST /requests/REQ-000042/receipts   (repeat as needed, draft only)
POST /requests/REQ-000042/submit  → requires >= 1 receipt → status = pending
```

After submit, receipts are locked.

---

# 35. Implementation Checklist

Before considering the domain complete:

### Request

- [ ] Create request
- [ ] Submit draft request
- [ ] Draft visibility rules
- [ ] List requests
- [ ] Get request detail
- [ ] Get previous requests
- [ ] Withdraw request
- [ ] Resubmit request
- [ ] Request ID sequence
- [ ] Request validation
- [ ] Role-based access control
- [ ] Pagination and filtering

### Receipts

- [ ] Attach receipt to request
- [ ] Support multiple receipts per request
- [ ] Lock receipts after submit
- [ ] Carry or replace receipts on resubmit
- [ ] List request receipts
- [ ] File validation
- [ ] 10 MB size limit
- [ ] S3/MinIO integration through `FileService`
- [ ] OCR status
- [ ] OCR result storage
- [ ] OCR failure handling

### Comments

- [ ] Add comment
- [ ] List comments
- [ ] Comment validation
- [ ] Role-based access control

### Integration

- [ ] Approval integration
- [ ] Audit integration
- [ ] Request timeline
- [ ] Unit tests
- [ ] Repository integration tests
- [ ] Transaction tests
- [ ] Authorization tests

---

# 36. Core Design Decision

The final decision for this domain is:

> **Requests, Receipts, and Comments are one business domain.**
>
> **Request is the aggregate/root resource.**
>
> **Receipts and Comments are child entities of Request.**
>
> **A Request may have many Receipts and many Comments.**
>
> **Receipts get their own database table because they are one-to-many data, but they do not get their own domain/service boundary.**

This keeps the implementation cohesive without pretending that every database table needs to become a separate domain.
