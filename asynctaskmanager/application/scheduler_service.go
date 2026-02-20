package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
	"bamboo/asynctaskmanager/domain/service"
	"bamboo/asynctaskmanager/infrastructure/redis"
)

// SchedulerService 调度服务
type SchedulerService struct {
	taskRepo         repository.TaskRepository
	taskLogRepo      repository.TaskLogRepository
	workerRepo       repository.WorkerRepository
	leaderElection   *redis.LeaderElection
	queueManager     *redis.QueueManager
	loadBalancer     service.LoadBalancer
	scanInterval     time.Duration
	heartbeatTimeout time.Duration
}

// NewSchedulerService 创建调度服务
func NewSchedulerService(
	taskRepo repository.TaskRepository,
	taskLogRepo repository.TaskLogRepository,
	workerRepo repository.WorkerRepository,
	leaderElection *redis.LeaderElection,
	queueManager *redis.QueueManager,
	loadBalancer service.LoadBalancer,
	scanInterval time.Duration,
	heartbeatTimeout time.Duration,
) *SchedulerService {
	return &SchedulerService{
		taskRepo:         taskRepo,
		taskLogRepo:      taskLogRepo,
		workerRepo:       workerRepo,
		leaderElection:   leaderElection,
		queueManager:     queueManager,
		loadBalancer:     loadBalancer,
		scanInterval:     scanInterval,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// Start 启动调度服务
func (s *SchedulerService) Start(ctx context.Context) error {
	// 尝试成为 Leader
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			acquired, err := s.leaderElection.TryAcquire(ctx)
			if err != nil {
				log.Printf("try acquire leader failed: %v", err)
				continue
			}

			if acquired {
				log.Println("became leader, starting schedule loop")
				return s.runAsLeader(ctx)
			}
		}
	}
}

// runAsLeader 作为 Leader 运行
func (s *SchedulerService) runAsLeader(ctx context.Context) error {
	scanTicker := time.NewTicker(s.scanInterval)
	renewTicker := time.NewTicker(3 * time.Second)
	timeoutTicker := time.NewTicker(30 * time.Second)

	defer scanTicker.Stop()
	defer renewTicker.Stop()
	defer timeoutTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = s.leaderElection.Release(ctx)
			return ctx.Err()

		case <-renewTicker.C:
			// 续约 Leader 锁
			if err := s.leaderElection.Renew(ctx); err != nil {
				log.Printf("renew leader lock failed: %v", err)
				return fmt.Errorf("lost leadership")
			}

		case <-scanTicker.C:
			// 扫描并调度任务
			if err := s.scanAndSchedule(ctx); err != nil {
				log.Printf("scan and schedule failed: %v", err)
			}

		case <-timeoutTicker.C:
			// 检查超时任务
			if err := s.checkTimeoutTasks(ctx); err != nil {
				log.Printf("check timeout tasks failed: %v", err)
			}
		}
	}
}

// scanAndSchedule 扫描并调度任务
func (s *SchedulerService) scanAndSchedule(ctx context.Context) error {
	// 从队列获取任务
	taskID, err := s.queueManager.PopTask(ctx)
	if err != nil {
		return nil // 队列为空
	}

	// 获取任务详情
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		log.Printf("get task failed: %v", err)
		return err
	}

	// 检查任务状态
	if task.Status != model.StatusPending {
		return nil
	}

	// 获取支持该任务类型的 Worker
	workers, err := s.workerRepo.FindByTaskType(ctx, task.TaskType)
	if err != nil {
		log.Printf("find workers failed: %v", err)
		// 重新放回队列
		_ = s.queueManager.PushTask(ctx, taskID, task.Priority)
		return err
	}

	// 过滤健康的 Worker
	healthyWorkers := make([]*model.Worker, 0)
	for _, worker := range workers {
		if worker.IsHealthy(s.heartbeatTimeout) && worker.CanAcceptTask() {
			healthyWorkers = append(healthyWorkers, worker)
		}
	}

	if len(healthyWorkers) == 0 {
		log.Printf("no available workers for task %s", taskID)
		// 重新放回队列
		_ = s.queueManager.PushTask(ctx, taskID, task.Priority)
		return nil
	}

	// 负载均衡选择 Worker
	worker, err := s.loadBalancer.Select(healthyWorkers, taskID)
	if err != nil {
		log.Printf("select worker failed: %v", err)
		// 重新放回队列
		_ = s.queueManager.PushTask(ctx, taskID, task.Priority)
		return err
	}

	// 更新任务状态
	task.MarkAsProcessing(worker.WorkerID)
	if err := s.taskRepo.Update(ctx, task); err != nil {
		log.Printf("update task failed: %v", err)
		return err
	}

	// 分配任务给 Worker
	if err := s.queueManager.PushToWorkerQueue(ctx, worker.WorkerID, taskID); err != nil {
		log.Printf("push to worker queue failed: %v", err)
		return err
	}

	// 更新 Worker 负载
	worker.AcceptTask()
	if err := s.workerRepo.UpdateLoad(ctx, worker.WorkerID, worker.CurrentLoad); err != nil {
		log.Printf("update worker load failed: %v", err)
	}

	// 记录日志
	logEntry := model.NewStateChangeLog(
		taskID,
		model.StatusPending,
		model.StatusProcessing,
		worker.WorkerID,
		"Task assigned to worker",
	)
	_ = s.taskLogRepo.Create(ctx, logEntry)

	log.Printf("scheduled task %s to worker %s", taskID, worker.WorkerID)

	return nil
}

// checkTimeoutTasks 检查超时任务
func (s *SchedulerService) checkTimeoutTasks(ctx context.Context) error {
	tasks, err := s.taskRepo.FindTimeoutTasks(ctx)
	if err != nil {
		return fmt.Errorf("find timeout tasks failed: %w", err)
	}

	for _, task := range tasks {
		log.Printf("task %s timeout, rescheduling", task.TaskID)

		// 标记为超时
		task.MarkAsTimeout()

		// 判断是否需要重试
		if task.CanRetry() {
			task.MarkAsRetrying()
			if err := s.taskRepo.Update(ctx, task); err != nil {
				log.Printf("update timeout task failed: %v", err)
				continue
			}

			// 重新推送到队列
			if err := s.queueManager.PushTask(ctx, task.TaskID, task.Priority); err != nil {
				log.Printf("push timeout task to queue failed: %v", err)
			}

			// 记录重试日志
			logEntry := model.NewRetryLog(
				task.TaskID,
				task.RetryCount,
				fmt.Sprintf("Task timeout, retry %d/%d", task.RetryCount, task.MaxRetry),
			)
			_ = s.taskLogRepo.Create(ctx, logEntry)
		} else {
			// 达到最大重试次数
			if err := s.taskRepo.Update(ctx, task); err != nil {
				log.Printf("update timeout task failed: %v", err)
			}

			// 记录错误日志
			logEntry := model.NewErrorLog(
				task.TaskID,
				task.WorkerID,
				"Task timeout and max retry reached",
				"",
			)
			_ = s.taskLogRepo.Create(ctx, logEntry)
		}
	}

	return nil
}
