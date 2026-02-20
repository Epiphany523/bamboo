package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

// TaskRepositoryImpl Task 仓储 MySQL 实现
type TaskRepositoryImpl struct {
	client *Client
}

// NewTaskRepository 创建 Task 仓储
func NewTaskRepository(client *Client) repository.TaskRepository {
	return &TaskRepositoryImpl{client: client}
}

// Create 创建任务
func (r *TaskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	query := `INSERT INTO task (task_id, task_type, priority, status, payload, retry_count, max_retry, timeout, scheduled_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err = r.client.db.ExecContext(ctx, query,
		task.TaskID,
		task.TaskType,
		task.Priority.Value(),
		task.Status,
		payload,
		task.RetryCount,
		task.MaxRetry,
		task.Timeout,
		task.ScheduledAt,
		task.CreatedAt,
		now,
	)

	if err != nil {
		return fmt.Errorf("insert task failed: %w", err)
	}

	return nil
}

// GetByID 根据ID查找任务
func (r *TaskRepositoryImpl) GetByID(ctx context.Context, taskID string) (*model.Task, error) {
	query := `SELECT id, task_id, task_type, priority, status, payload, result, error_message, worker_id, 
		retry_count, max_retry, timeout, scheduled_at, created_at, updated_at, started_at, completed_at
		FROM task WHERE task_id = ?`

	row := r.client.db.QueryRowContext(ctx, query, taskID)

	task := &model.Task{}
	var payload, result []byte
	var errorMessage, workerID sql.NullString
	var startedAt, completedAt sql.NullTime
	var priority int

	err := row.Scan(
		&task.ID,
		&task.TaskID,
		&task.TaskType,
		&priority,
		&task.Status,
		&payload,
		&result,
		&errorMessage,
		&workerID,
		&task.RetryCount,
		&task.MaxRetry,
		&task.Timeout,
		&task.ScheduledAt,
		&task.CreatedAt,
		&task.UpdatedAt,
		&startedAt,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("query task failed: %w", err)
	}

	// 解析 priority
	if priority == 1 {
		task.Priority = model.PriorityHigh
	} else {
		task.Priority = model.PriorityNormal
	}

	// 解析 payload
	if err := json.Unmarshal(payload, &task.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload failed: %w", err)
	}

	// 解析 result
	if len(result) > 0 {
		if err := json.Unmarshal(result, &task.Result); err != nil {
			return nil, fmt.Errorf("unmarshal result failed: %w", err)
		}
	}

	// 处理可空字段
	if errorMessage.Valid {
		task.ErrorMsg = errorMessage.String
	}
	if workerID.Valid {
		task.WorkerID = workerID.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return task, nil
}

// Update 更新任务
func (r *TaskRepositoryImpl) Update(ctx context.Context, task *model.Task) error {
	result, err := json.Marshal(task.Result)
	if err != nil {
		return fmt.Errorf("marshal result failed: %w", err)
	}

	query := `UPDATE task SET status = ?, result = ?, error_message = ?, worker_id = ?, 
		retry_count = ?, scheduled_at = ?, started_at = ?, completed_at = ?, updated_at = ? WHERE task_id = ?`

	_, err = r.client.db.ExecContext(ctx, query,
		task.Status,
		result,
		task.ErrorMsg,
		task.WorkerID,
		task.RetryCount,
		task.ScheduledAt,
		task.StartedAt,
		task.CompletedAt,
		time.Now(),
		task.TaskID,
	)

	if err != nil {
		return fmt.Errorf("update task failed: %w", err)
	}

	return nil
}

// Delete 删除任务
func (r *TaskRepositoryImpl) Delete(ctx context.Context, taskID string) error {
	query := `DELETE FROM task WHERE task_id = ?`
	_, err := r.client.db.ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("delete task failed: %w", err)
	}
	return nil
}

// FindPendingTasks 查找待执行的任务
func (r *TaskRepositoryImpl) FindPendingTasks(ctx context.Context, limit int) ([]*model.Task, error) {
	query := `SELECT id, task_id, task_type, priority, status, payload, result, error_message, worker_id, 
		retry_count, max_retry, timeout, scheduled_at, created_at, updated_at, started_at, completed_at
		FROM task WHERE status = ? AND scheduled_at <= ?
		ORDER BY priority DESC, created_at ASC LIMIT ?`

	rows, err := r.client.db.QueryContext(ctx, query, model.StatusPending, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("query pending tasks failed: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// FindProcessingTasks 查找正在执行的任务
func (r *TaskRepositoryImpl) FindProcessingTasks(ctx context.Context) ([]*model.Task, error) {
	query := `SELECT id, task_id, task_type, priority, status, payload, result, error_message, worker_id, 
		retry_count, max_retry, timeout, scheduled_at, created_at, updated_at, started_at, completed_at
		FROM task WHERE status = ?`

	rows, err := r.client.db.QueryContext(ctx, query, model.StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("query processing tasks failed: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// FindTimeoutTasks 查找超时的任务
func (r *TaskRepositoryImpl) FindTimeoutTasks(ctx context.Context) ([]*model.Task, error) {
	query := `SELECT id, task_id, task_type, priority, status, payload, result, error_message, worker_id, 
		retry_count, max_retry, timeout, scheduled_at, created_at, updated_at, started_at, completed_at
		FROM task WHERE status = ? AND started_at IS NOT NULL 
		AND TIMESTAMPDIFF(SECOND, started_at, NOW()) > timeout`

	rows, err := r.client.db.QueryContext(ctx, query, model.StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("query timeout tasks failed: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// FindByStatus 根据状态查找任务
func (r *TaskRepositoryImpl) FindByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error) {
	query := `SELECT id, task_id, task_type, priority, status, payload, result, error_message, worker_id, 
		retry_count, max_retry, timeout, scheduled_at, created_at, updated_at, started_at, completed_at
		FROM task WHERE status = ? ORDER BY created_at DESC LIMIT ?`

	rows, err := r.client.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("query tasks by status failed: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// scanTasks 扫描任务列表
func (r *TaskRepositoryImpl) scanTasks(rows *sql.Rows) ([]*model.Task, error) {
	tasks := make([]*model.Task, 0)

	for rows.Next() {
		task := &model.Task{}
		var payload, result []byte
		var errorMessage, workerID sql.NullString
		var startedAt, completedAt sql.NullTime
		var priority int

		err := rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.TaskType,
			&priority,
			&task.Status,
			&payload,
			&result,
			&errorMessage,
			&workerID,
			&task.RetryCount,
			&task.MaxRetry,
			&task.Timeout,
			&task.ScheduledAt,
			&task.CreatedAt,
			&task.UpdatedAt,
			&startedAt,
			&completedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan task failed: %w", err)
		}

		// 解析 priority
		if priority == 1 {
			task.Priority = model.PriorityHigh
		} else {
			task.Priority = model.PriorityNormal
		}

		// 解析 payload
		if err := json.Unmarshal(payload, &task.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload failed: %w", err)
		}

		// 解析 result
		if len(result) > 0 {
			if err := json.Unmarshal(result, &task.Result); err != nil {
				return nil, fmt.Errorf("unmarshal result failed: %w", err)
			}
		}

		// 处理可空字段
		if errorMessage.Valid {
			task.ErrorMsg = errorMessage.String
		}
		if workerID.Valid {
			task.WorkerID = workerID.String
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return tasks, nil
}
