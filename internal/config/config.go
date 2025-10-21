package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Cache        CacheConfig        `yaml:"cache"`
	Queue        QueueConfig        `yaml:"queue"`
	Worker       WorkerConfig       `yaml:"worker"`
	Observability ObservabilityConfig `yaml:"observability"`
	Deployment   DeploymentConfig   `yaml:"deployment"`
	Plugins      PluginsConfig      `yaml:"plugins"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // dev, production
	Host string `yaml:"host"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"` // sqlite, postgresql, mysql
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Path     string `yaml:"path"` // For SQLite
	PoolSize int    `yaml:"pool_size"`
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	Type    string   `yaml:"type"` // memory, redis
	Nodes   []string `yaml:"nodes"`
	Cluster bool     `yaml:"cluster"`
}

// QueueConfig contains task queue configuration
type QueueConfig struct {
	Type string   `yaml:"type"` // channel, database, nats
	URLs []string `yaml:"urls"`
}

// WorkerConfig contains worker configuration
type WorkerConfig struct {
	MaxConcurrentJobs  int    `yaml:"max_concurrent_jobs"`
	HeartbeatInterval  string `yaml:"heartbeat_interval"`
	TaskPollInterval   string `yaml:"task_poll_interval"`
	WorkerCount        int    `yaml:"worker_count"`
}

// ObservabilityConfig contains observability configuration
type ObservabilityConfig struct {
	Logs    LogsConfig    `yaml:"logs"`
	Metrics MetricsConfig `yaml:"metrics"`
	Traces  TracesConfig  `yaml:"traces"`
}

// LogsConfig contains logging configuration
type LogsConfig struct {
	Local        LocalLogsConfig        `yaml:"local"`
	OpenObserve  OpenObserveLogsConfig  `yaml:"openobserve"`
}

// LocalLogsConfig contains local file logging configuration
type LocalLogsConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Path       string `yaml:"path"`
	MaxSize    string `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
}

// OpenObserveLogsConfig contains OpenObserve logging configuration
type OpenObserveLogsConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`
	Organization  string `yaml:"organization"`
	Stream        string `yaml:"stream"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	BatchSize     int    `yaml:"batch_size"`
	FlushInterval string `yaml:"flush_interval"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	OpenObserve OpenObserveMetricsConfig `yaml:"openobserve"`
}

// OpenObserveMetricsConfig contains OpenObserve metrics configuration
type OpenObserveMetricsConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Endpoint     string `yaml:"endpoint"`
	PushInterval string `yaml:"push_interval"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

// TracesConfig contains tracing configuration
type TracesConfig struct {
	OpenObserve OpenObserveTracesConfig `yaml:"openobserve"`
}

// OpenObserveTracesConfig contains OpenObserve tracing configuration
type OpenObserveTracesConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Endpoint     string `yaml:"endpoint"`
	Organization string `yaml:"organization"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

// DeploymentConfig contains deployment mode configuration
type DeploymentConfig struct {
	Mode string `yaml:"mode"` // standalone, lightweight, distributed
	Role string `yaml:"role"` // master, worker
}

// PluginsConfig contains plugin configuration
type PluginsConfig struct {
	Enabled  PluginTypesConfig `yaml:"enabled"`
	Disabled PluginTypesConfig `yaml:"disabled"`
}

// PluginTypesConfig contains enabled/disabled plugins by type
type PluginTypesConfig struct {
	Input     []string `yaml:"input"`
	Processor []string `yaml:"processor"`
	Output    []string `yaml:"output"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	config.applyDefaults()

	return &config, nil
}

// applyDefaults applies default values to configuration
func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "production"
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}

	if c.Database.Type == "" {
		c.Database.Type = "sqlite"
	}
	if c.Database.Path == "" && c.Database.Type == "sqlite" {
		c.Database.Path = "./fustgo.db"
	}
	if c.Database.PoolSize == 0 {
		c.Database.PoolSize = 20
	}

	if c.Cache.Type == "" {
		c.Cache.Type = "memory"
	}

	if c.Queue.Type == "" {
		c.Queue.Type = "channel"
	}

	if c.Worker.MaxConcurrentJobs == 0 {
		c.Worker.MaxConcurrentJobs = 5
	}
	if c.Worker.HeartbeatInterval == "" {
		c.Worker.HeartbeatInterval = "10s"
	}
	if c.Worker.TaskPollInterval == "" {
		c.Worker.TaskPollInterval = "5s"
	}
	if c.Worker.WorkerCount == 0 {
		c.Worker.WorkerCount = 4
	}

	if c.Deployment.Mode == "" {
		c.Deployment.Mode = "standalone"
	}

	if c.Observability.Logs.Local.Path == "" {
		c.Observability.Logs.Local.Path = "/var/log/fustgo"
	}
	if c.Observability.Logs.Local.MaxSize == "" {
		c.Observability.Logs.Local.MaxSize = "100MB"
	}
	if c.Observability.Logs.Local.MaxAge == 0 {
		c.Observability.Logs.Local.MaxAge = 7
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Type != "sqlite" && c.Database.Type != "postgresql" && c.Database.Type != "mysql" {
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}

	if c.Deployment.Mode != "standalone" && c.Deployment.Mode != "lightweight" && c.Deployment.Mode != "distributed" {
		return fmt.Errorf("invalid deployment mode: %s", c.Deployment.Mode)
	}

	return nil
}
