-- Gate column. Existing users intentionally start unverified: everyone must verify.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Normalize stored emails to match the app-level normalization (trim + lowercase).
UPDATE users SET email = LOWER(BTRIM(email)) WHERE email <> LOWER(BTRIM(email));

-- Verification codes table stored by email and code.
CREATE TABLE IF NOT EXISTS verification_codes (
    id         VARCHAR(36) PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    code       VARCHAR(6) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indices for fast lookups by email and periodic cleanup.
CREATE INDEX IF NOT EXISTS idx_verification_codes_email ON verification_codes(email);
CREATE INDEX IF NOT EXISTS idx_verification_codes_expires_at ON verification_codes(expires_at);
