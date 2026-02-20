package application

import (
	"context"
	"fmt"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/repository"
	"bamboo/asynctaskmanager/infrastructure/redis"

	"github.com/google/uuid"
)

// TaskService 任务服务
type TaskService struct {
	taskRepo       repository.TaskRepository
	taskLogRepo    repository.TaskLogRepository
	taskConfigRepo repository.TaskConfigRepository
	queueManager   *redis.QueueManager
}

// NewTaskService 创建任务服务
func NewTaskService(
	taskRepo repository.TaskRepository,
	taskLogRepo repository.TaskLogRepository,
	taskConfigRepo repository.TaskConfigRepository,
	queueManager *redis.QueueManager,
) *TaskService {
	return &TaskService{
		taskRepo:       taskRepo,
		taskLogRepo:    taskLogRepo,
		taskConfigRepo: taskConfigRepo,
		queueManager:   queueManager,
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, taskType string, priority model.TaskPriority, payload map[string]interface{}) (*model.Task, error) {
	// 获取任务配置
	config, err := s.taskConfigRepo.GetByType(ctx, taskType)
	if err != nil {
		return nil, fmt.Errorf("get task config failed: %w", err)
	}

	if !config.IsEnabled() {
		return nil, fmt.Errorf("task type %s is disabled", taskType)
	}

	// 生成任务ID
	taskID := uuid.New().String()

	// 创建任务
	task := config.CreateTask(taskID, priority, payload)

	// 保存任务
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task failed: %w", err)
	}

	// 记录日志
	logEntry := model.NewStateChangeLog(
		taskID,
		"",
		model.StatusPending,
		"",
		"Task created",
	)
	_ = s.taskLogRepo.Create(ctx, logEntry)

	// 推送到队列
	if err := s.queueManager.PushTask(ctx, taskID, priority); err != nil {
		return nil, fmt.Errorf("push task to queue failed: %w", err)
	}

	return task, nil
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	return s.taskRepo.GetByID(ctx, taskID)
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error {
	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task failed: %w", err)
	}

	// 检查任务状态
	if task.Status != model.StatusPending && task.Status != model.StatusProcessing {
		return fmt.Errorf("task cannot be cancelled, current status: %s", task.Status)
	}

	if task.Status == model.StatusPending {
		// 直接标记为已取消
		task.MarkAsCancelled()
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("update task failed: %w", err)
		}
	} else {
		// 设置取消标记，Worker 会检测到
		if err := s.queueManager.SetCancelMark(ctx, taskID); err != nil {
			return fmt.Errorf("set cancel mark failed: %w", err)
		}
	}

	// 记录日志
	logEntry := model.NewStateChangeLog(
		taskID,
		task.Status,
		model.StatusCancelled,
		"",
		"Task cancelled by user",
	)
	_ = s.taskLogRepo.Create(ctx, logEntry)

	return nil
}

// GetTaskLogs 获取任务日志
func (s *TaskService) GetTaskLogs(ctx context.Context, taskID string) ([]*model.TaskLog, error) {
	return s.taskLogRepo.GetByTaskID(ctx, taskID)
}
