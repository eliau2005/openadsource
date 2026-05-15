-- 0003_users.up.sql
-- Adds the users table that the dashboard's local email/password auth reads
-- and writes against. Roles are intentionally limited to 'admin' for v1; the
-- CHECK constraint makes Phase 5 RBAC additions explicit. Email uniqueness
-- is enforced case-insensitively via the LOWER() functional index.

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'admin'
                      CHECK (role IN ('admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_email_unique ON users (LOWER(email));
