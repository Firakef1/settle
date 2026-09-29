DROP TABLE IF EXISTS verification_codes;
DROP TABLE IF EXISTS email_verification_codes;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
