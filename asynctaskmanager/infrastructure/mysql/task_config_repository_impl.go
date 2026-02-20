package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

// TaskConfigRepositoryImpl TaskConfig 仓储 MySQL 实现
type TaskConfigRepositoryImpl struct {
	client *Client
}

// NewTaskConfigRepository 创建 TaskConfig 仓储
func NewTaskConfigRepository(client *Client) repository.TaskConfigRepository {
	return &TaskConfigRepositoryImpl{client: client}
}

// Create 创建任务配置
func (r *TaskConfigRepositoryImpl) Create(ctx context.Context, config *model.TaskConfig) error {
	executorConfig, err := json.Marshal(config.ExecutorConfig)
	if err != nil {
		return fmt.Errorf("marshal executor config failed: %w", err)
	}

	query := `INSERT INTO task_config (task_type, task_name, description, executor_type, executor_config,
		default_timeout, default_max_retry, retry_strategy, retry_delay, backoff_rate, max_concurrent, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.client.db.ExecContext(ctx, query,
		config.TaskType,
		config.TaskName,
		config.Description,
		config.ExecutorType,
		executorConfig,
		config.DefaultTimeout,
		config.DefaultMaxRetry,
		config.RetryStrategy,
		config.RetryDelay,
		config.BackoffRate,
		config.MaxConcurrent,
		config.Enabled,
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert task config failed: %w", err)
	}

	return nil
}

// GetByType 根据任务类型查找配置
func (r *TaskConfigRepositoryImpl) GetByType(ctx context.Context, taskType string) (*model.TaskConfig, error) {
	query := `SELECT task_type, task_name, description, executor_type, executor_config,
		default_timeout, default_max_retry, retry_strategy, retry_delay, backoff_rate, max_concurrent, enabled, created_at, updated_at
		FROM task_config WHERE task_type = ?`

	row := r.client.db.QueryRowContext(ctx, query, taskType)

	config := &model.TaskConfig{}
	var executorConfig []byte
	var description sql.NullString

	err := row.Scan(
		&config.TaskType,
		&config.TaskName,
		&description,
		&config.ExecutorType,
		&executorConfig,
		&config.DefaultTimeout,
		&config.DefaultMaxRetry,
		&config.RetryStrategy,
		&config.RetryDelay,
		&config.BackoffRate,
		&config.MaxConcurrent,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task config not found: %s", taskType)
	}
	if err != nil {
		return nil, fmt.Errorf("query task config failed: %w", err)
	}

	// 解析 executor_config
	if err := json.Unmarshal(executorConfig, &config.ExecutorConfig); err != nil {
		return nil, fmt.Errorf("unmarshal executor config failed: %w", err)
	}

	// 处理可空字段
	if description.Valid {
		config.Description = description.String
	}

	return config, nil
}

// Update 更新任务配置
func (r *TaskConfigRepositoryImpl) Update(ctx context.Context, config *model.TaskConfig) error {
	executorConfig, err := json.Marshal(config.ExecutorConfig)
	if err != nil {
		return fmt.Errorf("marshal executor config failed: %w", err)
	}

	query := `UPDATE task_config SET task_name = ?, description = ?, executor_type = ?, executor_config = ?,
		default_timeout = ?, default_max_retry = ?, retry_strategy = ?, retry_delay = ?, backoff_rate = ?, 
		max_concurrent = ?, enabled = ?, updated_at = ? WHERE task_type = ?`

	_, err = r.client.db.ExecContext(ctx, query,
		config.TaskName,
		config.Description,
		config.ExecutorType,
		executorConfig,
		config.DefaultTimeout,
		config.DefaultMaxRetry,
		config.RetryStrategy,
		config.RetryDelay,
		config.BackoffRate,
		config.MaxConcurrent,
		config.Enabled,
		config.UpdatedAt,
		config.TaskType,
	)

	if err != nil {
		return fmt.Errorf("update task config failed: %w", err)
	}

	return nil
}

// Delete 删除任务配置
func (r *TaskConfigRepositoryImpl) Delete(ctx context.Context, taskType string) error {
	query := `DELETE FROM task_config WHERE task_type = ?`
	_, err := r.client.db.ExecContext(ctx, query, taskType)
	if err != nil {
		return fmt.Errorf("delete task config failed: %w", err)
	}
	return nil
}

// FindAll 查找所有任务配置
func (r *TaskConfigRepositoryImpl) FindAll(ctx context.Context) ([]*model.TaskConfig, error) {
	query := `SELECT task_type, task_name, description, executor_type, executor_config,
		default_timeout, default_max_retry, retry_strategy, retry_delay, backoff_rate, max_concurrent, enabled, created_at, updated_at
		FROM task_config ORDER BY task_type`

	rows, err := r.client.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all task configs failed: %w", err)
	}
	defer rows.Close()

	return r.scanConfigs(rows)
}

// FindEnabled 查找启用的任务配置
func (r *TaskConfigRepositoryImpl) FindEnabled(ctx context.Context) ([]*model.TaskConfig, error) {
	query := `SELECT task_type, task_name, description, executor_type, executor_config,
		default_timeout, default_max_retry, retry_strategy, retry_delay, backoff_rate, max_concurrent, enabled, created_at, updated_at
		FROM task_config WHERE enabled = TRUE ORDER BY task_type`

	rows, err := r.client.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query enabled task configs failed: %w", err)
	}
	defer rows.Close()

	return r.scanConfigs(rows)
}

// scanConfigs 扫描配置列表
func (r *TaskConfigRepositoryImpl) scanConfigs(rows *sql.Rows) ([]*model.TaskConfig, error) {
	configs := make([]*model.TaskConfig, 0)

	for rows.Next() {
		config := &model.TaskConfig{}
		var executorConfig []byte
		var description sql.NullString

		err := rows.Scan(
			&config.TaskType,
			&config.TaskName,
			&description,
			&config.ExecutorType,
			&executorConfig,
			&config.DefaultTimeout,
			&config.DefaultMaxRetry,
			&config.RetryStrategy,
			&config.RetryDelay,
			&config.BackoffRate,
			&config.MaxConcurrent,
			&config.Enabled,
			&config.CreatedAt,
			&config.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan task config failed: %w", err)
		}

		// 解析 executor_config
		if err := json.Unmarshal(executorConfig, &config.ExecutorConfig); err != nil {
			return nil, fmt.Errorf("unmarshal executor config failed: %w", err)
		}

		// 处理可空字段
		if description.Valid {
			config.Description = description.String
		}

		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return configs, nil
}
