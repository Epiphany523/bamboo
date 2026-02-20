package service

import (
	"context"

	"bamboo/pkg/distributeschedule/domain/model"
)

// Executor 执行器接口
type Executor interface {
	// Execute 执行任务
	Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error)

	// Type 任务类型
	Type() string

	// Protocol 支持的协议（http/rpc/local）
	Protocol() string
}

// ExecutorRegistry 执行器注册表
type ExecutorRegistry interface {
	// Register 注册执行器
	Register(executor Executor)

	// Get 获取执行器
	Get(executorType string) (Executor, bool)

	// List 列出所有执行器
	List() []Executor
}
