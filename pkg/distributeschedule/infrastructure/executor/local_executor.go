package executor

import (
	"context"
	"fmt"

	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/service"
)

// LocalExecutor 本地执行器
type LocalExecutor struct {
	handlers map[string]func(ctx context.Context, payload interface{}) (*model.TaskResult, error)
}

// NewLocalExecutor 创建本地执行器
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{
		handlers: make(map[string]func(ctx context.Context, payload interface{}) (*model.TaskResult, error)),
	}
}

// RegisterHandler 注册处理函数
func (e *LocalExecutor) RegisterHandler(name string, handler func(ctx context.Context, payload interface{}) (*model.TaskResult, error)) {
	e.handlers[name] = handler
}

func (e *LocalExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
	// 从 payload 中解析处理函数名称
	payload, ok := task.Result.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload format")
	}

	handlerName, _ := payload["handler"].(string)
	if handlerName == "" {
		return nil, fmt.Errorf("handler name is required")
	}

	handler, ok := e.handlers[handlerName]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", handlerName)
	}

	return handler(ctx, payload["data"])
}

func (e *LocalExecutor) Type() string {
	return "local"
}

func (e *LocalExecutor) Protocol() string {
	return "local"
}

var _ service.Executor = (*LocalExecutor)(nil)
