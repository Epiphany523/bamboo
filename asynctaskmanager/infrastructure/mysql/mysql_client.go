package mysql

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Client MySQL 客户端
type Client struct {
	db *sql.DB
}

// Config MySQL 配置
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	MaxOpen  int
	MaxIdle  int
	MaxLife  time.Duration
}

// NewClient 创建 MySQL 客户端
func NewClient(cfg Config) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed: %w", err)
	}

	// 设置连接池
	if cfg.MaxOpen > 0 {
		db.SetMaxOpenConns(cfg.MaxOpen)
	}
	if cfg.MaxIdle > 0 {
		db.SetMaxIdleConns(cfg.MaxIdle)
	}
	if cfg.MaxLife > 0 {
		db.SetConnMaxLifetime(cfg.MaxLife)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	return &Client{db: db}, nil
}

// DB 获取数据库连接
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.db.Close()
}

// InitSchema 初始化数据库表结构
func (c *Client) InitSchema() error {
	schemas := []string{
		// task 表
		`CREATE TABLE IF NOT EXISTS task (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			task_id VARCHAR(64) UNIQUE NOT NULL,
			task_type VARCHAR(64) NOT NULL,
			priority INT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL,
			payload JSON,
			result JSON,
			error_message TEXT,
			worker_id VARCHAR(64),
			retry_count INT NOT NULL DEFAULT 0,
			max_retry INT NOT NULL DEFAULT 3,
			timeout INT NOT NULL DEFAULT 30,
			scheduled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			started_at TIMESTAMP NULL,
			completed_at TIMESTAMP NULL,
			INDEX idx_task_id (task_id),
			INDEX idx_status (status),
			INDEX idx_task_type (task_type),
			INDEX idx_worker_id (worker_id),
			INDEX idx_priority (priority),
			INDEX idx_scheduled_at (scheduled_at),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// task_config 表
		`CREATE TABLE IF NOT EXISTS task_config (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			task_type VARCHAR(64) UNIQUE NOT NULL,
			task_name VARCHAR(128) NOT NULL,
			description TEXT,
			executor_type VARCHAR(32) NOT NULL,
			executor_config JSON,
			default_timeout INT NOT NULL DEFAULT 30,
			default_max_retry INT NOT NULL DEFAULT 3,
			retry_strategy VARCHAR(32) NOT NULL,
			retry_delay INT NOT NULL DEFAULT 5,
			backoff_rate DECIMAL(10,2) NOT NULL DEFAULT 2.0,
			max_concurrent INT NOT NULL DEFAULT 10,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_task_type (task_type),
			INDEX idx_enabled (enabled)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// task_log 表
		`CREATE TABLE IF NOT EXISTS task_log (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL,
			log_type VARCHAR(32) NOT NULL,
			from_status VARCHAR(32),
			to_status VARCHAR(32),
			message TEXT,
			worker_id VARCHAR(64),
			retry_count INT DEFAULT 0,
			error_detail TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_task_id (task_id),
			INDEX idx_log_type (log_type),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		// worker 表
		`CREATE TABLE IF NOT EXISTS worker (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			worker_id VARCHAR(64) UNIQUE NOT NULL,
			worker_name VARCHAR(128) NOT NULL,
			address VARCHAR(256) NOT NULL,
			status VARCHAR(32) NOT NULL,
			capacity INT NOT NULL DEFAULT 10,
			current_load INT NOT NULL DEFAULT 0,
			supported_types JSON,
			last_heartbeat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_worker_id (worker_id),
			INDEX idx_status (status),
			INDEX idx_last_heartbeat (last_heartbeat)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, schema := range schemas {
		if _, err := c.db.Exec(schema); err != nil {
			return fmt.Errorf("create table failed: %w", err)
		}
	}

	return nil
}
