package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/repository"
	"bamboo/pkg/distributeschedule/domain/service"
	"bamboo/pkg/distributeschedule/infrastructure/redis"
)

// WorkerService Worker 服务
type WorkerService struct {
	worker           *model.Worker
	taskRepo         repository.TaskRepository
	workerRepo       repository.WorkerRepository
	executorRegistry service.ExecutorRegistry
	heartbeatInterval time.Duration
}

// NewWorkerService 创建 Worker 服务
func NewWorkerService(
	worker *model.Worker,
	taskRepo repository.TaskRepository,
	workerRepo repository.WorkerRepository,
	executorRegistry service.ExecutorRegistry,
	heartbeatInterval time.Duration,
) *WorkerService {
	return &WorkerService{
		worker:           worker,
		taskRepo:         taskRepo,
		workerRepo:       workerRepo,
		executorRegistry: executorRegistry,
		heartbeatInterval: heartbeatInterval,
	}
}

// Start 启动 Worker 服务
func (s *WorkerService) Start(ctx context.Context) error {
	// 注册 Worker
	if err := s.workerRepo.Register(ctx, s.worker); err != nil {
		return fmt.Errorf("register worker failed: %w", err)
	}

	log.Printf("worker %s registered", s.worker.ID)

	// 启动心跳
	go s.heartbeatLoop(ctx)

	// 启动任务处理循环
	return s.taskLoop(ctx)
}

// heartbeatLoop 心跳循环
func (s *WorkerService) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.workerRepo.UpdateHeartbeat(ctx, s.worker.ID); err != nil {
				log.Printf("update heartbeat failed: %v", err)
			}
		}
	}
}

// taskLoop 任务处理循环
func (s *WorkerService) taskLoop(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.processTask(ctx); err != nil {
				log.Printf("process task failed: %v", err)
			}
		}
	}
}

// processTask 处理任务
func (s *WorkerService) processTask(ctx context.Context) error {
	// 从队列获取任务
	taskRepo, ok := s.taskRepo.(*redis.TaskRepositoryImpl)
	if !ok {
		return fmt.Errorf("invalid task repository type")
	}

	taskID, err := taskRepo.PopFromQueue(ctx, s.worker.ID)
	if err != nil {
		return nil // 队列为空
	}

	// 获取任务详情
	task, err := taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task failed: %w", err)
	}

	log.Printf("worker %s processing task %s", s.worker.ID, task.ID)

	// 获取执行器
	executor, found := s.executorRegistry.Get(task.ConfigID)
	if !found {
		task.MarkAsFailed(fmt.Sprintf("executor not found: %s", task.ConfigID))
		_ = taskRepo.Update(ctx, task)
		s.worker.CompleteTask()
		_ = s.workerRepo.Update(ctx, s.worker)
		return fmt.Errorf("executor not found: %s", task.ConfigID)
	}

	// 执行任务
	result, err := executor.Execute(ctx, task)
	if err != nil {
		task.MarkAsFailed(err.Error())
		log.Printf("task %s failed: %v", task.ID, err)
	} else {
		task.MarkAsSuccess(result)
		log.Printf("task %s succeeded", task.ID)
	}

	// 更新任务状态
	if err := taskRepo.Update(ctx, task); err != nil {
		log.Printf("update task failed: %v", err)
	}

	// 保存结果
	if err := taskRepo.SaveResult(ctx, task.ID, result); err != nil {
		log.Printf("save result failed: %v", err)
	}

	// 更新 Worker 状态
	s.worker.CompleteTask()
	if err := s.workerRepo.Update(ctx, s.worker); err != nil {
		log.Printf("update worker failed: %v", err)
	}

	return nil
}
