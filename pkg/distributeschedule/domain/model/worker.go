package model

import (
	"time"
)

// WorkerStatus Worker 状态
type WorkerStatus string

const (
	WorkerIdle WorkerStatus = "idle" // 空闲
	WorkerBusy WorkerStatus = "busy" // 繁忙
)

// Worker 工作节点（聚合根）
type Worker struct {
	ID            string
	Address       string
	Status        WorkerStatus
	LastHeartbeat time.Time
	Capacity      int // 最大并发任务数
	RunningTasks  int // 当前运行任务数
}

// IsHealthy 判断 Worker 是否健康
func (w *Worker) IsHealthy(timeout time.Duration) bool {
	return time.Since(w.LastHeartbeat) <= timeout
}

// CanAcceptTask 判断是否可以接受新任务
func (w *Worker) CanAcceptTask() bool {
	return w.RunningTasks < w.Capacity
}

// AcceptTask 接受任务
func (w *Worker) AcceptTask() {
	w.RunningTasks++
	if w.RunningTasks >= w.Capacity {
		w.Status = WorkerBusy
	}
}

// CompleteTask 完成任务
func (w *Worker) CompleteTask() {
	if w.RunningTasks > 0 {
		w.RunningTasks--
	}
	if w.RunningTasks < w.Capacity {
		w.Status = WorkerIdle
	}
}

// UpdateHeartbeat 更新心跳
func (w *Worker) UpdateHeartbeat() {
	w.LastHeartbeat = time.Now()
}
