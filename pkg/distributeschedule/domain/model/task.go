package model

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"   // 待执行
	TaskRunning   TaskStatus = "running"   // 执行中
	TaskSuccess   TaskStatus = "success"   // 成功
	TaskFailed    TaskStatus = "failed"    // 失败
	TaskRetrying  TaskStatus = "retrying"  // 重试中
	TaskCancelled TaskStatus = "cancelled" // 已取消
)

// Task 任务实例（聚合根）
type Task struct {
	ID            string
	ConfigID      string
	Status        TaskStatus
	WorkerID      string
	ScheduledTime time.Time
	StartTime     time.Time
	EndTime       time.Time
	RetryCount    int
	Result        *TaskResult
	Error         string
}

// TaskResult 任务执行结果
type TaskResult struct {
	Code    int
	Message string
	Data    interface{}
}

// CanRetry 判断任务是否可以重试
func (t *Task) CanRetry(maxRetries int) bool {
	return t.Status == TaskFailed && t.RetryCount < maxRetries
}

// MarkAsRunning 标记任务为执行中
func (t *Task) MarkAsRunning(workerID string) {
	t.Status = TaskRunning
	t.WorkerID = workerID
	t.StartTime = time.Now()
}

// MarkAsSuccess 标记任务为成功
func (t *Task) MarkAsSuccess(result *TaskResult) {
	t.Status = TaskSuccess
	t.Result = result
	t.EndTime = time.Now()
}

// MarkAsFailed 标记任务为失败
func (t *Task) MarkAsFailed(err string) {
	t.Status = TaskFailed
	t.Error = err
	t.EndTime = time.Now()
}

// MarkAsRetrying 标记任务为重试中
func (t *Task) MarkAsRetrying() {
	t.Status = TaskRetrying
	t.RetryCount++
}

// IsTimeout 判断任务是否超时
func (t *Task) IsTimeout(timeout time.Duration) bool {
	if t.Status != TaskRunning {
		return false
	}
	return time.Since(t.StartTime) > timeout
}
