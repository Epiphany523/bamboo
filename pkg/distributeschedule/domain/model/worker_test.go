package model

import (
	"testing"
	"time"
)

func TestWorker_IsHealthy(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		worker  *Worker
		timeout time.Duration
		want    bool
	}{
		{
			name: "healthy when heartbeat within timeout",
			worker: &Worker{
				LastHeartbeat: now.Add(-10 * time.Second),
			},
			timeout: 30 * time.Second,
			want:    true,
		},
		{
			name: "unhealthy when heartbeat exceeded timeout",
			worker: &Worker{
				LastHeartbeat: now.Add(-40 * time.Second),
			},
			timeout: 30 * time.Second,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.worker.IsHealthy(tt.timeout); got != tt.want {
				t.Errorf("Worker.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorker_CanAcceptTask(t *testing.T) {
	tests := []struct {
		name   string
		worker *Worker
		want   bool
	}{
		{
			name: "can accept when under capacity",
			worker: &Worker{
				Capacity:     10,
				RunningTasks: 5,
			},
			want: true,
		},
		{
			name: "cannot accept when at capacity",
			worker: &Worker{
				Capacity:     10,
				RunningTasks: 10,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.worker.CanAcceptTask(); got != tt.want {
				t.Errorf("Worker.CanAcceptTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorker_AcceptTask(t *testing.T) {
	worker := &Worker{
		Status:       WorkerIdle,
		Capacity:     2,
		RunningTasks: 0,
	}

	// Accept first task
	worker.AcceptTask()
	if worker.RunningTasks != 1 {
		t.Errorf("Expected RunningTasks = 1, got %d", worker.RunningTasks)
	}
	if worker.Status != WorkerIdle {
		t.Errorf("Expected Status = WorkerIdle, got %s", worker.Status)
	}

	// Accept second task (reach capacity)
	worker.AcceptTask()
	if worker.RunningTasks != 2 {
		t.Errorf("Expected RunningTasks = 2, got %d", worker.RunningTasks)
	}
	if worker.Status != WorkerBusy {
		t.Errorf("Expected Status = WorkerBusy, got %s", worker.Status)
	}
}

func TestWorker_CompleteTask(t *testing.T) {
	worker := &Worker{
		Status:       WorkerBusy,
		Capacity:     2,
		RunningTasks: 2,
	}

	// Complete first task
	worker.CompleteTask()
	if worker.RunningTasks != 1 {
		t.Errorf("Expected RunningTasks = 1, got %d", worker.RunningTasks)
	}
	if worker.Status != WorkerIdle {
		t.Errorf("Expected Status = WorkerIdle, got %s", worker.Status)
	}

	// Complete second task
	worker.CompleteTask()
	if worker.RunningTasks != 0 {
		t.Errorf("Expected RunningTasks = 0, got %d", worker.RunningTasks)
	}
}
