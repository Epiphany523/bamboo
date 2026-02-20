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

// ScheduleService 调度服务
type ScheduleService struct {
	taskRepo         repository.TaskRepository
	workerRepo       repository.WorkerRepository
	leaderElection   *redis.LeaderElection
	loadBalancer     service.LoadBalancer
	scanInterval     time.Duration
	heartbeatTimeout time.Duration
}

// NewScheduleService 创建调度服务
func NewScheduleService(
	taskRepo repository.TaskRepository,
	workerRepo repository.WorkerRepository,
	leaderElection *redis.LeaderElection,
	loadBalancer service.LoadBalancer,
	scanInterval time.Duration,
	heartbeatTimeout time.Duration,
) *ScheduleService {
	return &ScheduleService{
		taskRepo:         taskRepo,
		workerRepo:       workerRepo,
		leaderElection:   leaderElection,
		loadBalancer:     loadBalancer,
		scanInterval:     scanInterval,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// Start 启动调度服务
func (s *ScheduleService) Start(ctx context.Context) error {
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
func (s *ScheduleService) runAsLeader(ctx context.Context) error {
	scanTicker := time.NewTicker(s.scanInterval)
	renewTicker := time.NewTicker(3 * time.Second)
	defer scanTicker.Stop()
	defer renewTicker.Stop()

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

			// 检查超时任务
			if err := s.checkTimeoutTasks(ctx); err != nil {
				log.Printf("check timeout tasks failed: %v", err)
			}
		}
	}
}

// scanAndSchedule 扫描并调度任务
func (s *ScheduleService) scanAndSchedule(ctx context.Context) error {
	// 查找待执行的任务
	tasks, err := s.taskRepo.FindPendingTasks(ctx, 100)
	if err != nil {
		return fmt.Errorf("find pending tasks failed: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	// 查找健康的 Worker
	workers, err := s.workerRepo.FindHealthy(ctx, s.heartbeatTimeout)
	if err != nil {
		return fmt.Errorf("find healthy workers failed: %w", err)
	}

	if len(workers) == 0 {
		log.Println("no available workers")
		return nil
	}

	// 分配任务
	for _, task := range tasks {
		worker, err := s.loadBalancer.Select(workers, task.ID)
		if err != nil {
			log.Printf("select worker failed: %v", err)
			continue
		}

		// 标记任务为执行中
		task.MarkAsRunning(worker.ID)
		if err := s.taskRepo.Update(ctx, task); err != nil {
			log.Printf("update task failed: %v", err)
			continue
		}

		// 推送任务到 Worker 队列
		if taskRepo, ok := s.taskRepo.(*redis.TaskRepositoryImpl); ok {
			if err := taskRepo.PushToQueue(ctx, worker.ID, task.ID); err != nil {
				log.Printf("push task to queue failed: %v", err)
				continue
			}
		} else {
			log.Printf("invalid task repository type")
			continue
		}

		// 更新 Worker 状态
		worker.AcceptTask()
		if err := s.workerRepo.Update(ctx, worker); err != nil {
			log.Printf("update worker failed: %v", err)
		}

		log.Printf("scheduled task %s to worker %s", task.ID, worker.ID)
	}

	return nil
}

// checkTimeoutTasks 检查超时任务
func (s *ScheduleService) checkTimeoutTasks(ctx context.Context) error {
	tasks, err := s.taskRepo.FindTimeoutTasks(ctx, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("find timeout tasks failed: %w", err)
	}

	for _, task := range tasks {
		log.Printf("task %s timeout, rescheduling", task.ID)

		// 重置任务状态
		task.Status = model.TaskPending
		task.WorkerID = ""

		if err := s.taskRepo.Update(ctx, task); err != nil {
			log.Printf("update timeout task failed: %v", err)
		}
	}

	return nil
}
