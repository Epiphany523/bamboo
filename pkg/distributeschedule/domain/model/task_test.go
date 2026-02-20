package model

import (
	"testing"
	"time"
)

func TestTask_CanRetry(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		maxRetries int
		want       bool
	}{
		{
			name: "can retry when failed and under max retries",
			task: &Task{
				Status:     TaskFailed,
				RetryCount: 2,
			},
			maxRetries: 3,
			want:       true,
		},
		{
			name: "cannot retry when reached max retries",
			task: &Task{
				Status:     TaskFailed,
				RetryCount: 3,
			},
			maxRetries: 3,
			want:       false,
		},
		{
			name: "cannot retry when not failed",
			task: &Task{
				Status:     TaskSuccess,
				RetryCount: 0,
			},
			maxRetries: 3,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.CanRetry(tt.maxRetries); got != tt.want {
				t.Errorf("Task.CanRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_IsTimeout(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		task    *Task
		timeout time.Duration
		want    bool
	}{
		{
			name: "timeout when running and exceeded timeout",
			task: &Task{
				Status:    TaskRunning,
				StartTime: now.Add(-10 * time.Minute),
			},
			timeout: 5 * time.Minute,
			want:    true,
		},
		{
			name: "not timeout when running and within timeout",
			task: &Task{
				Status:    TaskRunning,
				StartTime: now.Add(-2 * time.Minute),
			},
			timeout: 5 * time.Minute,
			want:    false,
		},
		{
			name: "not timeout when not running",
			task: &Task{
				Status:    TaskPending,
				StartTime: now.Add(-10 * time.Minute),
			},
			timeout: 5 * time.Minute,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsTimeout(tt.timeout); got != tt.want {
				t.Errorf("Task.IsTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
