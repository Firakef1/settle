# Requests Domain — Implementation Phase Design

Companion to `requests_domain_design.md`. That document says **what** the domain is; this one says **how to build it, in what order, and when each phase is done**.

Assumptions: Go, Gin, PostgreSQL. Where your project already has a convention (migration tool, ID types, error-response format, auth middleware), follow it. Anything marked **(adapt)** depends on that.

## How the phases fit together

```text
Phase 1  Foundations        compiles, migrates, tested lower layers, routes return 501
   │
   ▼
Phase 2  Endpoints          all 10 routes + submit work end to end (4 mergeable slices)
   │
   ▼
Phase 3  Finishing touches  real approval/audit data, full timeline, hardening, docs
```

Rules that keep each phase standalone:

- A phase only imports code from itself and earlier phases.
- Every phase ends with a green build, green tests, and something demonstrable.
- Cross-domain dependencies (Approval, Audit, file storage, OCR) are defined as **ports** (small interfaces owned by this domain) in Phase 1, with no-op or mock implementations. Phase 2 calls the ports. Phase 3 swaps in the real adapters. No Phase 2 code changes when the other developer's code lands.

---

# Phase 1 — Foundations

**Goal:** everything the endpoints sit on. No business behavior is exposed yet.

**Depends on:** nothing in this domain.

## 1.1 Package layout

```text
internal/requests/
├── errors.go                     domain errors (no imports)
├── handler/
│   ├── request_handler.go        RequestHandler + 501 stubs
│   ├── auth_context.go           extract Actor from gin.Context
│   ├── errors.go                 domain error → HTTP mapping
│   └── routes.go                 RegisterRoutes
├── service/
│   ├── request_service.go        RequestService, constructor, stubs
│   ├── actor.go                  Actor, Role
│   ├── access.go                 access-control helpers
│   ├── ports.go                  all interfaces the service depends on
│   └── noop.go                   no-op AuditEmitter / ApprovalReader
├── repository/
│   ├── request_repo.go           requests, sequence, transactions
│   ├── receipt_repo.go           receipts
│   └── comment_repo.go           comments
├── dto/                          as in design §3
├── validator/request_validator.go
├── model/
│   ├── request.go
│   ├── receipt.go
│   ├── comment.go
│   └── read_models.go            RequestListItem, RequestWithRequester, CommentView
├── infra/
│   └── mockocr/mock_ocr.go       mock OCRService
└── testutil/                     test DB helper, fake FileService
```

The design says one handler, one service, one repository. Splitting each into a few files in the same package keeps that single type per layer while avoiding a 1,000-line file.

## 1.2 Migrations

One migration set, **(adapt)** to your tool and ID types.

```sql
CREATE SEQUENCE request_id_seq START 1;

CREATE TABLE requests (
    id             TEXT PRIMARY KEY,                 -- REQ-000042
    org_id         UUID NOT NULL,
    requester_id   UUID NOT NULL,
    type           TEXT NOT NULL
                   CHECK (type IN ('reimbursement','advance','stipend')),
    amount         NUMERIC(12,2) NOT NULL
                   CHECK (amount > 0 AND amount <= 999999.99),
    purpose        TEXT NOT NULL
                   CHECK (char_length(purpose) BETWEEN 10 AND 500),
    urgency        TEXT NOT NULL
                   CHECK (urgency IN ('routine','urgent','critical')),
    status         TEXT NOT NULL,                    -- no CHECK: Approval owns more statuses
    resubmitted_as TEXT NULL REFERENCES requests(id),
    resubmitted_at TIMESTAMPTZ NULL,
    submitted_at   TIMESTAMPTZ NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_requests_org_status    ON requests (org_id, status);
CREATE INDEX idx_requests_org_requester ON requests (org_id, requester_id, created_at DESC);

CREATE TABLE receipts (
    id               UUID PRIMARY KEY,
    request_id       TEXT NOT NULL REFERENCES requests(id),
    org_id           UUID NOT NULL,
    file_key         TEXT NOT NULL,                  -- storage key, URL is derived
    file_name        TEXT NOT NULL,
    file_size        BIGINT NOT NULL,
    file_type        TEXT NOT NULL,
    ocr_status       TEXT NOT NULL DEFAULT 'pending'
                     CHECK (ocr_status IN ('pending','processing','success','failed')),
    extracted_amount NUMERIC(12,2) NULL,
    merchant         TEXT NULL,
    extracted_date   DATE NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_receipts_request ON receipts (request_id);

CREATE TABLE comments (
    id         UUID PRIMARY KEY,
    request_id TEXT NOT NULL REFERENCES requests(id),
    author_id  UUID NOT NULL,
    text       TEXT NOT NULL CHECK (char_length(text) BETWEEN 1 AND 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_comments_request ON comments (request_id, created_at);
```

Notes:

- The receipt stores `file_key`, not a URL. Presigned URLs expire, so the URL is produced at read time with `FileService.GetURL(key)`.
- `resubmitted_at` is an addition to the design. It lets the timeline show when a resubmission happened without an extra lookup.
- The DB `CHECK`s duplicate the validator on purpose. They are the last line of defense.

## 1.3 Models (`model/`)

- `Request`, `Receipt`, `Comment` as in design §27, with these changes: `Request` has `SubmittedAt *time.Time` and `ResubmittedAt *time.Time`, no `ReceiptID`; `Receipt` has `FileKey` instead of `FileURL`.
- Read models returned by joins:
  - `RequestListItem` (id, amount, purpose, urgency, status, created_at, requester id/name/email, days_pending)
  - `RequestWithRequester` (full request plus requester id/name/email)
  - `CommentView` (comment plus author id/name/role)
- Constants: `StatusDraft`, `StatusPending`, `StatusWithdrawn`, `StatusRejected`, `StatusFailed`; `TypeReimbursement`, ...; `UrgencyRoutine`, ...; `OCRPending`, ...

## 1.4 Domain errors (`errors.go`)

The list from design §29, plus:

```go
ErrReceiptsLocked   = errors.New("receipts can only be changed while the request is a draft")
ErrCannotSubmit     = errors.New("only draft requests can be submitted")
ErrValidation       = errors.New("validation failed")   // wrapped with field details
ErrNotImplemented   = errors.New("not implemented")
```

`handler/errors.go` maps them:

| Error | HTTP |
|---|---|
| `ErrRequestNotFound`, `ErrReceiptNotFound` | 404 |
| `ErrUnauthorizedRequest`, `ErrCommentNotAllowed` | 403 |
| `ErrValidation`, `ErrReceiptRequired` | 400 |
| `ErrInvalidRequestStatus`, `ErrCannotWithdraw`, `ErrCannotResubmit`, `ErrCannotSubmit`, `ErrReceiptsLocked` | 409 |
| `ErrNotImplemented` | 501 |
| anything else | 500 (log the cause, return a generic message) |

Use your project's error-response body format **(adapt)**.

## 1.5 DTOs (`dto/`)

Exactly the shapes in the design, with the updates already made there: `CreateRequestDTO` (no `receipt_id`), `ResubmitRequestDTO` (`receipt_mode`), `CreateCommentDTO`, `RequestResponse` (with `Receipts []`), `RequestListItem` response, `RequestListResponse`, `PaginationMeta`, `PreviousRequestsResponse`, `ReceiptResponse` (with `request_id`), `CommentResponse`, `AuthorResponse`, `TimelineEvent`, `ApprovalResponse`.

Add a small `StatusChangeResponse { id, status, updated_at }` for submit and withdraw.

Conversion functions (`model` → `dto`) live in `dto` or a `mapper.go` next to it. Keep them pure so they are trivially unit-testable.

## 1.6 Validator (`validator/`)

Implements design §28 as pure functions with no I/O:

- `ValidateCreate(dto)`: amount range, purpose length (count runes, not bytes, after trimming), type, urgency.
- `ValidateResubmit(dto)`: provided fields only, plus `receipt_mode ∈ {carry,new}`. The merged result is re-validated with `ValidateCreate` by the service.
- `ValidateComment(dto)`: trimmed text, 1–1000 runes.
- `ValidateListFilters(f) (ListFilter, error)`: whitelists `SortBy` (`created_at`, `amount`, `urgency`, `status`) and `SortOrder` (`asc`, `desc`), validates `Status`/`Urgency` values, and **applies defaults and limits**: `Limit` default 20, max 100, `Offset` ≥ 0.
- `ValidateReceiptFile(name, size, header []byte)`: size ≤ 10 MB, content type detected from the first 512 bytes (`http.DetectContentType`), allowed `image/jpeg`, `image/png`, `application/pdf`. The filename extension is never trusted.

Errors are returned wrapped around `ErrValidation` with the offending field named.

## 1.7 Ports (`service/ports.go`)

Interfaces are defined where they are consumed (the service package).

```go
type RequestRepository interface {
    WithTx(ctx context.Context, fn func(RequestRepository) error) error

    // Requests
    NextSequence(ctx context.Context) (int64, error)
    Create(ctx context.Context, r *model.Request) error
    FindByID(ctx context.Context, orgID, id string) (*model.Request, error)
    FindWithRequester(ctx context.Context, orgID, id string) (*model.RequestWithRequester, error)
    FindByIDForUpdate(ctx context.Context, orgID, id string) (*model.Request, error)
    List(ctx context.Context, f model.ListFilter) ([]model.RequestListItem, error)
    Count(ctx context.Context, f model.ListFilter) (int, error)
    TransitionStatus(ctx context.Context, orgID, id, from, to string) error
    MarkSubmitted(ctx context.Context, orgID, id string) error
    SetResubmittedAs(ctx context.Context, orgID, id, newID string) error

    // Receipts
    CreateReceipt(ctx context.Context, r *model.Receipt) error
    ListReceipts(ctx context.Context, orgID, requestID string) ([]model.Receipt, error)
    CountReceipts(ctx context.Context, orgID, requestID string) (int, error)
    CopyReceipts(ctx context.Context, orgID, fromID, toID string) error
    UpdateReceiptOCR(ctx context.Context, receiptID, status string, res *OCRResult) error

    // Comments
    CreateComment(ctx context.Context, c *model.Comment) error
    ListComments(ctx context.Context, orgID, requestID string) ([]model.CommentView, error)
}

type FileService interface {
    Upload(ctx context.Context, key string, file io.Reader) (string, error)
    GetURL(key string) string
    Delete(ctx context.Context, key string) error   // added: cleanup of orphaned uploads
}

type OCRService interface {
    Process(ctx context.Context, fileURL string) (*OCRResult, error)
}

type AuditEmitter interface {
    Emit(ctx context.Context, e AuditEvent)
}

type ApprovalReader interface {
    GetForRequest(ctx context.Context, orgID, requestID string) (*ApprovalInfo, error)
}
```

Differences from the design's repository interface, and why:

| Change | Reason |
|---|---|
| `WithTx` | Gives the service a way to run several repository calls in one transaction. |
| `NextSequence` | The service must be able to get the next `request_id_seq` value. |
| `FindByIDForUpdate` | Row lock for submit, upload, and resubmit. |
| `TransitionStatus(from, to)` replaces `UpdateStatus` | A conditional `UPDATE ... WHERE status = from` makes status changes race-safe. Zero rows affected returns `ErrInvalidRequestStatus`. |
| `CountReceipts`, `CopyReceipts`, `UpdateReceiptOCR` | Needed for submit, resubmit-carry, and OCR results. |
| `ListFilter` also has `ExcludeDrafts` and `ExcludeID` | Draft visibility and the "previous requests" query. |
| `FileService.Delete` | Cleans up the stored file if the DB insert fails. If the shared `FileService` can't be changed, skip it and log orphans instead. |

`noop.go` provides `noopAudit` (does nothing) and `noopApproval` (returns `nil, nil`).

## 1.8 Repository (`repository/`)

- `request_repo.go` holds a `*sql.DB` or `pgxpool.Pool` **(adapt)** and a "querier" that is either the pool or the current transaction. `WithTx` creates a copy of the repo bound to a transaction and commits or rolls back based on the returned error.
- `NextSequence`: `SELECT nextval('request_id_seq')`. Gaps after rollbacks are acceptable.
- `List` / `Count` build the query from `ListFilter` with parameterized SQL only. Sorting uses a whitelist map from `SortBy` to a SQL fragment, never string-concatenated client input. The default order is the design's urgency-then-oldest-first. `days_pending` uses `COALESCE(submitted_at, created_at)`.
- `List` applies: `org_id`, `requester_id` (if set), `status`, `urgency`, `ExcludeDrafts` (`status <> 'draft'`), `ExcludeID`.
- Every query that takes an ID also takes `org_id`, so a wrong-org lookup simply returns "not found".
- Map "no rows" to `ErrRequestNotFound` / `ErrReceiptNotFound`.

## 1.9 Service skeleton (`service/`)

```go
type RequestService struct {
    repo      RequestRepository
    files     FileService
    ocr       OCRService
    audit     AuditEmitter     // defaults to noopAudit
    approvals ApprovalReader   // defaults to noopApproval
    validator *validator.RequestValidator
    now       func() time.Time // injectable for tests
}

func NewRequestService(repo RequestRepository, files FileService, ocr OCRService,
    v *validator.RequestValidator, opts ...Option) *RequestService
```

`Option`s: `WithAudit`, `WithApprovals`, `WithClock`.

Public methods all exist with the final signatures and return `ErrNotImplemented`. They take an `Actor`:

```go
type Role string
const (RoleStaff Role = "staff"; RoleFinance Role = "finance"; RoleOrgAdmin Role = "org_admin")

type Actor struct { UserID, OrgID string; Role Role }
```

(Passing one `Actor` instead of separate `orgID`/`userID` is a small change from the design's Withdraw example. It is needed because most operations also depend on the role.)

### Access helpers (`access.go`), fully implemented and unit-tested in this phase

```go
func (a Actor) isOwner(r *model.Request) bool
func (a Actor) isReviewer() bool                      // finance or org_admin
func (s *RequestService) authorizeView(a Actor, r *model.Request) error
```

`authorizeView` applies, in this order:

1. `r.OrgID != a.OrgID` → `ErrRequestNotFound`
2. `r.Status == draft && !owner` → `ErrRequestNotFound` (drafts are invisible to everyone but the owner)
3. staff and not owner → `ErrUnauthorizedRequest`
4. otherwise allow

`buildListFilter(a Actor, f ListFilter) ListFilter`: for staff, forces `RequesterID = a.UserID`; for reviewers, sets `ExcludeDrafts = true`.

## 1.10 Handler skeleton and routes

- `auth_context.go`: `actorFrom(c *gin.Context) (service.Actor, error)` reads what your auth middleware stored **(adapt)**.
- `request_handler.go`: all 11 methods (`Create`, `Submit`, `List`, `GetDetail`, `GetPrevious`, `Withdraw`, `Resubmit`, `UploadReceipt`, `ListReceipts`, `AddComment`, `ListComments`). In this phase each one calls its service stub and maps the error, which produces `501`.
- `routes.go`:

```go
func RegisterRoutes(rg *gin.RouterGroup, h *RequestHandler) {
    r := rg.Group("/requests")
    r.POST("", h.Create)
    r.GET("", h.List)
    r.GET("/:id", h.GetDetail)
    r.GET("/:id/previous", h.GetPrevious)
    r.POST("/:id/submit", h.Submit)
    r.PUT("/:id/withdraw", h.Withdraw)
    r.POST("/:id/resubmit", h.Resubmit)
    r.POST("/:id/receipts", h.UploadReceipt)
    r.GET("/:id/receipts", h.ListReceipts)
    r.POST("/:id/comments", h.AddComment)
    r.GET("/:id/comments", h.ListComments)
}
```

- Wire it in `main`/router setup behind the existing auth middleware.

## 1.11 Mock OCR and test helpers

- `infra/mockocr`: returns a canned `OCRResult` after a short delay. Include a switch (e.g. a file name containing `fail`) to force a failure, so Phase 2 can test the failure path.
- `testutil`: starts a throwaway PostgreSQL (testcontainers or your existing approach), runs migrations, and provides a fake in-memory `FileService`.

## 1.12 Build order inside the phase

1. Migrations and models.
2. Errors, constants, DTOs.
3. Validator (+ unit tests).
4. Ports.
5. Repository (+ integration tests).
6. Actor, access helpers (+ unit tests).
7. Service skeleton, handler skeleton, routes.
8. Mock OCR, test utilities.

## 1.13 Tests

- Validator: every rule boundary (amount `0`, `0.01`, `999999.99`, `1000000`; purpose 9/10/500/501 characters; file types and sizes).
- `formatRequestID`: `REQ-000001`, `REQ-001024`; sequence uniqueness under concurrent `NextSequence` calls.
- Repository integration: create/find, org isolation, list filters, sorting whitelist, pagination plus count, `ExcludeDrafts`, `ExcludeID`, `TransitionStatus` conditional update, `FindByIDForUpdate` inside `WithTx`, `WithTx` rollback, receipt and comment CRUD, `CopyReceipts`.
- Access helpers: full matrix of {staff owner, staff non-owner, finance, admin, wrong org} × {draft, pending}.
- Handler: every route returns `501` with the right error body, and unauthenticated calls are rejected.

## 1.14 Definition of done

- [ ] `go build ./...` and `go vet` pass; migrations apply and roll back cleanly.
- [ ] All Phase 1 tests pass.
- [ ] All 11 routes are registered and return `501`.
- [ ] Nothing outside `internal/requests/` (besides router wiring and migrations) was changed.

---

# Phase 2 — Endpoints

**Goal:** every route behaves as specified.

**Depends on:** Phase 1 only.

Build in four slices. Each slice is a separate PR, merges on its own, and leaves the system working.

## Slice 2A — Core requests

### `POST /requests` → `201`

```text
handler: bind CreateRequestDTO, actorFrom
service.Create:
  1. validator.ValidateCreate
  2. seq := repo.NextSequence → id := formatRequestID(seq)
  3. status := draft if type == reimbursement else pending
     submitted_at := now if pending, else nil
  4. repo.Create
  5. audit.Emit(request.created)        // no-op until Phase 3
  6. return repo.FindWithRequester → response
```

Any authenticated user in the org may create a request for themselves. `requester_id` and `org_id` always come from the `Actor`, never from the body.

### `GET /requests` → `200`

```text
handler: bind query (status, urgency, requester_id, sort_by, sort_order, limit, offset)
service.List:
  1. validator.ValidateListFilters (defaults, limits, whitelists)
  2. buildListFilter(actor, filters)        // staff forced to own, reviewers exclude drafts
  3. items := repo.List, total := repo.Count
  4. meta := {total, limit, offset, has_more: offset+len(items) < total}
```

A staff user passing someone else's `requester_id` is silently overridden with their own ID (design §9).

### `GET /requests/:id` → `200`

```text
service.GetDetail:
  1. repo.FindWithRequester(org, id)
  2. authorizeView
  3. receipts := repo.ListReceipts (URLs via files.GetURL(key))
  4. comments := repo.ListComments
  5. timeline := basic: "submitted" event from submitted_at (if set)
  6. approval := nil               // filled in Phase 3
```

### `GET /requests/:id/previous` → `200`

```text
service.GetPrevious:
  1. repo.FindByID + authorizeView          // same rules as detail
  2. repo.List with RequesterID = request.RequesterID,
        ExcludeID = id, ExcludeDrafts = true, sort created_at desc, limit 20
  3. return {data: [...]}
```

**Slice 2A tests:** all creation cases (§33), list access/filter/pagination/count/`days_pending`, detail access matrix, previous excludes the current request and drafts.

## Slice 2B — Lifecycle

### `POST /requests/:id/submit` → `200` (`StatusChangeResponse`)

```text
service.Submit (inside repo.WithTx):
  1. FindByIDForUpdate
  2. org check (else not found); owner check (else ErrUnauthorizedRequest)
  3. status == draft else ErrCannotSubmit
  4. if type == reimbursement: CountReceipts >= 1 else ErrReceiptRequired
  5. MarkSubmitted (status = pending, submitted_at = now)
after commit: audit.Emit(request.submitted)
```

### `PUT /requests/:id/withdraw` → `200` (`StatusChangeResponse`)

```text
service.Withdraw:
  1. FindByID, org check, owner check
  2. status in {draft, pending} else ErrCannotWithdraw
  3. TransitionStatus(from = current status, to = withdrawn)   // race-safe
after: audit.Emit(request.withdrawn)
```

### `POST /requests/:id/resubmit` → `201` (new request)

```text
service.Resubmit (inside repo.WithTx):
  1. FindByIDForUpdate(old); org + owner checks
  2. status in {rejected, failed} else ErrCannotResubmit
  3. old.ResubmittedAs != nil → ErrCannotResubmit (already resubmitted)
  4. merge: copy old fields, apply non-nil overrides from DTO
  5. validator.ValidateCreate(merged)
  6. receipt_mode := dto.ReceiptMode or "carry"
  7. determine status:
       not reimbursement                           → pending
       reimbursement + carry + old has >= 1 receipt → pending
       otherwise                                    → draft
  8. NextSequence → new id; Create(new)
  9. if carry: CopyReceipts(old → new)    // new rows, same file_key, OCR data copied
 10. SetResubmittedAs(old, new id)        // also sets resubmitted_at
after commit: audit.Emit(request.resubmitted)
```

Rule 3 (a request can be resubmitted only once) is an addition to the design. Without it, the same rejected request could spawn several new requests.

**Slice 2B tests:** full submit, withdraw, and resubmit lists from §33, including carry/new modes, resulting status, linkage, old receipts untouched, and rollback when any step fails (use a repo wrapper that errors at step 9 or 10).

## Slice 2C — Receipts

### `POST /requests/:id/receipts` → `201`

Multipart form with a single `file` field.

```text
handler:
  - cap body with http.MaxBytesReader (10 MB + small overhead)
  - read file header, pass UploadReceiptInput{FileName, Size, Header, Reader}
service.UploadReceipt:
  1. repo.FindByID; org + owner checks; status == draft else ErrReceiptsLocked   // cheap pre-check
  2. validator.ValidateReceiptFile (size, sniffed content type)
  3. key := receipts/{orgID}/{uuid}.{ext}      // ext from the detected type
  4. files.Upload(key, reader)                  // outside any DB transaction
  5. repo.WithTx:
       FindByIDForUpdate; re-check draft + owner
       CreateReceipt(ocr_status = processing)
     on failure: files.Delete(key) (best effort), return error
  6. start OCR (below)
  7. return ReceiptResponse (ocr_status = processing)
```

Storing the file before the transaction avoids holding a row lock during a slow upload. The re-check inside the transaction closes the race with a concurrent Submit.

**OCR execution:** run `ocr.Process` in a goroutine using `context.WithoutCancel(ctx)` plus a timeout. It must `recover()` from panics. On success it calls `UpdateReceiptOCR(success, result)`. On error it calls `UpdateReceiptOCR(failed, nil)`. The client polls `GET /requests/:id/receipts` for the result.

### `GET /requests/:id/receipts` → `200`

```text
service.ListReceipts:
  1. FindByID; authorizeView
  2. repo.ListReceipts → {data: [ReceiptResponse]}   // URL via files.GetURL(file_key)
```

A receipt can only be reached through its parent request, so a receipt from another request or org is never accessible.

**Slice 2C tests:** JPEG/PNG/PDF accepted; renamed `.exe` → `.pdf` rejected by content sniffing; >10 MB rejected; multiple uploads on one request; upload on pending request → locked; non-owner upload rejected; storage failure leaves no DB row; DB failure triggers `Delete`; OCR success and failure both update the status; detail response now includes receipts.

## Slice 2D — Comments

### `POST /requests/:id/comments` → `201`

```text
service.AddComment:
  1. validator.ValidateComment
  2. repo.FindByID; authorizeView          // staff: own requests only, reviewers: whole org (non-draft)
  3. CreateComment (author = actor)
  4. return CommentResponse (author name/role via the view model)
```

### `GET /requests/:id/comments` → `200`

```text
service.ListComments:
  1. FindByID; authorizeView
  2. repo.ListComments (ORDER BY created_at ASC)
```

Comments are allowed on any status the actor can see. `ErrCommentNotAllowed` stays reserved for a future status-based rule.

**Slice 2D tests:** valid, empty, whitespace-only, 1000 vs 1001 characters, staff on own / other's request, finance on org request, wrong org, finance commenting on a draft (404).

## 2.x Phase 2 definition of done

- [ ] All 11 routes behave per the design, with the correct status codes.
- [ ] Every test list in design §33 for creation, listing, detail, submit, withdraw, resubmit, receipts, and comments passes.
- [ ] Audit and approval ports are called at the right places (no-ops for now).
- [ ] Detail returns request, requester, receipts, comments, basic timeline. `approval` is absent.
- [ ] A manual walkthrough works: create reimbursement → upload 2 receipts → submit → withdraw, and reject (simulated in the DB) → resubmit.

---

# Phase 3 — Finishing touches

**Goal:** connect to the rest of the system and harden the domain.

**Depends on:** Phases 1 and 2 only. The Approval and Audit developer's code is consumed through the ports from Phase 1.

## 3.1 Approval integration

- Write `approvalAdapter` implementing `ApprovalReader` against the other developer's public interface. It maps their type to `ApprovalInfo` (decision, decided_by, decided_at, comment, payment info if exposed).
- Wire with `WithApprovals(adapter)` in the composition root.
- `GetDetail` fills `RequestResponse.Approval`.
- **Failure policy:** if the approval lookup fails, log the error and return the detail without `approval`. A broken approval service should not make the whole request unreadable. Change this if you prefer strictness.

## 3.2 Audit integration

- Write `auditAdapter` implementing `AuditEmitter` against the Audit domain's interface. Wire with `WithAudit`.
- Events (all already emitted in Phase 2): `request.created`, `request.submitted`, `request.withdrawn`, `request.resubmitted`. Optionally `receipt.uploaded` and `comment.added`.
- Payload: org, actor, entity type/ID, action, relevant metadata (e.g. old and new request IDs on resubmit).
- Emit **after** the transaction commits, so there is never an audit event for something that rolled back.
- **Failure policy:** log and continue (best effort). If audit must be guaranteed, the Audit developer needs to offer a transactional/outbox variant, which is a bigger change.

## 3.3 Full timeline

A pure function `buildTimeline(req, approval) []TimelineEvent`, sorted by timestamp:

| Event | Source |
|---|---|
| `submitted` | `submitted_at` |
| `approved` / `rejected` / `paid` / `failed` | approval info timestamps |
| `withdrawn` | `updated_at` when status is `withdrawn` |
| `resubmitted` | `resubmitted_at` when `resubmitted_as` is set |

Drafts have no events. The function has no I/O, so it is tested with a table of inputs.

## 3.4 Hardening

- **Transactions:** failure-injection tests for resubmit (fail after insert, after receipt copy, after link) assert that no partial rows remain.
- **Concurrency**, using real Postgres and parallel goroutines:
  - two simultaneous submits → exactly one succeeds
  - upload racing with submit → either the receipt is attached and counted, or upload gets `ErrReceiptsLocked`; never a pending request with a late receipt
  - two simultaneous resubmits of the same request → exactly one new request
  - two simultaneous withdraws → one succeeds
- **OCR:** failure and timeout set `failed`. Add a sweeper (or startup job) that moves receipts stuck in `processing` longer than N minutes to `failed`, since a process restart kills the goroutine.
- **Pagination:** verify defaults and the maximum limit through the HTTP layer; verify `has_more` and `total` at page boundaries.
- **Input hardening:** body size limits on JSON endpoints, and consistent trimming/normalization of text fields.
- **Real OCR provider:** if one is available, implement `OCRService` for it and swap it in the composition root. No other code changes.

## 3.5 Full test pass

Role-by-endpoint matrix, run at the HTTP level:

| Endpoint | Staff owner | Staff other | Finance | Org admin | Other org |
|---|---|---|---|---|---|
| Create | ✓ | n/a | ✓ (own) | ✓ (own) | n/a |
| List | own only | n/a | org, no drafts | org, no drafts | not visible |
| Detail / previous / receipts list / comments list | ✓ | 403 | ✓ (draft → 404) | ✓ (draft → 404) | 404 |
| Submit / Withdraw / Resubmit | ✓ (owner) | 403 | 403 | 403 | 404 |
| Upload receipt | ✓ (draft only) | 403 | 403 | 403 | 404 |
| Add comment | ✓ | 403 | ✓ | ✓ | 404 |

(`403` for a non-owner assumes the request is visible to that user. Drafts and other orgs always return `404`.)

Plus end-to-end scenarios:

1. Reimbursement happy path with two receipts through submit.
2. Rejection → resubmit with `carry` → pending.
3. Rejection → resubmit with `new` → draft → upload → submit.
4. Withdraw a draft.

## 3.6 Documentation and cleanup

- OpenAPI/Swagger for all 11 routes, including error bodies and the multipart upload.
- Consistent structured logging with request ID, org ID, and user ID on every service call.
- Remove temporary stubs and dead code, and confirm the `501` path no longer exists.
- Update `requests_domain_design.md` if any behavior changed during implementation.
- Short README in `internal/requests/`: layers, ports, how to run the tests.

## 3.7 Definition of done

- [ ] Detail returns real approval data and the complete timeline.
- [ ] Audit events appear in the Audit domain for every lifecycle action.
- [ ] All concurrency and rollback tests pass reliably (run them repeatedly, with `-race`).
- [ ] The role matrix is covered by passing tests.
- [ ] API docs are published and the checklist in design §35 is fully ticked.

---

# Appendix — Additions to the original design

These came up while turning the design into an implementation. Each is small, but you should know about them.

| Addition | Where | Why |
|---|---|---|
| `resubmitted_at` column | Phase 1 migration | Timeline needs it |
| `NextSequence`, `WithTx`, `FindByIDForUpdate`, `CountReceipts`, `CopyReceipts`, `UpdateReceiptOCR` on the repository | Phase 1 ports | Required by submit, resubmit, upload, OCR, and the ID sequence |
| `TransitionStatus(from, to)` instead of `UpdateStatus` | Phase 1 ports | Race-safe status changes |
| `Receipt.FileKey` instead of `FileURL` | Phase 1 model | Presigned URLs expire |
| `FileService.Delete` | Phase 1 ports | Cleanup of orphaned uploads |
| `Actor` struct instead of separate user/org IDs | Phase 1 service | Role is needed almost everywhere |
| Port interfaces with no-op defaults for Audit and Approval | Phase 1 | Lets Phase 2 be complete without the other developer's code |
| A request can be resubmitted only once | Phase 2B | Prevents duplicate resubmissions |
| Pagination defaults and max limit live in the validator | Phase 1 | A list endpoint shouldn't ship without them; Phase 3 only verifies |
