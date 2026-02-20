package model

import (
	"time"
)

// LogType 日志类型
type LogType string

const (
	LogTypeStateChange LogType = "STATE_CHANGE" // 状态变更
	LogTypeRetry       LogType = "RETRY"        // 重试
	LogTypeError       LogType = "ERROR"        // 错误
	LogTypeInfo        LogType = "INFO"         // 信息
)

// TaskLog 任务日志（实体）
type TaskLog struct {
	ID          int64
	TaskID      string
	LogType     LogType
	FromStatus  TaskStatus
	ToStatus    TaskStatus
	Message     string
	WorkerID    string
	RetryCount  int
	ErrorDetail string
	CreatedAt   time.Time
}

// NewStateChangeLog 创建状态变更日志
func NewStateChangeLog(taskID string, fromStatus, toStatus TaskStatus, workerID, message string) *TaskLog {
	return &TaskLog{
		TaskID:     taskID,
		LogType:    LogTypeStateChange,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		WorkerID:   workerID,
		Message:    message,
		CreatedAt:  time.Now(),
	}
}

// NewRetryLog 创建重试日志
func NewRetryLog(taskID string, retryCount int, message string) *TaskLog {
	return &TaskLog{
		TaskID:     taskID,
		LogType:    LogTypeRetry,
		RetryCount: retryCount,
		Message:    message,
		CreatedAt:  time.Now(),
	}
}

// NewErrorLog 创建错误日志
func NewErrorLog(taskID, workerID, message, errorDetail string) *TaskLog {
	return &TaskLog{
		TaskID:      taskID,
		LogType:     LogTypeError,
		WorkerID:    workerID,
		Message:     message,
		ErrorDetail: errorDetail,
		CreatedAt:   time.Now(),
	}
}

// NewInfoLog 创建信息日志
func NewInfoLog(taskID, message string) *TaskLog {
	return &TaskLog{
		TaskID:    taskID,
		LogType:   LogTypeInfo,
		Message:   message,
		CreatedAt: time.Now(),
	}
}
