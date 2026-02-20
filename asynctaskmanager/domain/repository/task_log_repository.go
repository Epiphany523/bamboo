package repository

import (
	"context"

	"bamboo/asynctaskmanager/domain/model"
)

// TaskLogRepository 任务日志仓储接口
type TaskLogRepository interface {
	// Create 创建任务日志
	Create(ctx context.Context, log *model.TaskLog) error

	// GetByTaskID 根据任务ID查找日志
	GetByTaskID(ctx context.Context, taskID string) ([]*model.TaskLog, error)

	// GetByTaskIDAndType 根据任务ID和日志类型查找日志
	GetByTaskIDAndType(ctx context.Context, taskID string, logType model.LogType) ([]*model.TaskLog, error)
}
