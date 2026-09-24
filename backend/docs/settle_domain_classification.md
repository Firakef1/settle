# ⚡ SETTLE — Backend Domain Classification & Implementation Guide

> _Two developers. Nine days. Zero conflicts._

---

## 🎯 The Split — At a Glance

| | 🟢 **Menweyelet** | 🔵 **Hosama** |
|---|---|---|
| **Title** | The Foundation Builder | The Decision Engine |
| **Domains** | Auth · Requests · Receipts · Comments | Approvals · Admin · Dashboard · Audit |
| **Packages** | `internal/auth/` `internal/requests/` `internal/comments/` | `internal/approvals/` `internal/admin/` |
| **Tables** | `users` · `requests` · `receipts` · `comments` | `organizations` · `org_members` · `approvals` · `invitations` · `audit_log` |
| **Endpoints** | 12 | 17 |
| **Complexity** | JWT infra + File upload + OCR | State machine + Multi-entity CRUD |
| **Effort** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

> [!IMPORTANT]
> Hosama has more endpoints but most are simple CRUD. Menweyelet has fewer but heavier ones (JWT, file uploads, OCR). **Balanced.**

---

## 🟢 MENWEYELET — Domain Breakdown

### Domain 1: Auth (`internal/auth/`)

**What it does:** User registration, login, JWT token management.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `POST` | `/auth/login` | Validate credentials → return JWT |
| `POST` | `/auth/signup` | Create user account (no org yet) |

**Files to create:**

```
internal/auth/
├── handler/auth_handler.go       → Parse HTTP, call service, return JSON
├── service/auth_service.go       → Login logic, signup logic
├── repository/user_repo.go       → DB queries: create user, find by email, get memberships
├── dto/login_request.go          → { email, password }
├── dto/signup_request.go         → { email, password, name }
├── dto/login_response.go         → { token, user, orgs }
└── validator/auth_validator.go   → Email format, password min length
```

**How to implement:**

```go
// handler/auth_handler.go
func (h *AuthHandler) Login(c *gin.Context) {
    // 1. Parse body → LoginRequest DTO
    // 2. Validate (email not empty, password not empty)
    // 3. Call h.service.Login(email, password)
    // 4. Return 200 + { token, user, orgs } or 401
}

// service/auth_service.go
func (s *AuthService) Login(email, password string) (*LoginResponse, error) {
    // 1. user := s.repo.FindByEmail(email) → if nil, UnauthorizedError
    // 2. s.hashService.Compare(password, user.PasswordHash) → if false, UnauthorizedError
    // 3. memberships := s.repo.GetOrgMemberships(user.ID)
    // 4. token := s.jwtService.Generate(user.ID, email, firstOrg.ID, membership.Role)
    // 5. Return { token, user, memberships }
}

// repository/user_repo.go
func (r *UserRepo) FindByEmail(email string) (*models.User, error) {
    // SELECT id, email, name, password_hash, department, status
    // FROM users WHERE email = $1 AND status = 'active'
}

func (r *UserRepo) GetOrgMemberships(userID string) ([]MembershipInfo, error) {
    // SELECT om.org_id, om.role, o.name, o.slug
    // FROM org_members om
    // JOIN organizations o ON o.id = om.org_id
    // WHERE om.user_id = $1 AND om.status = 'active'
}
```

**Shared services Menweyelet builds (used by both devs):**

```go
// shared/service/jwt_service.go
type JWTService struct { secret string }
func (s *JWTService) Generate(userID, email, orgID, role string) (string, error)
func (s *JWTService) Verify(token string) (*Claims, error)

// shared/service/hash_service.go
type HashService struct {}
func (s *HashService) Hash(password string) (string, error)      // bcrypt
func (s *HashService) Compare(password, hash string) bool        // bcrypt compare
```

---

### Domain 2: Requests (`internal/requests/`)

**What it does:** Core request CRUD — staff submits, views, withdraws, resubmits payout requests.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `POST` | `/requests` | Staff submits new payout request |
| `GET` | `/requests` | List requests (staff=own, finance=all in org) |
| `GET` | `/requests/:id` | Full detail (with receipt, approval, comments) |
| `GET` | `/requests/:id/previous` | Requester's history (for finance context) |
| `PUT` | `/requests/:id/withdraw` | Staff withdraws a pending request |
| `POST` | `/requests/:id/resubmit` | Staff resubmits rejected/failed as new request |

**Files to create:**

```
internal/requests/
├── handler/request_handler.go        → 6 route handlers
├── service/request_service.go        → Business logic + ID generation
├── repository/request_repo.go        → All DB queries for requests table
├── dto/create_request.go             → { type, amount, purpose, urgency, ... }
├── dto/request_response.go           → Full response shape with nested objects
├── dto/request_list_response.go      → Paginated list shape
└── validator/request_validator.go    → Amount > 0, purpose 10-500 chars, etc.
```

**How to implement:**

```go
// service/request_service.go

func (s *RequestService) Create(orgID, userID string, dto CreateRequestDTO) (*Request, error) {
    // 1. Validate: amount > 0, amount < 999999.99
    // 2. Validate: purpose length 10-500
    // 3. Validate: type ∈ {reimbursement, advance, stipend}
    // 4. Validate: if type == "reimbursement" → receipt required
    // 5. Generate ID: "REQ-" + zero-padded sequence number
    // 6. INSERT INTO requests (...) VALUES (...)
    // 7. INSERT INTO audit_log (action='created', ...)
    // 8. Return request
}

func (s *RequestService) List(orgID, userID, role string, filters ListFilters) (*PaginatedResult, error) {
    // If role == "staff" → filter by requester_id = userID
    // If role == "finance" or "org_admin" → show all in org
    // Apply filters: status, urgency, sort, limit, offset
    // Calculate days_pending for each: NOW() - created_at
    // Return { data: [...], meta: { total, limit, offset, has_more } }
}

func (s *RequestService) GetDetail(orgID, requestID, userID, role string) (*RequestDetail, error) {
    // 1. Get request base data
    // 2. Get receipt data (LEFT JOIN receipts)
    // 3. Get approval data (LEFT JOIN approvals)
    // 4. Get comments (SELECT FROM comments WHERE request_id = ?)
    // 5. Build timeline: "Submitted Jan 5 → Approved Jan 7 → Paid Jan 8"
    // 6. Access check: staff can only see own, finance sees all in org
}

func (s *RequestService) Withdraw(orgID, requestID, userID, reason string) error {
    // 1. Get request → validate exists in org
    // 2. Validate: request.RequesterID == userID (only owner can withdraw)
    // 3. Validate: request.Status == "pending" (only pending can be withdrawn)
    // 4. BEGIN TRANSACTION
    //    UPDATE requests SET status='withdrawn' WHERE id=?
    //    INSERT INTO audit_log (action='withdrawn', ...)
    //    COMMIT
}

func (s *RequestService) Resubmit(orgID, requestID, userID string, dto ResubmitDTO) (*Request, error) {
    // 1. Get old request → validate exists
    // 2. Validate: status ∈ {"rejected", "failed"}
    // 3. Validate: requester_id == userID
    // 4. BEGIN TRANSACTION
    //    Create NEW request (new ID, status=pending, copy fields + overrides)
    //    UPDATE old request: SET resubmitted_as = newID
    //    INSERT INTO audit_log
    //    COMMIT
    // 5. Return new request
}
```

**Request ID generation:**

```go
func (s *RequestService) generateID(ctx context.Context) string {
    // Option A: PostgreSQL sequence
    //   SELECT nextval('request_id_seq') → pad to 6 digits → "REQ-000042"
    //
    // Option B: Count-based (simpler for hackathon)
    //   SELECT COUNT(*) + 1 FROM requests → "REQ-000042"
    //
    // Format: "REQ-" + 6-digit zero-padded number
}
```

**Key query — request list with days_pending:**

```sql
SELECT r.id, r.amount, r.purpose, r.urgency, r.status, r.created_at,
       u.name as requester_name, u.email as requester_email,
       EXTRACT(DAY FROM NOW() - r.created_at)::int as days_pending
FROM requests r
JOIN users u ON r.requester_id = u.id
WHERE r.org_id = $1
  AND ($2 = '' OR r.status = $2)          -- status filter
  AND ($3 = '' OR r.urgency = $3)          -- urgency filter
  AND ($4 = '' OR r.requester_id = $4)     -- requester filter (finance only)
ORDER BY
  CASE r.urgency
    WHEN 'critical' THEN 1
    WHEN 'urgent' THEN 2
    WHEN 'routine' THEN 3
  END ASC,
  r.created_at ASC                         -- oldest first (escalation)
LIMIT $5 OFFSET $6;
```

---

### Domain 3: Receipts (inside `internal/requests/` or separate)

**What it does:** File upload to S3/MinIO, trigger async OCR, return extracted data.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `POST` | `/receipts/upload` | Upload file → store → trigger OCR |
| `GET` | `/receipts/:id` | Get receipt with OCR status/results |

**How to implement:**

```go
// service/receipt_service.go

func (s *ReceiptService) Upload(file multipart.File, header *multipart.FileHeader, orgID string) (*Receipt, error) {
    // 1. Validate file type: header.Header.Get("Content-Type") ∈ {image/jpeg, image/png, application/pdf}
    // 2. Validate file size: header.Size <= 10 * 1024 * 1024 (10MB)
    // 3. Generate S3 key: "receipts/{orgID}/{uuid}.{ext}"
    // 4. s.fileService.Upload(key, file) → returns fileURL
    // 5. INSERT INTO receipts (id, file_url, file_name, file_size, file_type, ocr_status='pending')
    // 6. Launch async OCR:
    //    go s.processOCR(receipt.ID, fileURL)
    // 7. Return receipt (ocr_status: "processing")
}

func (s *ReceiptService) processOCR(receiptID, fileURL string) {
    // RUNS IN BACKGROUND GOROUTINE
    // For hackathon: mock OCR that extracts hardcoded data
    // For production: call AWS Textract or Google Vision
    //
    // On success: UPDATE receipts SET ocr_status='success',
    //             extracted_amount=?, extracted_merchant=?, extracted_date=?
    // On failure: UPDATE receipts SET ocr_status='failed'
}
```

**Shared file service Menweyelet builds:**

```go
// shared/service/file_service.go
type FileService struct { s3Client *s3.Client; bucket string }
func (s *FileService) Upload(key string, file io.Reader) (string, error)
func (s *FileService) GetURL(key string) string
```

---

### Domain 4: Comments (`internal/comments/`)

**What it does:** Discussion threads on requests between staff and finance.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `POST` | `/requests/:id/comments` | Add comment (staff or finance) |
| `GET` | `/requests/:id/comments` | List all comments on a request |

**How to implement:**

```go
// service/comment_service.go

func (s *CommentService) Add(orgID, requestID, authorID, role, text string) (*Comment, error) {
    // 1. Validate: text length 1-1000
    // 2. Verify request exists in this org
    // 3. Access check:
    //    - Staff: can only comment on OWN requests
    //    - Finance/Admin: can comment on ANY request in org
    // 4. INSERT INTO comments (id, request_id, author_id, text, created_at)
    // 5. INSERT INTO audit_log (action='commented', ...)
    // 6. Return comment with author name + role
}

func (s *CommentService) List(orgID, requestID string) ([]Comment, error) {
    // SELECT c.*, u.name as author_name, om.role as author_role
    // FROM comments c
    // JOIN users u ON c.author_id = u.id
    // JOIN org_members om ON om.user_id = c.author_id AND om.org_id = $1
    // WHERE c.request_id = $2
    // ORDER BY c.created_at ASC
}
```

---

## 🔵 HOSAMA — Domain Breakdown

### Domain 1: Approvals (`internal/approvals/`)

**What it does:** Finance decides on requests — approve, reject, mark paid, record failures.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `PUT` | `/requests/:id/approve` | Finance approves pending request |
| `PUT` | `/requests/:id/reject` | Finance rejects pending request (reason required) |
| `PUT` | `/requests/:id/mark-paid` | Finance records successful payment |
| `PUT` | `/requests/:id/payment-failed` | Finance records payment failure |

**Files to create:**

```
internal/approvals/
├── handler/approval_handler.go       → 4 route handlers
├── service/approval_service.go       → State validation + transactions (injects RequestReader)
├── repository/approval_repo.go       → Approval CRUD only (NOT request queries)
├── dto/approve_request.go            → { note }
├── dto/reject_request.go             → { reason }  ← reason REQUIRED
├── dto/payment_request.go            → { payment_method } or { failure_reason }
├── dto/approval_response.go          → Full approval response shape
└── validator/approval_validator.go   → State transition validation
```

**How to implement:**

```go
// service/approval_service.go
// Uses injected interfaces — no import from Menweyelet's packages

type ApprovalService struct {
    approvalRepo  *repository.ApprovalRepo
    requestReader interfaces.RequestReader   // ← injected via constructor
    auditWriter   interfaces.AuditWriter     // ← injected via constructor
}

func (s *ApprovalService) Approve(ctx context.Context, orgID, requestID, approverID, note string) (*ApprovalResponse, error) {
    // 1. req := s.requestReader.GetByID(orgID, requestID) → 404 if not found
    // 2. Validate: req.Status == "pending" → 409 if not
    // 3. BEGIN TRANSACTION
    //    a. INSERT INTO approvals (request_id, approver_id, decision='approved',
    //       decision_note=note, payment_status='pending_payment')
    //    b. s.requestReader.UpdateStatus(requestID, "approved")
    //    c. s.auditWriter.Log(action='approved', actor=approverID,
    //       old_value='{"status":"pending"}', new_value='{"status":"approved"}')
    //    COMMIT
    // 4. Return approval response
}

func (s *ApprovalService) Reject(ctx context.Context, orgID, requestID, approverID, reason string) (*ApprovalResponse, error) {
    // 1. Validate: reason is NOT empty (required for rejection)
    // 2. req := s.requestReader.GetByID(orgID, requestID) → 404 if not found
    // 3. Validate: req.Status == "pending" → 409 if not
    // 4. BEGIN TRANSACTION
    //    a. INSERT INTO approvals (decision='rejected', decision_note=reason)
    //    b. s.requestReader.UpdateStatus(requestID, "rejected")
    //    c. s.auditWriter.Log(action='rejected')
    //    COMMIT
}

func (s *ApprovalService) MarkPaid(ctx context.Context, orgID, requestID, approverID, method string) (*ApprovalResponse, error) {
    // 1. Validate: method ∈ {bank_transfer, check, cash, other}
    // 2. req := s.requestReader.GetByID(orgID, requestID) → must be "approved"
    // 3. BEGIN TRANSACTION
    //    a. UPDATE approvals SET payment_status='paid', payment_method=method, payment_at=NOW()
    //    b. s.requestReader.UpdateStatus(requestID, "paid")
    //    c. s.auditWriter.Log(action='paid')
    //    COMMIT
}

func (s *ApprovalService) MarkFailed(ctx context.Context, orgID, requestID, approverID, reason string) (*ApprovalResponse, error) {
    // 1. Validate: reason not empty
    // 2. req must be "approved" (you can't fail a payment that wasn't approved)
    // 3. BEGIN TRANSACTION
    //    a. UPDATE approvals SET payment_status='failed', failure_reason=reason, failure_at=NOW()
    //    b. s.requestReader.UpdateStatus(requestID, "failed")
    //    c. s.auditWriter.Log(action='failed')
    //    COMMIT
}
```

**State validation cheat sheet for Hosama:**

```
approve():      only if status == "pending"     → 409 otherwise
reject():       only if status == "pending"     → 409 otherwise
mark_paid():    only if status == "approved"    → 409 otherwise
payment_failed(): only if status == "approved"  → 409 otherwise
```

---

### Domain 2: Admin — Organizations (`internal/admin/`)

**What it does:** Organization CRUD, settings management.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `POST` | `/organizations` | Create org (creator becomes org_admin) |
| `GET` | `/organizations/:id` | Get org info |
| `PUT` | `/organizations/:id` | Update name, currency |

**How to implement:**

```go
// service/org_service.go

func (s *OrgService) Create(userID string, dto CreateOrgDTO) (*Organization, error) {
    // 1. Generate slug from name: "Acme Research Lab" → "acme-research-lab"
    // 2. Ensure slug is unique (append number if needed)
    // 3. BEGIN TRANSACTION
    //    a. INSERT INTO organizations (id, name, slug, plan='free', currency=dto.Currency)
    //    b. INSERT INTO org_members (org_id, user_id=userID, role='org_admin',
    //       status='active', joined_at=NOW())
    //    c. INSERT INTO audit_log (action='org_created')
    //    COMMIT
    // 4. Return org
}

func (s *OrgService) Update(orgID, userID string, dto UpdateOrgDTO) (*Organization, error) {
    // 1. Only org_admin can update → check role from context
    // 2. UPDATE organizations SET name=?, currency=? WHERE id=?
    // 3. INSERT INTO audit_log (action='settings_updated',
    //    old_value=oldSettings, new_value=newSettings)
    // 4. Return updated org
}
```

**Slug generation:**

```go
func generateSlug(name string) string {
    // 1. Lowercase
    // 2. Replace spaces with hyphens
    // 3. Remove special characters
    // 4. Trim trailing hyphens
    // "Acme Research Lab" → "acme-research-lab"
}
```

---

### Domain 3: Admin — Members & Invitations

**What it does:** Invite users, accept invites, manage roles, remove members.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `GET` | `/organizations/:id/members` | List all org members |
| `POST` | `/organizations/:id/invitations` | Create invite (generate token link) |
| `POST` | `/invitations/:token/accept` | Accept invite (create account + join org) |
| `DELETE` | `/organizations/:id/members/:uid` | Remove member from org |
| `PUT` | `/organizations/:id/members/:uid/role` | Change role (staff ↔ finance) |

**How to implement:**

```go
// service/invitation_service.go

func (s *InvitationService) Create(orgID, adminID, email, role string) (*Invitation, error) {
    // 1. Only org_admin can invite → verify role
    // 2. Check: email not already a member of this org
    // 3. Generate token: uuid.New().String()
    // 4. Set expiry: time.Now().Add(10 * 24 * time.Hour)
    // 5. INSERT INTO invitations (org_id, email, role, token, created_by, expires_at)
    // 6. Build invite link: fmt.Sprintf("https://settle.app/invite/%s", token)
    // 7. INSERT INTO audit_log (action='user_invited')
    // 8. Return { invite_link, expires_at }
}

func (s *InvitationService) Accept(token, name, password string) (*AcceptResponse, error) {
    // 1. Find invitation by token
    // 2. Validate: not expired (expires_at > NOW())
    // 3. Validate: not already accepted (accepted_at IS NULL)
    // 4. BEGIN TRANSACTION
    //    a. Hash password → bcrypt (use shared hashService)
    //    b. INSERT INTO users (email=invitation.Email, name, password_hash)
    //       OR if user exists (signed up before): just use existing user
    //    c. INSERT INTO org_members (org_id, user_id, role=invitation.Role,
    //       status='active', joined_at=NOW())
    //    d. UPDATE invitations SET accepted_at=NOW() WHERE id=?
    //    e. INSERT INTO audit_log (action='user_added')
    //    COMMIT
    // 5. Generate JWT for new user
    // 6. Return { user_id, org_id, token: JWT }
}

// service/member_service.go

func (s *MemberService) Remove(orgID, adminID, targetUserID string) error {
    // 1. Only org_admin can remove
    // 2. Can't remove yourself (prevent last admin removal)
    // 3. UPDATE org_members SET status='removed' WHERE org_id=? AND user_id=?
    // 4. INSERT INTO audit_log (action='user_removed')
}

func (s *MemberService) ChangeRole(orgID, adminID, targetUserID, newRole string) error {
    // 1. Only org_admin can change roles
    // 2. Validate: newRole ∈ {staff, finance, org_admin}
    // 3. UPDATE org_members SET role=? WHERE org_id=? AND user_id=?
    // 4. INSERT INTO audit_log (action='role_changed', old_value, new_value)
}
```

---

### Domain 4: Admin — Dashboard & Stats

**What it does:** Finance dashboard with counts, escalation, org-wide statistics.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `GET` | `/dashboard/summary` | Pending counts, urgency breakdown, escalated list |
| `GET` | `/organizations/:id/stats` | Member count, request stats, financials, plan usage |

**How to implement:**

```go
// service/dashboard_service.go

func (s *DashboardService) GetSummary(orgID string) (*DashboardSummary, error) {
    // Single optimized query OR parallel queries:
    //
    // Query 1: Counts
    //   SELECT
    //     COUNT(*) FILTER (WHERE status='pending') as pending_count,
    //     COUNT(*) FILTER (WHERE status='pending' AND urgency='urgent') as urgent_count,
    //     COUNT(*) FILTER (WHERE status='pending' AND urgency='critical') as critical_count
    //   FROM requests WHERE org_id = $1
    //
    // Query 2: Aging breakdown
    //   SELECT
    //     COUNT(*) FILTER (WHERE age <= 3) as "0_to_3_days",
    //     COUNT(*) FILTER (WHERE age > 3 AND age <= 7) as "3_to_7_days",
    //     COUNT(*) FILTER (WHERE age > 7) as "7_plus_days"
    //   FROM (
    //     SELECT EXTRACT(DAY FROM NOW() - created_at) as age
    //     FROM requests WHERE org_id=$1 AND status='pending'
    //   ) sub
    //
    // Query 3: Escalated requests (7+ days)
    //   SELECT r.id, u.name, r.amount, r.urgency,
    //          EXTRACT(DAY FROM NOW()-r.created_at)::int as days_pending
    //   FROM requests r JOIN users u ON r.requester_id = u.id
    //   WHERE r.org_id=$1 AND r.status='pending'
    //     AND NOW()-r.created_at > INTERVAL '7 days'
    //   ORDER BY r.created_at ASC
}

func (s *DashboardService) GetOrgStats(orgID string) (*OrgStats, error) {
    // Parallel queries:
    // 1. Member count: SELECT COUNT(*), role FROM org_members WHERE org_id=$1 GROUP BY role
    // 2. Request stats: SELECT status, COUNT(*) FROM requests WHERE org_id=$1 GROUP BY status
    // 3. Financials: SELECT SUM(amount) FROM requests WHERE org_id=$1 AND status='paid'
    // 4. This month: ... AND created_at >= date_trunc('month', NOW())
    // 5. Plan usage: count requests this month vs plan limit
}
```

---

### Domain 5: Admin — Audit Log

**What it does:** Immutable record of every action in the org.

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `GET` | `/organizations/:id/audit-log` | Query audit entries (filterable, paginated) |

**How to implement:**

```go
// service/audit_service.go

// This is also a SHARED UTILITY — both devs write to audit_log
func (s *AuditService) Log(ctx context.Context, tx *sql.Tx, entry AuditEntry) error {
    // INSERT INTO audit_log
    //   (id, org_id, request_id, action, actor_id, old_value, new_value, timestamp, ip_address)
    // VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
}

func (s *AuditService) Query(orgID string, filters AuditFilters) (*PaginatedAudit, error) {
    // SELECT al.*, u.name as actor_name
    // FROM audit_log al JOIN users u ON al.actor_id = u.id
    // WHERE al.org_id = $1
    //   AND ($2 = '' OR al.action = $2)
    //   AND ($3 IS NULL OR al.timestamp >= $3)
    //   AND ($4 IS NULL OR al.timestamp <= $4)
    // ORDER BY al.timestamp DESC
    // LIMIT $5 OFFSET $6
}
```

---

### Domain 6: Admin — Billing (Lightweight)

**What it does:** Plan display, upgrade UI support (mock for hackathon).

**Endpoints:**

| Method | Route | What It Does |
|---|---|---|
| `GET` | `/billing/plans` | Return static list of plans (Free/Starter/Pro) |
| `PUT` | `/organizations/:id/plan` | Change org plan (just DB update for now) |

**How to implement:**

```go
// service/billing_service.go

func (s *BillingService) GetPlans() []Plan {
    // Return static data — no DB query needed
    return []Plan{
        {ID: "free",    Name: "Free",    Price: 0,   RequestLimit: 100,  UserLimit: 5},
        {ID: "starter", Name: "Starter", Price: 49,  RequestLimit: 1000, UserLimit: 25},
        {ID: "pro",     Name: "Pro",     Price: 199, RequestLimit: 0,    UserLimit: 0}, // 0 = unlimited
    }
}

func (s *BillingService) ChangePlan(orgID, adminID, newPlan string) error {
    // 1. Validate: newPlan ∈ {free, starter, pro}
    // 2. UPDATE organizations SET plan = $1 WHERE id = $2
    // 3. INSERT INTO audit_log (action='plan_changed')
    // For hackathon: no Stripe. Just DB update.
}

// Plan enforcement (middleware or called by Menweyelet's request creation):
func (s *BillingService) CheckLimit(orgID string) error {
    // 1. Get org plan
    // 2. Count requests this month
    // 3. If count >= plan.RequestLimit → return error "Plan limit reached"
}
```

---

## 🤝 Shared Layer — The Contract

> [!CAUTION]
> **Lock this on Day 1. After that, changes require both developers to agree.**

### What Lives in Shared

```
internal/shared/
│
├── database/
│   └── db.go                    → PostgreSQL pool + WithTransaction() helper
│
├── interfaces/                  → Cross-feature contracts (THE KEY TO NON-BLOCKING)
│   ├── request_reader.go        → RequestReader interface (Menweyelet implements, Hosama injects)
│   └── audit_writer.go          → AuditWriter interface (Hosama implements, both inject)
│
├── models/                      → Go structs mirroring DB tables
│   ├── user.go                  → type User struct { ID, Email, Name, ... }
│   ├── organization.go          → type Organization struct { ID, Name, Slug, Plan, ... }
│   ├── request.go               → type Request struct { ID, OrgID, Status, Amount, ... }
│   ├── approval.go              → type Approval struct { ID, RequestID, Decision, ... }
│   ├── receipt.go               → type Receipt struct { ID, FileURL, OCRStatus, ... }
│   ├── comment.go               → type Comment struct { ID, RequestID, AuthorID, Text, ... }
│   ├── invitation.go            → type Invitation struct { ID, OrgID, Email, Token, ... }
│   ├── audit_log.go             → type AuditLog struct { ID, OrgID, Action, ActorID, ... }
│   └── org_member.go            → type OrgMember struct { ID, OrgID, UserID, Role, ... }
│
├── middleware/
│   ├── auth_middleware.go       → Verify JWT → inject userID, email, orgID, role into context
│   ├── org_middleware.go        → Verify user belongs to org → inject orgID
│   ├── role_middleware.go       → RequireRole("finance") → 403 if wrong role
│   ├── error_handler.go        → Catch AppError → return standardized JSON
│   └── request_logger.go       → Log every HTTP request (method, path, status, duration)
│
├── service/
│   ├── jwt_service.go           → Generate / Verify JWT tokens
│   ├── hash_service.go          → bcrypt Hash / Compare
│   ├── file_service.go          → S3/MinIO upload/download
│   └── audit_writer.go         → Shared utility to INSERT INTO audit_log
│
├── config/
│   └── config.go                → Load .env: DB_URL, JWT_SECRET, S3_BUCKET, PORT
│
├── enums/
│   └── enums.go                 → All constants:
│       // Roles
│       RoleStaff    = "staff"
│       RoleFinance  = "finance"
│       RoleOrgAdmin = "org_admin"
│       // Request Status
│       StatusPending   = "pending"
│       StatusApproved  = "approved"
│       StatusRejected  = "rejected"
│       StatusPaid      = "paid"
│       StatusFailed    = "failed"
│       StatusWithdrawn = "withdrawn"
│       // Request Type
│       TypeReimbursement = "reimbursement"
│       TypeAdvance       = "advance"
│       TypeStipend       = "stipend"
│       // Urgency
│       UrgencyRoutine  = "routine"
│       UrgencyUrgent   = "urgent"
│       UrgencyCritical = "critical"
│
└── utils/
    ├── errors.go                → AppError, ValidationError, NotFoundError, ConflictError, etc.
    ├── response.go              → Helper: SuccessResponse(c, data), ErrorResponse(c, err)
    └── helpers.go               → UUID generation, time formatting, slug generation
```

### The Golden Rule — Interface-Based Service Injection

```go
// ❌ WRONG — Hosama imports Menweyelet's concrete package directly
import "settle/internal/requests/service"
func (s *ApprovalService) Approve() {
    requestService.GetRequest(id)  // CIRCULAR DEPENDENCY RISK 💀
}
```

```go
// ✅ RIGHT — Define an interface in shared, Menweyelet implements it,
//           Hosama receives it via constructor injection

// ──────────────────────────────────────────────────
// Step 1: Define interfaces in shared (agreed Day 1)
// ──────────────────────────────────────────────────
// shared/interfaces/request_reader.go
package interfaces

import "settle/internal/shared/models"

type RequestReader interface {
    GetByID(ctx context.Context, orgID, requestID string) (*models.Request, error)
    UpdateStatus(ctx context.Context, requestID, status string) error
}

type AuditWriter interface {
    Log(ctx context.Context, entry models.AuditEntry) error
}
```

```go
// ──────────────────────────────────────────────────────────────
// Step 2: Menweyelet implements it (his repo already does this)
// ──────────────────────────────────────────────────────────────
// requests/repository/request_repo.go
package repository

type RequestRepo struct { db *sql.DB }

func (r *RequestRepo) GetByID(ctx context.Context, orgID, requestID string) (*models.Request, error) {
    // SELECT * FROM requests WHERE id = $1 AND org_id = $2
}

func (r *RequestRepo) UpdateStatus(ctx context.Context, requestID, status string) error {
    // UPDATE requests SET status = $1, updated_at = NOW() WHERE id = $2
}
// RequestRepo automatically satisfies interfaces.RequestReader ✅
```

```go
// ──────────────────────────────────────────────────────────────
// Step 3: Hosama receives it via constructor injection
// ──────────────────────────────────────────────────────────────
// approvals/service/approval_service.go
package service

import "settle/internal/shared/interfaces"

type ApprovalService struct {
    approvalRepo  *repository.ApprovalRepo
    requestReader interfaces.RequestReader   // ← injected, not imported
    auditWriter   interfaces.AuditWriter     // ← same pattern
}

func NewApprovalService(
    approvalRepo  *repository.ApprovalRepo,
    requestReader interfaces.RequestReader,   // ← Menweyelet's repo passed in
    auditWriter   interfaces.AuditWriter,
) *ApprovalService {
    return &ApprovalService{
        approvalRepo:  approvalRepo,
        requestReader: requestReader,
        auditWriter:   auditWriter,
    }
}

func (s *ApprovalService) Approve(ctx context.Context, orgID, requestID, approverID, note string) (*ApprovalResponse, error) {
    req, err := s.requestReader.GetByID(ctx, orgID, requestID)  // ← clean interface call
    if err != nil { return nil, err }
    if req.Status != enums.StatusPending {
        return nil, utils.NewConflictError("Request is not pending")
    }
    // ... create approval record, update status via requestReader, write audit ...
}
```

```go
// ──────────────────────────────────────────────────
// Step 4: Wire everything in main.go
// ──────────────────────────────────────────────────
// cmd/settle/main.go
func main() {
    db := database.Connect(cfg)

    // Menweyelet's repos
    userRepo    := auth.NewUserRepo(db)
    requestRepo := requests.NewRequestRepo(db)    // implements interfaces.RequestReader
    receiptRepo := requests.NewReceiptRepo(db)
    commentRepo := comments.NewCommentRepo(db)

    // Hosama's repos
    approvalRepo   := approvals.NewApprovalRepo(db)
    orgRepo        := admin.NewOrgRepo(db)
    memberRepo     := admin.NewMemberRepo(db)
    invitationRepo := admin.NewInvitationRepo(db)
    auditRepo      := admin.NewAuditRepo(db)

    // Shared services
    jwtService  := shared.NewJWTService(cfg.JWTSecret)
    hashService := shared.NewHashService()
    fileService := shared.NewFileService(cfg.S3Bucket)
    auditWriter := admin.NewAuditWriter(auditRepo)  // implements interfaces.AuditWriter

    // Menweyelet's services
    authService    := auth.NewAuthService(userRepo, jwtService, hashService)
    requestService := requests.NewRequestService(requestRepo, receiptRepo, auditWriter)
    commentService := comments.NewCommentService(commentRepo, requestRepo, auditWriter)

    // Hosama's services — INJECT Menweyelet's requestRepo as RequestReader
    approvalService := approvals.NewApprovalService(
        approvalRepo,
        requestRepo,    // ← satisfies interfaces.RequestReader, no import needed ✅
        auditWriter,
    )
    orgService        := admin.NewOrgService(orgRepo, memberRepo, auditWriter)
    invitationService := admin.NewInvitationService(
        invitationRepo, memberRepo, hashService, jwtService, auditWriter,
    )
    dashboardService := admin.NewDashboardService(orgRepo, requestRepo)
    //                                                      ^^^^^^^^^^^
    //                                     Same pattern — injected for read access

    // Handlers → Router → Serve ...
}
```

**Why interface injection wins:**

| Approach | Duplicated SQL? | Circular Import? | Testable? |
|---|---|---|---|
| ❌ Direct import | No | **Yes** 💀 | Hard |
| ❌ Each dev writes own queries | **Yes** 🔁 | No | Ok |
| ✅ Interface injection | **No** | **No** | **Easy** (mock it) |

> [!TIP]
> **Testing bonus** — Hosama can mock `RequestReader` in unit tests without needing a real database or Menweyelet's code:
> ```go
> type mockRequestReader struct {}
> func (m *mockRequestReader) GetByID(ctx, orgID, id string) (*models.Request, error) {
>     return &models.Request{ID: id, Status: "pending"}, nil
> }
> // Now test ApprovalService in complete isolation ✅
> ```

---

### Router — Marked Sections

```go
// internal/router/router.go

func SetupRouter(
    authHandler    *auth.AuthHandler,       // Menweyelet injects
    requestHandler *requests.RequestHandler, // Menweyelet injects
    commentHandler *comments.CommentHandler, // Menweyelet injects
    receiptHandler *receipts.ReceiptHandler, // Menweyelet injects
    approvalHandler *approvals.ApprovalHandler, // Hosama injects
    adminHandler    *admin.AdminHandler,        // Hosama injects
) *gin.Engine {

    r := gin.Default()
    r.Use(middleware.ErrorHandler())
    r.Use(middleware.RequestLogger())

    v1 := r.Group("/v1")

    // ════════════════════════════════════════
    // 🟢 MENWEYELET'S ROUTES — DO NOT TOUCH
    // ════════════════════════════════════════
    v1.POST("/auth/login", authHandler.Login)
    v1.POST("/auth/signup", authHandler.Signup)

    authenticated := v1.Group("", middleware.AuthRequired())
    {
        authenticated.POST("/requests", middleware.RequireRole("staff"), requestHandler.Create)
        authenticated.GET("/requests", requestHandler.List)
        authenticated.GET("/requests/:id", requestHandler.GetDetail)
        authenticated.GET("/requests/:id/previous", middleware.RequireRole("finance", "org_admin"), requestHandler.GetPrevious)
        authenticated.PUT("/requests/:id/withdraw", middleware.RequireRole("staff"), requestHandler.Withdraw)
        authenticated.POST("/requests/:id/resubmit", middleware.RequireRole("staff"), requestHandler.Resubmit)
        authenticated.POST("/requests/:id/comments", commentHandler.Add)
        authenticated.GET("/requests/:id/comments", commentHandler.List)
        authenticated.POST("/receipts/upload", receiptHandler.Upload)
        authenticated.GET("/receipts/:id", receiptHandler.Get)
    }

    // ════════════════════════════════════════
    // 🔵 HOSAMA'S ROUTES — DO NOT TOUCH
    // ════════════════════════════════════════
    authenticated.PUT("/requests/:id/approve", middleware.RequireRole("finance", "org_admin"), approvalHandler.Approve)
    authenticated.PUT("/requests/:id/reject", middleware.RequireRole("finance", "org_admin"), approvalHandler.Reject)
    authenticated.PUT("/requests/:id/mark-paid", middleware.RequireRole("finance", "org_admin"), approvalHandler.MarkPaid)
    authenticated.PUT("/requests/:id/payment-failed", middleware.RequireRole("finance", "org_admin"), approvalHandler.MarkFailed)

    authenticated.POST("/organizations", adminHandler.CreateOrg)
    authenticated.GET("/organizations/:id", adminHandler.GetOrg)
    authenticated.PUT("/organizations/:id", middleware.RequireRole("org_admin"), adminHandler.UpdateOrg)
    authenticated.GET("/organizations/:id/members", middleware.RequireRole("org_admin"), adminHandler.GetMembers)
    authenticated.POST("/organizations/:id/invitations", middleware.RequireRole("org_admin"), adminHandler.CreateInvitation)
    authenticated.DELETE("/organizations/:id/members/:uid", middleware.RequireRole("org_admin"), adminHandler.RemoveMember)
    authenticated.PUT("/organizations/:id/members/:uid/role", middleware.RequireRole("org_admin"), adminHandler.ChangeRole)
    authenticated.GET("/dashboard/summary", middleware.RequireRole("finance", "org_admin"), adminHandler.DashboardSummary)
    authenticated.GET("/organizations/:id/stats", middleware.RequireRole("finance", "org_admin"), adminHandler.OrgStats)
    authenticated.GET("/organizations/:id/audit-log", middleware.RequireRole("finance", "org_admin"), adminHandler.AuditLog)
    authenticated.PUT("/organizations/:id/plan", middleware.RequireRole("org_admin"), adminHandler.ChangePlan)

    // Public
    v1.GET("/billing/plans", adminHandler.GetPlans)
    v1.POST("/invitations/:token/accept", adminHandler.AcceptInvitation)

    return r
}
```

---

## 📅 9-Day Schedule

| Day | 🟢 Menweyelet | 🔵 Hosama | Deliverable |
|---|---|---|---|
| **1** | 🤝 **Together**: Go project init, docker-compose, all migrations, shared models/enums/middleware/config, router skeleton, test helpers | 🤝 Same | `go build` compiles, DB runs, empty server starts |
| **2** | `auth/` — login, signup, JWT service, hash service, user repo | `admin/` — org create/get/update, org repo, member repo, org middleware | Login returns JWT ✅ Create org works ✅ |
| **3** | `requests/` — create, list, detail, ID generation, pagination, filters | `admin/` — invitations (create, accept), member CRUD (list, remove, change role) | Submit + list requests ✅ Full invite flow ✅ |
| **4** | `receipts/` — file upload, S3 service, OCR worker (mock), receipt GET | `approvals/` — approve, reject, mark-paid, payment-failed, state validation | Upload receipt ✅ Approve→Paid flow ✅ |
| **5** | `comments/` — add, list · `requests/` — withdraw, resubmit, previous | `admin/` — audit log service/handler, dashboard summary with escalation | Comments ✅ Withdraw ✅ Audit ✅ Dashboard ✅ |
| **6** | Enrich request detail (receipt+approval+comments+timeline), validation hardening | Billing (plans, change plan), org stats, plan limit enforcement | Rich detail ✅ Billing ✅ |
| **7** | Integration tests: signup→request→receipt→comment→withdraw | Integration tests: approve→paid→audit, invite→accept→submit→approve | Cross-feature flows verified ✅ |
| **8** | Security: JWT expiry, rate limiting, SQL review, error polish | Security: org isolation audit, double-approve prevention, edge cases | Hardened ✅ |
| **9** | Final tests, API docs (auth+requests), demo rehearsal | Final tests, API docs (approvals+admin), demo rehearsal | 🚀 **Ship it** |

---

## 🌿 Git Branches

| Developer | Branch Pattern | Example |
|---|---|---|
| 🟢 Menweyelet | `feature/m-{feature}` | `feature/m-auth`, `feature/m-requests`, `feature/m-receipts`, `feature/m-comments` |
| 🔵 Hosama | `feature/h-{feature}` | `feature/h-admin`, `feature/h-approvals`, `feature/h-dashboard`, `feature/h-billing` |
| 🤝 Both | `shared/setup` | Day 1 only |

**Rules:** Each reviews the other's PR before merging to `main`. No force pushes. CI tests must pass.

---

## 🏁 Migration Ownership

| # | File | Owner | Table |
|---|---|---|---|
| 001 | `create_users.sql` | 🟢 Menweyelet | `users` |
| 002 | `create_organizations.sql` | 🔵 Hosama | `organizations` |
| 003 | `create_org_members.sql` | 🔵 Hosama | `org_members` |
| 004 | `create_requests.sql` | 🟢 Menweyelet | `requests` |
| 005 | `create_receipts.sql` | 🟢 Menweyelet | `receipts` |
| 006 | `create_approvals.sql` | 🔵 Hosama | `approvals` |
| 007 | `create_comments.sql` | 🟢 Menweyelet | `comments` |
| 008 | `create_audit_log.sql` | 🔵 Hosama | `audit_log` |
| 009 | `create_invitations.sql` | 🔵 Hosama | `invitations` |

> [!NOTE]
> All migration files are written together on **Day 1** to avoid ordering conflicts. After that, no one touches another's migration.

---

> _"Clear domains. Clean code. Ship fast."_
> _— Menweyelet & Hosama, 9 days to launch 🚀_
