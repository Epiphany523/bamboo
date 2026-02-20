# 可扩展性设计

## 设计原则

1. **面向接口编程**: 核心组件都定义为接口，易于替换实现
2. **插件化架构**: 支持动态注册和加载插件
3. **配置驱动**: 通过配置控制行为，无需修改代码
4. **事件驱动**: 使用事件机制解耦组件
5. **依赖注入**: 通过依赖注入管理组件生命周期

---

## 1. 自定义 Executor（执行器）

### 接口定义

```go
type Executor interface {
    // Execute 执行任务
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
    
    // Type 执行器类型标识
    Type() string
    
    // SupportedTaskTypes 支持的任务类型列表
    SupportedTaskTypes() []string
    
    // Initialize 初始化执行器
    Initialize(config map[string]interface{}) error
    
    // Close 关闭执行器，释放资源
    Close() error
}
```

### 示例：Kafka Executor

```go
type KafkaExecutor struct {
    producer *kafka.Producer
    config   *KafkaConfig
}

func NewKafkaExecutor(config *KafkaConfig) *KafkaExecutor {
    return &KafkaExecutor{
        config: config,
    }
}

func (e *KafkaExecutor) Initialize(config map[string]interface{}) error {
    // 初始化 Kafka Producer
    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": e.config.Brokers,
    })
    if err != nil {
        return err
    }
    e.producer = producer
    return nil
}

func (e *KafkaExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 解析 payload
    payload := task.Payload.(map[string]interface{})
    topic := payload["topic"].(string)
    message := payload["message"].(string)
    
    // 发送消息到 Kafka
    err := e.producer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &topic,
            Partition: kafka.PartitionAny,
        },
        Value: []byte(message),
    }, nil)
    
    if err != nil {
        return nil, err
    }
    
    return &TaskResult{
        Success: true,
        Data:    map[string]interface{}{"topic": topic},
    }, nil
}

func (e *KafkaExecutor) Type() string {
    return "kafka"
}

func (e *KafkaExecutor) SupportedTaskTypes() []string {
    return []string{"send_kafka_message", "publish_event"}
}

func (e *KafkaExecutor) Close() error {
    if e.producer != nil {
        e.producer.Close()
    }
    return nil
}

// 注册到 Worker
func main() {
    worker := NewWorker(config)
    
    kafkaExecutor := NewKafkaExecutor(&KafkaConfig{
        Brokers: "localhost:9092",
    })
    
    worker.RegisterExecutor(kafkaExecutor)
    worker.Start(context.Background())
}
```

### 示例：数据库 Executor

```go
type DatabaseExecutor struct {
    db *sql.DB
}

func (e *DatabaseExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    payload := task.Payload.(map[string]interface{})
    query := payload["query"].(string)
    params := payload["params"].([]interface{})
    
    result, err := e.db.ExecContext(ctx, query, params...)
    if err != nil {
        return nil, err
    }
    
    rowsAffected, _ := result.RowsAffected()
    
    return &TaskResult{
        Success: true,
        Data: map[string]interface{}{
            "rows_affected": rowsAffected,
        },
    }, nil
}

func (e *DatabaseExecutor) Type() string {
    return "database"
}

func (e *DatabaseExecutor) SupportedTaskTypes() []string {
    return []string{"execute_sql", "batch_insert"}
}
```

---

## 2. 自定义负载均衡策略

### 接口定义

```go
type LoadBalancer interface {
    // Select 选择一个 Worker
    Select(workers []*Worker, task *Task) (*Worker, error)
    
    // Name 策略名称
    Name() string
}
```

### 示例：加权轮询

```go
type WeightedRoundRobinLoadBalancer struct {
    counter uint64
}

func (lb *WeightedRoundRobinLoadBalancer) Select(workers []*Worker, task *Task) (*Worker, error) {
    if len(workers) == 0 {
        return nil, errors.New("no available workers")
    }
    
    // 计算总权重
    totalWeight := 0
    for _, w := range workers {
        totalWeight += w.Weight
    }
    
    // 加权轮询
    offset := atomic.AddUint64(&lb.counter, 1) % uint64(totalWeight)
    
    for _, w := range workers {
        if offset < uint64(w.Weight) {
            return w, nil
        }
        offset -= uint64(w.Weight)
    }
    
    return workers[0], nil
}

func (lb *WeightedRoundRobinLoadBalancer) Name() string {
    return "weighted_round_robin"
}
```

### 示例：基于任务类型的亲和性

```go
type AffinityLoadBalancer struct {
    // 记录任务类型到 Worker 的映射
    affinityMap sync.Map
}

func (lb *AffinityLoadBalancer) Select(workers []*Worker, task *Task) (*Worker, error) {
    // 检查是否有亲和性记录
    if workerID, ok := lb.affinityMap.Load(task.TaskType); ok {
        for _, w := range workers {
            if w.ID == workerID.(string) {
                return w, nil
            }
        }
    }
    
    // 没有亲和性记录，选择负载最低的
    var selected *Worker
    minLoad := int(^uint(0) >> 1)
    
    for _, w := range workers {
        if w.CurrentLoad < minLoad {
            selected = w
            minLoad = w.CurrentLoad
        }
    }
    
    // 记录亲和性
    if selected != nil {
        lb.affinityMap.Store(task.TaskType, selected.ID)
    }
    
    return selected, nil
}
```

---

## 3. 自定义存储后端

### 接口定义

```go
type TaskRepository interface {
    Create(ctx context.Context, task *Task) error
    GetByID(ctx context.Context, taskID string) (*Task, error)
    Update(ctx context.Context, task *Task) error
    Delete(ctx context.Context, taskID string) error
    List(ctx context.Context, filter *TaskFilter) ([]*Task, error)
}

type TaskLogRepository interface {
    Create(ctx context.Context, log *TaskLog) error
    GetByTaskID(ctx context.Context, taskID string) ([]*TaskLog, error)
}
```

### 示例：MongoDB 实现

```go
type MongoTaskRepository struct {
    client     *mongo.Client
    collection *mongo.Collection
}

func NewMongoTaskRepository(uri, database string) (*MongoTaskRepository, error) {
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }
    
    collection := client.Database(database).Collection("tasks")
    
    return &MongoTaskRepository{
        client:     client,
        collection: collection,
    }, nil
}

func (r *MongoTaskRepository) Create(ctx context.Context, task *Task) error {
    _, err := r.collection.InsertOne(ctx, task)
    return err
}

func (r *MongoTaskRepository) GetByID(ctx context.Context, taskID string) (*Task, error) {
    var task Task
    err := r.collection.FindOne(ctx, bson.M{"task_id": taskID}).Decode(&task)
    if err != nil {
        return nil, err
    }
    return &task, nil
}

func (r *MongoTaskRepository) Update(ctx context.Context, task *Task) error {
    _, err := r.collection.UpdateOne(
        ctx,
        bson.M{"task_id": task.TaskID},
        bson.M{"$set": task},
    )
    return err
}
```

---

## 4. 自定义监控指标

### 接口定义

```go
type MetricsCollector interface {
    // 任务指标
    RecordTaskCreated(taskType string)
    RecordTaskCompleted(taskType string, duration time.Duration, status string)
    RecordTaskFailed(taskType string, reason string)
    
    // Worker 指标
    RecordWorkerRegistered(workerID string)
    RecordWorkerOffline(workerID string)
    RecordWorkerLoad(workerID string, load int)
    
    // 队列指标
    RecordQueueLength(queueName string, length int)
    
    // 调度器指标
    RecordSchedulerElection(schedulerID string, isLeader bool)
}
```

### 示例：Prometheus 实现

```go
type PrometheusCollector struct {
    taskCreated   *prometheus.CounterVec
    taskCompleted *prometheus.HistogramVec
    taskFailed    *prometheus.CounterVec
    workerLoad    *prometheus.GaugeVec
    queueLength   *prometheus.GaugeVec
}

func NewPrometheusCollector() *PrometheusCollector {
    collector := &PrometheusCollector{
        taskCreated: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "task_created_total",
                Help: "Total number of tasks created",
            },
            []string{"task_type"},
        ),
        taskCompleted: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "task_duration_seconds",
                Help:    "Task execution duration",
                Buckets: prometheus.DefBuckets,
            },
            []string{"task_type", "status"},
        ),
        // ... 其他指标
    }
    
    // 注册到 Prometheus
    prometheus.MustRegister(collector.taskCreated)
    prometheus.MustRegister(collector.taskCompleted)
    
    return collector
}

func (c *PrometheusCollector) RecordTaskCreated(taskType string) {
    c.taskCreated.WithLabelValues(taskType).Inc()
}

func (c *PrometheusCollector) RecordTaskCompleted(taskType string, duration time.Duration, status string) {
    c.taskCompleted.WithLabelValues(taskType, status).Observe(duration.Seconds())
}
```

---

## 5. 事件系统

### 接口定义

```go
type EventBus interface {
    // Publish 发布事件
    Publish(ctx context.Context, event *Event) error
    
    // Subscribe 订阅事件
    Subscribe(ctx context.Context, eventType string, handler EventHandler) error
    
    // Unsubscribe 取消订阅
    Unsubscribe(ctx context.Context, eventType string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event *Event) error

type Event struct {
    Type      string
    TaskID    string
    Data      interface{}
    Timestamp time.Time
}
```

### 示例：基于 Redis Pub/Sub

```go
type RedisEventBus struct {
    redis    *redis.Client
    handlers map[string][]EventHandler
    mu       sync.RWMutex
}

func (bus *RedisEventBus) Publish(ctx context.Context, event *Event) error {
    data, _ := json.Marshal(event)
    return bus.redis.Publish(ctx, "task:events", data).Err()
}

func (bus *RedisEventBus) Subscribe(ctx context.Context, eventType string, handler EventHandler) error {
    bus.mu.Lock()
    defer bus.mu.Unlock()
    
    if bus.handlers == nil {
        bus.handlers = make(map[string][]EventHandler)
    }
    
    bus.handlers[eventType] = append(bus.handlers[eventType], handler)
    return nil
}

// 启动事件监听
func (bus *RedisEventBus) Start(ctx context.Context) {
    pubsub := bus.redis.Subscribe(ctx, "task:events")
    defer pubsub.Close()
    
    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-pubsub.Channel():
            var event Event
            json.Unmarshal([]byte(msg.Payload), &event)
            
            bus.mu.RLock()
            handlers := bus.handlers[event.Type]
            bus.mu.RUnlock()
            
            for _, handler := range handlers {
                go handler(ctx, &event)
            }
        }
    }
}
```

### 使用示例

```go
// 订阅任务完成事件
eventBus.Subscribe(ctx, "task_completed", func(ctx context.Context, event *Event) error {
    log.Printf("Task %s completed", event.TaskID)
    
    // 发送通知
    notificationService.Send(event.TaskID)
    
    // 更新统计
    statsService.UpdateStats(event.TaskID)
    
    return nil
})

// 发布事件
eventBus.Publish(ctx, &Event{
    Type:      "task_completed",
    TaskID:    taskID,
    Data:      result,
    Timestamp: time.Now(),
})
```

---

## 6. 插件系统

### 插件接口

```go
type Plugin interface {
    // Name 插件名称
    Name() string
    
    // Version 插件版本
    Version() string
    
    // Initialize 初始化插件
    Initialize(config map[string]interface{}) error
    
    // Start 启动插件
    Start(ctx context.Context) error
    
    // Stop 停止插件
    Stop(ctx context.Context) error
}
```

### 插件管理器

```go
type PluginManager struct {
    plugins map[string]Plugin
    mu      sync.RWMutex
}

func (pm *PluginManager) Register(plugin Plugin) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    if _, exists := pm.plugins[plugin.Name()]; exists {
        return fmt.Errorf("plugin %s already registered", plugin.Name())
    }
    
    pm.plugins[plugin.Name()] = plugin
    return nil
}

func (pm *PluginManager) Get(name string) (Plugin, error) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    
    plugin, exists := pm.plugins[name]
    if !exists {
        return nil, fmt.Errorf("plugin %s not found", name)
    }
    
    return plugin, nil
}

func (pm *PluginManager) StartAll(ctx context.Context) error {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    
    for _, plugin := range pm.plugins {
        if err := plugin.Start(ctx); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 示例：告警插件

```go
type AlertPlugin struct {
    config      *AlertConfig
    alertClient *AlertClient
}

func (p *AlertPlugin) Name() string {
    return "alert"
}

func (p *AlertPlugin) Version() string {
    return "1.0.0"
}

func (p *AlertPlugin) Initialize(config map[string]interface{}) error {
    // 解析配置
    p.config = parseAlertConfig(config)
    
    // 初始化告警客户端
    p.alertClient = NewAlertClient(p.config)
    
    return nil
}

func (p *AlertPlugin) Start(ctx context.Context) error {
    // 订阅任务失败事件
    eventBus.Subscribe(ctx, "task_failed", p.handleTaskFailed)
    
    // 订阅 Worker 离线事件
    eventBus.Subscribe(ctx, "worker_offline", p.handleWorkerOffline)
    
    return nil
}

func (p *AlertPlugin) handleTaskFailed(ctx context.Context, event *Event) error {
    // 发送告警
    return p.alertClient.Send(&Alert{
        Level:   "ERROR",
        Title:   "Task Failed",
        Message: fmt.Sprintf("Task %s failed", event.TaskID),
    })
}
```

---

## 7. 配置驱动

### 配置文件示例

```yaml
# config.yaml
app:
  name: "async-task-manager"
  version: "1.0.0"

scheduler:
  enabled: true
  load_balance_strategy: "least_task"
  scan_interval: 100ms
  
worker:
  enabled: true
  capacity: 10
  supported_types:
    - "send_email"
    - "generate_report"

executors:
  - type: "rpc"
    config:
      timeout: 30s
      max_retries: 3
  - type: "http"
    config:
      timeout: 60s
      max_retries: 3
  - type: "kafka"
    config:
      brokers: "localhost:9092"

plugins:
  - name: "alert"
    enabled: true
    config:
      webhook_url: "https://alert.example.com/webhook"
  - name: "metrics"
    enabled: true
    config:
      prometheus_port: 9090

storage:
  type: "mysql"  # mysql, mongodb, postgresql
  config:
    host: "localhost"
    port: 3306
    database: "task_manager"
```

### 动态加载配置

```go
type ConfigLoader interface {
    Load(path string) (*Config, error)
    Watch(path string, callback func(*Config)) error
}

// 热更新配置
func (app *Application) watchConfig(ctx context.Context) {
    configLoader.Watch("config.yaml", func(newConfig *Config) {
        // 更新负载均衡策略
        if newConfig.Scheduler.LoadBalanceStrategy != app.config.Scheduler.LoadBalanceStrategy {
            app.scheduler.SetLoadBalancer(
                LoadBalancerFactory(newConfig.Scheduler.LoadBalanceStrategy),
            )
        }
        
        // 更新 Worker 容量
        if newConfig.Worker.Capacity != app.config.Worker.Capacity {
            app.worker.SetCapacity(newConfig.Worker.Capacity)
        }
        
        app.config = newConfig
    })
}
```

---

## 8. 扩展点总结

| 扩展点 | 接口 | 用途 |
|--------|------|------|
| 执行器 | `Executor` | 支持不同的任务执行方式 |
| 负载均衡 | `LoadBalancer` | 自定义 Worker 选择策略 |
| 存储后端 | `TaskRepository` | 支持不同的数据库 |
| 监控指标 | `MetricsCollector` | 集成不同的监控系统 |
| 事件总线 | `EventBus` | 事件驱动的扩展 |
| 插件系统 | `Plugin` | 动态加载功能模块 |
| 配置加载 | `ConfigLoader` | 支持不同的配置源 |
