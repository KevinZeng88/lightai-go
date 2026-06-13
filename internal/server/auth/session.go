package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"lightai-go/internal/server/db"
)

// SessionConfig holds session management configuration.
type SessionConfig struct {
	CookieName         string
	IdleTimeoutHours   int
	RefreshWindowHours int
	Secure             bool
}

// DefaultSessionConfig returns a safe default.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		CookieName:         "lightai_session",
		IdleTimeoutHours:   12,
		RefreshWindowHours: 6,
		Secure:             false,
	}
}

// SessionStore manages server-side sessions.
type SessionStore struct {
	db  *db.DB
	cfg SessionConfig
}

// NewSessionStore creates a new session store.
func NewSessionStore(database *db.DB, cfg SessionConfig) *SessionStore {
	return &SessionStore{db: database, cfg: cfg}
}

// CreateSession creates a new session for a user in a tenant.
func (s *SessionStore) CreateSession(userID, tenantID string) (sessionID, csrfSecret string, err error) {
	sessionID, err = generateRandomHex(32)
	if err != nil {
		return "", "", err
	}

	csrfSecret, err = generateRandomHex(32)
	if err != nil {
		return "", "", err
	}

	csrfSecretHash := hashString(csrfSecret)
	sessionIDHash := hashString(sessionID)

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.cfg.IdleTimeoutHours) * time.Hour)

	_, err = s.db.Exec(
		`INSERT INTO sessions (id, user_id, current_tenant_id, csrf_secret_hash, created_at, last_seen_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionIDHash, userID, tenantID, csrfSecretHash,
		now.Format(time.RFC3339), now.Format(time.RFC3339), expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}

	return sessionID, csrfSecret, nil
}

// ValidateSession validates a session by its plain session ID.
// Returns nil if the session is invalid, expired, or revoked.
func (s *SessionStore) ValidateSession(sessionID string) (*SessionInfo, error) {
	if sessionID == "" {
		return nil, nil
	}

	sessionIDHash := hashString(sessionID)

	row := s.db.QueryRow(
		`SELECT s.user_id, s.current_tenant_id, s.csrf_secret_hash,
		        s.created_at, s.last_seen_at, s.expires_at, s.revoked_at,
		        u.is_platform_admin, u.status as user_status,
		        t.status as tenant_status
		 FROM sessions s
		 JOIN users u ON s.user_id = u.id
		 JOIN tenants t ON s.current_tenant_id = t.id
		 WHERE s.id = ?`,
		sessionIDHash,
	)

	var info SessionInfo
	var createdAt, lastSeenAt, expiresAt string
	var revokedAt sql.NullString
	var isPlatformAdmin int

	// Store the PLAIN session ID so RevokeSession can hash it correctly.
	info.SessionID = sessionID

	err := row.Scan(
		&info.UserID, &info.TenantID, &info.CSRFSecretHash,
		&createdAt, &lastSeenAt, &expiresAt, &revokedAt,
		&isPlatformAdmin, &info.UserStatus, &info.TenantStatus,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	info.IsPlatformAdmin = isPlatformAdmin == 1

	// Parse timestamps.
	info.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	info.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)
	info.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	if revokedAt.Valid {
		t, _ := time.Parse(time.RFC3339, revokedAt.String)
		info.RevokedAt = &t
	}

	// Check validity.
	if info.RevokedAt != nil {
		return nil, nil
	}
	if time.Now().After(info.ExpiresAt) {
		return nil, nil
	}
	if info.UserStatus != "active" || info.TenantStatus != "active" {
		return nil, nil
	}

	// Sliding expiration refresh.
	if time.Since(info.LastSeenAt) > time.Duration(s.cfg.RefreshWindowHours)*time.Hour {
		newExpiresAt := time.Now().Add(time.Duration(s.cfg.IdleTimeoutHours) * time.Hour)
		_, err = s.db.Exec(
			`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`,
			time.Now().Format(time.RFC3339), newExpiresAt.Format(time.RFC3339), sessionIDHash,
		)
		if err != nil {
			return nil, fmt.Errorf("refresh session: %w", err)
		}
	}

	return &info, nil
}

// RevokeSession revokes a session by its plain session ID.
func (s *SessionStore) RevokeSession(sessionID string) error {
	sessionIDHash := hashString(sessionID)
	_, err := s.db.Exec(
		`UPDATE sessions SET revoked_at = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), sessionIDHash,
	)
	return err
}

// RevokeUserSessions revokes all sessions for a user except the given session ID.
func (s *SessionStore) RevokeUserSessions(userID string, exceptSessionID string) error {
	exceptHash := hashString(exceptSessionID)
	_, err := s.db.Exec(
		`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND id != ? AND revoked_at IS NULL`,
		time.Now().Format(time.RFC3339), userID, exceptHash,
	)
	return err
}

// SessionInfo holds the resolved session context.
type SessionInfo struct {
	SessionID       string     `json:"-"`
	UserID          string     `json:"user_id"`
	TenantID        string     `json:"tenant_id"`
	IsPlatformAdmin bool       `json:"is_platform_admin"`
	UserStatus      string     `json:"-"`
	TenantStatus    string     `json:"-"`
	CSRFSecretHash  string     `json:"-"`
	CreatedAt       time.Time  `json:"-"`
	LastSeenAt      time.Time  `json:"-"`
	ExpiresAt       time.Time  `json:"-"`
	RevokedAt       *time.Time `json:"-"`
}

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, sessionID string, cfg SessionConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		Expires:  time.Now().Add(time.Duration(cfg.IdleTimeoutHours) * time.Hour),
	})
}

// ClearSessionCookie clears the session cookie.
func ClearSessionCookie(w http.ResponseWriter, cfg SessionConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		MaxAge:   -1,
	})
}

func generateRandomHex(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
