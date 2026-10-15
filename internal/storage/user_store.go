package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	// ErrDuplicateEmail is returned when attempting to register an email that is already registered.
	ErrDuplicateEmail = errors.New("user with this email already exists")

	// ErrUserNotFound is returned when the queried user cannot be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrSessionNotFound is returned when the queried session token does not exist.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when the queried session token has expired.
	ErrSessionExpired = errors.New("session has expired")
)

// UserStore manages user persistence and authenticated sessions in SQLite.
type UserStore struct {
	db *DB
}

// NewUserStore initializes a UserStore and ensures that user schema migrations are executed.
func NewUserStore(db *DB) (*UserStore, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.MigrateUserSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to run user schema migration: %w", err)
	}

	return &UserStore{db: db}, nil
}

// MigrateUserSchema applies SQLite table definitions for users and sessions.
func (db *DB) MigrateUserSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	// Ensure google_id column exists for OAuth SSO
	var hasGoogleID bool
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(users);")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if name == "google_id" {
					hasGoogleID = true
				}
			}
		}
	}
	if !hasGoogleID {
		_, _ = db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN google_id TEXT;")
	}
	_, _ = db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);")

	return nil
}

// CreateUser persists a new user record. If the email already exists, ErrDuplicateEmail is returned.
func (s *UserStore) CreateUser(ctx context.Context, u *model.User) (*model.User, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	normalizedEmail := strings.ToLower(strings.TrimSpace(u.Email))
	trimmedName := strings.TrimSpace(u.Name)

	query := `
		INSERT INTO users (email, password_hash, name, google_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query, normalizedEmail, u.PasswordHash, trimmedName, u.GoogleID, nowStr, nowStr)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "unique") || strings.Contains(errLower, "constraint") {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve last insert id: %w", err)
	}

	createdUser := &model.User{
		ID:           id,
		Email:        normalizedEmail,
		PasswordHash: u.PasswordHash,
		Name:         trimmedName,
		GoogleID:     u.GoogleID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return createdUser, nil
}

// GetUserByEmail queries a user record by email.
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	query := `
		SELECT id, email, password_hash, name, google_id, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	row := s.db.QueryRowContext(ctx, query, normalizedEmail)

	var u model.User
	var createdAtStr, updatedAtStr string
	var gid sql.NullString

	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &gid, &createdAtStr, &updatedAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	if gid.Valid {
		u.GoogleID = &gid.String
	}
	u.CreatedAt = parseTimeOrNow(createdAtStr)
	u.UpdatedAt = parseTimeOrNow(updatedAtStr)

	return &u, nil
}

// GetUserByID queries a user record by its primary key.
func (s *UserStore) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, google_id, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)

	var u model.User
	var createdAtStr, updatedAtStr string
	var gid sql.NullString

	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &gid, &createdAtStr, &updatedAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	if gid.Valid {
		u.GoogleID = &gid.String
	}
	u.CreatedAt = parseTimeOrNow(createdAtStr)
	u.UpdatedAt = parseTimeOrNow(updatedAtStr)

	return &u, nil
}

// GetUserByGoogleID queries a user record by Google subject identifier.
func (s *UserStore) GetUserByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	if googleID == "" {
		return nil, ErrUserNotFound
	}

	query := `
		SELECT id, email, password_hash, name, google_id, created_at, updated_at
		FROM users
		WHERE google_id = ?
	`

	row := s.db.QueryRowContext(ctx, query, googleID)

	var u model.User
	var createdAtStr, updatedAtStr string
	var gid sql.NullString

	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &gid, &createdAtStr, &updatedAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by google id: %w", err)
	}

	if gid.Valid {
		u.GoogleID = &gid.String
	}
	u.CreatedAt = parseTimeOrNow(createdAtStr)
	u.UpdatedAt = parseTimeOrNow(updatedAtStr)

	return &u, nil
}

// LinkGoogleAccount binds an existing user record to a Google subject identifier.
func (s *UserStore) LinkGoogleAccount(ctx context.Context, userID int64, googleID string) error {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	query := `
		UPDATE users
		SET google_id = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := s.db.ExecContext(ctx, query, googleID, nowStr, userID)
	if err != nil {
		return fmt.Errorf("failed to link google account: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CreateGoogleUser provisions a new user account authenticated via Google OAuth.
func (s *UserStore) CreateGoogleUser(ctx context.Context, email, name, googleID string) (*model.User, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = strings.Split(normalizedEmail, "@")[0]
	}

	query := `
		INSERT INTO users (email, password_hash, name, google_id, created_at, updated_at)
		VALUES (?, '', ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query, normalizedEmail, trimmedName, googleID, nowStr, nowStr)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "unique") || strings.Contains(errLower, "constraint") {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to insert google user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve last insert id: %w", err)
	}

	gid := googleID
	return &model.User{
		ID:        id,
		Email:     normalizedEmail,
		Name:      trimmedName,
		GoogleID:  &gid,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CreateSession persists an authenticated user session token with an expiration timestamp.
func (s *UserStore) CreateSession(ctx context.Context, session *model.Session) error {
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}

	query := `
		INSERT INTO sessions (token, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		session.Token,
		session.UserID,
		session.CreatedAt.UTC().Format(time.RFC3339),
		session.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByToken retrieves a session by token, validating that it has not expired.
func (s *UserStore) GetSessionByToken(ctx context.Context, token string) (*model.Session, error) {
	query := `
		SELECT token, user_id, created_at, expires_at
		FROM sessions
		WHERE token = ?
	`

	row := s.db.QueryRowContext(ctx, query, token)

	var session model.Session
	var createdAtStr, expiresAtStr string

	if err := row.Scan(&session.Token, &session.UserID, &createdAtStr, &expiresAtStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	session.CreatedAt = parseTimeOrNow(createdAtStr)
	session.ExpiresAt = parseTimeOrNow(expiresAtStr)

	// Verify expiration
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// DeleteSession removes a session by token (used for user logout).
func (s *UserStore) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = ?`
	_, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all sessions whose expires_at timestamp is prior to now.
func (s *UserStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	query := `DELETE FROM sessions WHERE expires_at < ?`

	result, err := s.db.ExecContext(ctx, query, nowStr)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rows, nil
}

func parseTimeOrNow(str string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, str); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}
