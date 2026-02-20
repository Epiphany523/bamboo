# 项目结构说明

## DDD 分层架构

```
asynctaskmanager/
├── domain/                    # 领域层（核心业务逻辑）
│   ├── model/                # 领域模型
│   │   ├── task.go          # 任务实体（聚合根）
│   │   ├── task_config.go   # 任务配置实体（聚合根）
│   │   ├── task_log.go      # 任务日志实体
│   │   └── worker.go        # Worker 实体（聚合根）
│   ├── repository/           # 仓储接口
│   │   ├── task_repository.go
│   │   ├── task_config_repository.go
│   │   ├── task_log_repository.go
│   │   └── worker_repository.go
│   └── service/              # 领域服务
│       ├── executor.go       # 执行器接口
│       └── load_balancer.go  # 负载均衡器
│
├── application/              # 应用层（用例编排）
│   ├── scheduler_service.go  # 调度服务
│   ├── worker_service.go     # Worker 服务
│   └── task_service.go       # 任务服务
│
├── infrastructure/           # 基础设施层（技术实现）
│   ├── redis/               # Redis 实现
│   │   ├── redis_client.go
│   │   ├── leader_election.go
│   │   ├── worker_registry_impl.go
│   │   └── queue_manager.go
│   ├── executor/            # 执行器实现
│   │   ├── executor_registry_impl.go
│   │   ├── http_executor.go
│   │   └── local_executor.go
│   └── mysql/               # MySQL 实现（待实现）
│       ├── task_repository_impl.go
│       ├── task_config_repository_impl.go
│       └── task_log_repository_impl.go
│
├── config/                  # 配置
│   └── config.go
│
├── cmd/                     # 命令行入口
│   └── server/
│       └── main.go
│
├── go.mod                   # Go 模块定义
└── PROJECT_STRUCTURE.md     # 本文件
```

## 依赖关系

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│  (scheduler_service, worker_service)    │
└──────────────┬──────────────────────────┘
               │ depends on
               ↓
┌─────────────────────────────────────────┐
│          Domain Layer                   │
│  (model, repository, service)           │
└──────────────↑──────────────────────────┘
               │ implements
               │
┌──────────────┴──────────────────────────┐
│      Infrastructure Layer               │
│  (redis, executor, mysql)               │
└─────────────────────────────────────────┘
```

## 核心组件

### 1. Domain Layer（领域层）

#### Model（领域模型）
- **Task**: 任务实体，管理任务生命周期
- **TaskConfig**: 任务配置实体，定义任务执行规则
- **TaskLog**: 任务日志实体，记录任务执行过程
- **Worker**: Worker 实体，管理工作节点状态

#### Repository（仓储接口）
- 定义数据访问的抽象接口
- 不关心具体实现（MySQL、Redis 等）

#### Service（领域服务）
- **Executor**: 任务执行器接口
- **LoadBalancer**: 负载均衡器接口

### 2. Application Layer（应用层）

- **SchedulerService**: 调度服务，负责任务调度和分配
- **WorkerService**: Worker 服务，负责任务执行
- **TaskService**: 任务服务，负责任务的 CRUD 操作

### 3. Infrastructure Layer（基础设施层）

#### Redis 实现
- **RedisClient**: Redis 客户端封装
- **LeaderElection**: Leader 选举实现
- **WorkerRegistryImpl**: Worker 注册表实现
- **QueueManager**: 队列管理器

#### Executor 实现
- **ExecutorRegistry**: 执行器注册表
- **HTTPExecutor**: HTTP 执行器
- **LocalExecutor**: 本地执行器

#### MySQL 实现（待实现）
- **TaskRepositoryImpl**: 任务仓储实现
- **TaskConfigRepositoryImpl**: 任务配置仓储实现
- **TaskLogRepositoryImpl**: 任务日志仓储实现

## 已实现的功能

✅ 领域模型定义
✅ 仓储接口定义
✅ 领域服务（Executor、LoadBalancer）
✅ Redis 基础设施（客户端、Leader 选举、Worker 注册、队列管理）
✅ 执行器实现（HTTP、Local）
✅ 应用服务（Scheduler、Worker、Task）
✅ 配置管理
✅ 示例主程序

## 待实现的功能

⏳ MySQL 仓储实现
⏳ HTTP API 接口
⏳ 监控指标收集
⏳ 单元测试
⏳ 集成测试

## 如何扩展

### 1. 添加新的执行器

```go
// 1. 实现 Executor 接口
type MyExecutor struct{}

func (e *MyExecutor) Execute(ctx context.Context, task *model.Task) (map[string]interface{}, error) {
    // 实现逻辑
    return nil, nil
}

func (e *MyExecutor) Type() model.ExecutorType {
    return "my_type"
}

func (e *MyExecutor) SupportedTaskTypes() []string {
    return []string{"my_task"}
}

// 2. 注册到 Worker
executorRegistry.Register(&MyExecutor{})
```

### 2. 添加新的负载均衡策略

```go
// 1. 实现 LoadBalancer 接口
type MyLoadBalancer struct{}

func (lb *MyLoadBalancer) Select(workers []*model.Worker, taskID string) (*model.Worker, error) {
    // 实现逻辑
    return workers[0], nil
}

// 2. 在 LoadBalancerFactory 中添加
```

### 3. 实现 MySQL 仓储

```go
// 在 infrastructure/mysql 目录下实现
type taskRepositoryImpl struct {
    db *sql.DB
}

func (r *taskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
    // 实现 SQL 插入
    return nil
}

// ... 实现其他方法
```

## 设计原则

1. **依赖倒置**: 应用层依赖领域层接口，基础设施层实现接口
2. **单一职责**: 每个组件只负责一件事
3. **开闭原则**: 对扩展开放，对修改关闭
4. **接口隔离**: 定义小而精的接口
5. **依赖注入**: 通过构造函数注入依赖

## 运行说明

### 前置条件

- Go 1.23+
- Redis 6.0+
- MySQL 8.0+（待实现仓储后）

### 启动步骤

1. 启动 Redis
```bash
docker run -d -p 6379:6379 redis:latest
```

2. 下载依赖
```bash
cd asynctaskmanager
go mod download
```

3. 运行服务
```bash
go run cmd/server/main.go
```

### 注意事项

当前版本只实现了 Redis 相关功能，MySQL 仓储需要自行实现。

实现 MySQL 仓储后，取消 `cmd/server/main.go` 中的注释即可启动完整功能。
