package config

import (
	"time"
)

// Config 配置
type Config struct {
	App       AppConfig       `yaml:"app"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Worker    WorkerConfig    `yaml:"worker"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Port    int    `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Enabled             bool          `yaml:"enabled"`
	ScanInterval        time.Duration `yaml:"scan_interval"`
	BatchSize           int           `yaml:"batch_size"`
	TimeoutCheckInterval time.Duration `yaml:"timeout_check_interval"`
	LoadBalanceStrategy string        `yaml:"load_balance_strategy"`
}

// WorkerConfig Worker 配置
type WorkerConfig struct {
	Enabled           bool          `yaml:"enabled"`
	ID                string        `yaml:"id"`
	Name              string        `yaml:"name"`
	Capacity          int           `yaml:"capacity"`
	SupportedTypes    []string      `yaml:"supported_types"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	HeartbeatTimeout  time.Duration `yaml:"heartbeat_timeout"`
	QueuePollInterval time.Duration `yaml:"queue_poll_interval"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "async-task-manager",
			Version: "1.0.0",
			Port:    8080,
		},
		Database: DatabaseConfig{
			Host:            "127.0.0.1",
			Port:            3306,
			User:            "root",
			Password:        "root",
			Database:        "task_manager",
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Redis: RedisConfig{
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       0,
			PoolSize: 100,
		},
		Scheduler: SchedulerConfig{
			Enabled:             true,
			ScanInterval:        100 * time.Millisecond,
			BatchSize:           10,
			TimeoutCheckInterval: 30 * time.Second,
			LoadBalanceStrategy: "least_task",
		},
		Worker: WorkerConfig{
			Enabled:           true,
			ID:                "worker-001",
			Name:              "worker-node-1",
			Capacity:          10,
			SupportedTypes:    []string{},
			HeartbeatInterval: 10 * time.Second,
			HeartbeatTimeout:  30 * time.Second,
			QueuePollInterval: 100 * time.Millisecond,
		},
	}
}
