// Package config provides configuration loading for LightAI Go.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds all server configuration.
type ServerConfig struct {
	Host          string              `yaml:"host" json:"host"`
	Port          int                 `yaml:"port" json:"port"`
	LogLevel      string              `yaml:"log_level" json:"log_level"`
	DBPath        string              `yaml:"db_path" json:"db_path"`
	DevMode       bool                `yaml:"dev_mode" json:"dev_mode"`
	AgentToken    string              `yaml:"agent_token" json:"-"`
	Logging       LoggingConfig       `yaml:"logging" json:"logging"`
	Observability ObservabilityConfig `yaml:"observability" json:"observability"`
	GPU           GPUProfileConfig    `yaml:"gpu" json:"gpu"`
	// NodeOfflineThreshold is how long without heartbeat before marking offline.
	NodeOfflineThreshold time.Duration `yaml:"node_offline_threshold" json:"node_offline_threshold"`
}

// LoggingConfig holds file logging configuration.
type LoggingConfig struct {
	Level         string `yaml:"level" json:"level"`
	Dir           string `yaml:"dir" json:"dir"`
	File          string `yaml:"file" json:"file"`
	Stdout        bool   `yaml:"stdout" json:"stdout"`
	FileEnabled   bool   `yaml:"file_enabled" json:"file_enabled"`
	MaxSizeMB     int    `yaml:"max_size_mb" json:"max_size_mb"`
	MaxFiles      int    `yaml:"max_files" json:"max_files"`
	RetentionDays int    `yaml:"retention_days" json:"retention_days"`
}

// ObservabilityConfig holds observability settings.
type ObservabilityConfig struct {
	Mode string `yaml:"mode" json:"mode"`
}

// GPUProfileConfig holds GPU collector profile settings.
type GPUProfileConfig struct {
	Profile string `yaml:"profile" json:"profile"`
}

// AgentConfig holds all agent configuration.
type AgentConfig struct {
	AgentID        string               `yaml:"agent_id" json:"agent_id"`
	ServerURL      string               `yaml:"server_url" json:"server_url"`
	AgentToken     string               `yaml:"agent_token" json:"-"`
	LogLevel       string               `yaml:"log_level" json:"log_level"`
	DataDir        string               `yaml:"data_dir" json:"data_dir"`
	AdvertisedAddr string               `yaml:"advertised_address" json:"advertised_address"`
	RequestTimeout time.Duration        `yaml:"request_timeout" json:"request_timeout"`
	Metrics        AgentMetricsConfig   `yaml:"metrics" json:"metrics"`
	Heartbeat      HeartbeatConfig      `yaml:"heartbeat" json:"heartbeat"`
	Collectors     AgentCollectorConfig `yaml:"collectors" json:"collectors"`
	GPU            GPUProfileConfig     `yaml:"gpu" json:"gpu"`
	Logging        LoggingConfig        `yaml:"logging" json:"logging"`
}

// AgentMetricsConfig holds agent metrics server settings.
type AgentMetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Host    string `yaml:"host" json:"host"`
	Scheme  string `yaml:"scheme" json:"scheme"`
	Port    int    `yaml:"port" json:"port"`
	Path    string `yaml:"path" json:"path"`
}

// HeartbeatConfig holds heartbeat settings.
type HeartbeatConfig struct {
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// AgentCollectorConfig holds agent collector settings.
type AgentCollectorConfig struct {
	System         SystemCollectorConfig  `yaml:"system" json:"system"`
	MockGPU        MockGPUCollectorConfig `yaml:"mock_gpu" json:"mock_gpu"`
	Nvidia         NvidiaCollectorConfig  `yaml:"nvidia" json:"nvidia"`
	ReportInterval time.Duration          `yaml:"report_interval" json:"report_interval"`
}

// SystemCollectorConfig holds system collector settings.
type SystemCollectorConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// MockGPUCollectorConfig holds mock GPU collector settings.
type MockGPUCollectorConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// NvidiaCollectorConfig holds NVIDIA GPU collector settings.
type NvidiaCollectorConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// DefaultServerConfig returns a ServerConfig with safe defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:       "127.0.0.1",
		Port:       18080,
		LogLevel:   "info",
		DBPath:     "data/lightai.db",
		DevMode:    false,
		AgentToken: "lightai-agent-token-change-me",
		Logging: LoggingConfig{
			Level:         "info",
			Dir:           "logs",
			File:          "lightai-server.log",
			Stdout:        true,
			FileEnabled:   true,
			MaxSizeMB:     50,
			MaxFiles:      5,
			RetentionDays: 7,
		},
		Observability: ObservabilityConfig{
			Mode: "disabled",
		},
		GPU: GPUProfileConfig{
			Profile: "production",
		},
		NodeOfflineThreshold: 20 * time.Second,
	}
}

// DefaultAgentConfig returns an AgentConfig with safe defaults.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		ServerURL:      "http://127.0.0.1:18080",
		AgentToken:     "lightai-agent-token-change-me",
		LogLevel:       "info",
		DataDir:        "data",
		RequestTimeout: 5 * time.Second,
		Metrics: AgentMetricsConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Scheme:  "http",
			Port:    19091,
			Path:    "/metrics",
		},
		Heartbeat: HeartbeatConfig{
			Interval: 2 * time.Second,
		},
		Collectors: AgentCollectorConfig{
			ReportInterval: 5 * time.Second,
			System: SystemCollectorConfig{
				Enabled:  true,
				Interval: 5 * time.Second,
			},
			MockGPU: MockGPUCollectorConfig{
				Enabled: false,
			},
			Nvidia: NvidiaCollectorConfig{
				Enabled:  true,
				Interval: 5 * time.Second,
			},
		},
		GPU: GPUProfileConfig{
			Profile: "production",
		},
		Logging: LoggingConfig{
			Level:         "info",
			Dir:           "logs",
			File:          "lightai-agent.log",
			Stdout:        true,
			FileEnabled:   true,
			MaxSizeMB:     50,
			MaxFiles:      5,
			RetentionDays: 7,
		},
	}
}

// LoadServerConfig loads server config from a YAML file, with env var overrides.
func LoadServerConfig(path string) (*ServerConfig, error) {
	cfg := DefaultServerConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config file %s: %w", path, err)
		}
	}

	applyServerEnvOverrides(&cfg)
	return &cfg, nil
}

// LoadAgentConfig loads agent config from a YAML file, with env var overrides.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	cfg := DefaultAgentConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config file %s: %w", path, err)
		}
	}

	applyAgentEnvOverrides(&cfg)
	return &cfg, nil
}

func applyServerEnvOverrides(cfg *ServerConfig) {
	if v := os.Getenv("LIGHTAI_SERVER_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("LIGHTAI_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("LIGHTAI_SERVER_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LIGHTAI_SERVER_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("LIGHTAI_SERVER_DEV_MODE"); v == "true" {
		cfg.DevMode = true
	}
	if v := os.Getenv("LIGHTAI_AGENT_TOKEN"); v != "" {
		cfg.AgentToken = v
	}
}

func applyAgentEnvOverrides(cfg *AgentConfig) {
	if v := os.Getenv("LIGHTAI_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("LIGHTAI_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("LIGHTAI_AGENT_TOKEN"); v != "" {
		cfg.AgentToken = v
	}
	if v := os.Getenv("LIGHTAI_AGENT_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LIGHTAI_AGENT_METRICS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Metrics.Port = p
		}
	}
}
