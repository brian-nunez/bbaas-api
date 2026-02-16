package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type UserRecord struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SessionRecord struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type ApplicationRecord struct {
	ID          string
	OwnerUserID string
	Name        string
	Description string
	GitHubLink  string
	Domain      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type APIKeyRecord struct {
	ID            string
	ApplicationID string
	Name          string
	KeyPrefix     string
	KeyHash       string
	CanRead       bool
	CanWrite      bool
	CanDelete     bool
	CreatedAt     time.Time
	LastUsedAt    *time.Time
	RevokedAt     *time.Time
}

type APIKeyAuthRecord struct {
	Key         APIKeyRecord
	Application ApplicationRecord
}

type BrowserSessionRecord struct {
	ID                string
	ApplicationID     string
	ExternalBrowserID string
	Status            string
	CDPURL            string
	CDPHTTPURL        string
	Headless          bool
	SpawnTaskProcess  string
	SpawnedByWorkerID *int
	CreatedAt         time.Time
	LastActiveAt      time.Time
	IdleTimeout       int
	ExpiresAt         time.Time
	ClosedAt          *time.Time
}

func (s *Store) CreateUser(ctx context.Context, record UserRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		record.ID,
		record.Email,
		record.PasswordHash,
		record.Role,
		record.CreatedAt,
		record.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (UserRecord, bool, error) {
	var record UserRecord
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(
		&record.ID,
		&record.Email,
		&record.PasswordHash,
		&record.Role,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, false, nil
		}

		return UserRecord{}, false, fmt.Errorf("query user by email: %w", err)
	}

	return record, true, nil
}

func (s *Store) GetUserByID(ctx context.Context, userID string) (UserRecord, bool, error) {
	var record UserRecord
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(
		&record.ID,
		&record.Email,
		&record.PasswordHash,
		&record.Role,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, false, nil
		}

		return UserRecord{}, false, fmt.Errorf("query user by id: %w", err)
	}

	return record, true, nil
}

func (s *Store) ListUsers(ctx context.Context, limit int) ([]UserRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at
		 FROM users
		 ORDER BY created_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]UserRecord, 0, limit)
	for rows.Next() {
		var user UserRecord
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan listed user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listed users: %w", err)
	}

	return users, nil
}

func (s *Store) CreateSession(ctx context.Context, record SessionRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		record.ID,
		record.UserID,
		record.TokenHash,
		record.ExpiresAt,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

func (s *Store) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}

func (s *Store) GetSessionWithUserByTokenHash(ctx context.Context, tokenHash string) (SessionRecord, UserRecord, bool, error) {
	var session SessionRecord
	var user UserRecord
	query := `SELECT
		s.id, s.user_id, s.token_hash, s.expires_at, s.created_at,
		u.id, u.email, u.password_hash, u.role, u.created_at, u.updated_at
	FROM sessions s
	INNER JOIN users u ON u.id = s.user_id
	WHERE s.token_hash = $1`
	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionRecord{}, UserRecord{}, false, nil
		}

		return SessionRecord{}, UserRecord{}, false, fmt.Errorf("query session by token hash: %w", err)
	}

	return session, user, true, nil
}

func (s *Store) CreateApplication(ctx context.Context, record ApplicationRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO applications (id, owner_user_id, name, description, github_link, domain, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		record.ID,
		record.OwnerUserID,
		record.Name,
		record.Description,
		record.GitHubLink,
		record.Domain,
		record.CreatedAt,
		record.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}

	return nil
}

func (s *Store) ListApplicationsByUserID(ctx context.Context, userID string) ([]ApplicationRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, owner_user_id, name, description, github_link, domain, created_at, updated_at
		 FROM applications
		 WHERE owner_user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	applications := make([]ApplicationRecord, 0)
	for rows.Next() {
		var application ApplicationRecord
		if err := rows.Scan(
			&application.ID,
			&application.OwnerUserID,
			&application.Name,
			&application.Description,
			&application.GitHubLink,
			&application.Domain,
			&application.CreatedAt,
			&application.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}

		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applications: %w", err)
	}

	return applications, nil
}

func (s *Store) GetApplicationByID(ctx context.Context, applicationID string) (ApplicationRecord, bool, error) {
	var record ApplicationRecord
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, owner_user_id, name, description, github_link, domain, created_at, updated_at
		 FROM applications
		 WHERE id = $1`,
		applicationID,
	).Scan(
		&record.ID,
		&record.OwnerUserID,
		&record.Name,
		&record.Description,
		&record.GitHubLink,
		&record.Domain,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplicationRecord{}, false, nil
		}

		return ApplicationRecord{}, false, fmt.Errorf("query application by id: %w", err)
	}

	return record, true, nil
}

func (s *Store) CreateAPIKey(ctx context.Context, record APIKeyRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO api_keys (
			id, application_id, name, key_prefix, key_hash, can_read, can_write, can_delete, created_at, last_used_at, revoked_at
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		record.ID,
		record.ApplicationID,
		record.Name,
		record.KeyPrefix,
		record.KeyHash,
		boolToInt(record.CanRead),
		boolToInt(record.CanWrite),
		boolToInt(record.CanDelete),
		record.CreatedAt,
		record.LastUsedAt,
		record.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("insert API key: %w", err)
	}

	return nil
}

func (s *Store) ListAPIKeysByApplicationID(ctx context.Context, applicationID string) ([]APIKeyRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, application_id, name, key_prefix, key_hash, can_read, can_write, can_delete, created_at, last_used_at, revoked_at
		 FROM api_keys
		 WHERE application_id = $1
		 ORDER BY created_at DESC`,
		applicationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list API keys: %w", err)
	}
	defer rows.Close()

	keys := make([]APIKeyRecord, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate API keys: %w", err)
	}

	return keys, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, applicationID string, keyID string, revokedAt time.Time) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE api_keys
		 SET revoked_at = $1
		 WHERE id = $2 AND application_id = $3 AND revoked_at IS NULL`,
		revokedAt,
		keyID,
		applicationID,
	)
	if err != nil {
		return false, fmt.Errorf("revoke API key: %w", err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get API key revoke affected rows: %w", err)
	}

	return affectedRows > 0, nil
}

func (s *Store) TouchAPIKeyLastUsed(ctx context.Context, keyID string, usedAt time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE api_keys
		 SET last_used_at = $1
		 WHERE id = $2`,
		usedAt,
		keyID,
	)
	if err != nil {
		return fmt.Errorf("update API key last_used_at: %w", err)
	}

	return nil
}

func (s *Store) GetActiveAPIKeyAuthByHash(ctx context.Context, keyHash string) (APIKeyAuthRecord, bool, error) {
	query := `SELECT
		k.id, k.application_id, k.name, k.key_prefix, k.key_hash, k.can_read, k.can_write, k.can_delete, k.created_at, k.last_used_at, k.revoked_at,
		a.id, a.owner_user_id, a.name, a.description, a.github_link, a.domain, a.created_at, a.updated_at
	FROM api_keys k
	INNER JOIN applications a ON a.id = k.application_id
	WHERE k.key_hash = $1 AND k.revoked_at IS NULL`

	row := s.db.QueryRowContext(ctx, query, keyHash)

	var keyRecord APIKeyRecord
	var appRecord ApplicationRecord
	var canRead int
	var canWrite int
	var canDelete int
	var keyLastUsedAt sql.NullTime
	var keyRevokedAt sql.NullTime
	err := row.Scan(
		&keyRecord.ID,
		&keyRecord.ApplicationID,
		&keyRecord.Name,
		&keyRecord.KeyPrefix,
		&keyRecord.KeyHash,
		&canRead,
		&canWrite,
		&canDelete,
		&keyRecord.CreatedAt,
		&keyLastUsedAt,
		&keyRevokedAt,
		&appRecord.ID,
		&appRecord.OwnerUserID,
		&appRecord.Name,
		&appRecord.Description,
		&appRecord.GitHubLink,
		&appRecord.Domain,
		&appRecord.CreatedAt,
		&appRecord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return APIKeyAuthRecord{}, false, nil
		}

		return APIKeyAuthRecord{}, false, fmt.Errorf("query API key by hash: %w", err)
	}

	keyRecord.CanRead = canRead != 0
	keyRecord.CanWrite = canWrite != 0
	keyRecord.CanDelete = canDelete != 0
	keyRecord.LastUsedAt = nullableTimePtr(keyLastUsedAt)
	keyRecord.RevokedAt = nullableTimePtr(keyRevokedAt)

	return APIKeyAuthRecord{Key: keyRecord, Application: appRecord}, true, nil
}

func (s *Store) CreateBrowserSession(ctx context.Context, record BrowserSessionRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO browser_sessions (
			id, application_id, external_browser_id, status, cdp_url, cdp_http_url, headless, spawn_task_process_id, spawned_by_worker_id,
			created_at, last_active_at, idle_timeout_seconds, expires_at, closed_at
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		record.ID,
		record.ApplicationID,
		record.ExternalBrowserID,
		record.Status,
		record.CDPURL,
		record.CDPHTTPURL,
		boolToInt(record.Headless),
		nullableString(record.SpawnTaskProcess),
		nullableInt(record.SpawnedByWorkerID),
		record.CreatedAt,
		record.LastActiveAt,
		record.IdleTimeout,
		record.ExpiresAt,
		record.ClosedAt,
	)
	if err != nil {
		return fmt.Errorf("insert browser session: %w", err)
	}

	return nil
}

func (s *Store) UpdateBrowserSessionHeartbeat(ctx context.Context, applicationID string, externalBrowserID string, updated BrowserSessionRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE browser_sessions
		 SET status = 'RUNNING',
			 last_active_at = $1,
			 expires_at = $2,
			 cdp_url = $3,
			 cdp_http_url = $4,
			 idle_timeout_seconds = $5,
			 headless = $6
		 WHERE application_id = $7 AND external_browser_id = $8`,
		updated.LastActiveAt,
		updated.ExpiresAt,
		updated.CDPURL,
		updated.CDPHTTPURL,
		updated.IdleTimeout,
		boolToInt(updated.Headless),
		applicationID,
		externalBrowserID,
	)
	if err != nil {
		return fmt.Errorf("update browser session heartbeat: %w", err)
	}

	return nil
}

func (s *Store) MarkBrowserSessionCompleted(ctx context.Context, applicationID string, externalBrowserID string, closedAt time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE browser_sessions
		 SET status = 'COMPLETED',
			 closed_at = $1
		 WHERE application_id = $2 AND external_browser_id = $3`,
		closedAt,
		applicationID,
		externalBrowserID,
	)
	if err != nil {
		return fmt.Errorf("mark browser session completed: %w", err)
	}

	return nil
}

func (s *Store) ListBrowserSessionsByApplicationID(ctx context.Context, applicationID string) ([]BrowserSessionRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id, application_id, external_browser_id, status, cdp_url, cdp_http_url, headless,
			spawn_task_process_id, spawned_by_worker_id, created_at, last_active_at, idle_timeout_seconds, expires_at, closed_at
		 FROM browser_sessions
		 WHERE application_id = $1
		 ORDER BY created_at DESC`,
		applicationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list browser sessions by application: %w", err)
	}
	defer rows.Close()

	records := make([]BrowserSessionRecord, 0)
	for rows.Next() {
		record, err := scanBrowserSession(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate browser sessions by application: %w", err)
	}

	return records, nil
}

func (s *Store) ListBrowserSessionsByUserID(ctx context.Context, userID string, limit int) ([]BrowserSessionRecord, error) {
	if limit <= 0 {
		limit = 250
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			b.id, b.application_id, b.external_browser_id, b.status, b.cdp_url, b.cdp_http_url, b.headless,
			b.spawn_task_process_id, b.spawned_by_worker_id, b.created_at, b.last_active_at, b.idle_timeout_seconds, b.expires_at, b.closed_at
		 FROM browser_sessions b
		 INNER JOIN applications a ON a.id = b.application_id
		 WHERE a.owner_user_id = $1
		 ORDER BY b.created_at DESC
		 LIMIT $2`,
		userID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list browser sessions: %w", err)
	}
	defer rows.Close()

	records := make([]BrowserSessionRecord, 0, limit)
	for rows.Next() {
		record, err := scanBrowserSession(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate browser sessions: %w", err)
	}

	return records, nil
}

func (s *Store) GetBrowserSessionByExternalID(ctx context.Context, applicationID string, externalBrowserID string) (BrowserSessionRecord, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT
			id, application_id, external_browser_id, status, cdp_url, cdp_http_url, headless,
			spawn_task_process_id, spawned_by_worker_id, created_at, last_active_at, idle_timeout_seconds, expires_at, closed_at
		 FROM browser_sessions
		 WHERE application_id = $1 AND external_browser_id = $2`,
		applicationID,
		externalBrowserID,
	)

	record, err := scanBrowserSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BrowserSessionRecord{}, false, nil
		}

		return BrowserSessionRecord{}, false, fmt.Errorf("query browser session by external id: %w", err)
	}

	return record, true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(scanTarget scanner) (APIKeyRecord, error) {
	var key APIKeyRecord
	var canRead int
	var canWrite int
	var canDelete int
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime

	err := scanTarget.Scan(
		&key.ID,
		&key.ApplicationID,
		&key.Name,
		&key.KeyPrefix,
		&key.KeyHash,
		&canRead,
		&canWrite,
		&canDelete,
		&key.CreatedAt,
		&lastUsedAt,
		&revokedAt,
	)
	if err != nil {
		return APIKeyRecord{}, fmt.Errorf("scan API key: %w", err)
	}

	key.CanRead = canRead != 0
	key.CanWrite = canWrite != 0
	key.CanDelete = canDelete != 0
	key.LastUsedAt = nullableTimePtr(lastUsedAt)
	key.RevokedAt = nullableTimePtr(revokedAt)

	return key, nil
}

func scanBrowserSession(scanTarget scanner) (BrowserSessionRecord, error) {
	var record BrowserSessionRecord
	var headless int
	var spawnTaskProcess sql.NullString
	var spawnedByWorkerID sql.NullInt64
	var closedAt sql.NullTime

	err := scanTarget.Scan(
		&record.ID,
		&record.ApplicationID,
		&record.ExternalBrowserID,
		&record.Status,
		&record.CDPURL,
		&record.CDPHTTPURL,
		&headless,
		&spawnTaskProcess,
		&spawnedByWorkerID,
		&record.CreatedAt,
		&record.LastActiveAt,
		&record.IdleTimeout,
		&record.ExpiresAt,
		&closedAt,
	)
	if err != nil {
		return BrowserSessionRecord{}, err
	}

	record.Headless = headless != 0
	if spawnTaskProcess.Valid {
		record.SpawnTaskProcess = spawnTaskProcess.String
	}
	if spawnedByWorkerID.Valid {
		workerID := int(spawnedByWorkerID.Int64)
		record.SpawnedByWorkerID = &workerID
	}
	record.ClosedAt = nullableTimePtr(closedAt)

	return record, nil
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	timeValue := value.Time
	return &timeValue
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
