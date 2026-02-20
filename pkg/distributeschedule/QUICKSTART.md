# 快速入门

## 5 分钟上手

### 1. 准备环境

确保已安装：
- Go 1.21+
- Redis 6.0+
- Docker（可选，用于快速启动 Redis）

### 2. 启动 Redis

使用 Docker：
```bash
make docker-redis
```

或手动启动：
```bash
redis-server
```

### 3. 下载依赖

```bash
cd pkg/distributeschedule
make deps
```

### 4. 运行示例

```bash
make run
```

你会看到类似输出：
```
worker worker-1234567890 registered
scheduler started, press Ctrl+C to stop
became leader, starting schedule loop
```

### 5. 测试多实例

打开 3 个终端，分别运行：

终端 1：
```bash
WORKER_ID=worker-1 go run example/main.go
```

终端 2：
```bash
WORKER_ID=worker-2 go run example/main.go
```

终端 3：
```bash
WORKER_ID=worker-3 go run example/main.go
```

观察日志，你会看到：
- 只有一个实例成为 Leader
- 所有实例都注册为 Worker
- Leader 负责任务调度

## 基本使用

### 创建调度器

```go
package main

import (
	"context"
	"log"
	
	"bamboo/pkg/distributeschedule"
	"bamboo/pkg/distributeschedule/config"
)

func main() {
	// 1. 创建配置
	cfg := config.DefaultConfig()
	cfg.Worker.ID = "worker-1"
	cfg.Worker.Address = "localhost:8080"
	cfg.Redis.Addr = "localhost:6379"
	
	// 2. 创建调度器
	ds, err := distributeschedule.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()
	
	// 3. 启动
	ctx := context.Background()
	if err := ds.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
```

### 自定义执行器

```go
import (
	"context"
	"log"
	
	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/infrastructure/executor"
)

// 创建本地执行器
localExecutor := executor.NewLocalExecutor()

// 注册处理函数
localExecutor.RegisterHandler("send_email", func(ctx context.Context, payload interface{}) (*model.TaskResult, error) {
	log.Printf("Sending email: %v", payload)
	
	// 执行发送邮件逻辑
	// ...
	
	return &model.TaskResult{
		Code:    0,
		Message: "Email sent successfully",
		Data:    nil,
	}, nil
})

// 注册到调度器
ds.RegisterExecutor(localExecutor)
```

### HTTP 执行器示例

```go
// HTTP 执行器已内置，无需额外配置
// 任务 payload 格式：
payload := map[string]interface{}{
	"url":    "https://api.example.com/webhook",
	"method": "POST",
	"body": map[string]interface{}{
		"event": "task_completed",
		"data":  "some data",
	},
}
```

## 配置说明

### 最小配置

```go
cfg := &config.Config{
	Worker: config.WorkerConfig{
		ID:      "worker-1",
		Address: "localhost:8080",
	},
	Redis: config.RedisConfig{
		Addr: "localhost:6379",
	},
}
```

### 完整配置

```go
cfg := &config.Config{
	Schedule: config.ScheduleConfig{
		LeaderLockTTL:       10 * time.Second,
		LeaderRenewInterval: 3 * time.Second,
		ScanInterval:        5 * time.Second,
		LoadBalanceStrategy: "least_task", // least_task, round_robin, consistent_hash
	},
	Worker: config.WorkerConfig{
		ID:                  "worker-1",
		Address:             "localhost:8080",
		HeartbeatInterval:   10 * time.Second,
		HeartbeatTimeout:    30 * time.Second,
		MaxConcurrentTasks:  10,
	},
	Task: config.TaskConfig{
		DefaultTimeout: 5 * time.Minute,
		MaxRetry:       3,
		RetryDelay:     10 * time.Second,
		BackoffRate:    2.0,
	},
	Redis: config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 10,
	},
}
```

## 常用命令

```bash
# 运行测试
make test

# 生成测试覆盖率报告
make test-coverage

# 编译
make build

# 运行
make run

# 代码格式化
make fmt

# 清理
make clean

# 启动 Redis
make docker-redis

# 停止 Redis
make docker-redis-stop
```

## 监控和调试

### 查看 Leader

```bash
redis-cli GET schedule:leader
```

### 查看所有 Worker

```bash
redis-cli KEYS "worker:pool:*"
```

### 查看 Worker 详情

```bash
redis-cli GET "worker:pool:worker-1"
```

### 查看任务队列长度

```bash
redis-cli LLEN "task:queue:worker-1"
```

### 查看任务详情

```bash
redis-cli GET "task:detail:20240216150405-abc12345"
```

## 故障排查

### 问题：无法连接 Redis

检查：
```bash
redis-cli ping
```

解决：
- 确保 Redis 已启动
- 检查配置中的 Redis 地址

### 问题：没有 Worker 可用

检查：
```bash
redis-cli KEYS "worker:pool:*"
```

解决：
- 确保 Worker 服务已启动
- 检查心跳配置

### 问题：任务不执行

检查：
1. 是否有 Leader：`redis-cli GET schedule:leader`
2. 是否有待执行任务：`redis-cli KEYS "task:detail:*"`
3. 查看日志输出

## 下一步

- 阅读 [架构文档](./ARCHITECTURE.md) 了解设计细节
- 阅读 [使用指南](./USAGE.md) 了解高级用法
- 阅读 [项目结构](./PROJECT_STRUCTURE.md) 了解代码组织

## 常见场景

### 场景 1：定时任务

```go
// 创建定时任务配置
taskConfig := &model.TaskConfig{
	ID:       "daily-report",
	Name:     "每日报表",
	Type:     "http",
	CronExpr: "0 0 9 * * *", // 每天 9 点
	Timeout:  5 * time.Minute,
	RetryPolicy: model.RetryPolicy{
		MaxRetries:  3,
		RetryDelay:  10 * time.Second,
		BackoffRate: 2.0,
	},
	Executor: "http",
	Payload: map[string]interface{}{
		"url":    "https://api.example.com/report",
		"method": "POST",
	},
	Enabled: true,
}
```

### 场景 2：异步任务

```go
// 创建即时任务
task := &model.Task{
	ID:            generateTaskID(),
	ConfigID:      "send-notification",
	Status:        model.TaskPending,
	ScheduledTime: time.Now(),
}

// 保存任务
taskRepo.Save(ctx, task)
```

### 场景 3：批量任务

```go
// 批量创建任务
for i := 0; i < 100; i++ {
	task := &model.Task{
		ID:            generateTaskID(),
		ConfigID:      "batch-process",
		Status:        model.TaskPending,
		ScheduledTime: time.Now(),
	}
	taskRepo.Save(ctx, task)
}
```

## 性能优化建议

1. **合理设置并发数**：根据机器资源调整 `MaxConcurrentTasks`
2. **选择合适的负载均衡策略**：
   - 无状态任务：`least_task`（默认）
   - 有状态任务：`consistent_hash`
3. **调整扫描间隔**：根据任务量调整 `ScanInterval`
4. **使用连接池**：Redis 连接池大小根据并发量调整

## 生产环境建议

1. **使用 Redis 哨兵或集群**：提高可用性
2. **监控指标**：
   - Leader 选举次数
   - 任务执行成功率
   - Worker 健康状态
3. **日志收集**：集中收集和分析日志
4. **资源限制**：设置合理的 CPU 和内存限制
5. **备份策略**：定期备份 Redis 数据
