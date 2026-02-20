package executor

import (
	"context"
	"fmt"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/service"
)

// LocalExecutor 本地执行器
type LocalExecutor struct {
	handlers map[string]func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
}

// NewLocalExecutor 创建本地执行器
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{
		handlers: make(map[string]func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)),
	}
}

// RegisterHandler 注册处理函数
func (e *LocalExecutor) RegisterHandler(taskType string, handler func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)) {
	e.handlers[taskType] = handler
}

func (e *LocalExecutor) Execute(ctx context.Context, task *model.Task) (map[string]interface{}, error) {
	handler, ok := e.handlers[task.TaskType]
	if !ok {
		return nil, fmt.Errorf("handler not found for task type: %s", task.TaskType)
	}

	return handler(ctx, task.Payload)
}

func (e *LocalExecutor) Type() model.ExecutorType {
	return model.ExecutorTypeLocal
}

func (e *LocalExecutor) SupportedTaskTypes() []string {
	types := make([]string, 0, len(e.handlers))
	for taskType := range e.handlers {
		types = append(types, taskType)
	}
	return types
}

var _ service.Executor = (*LocalExecutor)(nil)
