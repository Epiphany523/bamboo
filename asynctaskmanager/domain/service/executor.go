package service

import (
	"context"

	"bamboo/asynctaskmanager/domain/model"
)

// Executor 执行器接口
type Executor interface {
	// Execute 执行任务
	Execute(ctx context.Context, task *model.Task) (map[string]interface{}, error)

	// Type 执行器类型
	Type() model.ExecutorType

	// SupportedTaskTypes 支持的任务类型
	SupportedTaskTypes() []string
}

// ExecutorRegistry 执行器注册表
type ExecutorRegistry interface {
	// Register 注册执行器
	Register(executor Executor) error

	// Get 获取执行器
	Get(taskType string) (Executor, error)

	// List 列出所有执行器
	List() []Executor
}
