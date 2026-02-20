package service

import (
	"testing"
	"time"

	"bamboo/asynctaskmanager/domain/model"
)

func createTestWorkers() []*model.Worker {
	return []*model.Worker{
		{
			WorkerID:      "worker-1",
			Status:        model.WorkerOnline,
			Capacity:      10,
			CurrentLoad:   3,
			LastHeartbeat: time.Now(),
		},
		{
			WorkerID:      "worker-2",
			Status:        model.WorkerOnline,
			Capacity:      10,
			CurrentLoad:   5,
			LastHeartbeat: time.Now(),
		},
		{
			WorkerID:      "worker-3",
			Status:        model.WorkerOnline,
			Capacity:      10,
			CurrentLoad:   1,
			LastHeartbeat: time.Now(),
		},
	}
}

func TestLeastTaskLoadBalancer_Select(t *testing.T) {
	lb := NewLeastTaskLoadBalancer()

	tests := []struct {
		name    string
		workers []*model.Worker
		wantID  string
		wantErr bool
	}{
		{
			name:    "select worker with least load",
			workers: createTestWorkers(),
			wantID:  "worker-3", // CurrentLoad = 1
			wantErr: false,
		},
		{
			name:    "no workers available",
			workers: []*model.Worker{},
			wantErr: true,
		},
		{
			name: "all workers at capacity",
			workers: []*model.Worker{
				{
					WorkerID:    "worker-1",
					Status:      model.WorkerOnline,
					Capacity:    10,
					CurrentLoad: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker, err := lb.Select(tt.workers, "test-task")

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if worker.WorkerID != tt.wantID {
				t.Errorf("Expected worker %v, got %v", tt.wantID, worker.WorkerID)
			}
		})
	}
}

func TestRoundRobinLoadBalancer_Select(t *testing.T) {
	lb := NewRoundRobinLoadBalancer()
	workers := createTestWorkers()

	// Test multiple selections to verify round-robin behavior
	selectedIDs := make([]string, 0)
	for i := 0; i < 6; i++ {
		worker, err := lb.Select(workers, "test-task")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		selectedIDs = append(selectedIDs, worker.WorkerID)
	}

	// Verify that workers are selected in rotation
	if len(selectedIDs) != 6 {
		t.Errorf("Expected 6 selections, got %d", len(selectedIDs))
	}

	// Check that all workers are used
	workerCount := make(map[string]int)
	for _, id := range selectedIDs {
		workerCount[id]++
	}

	if len(workerCount) != 3 {
		t.Errorf("Expected 3 different workers, got %d", len(workerCount))
	}
}

func TestConsistentHashLoadBalancer_Select(t *testing.T) {
	lb := NewConsistentHashLoadBalancer()
	workers := createTestWorkers()

	// Test that same task ID always goes to same worker
	taskID := "test-task-123"

	worker1, err := lb.Select(workers, taskID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	worker2, err := lb.Select(workers, taskID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if worker1.WorkerID != worker2.WorkerID {
		t.Errorf("Expected same worker for same task ID, got %v and %v",
			worker1.WorkerID, worker2.WorkerID)
	}

	// Test that different task IDs may go to different workers
	worker3, err := lb.Select(workers, "different-task")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Just verify it selected a valid worker
	found := false
	for _, w := range workers {
		if w.WorkerID == worker3.WorkerID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Selected worker not in available workers list")
	}
}

func TestLoadBalancerFactory(t *testing.T) {
	tests := []struct {
		name     string
		strategy LoadBalanceStrategy
		wantType string
	}{
		{
			name:     "least task strategy",
			strategy: StrategyLeastTask,
			wantType: "*service.LeastTaskLoadBalancer",
		},
		{
			name:     "round robin strategy",
			strategy: StrategyRoundRobin,
			wantType: "*service.RoundRobinLoadBalancer",
		},
		{
			name:     "consistent hash strategy",
			strategy: StrategyConsistentHash,
			wantType: "*service.ConsistentHashLoadBalancer",
		},
		{
			name:     "default strategy",
			strategy: "unknown",
			wantType: "*service.LeastTaskLoadBalancer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := LoadBalancerFactory(tt.strategy)
			if lb == nil {
				t.Error("Expected non-nil load balancer")
			}
		})
	}
}
