# 分布式异步任务管理器故障容错修复

## 问题分析

### 问题1：Worker异常退出导致任务丢失
**现象**：调度器将任务分发给worker后，worker还没有来得及处理task，worker就异常退出了，导致已经分配给该worker的任务无法执行。

**根本原因**：
- 任务被分配到Worker的专用队列后，如果Worker异常退出，这些任务会永远留在队列中
- 调度器没有监控Worker的健康状态和恢复机制
- 缺少故障检测和任务恢复机制

### 问题2：无法取消正在执行中的任务
**现象**：在取消任务时，无法取消正在执行中的任务。

**根本原因**：
- 原有的取消机制只能在任务执行前检查取消标记
- 一旦任务开始执行，没有机制中断正在运行的任务
- 缺少实时的取消通知机制

## 修复方案

### 修复1：Worker故障恢复机制

#### 1.1 调度器增加故障恢复检查
在 `SchedulerService.runAsLeader()` 中增加定期故障恢复检查：

```go
case <-recoveryTicker.C:
    // 故障恢复：检查失效的 Worker 并恢复其任务
    if err := s.recoverFailedWorkerTasks(ctx); err != nil {
        log.Printf("recover failed worker tasks failed: %v", err)
    }
```

#### 1.2 实现故障检测和任务恢复
新增 `recoverFailedWorkerTasks()` 方法：
- 定期检查所有Worker的健康状态（基于心跳超时）
- 发现失效Worker时，从其队列中恢复所有未处理任务
- 将恢复的任务重新标记为待处理状态并推送到调度队列
- 标记失效Worker为离线状态

#### 1.3 任务恢复流程
```go
func (s *SchedulerService) recoverWorkerTasks(ctx context.Context, workerID string) error {
    for {
        // 从失效Worker队列取出任务
        taskID, err := s.queueManager.PopFromWorkerQueue(ctx, workerID)
        if err != nil {
            break // 队列为空
        }
        
        // 重置任务状态为待处理
        task.MarkAsPending()
        s.taskRepo.Update(ctx, task)
        
        // 重新推送到调度队列
        s.queueManager.PushTask(ctx, taskID, task.Priority)
    }
}
```

### 修复2：实时任务取消机制

#### 2.1 Worker增加运行任务跟踪
在 `WorkerService` 中增加正在执行任务的跟踪：

```go
type WorkerService struct {
    // ... 其他字段
    runningTasks map[string]context.CancelFunc // 正在执行的任务及其取消函数
    taskMutex    sync.RWMutex                  // 保护并发访问
}
```

#### 2.2 任务执行时注册取消函数
在 `processTask()` 中：
```go
// 注册正在执行的任务，以便支持取消
s.taskMutex.Lock()
s.runningTasks[taskID] = cancel
s.taskMutex.Unlock()

// 执行完成后清理
defer func() {
    s.taskMutex.Lock()
    delete(s.runningTasks, taskID)
    s.taskMutex.Unlock()
}()
```

#### 2.3 Redis Pub/Sub取消通知机制
- **发布端**：TaskService在取消正在执行的任务时，通过Redis Pub/Sub发送取消通知
- **订阅端**：WorkerService启动时订阅专属的取消通知频道

```go
// 发布取消通知
func (qm *QueueManager) PublishCancelNotification(ctx context.Context, workerID, taskID string) error {
    channel := fmt.Sprintf("cancel:%s", workerID)
    return qm.client.Publish(ctx, channel, taskID)
}

// Worker监听取消通知
func (s *WorkerService) cancelListener(ctx context.Context) {
    pubsub := s.queueManager.SubscribeCancelChannel(ctx, s.worker.WorkerID)
    defer pubsub.Close()
    
    for {
        msg, err := pubsub.ReceiveMessage(ctx)
        if err == nil && msg.Payload != "" {
            s.cancelRunningTask(msg.Payload)
        }
    }
}
```

#### 2.4 增强的任务取消逻辑
在 `TaskService.CancelTask()` 中：
```go
if task.Status == model.StatusProcessing {
    // 任务正在执行中，需要通知 Worker 取消
    if task.WorkerID != "" {
        // 发送取消通知给 Worker
        qm.PublishCancelNotification(ctx, task.WorkerID, taskID)
    }
    
    // 设置取消标记，作为备用机制
    qm.SetCancelMark(ctx, taskID)
}
```

## 技术实现细节

### 故障检测机制
- **心跳超时检测**：基于Worker心跳时间戳判断Worker是否失效
- **定期检查**：调度器每60秒检查一次Worker健康状态
- **任务状态验证**：只恢复状态为"处理中"且分配给失效Worker的任务

### 取消通知机制
- **Redis Pub/Sub**：使用Redis发布订阅模式实现实时通知
- **Context取消**：通过Go的context.CancelFunc中断正在执行的任务
- **双重保障**：Pub/Sub通知 + 取消标记，确保取消操作的可靠性

### 并发安全
- **读写锁**：使用sync.RWMutex保护runningTasks映射的并发访问
- **原子操作**：任务状态更新和队列操作保持原子性
- **资源清理**：确保任务执行完成后及时清理相关资源

## 修复效果

### 问题1修复效果
- ✅ Worker异常退出后，其未处理任务会被自动恢复
- ✅ 恢复的任务重新进入调度队列，可被其他健康Worker处理
- ✅ 系统具备自愈能力，提高整体可用性

### 问题2修复效果
- ✅ 支持取消正在执行中的任务
- ✅ 实时取消通知，响应速度快
- ✅ 任务执行上下文被正确中断，避免资源浪费

## 测试建议

### 故障恢复测试
1. 启动多个Worker实例
2. 提交一批任务
3. 在任务分配后、执行前强制终止某个Worker
4. 观察任务是否被自动恢复并重新分配

### 取消功能测试
1. 提交一个长时间运行的任务
2. 在任务执行过程中调用取消接口
3. 验证任务是否被及时中断
4. 检查任务状态是否正确更新为"已取消"

这些修复确保了分布式异步任务管理器在面对Worker故障和任务取消需求时的健壮性和可靠性。