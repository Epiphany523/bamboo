# 架构文档

## DDD 分层架构

本项目采用领域驱动设计（DDD）的分层架构：

```
pkg/distributeschedule/
├── domain/              # 领域层
│   ├── model/          # 领域模型（聚合根、实体、值对象）
│   ├── repository/     # 仓储接口
│   └── service/        # 领域服务
├── application/        # 应用层
│   ├── schedule_service.go  # 调度用例
│   └── worker_service.go    # Worker 用例
├── infrastructure/     # 基础设施层
│   ├── redis/         # Redis 实现
│   └── executor/      # 执行器实现
├── interfaces/        # 接口层
│   └── scheduler.go   # 对外接口
└── config/           # 配置
```

## 核心概念

### 1. 聚合根（Aggregate Root）

#### Worker
- 工作节点的核心实体
- 管理自身状态和任务容量
- 提供任务接受和完成的业务逻辑

#### Task
- 任务实例的核心实体
- 管理任务生命周期
- 提供状态转换的业务逻辑

#### TaskConfig
- 任务配置的核心实体
- 定义任务的执行规则
- 创建任务实例

### 2. 值对象（Value Object）

#### RetryPolicy
- 重试策略的不可变对象
- 封装重试逻辑

#### TaskResult
- 任务执行结果的不可变对象

### 3. 仓储（Repository）

仓储模式将数据访问逻辑与业务逻辑分离：

- `TaskRepository`: 任务数据访问
- `WorkerRepository`: Worker 数据访问
- `TaskConfigRepository`: 任务配置数据访问

### 4. 领域服务（Domain Service）

#### Executor
- 任务执行的领域服务
- 支持多种执行协议（HTTP、RPC、Local）

#### LoadBalancer
- 负载均衡的领域服务
- 支持多种策略（最少任务、轮询、一致性哈希）

## 技术选型

### 存储层
- Redis: 分布式锁、服务发现、任务队列

### 优势
- 轻量级：最小化依赖
- 高性能：基于内存的 Redis
- 易扩展：清晰的分层架构

## 设计模式

### 1. 仓储模式（Repository Pattern）
- 抽象数据访问层
- 便于测试和替换实现

### 2. 策略模式（Strategy Pattern）
- 负载均衡策略
- 执行器策略

### 3. 工厂模式（Factory Pattern）
- 负载均衡器工厂
- 执行器注册表

### 4. 门面模式（Facade Pattern）
- DistributeSchedule 提供统一接口

## 数据流

### 任务调度流程

```
1. Leader 扫描待执行任务
   ↓
2. 从 Redis 获取健康的 Worker 列表
   ↓
3. 负载均衡器选择 Worker
   ↓
4. 任务推入 Worker 队列
   ↓
5. Worker 从队列拉取任务
   ↓
6. 执行器执行任务
   ↓
7. 结果写回 Redis
   ↓
8. Leader 更新任务状态
```

### Leader 选举流程

```
1. 所有实例尝试获取 Redis 锁
   ↓
2. 获取成功的成为 Leader
   ↓
3. Leader 定期续约（3s）
   ↓
4. 续约失败则释放 Leader 身份
   ↓
5. 其他实例重新竞争
```

## 扩展点

### 1. 自定义执行器
实现 `Executor` 接口，注册到框架

### 2. 自定义负载均衡策略
实现 `LoadBalancer` 接口

### 3. 自定义存储
实现 `Repository` 接口

## 性能优化

### 1. 批量操作
- 批量获取任务
- 批量更新状态

### 2. 连接池
- Redis 连接池
- HTTP 客户端复用

### 3. 异步处理
- 心跳异步发送
- 结果异步写入

## 安全考虑

### 1. 任务隔离
- 每个 Worker 独立队列
- 任务执行超时控制

### 2. 故障隔离
- Leader 故障不影响 Worker
- Worker 故障不影响其他 Worker

### 3. 数据一致性
- 使用 Redis 事务
- 乐观锁控制并发
