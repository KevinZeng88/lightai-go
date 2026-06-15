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

	if currentVersion < 3 {
		if err := db.migrateV3(); err != nil {
			return fmt.Errorf("migrate v3: %w", err)
		}
	}

	if currentVersion < 4 {
		if err := db.migrateV4(); err != nil {
			return fmt.Errorf("migrate v4: %w", err)
		}
	}

	if currentVersion < 5 {
		if err := db.migrateV5(); err != nil {
			return fmt.Errorf("migrate v5: %w", err)
		}
	}

	if currentVersion < 6 {
		if err := db.migrateV6(); err != nil {
			return fmt.Errorf("migrate v6: %w", err)
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

// migrateV3 adds model runtime serving tables (Phase 1).
func (db *DB) migrateV3() error {
	schema := `
		CREATE TABLE IF NOT EXISTS model_artifacts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL DEFAULT '',
			source_type TEXT NOT NULL DEFAULT 'local_path',
			path TEXT NOT NULL DEFAULT '',
			format TEXT NOT NULL DEFAULT 'custom',
			task_type TEXT NOT NULL DEFAULT 'chat',
			architecture TEXT NOT NULL DEFAULT 'custom',
			size_label TEXT NOT NULL DEFAULT '',
			quantization TEXT NOT NULL DEFAULT 'unknown',
			default_context_length INTEGER NOT NULL DEFAULT 0,
			estimated_vram_bytes INTEGER NOT NULL DEFAULT 0,
			required_gpu_count INTEGER NOT NULL DEFAULT 1,
			tenant_id TEXT NOT NULL,
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS runtime_environments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			runtime_type TEXT NOT NULL DEFAULT 'docker',
			backend_type TEXT NOT NULL DEFAULT 'custom',
			vendor TEXT NOT NULL DEFAULT 'custom',
			openai_compatible INTEGER NOT NULL DEFAULT 0,
			default_port INTEGER NOT NULL DEFAULT 8000,
			health_check_path TEXT NOT NULL DEFAULT '/health',
			description TEXT NOT NULL DEFAULT '',
			tenant_id TEXT,
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(tenant_id, name)
		);

		CREATE TABLE IF NOT EXISTS runtime_environment_docker_specs (
			id TEXT PRIMARY KEY,
			runtime_environment_id TEXT NOT NULL UNIQUE REFERENCES runtime_environments(id),
			image TEXT NOT NULL DEFAULT '',
			image_pull_policy TEXT NOT NULL DEFAULT 'never',
			devices TEXT NOT NULL DEFAULT '[]',
			privileged TEXT NOT NULL DEFAULT '{"enabled":false}',
			ipc_mode TEXT NOT NULL DEFAULT '{"enabled":false}',
			uts_mode TEXT NOT NULL DEFAULT '{"enabled":false}',
			network_mode TEXT NOT NULL DEFAULT '{"enabled":false}',
			shm_size TEXT NOT NULL DEFAULT '{"enabled":false}',
			group_add TEXT NOT NULL DEFAULT '{"enabled":false}',
			security_options TEXT NOT NULL DEFAULT '{"enabled":false}',
			ulimits TEXT NOT NULL DEFAULT '{"enabled":false}',
			restart_policy TEXT NOT NULL DEFAULT '{"enabled":false}',
			gpu_visible_env_key TEXT NOT NULL DEFAULT 'CUDA_VISIBLE_DEVICES',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS run_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			runtime_type TEXT NOT NULL DEFAULT 'docker',
			vendor TEXT NOT NULL DEFAULT 'custom',
			backend_type TEXT NOT NULL DEFAULT 'custom',
			required_variables TEXT NOT NULL DEFAULT '[]',
			optional_variables TEXT NOT NULL DEFAULT '[]',
			env_mappings TEXT NOT NULL DEFAULT '{"enabled":false}',
			args_template TEXT NOT NULL DEFAULT '[]',
			volume_mappings TEXT NOT NULL DEFAULT '{"enabled":false}',
			port_mappings TEXT NOT NULL DEFAULT '{"enabled":false}',
			backend_flags TEXT NOT NULL DEFAULT '{"enabled":false}',
			description TEXT NOT NULL DEFAULT '',
			tenant_id TEXT,
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(tenant_id, name)
		);

		CREATE TABLE IF NOT EXISTS model_deployments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			model_artifact_id TEXT NOT NULL REFERENCES model_artifacts(id),
			runtime_environment_id TEXT NOT NULL REFERENCES runtime_environments(id),
			run_template_id TEXT NOT NULL REFERENCES run_templates(id),
			replicas INTEGER NOT NULL DEFAULT 1,
			desired_state TEXT NOT NULL DEFAULT 'stopped',
			status TEXT NOT NULL DEFAULT 'stopped',
			node_id TEXT NOT NULL DEFAULT '',
			gpu_ids TEXT NOT NULL DEFAULT '[]',
			host_port INTEGER NOT NULL DEFAULT 0,
			served_model_name TEXT NOT NULL DEFAULT '',
			max_model_len INTEGER NOT NULL DEFAULT 0,
			tensor_parallel_size INTEGER NOT NULL DEFAULT 1,
			gpu_memory_utilization REAL NOT NULL DEFAULT 0.9,
			dtype TEXT NOT NULL DEFAULT 'auto',
			gpu_visible_env_key TEXT NOT NULL DEFAULT '',
			env_overrides TEXT NOT NULL DEFAULT '{}',
			arg_overrides TEXT NOT NULL DEFAULT '{}',
			extra_args TEXT NOT NULL DEFAULT '[]',
			schedule_mode TEXT NOT NULL DEFAULT 'manual',
			placement_strategy TEXT NOT NULL DEFAULT 'manual',
			expose_mode TEXT NOT NULL DEFAULT 'direct',
			service_path TEXT NOT NULL DEFAULT '',
			tenant_id TEXT NOT NULL,
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS model_instances (
			id TEXT PRIMARY KEY,
			deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
			replica_index INTEGER NOT NULL DEFAULT 0,
			node_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			runtime_type TEXT NOT NULL DEFAULT 'docker',
			gpu_ids TEXT NOT NULL DEFAULT '[]',
			gpu_lease_ids TEXT NOT NULL DEFAULT '[]',
			desired_state TEXT NOT NULL DEFAULT 'stopped',
			actual_state TEXT NOT NULL DEFAULT 'pending',
			container_id TEXT NOT NULL DEFAULT '',
			process_id INTEGER NOT NULL DEFAULT 0,
			remote_url TEXT NOT NULL DEFAULT '',
			endpoint_url TEXT NOT NULL DEFAULT '',
			host_port INTEGER NOT NULL DEFAULT 0,
			container_port INTEGER NOT NULL DEFAULT 0,
			restart_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			last_exit_code INTEGER NOT NULL DEFAULT 0,
			resolved_run_spec TEXT NOT NULL DEFAULT '{}',
			started_at TEXT,
			stopped_at TEXT,
			last_heartbeat_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS gpu_leases (
			id TEXT PRIMARY KEY,
			gpu_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			deployment_id TEXT NOT NULL,
			instance_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'reserved',
			expires_at TEXT,
			reserved_at TEXT NOT NULL DEFAULT (datetime('now')),
			activated_at TEXT,
			released_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_model_artifacts_tenant ON model_artifacts(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_runtime_environments_tenant ON runtime_environments(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_run_templates_tenant ON run_templates(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_model_deployments_tenant ON model_deployments(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_model_deployments_status ON model_deployments(status);
		CREATE INDEX IF NOT EXISTS idx_model_instances_deployment ON model_instances(deployment_id);
		CREATE INDEX IF NOT EXISTS idx_model_instances_actual_state ON model_instances(actual_state);
		CREATE INDEX IF NOT EXISTS idx_gpu_leases_gpu ON gpu_leases(gpu_id);
		CREATE INDEX IF NOT EXISTS idx_gpu_leases_status ON gpu_leases(status);
		CREATE INDEX IF NOT EXISTS idx_gpu_leases_tenant ON gpu_leases(tenant_id);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (3, 'V3: model runtime serving Phase 1 tables')`); err != nil {
		return err
	}
	return nil
}

// migrateV4 adds GpuLease lifecycle timestamp columns for Phase 2 readiness.
func (db *DB) migrateV4() error {
	cols := []struct {
		name    string
		sqlType string
	}{
		{"reserved_at", "TEXT NOT NULL DEFAULT (datetime('now'))"},
		{"activated_at", "TEXT"},
		{"released_at", "TEXT"},
	}
	for _, col := range cols {
		db.Exec("ALTER TABLE gpu_leases ADD COLUMN " + col.name + " " + col.sqlType)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (4, 'V4: add gpu_leases lifecycle timestamps (reserved_at, activated_at, released_at)')`); err != nil {
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

// migrateV5 adds agent_tasks table for Phase 2B task dispatch.
func (db *DB) migrateV5() error {
	schema := `
		CREATE TABLE IF NOT EXISTS agent_tasks (
			id TEXT PRIMARY KEY,
			task_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			tenant_id TEXT NOT NULL,
			deployment_id TEXT NOT NULL,
			instance_id TEXT,
			node_id TEXT NOT NULL,
			requested_by TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT '{}',
			result TEXT NOT NULL DEFAULT '{}',
			timeout_seconds INTEGER NOT NULL DEFAULT 300,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent_tasks(status);
		CREATE INDEX IF NOT EXISTS idx_agent_tasks_node ON agent_tasks(node_id);
		CREATE INDEX IF NOT EXISTS idx_agent_tasks_tenant ON agent_tasks(tenant_id);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (5, 'V5: agent tasks for Phase 2B task dispatch')`); err != nil {
		return err
	}
	return nil
}

// migrateV6 adds task lifecycle columns, model_instances tenant_id, and status cleanup.
func (db *DB) migrateV6() error {
	schema := `
		ALTER TABLE agent_tasks ADD COLUMN claimed_at TEXT;
		ALTER TABLE agent_tasks ADD COLUMN started_at TEXT;
		ALTER TABLE agent_tasks ADD COLUMN finished_at TEXT;
		ALTER TABLE agent_tasks ADD COLUMN agent_id TEXT NOT NULL DEFAULT '';
		ALTER TABLE agent_tasks ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE model_instances ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// Backfill model_instances.tenant_id from parent deployment.
	db.Exec(`UPDATE model_instances SET tenant_id = (
		SELECT COALESCE(md.tenant_id, '') FROM model_deployments md WHERE md.id = model_instances.deployment_id
	) WHERE tenant_id = '' OR tenant_id IS NULL`)
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (6, 'V6: task lifecycle, instance tenant_id, claim support')`); err != nil {
		return err
	}
	return nil
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
