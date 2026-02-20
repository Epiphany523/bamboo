## distribute schedule
通用的分布式调度框架

### 目标
一个高可用的轻量级分布式调度服务，专注于任务调度功能，支持3-5个Pod的部署规模。

### 核心特性
- 高可用性：多个实例同时运行，避免单点故障
- 分布式协调：通过Redis分布式锁实现调度协调
- 简单轻量：最小化依赖，易于部署和维护
- 容错处理：节点故障自动恢复
- 弹性扩展：支持动态增减 Worker 节点

---

## 架构设计

### 整体架构
```
┌─────────────────────────────────────────────────────────┐
│                        Redis                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Leader Lock  │  │ Worker Pool  │  │  Task Queue  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
           ↑                  ↑                  ↑
           │                  │                  │
    ┌──────┴──────────────────┴──────────────────┴──────┐
    │                                                     │
┌───▼────┐        ┌──────────┐        ┌──────────┐      │
│Schedule│        │ Schedule │        │ Schedule │      │
│(Leader)│        │(Follower)│        │(Follower)│      │
└───┬────┘        └──────────┘        └──────────┘      │
    │                                                     │
    │ 分发任务                                            │
    │                                                     │
┌───▼────┐        ┌──────────┐        ┌──────────┐      │
│ Worker │        │  Worker  │        │  Worker  │      │
└───┬────┘        └────┬─────┘        └────┬─────┘      │
    │                  │                   │             │
┌───▼────┐        ┌────▼─────┐        ┌────▼─────┐      │
│Executor│        │ Executor │        │ Executor │      │
└────────┘        └──────────┘        └──────────┘      │
```

### 核心组件说明

#### 1. Schedule（调度器）
负责任务的调度和分发，每个 Pod 都包含 Schedule 组件。

**职责：**
- Leader 选举：通过 Redis 分布式锁实现
- 任务扫描：Leader 定期扫描待执行任务
- 任务分发：将任务分配给可用的 Worker
- 健康检查：监控 Worker 的健康状态

**Leader 选举机制：**
```
1. 所有 Schedule 实例启动时尝试获取 Redis 锁（key: schedule:leader, ttl: 10s）
2. 获取成功的成为 Leader，定期续约（每 3s 续约一次）
3. Leader 失效后，其他实例自动竞争成为新 Leader
4. 重选举不影响正在执行的任务
```

**Leader 如何感知 Worker：**
- Worker 通过心跳机制注册到 Redis（key: `worker:pool:{worker_id}`）
- Leader 定期从 Redis 读取 Worker 列表
- Worker 心跳超时（30s）自动从池中移除

#### 2. Worker（工作节点）
负责接收和执行任务，每个 Pod 都包含 Worker 组件。

**职责：**
- 服务注册：启动时注册到 Redis Worker Pool
- 心跳维护：定期发送心跳（每 10s）
- 任务执行：接收 Leader 分发的任务并执行
- 状态上报：执行结果回写到 Redis

**Worker 注册方案：**
```
方案：基于 Redis 的服务发现
1. Worker 启动时生成唯一 ID（worker_id = pod_name + uuid）
2. 注册信息写入 Redis：
   key: worker:pool:{worker_id}
   value: {
     "id": "worker_id",
     "address": "pod_ip:port",
     "status": "idle/busy",
     "last_heartbeat": timestamp,
     "capacity": 10  // 并发任务数
   }
   ttl: 30s
3. 定期续约（每 10s 更新一次）
4. Leader 从 Redis 读取所有 worker:pool:* 获取 Worker 列表
```

**优势：**
- 无需知道 Leader 地址，只需连接 Redis
- Leader 切换对 Worker 透明
- 自动故障检测（心跳超时自动移除）

#### 3. Executor（执行器）
业务扩展接口，根据任务类型执行具体逻辑。

**接口定义：**
```go
type Executor interface {
    // 执行任务
    Execute(ctx context.Context, task *Task) (*Result, error)
    
    // 任务类型
    Type() string
    
    // 支持的协议（http/rpc/mq）
    Protocol() string
}
```

**内置实现：**
- HTTPExecutor：通过 HTTP 调用外部服务
- RPCExecutor：通过 gRPC 调用外部服务
- LocalExecutor：本地函数执行

---

## 领域模型

### TaskConfig（任务配置）
```go
type TaskConfig struct {
    ID          string        // 任务配置ID
    Name        string        // 任务名称
    Type        string        // 任务类型（http/rpc/local）
    CronExpr    string        // Cron 表达式
    Timeout     time.Duration // 超时时间
    RetryPolicy RetryPolicy   // 重试策略
    Executor    string        // 执行器类型
    Payload     interface{}   // 任务参数
    Enabled     bool          // 是否启用
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type RetryPolicy struct {
    MaxRetries  int           // 最大重试次数
    RetryDelay  time.Duration // 重试间隔
    BackoffRate float64       // 退避倍率（指数退避）
}
```

### Task（任务实例）
```go
type Task struct {
    ID            string        // 任务实例ID
    ConfigID      string        // 关联的配置ID
    Status        TaskStatus    // 任务状态
    WorkerID      string        // 执行的 Worker ID
    ScheduledTime time.Time     // 计划执行时间
    StartTime     time.Time     // 实际开始时间
    EndTime       time.Time     // 结束时间
    RetryCount    int           // 已重试次数
    Result        *TaskResult   // 执行结果
    Error         string        // 错误信息
}

type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"   // 待执行
    TaskRunning   TaskStatus = "running"   // 执行中
    TaskSuccess   TaskStatus = "success"   // 成功
    TaskFailed    TaskStatus = "failed"    // 失败
    TaskRetrying  TaskStatus = "retrying"  // 重试中
    TaskCancelled TaskStatus = "cancelled" // 已取消
)

type TaskResult struct {
    Code    int         // 结果码
    Message string      // 结果消息
    Data    interface{} // 结果数据
}
```

---

## 核心流程

### 1. 任务调度流程
```
1. Leader 定期扫描（每 5s）待执行任务
2. 从 Redis 获取可用 Worker 列表
3. 根据负载均衡策略选择 Worker
4. 将任务信息写入 Redis 队列：task:queue:{worker_id}
5. Worker 从队列拉取任务并执行
6. 执行结果写回 Redis：task:result:{task_id}
7. Leader 定期收集结果并更新任务状态
```

### 2. 重试机制
```
1. 任务执行失败后，检查重试策略
2. 如果未达到最大重试次数：
   - 计算下次重试时间（指数退避）
   - 更新任务状态为 retrying
   - 重新加入调度队列
3. 达到最大重试次数后标记为 failed
```

### 3. 故障恢复
```
Leader 故障：
- 其他实例自动竞争成为新 Leader
- 新 Leader 接管任务调度
- 正在执行的任务不受影响

Worker 故障：
- 心跳超时自动从池中移除
- Leader 检测到任务超时，重新调度到其他 Worker
- 任务状态从 running 恢复为 pending
```

---

## Redis 数据结构

### 1. Leader 锁
```
key: schedule:leader
value: {leader_id}
ttl: 10s
```

### 2. Worker 池
```
key: worker:pool:{worker_id}
value: {
  "id": "worker_id",
  "address": "ip:port",
  "status": "idle",
  "last_heartbeat": 1234567890,
  "capacity": 10,
  "running_tasks": 3
}
ttl: 30s
```

### 3. 任务队列
```
key: task:queue:{worker_id}
type: list
value: [task_id1, task_id2, ...]
```

### 4. 任务详情
```
key: task:detail:{task_id}
value: {Task JSON}
ttl: 7 days
```

### 5. 任务结果
```
key: task:result:{task_id}
value: {TaskResult JSON}
ttl: 7 days
```

---

## 负载均衡策略

### 1. 最少任务优先（默认）
选择当前运行任务数最少的 Worker

### 2. 轮询
按顺序依次分配给 Worker

### 3. 一致性哈希
根据任务 ID 哈希到固定 Worker（适合有状态任务）

---

## 监控指标

### Schedule 指标
- Leader 选举次数
- 任务调度延迟
- 任务分发成功率
- Worker 在线数量

### Worker 指标
- 任务执行成功率
- 任务执行耗时（P50/P95/P99）
- 当前并发任务数
- 心跳延迟

### Task 指标
- 任务总数（按状态分类）
- 任务重试率
- 任务失败率
- 任务执行时长分布

---

## 配置示例

```yaml
schedule:
  leader_lock_ttl: 10s        # Leader 锁 TTL
  leader_renew_interval: 3s   # Leader 续约间隔
  scan_interval: 5s           # 任务扫描间隔
  
worker:
  heartbeat_interval: 10s     # 心跳间隔
  heartbeat_timeout: 30s      # 心跳超时
  max_concurrent_tasks: 10    # 最大并发任务数
  
task:
  default_timeout: 5m         # 默认超时时间
  max_retry: 3                # 默认最大重试次数
  retry_delay: 10s            # 默认重试间隔
  backoff_rate: 2.0           # 退避倍率

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10
```

---

## 部署建议

### 最小部署（3 个 Pod）
- 每个 Pod 同时运行 Schedule + Worker
- 保证至少 2 个 Pod 存活即可正常工作
- 适合小规模任务场景

### 推荐部署（5 个 Pod）
- 每个 Pod 同时运行 Schedule + Worker
- 提供更好的容错能力和负载分散
- 适合中等规模任务场景

### 扩展部署
- 可独立扩展 Worker 数量（只运行 Worker 组件）
- Schedule 保持 3-5 个实例即可
- 适合大规模任务场景

---

## 后续优化方向

1. 任务优先级支持
2. 任务依赖关系（DAG）
3. 分片任务支持（大任务拆分）
4. 动态调整 Worker 容量
5. 任务执行日志收集
6. Web 管理界面
7. 任务执行统计和报表

---

## 代码实现

代码已按照 DDD 模式实现，目录结构如下：

```
pkg/distributeschedule/
├── domain/                    # 领域层
│   ├── model/                # 领域模型
│   │   ├── task.go          # 任务实体
│   │   ├── task_config.go   # 任务配置实体
│   │   └── worker.go        # Worker 实体
│   ├── repository/           # 仓储接口
│   │   ├── task_repository.go
│   │   ├── task_config_repository.go
│   │   └── worker_repository.go
│   └── service/              # 领域服务
│       ├── executor.go       # 执行器接口
│       └── load_balancer.go  # 负载均衡器
├── application/              # 应用层
│   ├── schedule_service.go   # 调度服务
│   └── worker_service.go     # Worker 服务
├── infrastructure/           # 基础设施层
│   ├── redis/               # Redis 实现
│   │   ├── redis_client.go
│   │   ├── leader_election.go
│   │   ├── task_repository_impl.go
│   │   └── worker_repository_impl.go
│   └── executor/            # 执行器实现
│       ├── executor_registry_impl.go
│       ├── http_executor.go
│       └── local_executor.go
├── interfaces/              # 接口层
│   └── scheduler.go         # 对外接口
├── config/                  # 配置
│   └── config.go
├── example/                 # 示例代码
│   └── main.go
├── distributeschedule.go    # 门面
├── ARCHITECTURE.md          # 架构文档
├── USAGE.md                 # 使用指南
└── readme.md               # 本文件
```

### 快速开始

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
	cfg.Redis.Addr = "localhost:6379"
	
	// 创建并启动调度框架
	ds, err := distributeschedule.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()
	
	ctx := context.Background()
	if err := ds.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
```

### 文档

- [架构文档](./ARCHITECTURE.md) - 详细的架构设计和技术选型
- [使用指南](./USAGE.md) - 完整的使用说明和最佳实践

