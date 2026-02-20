package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/infrastructure/mysql"
)

func main() {
	log.Println("=== MySQL Repository Usage Example ===")

	// 创建 MySQL 客户端
	cfg := mysql.Config{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "password",
		Database: "asynctask",
		MaxOpen:  10,
		MaxIdle:  5,
		MaxLife:  time.Hour,
	}

	client, err := mysql.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create MySQL client: %v", err)
	}
	defer client.Close()

	log.Println("✓ MySQL client connected")

	// 初始化数据库表结构
	if err := client.InitSchema(); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}
	log.Println("✓ Database schema initialized")

	ctx := context.Background()

	// 创建仓储实例
	taskRepo := mysql.NewTaskRepository(client)
	taskConfigRepo := mysql.NewTaskConfigRepository(client)
	taskLogRepo := mysql.NewTaskLogRepository(client)
	workerRepo := mysql.NewWorkerRepository(client)

	// 示例 1: 创建任务配置
	log.Println("\n--- Example 1: Create Task Config ---")
	taskConfig := &model.TaskConfig{
		TaskType:        "email_task",
		TaskName:        "Send Email",
		Description:     "Send email notification",
		ExecutorType:    model.ExecutorTypeHTTP,
		ExecutorConfig: map[string]interface{}{
			"url":    "http://email-service/send",
			"method": "POST",
		},
		DefaultTimeout:  30,
		DefaultMaxRetry: 3,
		RetryStrategy:   model.RetryStrategyExponential,
		RetryDelay:      5,
		BackoffRate:     2.0,
		MaxConcurrent:   10,
		Enabled:         true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := taskConfigRepo.Create(ctx, taskConfig); err != nil {
		log.Printf("Failed to create task config: %v", err)
	} else {
		log.Printf("✓ Task config created: %s", taskConfig.TaskType)
	}

	// 示例 2: 查询任务配置
	log.Println("\n--- Example 2: Get Task Config ---")
	config, err := taskConfigRepo.GetByType(ctx, "email_task")
	if err != nil {
		log.Printf("Failed to get task config: %v", err)
	} else {
		log.Printf("✓ Task config found: %s (%s)", config.TaskName, config.TaskType)
		log.Printf("  Executor: %s", config.ExecutorType)
		log.Printf("  Timeout: %d seconds", config.DefaultTimeout)
		log.Printf("  Max Retry: %d", config.DefaultMaxRetry)
	}

	// 示例 3: 创建任务
	log.Println("\n--- Example 3: Create Task ---")
	task := &model.Task{
		TaskID:      "task-001",
		TaskType:    "email_task",
		Priority:    model.PriorityHigh,
		Status:      model.TaskPending,
		Payload: map[string]interface{}{
			"to":      "user@example.com",
			"subject": "Welcome",
			"body":    "Welcome to our service!",
		},
		RetryCount:  0,
		MaxRetry:    3,
		Timeout:     30,
		CreatedAt:   time.Now(),
		ScheduledAt: nil,
	}

	if err := taskRepo.Create(ctx, task); err != nil {
		log.Printf("Failed to create task: %v", err)
	} else {
		log.Printf("✓ Task created: %s (Priority: %s)", task.TaskID, task.Priority.String())
	}

	// 示例 4: 查询任务
	log.Println("\n--- Example 4: Get Task ---")
	foundTask, err := taskRepo.GetByID(ctx, "task-001")
	if err != nil {
		log.Printf("Failed to get task: %v", err)
	} else {
		log.Printf("✓ Task found: %s", foundTask.TaskID)
		log.Printf("  Type: %s", foundTask.TaskType)
		log.Printf("  Priority: %s", foundTask.Priority.String())
		log.Printf("  Status: %s", foundTask.Status)
		log.Printf("  Payload: %v", foundTask.Payload)
	}

	// 示例 5: 更新任务状态
	log.Println("\n--- Example 5: Update Task Status ---")
	foundTask.Status = model.TaskProcessing
	foundTask.WorkerID = "worker-001"
	now := time.Now()
	foundTask.StartedAt = &now

	if err := taskRepo.Update(ctx, foundTask); err != nil {
		log.Printf("Failed to update task: %v", err)
	} else {
		log.Printf("✓ Task updated: %s -> %s", foundTask.TaskID, foundTask.Status)
	}

	// 示例 6: 创建任务日志
	log.Println("\n--- Example 6: Create Task Log ---")
	taskLog := &model.TaskLog{
		LogID:      "log-001",
		TaskID:     "task-001",
		LogType:    model.LogTypeStateChange,
		FromStatus: model.TaskPending,
		ToStatus:   model.TaskProcessing,
		Message:    "Task started processing",
		Details: map[string]interface{}{
			"worker_id": "worker-001",
		},
		CreatedAt: time.Now(),
	}

	if err := taskLogRepo.Create(ctx, taskLog); err != nil {
		log.Printf("Failed to create task log: %v", err)
	} else {
		log.Printf("✓ Task log created: %s", taskLog.LogID)
	}

	// 示例 7: 查询任务日志
	log.Println("\n--- Example 7: Get Task Logs ---")
	logs, err := taskLogRepo.GetByTaskID(ctx, "task-001")
	if err != nil {
		log.Printf("Failed to get task logs: %v", err)
	} else {
		log.Printf("✓ Found %d log entries for task-001:", len(logs))
		for i, l := range logs {
			log.Printf("  %d. [%s] %s -> %s: %s", i+1, l.LogType, l.FromStatus, l.ToStatus, l.Message)
		}
	}

	// 示例 8: 注册 Worker
	log.Println("\n--- Example 8: Register Worker ---")
	worker := &model.Worker{
		WorkerID:       "worker-001",
		WorkerName:     "Worker 1",
		Address:        "localhost:8081",
		Status:         model.WorkerOnline,
		Capacity:       10,
		CurrentLoad:    1,
		SupportedTypes: []string{"email_task", "sms_task"},
		LastHeartbeat:  time.Now(),
	}

	if err := workerRepo.Register(ctx, worker); err != nil {
		log.Printf("Failed to register worker: %v", err)
	} else {
		log.Printf("✓ Worker registered: %s (%s)", worker.WorkerID, worker.WorkerName)
	}

	// 示例 9: 查询健康的 Worker
	log.Println("\n--- Example 9: Find Healthy Workers ---")
	healthyWorkers, err := workerRepo.FindHealthy(ctx, 30*time.Second)
	if err != nil {
		log.Printf("Failed to find healthy workers: %v", err)
	} else {
		log.Printf("✓ Found %d healthy workers:", len(healthyWorkers))
		for i, w := range healthyWorkers {
			log.Printf("  %d. %s - Load: %d/%d, Types: %v",
				i+1, w.WorkerName, w.CurrentLoad, w.Capacity, w.SupportedTypes)
		}
	}

	// 示例 10: 查询待执行的任务
	log.Println("\n--- Example 10: Find Pending Tasks ---")
	pendingTasks, err := taskRepo.FindPendingTasks(ctx, 10)
	if err != nil {
		log.Printf("Failed to find pending tasks: %v", err)
	} else {
		log.Printf("✓ Found %d pending tasks:", len(pendingTasks))
		for i, t := range pendingTasks {
			log.Printf("  %d. %s - Type: %s, Priority: %s",
				i+1, t.TaskID, t.TaskType, t.Priority.String())
		}
	}

	log.Println("\n=== All examples completed ===")
}
