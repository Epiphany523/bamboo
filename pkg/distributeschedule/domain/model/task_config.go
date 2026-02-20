package model

import (
	"time"
)

// TaskConfig 任务配置（聚合根）
type TaskConfig struct {
	ID          string
	Name        string
	Type        string
	CronExpr    string
	Timeout     time.Duration
	RetryPolicy RetryPolicy
	Executor    string
	Payload     interface{}
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RetryPolicy 重试策略（值对象）
type RetryPolicy struct {
	MaxRetries  int           // 最大重试次数
	RetryDelay  time.Duration // 重试间隔
	BackoffRate float64       // 退避倍率（指数退避）
}

// CalculateNextRetryTime 计算下次重试时间
func (rp *RetryPolicy) CalculateNextRetryTime(retryCount int) time.Time {
	delay := float64(rp.RetryDelay)
	for i := 0; i < retryCount; i++ {
		delay *= rp.BackoffRate
	}
	return time.Now().Add(time.Duration(delay))
}

// IsEnabled 判断任务配置是否启用
func (tc *TaskConfig) IsEnabled() bool {
	return tc.Enabled
}

// CreateTask 创建任务实例
func (tc *TaskConfig) CreateTask(scheduledTime time.Time) *Task {
	return &Task{
		ID:            generateTaskID(),
		ConfigID:      tc.ID,
		Status:        TaskPending,
		ScheduledTime: scheduledTime,
		RetryCount:    0,
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
