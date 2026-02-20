package repository

import (
	"context"
	"time"

	"bamboo/pkg/distributeschedule/domain/model"
)

// TaskRepository 任务仓储接口
type TaskRepository interface {
	// Save 保存任务
	Save(ctx context.Context, task *model.Task) error

	// FindByID 根据ID查找任务
	FindByID(ctx context.Context, id string) (*model.Task, error)

	// FindPendingTasks 查找待执行的任务
	FindPendingTasks(ctx context.Context, limit int) ([]*model.Task, error)

	// FindRunningTasks 查找正在执行的任务
	FindRunningTasks(ctx context.Context) ([]*model.Task, error)

	// FindTimeoutTasks 查找超时的任务
	FindTimeoutTasks(ctx context.Context, timeout time.Duration) ([]*model.Task, error)

	// Update 更新任务
	Update(ctx context.Context, task *model.Task) error

	// Delete 删除任务
	Delete(ctx context.Context, id string) error
}
