// Package db provides SQLite database initialization and access.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at the given path.
func Open(dbPath string) (*DB, error) {
	// Ensure the directory exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode and foreign keys.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return &DB{conn}, nil
}

// Migrate creates all required tables if they don't exist.
func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		is_platform_admin INTEGER NOT NULL DEFAULT 0,
		must_change_password INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS tenant_memberships (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL REFERENCES tenants(id),
		user_id TEXT NOT NULL REFERENCES users(id),
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(tenant_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY,
		tenant_id TEXT,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		built_in INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(tenant_id, name)
	);

	CREATE TABLE IF NOT EXISTS permissions (
		id TEXT PRIMARY KEY,
		code TEXT NOT NULL UNIQUE,
		scope TEXT NOT NULL DEFAULT 'tenant',
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		id TEXT PRIMARY KEY,
		role_id TEXT NOT NULL REFERENCES roles(id),
		permission_id TEXT NOT NULL REFERENCES permissions(id),
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(role_id, permission_id)
	);

	CREATE TABLE IF NOT EXISTS tenant_membership_roles (
		id TEXT PRIMARY KEY,
		membership_id TEXT NOT NULL REFERENCES tenant_memberships(id),
		role_id TEXT NOT NULL REFERENCES roles(id),
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(membership_id, role_id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		current_tenant_id TEXT NOT NULL REFERENCES tenants(id),
		csrf_secret_hash TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		last_seen_at TEXT NOT NULL DEFAULT (datetime('now')),
		expires_at TEXT NOT NULL,
		revoked_at TEXT
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL UNIQUE,
		hostname TEXT NOT NULL DEFAULT '',
		advertised_address TEXT NOT NULL DEFAULT '',
		metrics_enabled INTEGER NOT NULL DEFAULT 1,
		metrics_scheme TEXT NOT NULL DEFAULT 'http',
		metrics_port INTEGER NOT NULL DEFAULT 9090,
		metrics_path TEXT NOT NULL DEFAULT '/metrics',
		status TEXT NOT NULL DEFAULT 'offline',
		last_heartbeat_at TEXT,
		tenant_id TEXT NOT NULL DEFAULT 'default',
		owner_id TEXT,
		created_by TEXT NOT NULL DEFAULT 'system',
		updated_by TEXT NOT NULL DEFAULT 'system',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant ON tenant_memberships(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user ON tenant_memberships(user_id);
	CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_agent ON nodes(agent_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
	`

	_, err := db.Exec(schema)
	return err
}
