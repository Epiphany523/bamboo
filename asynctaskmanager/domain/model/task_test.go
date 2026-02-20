package model

import (
	"testing"
	"time"
)

func TestTaskPriority_IsHigh(t *testing.T) {
	tests := []struct {
		name     string
		priority TaskPriority
		want     bool
	}{
		{"high priority", PriorityHigh, true},
		{"normal priority", PriorityNormal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.IsHigh(); got != tt.want {
				t.Errorf("TaskPriority.IsHigh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskPriority_IsNormal(t *testing.T) {
	tests := []struct {
		name     string
		priority TaskPriority
		want     bool
	}{
		{"normal priority", PriorityNormal, true},
		{"high priority", PriorityHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.IsNormal(); got != tt.want {
				t.Errorf("TaskPriority.IsNormal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskPriority_String(t *testing.T) {
	tests := []struct {
		name     string
		priority TaskPriority
		want     string
	}{
		{"high priority", PriorityHigh, "HIGH"},
		{"normal priority", PriorityNormal, "NORMAL"},
		{"unknown priority", TaskPriority(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.want {
				t.Errorf("TaskPriority.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskPriority_Value(t *testing.T) {
	tests := []struct {
		name     string
		priority TaskPriority
		want     int
	}{
		{"high priority", PriorityHigh, 1},
		{"normal priority", PriorityNormal, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.Value(); got != tt.want {
				t.Errorf("TaskPriority.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		want       bool
	}{
		{
			name: "can retry when failed and under max retries",
			task: &Task{
				Status:     StatusFailed,
				RetryCount: 2,
				MaxRetry:   3,
			},
			want: true,
		},
		{
			name: "can retry when timeout and under max retries",
			task: &Task{
				Status:     StatusTimeout,
				RetryCount: 1,
				MaxRetry:   3,
			},
			want: true,
		},
		{
			name: "cannot retry when reached max retries",
			task: &Task{
				Status:     StatusFailed,
				RetryCount: 3,
				MaxRetry:   3,
			},
			want: false,
		},
		{
			name: "cannot retry when success",
			task: &Task{
				Status:     StatusSuccess,
				RetryCount: 0,
				MaxRetry:   3,
			},
			want: false,
		},
		{
			name: "cannot retry when pending",
			task: &Task{
				Status:     StatusPending,
				RetryCount: 0,
				MaxRetry:   3,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.CanRetry(); got != tt.want {
				t.Errorf("Task.CanRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_MarkAsProcessing(t *testing.T) {
	task := &Task{
		TaskID: "test-task",
		Status: StatusPending,
	}

	workerID := "worker-001"
	task.MarkAsProcessing(workerID)

	if task.Status != StatusProcessing {
		t.Errorf("Expected status %v, got %v", StatusProcessing, task.Status)
	}

	if task.WorkerID != workerID {
		t.Errorf("Expected workerID %v, got %v", workerID, task.WorkerID)
	}

	if task.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

func TestTask_MarkAsSuccess(t *testing.T) {
	task := &Task{
		TaskID: "test-task",
		Status: StatusProcessing,
	}

	result := map[string]interface{}{
		"message": "success",
	}

	task.MarkAsSuccess(result)

	if task.Status != StatusSuccess {
		t.Errorf("Expected status %v, got %v", StatusSuccess, task.Status)
	}

	if task.Result == nil {
		t.Error("Expected Result to be set")
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestTask_MarkAsFailed(t *testing.T) {
	task := &Task{
		TaskID: "test-task",
		Status: StatusProcessing,
	}

	errorMsg := "execution failed"
	task.MarkAsFailed(errorMsg)

	if task.Status != StatusFailed {
		t.Errorf("Expected status %v, got %v", StatusFailed, task.Status)
	}

	if task.ErrorMsg != errorMsg {
		t.Errorf("Expected error message %v, got %v", errorMsg, task.ErrorMsg)
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestTask_MarkAsTimeout(t *testing.T) {
	task := &Task{
		TaskID: "test-task",
		Status: StatusProcessing,
	}

	task.MarkAsTimeout()

	if task.Status != StatusTimeout {
		t.Errorf("Expected status %v, got %v", StatusTimeout, task.Status)
	}

	if task.ErrorMsg == "" {
		t.Error("Expected ErrorMsg to be set")
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestTask_MarkAsCancelled(t *testing.T) {
	task := &Task{
		TaskID: "test-task",
		Status: StatusProcessing,
	}

	task.MarkAsCancelled()

	if task.Status != StatusCancelled {
		t.Errorf("Expected status %v, got %v", StatusCancelled, task.Status)
	}

	if task.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestTask_MarkAsRetrying(t *testing.T) {
	task := &Task{
		TaskID:     "test-task",
		Status:     StatusFailed,
		RetryCount: 1,
		WorkerID:   "worker-001",
	}

	task.MarkAsRetrying()

	if task.Status != StatusPending {
		t.Errorf("Expected status %v, got %v", StatusPending, task.Status)
	}

	if task.RetryCount != 2 {
		t.Errorf("Expected RetryCount 2, got %v", task.RetryCount)
	}

	if task.WorkerID != "" {
		t.Errorf("Expected WorkerID to be empty, got %v", task.WorkerID)
	}

	if task.StartedAt != nil {
		t.Error("Expected StartedAt to be nil")
	}

	if task.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil")
	}
}

func TestTask_IsTimeout(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		task    *Task
		want    bool
	}{
		{
			name: "timeout when running and exceeded timeout",
			task: &Task{
				Status:    StatusProcessing,
				StartedAt: func() *time.Time { t := now.Add(-10 * time.Minute); return &t }(),
				Timeout:   300, // 5 minutes
			},
			want: true,
		},
		{
			name: "not timeout when running and within timeout",
			task: &Task{
				Status:    StatusProcessing,
				StartedAt: func() *time.Time { t := now.Add(-2 * time.Minute); return &t }(),
				Timeout:   300, // 5 minutes
			},
			want: false,
		},
		{
			name: "not timeout when not running",
			task: &Task{
				Status:    StatusPending,
				StartedAt: func() *time.Time { t := now.Add(-10 * time.Minute); return &t }(),
				Timeout:   300,
			},
			want: false,
		},
		{
			name: "not timeout when StartedAt is nil",
			task: &Task{
				Status:    StatusProcessing,
				StartedAt: nil,
				Timeout:   300,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsTimeout(); got != tt.want {
				t.Errorf("Task.IsTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_IsFinalState(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want bool
	}{
		{
			name: "success is final",
			task: &Task{Status: StatusSuccess},
			want: true,
		},
		{
			name: "cancelled is final",
			task: &Task{Status: StatusCancelled},
			want: true,
		},
		{
			name: "failed with no retry is final",
			task: &Task{
				Status:     StatusFailed,
				RetryCount: 3,
				MaxRetry:   3,
			},
			want: true,
		},
		{
			name: "failed with retry available is not final",
			task: &Task{
				Status:     StatusFailed,
				RetryCount: 1,
				MaxRetry:   3,
			},
			want: false,
		},
		{
			name: "pending is not final",
			task: &Task{Status: StatusPending},
			want: false,
		},
		{
			name: "processing is not final",
			task: &Task{Status: StatusProcessing},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsFinalState(); got != tt.want {
				t.Errorf("Task.IsFinalState() = %v, want %v", got, tt.want)
			}
		})
	}
}
