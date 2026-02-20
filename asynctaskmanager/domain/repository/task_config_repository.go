package repository

import (
	"context"

	"bamboo/asynctaskmanager/domain/model"
)

// TaskConfigRepository 任务配置仓储接口
type TaskConfigRepository interface {
	// Create 创建任务配置
	Create(ctx context.Context, config *model.TaskConfig) error

	// GetByType 根据任务类型查找配置
	GetByType(ctx context.Context, taskType string) (*model.TaskConfig, error)

	// Update 更新任务配置
	Update(ctx context.Context, config *model.TaskConfig) error

	// Delete 删除任务配置
	Delete(ctx context.Context, taskType string) error

	// FindAll 查找所有任务配置
	FindAll(ctx context.Context) ([]*model.TaskConfig, error)

	// FindEnabled 查找启用的任务配置
	FindEnabled(ctx context.Context) ([]*model.TaskConfig, error)
}
