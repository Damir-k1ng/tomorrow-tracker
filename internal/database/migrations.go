package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is the full PostgreSQL schema, applied on every startup. Every
// statement is idempotent (IF NOT EXISTS), so running it repeatedly is safe
// and no separate migration framework is needed for an MVP of this size.
//
// Types are PostgreSQL-native: BIGSERIAL identity columns, TIMESTAMPTZ for all
// timestamps (so streak/leaderboard math is timezone-correct), and a real
// BOOLEAN for is_active.
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id             BIGSERIAL PRIMARY KEY,
    telegram_id    BIGINT      NOT NULL UNIQUE,
    username       TEXT        NOT NULL DEFAULT '',
    first_name     TEXT        NOT NULL DEFAULT '',
    role           TEXT        NOT NULL DEFAULT 'user',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_streak INTEGER     NOT NULL DEFAULT 0,
    best_streak    INTEGER     NOT NULL DEFAULT 0,
    last_study_at  TIMESTAMPTZ
);

-- Bring pre-existing databases up to date. ADD COLUMN IF NOT EXISTS makes this
-- a no-op once applied, so it is safe to run on every startup.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

CREATE TABLE IF NOT EXISTS sessions (
    id               BIGSERIAL   PRIMARY KEY,
    user_id          BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at       TIMESTAMPTZ NOT NULL,
    ended_at         TIMESTAMPTZ,
    duration_minutes INTEGER     NOT NULL DEFAULT 0,
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_active
    ON sessions(user_id, is_active);

CREATE INDEX IF NOT EXISTS idx_sessions_user_started
    ON sessions(user_id, started_at);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at
    ON sessions(started_at);

-- Hardening pass: admin session corrections may invalidate a session and may
-- record anti-cheat evidence. is_valid gates whether a finished session counts
-- toward a user's streak; anti_cheat_flags is a JSONB array of evidence flags
-- drawn from a fixed application-level whitelist. anti_cheat_flags is NOT NULL
-- DEFAULT '[]' so Go/JSON/export behavior stays deterministic with no nil/null
-- edge cases. Both ADD COLUMN IF NOT EXISTS, so re-running is a no-op.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS is_valid BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS anti_cheat_flags JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Audit-log foundation (Phase 1): schema only, no business logic yet.
-- before_data / after_data are JSONB so future admin actions can record
-- arbitrary state snapshots without further migrations.
CREATE TABLE IF NOT EXISTS admin_actions (
    id             BIGSERIAL   PRIMARY KEY,
    admin_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action         TEXT        NOT NULL,
    target_user_id BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    before_data    JSONB,
    after_data     JSONB,
    reason         TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_actions_admin
    ON admin_actions(admin_id, created_at DESC);

-- Phase 2: audit records gain explicit entity references so a correction can
-- be traced to the exact session/user it touched. Idempotent for existing DBs.
ALTER TABLE admin_actions ADD COLUMN IF NOT EXISTS entity_type TEXT;
ALTER TABLE admin_actions ADD COLUMN IF NOT EXISTS entity_id   BIGINT;

-- Hardening pass: constrain entity_type to a fixed taxonomy so the audit log
-- can never accumulate inconsistent spellings (Session/session/sessoin/...).
-- entity_id stays nullable for global/system actions (e.g. exports). Postgres
-- has no ADD CONSTRAINT IF NOT EXISTS, so the DO block makes this idempotent.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'admin_actions_entity_type_check'
    ) THEN
        ALTER TABLE admin_actions
            ADD CONSTRAINT admin_actions_entity_type_check
            CHECK (entity_type IN ('session', 'user', 'export'));
    END IF;
END $$;

-- Hardening pass: enforce audit-log immutability at the database level. The
-- application only ever INSERTs into admin_actions, but a DB-level guarantee
-- means no UPDATE/DELETE/TRUNCATE can rewrite history even by mistake or by a
-- direct psql session. CREATE OR REPLACE + DROP TRIGGER IF EXISTS keep this
-- idempotent across restarts.
CREATE OR REPLACE FUNCTION reject_admin_actions_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'admin_actions is append-only: % is not permitted', TG_OP
        USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_admin_actions_no_write ON admin_actions;
CREATE TRIGGER trg_admin_actions_no_write
    BEFORE UPDATE OR DELETE ON admin_actions
    FOR EACH ROW EXECUTE FUNCTION reject_admin_actions_mutation();

DROP TRIGGER IF EXISTS trg_admin_actions_no_truncate ON admin_actions;
CREATE TRIGGER trg_admin_actions_no_truncate
    BEFORE TRUNCATE ON admin_actions
    FOR EACH STATEMENT EXECUTE FUNCTION reject_admin_actions_mutation();
`

// migrate applies the schema. It runs as a single batched Exec inside one
// implicit transaction per statement; all statements are IF NOT EXISTS so
// re-running on an existing database is a no-op.
func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}
