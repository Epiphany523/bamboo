package executor

import (
	"fmt"
	"sync"

	"bamboo/asynctaskmanager/domain/service"
)

type executorRegistryImpl struct {
	executors map[string]service.Executor
	mu        sync.RWMutex
}

// NewExecutorRegistry 创建执行器注册表
func NewExecutorRegistry() service.ExecutorRegistry {
	return &executorRegistryImpl{
		executors: make(map[string]service.Executor),
	}
}

func (r *executorRegistryImpl) Register(executor service.Executor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, taskType := range executor.SupportedTaskTypes() {
		if _, exists := r.executors[taskType]; exists {
			return fmt.Errorf("executor for task type %s already registered", taskType)
		}
		r.executors[taskType] = executor
	}

	return nil
}

func (r *executorRegistryImpl) Get(taskType string) (service.Executor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, ok := r.executors[taskType]
	if !ok {
		return nil, fmt.Errorf("executor for task type %s not found", taskType)
	}

	return executor, nil
}

func (r *executorRegistryImpl) List() []service.Executor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executors := make([]service.Executor, 0, len(r.executors))
	seen := make(map[service.Executor]bool)

	for _, executor := range r.executors {
		if !seen[executor] {
			executors = append(executors, executor)
			seen[executor] = true
		}
	}

	return executors
}
