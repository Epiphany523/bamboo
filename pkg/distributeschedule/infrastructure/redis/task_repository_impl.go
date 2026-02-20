package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/repository"
)

const (
	taskDetailPrefix = "task:detail:"
	taskQueuePrefix  = "task:queue:"
	taskResultPrefix = "task:result:"
	taskTTL          = 7 * 24 * time.Hour
)

// TaskRepositoryImpl 任务仓储实现（导出以便类型断言）
type TaskRepositoryImpl struct {
	client *Client
}

// NewTaskRepository 创建任务仓储实现
func NewTaskRepository(client *Client) repository.TaskRepository {
	return &TaskRepositoryImpl{client: client}
}

func (r *TaskRepositoryImpl) Save(ctx context.Context, task *model.Task) error {
	key := taskDetailPrefix + task.ID
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task failed: %w", err)
	}

	return r.client.Set(ctx, key, data, taskTTL)
}

func (r *TaskRepositoryImpl) FindByID(ctx context.Context, id string) (*model.Task, error) {
	key := taskDetailPrefix + id
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get task failed: %w", err)
	}

	var task model.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task failed: %w", err)
	}

	return &task, nil
}

func (r *TaskRepositoryImpl) FindPendingTasks(ctx context.Context, limit int) ([]*model.Task, error) {
	keys, err := r.client.Keys(ctx, taskDetailPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("find task keys failed: %w", err)
	}

	tasks := make([]*model.Task, 0)
	for _, key := range keys {
		if len(tasks) >= limit {
			break
		}

		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var task model.Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}

		if task.Status == model.TaskPending && time.Now().After(task.ScheduledTime) {
			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) FindRunningTasks(ctx context.Context) ([]*model.Task, error) {
	keys, err := r.client.Keys(ctx, taskDetailPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("find task keys failed: %w", err)
	}

	tasks := make([]*model.Task, 0)
	for _, key := range keys {
		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var task model.Task
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			continue
		}

		if task.Status == model.TaskRunning {
			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) FindTimeoutTasks(ctx context.Context, timeout time.Duration) ([]*model.Task, error) {
	runningTasks, err := r.FindRunningTasks(ctx)
	if err != nil {
		return nil, err
	}

	timeoutTasks := make([]*model.Task, 0)
	for _, task := range runningTasks {
		if task.IsTimeout(timeout) {
			timeoutTasks = append(timeoutTasks, task)
		}
	}

	return timeoutTasks, nil
}

func (r *TaskRepositoryImpl) Update(ctx context.Context, task *model.Task) error {
	return r.Save(ctx, task)
}

func (r *TaskRepositoryImpl) Delete(ctx context.Context, id string) error {
	key := taskDetailPrefix + id
	return r.client.Del(ctx, key)
}

// PushToQueue 将任务推入队列
func (r *TaskRepositoryImpl) PushToQueue(ctx context.Context, workerID, taskID string) error {
	key := taskQueuePrefix + workerID
	return r.client.LPush(ctx, key, taskID)
}

// PopFromQueue 从队列弹出任务
func (r *TaskRepositoryImpl) PopFromQueue(ctx context.Context, workerID string) (string, error) {
	key := taskQueuePrefix + workerID
	return r.client.RPop(ctx, key)
}

// SaveResult 保存任务结果
func (r *TaskRepositoryImpl) SaveResult(ctx context.Context, taskID string, result *model.TaskResult) error {
	key := taskResultPrefix + taskID
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result failed: %w", err)
	}

	return r.client.Set(ctx, key, data, taskTTL)
}

// extractTaskID 从 key 中提取 task ID
func extractTaskID(key string) string {
	return strings.TrimPrefix(key, taskDetailPrefix)
}
