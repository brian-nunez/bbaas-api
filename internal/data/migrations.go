package data

import (
	"context"
	"database/sql"
	"fmt"
)

var schemaMigrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS applications (
		id TEXT PRIMARY KEY,
		owner_user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		github_link TEXT NOT NULL,
		domain TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		application_id TEXT NOT NULL,
		name TEXT NOT NULL,
		key_prefix TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		can_read INTEGER NOT NULL,
		can_write INTEGER NOT NULL,
		can_delete INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL,
		last_used_at TIMESTAMP,
		revoked_at TIMESTAMP,
		FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS browser_sessions (
		id TEXT PRIMARY KEY,
		application_id TEXT NOT NULL,
		external_browser_id TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL,
		cdp_url TEXT NOT NULL,
		cdp_http_url TEXT NOT NULL,
		headless INTEGER NOT NULL,
		spawn_task_process_id TEXT,
		spawned_by_worker_id INTEGER,
		created_at TIMESTAMP NOT NULL,
		last_active_at TIMESTAMP NOT NULL,
		idle_timeout_seconds INTEGER NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		closed_at TIMESTAMP,
		FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
		CHECK(status IN ('RUNNING', 'COMPLETED'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash)`,
	`CREATE INDEX IF NOT EXISTS idx_applications_owner_user_id ON applications(owner_user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_api_keys_application_id ON api_keys(application_id)`,
	`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash)`,
	`CREATE INDEX IF NOT EXISTS idx_browser_sessions_application_id ON browser_sessions(application_id)`,
	`CREATE INDEX IF NOT EXISTS idx_browser_sessions_status ON browser_sessions(status)`,
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	for migrationIndex, statement := range schemaMigrations {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("execute migration %d: %w", migrationIndex+1, err)
		}
	}

	return nil
}
