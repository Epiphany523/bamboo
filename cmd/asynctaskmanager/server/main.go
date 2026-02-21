package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"bamboo/asynctaskmanager/application"
	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/service"
	"bamboo/asynctaskmanager/infrastructure/executor"
	"bamboo/asynctaskmanager/infrastructure/mysql"
	"bamboo/asynctaskmanager/infrastructure/redis"
)

type MysqlDSN string

// Parse 解析 MySQL DSN 字符串
// 格式: user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
func (dsn MysqlDSN) Parse() mysql.Config {
	config := mysql.Config{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "",
		Database: "asynctask",
		MaxOpen:  100,
		MaxIdle:  10,
		MaxLife:  time.Hour,
	}

	dsnStr := string(dsn)
	if dsnStr == "" {
		return config
	}

	// 正则表达式匹配 DSN 格式
	// user:password@tcp(host:port)/database?params
	re := regexp.MustCompile(`^([^:]+):([^@]*)@tcp\(([^:]+):(\d+)\)/([^?]+)`)
	matches := re.FindStringSubmatch(dsnStr)

	if len(matches) >= 6 {
		config.User = matches[1]
		config.Password = matches[2]
		config.Host = matches[3]

		if port, err := strconv.Atoi(matches[4]); err == nil {
			config.Port = port
		}

		config.Database = matches[5]
	} else {
		// 尝试简化格式解析
		// 提取用户名和密码
		if idx := strings.Index(dsnStr, "@"); idx > 0 {
			userPass := dsnStr[:idx]
			if colonIdx := strings.Index(userPass, ":"); colonIdx > 0 {
				config.User = userPass[:colonIdx]
				config.Password = userPass[colonIdx+1:]
			} else {
				config.User = userPass
			}
		}

		// 提取主机和端口
		if idx := strings.Index(dsnStr, "tcp("); idx >= 0 {
			hostPort := dsnStr[idx+4:]
			if endIdx := strings.Index(hostPort, ")"); endIdx > 0 {
				hostPort = hostPort[:endIdx]
				if colonIdx := strings.Index(hostPort, ":"); colonIdx > 0 {
					config.Host = hostPort[:colonIdx]
					if port, err := strconv.Atoi(hostPort[colonIdx+1:]); err == nil {
						config.Port = port
					}
				} else {
					config.Host = hostPort
				}
			}
		}

		// 提取数据库名
		if idx := strings.Index(dsnStr, ")/"); idx >= 0 {
			dbPart := dsnStr[idx+2:]
			if qIdx := strings.Index(dbPart, "?"); qIdx > 0 {
				config.Database = dbPart[:qIdx]
			} else {
				config.Database = dbPart
			}
		}
	}

	return config
}

// ServerConfig 服务器配置
type ServerConfig struct {
	ServerID   string
	GRPCPort   int
	RedisAddr  string
	MysqlDSN   MysqlDSN
	WorkerPort int
}

// Server 服务器实例
type Server struct {
	config           *ServerConfig
	grpcServer       *GRPCServer
	schedulerService *application.SchedulerService
	workerService    *application.WorkerService
	taskService      *application.TaskService
	redisClient      *redis.Client
	mysqlClient      *mysql.Client
	wg               sync.WaitGroup
}

// NewServer 创建服务器
func NewServer(cfg *ServerConfig) (*Server, error) {
	// 创建 Redis 客户端
	redisClient := redis.NewClient(cfg.RedisAddr, "", 0)
	if err := redisClient.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	// 创建 MySQL 客户端
	mysqlClient, err := mysql.NewClient(cfg.MysqlDSN.Parse())
	if err != nil {
		return nil, fmt.Errorf("mysql connection failed: %w", err)
	}

	// 创建仓储
	taskRepo := mysql.NewTaskRepository(mysqlClient)
	taskLogRepo := mysql.NewTaskLogRepository(mysqlClient)
	taskConfigRepo := mysql.NewTaskConfigRepository(mysqlClient)
	//workerRepo := mysql.NewWorkerRepository(mysqlClient)

	// 使用redis存放worker
	workerRepo := redis.NewWorkerRepository(redisClient)

	// 创建队列管理器
	queueManager := redis.NewQueueManager(redisClient)

	// 创建执行器注册表
	executorRegistry := executor.NewExecutorRegistry()

	// 注册 HTTP 执行器
	httpExecutor := executor.NewHTTPExecutor()
	if err := executorRegistry.Register(httpExecutor); err != nil {
		return nil, fmt.Errorf("register http executor failed: %w", err)
	}

	// 注册本地执行器
	localExecutor := executor.NewLocalExecutor()
	localExecutor.RegisterHandler("example_task", func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
		log.Printf("[%s] Executing example task with payload: %v", cfg.ServerID, payload)
		time.Sleep(2 * time.Second)
		return map[string]interface{}{
			"result":    "success",
			"server_id": cfg.ServerID,
			"timestamp": time.Now().Unix(),
		}, nil
	})
	if err := executorRegistry.Register(localExecutor); err != nil {
		return nil, fmt.Errorf("register local executor failed: %w", err)
	}

	// 创建任务服务
	taskService := application.NewTaskService(
		taskRepo,
		taskLogRepo,
		taskConfigRepo,
		queueManager,
	)

	// 创建 Scheduler 服务
	leaderElection := redis.NewLeaderElection(redisClient, cfg.ServerID)
	loadBalancer := service.LoadBalancerFactory(service.StrategyRoundRobin)

	schedulerService := application.NewSchedulerService(
		taskRepo,
		taskLogRepo,
		workerRepo,
		leaderElection,
		queueManager,
		loadBalancer,
		5*time.Second,
		30*time.Second,
	)

	// 创建 Worker
	worker := &model.Worker{
		WorkerID:       fmt.Sprintf("%s-worker", cfg.ServerID),
		WorkerName:     fmt.Sprintf("Worker-%s", cfg.ServerID),
		Address:        fmt.Sprintf("localhost:%d", cfg.WorkerPort),
		Status:         model.WorkerOnline,
		Capacity:       10,
		CurrentLoad:    0,
		SupportedTypes: []string{"example_task", "http_request"},
		LastHeartbeat:  time.Now(),
	}

	workerService := application.NewWorkerService(
		worker,
		taskRepo,
		taskLogRepo,
		workerRepo,
		queueManager,
		executorRegistry,
		5*time.Second,
	)

	// 创建 gRPC 服务器
	grpcServer := NewGRPCServer(taskService, cfg.GRPCPort)

	return &Server{
		config:           cfg,
		grpcServer:       grpcServer,
		schedulerService: schedulerService,
		workerService:    workerService,
		taskService:      taskService,
		redisClient:      redisClient,
		mysqlClient:      mysqlClient,
	}, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	log.Printf("[%s] Starting server...", s.config.ServerID)

	s.wg.Add(1)
	// 启动 Worker 服务
	go func() {
		defer s.wg.Done()
		if err := s.workerService.Start(ctx); err != nil {
			log.Printf("[%s] Worker service stopped: %v", s.config.ServerID, err)
		}
	}()

	s.wg.Add(1)
	// 启动 Scheduler 服务
	go func() {
		defer s.wg.Done()
		if err := s.schedulerService.Start(ctx); err != nil {
			log.Printf("[%s] Scheduler service stopped: %v", s.config.ServerID, err)
		}
	}()

	s.wg.Add(1)
	// 启动 gRPC 服务器
	go func() {
		defer s.wg.Done()
		if err := s.grpcServer.Start(); err != nil {
			log.Printf("[%s] gRPC server stopped: %v", s.config.ServerID, err)
		}
	}()

	log.Printf("[%s] Server started successfully (gRPC port: %d)", s.config.ServerID, s.config.GRPCPort)
	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	log.Printf("[%s] Stopping server...", s.config.ServerID)
	s.grpcServer.Stop()
	s.wg.Wait()
	s.workerService.Stop()
	s.redisClient.Close()
	s.mysqlClient.Close()
	return nil
}

// Run 运行服务器（阻塞）
func Run(cfg *ServerConfig) error {
	// 创建服务器
	server, err := NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务器
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Received shutdown signal")
	cancel()

	// 停止服务器
	if err := server.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	time.Sleep(1 * time.Second)
	log.Println("Server stopped")
	return nil
}
