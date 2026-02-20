package memory

import (
	"context"
	"sync"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

type taskLogRepositoryImpl struct {
	logs map[string][]*model.TaskLog
	mu   sync.RWMutex
}

func NewTaskLogRepository() repository.TaskLogRepository {
	return &taskLogRepositoryImpl{
		logs: make(map[string][]*model.TaskLog),
	}
}

func (r *taskLogRepositoryImpl) Create(ctx context.Context, log *model.TaskLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logs[log.TaskID] = append(r.logs[log.TaskID], log)
	return nil
}

func (r *taskLogRepositoryImpl) GetByTaskID(ctx context.Context, taskID string) ([]*model.TaskLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logs, exists := r.logs[taskID]
	if !exists {
		return []*model.TaskLog{}, nil
	}

	return logs, nil
}

func (r *taskLogRepositoryImpl) GetByTaskIDAndType(ctx context.Context, taskID string, logType model.LogType) ([]*model.TaskLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allLogs, exists := r.logs[taskID]
	if !exists {
		return []*model.TaskLog{}, nil
	}

	filtered := make([]*model.TaskLog, 0)
	for _, log := range allLogs {
		if log.LogType == logType {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}
