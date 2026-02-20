package repository

import (
	"context"
	"time"

	"bamboo/asynctaskmanager/domain/model"
)

// WorkerRepository Worker 仓储接口
type WorkerRepository interface {
	// Register 注册 Worker
	Register(ctx context.Context, worker *model.Worker) error

	// GetByID 根据ID查找 Worker
	GetByID(ctx context.Context, workerID string) (*model.Worker, error)

	// Update 更新 Worker
	Update(ctx context.Context, worker *model.Worker) error

	// Remove 移除 Worker
	Remove(ctx context.Context, workerID string) error

	// FindAll 查找所有 Worker
	FindAll(ctx context.Context) ([]*model.Worker, error)

	// FindHealthy 查找健康的 Worker
	FindHealthy(ctx context.Context, timeout time.Duration) ([]*model.Worker, error)

	// FindByTaskType 根据任务类型查找支持的 Worker
	FindByTaskType(ctx context.Context, taskType string) ([]*model.Worker, error)

	// UpdateHeartbeat 更新心跳
	UpdateHeartbeat(ctx context.Context, workerID string) error

	// UpdateLoad 更新负载
	UpdateLoad(ctx context.Context, workerID string, load int) error
}
