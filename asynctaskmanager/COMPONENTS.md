# 组件职责与依赖关系

## 架构总览

```
┌─────────────────────────────────────────────────────────────┐
│                        用户/业务方                            │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP/gRPC
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway                             │
│                   (TaskService API)                          │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ↓            ↓            ↓
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Scheduler│  │  Worker  │  │  Worker  │
│ (Leader) │  │    #1    │  │    #2    │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │
     │             ↓             ↓
     │      ┌──────────┐  ┌──────────┐
     │      │Executor  │  │Executor  │
     │      │Registry  │  │Registry  │
     │      └────┬─────┘  └────┬─────┘
     │           │             │
     │           ↓             ↓
     │      ┌──────────────────────┐
     │      │   RPC/HTTP Executor  │
     │      │   (调用业务方接口)    │
     │      └──────────────────────┘
     │
     ↓
┌─────────────────────────────────────────┐
│              Redis                       │
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │ 队列     │  │ 分布式锁 │  │ 服务   ││
│  │          │  │          │  │ 发现   ││
│  └──────────┘  └──────────┘  └────────┘│
└─────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────┐
│              MySQL                       │
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │ task     │  │task_logs │  │task_   ││
│  │          │  │          │  │config  ││
│  └──────────┘  └──────────┘  └────────┘│
└─────────────────────────────────────────┘
```

---

## 1. Scheduler（调度器）

### 职责

**核心职责**:
- Leader 选举和维护
- 任务队列消费
- 任务分配给 Worker
- 超时任务检测和恢复
- 失败任务检测和重试
- 任务限流和流控

**详细职责**:

1. **Leader 选举**
   - 基于 Redis 分布式锁实现
   - 多个 Scheduler 实例竞争成为 Leader
   - 只有 Leader 执行任务调度
   - 定期续约，失败则重新选举

2. **队列管理**
   - 优先消费高优先级队列 (`queue:high`)
   - 再消费普通优先级队列 (`queue:normal`)
   - 批量拉取任务提高效率

3. **任务分配**
   - 获取可用 Worker 列表
   - 根据负载均衡策略选择 Worker
   - 将任务推送到 Worker 队列
   - 记录分配信息

4. **健康检查**
   - 定期检查 Worker 心跳
   - 移除不健康的 Worker
   - 重新分配失联 Worker 的任务

5. **超时处理**
   - 扫描执行超时的任务
   - 标记为 TIMEOUT 状态
   - 触发重试机制

6. **失败重试处理**
   - 扫描执行失败的任务，看是否需要重试
   - 标记为 Retry 状态
   - 根据重试策略触发重试机制

7. **限流控制**
   - 基于任务类型的并发限制
   - 基于 Worker 的负载限制
   - 全局任务速率限制

### 依赖关系

```
Scheduler 依赖：
├── Redis
│   ├── 分布式锁（Leader 选举）
│   ├── 任务队列（消费任务）
│   └── Worker 注册表（服务发现）
├── MySQL
│   ├── task 表（查询任务详情）
│   ├── task_logs 表（记录日志）
│   └── task_config 表（获取配置）
└── LoadBalancer（负载均衡策略）
```

### 接口定义

```go
type Scheduler interface {
    // Start 启动调度器
    Start(ctx context.Context) error
    
    // Stop 停止调度器
    Stop(ctx context.Context) error
    
    // IsLeader 是否是 Leader
    IsLeader() bool
    
    // GetMetrics 获取调度指标
    GetMetrics() *SchedulerMetrics
}

type SchedulerMetrics struct {
    IsLeader          bool
    TasksScheduled    int64
    TasksFailed       int64
    AvailableWorkers  int
    QueueLength       int
}
```

### 配置示例

```yaml
scheduler:
  leader_lock_key: "scheduler:leader"
  leader_lock_ttl: 10s
  leader_renew_interval: 3s
  scan_interval: 100ms
  batch_size: 10
  timeout_check_interval: 30s
  load_balance_strategy: "least_task"
```

---

## 2. Worker（工作节点）

### 职责

**核心职责**:
- 服务注册和心跳维护
- 任务队列消费
- 任务执行协调
- 取消信号检测
- 负载管理

**详细职责**:

1. **服务注册**
   - 启动时注册到 Redis
   - 上报支持的任务类型
   - 上报最大并发能力

2. **心跳维护**
   - 定期发送心跳（默认 10s）
   - 更新负载信息
   - 心跳失败自动重连

3. **任务消费**
   - 从专属队列拉取任务
   - 控制并发数量
   - 优雅关闭处理

4. **任务执行**
   - 获取任务详情
   - 选择合适的 Executor
   - 设置超时控制
   - 处理执行结果

5. **取消处理**
   - 定期检查取消标记
   - 中断正在执行的任务
   - 更新任务状态

6. **负载管理**
   - 跟踪当前执行任务数
   - 拒绝超载任务
   - 上报负载信息

### 依赖关系

```
Worker 依赖：
├── Redis
│   ├── Worker 注册表（服务注册）
│   ├── 任务队列（消费任务）
│   └── 取消标记（检测取消）
├── MySQL
│   ├── task 表（查询和更新任务）
│   └── task_logs 表（记录日志）
├── ExecutorRegistry（执行器注册表）
└── Executor（具体执行器）
```

### 接口定义

```go
type Worker interface {
    // Start 启动 Worker
    Start(ctx context.Context) error
    
    // Stop 停止 Worker
    Stop(ctx context.Context) error
    
    // GetID 获取 Worker ID
    GetID() string
    
    // GetLoad 获取当前负载
    GetLoad() int
    
    // GetCapacity 获取最大容量
    GetCapacity() int
}

type WorkerConfig struct {
    ID                string
    Name              string
    Capacity          int
    SupportedTypes    []string
    HeartbeatInterval time.Duration
    QueuePollInterval time.Duration
}
```

### 配置示例

```yaml
worker:
  id: "worker-001"
  name: "worker-node-1"
  capacity: 10
  supported_types:
    - "send_email"
    - "generate_report"
    - "process_image"
  heartbeat_interval: 10s
  queue_poll_interval: 100ms
  graceful_shutdown_timeout: 30s
```

---

## 3. Executor（执行器）

### 职责

**核心职责**:
- 执行具体的业务逻辑
- 调用业务方接口
- 处理执行结果
- 错误处理和重试

**详细职责**:

1. **任务执行**
   - 解析任务参数
   - 调用业务方接口（RPC/HTTP）
   - 处理响应结果
   - 返回执行结果

2. **协议适配**
   - RPC 协议（gRPC）
   - HTTP 协议（RESTful）
   - 本地函数调用

3. **错误处理**
   - 捕获执行异常
   - 区分可重试和不可重试错误
   - 返回详细错误信息

4. **超时控制**
   - 遵守任务超时设置
   - 及时中断超时任务
   - 释放资源

### 依赖关系

```
Executor 依赖：
├── task_config（获取执行器配置）
├── RPC Client（gRPC 调用）
├── HTTP Client（HTTP 调用）
└── 业务方服务（实际执行逻辑）
```

### 接口定义

```go
type Executor interface {
    // Execute 执行任务
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
    
    // Type 执行器类型
    Type() string
    
    // SupportedTaskTypes 支持的任务类型
    SupportedTaskTypes() []string
}

type TaskResult struct {
    Success bool
    Data    interface{}
    Error   string
}

// ExecutorRegistry 执行器注册表
type ExecutorRegistry interface {
    // Register 注册执行器
    Register(executor Executor) error
    
    // Get 获取执行器
    Get(taskType string) (Executor, error)
    
    // List 列出所有执行器
    List() []Executor
}
```

### 内置执行器

#### 1. RPC Executor

```go
type RPCExecutor struct {
    client *grpc.ClientConn
}

func (e *RPCExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 1. 获取配置
    config := task.Config.ExecutorConfig
    
    // 2. 构建请求
    req := &pb.ExecuteRequest{
        TaskID:  task.TaskID,
        Payload: task.Payload,
    }
    
    // 3. 调用 RPC
    resp, err := e.client.Execute(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 4. 返回结果
    return &TaskResult{
        Success: resp.Success,
        Data:    resp.Data,
    }, nil
}
```

#### 2. HTTP Executor

```go
type HTTPExecutor struct {
    client *http.Client
}

func (e *HTTPExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 1. 获取配置
    config := task.Config.ExecutorConfig
    url := config["url"].(string)
    
    // 2. 构建请求
    body, _ := json.Marshal(task.Payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    // 3. 发送请求
    resp, err := e.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // 4. 解析响应
    var result TaskResult
    json.NewDecoder(resp.Body).Decode(&result)
    
    return &result, nil
}
```

---

## 组件交互流程

### 1. 任务创建到执行

```
API → TaskService → MySQL → Redis Queue
                              ↓
                         Scheduler (Leader)
                              ↓
                    选择 Worker → Worker Queue
                                      ↓
                                   Worker
                                      ↓
                              ExecutorRegistry
                                      ↓
                                  Executor
                                      ↓
                                业务方服务
```

### 2. Leader 选举流程

```
Scheduler #1          Scheduler #2          Scheduler #3
     ↓                     ↓                     ↓
尝试获取锁            尝试获取锁            尝试获取锁
     ↓                     ↓                     ↓
成功（成为 Leader）    失败（Follower）      失败（Follower）
     ↓                     ↓                     ↓
定期续约              等待 Leader 失效      等待 Leader 失效
     ↓                     ↓                     ↓
执行调度任务          监控状态              监控状态
```

### 3. Worker 注册流程

```
Worker 启动
   ↓
生成 Worker ID
   ↓
注册到 Redis
   - key: worker:{worker_id}
   - value: {id, capacity, types, ...}
   - ttl: 30s
   ↓
启动心跳协程
   - 每 10s 更新一次
   ↓
启动任务消费协程
   - 从 worker:{worker_id}:queue 拉取
```

---

## 扩展点设计

### 1. 自定义 Executor

```go
// 实现 Executor 接口
type CustomExecutor struct {
    // 自定义字段
}

func (e *CustomExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 自定义执行逻辑
    return &TaskResult{Success: true}, nil
}

func (e *CustomExecutor) Type() string {
    return "custom"
}

// 注册到 Worker
worker.RegisterExecutor(&CustomExecutor{})
```

### 2. 自定义负载均衡策略

```go
type LoadBalancer interface {
    Select(workers []*Worker, taskID string) *Worker
}

// 实现自定义策略
type CustomLoadBalancer struct{}

func (lb *CustomLoadBalancer) Select(workers []*Worker, taskID string) *Worker {
    // 自定义选择逻辑
    return workers[0]
}

// 注册到 Scheduler
scheduler.SetLoadBalancer(&CustomLoadBalancer{})
```

### 3. 自定义监控

```go
type MetricsCollector interface {
    RecordTaskCreated(taskType string)
    RecordTaskCompleted(taskType string, duration time.Duration)
    RecordTaskFailed(taskType string, reason string)
}

// 实现自定义监控
type PrometheusCollector struct {
    // Prometheus metrics
}

// 注入到服务
taskService.SetMetricsCollector(&PrometheusCollector{})
```
