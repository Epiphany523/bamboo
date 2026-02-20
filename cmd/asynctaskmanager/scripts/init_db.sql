-- 创建数据库
CREATE DATABASE IF NOT EXISTS asynctask DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE asynctask;

-- 任务表
CREATE TABLE IF NOT EXISTS task (
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
    INDEX idx_priority (priority),
    INDEX idx_scheduled_at (scheduled_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 任务日志表
CREATE TABLE IF NOT EXISTS task_log (
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
    INDEX idx_log_type (log_type)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 任务配置表
CREATE TABLE IF NOT EXISTS task_config (
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
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Worker 注册表
CREATE TABLE IF NOT EXISTS worker (
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
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入示例任务配置
INSERT INTO task_config (
    task_type, task_name, description, executor_type, executor_config,
    default_timeout, default_max_retry, retry_strategy, retry_delay,
    backoff_rate, max_concurrent, enabled
) VALUES (
    'example_task',
    'Example Task',
    'A simple example task for testing',
    'local',
    '{}',
    30,
    3,
    'fixed',
    5,
    2.0,
    10,
    TRUE
) ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;
