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
		'["-m","{{model_container_path}}","--host","0.0.0.0","--port","{{container_port}}","-ngl","99"]',
		'[]',
		'[{"name":"ngl","cli_name":"-ngl","type":"integer","default":99,"required":false},{"name":"ctx_size","cli_name":"--ctx-size","type":"integer","default":4096,"required":false}]',
		'{"path":"/health","expectedStatus":200,"startupTimeoutSeconds":60,"intervalSeconds":2,"timeoutSeconds":5}',
		8080,
		'{"nvidia":"ghcr.io/ggerganov/llama.cpp:server-b4817"}',
		'{}', 0)`)
}
