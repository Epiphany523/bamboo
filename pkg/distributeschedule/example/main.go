package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bamboo/pkg/distributeschedule"
	"bamboo/pkg/distributeschedule/config"
	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/service"
	"bamboo/pkg/distributeschedule/infrastructure/executor"
)

func main() {
	// 创建配置
	cfg := config.DefaultConfig()
	cfg.Worker.ID = fmt.Sprintf("worker-%d", time.Now().Unix())
	cfg.Worker.Address = "localhost:8081"
	cfg.Redis.Addr = "localhost:6379"

	// 创建调度框架
	ds, err := distributeschedule.New(cfg)
	if err != nil {
		log.Fatalf("create scheduler failed: %v", err)
	}
	defer ds.Close()

	// 注册自定义执行器
	localExecutor := executor.NewLocalExecutor()
	localExecutor.RegisterHandler("hello", func(ctx context.Context, payload interface{}) (*model.TaskResult, error) {
		log.Printf("executing hello task with payload: %v", payload)
		return &model.TaskResult{
			Code:    0,
			Message: "success",
			Data:    "Hello, World!",
		}, nil
	})
	ds.RegisterExecutor(localExecutor)

	// 启动调度框架
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := ds.Start(ctx); err != nil {
			log.Printf("scheduler stopped: %v", err)
		}
	}()

	log.Println("scheduler started, press Ctrl+C to stop")

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}

// CustomExecutor 自定义执行器示例
type CustomExecutor struct{}

func (e *CustomExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
	log.Printf("executing custom task: %s", task.ID)

	// 执行自定义逻辑
	time.Sleep(2 * time.Second)

	return &model.TaskResult{
		Code:    0,
		Message: "success",
		Data:    "custom task completed",
	}, nil
}

func (e *CustomExecutor) Type() string {
	return "custom"
}

func (e *CustomExecutor) Protocol() string {
	return "custom"
}

var _ service.Executor = (*CustomExecutor)(nil)
