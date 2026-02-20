package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

type taskRepositoryImpl struct {
	tasks map[string]*model.Task
	mu    sync.RWMutex
}

func NewTaskRepository() repository.TaskRepository {
	return &taskRepositoryImpl{
		tasks: make(map[string]*model.Task),
	}
}

func (r *taskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.TaskID]; exists {
		return fmt.Errorf("task already exists: %s", task.TaskID)
	}

	r.tasks[task.TaskID] = task
	return nil
}

func (r *taskRepositoryImpl) GetByID(ctx context.Context, taskID string) (*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

func (r *taskRepositoryImpl) Update(ctx context.Context, task *model.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.TaskID]; !exists {
		return fmt.Errorf("task not found: %s", task.TaskID)
	}

	task.UpdatedAt = time.Now()
	r.tasks[task.TaskID] = task
	return nil
}

func (r *taskRepositoryImpl) Delete(ctx context.Context, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tasks, taskID)
	return nil
}

func (r *taskRepositoryImpl) FindPendingTasks(ctx context.Context, limit int) ([]*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*model.Task, 0)
	for _, task := range r.tasks {
		if task.Status == model.StatusPending && time.Now().After(task.ScheduledAt) {
			tasks = append(tasks, task)
			if len(tasks) >= limit {
				break
			}
		}
	}

	return tasks, nil
}

func (r *taskRepositoryImpl) FindProcessingTasks(ctx context.Context) ([]*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*model.Task, 0)
	for _, task := range r.tasks {
		if task.Status == model.StatusProcessing {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (r *taskRepositoryImpl) FindTimeoutTasks(ctx context.Context) ([]*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*model.Task, 0)
	for _, task := range r.tasks {
		if task.IsTimeout() {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (r *taskRepositoryImpl) FindByStatus(ctx context.Context, status model.TaskStatus, limit int) ([]*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*model.Task, 0)
	for _, task := range r.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
			if len(tasks) >= limit {
				break
			}
		}
	}

	return tasks, nil
}
