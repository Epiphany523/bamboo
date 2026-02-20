package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
)

const (
	workerKeyPrefix = "worker:"
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
	key := workerKeyPrefix + worker.WorkerID

	// 序列化 Worker 信息
	data := map[string]interface{}{
		"worker_id":       worker.WorkerID,
		"worker_name":     worker.WorkerName,
		"address":         worker.Address,
		"status":          string(worker.Status),
		"capacity":        worker.Capacity,
		"current_load":    worker.CurrentLoad,
		"supported_types": strings.Join(worker.SupportedTypes, ","),
		"last_heartbeat":  worker.LastHeartbeat.Unix(),
	}

	// 存储到 Redis Hash
	if err := r.client.HSet(ctx, key, flattenMap(data)...); err != nil {
		return fmt.Errorf("register worker failed: %w", err)
	}

	// 设置过期时间
	if err := r.client.Expire(ctx, key, workerTTL); err != nil {
		return fmt.Errorf("set worker ttl failed: %w", err)
	}

	// 添加到任务类型索引
	for _, taskType := range worker.SupportedTypes {
		indexKey := fmt.Sprintf("worker:type:%s", taskType)
		if err := r.client.SAdd(ctx, indexKey, worker.WorkerID); err != nil {
			return fmt.Errorf("add to type index failed: %w", err)
		}
	}

	return nil
}

func (r *workerRepositoryImpl) GetByID(ctx context.Context, workerID string) (*model.Worker, error) {
	key := workerKeyPrefix + workerID
	data, err := r.client.HGetAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get worker failed: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("worker not found")
	}

	return parseWorker(data)
}

func (r *workerRepositoryImpl) Update(ctx context.Context, worker *model.Worker) error {
	return r.Register(ctx, worker)
}

func (r *workerRepositoryImpl) Remove(ctx context.Context, workerID string) error {
	key := workerKeyPrefix + workerID

	// 获取 Worker 信息以清理索引
	worker, err := r.GetByID(ctx, workerID)
	if err == nil {
		// 从任务类型索引中移除
		for _, taskType := range worker.SupportedTypes {
			indexKey := fmt.Sprintf("worker:type:%s", taskType)
			_ = r.client.SRem(ctx, indexKey, workerID)
		}
	}

	return r.client.Del(ctx, key)
}

func (r *workerRepositoryImpl) FindAll(ctx context.Context) ([]*model.Worker, error) {
	keys, err := r.client.Keys(ctx, workerKeyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("find worker keys failed: %w", err)
	}

	workers := make([]*model.Worker, 0, len(keys))
	for _, key := range keys {
		// 跳过索引键
		if strings.Contains(key, ":type:") || strings.Contains(key, ":queue") {
			continue
		}

		data, err := r.client.HGetAll(ctx, key)
		if err != nil {
			continue
		}

		worker, err := parseWorker(data)
		if err != nil {
			continue
		}

		workers = append(workers, worker)
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

func (r *workerRepositoryImpl) FindByTaskType(ctx context.Context, taskType string) ([]*model.Worker, error) {
	indexKey := fmt.Sprintf("worker:type:%s", taskType)
	workerIDs, err := r.client.SMembers(ctx, indexKey)
	if err != nil {
		return nil, fmt.Errorf("get workers by task type failed: %w", err)
	}

	workers := make([]*model.Worker, 0, len(workerIDs))
	for _, workerID := range workerIDs {
		worker, err := r.GetByID(ctx, workerID)
		if err != nil {
			continue
		}
		workers = append(workers, worker)
	}

	return workers, nil
}

func (r *workerRepositoryImpl) UpdateHeartbeat(ctx context.Context, workerID string) error {
	key := workerKeyPrefix + workerID

	// 更新心跳时间
	if err := r.client.HSet(ctx, key, "last_heartbeat", time.Now().Unix()); err != nil {
		return fmt.Errorf("update heartbeat failed: %w", err)
	}

	// 续约 TTL
	return r.client.Expire(ctx, key, workerTTL)
}

func (r *workerRepositoryImpl) UpdateLoad(ctx context.Context, workerID string, load int) error {
	key := workerKeyPrefix + workerID
	return r.client.HSet(ctx, key, "current_load", load)
}

// 辅助函数
func flattenMap(m map[string]interface{}) []interface{} {
	result := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		result = append(result, k, v)
	}
	return result
}

func parseWorker(data map[string]string) (*model.Worker, error) {
	worker := &model.Worker{
		WorkerID:   data["worker_id"],
		WorkerName: data["worker_name"],
		Address:    data["address"],
		Status:     model.WorkerStatus(data["status"]),
	}

	// 解析数值字段
	fmt.Sscanf(data["capacity"], "%d", &worker.Capacity)
	fmt.Sscanf(data["current_load"], "%d", &worker.CurrentLoad)

	// 解析时间
	var timestamp int64
	fmt.Sscanf(data["last_heartbeat"], "%d", &timestamp)
	worker.LastHeartbeat = time.Unix(timestamp, 0)

	// 解析支持的任务类型
	if types := data["supported_types"]; types != "" {
		worker.SupportedTypes = strings.Split(types, ",")
	}

	return worker, nil
}
