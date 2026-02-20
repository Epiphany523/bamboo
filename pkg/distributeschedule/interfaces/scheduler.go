package interfaces

import (
	"context"
	"fmt"
	"log"

	"bamboo/pkg/distributeschedule/application"
	"bamboo/pkg/distributeschedule/config"
	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/service"
	"bamboo/pkg/distributeschedule/infrastructure/executor"
	"bamboo/pkg/distributeschedule/infrastructure/redis"
)

// Scheduler 调度器（对外接口）
type Scheduler struct {
	config           *config.Config
	redisClient      *redis.Client
	scheduleService  *application.ScheduleService
	workerService    *application.WorkerService
	executorRegistry service.ExecutorRegistry
}

// NewScheduler 创建调度器
func NewScheduler(cfg *config.Config) (*Scheduler, error) {
	// 创建 Redis 客户端
	redisClient := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	// 测试连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	// 创建仓储
	taskRepo := redis.NewTaskRepository(redisClient)
	workerRepo := redis.NewWorkerRepository(redisClient)

	// 创建执行器注册表
	executorRegistry := executor.NewExecutorRegistry()

	// 注册默认执行器
	executorRegistry.Register(executor.NewHTTPExecutor())
	executorRegistry.Register(executor.NewLocalExecutor())

	// 创建 Leader 选举
	leaderElection := redis.NewLeaderElection(redisClient, cfg.Worker.ID)

	// 创建负载均衡器
	loadBalancer := service.LoadBalancerFactory(service.LoadBalanceStrategy(cfg.Schedule.LoadBalanceStrategy))

	// 创建调度服务
	scheduleService := application.NewScheduleService(
		taskRepo,
		workerRepo,
		leaderElection,
		loadBalancer,
		cfg.Schedule.ScanInterval,
		cfg.Worker.HeartbeatTimeout,
	)

	// 创建 Worker
	worker := &model.Worker{
		ID:            cfg.Worker.ID,
		Address:       cfg.Worker.Address,
		Status:        model.WorkerIdle,
		Capacity:      cfg.Worker.MaxConcurrentTasks,
		RunningTasks:  0,
	}

	// 创建 Worker 服务
	workerService := application.NewWorkerService(
		worker,
		taskRepo,
		workerRepo,
		executorRegistry,
		cfg.Worker.HeartbeatInterval,
	)

	return &Scheduler{
		config:           cfg,
		redisClient:      redisClient,
		scheduleService:  scheduleService,
		workerService:    workerService,
		executorRegistry: executorRegistry,
	}, nil
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	log.Println("starting scheduler...")

	// 启动 Worker 服务
	go func() {
		if err := s.workerService.Start(ctx); err != nil {
			log.Printf("worker service stopped: %v", err)
		}
	}()

	// 启动调度服务
	return s.scheduleService.Start(ctx)
}

// RegisterExecutor 注册执行器
func (s *Scheduler) RegisterExecutor(executor service.Executor) {
	s.executorRegistry.Register(executor)
}

// Close 关闭调度器
func (s *Scheduler) Close() error {
	return s.redisClient.Close()
}
