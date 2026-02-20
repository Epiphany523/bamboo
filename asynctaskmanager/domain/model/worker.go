package model

import (
	"time"
)

// WorkerStatus Worker 状态
type WorkerStatus string

const (
	WorkerOnline  WorkerStatus = "ONLINE"  // 在线
	WorkerOffline WorkerStatus = "OFFLINE" // 离线
)

// Worker 工作节点（聚合根）
type Worker struct {
	ID             int64
	WorkerID       string
	WorkerName     string
	Address        string
	Status         WorkerStatus
	Capacity       int
	CurrentLoad    int
	SupportedTypes []string
	LastHeartbeat  time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsHealthy 判断 Worker 是否健康
func (w *Worker) IsHealthy(timeout time.Duration) bool {
	return time.Since(w.LastHeartbeat) <= timeout
}

// CanAcceptTask 判断是否可以接受新任务
func (w *Worker) CanAcceptTask() bool {
	return w.Status == WorkerOnline && w.CurrentLoad < w.Capacity
}

// AcceptTask 接受任务
func (w *Worker) AcceptTask() {
	w.CurrentLoad++
}

// CompleteTask 完成任务
func (w *Worker) CompleteTask() {
	if w.CurrentLoad > 0 {
		w.CurrentLoad--
	}
}

// UpdateHeartbeat 更新心跳
func (w *Worker) UpdateHeartbeat() {
	w.LastHeartbeat = time.Now()
}

// SupportsTaskType 判断是否支持指定任务类型
func (w *Worker) SupportsTaskType(taskType string) bool {
	for _, t := range w.SupportedTypes {
		if t == taskType {
			return true
		}
	}
	return false
}

// MarkOnline 标记为在线
func (w *Worker) MarkOnline() {
	w.Status = WorkerOnline
}

// MarkOffline 标记为离线
func (w *Worker) MarkOffline() {
	w.Status = WorkerOffline
}
