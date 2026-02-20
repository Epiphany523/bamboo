package model

import (
	"math"
	"time"
)

// RetryStrategy 重试策略
type RetryStrategy string

const (
	RetryStrategyFixed       RetryStrategy = "FIXED"       // 固定间隔
	RetryStrategyExponential RetryStrategy = "EXPONENTIAL" // 指数退避
)

// ExecutorType 执行器类型
type ExecutorType string

const (
	ExecutorTypeRPC   ExecutorType = "RPC"   // RPC 调用
	ExecutorTypeHTTP  ExecutorType = "HTTP"  // HTTP 调用
	ExecutorTypeLocal ExecutorType = "LOCAL" // 本地执行
)

// TaskConfig 任务配置（聚合根）
type TaskConfig struct {
	ID              int64
	TaskType        string
	TaskName        string
	Description     string
	ExecutorType    ExecutorType
	ExecutorConfig  map[string]interface{}
	DefaultTimeout  int
	DefaultMaxRetry int
	RetryStrategy   RetryStrategy
	RetryDelay      int
	BackoffRate     float64
	MaxConcurrent   int
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CalculateNextRetryTime 计算下次重试时间
func (tc *TaskConfig) CalculateNextRetryTime(retryCount int) time.Time {
	var delay time.Duration

	if tc.RetryStrategy == RetryStrategyExponential {
		// 指数退避
		delay = time.Duration(tc.RetryDelay) * time.Second *
			time.Duration(math.Pow(tc.BackoffRate, float64(retryCount)))
	} else {
		// 固定间隔
		delay = time.Duration(tc.RetryDelay) * time.Second
	}

	return time.Now().Add(delay)
}

// IsEnabled 判断任务配置是否启用
func (tc *TaskConfig) IsEnabled() bool {
	return tc.Enabled
}

// CreateTask 创建任务实例
func (tc *TaskConfig) CreateTask(taskID string, priority TaskPriority, payload map[string]interface{}) *Task {
	return &Task{
		TaskID:      taskID,
		TaskType:    tc.TaskType,
		Priority:    priority,
		Status:      StatusPending,
		Payload:     payload,
		MaxRetry:    tc.DefaultMaxRetry,
		Timeout:     tc.DefaultTimeout,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
