package database

import (
	"database/sql"
	"fmt"
	"os"
)

// schema is intentionally written in standard SQL so the same migrations
// work against PostgreSQL with minor type tweaks (INTEGER -> BIGSERIAL etc.).
//
// New columns must also be listed in addedColumns below so existing
// installations get them via ALTER TABLE; that keeps migrations idempotent
// without a separate migrations framework.
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id     INTEGER NOT NULL UNIQUE,
    username        TEXT    NOT NULL DEFAULT '',
    first_name      TEXT    NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    current_streak  INTEGER NOT NULL DEFAULT 0,
    best_streak     INTEGER NOT NULL DEFAULT 0,
    last_study_at   TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL,
    started_at       DATETIME NOT NULL,
    ended_at         DATETIME,
    duration_minutes INTEGER NOT NULL DEFAULT 0,
    is_active        INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_active
    ON sessions(user_id, is_active);

CREATE INDEX IF NOT EXISTS idx_sessions_user_started
    ON sessions(user_id, started_at);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at
    ON sessions(started_at);
`

// addedColumns lists schema changes that must be applied to pre-existing
// databases via ALTER TABLE. Each entry is added only if missing, so this is
// safe to run on every startup.
var addedColumns = []alterColumn{
	{Table: "users", Column: "current_streak", Type: "INTEGER NOT NULL DEFAULT 0"},
	{Table: "users", Column: "best_streak", Type: "INTEGER NOT NULL DEFAULT 0"},
	{Table: "users", Column: "last_study_at", Type: "TEXT"},
}

type alterColumn struct {
	Table  string
	Column string
	Type   string
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	for _, c := range addedColumns {
		if err := ensureColumn(db, c); err != nil {
			return fmt.Errorf("ensure column %s.%s: %w", c.Table, c.Column, err)
		}
	}
	return nil
}

func ensureColumn(db *sql.DB, c alterColumn) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", c.Table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == c.Column {
			return nil // already present
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.Table, c.Column, c.Type))
	return err
}

// ensureDir is split out to keep database.go free of OS-specific imports.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
