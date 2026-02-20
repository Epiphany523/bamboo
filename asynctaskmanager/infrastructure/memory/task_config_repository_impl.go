package memory

import (
	"context"
	"fmt"
	"sync"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

type taskConfigRepositoryImpl struct {
	configs map[string]*model.TaskConfig
	mu      sync.RWMutex
}

func NewTaskConfigRepository() repository.TaskConfigRepository {
	return &taskConfigRepositoryImpl{
		configs: make(map[string]*model.TaskConfig),
	}
}

func (r *taskConfigRepositoryImpl) Create(ctx context.Context, config *model.TaskConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.configs[config.TaskType]; exists {
		return fmt.Errorf("task config already exists: %s", config.TaskType)
	}

	r.configs[config.TaskType] = config
	return nil
}

func (r *taskConfigRepositoryImpl) GetByType(ctx context.Context, taskType string) (*model.TaskConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.configs[taskType]
	if !exists {
		return nil, fmt.Errorf("task config not found: %s", taskType)
	}

	return config, nil
}

func (r *taskConfigRepositoryImpl) Update(ctx context.Context, config *model.TaskConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.configs[config.TaskType]; !exists {
		return fmt.Errorf("task config not found: %s", config.TaskType)
	}

	r.configs[config.TaskType] = config
	return nil
}

func (r *taskConfigRepositoryImpl) Delete(ctx context.Context, taskType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.configs, taskType)
	return nil
}

func (r *taskConfigRepositoryImpl) FindAll(ctx context.Context) ([]*model.TaskConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs := make([]*model.TaskConfig, 0, len(r.configs))
	for _, config := range r.configs {
		configs = append(configs, config)
	}

	return configs, nil
}

func (r *taskConfigRepositoryImpl) FindEnabled(ctx context.Context) ([]*model.TaskConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs := make([]*model.TaskConfig, 0)
	for _, config := range r.configs {
		if config.Enabled {
			configs = append(configs, config)
		}
	}

	return configs, nil
}
