package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

// TaskLogRepositoryImpl TaskLog 仓储 MySQL 实现
type TaskLogRepositoryImpl struct {
	client *Client
}

// NewTaskLogRepository 创建 TaskLog 仓储
func NewTaskLogRepository(client *Client) repository.TaskLogRepository {
	return &TaskLogRepositoryImpl{client: client}
}

// Create 创建任务日志
func (r *TaskLogRepositoryImpl) Create(ctx context.Context, log *model.TaskLog) error {
	query := `INSERT INTO task_log (task_id, log_type, from_status, to_status, message, worker_id, retry_count, error_detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.client.db.ExecContext(ctx, query,
		log.TaskID,
		log.LogType,
		log.FromStatus,
		log.ToStatus,
		log.Message,
		log.WorkerID,
		log.RetryCount,
		log.ErrorDetail,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert task log failed: %w", err)
	}

	return nil
}

// GetByTaskID 根据任务ID查找日志
func (r *TaskLogRepositoryImpl) GetByTaskID(ctx context.Context, taskID string) ([]*model.TaskLog, error) {
	query := `SELECT id, task_id, log_type, from_status, to_status, message, worker_id, retry_count, error_detail, created_at
		FROM task_log WHERE task_id = ? ORDER BY created_at ASC`

	rows, err := r.client.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query task logs failed: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByTaskIDAndType 根据任务ID和日志类型查找日志
func (r *TaskLogRepositoryImpl) GetByTaskIDAndType(ctx context.Context, taskID string, logType model.LogType) ([]*model.TaskLog, error) {
	query := `SELECT id, task_id, log_type, from_status, to_status, message, worker_id, retry_count, error_detail, created_at
		FROM task_log WHERE task_id = ? AND log_type = ? ORDER BY created_at ASC`

	rows, err := r.client.db.QueryContext(ctx, query, taskID, logType)
	if err != nil {
		return nil, fmt.Errorf("query task logs by type failed: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// scanLogs 扫描日志列表
func (r *TaskLogRepositoryImpl) scanLogs(rows *sql.Rows) ([]*model.TaskLog, error) {
	logs := make([]*model.TaskLog, 0)

	for rows.Next() {
		log := &model.TaskLog{}
		var fromStatus, toStatus, message, workerID, errorDetail sql.NullString
		var retryCount sql.NullInt64

		err := rows.Scan(
			&log.ID,
			&log.TaskID,
			&log.LogType,
			&fromStatus,
			&toStatus,
			&message,
			&workerID,
			&retryCount,
			&errorDetail,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan task log failed: %w", err)
		}

		// 处理可空字段
		if fromStatus.Valid {
			log.FromStatus = model.TaskStatus(fromStatus.String)
		}
		if toStatus.Valid {
			log.ToStatus = model.TaskStatus(toStatus.String)
		}
		if message.Valid {
			log.Message = message.String
		}
		if workerID.Valid {
			log.WorkerID = workerID.String
		}
		if retryCount.Valid {
			log.RetryCount = int(retryCount.Int64)
		}
		if errorDetail.Valid {
			log.ErrorDetail = errorDetail.String
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return logs, nil
}
