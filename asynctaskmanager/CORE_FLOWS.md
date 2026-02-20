# 核心业务流程

## 1. 创建任务流程

```
用户请求
   ↓
API Gateway
   ↓
TaskService.CreateTask()
   ↓
1. 参数验证
   ↓
2. 生成 task_id (UUID/雪花算法)
   ↓
3. 查询 task_config 获取默认配置
   ↓
4. 插入 task 表 (status=PENDING)
   ↓
5. 记录日志到 task_logs
   ↓
6. 根据优先级推送到 Redis 队列
   - priority=1 → queue:high
   - priority=0 → queue:normal
   ↓
7. 发布任务创建事件 (Redis Pub/Sub)
   ↓
8. 返回 task_id 给用户
```

**详细步骤**:

```go
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
    // 1. 参数验证
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    // 2. 生成任务ID
    taskID := generateTaskID()
    
    // 3. 获取任务配置
    config, err := s.taskConfigRepo.GetByType(ctx, req.TaskType)
    if err != nil {
        return nil, err
    }
    
    // 4. 创建任务对象
    task := &Task{
        TaskID:      taskID,
        TaskType:    req.TaskType,
        Priority:    req.Priority,
        Status:      StatusPending,
        Payload:     req.Payload,
        MaxRetry:    config.DefaultMaxRetry,
        Timeout:     config.DefaultTimeout,
        ScheduledAt: time.Now(),
    }
    
    // 5. 开启事务
    tx, _ := s.db.BeginTx(ctx)
    defer tx.Rollback()
    
    // 6. 插入任务
    if err := s.taskRepo.Create(ctx, tx, task); err != nil {
        return nil, err
    }
    
    // 7. 记录日志
    log := &TaskLog{
        TaskID:    taskID,
        LogType:   "STATE_CHANGE",
        ToStatus:  StatusPending,
        Message:   "Task created",
    }
    s.taskLogRepo.Create(ctx, tx, log)
    
    // 8. 提交事务
    tx.Commit()
    
    // 9. 推送到队列
    queueName := getQueueName(task.Priority)
    s.redis.LPush(ctx, queueName, taskID)
    
    // 10. 发布事件
    s.redis.Publish(ctx, "task:created", taskID)
    
    return task, nil
}
```

---

## 2. 任务调度流程

```
Scheduler (Leader)
   ↓
1. 从 Redis 队列拉取任务
   - 优先消费 queue:high
   - 再消费 queue:normal
   ↓
2. 获取任务详情 (MySQL)
   ↓
3. 检查任务状态
   - 如果不是 PENDING，跳过
   ↓
4. 获取可用 Worker 列表 (Redis)
   ↓
5. 负载均衡选择 Worker
   - 策略：最少任务优先/轮询/一致性哈希
   ↓
6. 更新任务状态为 PROCESSING
   ↓
7. 将任务分配给 Worker
   - 推送到 worker:{worker_id}:queue
   ↓
8. 记录分配日志
```

**详细实现**:

```go
func (s *Scheduler) scheduleLoop(ctx context.Context) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.scheduleTasks(ctx)
        }
    }
}

func (s *Scheduler) scheduleTasks(ctx context.Context) {
    // 1. 优先处理高优先级队列
    taskID, err := s.redis.RPop(ctx, "queue:high")
    if err == redis.Nil {
        // 2. 处理普通优先级队列
        taskID, err = s.redis.RPop(ctx, "queue:normal")
        if err == redis.Nil {
            return // 队列为空
        }
    }
    
    // 3. 获取任务详情
    task, err := s.taskRepo.GetByID(ctx, taskID)
    if err != nil {
        return
    }
    
    // 4. 检查任务状态
    if task.Status != StatusPending {
        return
    }
    
    // 5. 获取可用 Worker
    workers, err := s.workerRegistry.GetAvailableWorkers(ctx, task.TaskType)
    if len(workers) == 0 {
        // 没有可用 Worker，重新放回队列
        s.redis.LPush(ctx, getQueueName(task.Priority), taskID)
        return
    }
    
    // 6. 负载均衡选择 Worker
    worker := s.loadBalancer.Select(workers, taskID)
    
    // 7. 更新任务状态
    task.Status = StatusProcessing
    task.WorkerID = worker.ID
    task.StartedAt = time.Now()
    s.taskRepo.Update(ctx, task)
    
    // 8. 分配任务给 Worker
    s.redis.LPush(ctx, fmt.Sprintf("worker:%s:queue", worker.ID), taskID)
    
    // 9. 记录日志
    s.taskLogRepo.Create(ctx, &TaskLog{
        TaskID:     taskID,
        LogType:    "STATE_CHANGE",
        FromStatus: StatusPending,
        ToStatus:   StatusProcessing,
        WorkerID:   worker.ID,
        Message:    "Task assigned to worker",
    })
}
```

---

## 3. Worker 执行任务流程

```
Worker
   ↓
1. 从队列拉取任务 ID
   - worker:{worker_id}:queue
   ↓
2. 获取任务详情 (MySQL)
   ↓
3. 检查取消标记
   - 如果已取消，更新状态并退出
   ↓
4. 根据 task_type 获取 Executor
   ↓
5. 设置超时 Context
   ↓
6. 执行任务
   ↓
7. 处理执行结果
   - 成功 → SUCCESS
   - 失败 → FAILED (判断是否重试)
   - 超时 → TIMEOUT (判断是否重试)
   ↓
8. 更新任务状态和结果
   ↓
9. 记录执行日志
   ↓
10. 更新 Worker 负载
```

**详细实现**:

```go
func (w *Worker) processTask(ctx context.Context, taskID string) error {
    // 1. 获取任务
    task, err := w.taskRepo.GetByID(ctx, taskID)
    if err != nil {
        return err
    }
    
    // 2. 检查取消标记
    if w.isCancelled(ctx, taskID) {
        task.Status = StatusCancelled
        w.taskRepo.Update(ctx, task)
        return nil
    }
    
    // 3. 获取执行器
    executor, err := w.executorRegistry.Get(task.TaskType)
    if err != nil {
        return err
    }
    
    // 4. 设置超时
    execCtx, cancel := context.WithTimeout(ctx, time.Duration(task.Timeout)*time.Second)
    defer cancel()
    
    // 5. 执行任务
    result, err := executor.Execute(execCtx, task)
    
    // 6. 处理结果
    if err != nil {
        if execCtx.Err() == context.DeadlineExceeded {
            // 超时
            task.Status = StatusTimeout
            task.ErrorMsg = "Task execution timeout"
        } else {
            // 失败
            task.Status = StatusFailed
            task.ErrorMsg = err.Error()
        }
        
        // 判断是否需要重试
        if task.RetryCount < task.MaxRetry {
            w.retryTask(ctx, task)
            return nil
        }
    } else {
        // 成功
        task.Status = StatusSuccess
        task.Result = result
    }
    
    // 7. 更新任务
    task.CompletedAt = time.Now()
    w.taskRepo.Update(ctx, task)
    
    // 8. 记录日志
    w.taskLogRepo.Create(ctx, &TaskLog{
        TaskID:     taskID,
        LogType:    "STATE_CHANGE",
        FromStatus: StatusProcessing,
        ToStatus:   task.Status,
        WorkerID:   w.ID,
        Message:    "Task execution completed",
    })
    
    return nil
}
```

---

## 4. 查询任务流程

```
用户请求
   ↓
API Gateway
   ↓
TaskService.GetTask(taskID)
   ↓
1. 查询 Redis 缓存
   - key: task:cache:{task_id}
   ↓
2. 缓存命中？
   - 是 → 返回结果
   - 否 → 继续
   ↓
3. 查询 MySQL
   ↓
4. 查询到结果？
   - 是 → 缓存到 Redis (TTL: 5分钟)
   - 否 → 返回任务不存在
   ↓
5. 返回任务信息
```

---

## 5. 取消任务流程

```
用户请求
   ↓
API Gateway
   ↓
TaskService.CancelTask(taskID)
   ↓
1. 查询任务当前状态
   ↓
2. 判断状态
   ├─ PENDING
   │   ↓
   │   从队列中删除
   │   ↓
   │   更新状态为 CANCELLED
   │
   ├─ PROCESSING
   │   ↓
   │   设置取消标记 (Redis)
   │   - key: task:cancel:{task_id}
   │   ↓
   │   Worker 检测到后终止
   │
   └─ 其他状态
       ↓
       返回无法取消
```

---

## 6. 任务重试流程

```
任务失败/超时
   ↓
1. 检查重试次数
   - retry_count >= max_retry？
   - 是 → 标记为最终失败
   - 否 → 继续
   ↓
2. 增加重试次数
   ↓
3. 计算下次重试时间
   - FIXED: 固定延迟
   - EXPONENTIAL: delay * (backoff_rate ^ retry_count)
   ↓
4. 更新任务状态为 PENDING
   ↓
5. 延迟后推送到队列
   - 使用 Redis 延迟队列
   - 或使用定时任务扫描
   ↓
6. 记录重试日志
```

**重试策略实现**:

```go
func (s *TaskService) retryTask(ctx context.Context, task *Task) error {
    // 1. 增加重试次数
    task.RetryCount++
    
    // 2. 获取配置
    config, _ := s.taskConfigRepo.GetByType(ctx, task.TaskType)
    
    // 3. 计算延迟时间
    var delay time.Duration
    if config.RetryStrategy == "EXPONENTIAL" {
        delay = time.Duration(config.RetryDelay) * time.Second * 
                time.Duration(math.Pow(config.BackoffRate, float64(task.RetryCount)))
    } else {
        delay = time.Duration(config.RetryDelay) * time.Second
    }
    
    // 4. 更新任务状态
    task.Status = StatusPending
    task.ScheduledAt = time.Now().Add(delay)
    s.taskRepo.Update(ctx, task)
    
    // 5. 延迟推送到队列
    go func() {
        time.Sleep(delay)
        queueName := getQueueName(task.Priority)
        s.redis.LPush(context.Background(), queueName, task.TaskID)
    }()
    
    // 6. 记录日志
    s.taskLogRepo.Create(ctx, &TaskLog{
        TaskID:     task.TaskID,
        LogType:    "RETRY",
        RetryCount: task.RetryCount,
        Message:    fmt.Sprintf("Task retry scheduled, delay: %v", delay),
    })
    
    return nil
}
```
