package repository

import (
	"context"

	"bamboo/pkg/distributeschedule/domain/model"
)

// TaskConfigRepository 任务配置仓储接口
type TaskConfigRepository interface {
	// Save 保存任务配置
	Save(ctx context.Context, config *model.TaskConfig) error

	// FindByID 根据ID查找任务配置
	FindByID(ctx context.Context, id string) (*model.TaskConfig, error)

	// FindAll 查找所有任务配置
	FindAll(ctx context.Context) ([]*model.TaskConfig, error)

	// FindEnabled 查找启用的任务配置
	FindEnabled(ctx context.Context) ([]*model.TaskConfig, error)

	// Update 更新任务配置
	Update(ctx context.Context, config *model.TaskConfig) error

	// Delete 删除任务配置
	Delete(ctx context.Context, id string) error
}
