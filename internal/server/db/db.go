// Package db provides SQLite database initialization and access.
package db

import (
	"database/sql"
	"fmt"
	"time"

	"lightai-go/internal/common/log"
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
	migrateStart := time.Now()
	log.Info("db migrate: begin")
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

	if currentVersion < 7 {
		if err := db.migrateV7(); err != nil {
			return fmt.Errorf("migrate v7: %w", err)
		}
	}

	if currentVersion < 8 {
		if err := db.migrateV8(); err != nil {
			return fmt.Errorf("migrate v8: %w", err)
		}
	}

	if currentVersion < 9 {
		if err := db.migrateV9(); err != nil {
			return fmt.Errorf("migrate v9: %w", err)
		}
	}

	if currentVersion < 10 {
		if err := db.migrateV10(); err != nil {
			return fmt.Errorf("migrate v10: %w", err)
		}
	}

	if currentVersion < 11 {
		if err := db.migrateV11(); err != nil {
			return fmt.Errorf("migrate v11: %w", err)
		}
	}

	if currentVersion < 12 {
		if err := db.migrateV12(); err != nil {
			return fmt.Errorf("migrate v12: %w", err)
		}
	}

	if currentVersion < 13 {
		if err := db.migrateV13(); err != nil {
			return fmt.Errorf("migrate v13: %w", err)
		}
	}

	if currentVersion < 14 {
		if err := db.migrateV14(); err != nil {
			return fmt.Errorf("migrate v14: %w", err)
		}
	}
	if currentVersion < 15 {
		if err := db.migrateV15(); err != nil {
			return fmt.Errorf("migrate v15: %w", err)
		}
	}
	if currentVersion < 16 {
		if err := db.migrateV16(); err != nil {
			return fmt.Errorf("migrate v16: %w", err)
		}
	}
	if currentVersion < 17 {
		if err := db.migrateV17(); err != nil {
			return fmt.Errorf("migrate v17: %w", err)
		}
	}
	if currentVersion < 18 {
		if err := db.migrateV18(); err != nil {
			return fmt.Errorf("migrate v18: %w", err)
		}
	}
	if currentVersion < 19 {
		if err := db.migrateV19(); err != nil {
			return fmt.Errorf("migrate v19: %w", err)
		}
	}
	if currentVersion < 20 {
		if err := db.migrateV20(); err != nil {
			return fmt.Errorf("migrate v20: %w", err)
		}
	}
	if currentVersion < 21 {
		if err := db.migrateV21(); err != nil {
			return fmt.Errorf("migrate v21: %w", err)
		}
	}

	if currentVersion < 22 {
		if err := db.migrateV22(); err != nil {
			return fmt.Errorf("migrate v22: %w", err)
		}
	}
	if currentVersion < 23 {
		if err := db.migrateV23(); err != nil {
			return fmt.Errorf("migrate v23: %w", err)
		}
	}
	if currentVersion < 24 {
		if err := db.migrateV24(); err != nil {
			return fmt.Errorf("migrate v24: %w", err)
		}
	}
	if currentVersion < 25 {
		if err := db.migrateV25(); err != nil {
			return fmt.Errorf("migrate v25: %w", err)
		}
	}
	if currentVersion < 26 {
		if err := db.migrateV26(); err != nil {
			return fmt.Errorf("migrate v26: %w", err)
		}
	}
	// Target Backend Catalog seed is idempotent and must also repair existing
	// databases that reached V13 before the target stable IDs were added.
	db.seedTargetBackendCatalog()
	// V27: force-update backend capabilities AFTER seed (seed UPDATE may overwrite).
	// This runs every time (not gated on schema_version) because the seed
	// function's UPDATE can reset capabilities_json to the old struct value
	// if the DB was created with an older seed data version.
	db.repairBackendCapabilitiesV27()

	log.Info("db migrate: completed", "duration_ms", time.Since(migrateStart).Milliseconds())
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
			capabilities_json TEXT NOT NULL DEFAULT '[]',
			capability_sources_json TEXT NOT NULL DEFAULT '{}',
			default_test_mode TEXT NOT NULL DEFAULT 'auto',
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
			volumes TEXT NOT NULL DEFAULT '{"enabled":false}',
			extra_args TEXT NOT NULL DEFAULT '[]',
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
			extra_args TEXT NOT NULL DEFAULT '[]',
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

// migrateV7 adds tenant type, ResourcePool, and resource ownership fields.
func (db *DB) migrateV7() error {
	schema := `
		ALTER TABLE tenants ADD COLUMN type TEXT NOT NULL DEFAULT 'business';

		CREATE TABLE IF NOT EXISTS resource_pools (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			owner_tenant_id TEXT NOT NULL,
			visibility TEXT NOT NULL DEFAULT 'private',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS resource_pool_nodes (
			pool_id TEXT NOT NULL REFERENCES resource_pools(id),
			node_id TEXT NOT NULL,
			PRIMARY KEY (pool_id, node_id)
		);

		CREATE TABLE IF NOT EXISTS resource_pool_gpus (
			pool_id TEXT NOT NULL REFERENCES resource_pools(id),
			gpu_id TEXT NOT NULL,
			PRIMARY KEY (pool_id, gpu_id)
		);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Mark default tenant and existing tenants with sensible defaults.
	db.Exec(`UPDATE tenants SET type = 'business' WHERE type = '' OR type IS NULL`)
	db.Exec(`UPDATE tenants SET type = 'infrastructure' WHERE slug = 'default'`)

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (7, 'V7: tenant type, resource pools, ownership model')`); err != nil {
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

// migrateV8 adds a partial unique index on gpu_leases to prevent concurrent
// double-leasing of the same GPU (C4 fix: lease race condition).
func (db *DB) migrateV8() error {
	schema := `CREATE UNIQUE INDEX IF NOT EXISTS idx_gpu_leases_reserved_active
		ON gpu_leases(gpu_id) WHERE status IN ('reserved','active')`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate v8: %w (partial unique index may not be supported on this SQLite version)", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (8, 'V8: partial unique index on gpu_leases(gpu_id) for reserved/active')`); err != nil {
		return err
	}
	return nil
}

func (db *DB) migrateV9() error {
	// Add extra_args column to run_templates.
	if _, err := db.Exec("ALTER TABLE run_templates ADD COLUMN extra_args TEXT NOT NULL DEFAULT '[]'"); err != nil {
		// Column may already exist — ignore.
	}
	// Add volumes column to runtime_environment_docker_specs.
	if _, err := db.Exec("ALTER TABLE runtime_environment_docker_specs ADD COLUMN volumes TEXT NOT NULL DEFAULT '{\"enabled\":false}'"); err != nil {
	}
	// Add extra_args column to runtime_environment_docker_specs.
	if _, err := db.Exec("ALTER TABLE runtime_environment_docker_specs ADD COLUMN extra_args TEXT NOT NULL DEFAULT '[]'"); err != nil {
	}
	if _, err := db.Exec("INSERT OR IGNORE INTO schema_version (version, description) VALUES (9, 'V9: extra_args on run_templates, volumes+extra_args on docker_specs')"); err != nil {
		return err
	}
	return nil
}

// migrateV10 replaces the old Phase 1 model runtime chain with the new Backend/Runtime/RunPlan design.
func (db *DB) migrateV10() error {
	// 1. Drop old Phase 1 tables.
	db.Exec(`DROP TABLE IF EXISTS runtime_environment_docker_specs`)
	db.Exec(`DROP TABLE IF EXISTS runtime_environments`)
	db.Exec(`DROP TABLE IF EXISTS run_templates`)
	db.Exec(`DROP TABLE IF EXISTS model_deployments`)
	db.Exec(`DROP TABLE IF EXISTS model_instances`)
	db.Exec(`DROP TABLE IF EXISTS gpu_leases`)
	db.Exec(`DROP TABLE IF EXISTS agent_tasks`)
	// model_artifacts is preserved and restructured below.

	// 2. Clean up any stray model_artifacts_new from previous migration attempts.
	db.Exec(`DROP TABLE IF EXISTS model_artifacts_new`)
	db.Exec(`DROP TABLE IF EXISTS model_artifacts_old`)
	// model_artifacts from V3 is preserved as-is (with source_type column).

	// 3. Create inference_backends.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS inference_backends (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		protocol_json TEXT NOT NULL DEFAULT '{}',
		default_version TEXT NOT NULL DEFAULT '',
		parameter_format TEXT NOT NULL DEFAULT 'space',
		common_parameters_json TEXT NOT NULL DEFAULT '[]',
		default_env_json TEXT NOT NULL DEFAULT '{}',
		is_builtin INTEGER NOT NULL DEFAULT 0,
		is_enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 4. Create backend_versions.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS backend_versions (
		id TEXT PRIMARY KEY,
		backend_id TEXT NOT NULL REFERENCES inference_backends(id),
		version TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		is_default INTEGER NOT NULL DEFAULT 0,
		default_entrypoint_json TEXT NOT NULL DEFAULT '[]',
		default_args_json TEXT NOT NULL DEFAULT '[]',
		default_backend_params_json TEXT NOT NULL DEFAULT '[]',
		parameter_defs_json TEXT NOT NULL DEFAULT '[]',
		health_check_json TEXT NOT NULL DEFAULT '{}',
		default_container_port INTEGER NOT NULL DEFAULT 8000,
		default_images_json TEXT NOT NULL DEFAULT '{}',
		env_json TEXT NOT NULL DEFAULT '{}',
		is_deprecated INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(backend_id, version)
	)`); err != nil {
		return err
	}

	// 5. Create backend_runtimes.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS backend_runtimes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		backend_id TEXT NOT NULL REFERENCES inference_backends(id),
		backend_version_id TEXT NOT NULL REFERENCES backend_versions(id),
		source_template_name TEXT NOT NULL DEFAULT '',
		vendor TEXT NOT NULL DEFAULT 'custom',
		runtime_type TEXT NOT NULL DEFAULT 'docker',
		image_name TEXT NOT NULL DEFAULT '',
		image_pull_policy TEXT NOT NULL DEFAULT 'if_not_present',
		entrypoint_override_json TEXT NOT NULL DEFAULT '[]',
		args_override_json TEXT NOT NULL DEFAULT '[]',
		default_env_json TEXT NOT NULL DEFAULT '{}',
		docker_json TEXT NOT NULL DEFAULT '{}',
		model_mount_json TEXT NOT NULL DEFAULT '{}',
		health_check_override_json TEXT NOT NULL DEFAULT '{}',
		is_builtin INTEGER NOT NULL DEFAULT 0,
		is_editable INTEGER NOT NULL DEFAULT 1,
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(tenant_id, name)
	)`); err != nil {
		return err
	}

	// 6. Create node_runtime_overrides.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_runtime_overrides (
		id TEXT PRIMARY KEY,
		node_id TEXT NOT NULL REFERENCES nodes(id),
		tenant_id TEXT NOT NULL DEFAULT '',
		backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
		image_name TEXT NOT NULL DEFAULT '',
		image_pull_policy TEXT NOT NULL DEFAULT '',
		env_json TEXT NOT NULL DEFAULT '{}',
		docker_override_json TEXT NOT NULL DEFAULT '{}',
		model_root_host_path TEXT NOT NULL DEFAULT '',
		is_enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(node_id, backend_runtime_id)
	)`); err != nil {
		return err
	}

	// 7. Create model_deployments.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_deployments (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		model_artifact_id TEXT NOT NULL REFERENCES model_artifacts(id),
		backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
		replicas INTEGER NOT NULL DEFAULT 1,
		placement_json TEXT NOT NULL DEFAULT '{}',
		service_json TEXT NOT NULL DEFAULT '{}',
		parameters_json TEXT NOT NULL DEFAULT '{}',
		env_overrides_json TEXT NOT NULL DEFAULT '{}',
		desired_state TEXT NOT NULL DEFAULT 'stopped',
		status TEXT NOT NULL DEFAULT 'stopped',
		tenant_id TEXT NOT NULL,
		owner_id TEXT,
		created_by TEXT NOT NULL DEFAULT 'system',
		updated_by TEXT NOT NULL DEFAULT 'system',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 8. Create model_instances.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_instances (
		id TEXT PRIMARY KEY,
		deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
		tenant_id TEXT NOT NULL DEFAULT '',
		replica_index INTEGER NOT NULL DEFAULT 0,
		node_id TEXT NOT NULL DEFAULT '',
		agent_id TEXT NOT NULL DEFAULT '',
		assigned_gpus_json TEXT NOT NULL DEFAULT '[]',
		gpu_lease_ids_json TEXT NOT NULL DEFAULT '[]',
		host_port INTEGER NOT NULL DEFAULT 0,
		container_port INTEGER NOT NULL DEFAULT 0,
		current_run_plan_id TEXT,
		actual_state TEXT NOT NULL DEFAULT 'pending',
		desired_state TEXT NOT NULL DEFAULT 'running',
		container_id TEXT NOT NULL DEFAULT '',
		endpoint_url TEXT NOT NULL DEFAULT '',
		restart_count INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		started_at TEXT,
		stopped_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 9. Create resolved_run_plans.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS resolved_run_plans (
		id TEXT PRIMARY KEY,
		deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
		instance_id TEXT,
		tenant_id TEXT NOT NULL DEFAULT '',
		backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
		node_runtime_override_id TEXT REFERENCES node_runtime_overrides(id),
		plan_json TEXT NOT NULL DEFAULT '{}',
		docker_preview TEXT NOT NULL DEFAULT '',
		input_hash TEXT NOT NULL DEFAULT '',
		plan_hash TEXT NOT NULL DEFAULT '',
		created_by TEXT NOT NULL DEFAULT 'system',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 10. Create gpu_leases.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS gpu_leases (
		id TEXT PRIMARY KEY,
		gpu_id TEXT NOT NULL,
		node_id TEXT NOT NULL,
		deployment_id TEXT NOT NULL,
		instance_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'reserved',
		expires_at TEXT,
		reserved_at TEXT NOT NULL DEFAULT (datetime('now')),
		activated_at TEXT,
		released_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 11. Create agent_tasks.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_tasks (
		id TEXT PRIMARY KEY,
		task_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		tenant_id TEXT NOT NULL DEFAULT '',
		deployment_id TEXT NOT NULL,
		instance_id TEXT,
		node_id TEXT NOT NULL,
		agent_id TEXT,
		requested_by TEXT NOT NULL DEFAULT 'system',
		payload TEXT NOT NULL DEFAULT '{}',
		result TEXT,
		timeout_seconds INTEGER NOT NULL DEFAULT 300,
		retry_count INTEGER NOT NULL DEFAULT 0,
		claimed_at TEXT,
		started_at TEXT,
		finished_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	// 12. Create indexes.
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_artifacts_tenant ON model_artifacts(tenant_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_backend_runtimes_tenant ON backend_runtimes(tenant_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_node_runtime_overrides_node ON node_runtime_overrides(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_deployments_tenant ON model_deployments(tenant_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_deployments_status ON model_deployments(status)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_instances_deployment ON model_instances(deployment_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_instances_state ON model_instances(actual_state)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_instances_tenant ON model_instances(tenant_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_resolved_run_plans_deployment ON resolved_run_plans(deployment_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_resolved_run_plans_instance ON resolved_run_plans(instance_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_gpu_leases_gpu ON gpu_leases(gpu_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_gpu_leases_status ON gpu_leases(status)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_gpu_leases_tenant ON gpu_leases(tenant_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent_tasks(status)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_tasks_node ON agent_tasks(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_tasks_tenant ON agent_tasks(tenant_id)`)

	// 13. Seed built-in inference backends.
	db.seedBuiltInBackends()

	// 14. Record schema version.
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (10, 'V10: Phase 3 backend/runplan/runtime new tables (replaces old Phase 1 model runtime)')`); err != nil {
		return err
	}

	return nil
}

// migrateV11 adds task lease and idempotency columns (REVIEW-004).
func (db *DB) migrateV11() error {
	cols := []string{
		"ALTER TABLE agent_tasks ADD COLUMN lease_owner TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agent_tasks ADD COLUMN lease_expires_at TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agent_tasks ADD COLUMN operation_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agent_tasks ADD COLUMN generation INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE agent_tasks ADD COLUMN attempt INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE agent_tasks ADD COLUMN max_attempts INTEGER NOT NULL DEFAULT 3",
	}
	for _, c := range cols {
		if _, err := db.Exec(c); err != nil {
			// Column may already exist — non-fatal.
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (11, 'V11: task lease and idempotency columns on agent_tasks')`); err != nil {
		return err
	}
	return nil
}

// migrateV12 adds audit_logs tenant_id, centralizes resource tables, and splits
// collected_at/reported_at (REVIEW-009,010,021).
func (db *DB) migrateV12() error {
	// Add tenant_id to audit_logs (REVIEW-009).
	if _, err := db.Exec(`ALTER TABLE audit_logs ADD COLUMN tenant_id TEXT NOT NULL DEFAULT ''`); err != nil {
		// Column may exist — non-fatal.
	}

	// Create resource tables in central migration (REVIEW-010).
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS gpu_devices (
		id TEXT PRIMARY KEY,
		node_id TEXT NOT NULL,
		vendor TEXT NOT NULL,
		index_num INTEGER NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		uuid TEXT NOT NULL DEFAULT '',
		pci_bus_id TEXT NOT NULL DEFAULT '',
		driver_version TEXT NOT NULL DEFAULT '',
		memory_total_bytes INTEGER NOT NULL DEFAULT 0,
		memory_used_bytes INTEGER NOT NULL DEFAULT 0,
		memory_free_bytes INTEGER NOT NULL DEFAULT 0,
		gpu_utilization_percent REAL,
		memory_utilization_percent REAL,
		temperature_celsius REAL,
		power_draw_watts REAL,
		health TEXT NOT NULL DEFAULT 'unknown',
		status TEXT NOT NULL DEFAULT 'available',
		collected_at TEXT,
		reported_at TEXT NOT NULL DEFAULT '',
		tenant_id TEXT NOT NULL DEFAULT 'default',
		owner_id TEXT,
		created_by TEXT NOT NULL DEFAULT 'system',
		updated_by TEXT NOT NULL DEFAULT 'system',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create gpu_devices: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_system_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		cpu_utilization_percent TEXT NOT NULL DEFAULT '0',
		memory_total_bytes INTEGER NOT NULL DEFAULT 0,
		memory_used_bytes INTEGER NOT NULL DEFAULT 0,
		swap_total_bytes INTEGER NOT NULL DEFAULT 0,
		swap_used_bytes INTEGER NOT NULL DEFAULT 0,
		uptime_seconds TEXT NOT NULL DEFAULT '0',
		cpu_cores INTEGER NOT NULL DEFAULT 0,
		load1 TEXT NOT NULL DEFAULT '0',
		load5 TEXT NOT NULL DEFAULT '0',
		load15 TEXT NOT NULL DEFAULT '0',
		collected_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create node_system_snapshots: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_filesystem_snapshots (
		node_id TEXT NOT NULL,
		mount_point TEXT NOT NULL,
		device TEXT NOT NULL DEFAULT '',
		fs_type TEXT NOT NULL DEFAULT '',
		total_bytes INTEGER NOT NULL DEFAULT 0,
		used_bytes INTEGER NOT NULL DEFAULT 0,
		free_bytes INTEGER NOT NULL DEFAULT 0,
		used_percent TEXT NOT NULL DEFAULT '0',
		collected_at TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (node_id, mount_point)
	)`); err != nil {
		return fmt.Errorf("create node_filesystem_snapshots: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_network_snapshots (
		node_id TEXT NOT NULL,
		interface_name TEXT NOT NULL,
		up INTEGER NOT NULL DEFAULT 0,
		bytes_recv INTEGER NOT NULL DEFAULT 0,
		bytes_sent INTEGER NOT NULL DEFAULT 0,
		collected_at TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (node_id, interface_name)
	)`); err != nil {
		return fmt.Errorf("create node_network_snapshots: %w", err)
	}

	// Add reported_at column to gpu_devices (REVIEW-021: split collected_at/reported_at).
	if _, err := db.Exec(`ALTER TABLE gpu_devices ADD COLUMN reported_at TEXT NOT NULL DEFAULT ''`); err != nil {
		// Column may exist — non-fatal.
	}

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (12, 'V12: audit tenant_id, central resource schema, collected_at/reported_at split')`); err != nil {
		return err
	}
	return nil
}

// migrateV13 aligns the runtime schema with the Backend / BackendVersion /
// BackendRuntime / NodeBackendRuntime / ModelLocation target design.
func (db *DB) migrateV13() error {
	alterStatements := []string{
		`ALTER TABLE inference_backends ADD COLUMN slug TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE inference_backends ADD COLUMN managed_by TEXT NOT NULL DEFAULT 'system'`,
		`ALTER TABLE inference_backends ADD COLUMN source TEXT NOT NULL DEFAULT 'embedded'`,
		`ALTER TABLE inference_backends ADD COLUMN catalog_version TEXT NOT NULL DEFAULT 'v1'`,
		`ALTER TABLE inference_backends ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE inference_backends ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`,
		`ALTER TABLE backend_versions ADD COLUMN slug TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN managed_by TEXT NOT NULL DEFAULT 'system'`,
		`ALTER TABLE backend_versions ADD COLUMN source TEXT NOT NULL DEFAULT 'embedded'`,
		`ALTER TABLE backend_versions ADD COLUMN catalog_version TEXT NOT NULL DEFAULT 'v1'`,
		`ALTER TABLE backend_versions ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`,
		`ALTER TABLE backend_runtimes ADD COLUMN slug TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN managed_by TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE backend_runtimes ADD COLUMN source TEXT NOT NULL DEFAULT 'api'`,
		`ALTER TABLE backend_runtimes ADD COLUMN catalog_version TEXT NOT NULL DEFAULT 'v1'`,
		`ALTER TABLE backend_runtimes ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`,
		`ALTER TABLE backend_runtimes ADD COLUMN verification_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE resolved_run_plans ADD COLUMN node_backend_runtime_id TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_locations (
		id TEXT PRIMARY KEY,
		model_artifact_id TEXT NOT NULL REFERENCES model_artifacts(id),
		node_id TEXT NOT NULL REFERENCES nodes(id),
		path_type TEXT NOT NULL DEFAULT 'directory',
		model_root TEXT NOT NULL DEFAULT '',
		relative_path TEXT NOT NULL DEFAULT '',
		absolute_path TEXT NOT NULL DEFAULT '',
		size_bytes INTEGER NOT NULL DEFAULT 0,
		checksum TEXT NOT NULL DEFAULT '',
		manifest_digest TEXT NOT NULL DEFAULT '',
		discovered_metadata_json TEXT NOT NULL DEFAULT '{}',
		match_status TEXT NOT NULL DEFAULT 'exact_match',
		verification_status TEXT NOT NULL DEFAULT 'verified',
		manual_override INTEGER NOT NULL DEFAULT 0,
		override_reason TEXT NOT NULL DEFAULT '',
		override_by TEXT NOT NULL DEFAULT '',
		override_at TEXT,
		last_scanned_at TEXT,
		last_error TEXT NOT NULL DEFAULT '',
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(model_artifact_id, node_id, absolute_path)
	)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_backend_runtimes (
		id TEXT PRIMARY KEY,
		backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
		node_id TEXT NOT NULL REFERENCES nodes(id),
		display_name TEXT NOT NULL DEFAULT '',
		runner_type TEXT NOT NULL DEFAULT 'docker',
		image_ref TEXT NOT NULL DEFAULT '',
		image_id TEXT NOT NULL DEFAULT '',
		image_digest TEXT NOT NULL DEFAULT '',
		image_present INTEGER NOT NULL DEFAULT 0,
		docker_available INTEGER NOT NULL DEFAULT 0,
		driver_version TEXT NOT NULL DEFAULT '',
		toolkit_version TEXT NOT NULL DEFAULT '',
		device_check_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT 'unknown',
		status_reason TEXT NOT NULL DEFAULT '',
		last_checked_at TEXT,
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(node_id, backend_runtime_id)
	)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS run_plan_groups (
		id TEXT PRIMARY KEY,
		deployment_plan_id TEXT NOT NULL REFERENCES model_deployments(id),
		mode TEXT NOT NULL DEFAULT 'single',
		desired_count INTEGER NOT NULL DEFAULT 1,
		ready_count INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		group_config_json TEXT NOT NULL DEFAULT '{}',
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}

	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_locations_artifact ON model_locations(model_artifact_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_model_locations_node ON model_locations(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_node_backend_runtimes_node ON node_backend_runtimes(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_node_backend_runtimes_runtime ON node_backend_runtimes(backend_runtime_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_run_plan_groups_deployment ON run_plan_groups(deployment_plan_id)`)

	db.seedTargetBackendCatalog()

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (13, 'V13: backend catalog, node backend runtime, model locations, run plan groups')`); err != nil {
		return err
	}
	return nil
}

// seedBuiltInBackends inserts the three built-in inference backends.
func (db *DB) seedBuiltInBackends() {
	// vLLM
	db.Exec(`INSERT OR IGNORE INTO inference_backends (id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled)
		VALUES ('backend-vllm', 'vllm', 'vLLM', 'vLLM inference backend',
		'{"type":"openai-compatible","modelsPath":"/v1/models","chatCompletionsPath":"/v1/chat/completions","completionsPath":"/v1/completions"}',
		'0.8.5', 'space', '["--tensor-parallel-size","--max-model-len","--gpu-memory-utilization","--served-model-name"]', '{"VLLM_USE_MODELSCOPE":"false"}', 1, 1)`)

	// SGLang
	db.Exec(`INSERT OR IGNORE INTO inference_backends (id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled)
		VALUES ('backend-sglang', 'sglang', 'SGLang', 'SGLang inference backend',
		'{"type":"openai-compatible","modelsPath":"/v1/models","chatCompletionsPath":"/v1/chat/completions","completionsPath":"/v1/completions"}',
		'0.4.6', 'space', '["--tp","--context-length","--mem-fraction-static","--served-model-name"]', '{}', 1, 1)`)

	// llama.cpp
	db.Exec(`INSERT OR IGNORE INTO inference_backends (id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled)
		VALUES ('backend-llamacpp', 'llamacpp', 'llama.cpp', 'llama.cpp inference backend',
		'{"type":"openai-compatible","modelsPath":"/v1/models","chatCompletionsPath":"/v1/chat/completions","completionsPath":"/v1/completions"}',
		'b4817', 'space', '["-ngl","--ctx-size","--batch-size","--model"]', '{}', 1, 1)`)

	// Seed backend versions for vLLM
	db.Exec(`INSERT OR IGNORE INTO backend_versions (id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated)
		VALUES ('bver-vllm-0.8.5', 'backend-vllm', '0.8.5', 'vLLM 0.8.5', 1,
		'["vllm","serve"]',
		'["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}","--max-model-len","{{max_model_len}}","--gpu-memory-utilization","{{gpu_memory_utilization}}"]',
		'["--enforce-eager"]',
		'[{"name":"max_model_len","cli_name":"--max-model-len","type":"integer","default":8192,"required":false},{"name":"gpu_memory_utilization","cli_name":"--gpu-memory-utilization","type":"number","default":0.9,"required":false},{"name":"served_model_name","cli_name":"--served-model-name","type":"string","required":true},{"name":"tensor_parallel_size","cli_name":"--tensor-parallel-size","type":"integer","default":1,"required":false}]',
		'{"path":"/v1/models","expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}',
		8000,
		'{"nvidia":"vllm/vllm-openai:v0.8.5"}',
		'{"VLLM_USE_MODELSCOPE":"true"}', 0)`)

	db.Exec(`INSERT OR IGNORE INTO backend_versions (id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated)
		VALUES ('bver-vllm-0.10.0', 'backend-vllm', '0.10.0', 'vLLM 0.10.0', 0,
		'["vllm","serve"]',
		'["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}","--max-model-len","{{max_model_len}}","--gpu-memory-utilization","{{gpu_memory_utilization}}"]',
		'[]',
		'[{"name":"max_model_len","cli_name":"--max-model-len","type":"integer","default":8192,"required":false},{"name":"gpu_memory_utilization","cli_name":"--gpu-memory-utilization","type":"number","default":0.9,"required":false},{"name":"served_model_name","cli_name":"--served-model-name","type":"string","required":true},{"name":"tensor_parallel_size","cli_name":"--tensor-parallel-size","type":"integer","default":1,"required":false}]',
		'{"path":"/v1/models","expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}',
		8000,
		'{"nvidia":"vllm/vllm-openai:v0.10.0"}',
		'{}', 0)`)

	// Seed backend versions for SGLang
	db.Exec(`INSERT OR IGNORE INTO backend_versions (id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated)
		VALUES ('bver-sglang-0.4.6', 'backend-sglang', '0.4.6', 'SGLang 0.4.6', 1,
		'["python","-m","sglang.launch_server"]',
		'["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}"]',
		'[]',
		'[{"name":"served_model_name","cli_name":"--served-model-name","type":"string","required":true},{"name":"tp","cli_name":"--tp","type":"integer","default":1,"required":false}]',
		'{"path":"/v1/models","expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}',
		30000,
		'{"nvidia":"lmsysorg/sglang:v0.4.6"}',
		'{}', 0)`)

	db.Exec(`INSERT OR IGNORE INTO backend_versions (id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated)
		VALUES ('bver-sglang-0.5.0', 'backend-sglang', '0.5.0', 'SGLang 0.5.0', 0,
		'["python","-m","sglang.launch_server"]',
		'["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}"]',
		'[]',
		'[{"name":"served_model_name","cli_name":"--served-model-name","type":"string","required":true},{"name":"tp","cli_name":"--tp","type":"integer","default":1,"required":false}]',
		'{"path":"/v1/models","expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}',
		30000,
		'{"nvidia":"lmsysorg/sglang:v0.5.0"}',
		'{}', 0)`)

	// Seed backend versions for llama.cpp
	db.Exec(`INSERT OR IGNORE INTO backend_versions (id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated)
		VALUES ('bver-llamacpp-b4817', 'backend-llamacpp', 'b4817', 'llama.cpp b4817', 1,
		'[]',
		'["-m","{{model_container_file}}","--host","0.0.0.0","--port","{{container_port}}","-ngl","99"]',
		'[]',
		'[{"name":"ngl","cli_name":"-ngl","type":"integer","default":99,"required":false},{"name":"ctx_size","cli_name":"--ctx-size","type":"integer","default":4096,"required":false}]',
		'{"path":"/health","expectedStatus":200,"startupTimeoutSeconds":60,"intervalSeconds":2,"timeoutSeconds":5}',
		8080,
		'{"nvidia":"ghcr.io/ggerganov/llama.cpp:server-b4817"}',
		'{}', 0)`)
}

// migrateV14 adds model_browser_extra_roots to nodes and schema v14.
func (db *DB) migrateV14() error {
	for _, col := range []struct{ name, sqlType string }{
		{"model_browser_extra_roots", "TEXT NOT NULL DEFAULT '[]'"},
	} {
		if _, err := db.Exec("ALTER TABLE nodes ADD COLUMN " + col.name + " " + col.sqlType); err != nil {
			// Column may already exist.
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (14, 'V14: model browser dynamic roots')`); err != nil {
		return err
	}
	return nil
}

// migrateV15 adds first-class node model root policy rows.
func (db *DB) migrateV15() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS node_model_roots (
		id TEXT PRIMARY KEY,
		node_id TEXT NOT NULL REFERENCES nodes(id),
		path TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'enabled',
		source TEXT NOT NULL DEFAULT 'user',
		description TEXT NOT NULL DEFAULT '',
		created_by TEXT NOT NULL DEFAULT '',
		tenant_id TEXT NOT NULL DEFAULT '',
		last_checked_at TEXT,
		last_error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(node_id, path)
	)`); err != nil {
		return err
	}
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_node_model_roots_node ON node_model_roots(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_node_model_roots_status ON node_model_roots(status)`)
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (15, 'V15: node model roots policy')`); err != nil {
		return err
	}
	return nil
}

func (db *DB) seedTargetBackendCatalog() {
	now := time.Now().Format(time.RFC3339)
	db.normalizeLegacyBackendCatalogIDs()
	backends := []struct {
		id, slug, name, display, formats, protocols string
	}{
		{"backend.llamacpp", "llamacpp", "llamacpp", "llama.cpp", `["gguf"]`, `{"type":"openai-compatible","modelsPath":"/v1/models"}`},
		{"backend.vllm", "vllm", "vllm", "vLLM", `["huggingface","safetensors"]`, `{"type":"openai-compatible","modelsPath":"/v1/models"}`},
		{"backend.sglang", "sglang", "sglang", "SGLang", `["huggingface","safetensors"]`, `{"type":"openai-compatible","modelsPath":"/v1/models"}`},
		{"backend.ollama", "ollama", "ollama", "Ollama", `["ollama"]`, `{"type":"ollama","modelsPath":"/api/tags"}`},
	}
	for _, b := range backends {
		db.Exec(`INSERT OR IGNORE INTO inference_backends
			(id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled, slug, managed_by, source, catalog_version, checksum, status, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			b.id, b.name, b.display, b.display+" inference backend", b.protocols, "latest", "space", "[]", "{}", 1, 1,
			b.slug, "system", "embedded", "v1", catalogChecksum(b.id+b.formats+b.protocols), "active", now, now)
		db.Exec(`UPDATE inference_backends
			SET slug = ?, managed_by = 'system', source = 'embedded', catalog_version = 'v1',
				checksum = ?, status = 'active', is_builtin = 1, is_enabled = 1, updated_at = ?
			WHERE id = ?`,
			b.slug, catalogChecksum(b.id+b.formats+b.protocols), now, b.id)
	}

	versions := []struct {
		id, backendID, slug, version, display, protocol, entrypoint, args, params, paramDefs, hc string
		port                                                                                     int
		images, imageCandidates, env, endpoints, caps, mount, refs, revision                     string
		isDefault                                                                                int
	}{
		{"vllm-v0.23.0", "backend.vllm", "vllm-v0-23-0", "v0.23.0", "vLLM v0.23.0", "openai-compatible", `["vllm","serve"]`, `["{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","--served-model-name","{{served_model_name}}"]`, `[]`, `[{"name":"--model","required":false,"value":"{{MODEL_CONTAINER_PATH}}"},{"name":"--host","default":"0.0.0.0"},{"name":"--port","default":"8000"},{"name":"--served-model-name","optional":true},{"name":"--tensor-parallel-size","optional":true},{"name":"--max-model-len","optional":true},{"name":"--gpu-memory-utilization","optional":true},{"name":"--enforce-eager","optional":true},{"name":"--trust-remote-code","optional":true}]`, `{"type":"http","path":"/v1/models","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}`, 8000, `{"default":"vllm/vllm-openai:v0.23.0"}`, `["vllm/vllm-openai:v0.23.0","vllm/vllm-openai:v0.23.0-cu129-ubuntu2404","vllm/vllm-openai:latest"]`, `{}`, `{"models":"/v1/models","chat_completions":"/v1/chat/completions","completions":"/v1/completions","embeddings":"/v1/embeddings"}`, `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`, `{"container_path":"/models","readonly":true}`, `["vLLM official Docker docs: vllm/vllm-openai runs OpenAI-compatible server.","vLLM online serving docs: supports /v1/completions, /v1/chat/completions, /v1/embeddings.","vLLM v0.23.0 release line used as current system version baseline."]`, "v0.23.0", 1},
		{"sglang-v0.5.12.post1", "backend.sglang", "sglang-v0-5-12-post1", "v0.5.12.post1", "SGLang v0.5.12.post1", "openai-compatible", `["sglang","serve"]`, `["--model-path","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]`, `[]`, `[{"name":"--model-path","required":true,"value":"{{MODEL_CONTAINER_PATH}}"},{"name":"--host","default":"0.0.0.0"},{"name":"--port","default":"30000"},{"name":"--tp","optional":true},{"name":"--tensor-parallel-size","optional":true},{"name":"--dp","optional":true},{"name":"--enable-metrics","optional":true},{"name":"--log-level","optional":true},{"name":"--served-model-name","optional":true},{"name":"--mem-fraction-static","optional":true},{"name":"--context-length","optional":true},{"name":"--disable-cuda-graph","optional":true}]`, `{"type":"http","path":"/v1/models","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}`, 30000, `{"default":"lmsysorg/sglang:v0.5.12.post1"}`, `["lmsysorg/sglang:v0.5.12.post1","lmsysorg/sglang:latest-runtime","lmsysorg/sglang:latest","lmsysorg/sglang:v0.5.13.post1-cu129-runtime","lmsysorg/sglang:v0.5.13.post1-cu130-runtime"]`, `{}`, `{"models":"/v1/models","chat_completions":"/v1/chat/completions","completions":"/v1/completions","embeddings":"/v1/embeddings"}`, `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`, `{"container_path":"/models","readonly":true}`, `["SGLang install docs: Docker images are lmsysorg/sglang and lmsysorg/sglang:latest-runtime.","SGLang launch command: python3 -m sglang.launch_server --model-path ... --host ... --port ...","SGLang OpenAI API docs: supports chat/completions and completions.","SGLang server arguments docs: server args are available from python3 -m sglang.launch_server --help."]`, "v0.5.12.post1", 1},
		{"sglang-v0.5.13.post1", "backend.sglang", "sglang-v0-5-13-post1", "v0.5.13.post1", "SGLang v0.5.13.post1", "openai-compatible", `["sglang","serve"]`, `["--model-path","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]`, `[]`, `[{"name":"--model-path","required":true,"value":"{{MODEL_CONTAINER_PATH}}"},{"name":"--host","default":"0.0.0.0"},{"name":"--port","default":"30000"},{"name":"--tp","optional":true},{"name":"--tensor-parallel-size","optional":true},{"name":"--dp","optional":true},{"name":"--enable-metrics","optional":true},{"name":"--log-level","optional":true},{"name":"--served-model-name","optional":true},{"name":"--mem-fraction-static","optional":true},{"name":"--context-length","optional":true},{"name":"--disable-cuda-graph","optional":true}]`, `{"type":"http","path":"/v1/models","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}`, 30000, `{"default":"lmsysorg/sglang:v0.5.13.post1-cu129-runtime"}`, `["lmsysorg/sglang:v0.5.13.post1-cu129-runtime","lmsysorg/sglang:v0.5.13.post1-cu130-runtime","lmsysorg/sglang:latest-runtime","lmsysorg/sglang:latest"]`, `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`, `{"models":"/v1/models","chat_completions":"/v1/chat/completions","completions":"/v1/completions","embeddings":"/v1/embeddings"}`, `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`, `{"container_path":"/models","readonly":true}`, `["SGLang v0.5.13.post1 tag verified with git ls-remote against github.com/sgl-project/sglang.git.","SGLang install docs: Docker images are lmsysorg/sglang and lmsysorg/sglang:latest-runtime."]`, "v0.5.13.post1", 0},
		{"sglang-0.4.6-compatible", "backend.sglang", "sglang-0-4-6-compatible", "0.4.6-compatible", "SGLang 0.4.6-compatible", "openai-compatible", `["sglang","serve"]`, `["--model-path","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}"]`, `[]`, `[{"name":"--model-path","required":true,"value":"{{MODEL_CONTAINER_PATH}}"},{"name":"--host","default":"0.0.0.0"},{"name":"--port","default":"30000"},{"name":"--tp","optional":true},{"name":"--dp","optional":true},{"name":"--dist-init-addr","optional":true},{"name":"--nnodes","optional":true},{"name":"--node-rank","optional":true},{"name":"--trust-remote-code","optional":true},{"name":"--attention-backend","optional":true},{"name":"--enable-dp-attention","optional":true},{"name":"--enable-ep-moe","optional":true}]`, `{"type":"http","path":"/v1/models","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":120,"intervalSeconds":2,"timeoutSeconds":5}`, 30000, `{}`, `[]`, `{}`, `{"models":"/v1/models","chat_completions":"/v1/chat/completions","completions":"/v1/completions"}`, `["models","chat_completions","completions","openai_compatible"]`, `{"container_path":"/models","readonly":true}`, `["SGLang 0.4.6-compatible is a vendor abstraction layer for MacaRT-SGLang and similar distributions.","MacaRT-SGLang wraps SGLang 0.4.6 with MetaX MXMACA acceleration.","Use BackendRuntime (not BackendVersion) to configure MacaRT-SGLang hardware/runtime parameters."]`, "0.4.6-compatible", 0},
		{"llamacpp-b9700", "backend.llamacpp", "llamacpp-b9700", "b9700", "llama.cpp b9700", "openai-compatible-subset", `[]`, `["-m","{{model_container_file}}","--host","0.0.0.0","--port","{{container_port}}"]`, `[]`, `[{"name":"-m","alias":"--model","required":true,"value":"{{MODEL_CONTAINER_PATH}}"},{"name":"--host","default":"0.0.0.0"},{"name":"--port","default":"8080"},{"name":"--ctx-size","alias":"-c","optional":true},{"name":"--n-gpu-layers","alias":"-ngl","optional":true},{"name":"--threads","alias":"-t","optional":true},{"name":"--threads-batch","alias":"-tb","optional":true}]`, `{"type":"http","path":"/v1/models","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":60,"intervalSeconds":2,"timeoutSeconds":5}`, 8080, `{"default":"ghcr.io/ggml-org/llama.cpp:server"}`, `["ghcr.io/ggml-org/llama.cpp:server","ghcr.io/ggml-org/llama.cpp:server-cuda","ghcr.io/ggml-org/llama.cpp:server-cuda13"]`, `{"supported_formats":["gguf"],"supported_tasks":["chat","completion"],"supported_capabilities":["chat","completion"],"model_path_modes":["file"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions"}}`, `{"models":"/v1/models","chat_completions":"/v1/chat/completions","completions":"/v1/completions","embeddings":"/v1/embeddings"}`, `{"supported_formats":["gguf"],"supported_tasks":["chat","completion"],"supported_capabilities":["chat","completion"],"model_path_modes":["file"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions"}}`, `{"container_path":"/models","readonly":true}`, `["llama.cpp tools/server README: llama-server is HTTP server with REST APIs and Web UI.","llama.cpp server README: supports OpenAI API-compatible chat completions, responses, embeddings routes.","llama.cpp Docker docs: ghcr.io/ggml-org/llama.cpp:server contains llama-server only.","llama.cpp releases use build tags such as b9700 rather than stable semver."]`, "b9700", 1},
		{"backend-version.ollama.latest", "backend.ollama", "ollama-latest", "latest", "Ollama Latest", "ollama", `["ollama","serve"]`, `[]`, `[]`, `[]`, `{"type":"http","path":"/api/tags","success_status":[200],"expectedStatus":200,"startupTimeoutSeconds":60,"intervalSeconds":2,"timeoutSeconds":5}`, 11434, `{"default":"ollama/ollama:latest"}`, `["ollama/ollama:latest"]`, `{}`, `{"models":"/api/tags"}`, `["ollama"]`, `{}`, `[]`, "latest", 1},
	}
	for _, v := range versions {
		db.Exec(`INSERT OR IGNORE INTO backend_versions
			(id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, capabilities_json, docker_options_json, model_mount_json, vendor_options_json, readonly, protocol, image_candidates_json, default_host, default_endpoints_json, default_args_schema_json, default_env_schema_json, default_health_check_json, official_reference_json, revision, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			v.id, v.backendID, v.version, v.display, v.isDefault, v.entrypoint, v.args, v.params, v.paramDefs, v.hc, v.port, v.images, v.env, 0,
			v.slug, "system", "embedded", "v1", catalogChecksum(v.id+v.args+v.images), "active", v.display+" system software version", v.caps, "{}", v.mount, "{}", 1, v.protocol, v.imageCandidates, "0.0.0.0", v.endpoints, v.paramDefs, "[]", v.hc, v.refs, v.revision, now, now)
		db.Exec(`UPDATE backend_versions SET
			version=?, display_name=?, is_default=?, default_entrypoint_json=?, default_args_json=?, default_backend_params_json=?,
			parameter_defs_json=?, health_check_json=?, default_container_port=?, default_images_json=?, env_json=?, is_deprecated=0,
			slug=?, managed_by='system', source='embedded', catalog_version='v1', checksum=?, status='active',
			description=?, capabilities_json=?, docker_options_json='{}', model_mount_json=?, vendor_options_json='{}',
			readonly=1, protocol=?, image_candidates_json=?, default_host='0.0.0.0', default_endpoints_json=?,
			default_args_schema_json=?, default_env_schema_json='[]', default_health_check_json=?, official_reference_json=?, revision=?, updated_at=?
			WHERE id=?`,
			v.version, v.display, v.isDefault, v.entrypoint, v.args, v.params, v.paramDefs, v.hc, v.port, v.images, v.env,
			v.slug, catalogChecksum(v.id+v.args+v.images), v.display+" system software version", v.caps, v.mount, v.protocol,
			v.imageCandidates, v.endpoints, v.paramDefs, v.hc, v.refs, v.revision, now, v.id)
	}
	db.Exec(`UPDATE backend_versions SET status='deprecated', is_deprecated=1
		WHERE id IN (
			'bver-vllm-0.8.5','bver-vllm-0.10.0','bver-sglang-0.4.6','bver-sglang-0.5.0','bver-llamacpp-b4817',
			'backend-version.vllm.openai-latest','backend-version.vllm.openai-0.9',
			'backend-version.sglang.openai-latest','backend-version.llamacpp.server','backend-version.llamacpp.server-metax'
		)`)

	type runtimeSeed struct {
		id, name, display, backendID, versionID, slug, vendor, image, env, docker, mount, verification string
	}
	runtimes := []runtimeSeed{
		{"runtime.vllm.nvidia-docker", "vllm-nvidia-docker", "vLLM NVIDIA Docker", "backend.vllm", "vllm-v0.23.0", "vllm-nvidia-docker", "nvidia", "vllm/vllm-openai:v0.23.0", `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}`, `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES","privileged":false,"ipc_mode":"host","shm_size":"16gb","gpu_driver":"","gpu_capabilities":[["gpu"]]}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
		{"runtime.vllm.metax-docker", "vllm-metax-docker", "vLLM MetaX Docker", "backend.vllm", "vllm-v0.23.0", "vllm-metax-docker", "metax", "vllm/vllm-openai:v0.23.0", `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}","MACA_SMALL_PAGESIZE_ENABLE":"1","PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM":"1"}`, `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES","privileged":true,"ipc_mode":"host","uts_mode":"host","shm_size":"100gb","devices":[{"host_path":"/dev/dri","container_path":"/dev/dri"},{"host_path":"/dev/mxcd","container_path":"/dev/mxcd"},{"host_path":"/dev/infiniband","container_path":"/dev/infiniband"}],"group_add":["video"],"security_options":["seccomp=unconfined","apparmor=unconfined"],"ulimits":{"memlock":"-1"}}`, `{"container_path":"/models","readonly":true}`, `{"status":"requires_hardware_validation","optional_high_risk_devices":["/dev/mem"]}`},
		{"runtime.vllm.huawei-docker", "vllm-huawei-docker", "vLLM Huawei Docker", "backend.vllm", "vllm-v0.23.0", "vllm-huawei-docker", "huawei", "vllm/vllm-openai:v0.23.0", `{"ASCEND_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}`, `{"gpu_visible_env_key":"ASCEND_VISIBLE_DEVICES","devices":[{"host_path":"/dev/davinci_manager","container_path":"/dev/davinci_manager"},{"host_path":"/dev/devmm_svm","container_path":"/dev/devmm_svm"},{"host_path":"/dev/hisi_hdc","container_path":"/dev/hisi_hdc"},{"host_path":"/usr/local/dcmi","container_path":"/usr/local/dcmi"},{"host_path":"/usr/local/bin/npu-smi","container_path":"/usr/local/bin/npu-smi"},{"host_path":"/usr/local/Ascend/driver/lib64","container_path":"/usr/local/Ascend/driver/lib64"},{"host_path":"/usr/local/Ascend/driver/version.info","container_path":"/usr/local/Ascend/driver/version.info"},{"host_path":"/etc/ascend_install.info","container_path":"/etc/ascend_install.info"}]}`, `{"container_path":"/models","readonly":true}`, `{"status":"template_only"}`},
		{"runtime.llamacpp.nvidia-docker", "llamacpp-nvidia-docker", "llama.cpp NVIDIA Docker", "backend.llamacpp", "llamacpp-b9700", "llamacpp-nvidia-docker", "nvidia", "ghcr.io/ggml-org/llama.cpp:server-cuda13", `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}`, `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES","ipc_mode":"host","shm_size":"8gb","gpu_driver":"","gpu_capabilities":[["gpu"]]}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
		{"runtime.llamacpp.cpu-docker", "llamacpp-cpu-docker", "llama.cpp CPU Docker", "backend.llamacpp", "llamacpp-b9700", "llamacpp-cpu-docker", "cpu", "ghcr.io/ggml-org/llama.cpp:server", `{}`, `{}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
		{"runtime.sglang.nvidia-docker", "sglang-nvidia-docker", "SGLang NVIDIA Docker", "backend.sglang", "sglang-v0.5.13.post1", "sglang-nvidia-docker", "nvidia", "lmsysorg/sglang:v0.5.13.post1-cu129-runtime", `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}`, `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES","ipc_mode":"host","shm_size":"32gb","gpu_driver":"","gpu_capabilities":[["gpu"]]}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
		{"runtime.ollama.nvidia-docker", "ollama-nvidia-docker", "Ollama NVIDIA Docker", "backend.ollama", "backend-version.ollama.latest", "ollama-nvidia-docker", "nvidia", "ollama/ollama:latest", `{"CUDA_VISIBLE_DEVICES":"{{vendor_visible_devices}}"}`, `{"gpu_visible_env_key":"CUDA_VISIBLE_DEVICES"}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
		{"runtime.ollama.cpu-docker", "ollama-cpu-docker", "Ollama CPU Docker", "backend.ollama", "backend-version.ollama.latest", "ollama-cpu-docker", "cpu", "ollama/ollama:latest", `{}`, `{}`, `{"container_path":"/models","readonly":true}`, `{"status":"verified"}`},
	}
	for _, rt := range runtimes {
		db.Exec(`INSERT OR IGNORE INTO backend_runtimes
			(id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, verification_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			rt.id, rt.name, rt.display, rt.backendID, rt.versionID, rt.slug, rt.vendor, "docker", rt.image, "if_not_present", "[]", "[]", rt.env, rt.docker, rt.mount, "{}", 1, 0, "",
			rt.slug, "system", "embedded", "v1", catalogChecksum(rt.id+rt.docker+rt.image), "active", rt.verification, now, now)
	}
}

// migrateV20 adds hardware/runtime catalog columns to backend_runtimes
// for the BackendRuntime layered catalog design.
func (db *DB) migrateV20() error {
	alterStatements := []string{
		`ALTER TABLE backend_runtimes ADD COLUMN hardware_family TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN accelerator_api TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN runtime_distribution TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN runtime_distribution_version TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN compatibility_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_runtimes ADD COLUMN image_candidates_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_runtimes ADD COLUMN image_note TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN devices_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_runtimes ADD COLUMN volumes_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_runtimes ADD COLUMN env_schema_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_runtimes ADD COLUMN args_schema_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_runtimes ADD COLUMN ports_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_runtimes ADD COLUMN high_risk_flags_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_runtimes ADD COLUMN config_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN loaded_from TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN loaded_at TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (20, 'V20: backend runtime catalog hardware/runtime columns')`); err != nil {
		return err
	}
	return nil
}

// migrateV21 adds a user-visible name to NodeBackendRuntime records.
func (db *DB) migrateV21() error {
	if _, err := db.Exec(`ALTER TABLE node_backend_runtimes ADD COLUMN display_name TEXT NOT NULL DEFAULT ''`); err != nil {
		// Column may already exist from a partially applied development DB.
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (21, 'V21: node backend runtime display names')`); err != nil {
		return err
	}
	return nil
}

// migrateV22 adds config_snapshot_json to model_deployments so each deployment
// captures a frozen copy of the runtime configuration at creation time.
// This decouples deployments from future BackendRuntime / NodeBackendRuntime edits.
func (db *DB) migrateV22() error {
	_, err := db.Exec(`ALTER TABLE model_deployments ADD COLUMN config_snapshot_json TEXT NOT NULL DEFAULT '{}'`)
	if err != nil {
		// Column may already exist from a partially applied development DB;
		// log the error but don't fail the migration.
		log.Warn("db.migrateV22 add column warning", "error", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (22, 'V22: deployment config snapshot for RunPlan consistency')`); err != nil {
		return err
	}
	return nil
}

// migrateV23 adds source template metadata columns to model_deployments
// so deployments can trace their origin and support manual template sync.
func (db *DB) migrateV23() error {
	cols := []string{
		`ALTER TABLE model_deployments ADD COLUMN source_backend_runtime_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE model_deployments ADD COLUMN source_node_backend_runtime_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE model_deployments ADD COLUMN source_template_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE model_deployments ADD COLUMN source_template_version TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE model_deployments ADD COLUMN source_config_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE model_deployments ADD COLUMN copied_at TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range cols {
		if _, err := db.Exec(stmt); err != nil {
			log.Warn("db.migrateV23 add column warning", "error", err)
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (23, 'V23: deployment source template metadata for manual sync')`); err != nil {
		return err
	}
	return nil
}

// migrateV24 adds probe_results_json to node_backend_runtimes so each
// NodeBackendRuntime stores the full Image Capability Probe evidence
// (image existence, Docker inspect metadata, backend type match, version probe).
func (db *DB) migrateV24() error {
	if _, err := db.Exec(`ALTER TABLE node_backend_runtimes ADD COLUMN probe_results_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
		// Column may already exist from a partially applied development DB.
		log.Warn("db.migrateV24 add column warning", "error", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (24, 'V24: node_backend_runtimes image capability probe results')`); err != nil {
		return err
	}
	return nil
}

// migrateV25 adds model capability persistence columns to model_artifacts
// (Phase 2: capabilities_json, capability_sources_json, default_test_mode).
func (db *DB) migrateV25() error {
	alterStatements := []string{
		`ALTER TABLE model_artifacts ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE model_artifacts ADD COLUMN capability_sources_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE model_artifacts ADD COLUMN default_test_mode TEXT NOT NULL DEFAULT 'auto'`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Column may already exist from a partially applied development DB.
			log.Warn("db.migrateV25 add column warning", "error", err)
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (25, 'V25: model_artifacts capability persistence columns')`); err != nil {
		return err
	}
	return nil
}

// migrateV26 force-updates llama.cpp backend version and frozen NBR/deployment
// config snapshots to use {{model_container_file}} instead of {{model_container_path}}
// for the -m flag. Previous seed-only fixes were ignored by INSERT OR IGNORE on
// existing DB rows, and frozen NodeBackendRuntime snapshots continued to inject
// the old variable (WEB-AI-RC-005).
func (db *DB) migrateV26() error {
	// 1. Update backend_versions.default_args_json for llama.cpp.
	if _, err := db.Exec(`UPDATE backend_versions SET default_args_json = REPLACE(default_args_json, '"{{model_container_path}}"', '"{{model_container_file}}"'), updated_at = datetime('now') WHERE default_args_json LIKE '%{{model_container_path}}%' AND id LIKE '%llama%'`); err != nil {
		log.Warn("db.migrateV26 update backend_versions warning", "error", err)
	}
	// Also fix uppercase variant.
	if _, err := db.Exec(`UPDATE backend_versions SET default_args_json = REPLACE(default_args_json, '"{{MODEL_CONTAINER_PATH}}"', '"{{MODEL_CONTAINER_FILE}}"'), updated_at = datetime('now') WHERE default_args_json LIKE '%{{MODEL_CONTAINER_PATH}}%' AND id LIKE '%llama%'`); err != nil {
		log.Warn("db.migrateV26 update backend_versions uppercase warning", "error", err)
	}

	// 2. Update node_backend_runtimes.config_snapshot_json (frozen at NBR creation).
	if _, err := db.Exec(`UPDATE node_backend_runtimes SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{MODEL_CONTAINER_PATH}}"', '"{{MODEL_CONTAINER_FILE}}"'), updated_at = datetime('now') WHERE config_snapshot_json LIKE '%{{MODEL_CONTAINER_PATH}}%' AND (backend_runtime_id LIKE '%llama%' OR config_snapshot_json LIKE '%llama%')`); err != nil {
		log.Warn("db.migrateV26 update nbr snapshot warning", "error", err)
	}
	if _, err := db.Exec(`UPDATE node_backend_runtimes SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{model_container_path}}"', '"{{model_container_file}}"'), updated_at = datetime('now') WHERE config_snapshot_json LIKE '%{{model_container_path}}%' AND (backend_runtime_id LIKE '%llama%' OR config_snapshot_json LIKE '%llama%')`); err != nil {
		log.Warn("db.migrateV26 update nbr snapshot lowercase warning", "error", err)
	}

	// 3. Update model_deployments.config_snapshot_json (frozen at deployment creation).
	if _, err := db.Exec(`UPDATE model_deployments SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{MODEL_CONTAINER_PATH}}"', '"{{MODEL_CONTAINER_FILE}}"'), updated_at = datetime('now') WHERE config_snapshot_json LIKE '%{{MODEL_CONTAINER_PATH}}%' AND (config_snapshot_json LIKE '%llama%' OR config_snapshot_json LIKE '%llamacpp%')`); err != nil {
		log.Warn("db.migrateV26 update deployment snapshot warning", "error", err)
	}
	if _, err := db.Exec(`UPDATE model_deployments SET config_snapshot_json = REPLACE(config_snapshot_json, '"{{model_container_path}}"', '"{{model_container_file}}"'), updated_at = datetime('now') WHERE config_snapshot_json LIKE '%{{model_container_path}}%' AND (config_snapshot_json LIKE '%llama%' OR config_snapshot_json LIKE '%llamacpp%')`); err != nil {
		log.Warn("db.migrateV26 update deployment snapshot lowercase warning", "error", err)
	}

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (26, 'V26: force-update llama.cpp -m to use model_container_file in existing DB records')`); err != nil {
		return err
	}
	return nil
}

// migrateV27 force-updates built-in backend_versions.capabilities_json
// to the current capability contract (Batch 3: Phase D).
// repairBackendCapabilitiesV27 runs unconditionally after every seed to ensure
// backend_versions.capabilities_json has the current structured capability contract.
func (db *DB) repairBackendCapabilitiesV27() {
	caps := map[string]string{
		"vllm-v0.23.0":            `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-v0.5.13.post1":    `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-v0.5.12.post1":    `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-0.4.6-compatible": `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"bver-vllm-0.8.5":         `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"bver-vllm-0.10.0":        `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"llamacpp-b9700":          `{"supported_formats":["gguf"],"supported_tasks":["chat","completion"],"supported_capabilities":["chat","completion"],"model_path_modes":["file"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions"}}`,
	}
	for id, capsJSON := range caps {
		if _, err := db.Exec(`UPDATE backend_versions SET capabilities_json = ?, updated_at = datetime('now') WHERE id = ?`, capsJSON, id); err != nil {
			log.Warn("db.repairBackendCapabilitiesV27 warning", "error", err, "id", id)
		}
	}
}

func (db *DB) migrateV27() error {
	caps := map[string]string{
		"vllm-v0.23.0":            `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-v0.5.13.post1":    `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-v0.5.12.post1":    `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"sglang-0.4.6-compatible": `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 SGLang runtime 未验证 InternVL2.5 兼容性。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"bver-vllm-0.8.5":         `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"bver-vllm-0.10.0":        `{"supported_formats":["huggingface","sentence_transformers"],"supported_tasks":["chat","completion","embedding","rerank","vision_chat"],"supported_capabilities":["chat","completion","embedding","rerank","vision"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions","embedding":"/v1/embeddings","rerank":"/v1/rerank"},"blocked_architectures":{"InternVLChatModel":"当前 vLLM runtime 无法加载 InternVL2.5 tokenizer（已安装 sentencepiece 但仍失败），该架构需要额外 runtime 依赖或更新的 vLLM 版本。建议使用 Chat/Embedding/Reranker 模型。"}}`,
		"llamacpp-b9700":          `{"supported_formats":["gguf"],"supported_tasks":["chat","completion"],"supported_capabilities":["chat","completion"],"model_path_modes":["file"],"test_endpoints":{"chat":"/v1/chat/completions","completion":"/v1/completions"}}`,
	}
	for id, capsJSON := range caps {
		if _, err := db.Exec(`UPDATE backend_versions SET capabilities_json = ?, updated_at = datetime('now') WHERE id = ?`, capsJSON, id); err != nil {
			log.Warn("db.migrateV27 update capabilities_json warning", "error", err, "id", id)
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description) VALUES (27, 'V27: backend_versions capabilities_json contract')`); err != nil {
		return err
	}
	return nil
}

func (db *DB) normalizeLegacyBackendCatalogIDs() {
	type mapping struct {
		oldID, newID, name string
	}
	mappings := []mapping{
		{"backend-vllm", "backend.vllm", "vllm"},
		{"backend-sglang", "backend.sglang", "sglang"},
		{"backend-llamacpp", "backend.llamacpp", "llamacpp"},
	}
	db.Exec(`PRAGMA foreign_keys=OFF`)
	defer db.Exec(`PRAGMA foreign_keys=ON`)
	for _, m := range mappings {
		db.Exec(`UPDATE backend_versions SET backend_id = ? WHERE backend_id = ?`, m.newID, m.oldID)
		db.Exec(`UPDATE backend_runtimes SET backend_id = ? WHERE backend_id = ?`, m.newID, m.oldID)
		db.Exec(`UPDATE inference_backends SET id = ? WHERE id = ? AND name = ?`, m.newID, m.oldID, m.name)
	}
}

func catalogChecksum(s string) string {
	h := 0
	for _, r := range s {
		h = h*31 + int(r)
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("checksum:%08x", h)
}

// migrateV16 adds config_snapshot_json to node_backend_runtimes so each
// NodeBackendRuntime captures a frozen copy of the resolved runtime
// configuration at enable/check time, decoupling node runs from future
// BackendRuntime template edits.
func (db *DB) migrateV16() error {
	alterStatements := []string{
		`ALTER TABLE node_backend_runtimes ADD COLUMN config_snapshot_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE node_backend_runtimes ADD COLUMN source_runtime_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE node_backend_runtimes ADD COLUMN source_runtime_revision TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (16, 'V16: node_backend_runtimes config snapshot')`); err != nil {
		return err
	}
	return nil
}

// migrateV17 adds user-managed BackendVersion metadata and BackendRuntime
// source version snapshots. This preserves the rule that BackendVersion
// changes do not silently mutate existing BackendRuntime templates.
func (db *DB) migrateV17() error {
	alterStatements := []string{
		`ALTER TABLE backend_versions ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_versions ADD COLUMN docker_options_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_versions ADD COLUMN model_mount_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_versions ADD COLUMN vendor_options_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_runtimes ADD COLUMN source_backend_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN source_backend_version_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN source_version_revision TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_runtimes ADD COLUMN version_snapshot_json TEXT NOT NULL DEFAULT '{}'`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}
	db.Exec(`UPDATE backend_runtimes
		SET source_backend_id = CASE WHEN source_backend_id = '' THEN backend_id ELSE source_backend_id END,
		    source_backend_version_id = CASE WHEN source_backend_version_id = '' THEN backend_version_id ELSE source_backend_version_id END,
		    source_version_revision = CASE WHEN source_version_revision = '' THEN updated_at ELSE source_version_revision END`)
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (17, 'V17: backend version user catalog and runtime version snapshots')`); err != nil {
		return err
	}
	return nil
}

// migrateV18 expands BackendVersion to represent the software capability layer
// explicitly. Hardware, node, image presence, and readiness remain outside
// BackendVersion.
func (db *DB) migrateV18() error {
	alterStatements := []string{
		`ALTER TABLE backend_versions ADD COLUMN readonly INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE backend_versions ADD COLUMN protocol TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN image_candidates_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_versions ADD COLUMN default_host TEXT NOT NULL DEFAULT '0.0.0.0'`,
		`ALTER TABLE backend_versions ADD COLUMN default_endpoints_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_versions ADD COLUMN default_args_schema_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_versions ADD COLUMN default_env_schema_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_versions ADD COLUMN default_health_check_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backend_versions ADD COLUMN official_reference_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE backend_versions ADD COLUMN revision TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}
	db.Exec(`UPDATE backend_versions
		SET readonly = CASE WHEN managed_by = 'system' THEN 1 ELSE 0 END,
		    protocol = CASE WHEN protocol = '' THEN 'openai-compatible' ELSE protocol END,
		    image_candidates_json = CASE WHEN image_candidates_json = '[]' THEN
		      CASE WHEN default_images_json = '{}' THEN '[]' ELSE json_array(default_images_json) END
		      ELSE image_candidates_json END,
		    default_health_check_json = CASE WHEN default_health_check_json = '{}' THEN health_check_json ELSE default_health_check_json END,
		    revision = CASE WHEN revision = '' THEN checksum ELSE revision END`)
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (18, 'V18: backend version software catalog fields')`); err != nil {
		return err
	}
	return nil
}

func (db *DB) migrateV19() error {
	alterStatements := []string{
		`ALTER TABLE backend_versions ADD COLUMN config_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN loaded_from TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backend_versions ADD COLUMN loaded_at TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil {
			// Columns may already exist from a partially applied development DB.
		}
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_version (version, description)
		VALUES (19, 'V19: backend catalog file projection metadata')`); err != nil {
		return err
	}
	return nil
}
