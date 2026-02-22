package application

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
	"bamboo/asynctaskmanager/domain/service"
	"bamboo/asynctaskmanager/infrastructure/redis"
)

// WorkerService Worker 服务
type WorkerService struct {
	worker            *model.Worker
	taskRepo          repository.TaskRepository
	taskLogRepo       repository.TaskLogRepository
	workerRepo        repository.WorkerRepository
	queueManager      *redis.QueueManager
	executorRegistry  service.ExecutorRegistry
	heartbeatInterval time.Duration
	runningTasks      map[string]context.CancelFunc // 正在执行的任务及其取消函数
	taskMutex         sync.RWMutex                  // 保护 runningTasks 的并发访问
}

// NewWorkerService 创建 Worker 服务
func NewWorkerService(
	worker *model.Worker,
	taskRepo repository.TaskRepository,
	taskLogRepo repository.TaskLogRepository,
	workerRepo repository.WorkerRepository,
	queueManager *redis.QueueManager,
	executorRegistry service.ExecutorRegistry,
	heartbeatInterval time.Duration,
) *WorkerService {
	return &WorkerService{
		worker:            worker,
		taskRepo:          taskRepo,
		taskLogRepo:       taskLogRepo,
		workerRepo:        workerRepo,
		queueManager:      queueManager,
		executorRegistry:  executorRegistry,
		heartbeatInterval: heartbeatInterval,
		runningTasks:      make(map[string]context.CancelFunc),
	}
}

// Start 启动 Worker 服务
func (s *WorkerService) Start(ctx context.Context) error {
	// 注册 Worker
	s.worker.MarkOnline()
	s.worker.UpdateHeartbeat()

	if err := s.workerRepo.Register(ctx, s.worker); err != nil {
		return fmt.Errorf("register worker failed: %w", err)
	}

	log.Printf("worker %s registered", s.worker.WorkerID)

	// 启动心跳
	go s.heartbeatLoop(ctx)

	// 启动取消监听
	go s.cancelListener(ctx)

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
			// 标记为离线
			s.worker.MarkOffline()
			_ = s.workerRepo.Update(ctx, s.worker)
			return
		case <-ticker.C:
			if err := s.workerRepo.UpdateHeartbeat(ctx, s.worker.WorkerID); err != nil {
				log.Printf("update heartbeat failed: %v", err)
			}
		}
	}
}

// taskLoop 任务处理循环
func (s *WorkerService) taskLoop(ctx context.Context) error {
	//ticker := time.NewTicker(100 * time.Millisecond)
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
	taskID, err := s.queueManager.PopFromWorkerQueue(ctx, s.worker.WorkerID)
	if err != nil {
		return nil // 队列为空
	}

	// 获取任务详情
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task failed: %w", err)
	}

	log.Printf("worker %s processing task %s", s.worker.WorkerID, task.TaskID)

	// 检查取消标记
	cancelled, err := s.queueManager.CheckCancelMark(ctx, taskID)
	if err == nil && cancelled {
		task.MarkAsCancelled()
		_ = s.taskRepo.Update(ctx, task)
		_ = s.queueManager.RemoveCancelMark(ctx, taskID)

		// 更新负载
		s.worker.CompleteTask()
		_ = s.workerRepo.UpdateLoad(ctx, s.worker.WorkerID, s.worker.CurrentLoad)

		log.Printf("task %s cancelled", taskID)
		return nil
	}

	// 获取执行器
	executor, err := s.executorRegistry.Get(task.TaskType)
	if err != nil {
		task.MarkAsFailed(fmt.Sprintf("executor not found: %s", task.TaskType))
		_ = s.taskRepo.Update(ctx, task)

		// 更新负载
		s.worker.CompleteTask()
		_ = s.workerRepo.UpdateLoad(ctx, s.worker.WorkerID, s.worker.CurrentLoad)

		return fmt.Errorf("executor not found: %s", task.TaskType)
	}

	// 设置超时和取消上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(task.Timeout)*time.Second)
	defer cancel()

	// 注册正在执行的任务，以便支持取消
	s.taskMutex.Lock()
	s.runningTasks[taskID] = cancel
	s.taskMutex.Unlock()

	// 执行完成后清理
	defer func() {
		s.taskMutex.Lock()
		delete(s.runningTasks, taskID)
		s.taskMutex.Unlock()
	}()

	// 执行任务
	result, err := executor.Execute(execCtx, task)

	// 处理结果
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			// 超时
			task.MarkAsTimeout()
			log.Printf("task %s timeout", taskID)
		} else if execCtx.Err() == context.Canceled {
			// 被取消
			task.MarkAsCancelled()
			log.Printf("task %s cancelled during execution", taskID)

			// 记录取消日志
			logEntry := model.NewStateChangeLog(
				taskID,
				model.StatusProcessing,
				model.StatusCancelled,
				s.worker.WorkerID,
				"Task cancelled during execution",
			)
			_ = s.taskLogRepo.Create(ctx, logEntry)

			// 更新负载并返回
			s.worker.CompleteTask()
			_ = s.workerRepo.UpdateLoad(ctx, s.worker.WorkerID, s.worker.CurrentLoad)
			return nil
		} else {
			// 失败
			task.MarkAsFailed(err.Error())
			log.Printf("task %s failed: %v", taskID, err)
		}

		// 判断是否需要重试
		if task.CanRetry() {
			task.MarkAsRetrying()
			_ = s.taskRepo.Update(ctx, task)

			// 重新推送到队列
			_ = s.queueManager.PushTask(ctx, taskID, task.Priority)

			// 记录重试日志
			logEntry := model.NewRetryLog(
				taskID,
				task.RetryCount,
				fmt.Sprintf("Task failed, retry %d/%d", task.RetryCount, task.MaxRetry),
			)
			_ = s.taskLogRepo.Create(ctx, logEntry)
		} else {
			// 达到最大重试次数
			_ = s.taskRepo.Update(ctx, task)

			// 记录错误日志
			logEntry := model.NewErrorLog(
				taskID,
				s.worker.WorkerID,
				"Task failed and max retry reached",
				err.Error(),
			)
			_ = s.taskLogRepo.Create(ctx, logEntry)
		}
	} else {
		// 成功
		task.MarkAsSuccess(result)
		_ = s.taskRepo.Update(ctx, task)

		// 记录日志
		logEntry := model.NewStateChangeLog(
			taskID,
			model.StatusProcessing,
			model.StatusSuccess,
			s.worker.WorkerID,
			"Task completed successfully",
		)
		_ = s.taskLogRepo.Create(ctx, logEntry)

		log.Printf("task %s succeeded", taskID)
	}

	// 更新 Worker 负载
	s.worker.CompleteTask()
	if err := s.workerRepo.UpdateLoad(ctx, s.worker.WorkerID, s.worker.CurrentLoad); err != nil {
		log.Printf("update worker load failed: %v", err)
	}

	return nil
}

// cancelListener 监听任务取消通知
func (s *WorkerService) cancelListener(ctx context.Context) {
	// 订阅取消通知频道
	pubsub := s.queueManager.SubscribeCancelChannel(ctx, s.worker.WorkerID)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 接收取消通知消息
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			taskID := msg.Payload
			if taskID != "" {
				s.cancelRunningTask(taskID)
			}
		}
	}
}

// cancelRunningTask 取消正在执行的任务
func (s *WorkerService) cancelRunningTask(taskID string) {
	s.taskMutex.RLock()
	cancelFunc, exists := s.runningTasks[taskID]
	s.taskMutex.RUnlock()

	if exists {
		log.Printf("cancelling running task %s on worker %s", taskID, s.worker.WorkerID)
		cancelFunc()
	}
}
