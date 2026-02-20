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
		{
			name: "healthy when heartbeat exactly at timeout",
			worker: &Worker{
				LastHeartbeat: now.Add(-30 * time.Second),
			},
			timeout: 30 * time.Second,
			want:    true,
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
			name: "can accept when under capacity and online",
			worker: &Worker{
				Status:      WorkerOnline,
				Capacity:    10,
				CurrentLoad: 5,
			},
			want: true,
		},
		{
			name: "cannot accept when at capacity",
			worker: &Worker{
				Status:      WorkerOnline,
				Capacity:    10,
				CurrentLoad: 10,
			},
			want: false,
		},
		{
			name: "cannot accept when offline",
			worker: &Worker{
				Status:      WorkerOffline,
				Capacity:    10,
				CurrentLoad: 5,
			},
			want: false,
		},
		{
			name: "can accept when load is zero",
			worker: &Worker{
				Status:      WorkerOnline,
				Capacity:    10,
				CurrentLoad: 0,
			},
			want: true,
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
		Status:      WorkerOnline,
		Capacity:    2,
		CurrentLoad: 0,
	}

	// Accept first task
	worker.AcceptTask()
	if worker.CurrentLoad != 1 {
		t.Errorf("Expected CurrentLoad = 1, got %d", worker.CurrentLoad)
	}

	// Accept second task
	worker.AcceptTask()
	if worker.CurrentLoad != 2 {
		t.Errorf("Expected CurrentLoad = 2, got %d", worker.CurrentLoad)
	}
}

func TestWorker_CompleteTask(t *testing.T) {
	worker := &Worker{
		Status:      WorkerOnline,
		Capacity:    2,
		CurrentLoad: 2,
	}

	// Complete first task
	worker.CompleteTask()
	if worker.CurrentLoad != 1 {
		t.Errorf("Expected CurrentLoad = 1, got %d", worker.CurrentLoad)
	}

	// Complete second task
	worker.CompleteTask()
	if worker.CurrentLoad != 0 {
		t.Errorf("Expected CurrentLoad = 0, got %d", worker.CurrentLoad)
	}

	// Complete when already at zero
	worker.CompleteTask()
	if worker.CurrentLoad != 0 {
		t.Errorf("Expected CurrentLoad = 0, got %d", worker.CurrentLoad)
	}
}

func TestWorker_UpdateHeartbeat(t *testing.T) {
	worker := &Worker{
		LastHeartbeat: time.Now().Add(-1 * time.Minute),
	}

	oldHeartbeat := worker.LastHeartbeat
	time.Sleep(10 * time.Millisecond)

	worker.UpdateHeartbeat()

	if !worker.LastHeartbeat.After(oldHeartbeat) {
		t.Error("Expected LastHeartbeat to be updated")
	}
}

func TestWorker_SupportsTaskType(t *testing.T) {
	worker := &Worker{
		SupportedTypes: []string{"send_email", "generate_report", "process_image"},
	}

	tests := []struct {
		name     string
		taskType string
		want     bool
	}{
		{"supports send_email", "send_email", true},
		{"supports generate_report", "generate_report", true},
		{"does not support unknown_task", "unknown_task", false},
		{"does not support empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := worker.SupportsTaskType(tt.taskType); got != tt.want {
				t.Errorf("Worker.SupportsTaskType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorker_MarkOnline(t *testing.T) {
	worker := &Worker{
		Status: WorkerOffline,
	}

	worker.MarkOnline()

	if worker.Status != WorkerOnline {
		t.Errorf("Expected status %v, got %v", WorkerOnline, worker.Status)
	}
}

func TestWorker_MarkOffline(t *testing.T) {
	worker := &Worker{
		Status: WorkerOnline,
	}

	worker.MarkOffline()

	if worker.Status != WorkerOffline {
		t.Errorf("Expected status %v, got %v", WorkerOffline, worker.Status)
	}
}
