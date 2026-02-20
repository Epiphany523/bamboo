# Redis 数据结构设计

## 概述

Redis 在系统中承担以下职责：
1. 任务队列（优先级队列）
2. 分布式锁（Leader 选举）
3. 服务发现（Worker 注册）
4. 取消标记
5. 任务缓存
6. 消息发布订阅

---

## 1. 任务队列

### 高优先级队列

```
key: queue:high
type: list
value: [task_id1, task_id2, ...]
```

**操作**:
- `LPUSH queue:high {task_id}` - 生产者推送任务
- `RPOP queue:high` - Scheduler 消费任务
- `LLEN queue:high` - 查询队列长度

### 普通优先级队列

```
key: queue:normal
type: list
value: [task_id1, task_id2, ...]
```

### Worker 专属队列

```
key: worker:{worker_id}:queue
type: list
value: [task_id1, task_id2, ...]
```

**说明**:
- 每个 Worker 有独立的任务队列
- Scheduler 将任务分配到 Worker 队列
- Worker 从自己的队列消费任务

---

## 2. 分布式锁（Leader 选举）

### Leader 锁

```
key: scheduler:leader
type: string
value: {scheduler_id}
ttl: 10s
```

**操作**:
```redis
# 尝试获取锁
SET scheduler:leader {scheduler_id} NX EX 10

# 续约（只有当前 Leader 可以续约）
SET scheduler:leader {scheduler_id} XX EX 10

# 释放锁
DEL scheduler:leader
```

**Lua 脚本（安全续约）**:
```lua
-- 只有当前持有锁的实例才能续约
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
```

---

## 3. Worker 注册表

### Worker 信息

```
key: worker:{worker_id}
type: hash
fields:
  - id: worker_id
  - name: worker_name
  - address: worker_address
  - capacity: 10
  - current_load: 3
  - supported_types: ["send_email", "generate_report"]
  - last_heartbeat: 1234567890
ttl: 30s
```

**操作**:
```redis
# 注册 Worker
HSET worker:{worker_id} id {worker_id} name {name} capacity 10 ...
EXPIRE worker:{worker_id} 30

# 更新心跳
HSET worker:{worker_id} last_heartbeat {timestamp}
EXPIRE worker:{worker_id} 30

# 更新负载
HINCRBY worker:{worker_id} current_load 1

# 获取所有 Worker
KEYS worker:*

# 获取 Worker 详情
HGETALL worker:{worker_id}
```

### Worker 索引（按任务类型）

```
key: worker:type:{task_type}
type: set
value: {worker_id1, worker_id2, ...}
```

**用途**: 快速查找支持特定任务类型的 Worker

**操作**:
```redis
# 添加 Worker 到类型索引
SADD worker:type:send_email worker-001
SADD worker:type:send_email worker-002

# 查询支持该类型的 Worker
SMEMBERS worker:type:send_email

# 移除 Worker
SREM worker:type:send_email worker-001
```

---

## 4. 取消标记

### 任务取消标记

```
key: task:cancel:{task_id}
type: string
value: "1"
ttl: 3600s (1小时)
```

**操作**:
```redis
# 设置取消标记
SET task:cancel:{task_id} 1 EX 3600

# 检查是否取消
EXISTS task:cancel:{task_id}

# 删除取消标记
DEL task:cancel:{task_id}
```

**Worker 检测逻辑**:
```go
func (w *Worker) isCancelled(ctx context.Context, taskID string) bool {
    key := fmt.Sprintf("task:cancel:%s", taskID)
    exists, _ := w.redis.Exists(ctx, key).Result()
    return exists > 0
}
```

---

## 5. 任务缓存

### 任务详情缓存

```
key: task:cache:{task_id}
type: string (JSON)
value: {task JSON}
ttl: 300s (5分钟)
```

**操作**:
```redis
# 缓存任务
SET task:cache:{task_id} {json} EX 300

# 获取缓存
GET task:cache:{task_id}

# 删除缓存（任务状态变更时）
DEL task:cache:{task_id}
```

### 任务状态缓存

```
key: task:status:{task_id}
type: string
value: "PROCESSING"
ttl: 300s
```

**用途**: 快速查询任务状态，减少数据库查询

---

## 6. 消息发布订阅

### 任务事件

```
channel: task:events
message: {
  "event": "task_created",
  "task_id": "xxx",
  "task_type": "send_email",
  "timestamp": 1234567890
}
```

**事件类型**:
- `task_created` - 任务创建
- `task_assigned` - 任务分配
- `task_completed` - 任务完成
- `task_failed` - 任务失败
- `task_cancelled` - 任务取消

**订阅示例**:
```go
pubsub := redis.Subscribe(ctx, "task:events")
for msg := range pubsub.Channel() {
    var event TaskEvent
    json.Unmarshal([]byte(msg.Payload), &event)
    handleEvent(event)
}
```

---

## 7. 监控指标

### 队列长度

```
key: metrics:queue:length
type: hash
fields:
  - high: 100
  - normal: 500
```

### Worker 统计

```
key: metrics:worker:count
type: string
value: 10
```

### 任务统计

```
key: metrics:task:count:{status}
type: string
value: 1000
```

**操作**:
```redis
# 增加计数
INCR metrics:task:count:success

# 获取统计
GET metrics:task:count:success
```

---

## 8. 限流控制

### 任务类型限流

```
key: ratelimit:task:{task_type}
type: string
value: {current_count}
ttl: 1s
```

**滑动窗口限流**:
```
key: ratelimit:task:{task_type}:{timestamp}
type: string
value: {count}
ttl: 60s
```

**操作**:
```redis
# 检查限流
INCR ratelimit:task:send_email
EXPIRE ratelimit:task:send_email 1

# 获取当前计数
GET ratelimit:task:send_email
```

---

## 9. 延迟队列（重试）

### 使用 Sorted Set 实现延迟队列

```
key: queue:delayed
type: sorted set
score: 执行时间戳
member: task_id
```

**操作**:
```redis
# 添加延迟任务
ZADD queue:delayed {timestamp} {task_id}

# 获取到期任务
ZRANGEBYSCORE queue:delayed 0 {current_timestamp}

# 移除已处理任务
ZREM queue:delayed {task_id}
```

**定时扫描器**:
```go
func (s *Scheduler) scanDelayedQueue(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            now := time.Now().Unix()
            // 获取到期任务
            taskIDs, _ := s.redis.ZRangeByScore(ctx, "queue:delayed", &redis.ZRangeBy{
                Min: "0",
                Max: fmt.Sprintf("%d", now),
            }).Result()
            
            for _, taskID := range taskIDs {
                // 移动到正常队列
                task, _ := s.taskRepo.GetByID(ctx, taskID)
                queueName := getQueueName(task.Priority)
                s.redis.LPush(ctx, queueName, taskID)
                
                // 从延迟队列移除
                s.redis.ZRem(ctx, "queue:delayed", taskID)
            }
        }
    }
}
```

---

## 10. 分布式锁（任务级别）

### 任务执行锁

```
key: lock:task:{task_id}
type: string
value: {worker_id}
ttl: 300s
```

**用途**: 防止同一任务被多个 Worker 同时执行

**操作**:
```redis
# 尝试获取锁
SET lock:task:{task_id} {worker_id} NX EX 300

# 释放锁
DEL lock:task:{task_id}
```

---

## Redis 高可用方案

### 1. Redis Sentinel（推荐）

```yaml
redis:
  sentinel:
    master: mymaster
    nodes:
      - sentinel1:26379
      - sentinel2:26379
      - sentinel3:26379
  password: xxx
  db: 0
```

**优势**:
- 自动故障转移
- 配置简单
- 适合中小规模

### 2. Redis Cluster

```yaml
redis:
  cluster:
    nodes:
      - redis1:6379
      - redis2:6379
      - redis3:6379
      - redis4:6379
      - redis5:6379
      - redis6:6379
```

**优势**:
- 数据分片
- 高可用
- 适合大规模

### 3. 数据持久化

```conf
# AOF 持久化（推荐）
appendonly yes
appendfsync everysec

# RDB 快照
save 900 1
save 300 10
save 60 10000
```

---

## 性能优化建议

### 1. 连接池配置

```go
redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,
    MinIdleConns: 10,
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

### 2. Pipeline 批量操作

```go
pipe := redis.Pipeline()
for _, taskID := range taskIDs {
    pipe.LPush(ctx, "queue:high", taskID)
}
pipe.Exec(ctx)
```

### 3. Lua 脚本原子操作

```go
script := redis.NewScript(`
    local task_id = ARGV[1]
    local worker_id = ARGV[2]
    
    -- 检查任务是否存在
    if redis.call("EXISTS", "task:cancel:" .. task_id) == 1 then
        return 0
    end
    
    -- 分配任务
    redis.call("LPUSH", "worker:" .. worker_id .. ":queue", task_id)
    return 1
`)

result, _ := script.Run(ctx, redis, []string{}, taskID, workerID).Result()
```

### 4. 监控关键指标

- 队列长度
- 连接数
- 命令执行时间
- 内存使用率
- 键空间命中率
