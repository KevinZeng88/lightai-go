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
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

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
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now')),
		description TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var currentVersion int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&currentVersion)
	if err != nil {
		currentVersion = 0
	}

	if currentVersion < 1 {
		if err := db.migrateV1(); err != nil {
			return fmt.Errorf("migrate v1: %w", err)
		}
	}

	if currentVersion < 2 {
		if err := db.migrateV2(); err != nil {
			return fmt.Errorf("migrate v2: %w", err)
		}
	}

	return nil
}

// DefaultTenantID returns the UUID of the default tenant (looked up by slug='default').
func (db *DB) DefaultTenantID() string {
	var id string
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'default'`).Scan(&id)
	return id
}

// migrateV2 adds node detail fields: primary_ip, os, arch, kernel, agent_version.
func (db *DB) migrateV2() error {
	cols := []struct {
		name    string
		sqlType string
	}{
		{"primary_ip", "TEXT NOT NULL DEFAULT ''"},
		{"os", "TEXT NOT NULL DEFAULT ''"},
		{"arch", "TEXT NOT NULL DEFAULT ''"},
		{"kernel", "TEXT NOT NULL DEFAULT ''"},
		{"agent_version", "TEXT NOT NULL DEFAULT ''"},
	}

	for _, col := range cols {
		if _, err := db.Exec("ALTER TABLE nodes ADD COLUMN " + col.name + " " + col.sqlType); err != nil {
			// Column may already exist from a prior partial migration — ignore.
			// SQLite doesn't support DROP COLUMN easily, so skip if column exists.
			continue
		}
	}

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (2, 'V2: add node detail fields (primary_ip, os, arch, kernel, agent_version)')`); err != nil {
		return err
	}
	return nil
}

// SchemaVersion returns the current database schema version.
func (db *DB) SchemaVersion() int {
	var v int
	db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&v)
	return v
}

// migrateV1 applies the initial RC1 schema.
func (db *DB) migrateV1() error {
	schema := `
		CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL DEFAULT '',
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
			tenant_id TEXT NOT NULL,
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			operator_user_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
		CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant ON tenant_memberships(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user ON tenant_memberships(user_id);
		CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
		CREATE INDEX IF NOT EXISTS idx_nodes_agent ON nodes(agent_id);
		CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
		CREATE INDEX IF NOT EXISTS idx_nodes_tenant_id ON nodes(tenant_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (1, 'RC1 schema: tenants(slug), users, memberships, roles, permissions, sessions, nodes, audit_logs')`); err != nil {
		return err
	}

	return nil
}
