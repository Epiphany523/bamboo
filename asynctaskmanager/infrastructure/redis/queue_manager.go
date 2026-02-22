package redis

import (
	"bamboo/asynctaskmanager/domain/model"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	QueueHigh   = "queue:high"
	QueueNormal = "queue:normal"
)

// QueueManager 队列管理器
type QueueManager struct {
	client *Client
}

// NewQueueManager 创建队列管理器
func NewQueueManager(client *Client) *QueueManager {
	return &QueueManager{client: client}
}

// PushTask 推送任务到队列
func (qm *QueueManager) PushTask(ctx context.Context, taskID string, priority model.TaskPriority) error {
	queueName := QueueNormal
	if priority.IsHigh() {
		queueName = QueueHigh
	}

	return qm.client.LPush(ctx, queueName, taskID)
}

// PopTask 从队列弹出任务
func (qm *QueueManager) PopTask(ctx context.Context) (string, error) {
	// 优先从高优先级队列获取
	taskID, err := qm.client.RPop(ctx, QueueHigh)
	if err == nil {
		return taskID, nil
	}

	// 从普通优先级队列获取
	return qm.client.RPop(ctx, QueueNormal)
}

// GetQueueLength 获取队列长度
func (qm *QueueManager) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	return qm.client.LLen(ctx, queueName)
}

// PushToWorkerQueue 推送任务到 Worker 队列
func (qm *QueueManager) PushToWorkerQueue(ctx context.Context, workerID, taskID string) error {
	key := fmt.Sprintf("worker:%s:queue", workerID)
	return qm.client.LPush(ctx, key, taskID)
}

// PopFromWorkerQueue 从 Worker 队列弹出任务
func (qm *QueueManager) PopFromWorkerQueue(ctx context.Context, workerID string) (string, error) {
	key := fmt.Sprintf("worker:%s:queue", workerID)
	return qm.client.RPop(ctx, key)
}

// SetCancelMark 设置取消标记
func (qm *QueueManager) SetCancelMark(ctx context.Context, taskID string) error {
	key := fmt.Sprintf("task:cancel:%s", taskID)
	return qm.client.Set(ctx, key, "1", 3600)
}

// CheckCancelMark 检查取消标记
func (qm *QueueManager) CheckCancelMark(ctx context.Context, taskID string) (bool, error) {
	key := fmt.Sprintf("task:cancel:%s", taskID)
	exists, err := qm.client.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// RemoveCancelMark 移除取消标记
func (qm *QueueManager) RemoveCancelMark(ctx context.Context, taskID string) error {
	key := fmt.Sprintf("task:cancel:%s", taskID)
	return qm.client.Del(ctx, key)
}

// PublishCancelNotification 发布取消通知给指定 Worker
func (qm *QueueManager) PublishCancelNotification(ctx context.Context, workerID, taskID string) error {
	channel := fmt.Sprintf("cancel:%s", workerID)
	return qm.client.Publish(ctx, channel, taskID)
}

// SubscribeCancelChannel 订阅取消通知频道
func (qm *QueueManager) SubscribeCancelChannel(ctx context.Context, workerID string) *redis.PubSub {
	channel := fmt.Sprintf("cancel:%s", workerID)
	return qm.client.Subscribe(ctx, channel)
}

// GetWorkerQueueLength 获取 Worker 队列长度
func (qm *QueueManager) GetWorkerQueueLength(ctx context.Context, workerID string) (int64, error) {
	key := fmt.Sprintf("worker:%s:queue", workerID)
	return qm.client.LLen(ctx, key)
}
