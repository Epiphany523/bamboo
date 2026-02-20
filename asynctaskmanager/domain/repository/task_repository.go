package repository

import (
	"context"

	"bamboo/asynctaskmanager/domain/model"
)

// TaskRepository 任务仓储接口
type TaskRepository interface {
	// Create 创建任务
	Create(ctx context.Context, task *model.Task) error

	// GetByID 根据ID查找任务
	GetByID(ctx context.Context, taskID string) (*model.Task, error)

	// Update 更新任务
	Update(ctx context.Context, task *model.Task) error

	// Delete 删除任务
	Delete(ctx context.Context, taskID string) error

	// FindPendingTasks 查找待执行的任务
	FindPendingTasks(ctx context.Context, limit int) ([]*model.Task, error)

	// FindProcessingTasks 查找正在执行的任务
	FindProcessingTasks(ctx context.Context) ([]*model.Task, error)

	// FindTimeoutTasks 查找超时的任务
	FindTimeoutTasks(ctx context.Context) ([]*model.Task, error)

	// FindByStatus 根据状态查找任务
	FindByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error)
}
