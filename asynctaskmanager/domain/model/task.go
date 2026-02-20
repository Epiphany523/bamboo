package model

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	StatusPending    TaskStatus = "PENDING"    // 待处理
	StatusProcessing TaskStatus = "PROCESSING" // 处理中
	StatusSuccess    TaskStatus = "SUCCESS"    // 成功
	StatusFailed     TaskStatus = "FAILED"     // 失败
	StatusCancelled  TaskStatus = "CANCELLED"  // 已取消
	StatusTimeout    TaskStatus = "TIMEOUT"    // 超时
)

// TaskPriority 任务优先级（值对象）
type TaskPriority int

const (
	PriorityNormal TaskPriority = 0 // 普通优先级
	PriorityHigh   TaskPriority = 1 // 高优先级
)

// IsHigh 判断是否是高优先级
func (p TaskPriority) IsHigh() bool {
	return p == PriorityHigh
}

// IsNormal 判断是否是普通优先级
func (p TaskPriority) IsNormal() bool {
	return p == PriorityNormal
}

// String 返回优先级字符串表示
func (p TaskPriority) String() string {
	switch p {
	case PriorityHigh:
		return "HIGH"
	case PriorityNormal:
		return "NORMAL"
	default:
		return "UNKNOWN"
	}
}

// Value 返回优先级数值
func (p TaskPriority) Value() int {
	return int(p)
}

// Task 任务实体（聚合根）
type Task struct {
	ID          int64
	TaskID      string
	TaskType    string
	Priority    TaskPriority
	Status      TaskStatus
	Payload     map[string]interface{}
	Result      map[string]interface{}
	ErrorMsg    string
	RetryCount  int
	MaxRetry    int
	Timeout     int
	WorkerID    string
	ScheduledAt time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CanRetry 判断任务是否可以重试
func (t *Task) CanRetry() bool {
	return (t.Status == StatusFailed || t.Status == StatusTimeout) && t.RetryCount < t.MaxRetry
}

// MarkAsProcessing 标记任务为处理中
func (t *Task) MarkAsProcessing(workerID string) {
	t.Status = StatusProcessing
	t.WorkerID = workerID
	now := time.Now()
	t.StartedAt = &now
}

// MarkAsSuccess 标记任务为成功
func (t *Task) MarkAsSuccess(result map[string]interface{}) {
	t.Status = StatusSuccess
	t.Result = result
	now := time.Now()
	t.CompletedAt = &now
}

// MarkAsFailed 标记任务为失败
func (t *Task) MarkAsFailed(errorMsg string) {
	t.Status = StatusFailed
	t.ErrorMsg = errorMsg
	now := time.Now()
	t.CompletedAt = &now
}

// MarkAsTimeout 标记任务为超时
func (t *Task) MarkAsTimeout() {
	t.Status = StatusTimeout
	t.ErrorMsg = "Task execution timeout"
	now := time.Now()
	t.CompletedAt = &now
}

// MarkAsCancelled 标记任务为已取消
func (t *Task) MarkAsCancelled() {
	t.Status = StatusCancelled
	now := time.Now()
	t.CompletedAt = &now
}

// MarkAsRetrying 标记任务为重试中
func (t *Task) MarkAsRetrying() {
	t.Status = StatusPending
	t.RetryCount++
	t.WorkerID = ""
	t.StartedAt = nil
	t.CompletedAt = nil
}

// IsTimeout 判断任务是否超时
func (t *Task) IsTimeout() bool {
	if t.Status != StatusProcessing || t.StartedAt == nil {
		return false
	}
	return time.Since(*t.StartedAt) > time.Duration(t.Timeout)*time.Second
}

// IsFinalState 判断是否是终态
func (t *Task) IsFinalState() bool {
	return t.Status == StatusSuccess || t.Status == StatusCancelled ||
		(t.Status == StatusFailed && !t.CanRetry()) ||
		(t.Status == StatusTimeout && !t.CanRetry())
}
