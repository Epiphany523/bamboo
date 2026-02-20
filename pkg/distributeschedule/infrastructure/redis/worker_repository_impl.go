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
	workerKeyPrefix = "worker:pool:"
	workerTTL       = 30 * time.Second
)

type workerRepositoryImpl struct {
	client *Client
}

// NewWorkerRepository 创建 Worker 仓储实现
func NewWorkerRepository(client *Client) repository.WorkerRepository {
	return &workerRepositoryImpl{client: client}
}

func (r *workerRepositoryImpl) Register(ctx context.Context, worker *model.Worker) error {
	key := workerKeyPrefix + worker.ID
	data, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("marshal worker failed: %w", err)
	}

	return r.client.Set(ctx, key, data, workerTTL)
}

func (r *workerRepositoryImpl) FindByID(ctx context.Context, id string) (*model.Worker, error) {
	key := workerKeyPrefix + id
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get worker failed: %w", err)
	}

	var worker model.Worker
	if err := json.Unmarshal([]byte(data), &worker); err != nil {
		return nil, fmt.Errorf("unmarshal worker failed: %w", err)
	}

	return &worker, nil
}

func (r *workerRepositoryImpl) FindAll(ctx context.Context) ([]*model.Worker, error) {
	keys, err := r.client.Keys(ctx, workerKeyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("find worker keys failed: %w", err)
	}

	workers := make([]*model.Worker, 0, len(keys))
	for _, key := range keys {
		data, err := r.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var worker model.Worker
		if err := json.Unmarshal([]byte(data), &worker); err != nil {
			continue
		}

		workers = append(workers, &worker)
	}

	return workers, nil
}

func (r *workerRepositoryImpl) FindHealthy(ctx context.Context, timeout time.Duration) ([]*model.Worker, error) {
	allWorkers, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	healthy := make([]*model.Worker, 0)
	for _, worker := range allWorkers {
		if worker.IsHealthy(timeout) {
			healthy = append(healthy, worker)
		}
	}

	return healthy, nil
}

func (r *workerRepositoryImpl) UpdateHeartbeat(ctx context.Context, workerID string) error {
	worker, err := r.FindByID(ctx, workerID)
	if err != nil {
		return err
	}

	worker.UpdateHeartbeat()
	return r.Update(ctx, worker)
}

func (r *workerRepositoryImpl) Update(ctx context.Context, worker *model.Worker) error {
	key := workerKeyPrefix + worker.ID
	data, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("marshal worker failed: %w", err)
	}

	return r.client.Set(ctx, key, data, workerTTL)
}

func (r *workerRepositoryImpl) Remove(ctx context.Context, id string) error {
	key := workerKeyPrefix + id
	return r.client.Del(ctx, key)
}

// extractWorkerID 从 key 中提取 worker ID
func extractWorkerID(key string) string {
	return strings.TrimPrefix(key, workerKeyPrefix)
}
