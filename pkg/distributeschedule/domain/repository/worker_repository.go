package repository

import (
	"context"
	"time"

	"bamboo/pkg/distributeschedule/domain/model"
)

// WorkerRepository Worker 仓储接口
type WorkerRepository interface {
	// Register 注册 Worker
	Register(ctx context.Context, worker *model.Worker) error

	// FindByID 根据ID查找 Worker
	FindByID(ctx context.Context, id string) (*model.Worker, error)

	// FindAll 查找所有 Worker
	FindAll(ctx context.Context) ([]*model.Worker, error)

	// FindHealthy 查找健康的 Worker
	FindHealthy(ctx context.Context, timeout time.Duration) ([]*model.Worker, error)

	// UpdateHeartbeat 更新心跳
	UpdateHeartbeat(ctx context.Context, workerID string) error

	// Update 更新 Worker
	Update(ctx context.Context, worker *model.Worker) error

	// Remove 移除 Worker
	Remove(ctx context.Context, id string) error
}
