# 项目结构说明

## 目录结构

```
pkg/distributeschedule/
├── domain/                           # 领域层（核心业务逻辑）
│   ├── model/                       # 领域模型
│   │   ├── task.go                 # 任务实体（聚合根）
│   │   ├── task_config.go          # 任务配置实体（聚合根）
│   │   ├── worker.go               # Worker 实体（聚合根）
│   │   ├── task_test.go            # 任务单元测试
│   │   └── worker_test.go          # Worker 单元测试
│   ├── repository/                  # 仓储接口（数据访问抽象）
│   │   ├── task_repository.go      # 任务仓储接口
│   │   ├── task_config_repository.go # 任务配置仓储接口
│   │   └── worker_repository.go    # Worker 仓储接口
│   └── service/                     # 领域服务
│       ├── executor.go             # 执行器接口
│       └── load_balancer.go        # 负载均衡器（含三种策略实现）
│
├── application/                     # 应用层（用例编排）
│   ├── schedule_service.go         # 调度服务（Leader 选举、任务分发）
│   └── worker_service.go           # Worker 服务（任务执行、心跳维护）
│
├── infrastructure/                  # 基础设施层（技术实现）
│   ├── redis/                      # Redis 实现
│   │   ├── redis_client.go        # Redis 客户端封装
│   │   ├── leader_election.go     # Leader 选举实现
│   │   ├── task_repository_impl.go # 任务仓储实现
│   │   └── worker_repository_impl.go # Worker 仓储实现
│   └── executor/                   # 执行器实现
│       ├── executor_registry_impl.go # 执行器注册表
│       ├── http_executor.go       # HTTP 执行器
│       └── local_executor.go      # 本地执行器
│
├── interfaces/                      # 接口层（对外接口）
│   └── scheduler.go                # 调度器接口（组装各层组件）
│
├── config/                          # 配置
│   └── config.go                   # 配置定义和默认值
│
├── example/                         # 示例代码
│   └── main.go                     # 使用示例
│
├── distributeschedule.go           # 门面（统一入口）
├── go.mod                          # Go 模块定义
├── Makefile                        # 构建脚本
├── readme.md                       # 项目说明
├── ARCHITECTURE.md                 # 架构文档
├── USAGE.md                        # 使用指南
└── PROJECT_STRUCTURE.md            # 本文件
```

## 分层职责

### 1. Domain Layer（领域层）

领域层是系统的核心，包含业务逻辑和规则，不依赖任何外部技术。

#### Model（领域模型）
- **聚合根（Aggregate Root）**：
  - `Task`: 任务实例，管理任务生命周期
  - `TaskConfig`: 任务配置，定义任务执行规则
  - `Worker`: 工作节点，管理任务容量和状态

- **值对象（Value Object）**：
  - `RetryPolicy`: 重试策略
  - `TaskResult`: 任务执行结果
  - `TaskStatus`: 任务状态枚举
  - `WorkerStatus`: Worker 状态枚举

#### Repository（仓储接口）
定义数据访问的抽象接口，不关心具体实现：
- `TaskRepository`: 任务数据访问
- `TaskConfigRepository`: 任务配置数据访问
- `WorkerRepository`: Worker 数据访问

#### Service（领域服务）
跨聚合根的业务逻辑：
- `Executor`: 任务执行接口（支持多种协议）
- `LoadBalancer`: 负载均衡接口（支持多种策略）

### 2. Application Layer（应用层）

应用层编排领域对象完成用例，不包含业务规则。

- `ScheduleService`: 调度服务
  - Leader 选举和续约
  - 任务扫描和分发
  - 超时任务检测

- `WorkerService`: Worker 服务
  - Worker 注册
  - 心跳维护
  - 任务执行循环

### 3. Infrastructure Layer（基础设施层）

基础设施层提供技术实现，支撑上层业务逻辑。

#### Redis 实现
- `RedisClient`: Redis 客户端封装
- `LeaderElection`: 基于 Redis 的 Leader 选举
- `TaskRepositoryImpl`: 任务仓储的 Redis 实现
- `WorkerRepositoryImpl`: Worker 仓储的 Redis 实现

#### Executor 实现
- `ExecutorRegistry`: 执行器注册表
- `HTTPExecutor`: HTTP 协议执行器
- `LocalExecutor`: 本地函数执行器

### 4. Interfaces Layer（接口层）

接口层是系统的入口，组装各层组件。

- `Scheduler`: 调度器接口
  - 创建和初始化所有组件
  - 提供统一的启动和关闭接口
  - 支持执行器注册

### 5. Config（配置层）

- `Config`: 配置定义
  - Schedule 配置
  - Worker 配置
  - Task 配置
  - Redis 配置

## 依赖关系

```
┌─────────────────────────────────────────┐
│         Interfaces Layer                │
│    (scheduler.go, distributeschedule.go)│
└──────────────┬──────────────────────────┘
               │ depends on
               ↓
┌─────────────────────────────────────────┐
│        Application Layer                │
│  (schedule_service, worker_service)     │
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
│  (redis, executor)                      │
└─────────────────────────────────────────┘
```

依赖规则：
- 外层依赖内层
- 内层不依赖外层
- Domain 层不依赖任何层
- Infrastructure 层实现 Domain 层的接口

## 核心流程

### 启动流程

```
1. 创建配置（Config）
   ↓
2. 创建 Redis 客户端
   ↓
3. 创建仓储实现（Repository Impl）
   ↓
4. 创建执行器注册表（Executor Registry）
   ↓
5. 创建 Leader 选举（Leader Election）
   ↓
6. 创建负载均衡器（Load Balancer）
   ↓
7. 创建应用服务（Schedule Service, Worker Service）
   ↓
8. 创建调度器（Scheduler）
   ↓
9. 启动 Worker 服务（心跳 + 任务执行）
   ↓
10. 启动调度服务（Leader 选举 + 任务调度）
```

### 任务调度流程

```
Leader 扫描待执行任务
   ↓
查询健康的 Worker（通过 WorkerRepository）
   ↓
负载均衡选择 Worker（通过 LoadBalancer）
   ↓
任务推入 Worker 队列（通过 TaskRepository）
   ↓
Worker 从队列拉取任务
   ↓
获取执行器（通过 ExecutorRegistry）
   ↓
执行任务（通过 Executor）
   ↓
保存结果（通过 TaskRepository）
   ↓
更新任务状态
```

## 扩展点

### 1. 添加新的执行器

```go
// 1. 实现 Executor 接口
type MyExecutor struct{}

func (e *MyExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
    // 实现逻辑
}

func (e *MyExecutor) Type() string { return "my_type" }
func (e *MyExecutor) Protocol() string { return "my_protocol" }

// 2. 注册执行器
ds.RegisterExecutor(&MyExecutor{})
```

### 2. 添加新的负载均衡策略

```go
// 1. 实现 LoadBalancer 接口
type MyLoadBalancer struct{}

func (lb *MyLoadBalancer) Select(workers []*model.Worker, taskID string) (*model.Worker, error) {
    // 实现逻辑
}

// 2. 在 LoadBalancerFactory 中添加
```

### 3. 替换存储实现

```go
// 1. 实现 Repository 接口
type MySQLTaskRepository struct{}

func (r *MySQLTaskRepository) Save(ctx context.Context, task *model.Task) error {
    // 实现逻辑
}
// ... 实现其他方法

// 2. 在 Scheduler 创建时使用新实现
```

## 测试策略

### 单元测试
- Domain 层：测试业务逻辑（已提供 task_test.go, worker_test.go）
- Service 层：使用 mock 仓储测试

### 集成测试
- 使用真实 Redis 测试仓储实现
- 测试完整的任务调度流程

### 端到端测试
- 启动多个实例测试 Leader 选举
- 测试故障恢复场景

## 开发建议

1. **遵循 DDD 原则**：
   - 业务逻辑放在 Domain 层
   - 技术实现放在 Infrastructure 层
   - 保持层次清晰

2. **接口优先**：
   - 先定义接口，再实现
   - 便于测试和替换实现

3. **单一职责**：
   - 每个类/函数只做一件事
   - 保持代码简洁

4. **依赖注入**：
   - 通过构造函数注入依赖
   - 避免全局变量

5. **错误处理**：
   - 使用 error 返回值
   - 记录详细的错误日志
