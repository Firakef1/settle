# Implementation Plan — Email Verification (v2)

**Goal**: After a user registers with email, they must verify their email address with a 6-digit code before their account is usable. **All users, including existing ones, must verify.** The code is stored as a keyed hash (HMAC-SHA256), expires after a configurable TTL, and is single-use. A successful verification signs the user in directly.

---

## Architecture Decision Summary

| Concern | Decision |
|---|---|
| Verification service & endpoints | `internal/auth/` domain (service + handler) |
| Email delivery service | `internal/shared/service/email_service.go`, **generic** `SendEmail` only, pluggable transport |
| Who triggers the code email | `AuthHandler.Signup` calls `authSvc.Signup` then `verSvc.SendCode` (services stay decoupled) |
| Identifier on public endpoints | **Email** (normalized: trimmed + lowercased). No `user_id` anywhere on the wire |
| Code storage | New `email_verification_codes` table, **one row per user** (new code replaces old, consume deletes) |
| Code format | 6-digit numeric string generated with `crypto/rand` |
| Code hashing | `HMAC-SHA256(secret, userID ‖ 0x00 ‖ code)`, secret from `EMAIL_CODE_SECRET`; compared in constant time |
| Code TTL | `EMAIL_CODE_TTL`, default 15 minutes |
| Consume | Single DB transaction: lock row, compare, delete code row, set `users.email_verified = TRUE` |
| Account gate | `users.email_verified` (default `FALSE`, **no backfill**). `Login` and `Refresh` reject unverified users |
| Sending | **Synchronous**, bounded by `EMAIL_SEND_TIMEOUT`. No goroutines on the request path |
| Enumeration | `resend-verification` always returns the same 200; `verify-email` returns one generic 400 for every failure |
| Signup on an existing *unverified* email | Overwrites name + password hash and sends a fresh code (prevents email squatting) |
| After successful verification | Response includes tokens (same shape as login), so the user is signed in immediately |
| Error status codes | `400` bad/expired code, `403` unverified login/refresh, never `401` for a wrong code |
| Config | `EmailConfig` sub-struct with startup validation |

---

## 1. Config Changes

### `internal/shared/config/config.go`

```go
// EmailConfig holds email delivery and verification-code configuration.
type EmailConfig struct {
    SMTPHost     string        // SMTP_HOST
    SMTPPort     string        // SMTP_PORT (default "587")
    SMTPUsername string        // SMTP_USERNAME (optional; if set, password is required)
    SMTPPassword string        // SMTP_PASSWORD
    FromAddress  string        // EMAIL_FROM_ADDRESS
    FromName     string        // EMAIL_FROM_NAME
    SendTimeout  time.Duration // EMAIL_SEND_TIMEOUT (default 10s), dial + whole SMTP conversation
    LogOnly      bool          // EMAIL_LOG_ONLY, dev only: log emails (incl. codes) instead of sending
    CodeTTL      time.Duration // EMAIL_CODE_TTL (default 15m)
    CodeSecret   string        // EMAIL_CODE_SECRET (required, >= 32 chars): HMAC key for code hashes
}

type Config struct {
    // ... existing fields ...
    Email EmailConfig
}
```

`Load()` additions:

```go
Email: EmailConfig{
    SMTPHost:     getEnvOrDefault("SMTP_HOST", ""),
    SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
    SMTPUsername: getEnvOrDefault("SMTP_USERNAME", ""),
    SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
    FromAddress:  getEnvOrDefault("EMAIL_FROM_ADDRESS", "noreply@settle.app"),
    FromName:     getEnvOrDefault("EMAIL_FROM_NAME", "Settle"),
    SendTimeout:  parseDurationOrDefault(os.Getenv("EMAIL_SEND_TIMEOUT"), 10*time.Second),
    LogOnly:      os.Getenv("EMAIL_LOG_ONLY") == "true",
    CodeTTL:      parseDurationOrDefault(os.Getenv("EMAIL_CODE_TTL"), 15*time.Minute),
    CodeSecret:   os.Getenv("EMAIL_CODE_SECRET"),
},
```

**Startup validation**: `Load()` calls `Email.Validate()` and fails startup (same mechanism `Load` already uses for other bad config):

```go
func (c EmailConfig) Validate() error {
    if len(c.CodeSecret) < 32 {
        return errors.New("EMAIL_CODE_SECRET must be at least 32 characters")
    }
    if c.CodeTTL <= 0 || c.SendTimeout <= 0 {
        return errors.New("EMAIL_CODE_TTL and EMAIL_SEND_TIMEOUT must be positive")
    }
    if c.LogOnly {
        return nil // dev mode: no SMTP needed
    }
    if c.SMTPHost == "" || c.FromAddress == "" {
        return errors.New("SMTP_HOST and EMAIL_FROM_ADDRESS are required unless EMAIL_LOG_ONLY=true")
    }
    if c.SMTPUsername != "" && c.SMTPPassword == "" {
        return errors.New("SMTP_PASSWORD is required when SMTP_USERNAME is set")
    }
    return nil
}
```

> [!IMPORTANT]
> Because every existing user must verify, **production SMTP must be working before this ships**. The startup validation above makes a misconfigured deploy fail fast instead of locking everyone out silently.

### `.env` additions

```
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=your@email.com
SMTP_PASSWORD=yourpassword
EMAIL_FROM_ADDRESS=noreply@settle.app
EMAIL_FROM_NAME=Settle
EMAIL_SEND_TIMEOUT=10s
EMAIL_CODE_TTL=15m
# Generate with: openssl rand -hex 32   (left blank on purpose: startup fails until set)
EMAIL_CODE_SECRET=
# Local dev only: log emails to stdout instead of sending
EMAIL_LOG_ONLY=false
```

---

## 2. Database Migration

### `migrations/011_add_email_verification.up.sql`

```sql
-- Gate column. Existing users intentionally start unverified: everyone must verify.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Normalize stored emails to match the app-level normalization (trim + lowercase).
-- Pre-check for collisions before running:
--   SELECT LOWER(BTRIM(email)), COUNT(*) FROM users GROUP BY 1 HAVING COUNT(*) > 1;
-- If a unique constraint on email exists, a collision makes this statement fail loudly (desired).
UPDATE users SET email = LOWER(BTRIM(email)) WHERE email <> LOWER(BTRIM(email));

-- One active code per user. A new code replaces the old one; consuming deletes the row.
CREATE TABLE IF NOT EXISTS email_verification_codes (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Supports the periodic expired-row cleanup.
CREATE INDEX IF NOT EXISTS idx_evc_expires_at ON email_verification_codes(expires_at);
```

### `migrations/011_add_email_verification.down.sql`

```sql
DROP TABLE IF EXISTS email_verification_codes;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
-- Email normalization is not reverted.
```

---

## 3. Shared Email Service (Delivery)

### `internal/shared/service/email_service.go`  *(new)*
### `internal/shared/service/email_transport.go`  *(new)*

Responsible **only** for generic delivery. No knowledge of codes, templates, or auth.

```go
// mailTransport moves an already-built RFC 5322 message to the network (or to a log).
type mailTransport interface {
    Send(ctx context.Context, from, to string, msg []byte) error
}

type EmailService struct {
    transport mailTransport
    from      mail.Address // {Name: FromName, Address: FromAddress}
}

// NewEmailService picks smtpTransport, or logTransport when EMAIL_LOG_ONLY=true,
// based on config.AppConfig.Email.
func NewEmailService() *EmailService

// newEmailServiceWithTransport is used by in-package tests to inject a capturing transport.
func newEmailServiceWithTransport(t mailTransport, from mail.Address) *EmailService

// SendEmail sends a generic HTML email. Synchronous; honours ctx and EMAIL_SEND_TIMEOUT.
func (s *EmailService) SendEmail(ctx context.Context, toAddr, toName, subject, htmlBody string) error
```

**`SendEmail` behaviour (header-injection safe):**

1. Reject `toAddr` if it contains `\r` or `\n`, or if `mail.ParseAddress(toAddr)` fails or returns a different address than given (blocks `Name <a@b>` tricks). Return a typed error.
2. Strip `\r`/`\n` from `toName` and `subject`.
3. Build headers: `From`/`To` via `mail.Address{...}.String()` (RFC 2047 encodes non-ASCII names), `Subject` via `mime.QEncoding.Encode("utf-8", subject)`, `Date`, `Message-ID` (`<random@from-domain>`), `MIME-Version: 1.0`, `Content-Type: text/html; charset=UTF-8`, `Content-Transfer-Encoding: quoted-printable`.
4. Encode the body with `mime/quotedprintable`.
5. Call `transport.Send`; wrap errors with `%w`.

**`smtpTransport`:**

- Port `465`: implicit TLS via `tls.Dialer{NetDialer: &net.Dialer{Timeout: ...}, Config: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}.DialContext`.
- Any other port: plain `net.Dialer.DialContext`, then **STARTTLS is mandatory**: check `client.Extension("STARTTLS")`, fail if absent, then `client.StartTLS(...)`. Exception: loopback hosts (`localhost`, `127.0.0.1`, `::1`) may skip TLS so local mail catchers work.
- `net/smtp` ignores `ctx`, so enforce it manually: call `conn.SetDeadline(min(ctx.Deadline, now+SendTimeout))` right after dialing, before `smtp.NewClient`.
- Auth via `smtp.PlainAuth` only when `SMTPUsername != ""`.

**`logTransport`** (dev only): logs `to`, `subject`, and the full body (which contains the code) at INFO; logs a loud WARN once at startup that email is not being sent.

> [!NOTE]
> `EmailService` sends whatever content it is given. Verification-specific templating lives in `VerificationService`, so other domains can reuse the same service.

---

## 4. Auth Domain Changes

### 4a. Model: `internal/auth/model/user_model.go`

```go
type User struct {
    ID            string    `json:"id"`
    Email         string    `json:"email"`
    Name          string    `json:"name"`
    PasswordHash  string    `json:"-"`
    Status        string    `json:"status"`
    EmailVerified bool      `json:"email_verified"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

### 4b. Model: `internal/auth/model/email_verification_code.go`  *(new)*

```go
type EmailVerificationCode struct {
    UserID    string
    CodeHash  string
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

### 4c. Repository: `internal/auth/repository/email_verification_repo.go`  *(new)*

```go
type EmailVerificationRepository interface {
    // Upsert stores the code for the user, replacing any existing one (one active code per user).
    Upsert(ctx context.Context, code *model.EmailVerificationCode) error

    // ConsumeAndVerify atomically, in ONE transaction:
    //   1. locks the user's code row (SELECT ... FOR UPDATE)
    //   2. returns ErrNotFound if there is no row, or it is expired (expires_at <= now)
    //   3. compares stored hash to codeHash with subtle.ConstantTimeCompare;
    //      mismatch -> ErrNotFound (row is left untouched)
    //   4. deletes the code row
    //   5. UPDATE users SET email_verified = TRUE, updated_at = NOW() WHERE id = $1
    // Two concurrent calls with the right code: exactly one succeeds.
    ConsumeAndVerify(ctx context.Context, userID, codeHash string) error

    // DeleteExpired removes rows whose expires_at is in the past. Returns rows deleted.
    DeleteExpired(ctx context.Context) (int64, error)
}
```

PostgreSQL implementation uses a `database/sql` transaction (adapt to your driver). Memory fallback (same pattern as `UserRepo` / `RefreshTokenRepo`) uses a mutex; because `ConsumeAndVerify` also flips the user flag, the constructor takes the in-memory user repo:

```go
// db == nil -> memory mode, which needs memUsers. In production: NewEmailVerificationRepo(database.DB, nil).
func NewEmailVerificationRepo(db *sql.DB, memUsers *UserRepo) *EmailVerificationRepo
```

### 4d. Repository: `internal/auth/repository/user_repo.go`  *(modified)*

- `CreateUser` SQL includes `email_verified` (default `FALSE`).
- `FindByEmail` / `FindByID` scan `email_verified`.
- **New** `UpdateUnverifiedCredentials(ctx, userID, name, passwordHash string) error`: `UPDATE users SET name=$2, password_hash=$3, updated_at=NOW() WHERE id=$1 AND email_verified = FALSE`; returns `ErrNotFound` if 0 rows (already verified or gone).
- Update the `UserRepository` interface accordingly. `MarkEmailVerified` is **not** needed; the flag flips inside `ConsumeAndVerify`.

### 4e. Email normalization helper: `internal/auth/service/email_normalize.go`  *(new)*

```go
func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
```

Used by **every** service method that accepts an email (`Signup`, `Login`, `SendCode`, `VerifyCode`), so all lookups and stored values agree.

### 4f. Service: `internal/auth/service/verification_service.go`  *(new)*

```go
// Defined locally in the auth domain (structural typing); *shared.EmailService satisfies it.
type EmailSender interface {
    SendEmail(ctx context.Context, toAddr, toName, subject, htmlBody string) error
}

type VerificationService struct {
    userRepo         repository.UserRepository
    verificationRepo repository.EmailVerificationRepository
    emailSender      EmailSender
    codeTTL          time.Duration
    secret           []byte
}

func NewVerificationService(
    userRepo repository.UserRepository,
    verificationRepo repository.EmailVerificationRepository,
    emailSender EmailSender,
    codeTTL time.Duration,
    codeSecret string,
) *VerificationService

// SendCode (re)issues a code for the account with this email and emails it, synchronously.
// Unknown email or already-verified account -> returns nil and does nothing (callers can't
// distinguish, which prevents enumeration). Only infrastructure failures return an error.
func (s *VerificationService) SendCode(ctx context.Context, email string) error

// VerifyCode checks the code and, on success, marks the user verified and returns the user.
// EVERY failure (unknown email, already verified, no code, wrong code, expired, reused)
// returns ErrInvalidCode. Only infrastructure failures return another error.
func (s *VerificationService) VerifyCode(ctx context.Context, email, code string) (*model.User, error)

// RunCleanup deletes expired rows every `every` until ctx is cancelled (blocking; run in a goroutine).
func (s *VerificationService) RunCleanup(ctx context.Context, every time.Duration)
```

**Sentinel error (single):**

```go
var ErrInvalidCode = errors.New("invalid or expired verification code")
```

**`SendCode` steps:** normalize email → `FindByEmail` (not found → `nil`) → if `EmailVerified` → `nil` → generate code → `Upsert{UserID, hash, now+codeTTL}` → render email template → `emailSender.SendEmail(...)`; wrap and return any send error.

**`VerifyCode` steps:** normalize email → `FindByEmail` (not found or already verified → `ErrInvalidCode`) → `ConsumeAndVerify(user.ID, hashCode(user.ID, code))`; repo `ErrNotFound` → `ErrInvalidCode` → re-fetch user via `FindByID` (now `EmailVerified=true`) and return it.

**Code generation & hashing:**

```go
func generateCode() (string, error) {
    n, err := rand.Int(rand.Reader, big.NewInt(1_000_000)) // crypto/rand, NOT math/rand
    if err != nil { return "", err }
    return fmt.Sprintf("%06d", n.Int64()), nil            // keeps leading zeros
}

// Binding the user ID means the same code hashes differently for different users.
func (s *VerificationService) hashCode(userID, code string) string {
    mac := hmac.New(sha256.New, s.secret)
    mac.Write([]byte(userID))
    mac.Write([]byte{0})
    mac.Write([]byte(code))
    return hex.EncodeToString(mac.Sum(nil))
}
```

**Email template** (`html/template`, so the user's name is auto-escaped). Subject: `Your Settle verification code`. The code must appear exactly once, as `<strong>482019</strong>`, and the template must contain no other 6-digit sequences (tests extract it with `<strong>(\d{6})</strong>`).

> [!IMPORTANT]
> Only the **HMAC** is stored. The plain code is emailed and never persisted or logged (except by the dev-only `logTransport`).

### 4g. Service: `internal/auth/service/auth_service.go`  *(modified)*

- **`Signup(ctx, req)`**: normalize email, then `FindByEmail`:
  - not found → hash password, create user with `EmailVerified=false` (as today);
  - found and **unverified** → hash the new password and call `UpdateUnverifiedCredentials(user.ID, req.Name, hash)`, then return that user (same result shape as a new signup);
  - found and **verified** → existing duplicate-email error (unchanged).

  `AuthService` does **not** import `VerificationService`; the handler sends the code (§4i).
- **`Login`**: normalize email; after password validation, if `!user.EmailVerified` return `ErrEmailNotVerified` and **do not issue tokens**. The gate runs *after* the password check, so a wrong password never reveals verification state.
- **`Refresh`**: after loading the user for the refresh token, if `!user.EmailVerified` return `ErrEmailNotVerified`. This stops pre-existing sessions of not-yet-verified users from bypassing the gate indefinitely.
- **New `IssueSession(ctx, user *model.User)`**: the token-issuing tail of `Login` extracted into a public method returning the same result type `Login` returns. `Login` calls it, and so does the verification handler.

```go
var ErrEmailNotVerified = errors.New("email address has not been verified")
```

### 4h. DTOs: new files in `internal/auth/dto/`

**`verify_email_request.go`**
```go
type VerifyEmailRequest struct {
    Email string `json:"email" binding:"required,email"`
    Code  string `json:"code"  binding:"required,len=6,numeric"`
}
```

**`resend_code_request.go`**
```go
type ResendCodeRequest struct {
    Email string `json:"email" binding:"required,email"`
}
```

> [!NOTE]
> Public verification endpoints take `email` because the user is not authenticated. Email is normalized in the service layer, not trusted from the client.

### 4i. Handlers

**`internal/auth/handler/auth_handler.go`** *(modified)*: `NewAuthHandler(authSvc, verSvc)`

`Signup`:
1. Bind request. → `400` on invalid body.
2. `user, err := authSvc.Signup(...)`. Duplicate verified email → `409` (existing behaviour).
3. `verSvc.SendCode(ctx, user.Email)`. On error → `500 {"error": "could not send verification email, please try again"}`. Retrying signup is safe: the unverified-email path re-sends a fresh code.
4. `201 {user, message: "verification code sent to your email"}`. Identical for new and re-registered unverified emails.

`Login` / `Refresh` mapping:

| Condition | HTTP |
|---|---|
| `ErrEmailNotVerified` | `403 Forbidden`, body `{"error": "email address has not been verified", "code": "email_not_verified"}` (stable machine-readable `code` so clients route to the verify screen) |

**`internal/auth/handler/verification_handler.go`** *(new)*: `NewVerificationHandler(verSvc, authSvc)`

`POST /auth/verify-email`

| Condition | HTTP |
|---|---|
| bind/validation error | `400 Bad Request` |
| `ErrInvalidCode` (any cause) | `400 Bad Request`, `{"error": "invalid or expired verification code"}` |
| other error | `500` |
| success | `200 OK`: `authSvc.IssueSession(user)` → body = the same response DTO `Login` returns, plus `{"message": "email verified successfully"}` |

`POST /auth/resend-verification`

| Condition | HTTP |
|---|---|
| bind/validation error | `400 Bad Request` |
| send/infrastructure failure | `500 Internal Server Error` |
| success **or** unknown email **or** already verified | `200 OK`, always `{"message": "if this account exists and is unverified, a verification code has been sent"}` |

> [!NOTE]
> `401` is deliberately not used for wrong codes: many clients treat `401` as "session expired" and trigger token refresh/logout.

---

## 5. Router Changes

### `internal/router/auth_router.go`  *(modified)*

```go
func RegisterAuthRoutes(rg *gin.RouterGroup, authH *authHandler.AuthHandler, verH *authHandler.VerificationHandler) {
    auth := rg.Group("/auth")
    {
        auth.POST("/signup",              authH.Signup)
        auth.POST("/login",               authH.Login)
        auth.POST("/logout",              authH.Logout)
        auth.POST("/refresh",             authH.Refresh)
        auth.POST("/verify-email",        verH.VerifyEmail)
        auth.POST("/resend-verification", verH.ResendCode)
    }
}
```

### `internal/router/router.go`  *(modified)*

Add `*authHandler.VerificationHandler` to the `Handlers` struct and pass it to `RegisterAuthRoutes`.

---

## 6. Wiring: `cmd/settle/main.go`  *(modified)*

```go
func SetupAuthHandler(ctx context.Context) (*authHandler.AuthHandler, *authHandler.VerificationHandler) {
    cfg := config.AppConfig.Email

    hashSvc  := sharedService.NewHashService()
    jwtSvc   := sharedService.NewJWTService()
    emailSvc := sharedService.NewEmailService() // NEW

    userRepo         := authRepo.NewUserRepo(database.DB)
    refreshTokenRepo := authRepo.NewRefreshTokenRepo(database.DB)
    verificationRepo := authRepo.NewEmailVerificationRepo(database.DB, nil) // NEW

    authSvc := authService.NewAuthService(userRepo, refreshTokenRepo, hashSvc, jwtSvc)
    verSvc  := authService.NewVerificationService(userRepo, verificationRepo, emailSvc, cfg.CodeTTL, cfg.CodeSecret) // NEW

    go verSvc.RunCleanup(ctx, time.Hour) // maintenance only; NOT on the request path

    authH := authHandler.NewAuthHandler(authSvc, verSvc)          // Signup sends the code
    verH  := authHandler.NewVerificationHandler(verSvc, authSvc)  // verify -> IssueSession

    return authH, verH
}
```

> [!NOTE]
> `AuthService` does not depend on `VerificationService`. `AuthHandler.Signup` calls `authSvc.Signup(...)` then `verSvc.SendCode(...)` in sequence, and `VerificationHandler` calls `authSvc.IssueSession(...)` after a successful verify.

---

## 7. Flows

### Registration + verification (new user)
```
Client                 AuthHandler     AuthService     VerificationService     EmailService
  |-- POST /auth/signup ->|                |                   |                     |
  |                       |-- Signup ----->| (normalize email; create user, verified=false)
  |                       |<-- user -------|                   |                     |
  |                       |-- SendCode(email) --------------->|                     |
  |                       |                                    |-- code, HMAC, Upsert|
  |                       |                                    |-- SendEmail(html) ->| SMTP (sync)
  |<-- 201 {user, "verification code sent"} ------------------|                     |
  |                                                                                  |
  |-- POST /auth/verify-email {email, code} -> VerificationHandler                   |
  |                       |-- VerifyCode -->| ConsumeAndVerify (1 tx: lock, compare, delete code, set flag)
  |                       |-- IssueSession(user)                                     |
  |<-- 200 {message, user, access_token, refresh_token}                              |
```

### Signup with an existing *unverified* email
```
POST /auth/signup -> Signup finds unverified user -> UpdateUnverifiedCredentials(name, new hash)
                  -> SendCode (new code replaces old) -> 201 (same response as a new signup)
```

### Existing user (pre-migration, now unverified)
```
POST /auth/login          -> password OK, email_verified=false -> 403 {code: "email_not_verified"}
POST /auth/resend-verification {email} -> 200 (generic) + code emailed
POST /auth/verify-email   {email, code} -> 200 + tokens (signed in)
```

### Login gate
```
POST /auth/login -> FindByEmail -> password check -> if !EmailVerified: 403 (no tokens)
POST /auth/refresh -> load user -> if !EmailVerified: 403
```

---

## 8. New Files Summary

| File | Type | Description |
|---|---|---|
| `migrations/011_add_email_verification.up.sql` | Migration | `email_verified` column, email normalization, `email_verification_codes` table |
| `migrations/011_add_email_verification.down.sql` | Migration | Rollback |
| `internal/shared/service/email_service.go` | Shared service | Generic `SendEmail`, message building, header sanitizing |
| `internal/shared/service/email_transport.go` | Shared service | `mailTransport`, `smtpTransport`, `logTransport` |
| `internal/auth/model/email_verification_code.go` | Model | `{UserID, CodeHash, ExpiresAt, CreatedAt}` |
| `internal/auth/repository/email_verification_repo.go` | Repository | `Upsert`, `ConsumeAndVerify`, `DeleteExpired` (+ memory mode) |
| `internal/auth/service/verification_service.go` | Service | Generate, HMAC, store, send, verify, cleanup |
| `internal/auth/service/email_normalize.go` | Helper | `normalizeEmail` |
| `internal/auth/dto/verify_email_request.go` | DTO | `{email, code}` |
| `internal/auth/dto/resend_code_request.go` | DTO | `{email}` |
| `internal/auth/handler/verification_handler.go` | Handler | `verify-email`, `resend-verification` |

## 9. Modified Files Summary

| File | Change |
|---|---|
| `internal/shared/config/config.go` | `EmailConfig` + `Validate()`; `Email` field on `Config` |
| `.env` | SMTP, `EMAIL_CODE_TTL`, `EMAIL_SEND_TIMEOUT`, `EMAIL_CODE_SECRET`, `EMAIL_LOG_ONLY` |
| `internal/auth/model/user_model.go` | Add `EmailVerified` |
| `internal/auth/repository/user_repo.go` | Scan `email_verified`; add `UpdateUnverifiedCredentials` |
| `internal/auth/service/auth_service.go` | Normalize email; unverified-signup overwrite; Login + Refresh gate; `IssueSession`; `ErrEmailNotVerified` |
| `internal/auth/handler/auth_handler.go` | Signup calls `verSvc.SendCode`; map `ErrEmailNotVerified` → 403 (Login, Refresh) |
| `internal/router/auth_router.go` | Register two new endpoints |
| `internal/router/router.go` | Add `VerificationHandler` to `Handlers` |
| `cmd/settle/main.go` | Wire email service, repo, service, handler, cleanup loop |

---

## 10. Security Notes

> [!IMPORTANT]
> - **Codes are stored as HMAC-SHA256 with a server secret**, so a DB leak alone cannot be brute-forced offline (a plain SHA-256 of a 6-digit code can be reversed instantly). The user ID is bound into the MAC.
> - **Constant-time comparison** of the stored and submitted hashes.
> - **Single-use is enforced by the database**: the row is locked, compared, and deleted in one transaction together with the `email_verified` update. Concurrent double-submits cannot both succeed, and there is no state where a code is consumed but the user is not verified.
> - **Codes expire** after `EMAIL_CODE_TTL`; **a new code replaces the old one** (one row per user).
> - **No account enumeration** on `resend-verification` and `verify-email` (uniform responses). The login gate only triggers after a correct password.
> - **Email header injection** blocked (CR/LF rejected/stripped, addresses parsed, names/subjects RFC 2047 encoded); template output is HTML-escaped.
> - **TLS enforced** for SMTP (implicit TLS on 465, mandatory STARTTLS otherwise, loopback exempt).
> - **Pre-existing sessions cannot bypass the gate**: `Refresh` also checks `email_verified`.

> [!NOTE]
> **Known residual risk of the squatting fix.** Overwriting credentials on re-signup of an unverified email means an attacker who re-registers *your* unverified email can set the password, then you receive a code you didn't expect. If you verify it, the account is yours by email but has the attacker's password. The overwrite is still the right trade-off (the alternative lets an attacker pre-register your email and keep their password after you verify). Mitigate with clear email copy ("If you didn't just sign up, ignore this email") and, later, a password-reset flow.

---

## 11. Deferred (explicitly out of scope for this pass)

| Item | Why it matters | Suggested later approach |
|---|---|---|
| **Verify-attempt limiting** | 1,000,000 possible codes and no lockout means a code can be brute-forced within its TTL. **Because a correct code now returns tokens, this is higher stakes than before**; do it before public launch | Add `attempts` to `email_verification_codes`; increment inside the same tx; delete the code after ~5 misses. Plus a per-IP limiter on `/verify-email` |
| **Resend cooldown / hourly cap** | Prevents using the endpoint to spam inboxes | Store `created_at` (already present): reject/skip if newer than 60s; per-email + per-IP hourly cap |
| **Timing side-channel on resend/signup** | Send is synchronous, so response time can reveal whether an unverified account exists | Move sending to a queue/outbox with a fixed-time response |
| **Signup with a verified email returns 409** | Pre-existing behaviour that reveals registered emails | Separate product decision |

---

## 12. Implementation Order

1. `migrations/011_...`: DB schema first
2. `config.go`: `EmailConfig` + `Validate()`
3. `internal/shared/service/email_transport.go` + `email_service.go`
4. `internal/auth/model/email_verification_code.go`
5. `internal/auth/model/user_model.go`: `EmailVerified`
6. `internal/auth/repository/user_repo.go`: queries + `UpdateUnverifiedCredentials`
7. `internal/auth/repository/email_verification_repo.go` (PG + memory)
8. `internal/auth/service/email_normalize.go`
9. `internal/auth/service/verification_service.go`
10. `internal/auth/service/auth_service.go`: normalization, signup overwrite, Login/Refresh gate, `IssueSession`
11. `internal/auth/dto/`: two DTOs
12. `internal/auth/handler/verification_handler.go`
13. `internal/auth/handler/auth_handler.go`: signup send + 403 mapping
14. `internal/router/auth_router.go` + `router.go`
15. `cmd/settle/main.go`: wiring + cleanup loop

**Rollout checklist:** SMTP credentials and `EMAIL_CODE_SECRET` set in prod → run collision pre-check query → deploy → existing users are prompted to verify on next login.

---

## 13. Testing Plan

> [!NOTE]
> Conventions: `testify/assert` + `testify/require`, in-package tests, memory repos (`NewUserRepo(nil)`, `NewEmailVerificationRepo(nil, userRepo)`), `httptest` + `gin.TestMode` with the existing `performRequest` helper. Run everything with `-race`.

### 13a. Shared email service

**File:** `internal/shared/service/email_service_test.go` (in-package; uses `newEmailServiceWithTransport` with a capturing transport)

| Test | Asserts |
|---|---|
| `TestSendEmail_BuildsMessage` | `From`, `To`, `Subject`, `Date`, `Message-ID`, `MIME-Version`, `Content-Type: text/html; charset=UTF-8`; body decodes (quoted-printable) to the input HTML |
| `TestSendEmail_RejectsCRLFInToAddr` | `toAddr` containing `\r`/`\n` → error, transport not called |
| `TestSendEmail_RejectsNonPlainAddress` | `"Eve <a@b.com>"` as `toAddr` → error |
| `TestSendEmail_StripsCRLFInNameAndSubject` | Injected `\r\nBcc: x` cannot create a new header |
| `TestSendEmail_EncodesNonASCII` | Non-ASCII name/subject are RFC 2047 encoded |
| `TestSendEmail_TransportErrorPropagates` | `errors.Is(err, transportErr)` |
| `TestLogTransport_NoError` | Sends without error and doesn't panic |

**File:** `internal/shared/config/config_test.go` (additions)

| Test | Asserts |
|---|---|
| `TestEmailConfigValidate_SecretTooShort` | `< 32` chars → error |
| `TestEmailConfigValidate_MissingSMTPHost` | error unless `LogOnly` |
| `TestEmailConfigValidate_LogOnlyNeedsNoSMTP` | valid with secret only |
| `TestEmailConfigValidate_UsernameWithoutPassword` | error |

### 13b. VerificationService

**File:** `internal/auth/service/verification_service_test.go` (`package service`)

```go
func setupVerificationService(mock *mockEmailSender) (*VerificationService, *repository.UserRepo, *repository.EmailVerificationRepo) {
    userRepo         := repository.NewUserRepo(nil)
    verificationRepo := repository.NewEmailVerificationRepo(nil, userRepo)
    svc := NewVerificationService(userRepo, verificationRepo, mock, 15*time.Minute, testSecret /* 32+ chars */)
    return svc, userRepo, verificationRepo
}
```

#### `SendCode`

| Test | Setup | Asserts |
|---|---|---|
| `TestSendCode_Success` | Unverified user | No error; exactly one email to the user's address; stored row has non-empty hash and `ExpiresAt ≈ now+TTL` |
| `TestSendCode_UnknownEmail` | Empty repo | Returns `nil`; no email sent; nothing stored |
| `TestSendCode_AlreadyVerified` | Verified user | Returns `nil`; no email sent |
| `TestSendCode_NormalizesEmail` | User `bob@example.com`, call with `"  Bob@Example.COM "` | Email sent |
| `TestSendCode_ReplacesPreviousCode` | Send twice, capture both codes | First code → `ErrInvalidCode`; second verifies |
| `TestSendCode_HashIsHMAC` | Capture plain code + stored hash | Hash ≠ plain, ≠ raw SHA-256 of plain; equals expected HMAC (via `hashCode`) |
| `TestHashCode_BindsUserID` | Same code, two user IDs | Different hashes |
| `TestGenerateCode_Format` | 500 iterations | Always matches `^\d{6}$` (incl. leading zeros) |
| `TestSendCode_SendFailure` | `mock.err` set | Returns wrapped error |

#### `VerifyCode`

| Test | Setup | Asserts |
|---|---|---|
| `TestVerifyCode_Success` | Send → capture code | Returns user with `EmailVerified=true`; repo user updated; code row gone |
| `TestVerifyCode_WrongCode` | Send, submit `"000000"` (ensure ≠ real) | `ErrInvalidCode`; user still unverified; real code still works |
| `TestVerifyCode_Reuse` | Verify, verify again | Second → `ErrInvalidCode` |
| `TestVerifyCode_Expired` | `Upsert` row with `ExpiresAt` in the past | `ErrInvalidCode` |
| `TestVerifyCode_NoCode` | User exists, no send | `ErrInvalidCode` |
| `TestVerifyCode_UnknownEmail` | Empty repo | `ErrInvalidCode` |
| `TestVerifyCode_AlreadyVerified` | Verified user | `ErrInvalidCode` |
| `TestVerifyCode_NormalizesEmail` | Mixed-case/padded email | Succeeds |
| `TestVerifyCode_ConcurrentDoubleSubmit` | 20 goroutines, same correct code | Exactly one `nil`; all others `ErrInvalidCode` |

### 13c. AuthService

**File:** `internal/auth/service/auth_service_test.go` (additions)

| Test | Asserts |
|---|---|
| `TestSignup_NewUser_Unverified` | `EmailVerified=false`; stored email normalized |
| `TestSignup_ExistingUnverified_OverwritesCredentials` | Same user ID returned; new password works, old password doesn't (after marking verified) |
| `TestSignup_ExistingVerified_Conflict` | Duplicate-email error; credentials untouched |
| `TestLogin_UnverifiedEmail_Blocked` | Correct password → `ErrorIs(err, ErrEmailNotVerified)`, no tokens |
| `TestLogin_UnverifiedEmail_WrongPassword` | Returns the normal invalid-credentials error, **not** `ErrEmailNotVerified` |
| `TestLogin_VerifiedEmail_Succeeds` | Mark verified → tokens returned |
| `TestLogin_EmailCaseInsensitive` | `BOB@x.com` logs in as `bob@x.com` |
| `TestRefresh_UnverifiedUser_Blocked` | Valid refresh token but unverified user → `ErrEmailNotVerified` |
| `TestIssueSession_ReturnsTokens` | Tokens for a given user |

### 13d. Handlers

**File:** `internal/auth/handler/verification_handler_test.go` (`package handler`)

```go
func setupVerificationHandlerTest() (*AuthHandler, *VerificationHandler, *repository.UserRepo, *mockEmailSender) {
    userRepo         := repository.NewUserRepo(nil)
    verificationRepo := repository.NewEmailVerificationRepo(nil, userRepo)
    mock             := &mockEmailSender{}
    authSvc          := service.NewAuthService(userRepo, /* memory refresh repo, hash, jwt */)
    verSvc           := service.NewVerificationService(userRepo, verificationRepo, mock, 15*time.Minute, testSecret)
    return NewAuthHandler(authSvc, verSvc), NewVerificationHandler(verSvc, authSvc), userRepo, mock
}
```

#### `POST /auth/signup`

| Test | Setup | Expected |
|---|---|---|
| `TestHandler_Signup_SendsCode` | New email | `201`; mock got one email to the normalized address containing a 6-digit code |
| `TestHandler_Signup_ExistingUnverified` | Signup twice, different passwords | Both `201`; second sends a new code; only the second password works after verify |
| `TestHandler_Signup_ExistingVerified` | Verified user | `409`; no email sent |
| `TestHandler_Signup_SendFailure` | `mock.err` set | `500` |

#### `POST /auth/verify-email`

| Test | Body | Expected |
|---|---|---|
| `TestHandler_VerifyEmail_Success` | Correct email + code | `200`; body has tokens, `message`, `user.email_verified=true` |
| `TestHandler_VerifyEmail_WrongCode` | `"000000"` | `400` |
| `TestHandler_VerifyEmail_Expired` | Expired row | `400` |
| `TestHandler_VerifyEmail_Reuse` | Verify twice | Second `400` |
| `TestHandler_VerifyEmail_AlreadyVerified` | Verified user | `400` (same body as wrong code) |
| `TestHandler_VerifyEmail_UnknownEmail` | Unregistered email | `400`, **byte-identical body** to wrong-code case |
| `TestHandler_VerifyEmail_InvalidJSON` | Malformed | `400` |
| `TestHandler_VerifyEmail_MissingFields` | `{}` | `400` |
| `TestHandler_VerifyEmail_BadCodeFormat` | `"12345"`, `"abcdef"` | `400` |

#### `POST /auth/resend-verification`

| Test | Body | Expected |
|---|---|---|
| `TestHandler_ResendCode_Unverified` | Existing unverified email | `200`; email sent; old code no longer works |
| `TestHandler_ResendCode_UnknownEmail` | Unregistered email | `200`, **identical body**; no email sent |
| `TestHandler_ResendCode_AlreadyVerified` | Verified email | `200`, identical body; no email sent |
| `TestHandler_ResendCode_InvalidJSON` | Malformed | `400` |
| `TestHandler_ResendCode_SendFailure` | `mock.err` set | `500` |

#### `POST /auth/login` and `/auth/refresh` (added to `auth_handler_test.go`)

| Test | Setup | Expected |
|---|---|---|
| `TestHandler_Login_UnverifiedEmail` | Signup → login | `403`, body `code == "email_not_verified"` |
| `TestHandler_Login_VerifiedEmail` | Signup → verify → login | `200` + tokens |
| `TestHandler_Refresh_UnverifiedUser` | Unverified user with a refresh token | `403` |

#### End-to-end flows

| Test | Steps |
|---|---|
| `TestFlow_NewUser` | Signup → capture code → verify-email → tokens work on refresh |
| `TestFlow_ExistingUnverifiedUser` | Insert unverified user directly (simulates pre-migration user) → login `403` → resend → verify → `200` + tokens |

### 13e. EmailVerificationRepository

**File:** `internal/auth/repository/email_verification_repo_test.go` (memory mode)

| Test | Asserts |
|---|---|
| `TestEVRepo_UpsertAndConsume` | Consume with the right hash succeeds; user marked verified; row deleted |
| `TestEVRepo_UpsertReplacesExisting` | Second `Upsert` for the same user replaces the first (old hash → `ErrNotFound`) |
| `TestEVRepo_ConsumeWrongHash` | `ErrNotFound`; row and user unchanged |
| `TestEVRepo_ConsumeExpired` | `ErrNotFound` |
| `TestEVRepo_ConsumeTwice` | Second call `ErrNotFound` |
| `TestEVRepo_ConcurrentConsume` | N goroutines, exactly one succeeds |
| `TestEVRepo_DeleteExpired` | Removes only expired rows; returns the count |

**Optional integration tests** (`//go:build integration`, real Postgres): concurrent `ConsumeAndVerify` yields exactly one winner; user flag and row deletion commit together; `ON DELETE CASCADE` removes codes when a user is deleted.

### 13f. Coverage targets

| Package | Target | Key areas |
|---|---|---|
| `internal/shared/service` (email) | ≥ 80% | Message building, sanitizing, error propagation |
| `internal/shared/config` (email) | ≥ 90% | All `Validate()` branches |
| `internal/auth/service` (verification) | ≥ 90% | All `SendCode` / `VerifyCode` branches |
| `internal/auth/service` (auth, updated) | ≥ 90% | Signup paths, Login/Refresh gate |
| `internal/auth/handler` | ≥ 85% | Every status-code path |
| `internal/auth/repository` (verification) | ≥ 85% | All memory-mode methods |

```bash
go test -race -v -cover ./internal/auth/... ./internal/shared/...
```

### 13g. Mock helper

Defined locally in each test file that needs it, with no mock framework:

```go
// mockEmailSender implements EmailSender and captures calls without SMTP.
type mockEmailSender struct {
    mu       sync.Mutex
    calls    int
    lastTo   string
    lastName string
    lastSubj string
    lastBody string
    err      error // set to simulate a send failure
}

func (m *mockEmailSender) SendEmail(_ context.Context, toAddr, toName, subject, htmlBody string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.calls++
    m.lastTo, m.lastName, m.lastSubj, m.lastBody = toAddr, toName, subject, htmlBody
    return m.err
}

var codeRe = regexp.MustCompile(`<strong>(\d{6})</strong>`)

// extractCode pulls the plain 6-digit code out of the captured email body.
func (m *mockEmailSender) extractCode(t *testing.T) string {
    t.Helper()
    m.mu.Lock()
    defer m.mu.Unlock()
    match := codeRe.FindStringSubmatch(m.lastBody)
    require.Len(t, match, 2, "email body should contain exactly one <strong>code</strong>")
    return match[1]
}
```
