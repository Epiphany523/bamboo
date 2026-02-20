package main

import (
	"context"
	"flag"
	"log"
	"time"
)

func main() {
	// 解析命令行参数
	serverAddr := flag.String("server", "localhost:9091", "gRPC server address")
	flag.Parse()

	log.Printf("=== Async Task Manager gRPC Client ===")
	log.Printf("Connecting to server: %s", *serverAddr)

	// 创建客户端
	grpcClient, err := NewGRPCClient(*serverAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer grpcClient.Close()

	ctx := context.Background()

	// 测试 1: 创建普通优先级任务
	log.Println("\n--- Test 1: Create Normal Priority Task ---")
	task1, err := grpcClient.CreateTask(ctx, "example_task", 0, map[string]interface{}{
		"message": "Hello from normal priority task",
		"number":  1,
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}
	log.Printf("✓ Task created: %s (Priority: %d, Status: %s)", task1, task1.Priority, task1.Status)

	// 测试 2: 创建高优先级任务
	log.Println("\n--- Test 2: Create High Priority Task ---")
	task2, err := grpcClient.CreateTask(ctx, "example_task", 1, map[string]interface{}{
		"message": "Urgent task!",
		"number":  2,
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}
	log.Printf("✓ Task created: %s (Priority: %d, Status: %s)", task2.TaskId, task2.Priority, task2.Status)

	// 测试 3: 批量创建任务
	log.Println("\n--- Test 3: Create Multiple Tasks ---")
	taskIDs := make([]string, 0)
	for i := 0; i < 5; i++ {
		priority := int32(0)
		if i%2 == 0 {
			priority = 1
		}

		task, err := grpcClient.CreateTask(ctx, "example_task", priority, map[string]interface{}{
			"message": "Batch task",
			"number":  i + 3,
		})
		if err != nil {
			log.Printf("Failed to create task %d: %v", i+1, err)
			continue
		}

		taskIDs = append(taskIDs, task.TaskId)
		log.Printf("✓ Task %d created: %s (Priority: %d)", i+1, task.TaskId, task.Priority)
	}

	// 等待任务执行
	log.Println("\n--- Waiting for tasks to execute (10 seconds) ---")
	time.Sleep(10 * time.Second)

	// 测试 4: 查询任务状态
	log.Println("\n--- Test 4: Query Task Status ---")
	for i, taskID := range []string{task1.TaskId, task2.TaskId} {
		task, err := grpcClient.GetTask(ctx, taskID)
		if err != nil {
			log.Printf("Failed to get task %d: %v", i+1, err)
			continue
		}

		log.Printf("Task %d: %s", i+1, taskID)
		log.Printf("  Status: %s", task.Status)
		log.Printf("  Priority: %d", task.Priority)
		log.Printf("  Worker: %s", task.WorkerId)
		if len(task.Result) > 0 {
			log.Printf("  Result: %v", task.Result)
		}
		if task.StartedAt != nil {
			log.Printf("  Started: %s", task.StartedAt.AsTime().Format("15:04:05"))
		}
		if task.CompletedAt != nil {
			log.Printf("  Completed: %s", task.CompletedAt.AsTime().Format("15:04:05"))
			duration := task.CompletedAt.AsTime().Sub(task.StartedAt.AsTime())
			log.Printf("  Duration: %v", duration)
		}
	}

	// 测试 5: 创建并立即取消任务
	log.Println("\n--- Test 5: Create and Cancel Task ---")
	task3, err := grpcClient.CreateTask(ctx, "example_task", 0, map[string]interface{}{
		"message": "This task will be cancelled",
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}
	log.Printf("✓ Task created: %s", task3.TaskId)

	// 立即取消
	time.Sleep(100 * time.Millisecond)
	success, message, err := grpcClient.CancelTask(ctx, task3.TaskId)
	if err != nil {
		log.Printf("Failed to cancel task: %v", err)
	} else if success {
		log.Printf("✓ Task cancelled: %s (%s)", task3.TaskId, message)
	} else {
		log.Printf("✗ Cancel failed: %s", message)
	}

	// 验证取消状态
	time.Sleep(1 * time.Second)
	task3, err = grpcClient.GetTask(ctx, task3.TaskId)
	if err != nil {
		log.Printf("Failed to get cancelled task: %v", err)
	} else {
		log.Printf("  Status after cancel: %s", task3.Status)
	}

	// 测试 6: 查询任务日志
	log.Println("\n--- Test 6: Query Task Logs ---")
	logs, err := grpcClient.GetTaskLogs(ctx, task1.TaskId)
	if err != nil {
		log.Printf("Failed to get task logs: %v", err)
	} else {
		log.Printf("Task %s has %d log entries:", task1.TaskId, len(logs))
		for i, logEntry := range logs {
			log.Printf("  %d. [%s] %s -> %s: %s",
				i+1,
				logEntry.LogType,
				logEntry.FromStatus,
				logEntry.ToStatus,
				logEntry.Message,
			)
		}
	}

	// 测试 7: 使用 WaitForTask 等待任务完成
	log.Println("\n--- Test 7: Wait for Task Completion ---")
	task4, err := grpcClient.CreateTask(ctx, "example_task", 1, map[string]interface{}{
		"message": "Task with wait",
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}
	log.Printf("✓ Task created: %s, waiting for completion...", task4.TaskId)

	completedTask, err := grpcClient.WaitForTask(ctx, task4.TaskId, 30*time.Second)
	if err != nil {
		log.Printf("Failed to wait for task: %v", err)
	} else {
		log.Printf("✓ Task completed: %s (Status: %s)", completedTask.TaskId, completedTask.Status)
	}

	// 统计信息
	log.Println("\n--- Test 8: Statistics ---")
	log.Printf("Total tasks created: %d", len(taskIDs)+4)

	log.Println("\n=== All tests completed ===")
}
