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
	Port          int                 `yaml:"port" json:"port"`
	LogLevel      string              `yaml:"log_level" json:"log_level"`
	DBPath        string              `yaml:"db_path" json:"db_path"`
	DevMode       bool                `yaml:"dev_mode" json:"dev_mode"`
	AgentToken    string              `yaml:"agent_token" json:"-"` // Bootstrap token for agent auth
	Observability ObservabilityConfig `yaml:"observability" json:"observability"`
	GPU           GPUProfileConfig    `yaml:"gpu" json:"gpu"`
}

// ObservabilityConfig holds observability settings.
type ObservabilityConfig struct {
	Mode string `yaml:"mode" json:"mode"` // builtin, external, disabled
}

// GPUProfileConfig holds GPU collector profile settings.
type GPUProfileConfig struct {
	Profile string `yaml:"profile" json:"profile"` // production, development, test
}

// AgentConfig holds all agent configuration.
type AgentConfig struct {
	AgentID    string               `yaml:"agent_id" json:"agent_id"`
	ServerURL  string               `yaml:"server_url" json:"server_url"`
	AgentToken string               `yaml:"agent_token" json:"-"`
	LogLevel   string               `yaml:"log_level" json:"log_level"`
	DataDir    string               `yaml:"data_dir" json:"data_dir"`
	Metrics    AgentMetricsConfig   `yaml:"metrics" json:"metrics"`
	Health     AgentHealthConfig    `yaml:"health" json:"health"`
	Collectors AgentCollectorConfig `yaml:"collectors" json:"collectors"`
	GPU        GPUProfileConfig     `yaml:"gpu" json:"gpu"`
}

// AgentMetricsConfig holds agent metrics server settings.
type AgentMetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Scheme  string `yaml:"scheme" json:"scheme"`
	Port    int    `yaml:"port" json:"port"`
	Path    string `yaml:"path" json:"path"`
}

// AgentHealthConfig holds agent health server settings.
type AgentHealthConfig struct {
	Port int `yaml:"port" json:"port"`
}

// AgentCollectorConfig holds agent collector settings.
type AgentCollectorConfig struct {
	System  SystemCollectorConfig  `yaml:"system" json:"system"`
	MockGPU MockGPUCollectorConfig `yaml:"mock_gpu" json:"mock_gpu"`
	Nvidia  NvidiaCollectorConfig  `yaml:"nvidia" json:"nvidia"`
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
		Port:       8080,
		LogLevel:   "info",
		DBPath:     "data/lightai.db",
		DevMode:    false,
		AgentToken: "lightai-agent-token-change-me",
		Observability: ObservabilityConfig{
			Mode: "disabled",
		},
		GPU: GPUProfileConfig{
			Profile: "production",
		},
	}
}

// DefaultAgentConfig returns an AgentConfig with safe defaults.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		ServerURL:  "http://localhost:8080",
		AgentToken: "lightai-agent-token-change-me",
		LogLevel:   "info",
		DataDir:    "data",
		Metrics: AgentMetricsConfig{
			Enabled: true,
			Scheme:  "http",
			Port:    9090,
			Path:    "/metrics",
		},
		Health: AgentHealthConfig{
			Port: 9091,
		},
		GPU: GPUProfileConfig{
			Profile: "production",
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
	if v := os.Getenv("LIGHTAI_AGENT_HEALTH_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Health.Port = p
		}
	}
}
