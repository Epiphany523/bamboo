package config

import (
	"time"
)

// Config 配置
type Config struct {
	Schedule ScheduleConfig `yaml:"schedule"`
	Worker   WorkerConfig   `yaml:"worker"`
	Task     TaskConfig     `yaml:"task"`
	Redis    RedisConfig    `yaml:"redis"`
}

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	LeaderLockTTL       time.Duration `yaml:"leader_lock_ttl"`
	LeaderRenewInterval time.Duration `yaml:"leader_renew_interval"`
	ScanInterval        time.Duration `yaml:"scan_interval"`
	LoadBalanceStrategy string        `yaml:"load_balance_strategy"`
}

// WorkerConfig Worker 配置
type WorkerConfig struct {
	ID                  string        `yaml:"id"`
	Address             string        `yaml:"address"`
	HeartbeatInterval   time.Duration `yaml:"heartbeat_interval"`
	HeartbeatTimeout    time.Duration `yaml:"heartbeat_timeout"`
	MaxConcurrentTasks  int           `yaml:"max_concurrent_tasks"`
}

// TaskConfig 任务配置
type TaskConfig struct {
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	MaxRetry       int           `yaml:"max_retry"`
	RetryDelay     time.Duration `yaml:"retry_delay"`
	BackoffRate    float64       `yaml:"backoff_rate"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Schedule: ScheduleConfig{
			LeaderLockTTL:       10 * time.Second,
			LeaderRenewInterval: 3 * time.Second,
			ScanInterval:        5 * time.Second,
			LoadBalanceStrategy: "least_task",
		},
		Worker: WorkerConfig{
			HeartbeatInterval:  10 * time.Second,
			HeartbeatTimeout:   30 * time.Second,
			MaxConcurrentTasks: 10,
		},
		Task: TaskConfig{
			DefaultTimeout: 5 * time.Minute,
			MaxRetry:       3,
			RetryDelay:     10 * time.Second,
			BackoffRate:    2.0,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
	}
}
