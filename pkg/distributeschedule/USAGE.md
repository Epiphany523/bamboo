# 使用指南

## 快速开始

### 1. 安装依赖

```bash
cd pkg/distributeschedule
go mod download
```

### 2. 启动 Redis

```bash
docker run -d -p 6379:6379 redis:latest
```

### 3. 基本使用

```go
package main

import (
	"context"
	"log"
	
	"bamboo/pkg/distributeschedule"
	"bamboo/pkg/distributeschedule/config"
)

func main() {
	// 创建配置
	cfg := config.DefaultConfig()
	cfg.Worker.ID = "worker-1"
	cfg.Worker.Address = "localhost:8080"
	
	// 创建调度框架
	ds, err := distributeschedule.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()
	
	// 启动
	ctx := context.Background()
	if err := ds.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## 自定义执行器

### 实现 Executor 接口

```go
type MyExecutor struct{}

func (e *MyExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
	// 执行业务逻辑
	return &model.TaskResult{
		Code:    0,
		Message: "success",
		Data:    "result data",
	}, nil
}

func (e *MyExecutor) Type() string {
	return "my_executor"
}

func (e *MyExecutor) Protocol() string {
	return "custom"
}
```

### 注册执行器

```go
ds.RegisterExecutor(&MyExecutor{})
```

## 配置说明

### 完整配置示例

```yaml
schedule:
  leader_lock_ttl: 10s
  leader_renew_interval: 3s
  scan_interval: 5s
  load_balance_strategy: least_task  # least_task/round_robin/consistent_hash

worker:
  id: worker-1
  address: localhost:8080
  heartbeat_interval: 10s
  heartbeat_timeout: 30s
  max_concurrent_tasks: 10

task:
  default_timeout: 5m
  max_retry: 3
  retry_delay: 10s
  backoff_rate: 2.0

redis:
  addr: localhost:6379
  password: ""
  db: 0
  pool_size: 10
```

## 部署方式

### 单机部署

```bash
# 启动单个实例
go run example/main.go
```

### 集群部署

```bash
# 启动多个实例（不同的 worker_id）
go run example/main.go -worker-id=worker-1 -port=8080
go run example/main.go -worker-id=worker-2 -port=8081
go run example/main.go -worker-id=worker-3 -port=8082
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: distribute-schedule
spec:
  replicas: 3
  selector:
    matchLabels:
      app: distribute-schedule
  template:
    metadata:
      labels:
        app: distribute-schedule
    spec:
      containers:
      - name: scheduler
        image: your-image:latest
        env:
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: REDIS_ADDR
          value: "redis-service:6379"
```

## 监控

### 查看 Leader

```bash
redis-cli GET schedule:leader
```

### 查看 Worker 列表

```bash
redis-cli KEYS "worker:pool:*"
```

### 查看任务队列

```bash
redis-cli LLEN "task:queue:worker-1"
```

## 故障处理

### Leader 故障

- 自动重新选举，无需人工干预
- 正在执行的任务不受影响

### Worker 故障

- 心跳超时自动从池中移除
- 超时任务会被重新调度

### Redis 故障

- 所有实例会尝试重连
- 建议使用 Redis 哨兵或集群模式

## 最佳实践

1. 合理设置并发任务数，避免资源耗尽
2. 为不同类型的任务设置合适的超时时间
3. 使用重试策略处理临时性故障
4. 定期清理过期的任务数据
5. 监控 Redis 的内存使用情况
