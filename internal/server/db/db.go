// Package db provides SQLite database initialization and access.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/catalog"

	_ "github.com/mattn/go-sqlite3"
)

const configSetSchemaVersion = 100

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

// Migrate creates the current ConfigSet schema. This refactor intentionally
// does not migrate legacy DBs; delete/rebuild the DB for this breaking change.
func (db *DB) Migrate() error {
	start := time.Now()
	log.Info("db migrate: begin")
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now')),
		description TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current); err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}
	if current > 0 && current != configSetSchemaVersion {
		return fmt.Errorf("unsupported legacy database schema version %d; remove the DB and rebuild with ConfigSet schema version %d", current, configSetSchemaVersion)
	}
	if current == 0 {
		if err := db.createConfigSetSchema(); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO schema_version (version, description) VALUES (?, ?)`, configSetSchemaVersion, "ConfigSet fresh schema"); err != nil {
			return fmt.Errorf("record schema version: %w", err)
		}
	}
	if err := catalog.SeedCatalog(db.DB, "", ""); err != nil {
		return fmt.Errorf("seed configset catalog: %w", err)
	}
	log.Info("db migrate: completed", "duration_ms", time.Since(start).Milliseconds())
	return nil
}

// DefaultTenantID returns the UUID of the default tenant.
func (db *DB) DefaultTenantID() string {
	var id string
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'default'`).Scan(&id)
	return id
}

func (db *DB) createConfigSetSchema() error {
	schema := `
CREATE TABLE tenants (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	type TEXT NOT NULL DEFAULT 'infrastructure',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO tenants (id, slug, name, type, status)
VALUES ('a0000000-0000-0000-0000-000000000001', 'default', 'Default Tenant', 'infrastructure', 'active');

CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	is_platform_admin INTEGER NOT NULL DEFAULT 0,
	must_change_password INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE tenant_memberships (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	tenant_id TEXT NOT NULL REFERENCES tenants(id),
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(user_id, tenant_id)
);
CREATE TABLE roles (
	id TEXT PRIMARY KEY,
	tenant_id TEXT,
	name TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	scope TEXT NOT NULL DEFAULT 'tenant',
	built_in INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(tenant_id, name)
);
CREATE TABLE permissions (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	scope TEXT NOT NULL DEFAULT 'tenant',
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE role_permissions (
	id TEXT PRIMARY KEY,
	role_id TEXT NOT NULL REFERENCES roles(id),
	permission_id TEXT NOT NULL REFERENCES permissions(id),
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(role_id, permission_id)
);
CREATE TABLE tenant_membership_roles (
	id TEXT PRIMARY KEY,
	membership_id TEXT NOT NULL REFERENCES tenant_memberships(id),
	role_id TEXT NOT NULL REFERENCES roles(id),
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(membership_id, role_id)
);
CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	current_tenant_id TEXT NOT NULL REFERENCES tenants(id),
	csrf_secret_hash TEXT NOT NULL DEFAULT '',
	expires_at TEXT NOT NULL,
	revoked_at TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	last_seen_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE nodes (
	id TEXT PRIMARY KEY,
	agent_id TEXT NOT NULL DEFAULT '',
	hostname TEXT NOT NULL DEFAULT '',
	primary_ip TEXT NOT NULL DEFAULT '',
	advertised_address TEXT NOT NULL DEFAULT '',
	os TEXT NOT NULL DEFAULT '',
	arch TEXT NOT NULL DEFAULT '',
	kernel TEXT NOT NULL DEFAULT '',
	agent_version TEXT NOT NULL DEFAULT '',
	metrics_enabled INTEGER NOT NULL DEFAULT 0,
	metrics_scheme TEXT NOT NULL DEFAULT 'http',
	metrics_port INTEGER NOT NULL DEFAULT 9090,
	metrics_path TEXT NOT NULL DEFAULT '/metrics',
	status TEXT NOT NULL DEFAULT 'offline',
	model_browser_extra_roots TEXT NOT NULL DEFAULT '[]',
	last_heartbeat_at TEXT,
	tenant_id TEXT NOT NULL DEFAULT 'a0000000-0000-0000-0000-000000000001',
	owner_id TEXT,
	created_by TEXT NOT NULL DEFAULT 'system',
	updated_by TEXT NOT NULL DEFAULT 'system',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE gpu_devices (
	id TEXT PRIMARY KEY,
	node_id TEXT NOT NULL REFERENCES nodes(id),
	vendor TEXT NOT NULL DEFAULT '',
	index_num INTEGER NOT NULL DEFAULT 0,
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
	tenant_id TEXT NOT NULL DEFAULT 'a0000000-0000-0000-0000-000000000001',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE node_system_snapshots (
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
);
CREATE TABLE node_filesystem_snapshots (
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
);
CREATE TABLE node_network_snapshots (
	node_id TEXT NOT NULL,
	interface_name TEXT NOT NULL,
	up INTEGER NOT NULL DEFAULT 0,
	bytes_recv INTEGER NOT NULL DEFAULT 0,
	bytes_sent INTEGER NOT NULL DEFAULT 0,
	collected_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (node_id, interface_name)
);

CREATE TABLE inference_backends (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	display_name TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	slug TEXT NOT NULL DEFAULT '',
	managed_by TEXT NOT NULL DEFAULT 'system',
	source TEXT NOT NULL DEFAULT 'config-registry',
	catalog_version TEXT NOT NULL DEFAULT 'configset-v1',
	checksum TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	config_set_json TEXT NOT NULL,
	source_metadata_json TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE backend_versions (
	id TEXT PRIMARY KEY,
	backend_id TEXT NOT NULL REFERENCES inference_backends(id),
	version TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	is_default INTEGER NOT NULL DEFAULT 0,
	is_deprecated INTEGER NOT NULL DEFAULT 0,
	slug TEXT NOT NULL DEFAULT '',
	managed_by TEXT NOT NULL DEFAULT 'system',
	source TEXT NOT NULL DEFAULT 'config-registry',
	catalog_version TEXT NOT NULL DEFAULT 'configset-v1',
	checksum TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	description TEXT NOT NULL DEFAULT '',
	readonly INTEGER NOT NULL DEFAULT 1,
	protocol TEXT NOT NULL DEFAULT '',
	revision TEXT NOT NULL DEFAULT '',
	config_set_json TEXT NOT NULL,
	source_metadata_json TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(backend_id, version)
);
CREATE TABLE backend_runtimes (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	backend_id TEXT NOT NULL REFERENCES inference_backends(id),
	backend_version_id TEXT NOT NULL REFERENCES backend_versions(id),
	source_template_name TEXT NOT NULL DEFAULT '',
	vendor TEXT NOT NULL DEFAULT 'custom',
	runtime_type TEXT NOT NULL DEFAULT 'docker',
	is_builtin INTEGER NOT NULL DEFAULT 0,
	is_editable INTEGER NOT NULL DEFAULT 1,
	tenant_id TEXT NOT NULL DEFAULT '',
	slug TEXT NOT NULL DEFAULT '',
	managed_by TEXT NOT NULL DEFAULT 'system',
	source TEXT NOT NULL DEFAULT 'config-registry',
	catalog_version TEXT NOT NULL DEFAULT 'configset-v1',
	checksum TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	visibility TEXT NOT NULL DEFAULT 'visible',
	support_level TEXT NOT NULL DEFAULT 'documented',
	verification_json TEXT NOT NULL DEFAULT '{}',
	hardware_family TEXT NOT NULL DEFAULT '',
	accelerator_api TEXT NOT NULL DEFAULT '',
	runtime_distribution TEXT NOT NULL DEFAULT '',
	runtime_distribution_version TEXT NOT NULL DEFAULT '',
	config_hash TEXT NOT NULL DEFAULT '',
	loaded_from TEXT NOT NULL DEFAULT '',
	loaded_at TEXT NOT NULL DEFAULT '',
	config_set_json TEXT NOT NULL,
	source_metadata_json TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(tenant_id, name)
);
CREATE TABLE node_backend_runtimes (
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
	probe_results_json TEXT NOT NULL DEFAULT '{}',
	config_set_json TEXT NOT NULL,
	source_metadata_json TEXT NOT NULL,
	tenant_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(node_id, backend_runtime_id)
);
CREATE TABLE model_artifacts (
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
	capability_set_json TEXT NOT NULL DEFAULT '{}',
	default_test_mode TEXT NOT NULL DEFAULT 'auto',
	tenant_id TEXT NOT NULL,
	owner_id TEXT,
	created_by TEXT NOT NULL DEFAULT 'system',
	updated_by TEXT NOT NULL DEFAULT 'system',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE model_locations (
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
);
CREATE TABLE node_model_roots (
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
);
CREATE TABLE model_deployments (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	model_artifact_id TEXT NOT NULL REFERENCES model_artifacts(id),
	backend_runtime_id TEXT NOT NULL DEFAULT '',
	node_backend_runtime_id TEXT NOT NULL DEFAULT '',
	replicas INTEGER NOT NULL DEFAULT 1,
	placement_json TEXT NOT NULL DEFAULT '{}',
	service_json TEXT NOT NULL DEFAULT '{}',
	config_overrides_json TEXT NOT NULL DEFAULT '{}',
	source_backend_runtime_id TEXT NOT NULL DEFAULT '',
	source_node_backend_runtime_id TEXT NOT NULL DEFAULT '',
	source_template_name TEXT NOT NULL DEFAULT '',
	source_template_version TEXT NOT NULL DEFAULT '',
	source_config_hash TEXT NOT NULL DEFAULT '',
	copied_at TEXT NOT NULL DEFAULT '',
	config_set_json TEXT NOT NULL,
	source_metadata_json TEXT NOT NULL,
	desired_state TEXT NOT NULL DEFAULT 'stopped',
	status TEXT NOT NULL DEFAULT 'stopped',
	tenant_id TEXT NOT NULL,
	owner_id TEXT,
	created_by TEXT NOT NULL DEFAULT 'system',
	updated_by TEXT NOT NULL DEFAULT 'system',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE model_instances (
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
);
CREATE TABLE resolved_run_plans (
	id TEXT PRIMARY KEY,
	deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
	instance_id TEXT,
	tenant_id TEXT NOT NULL DEFAULT '',
	backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
	node_backend_runtime_id TEXT NOT NULL DEFAULT '',
	plan_json TEXT NOT NULL DEFAULT '{}',
	docker_preview TEXT NOT NULL DEFAULT '',
	input_hash TEXT NOT NULL DEFAULT '',
	plan_hash TEXT NOT NULL DEFAULT '',
	created_by TEXT NOT NULL DEFAULT 'system',
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE gpu_leases (
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
);
CREATE TABLE agent_tasks (
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
	lease_owner TEXT NOT NULL DEFAULT '',
	lease_expires_at TEXT NOT NULL DEFAULT '',
	operation_id TEXT NOT NULL DEFAULT '',
	generation INTEGER NOT NULL DEFAULT 0,
	attempt INTEGER NOT NULL DEFAULT 1,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE run_plan_groups (
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
);
CREATE TABLE audit_logs (
	id TEXT PRIMARY KEY,
	tenant_id TEXT NOT NULL DEFAULT '',
	actor_id TEXT NOT NULL DEFAULT '',
	operator_user_id TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL,
	resource_type TEXT NOT NULL DEFAULT '',
	resource_id TEXT NOT NULL DEFAULT '',
	entity_type TEXT NOT NULL DEFAULT '',
	entity_id TEXT NOT NULL DEFAULT '',
	result TEXT NOT NULL DEFAULT '',
	operation_id TEXT NOT NULL DEFAULT '',
	request_id TEXT NOT NULL DEFAULT '',
	detail TEXT NOT NULL DEFAULT '',
	message TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_nodes_tenant_id ON nodes(tenant_id);
CREATE INDEX idx_gpu_devices_node ON gpu_devices(node_id);
CREATE INDEX idx_model_artifacts_tenant ON model_artifacts(tenant_id);
CREATE INDEX idx_backend_runtimes_tenant ON backend_runtimes(tenant_id);
CREATE INDEX idx_node_backend_runtimes_node ON node_backend_runtimes(node_id);
CREATE INDEX idx_node_backend_runtimes_runtime ON node_backend_runtimes(backend_runtime_id);
CREATE INDEX idx_model_deployments_tenant ON model_deployments(tenant_id);
CREATE INDEX idx_model_deployments_status ON model_deployments(status);
CREATE INDEX idx_model_instances_deployment ON model_instances(deployment_id);
CREATE INDEX idx_model_instances_state ON model_instances(actual_state);
CREATE INDEX idx_resolved_run_plans_deployment ON resolved_run_plans(deployment_id);
CREATE INDEX idx_agent_tasks_status ON agent_tasks(status);
CREATE UNIQUE INDEX idx_gpu_leases_reserved_active ON gpu_leases(gpu_id) WHERE status IN ('reserved','active');
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create ConfigSet schema: %w", err)
	}
	return nil
}
