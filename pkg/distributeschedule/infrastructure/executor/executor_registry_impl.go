package executor

import (
	"sync"

	"bamboo/pkg/distributeschedule/domain/service"
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

func (r *executorRegistryImpl) Register(executor service.Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[executor.Type()] = executor
}

func (r *executorRegistryImpl) Get(executorType string) (service.Executor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	executor, ok := r.executors[executorType]
	return executor, ok
}

func (r *executorRegistryImpl) List() []service.Executor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executors := make([]service.Executor, 0, len(r.executors))
	for _, executor := range r.executors {
		executors = append(executors, executor)
	}
	return executors
}
