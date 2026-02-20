# 快速开始

## 5 分钟快速体验

### 前置条件

- Go 1.23+
- MySQL 8.0+
- Redis 6.0+

### 1. 启动依赖服务

```bash
# 启动 MySQL
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=task_manager \
  -p 3306:3306 \
  mysql:8.0

# 启动 Redis
docker run -d --name redis \
  -p 6379:6379 \
  redis:latest
```

### 2. 初始化数据库

```bash
mysql -h 127.0.0.1 -u root -proot task_manager < schema.sql
```

### 3. 配置文件

创建 `config.yaml`:

```yaml
app:
  name: "async-task-manager"

database:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "root"
  database: "task_manager"

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

scheduler:
  enabled: true
  scan_interval: 100ms

worker:
  enabled: true
  id: "worker-001"
  capacity: 10
```

### 4. 启动服务

```bash
# 启动 Scheduler + Worker
go run cmd/server/main.go
```

### 5. 创建任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "send_email",
    "priority": 1,
    "payload": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "Test email"
    }
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### 6. 查询任务

```bash
curl http://localhost:8080/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "task_type": "send_email",
    "status": "SUCCESS",
    "result": {
      "message_id": "abc123"
    },
    "created_at": "2024-02-16T10:00:00Z",
    "completed_at": "2024-02-16T10:00:05Z"
  }
}
```

---

## API 文档

### 1. 创建任务

**请求**:
```http
POST /api/v1/tasks
Content-Type: application/json

{
  "task_type": "send_email",
  "priority": 1,
  "payload": {
    "to": "user@example.com",
    "subject": "Hello",
    "body": "Test email"
  },
  "timeout": 300,
  "max_retry": 3
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### 2. 查询任务

**请求**:
```http
GET /api/v1/tasks/{task_id}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "task_type": "send_email",
    "status": "PROCESSING",
    "payload": {...},
    "result": null,
    "retry_count": 0,
    "created_at": "2024-02-16T10:00:00Z",
    "started_at": "2024-02-16T10:00:01Z"
  }
}
```

### 3. 取消任务

**请求**:
```http
POST /api/v1/tasks/{task_id}/cancel
```

**响应**:
```json
{
  "code": 0,
  "message": "Task cancelled successfully"
}
```

### 4. 查询任务日志

**请求**:
```http
GET /api/v1/tasks/{task_id}/logs
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "log_type": "STATE_CHANGE",
      "from_status": null,
      "to_status": "PENDING",
      "message": "Task created",
      "created_at": "2024-02-16T10:00:00Z"
    },
    {
      "log_type": "STATE_CHANGE",
      "from_status": "PENDING",
      "to_status": "PROCESSING",
      "message": "Task assigned to worker",
      "worker_id": "worker-001",
      "created_at": "2024-02-16T10:00:01Z"
    }
  ]
}
```

---

## 代码示例

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourusername/async-task-manager/client"
)

func main() {
    // 创建客户端
    c := client.New("http://localhost:8080")
    
    // 创建任务
    task, err := c.CreateTask(context.Background(), &client.CreateTaskRequest{
        TaskType: "send_email",
        Priority: 1,
        Payload: map[string]interface{}{
            "to":      "user@example.com",
            "subject": "Hello",
            "body":    "Test email",
        },
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Task created: %s\n", task.TaskID)
    
    // 查询任务
    task, err = c.GetTask(context.Background(), task.TaskID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Task status: %s\n", task.Status)
}
```

### Python SDK

```python
from async_task_manager import Client

# 创建客户端
client = Client("http://localhost:8080")

# 创建任务
task = client.create_task(
    task_type="send_email",
    priority=1,
    payload={
        "to": "user@example.com",
        "subject": "Hello",
        "body": "Test email"
    }
)

print(f"Task created: {task.task_id}")

# 查询任务
task = client.get_task(task.task_id)
print(f"Task status: {task.status}")
```

---

## 自定义 Executor

### 1. 实现 Executor 接口

```go
package executor

import (
    "context"
    "fmt"
    
    "github.com/yourusername/async-task-manager/domain/model"
)

type EmailExecutor struct {
    smtpHost string
    smtpPort int
}

func NewEmailExecutor(host string, port int) *EmailExecutor {
    return &EmailExecutor{
        smtpHost: host,
        smtpPort: port,
    }
}

func (e *EmailExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
    // 解析 payload
    payload := task.Payload.(map[string]interface{})
    to := payload["to"].(string)
    subject := payload["subject"].(string)
    body := payload["body"].(string)
    
    // 发送邮件
    err := e.sendEmail(to, subject, body)
    if err != nil {
        return nil, err
    }
    
    return &model.TaskResult{
        Success: true,
        Data: map[string]interface{}{
            "message_id": "abc123",
        },
    }, nil
}

func (e *EmailExecutor) Type() string {
    return "email"
}

func (e *EmailExecutor) SupportedTaskTypes() []string {
    return []string{"send_email"}
}

func (e *EmailExecutor) sendEmail(to, subject, body string) error {
    // 实际的邮件发送逻辑
    fmt.Printf("Sending email to %s: %s\n", to, subject)
    return nil
}
```

### 2. 注册 Executor

```go
package main

import (
    "context"
    
    "github.com/yourusername/async-task-manager/executor"
    "github.com/yourusername/async-task-manager/worker"
)

func main() {
    // 创建 Worker
    w := worker.New(config)
    
    // 注册自定义 Executor
    emailExecutor := executor.NewEmailExecutor("smtp.example.com", 587)
    w.RegisterExecutor(emailExecutor)
    
    // 启动 Worker
    w.Start(context.Background())
}
```

---

## 监控

### Prometheus 指标

访问 `http://localhost:9090/metrics` 查看指标：

```
# 任务创建总数
task_created_total{task_type="send_email"} 1000

# 任务执行时长
task_duration_seconds{task_type="send_email",status="success",quantile="0.5"} 0.5
task_duration_seconds{task_type="send_email",status="success",quantile="0.95"} 1.2
task_duration_seconds{task_type="send_email",status="success",quantile="0.99"} 2.5

# Worker 负载
worker_load{worker_id="worker-001"} 5

# 队列长度
queue_length{queue="high"} 100
queue_length{queue="normal"} 500
```

### Grafana 仪表板

导入 `grafana-dashboard.json` 查看可视化监控面板。

---

## 故障排查

### 问题：任务一直处于 PENDING 状态

**可能原因**:
1. 没有可用的 Worker
2. Worker 不支持该任务类型
3. Scheduler 未启动或不是 Leader

**排查步骤**:
```bash
# 检查 Worker 状态
redis-cli KEYS "worker:*"

# 检查 Leader
redis-cli GET "scheduler:leader"

# 检查队列长度
redis-cli LLEN "queue:high"
redis-cli LLEN "queue:normal"
```

### 问题：任务执行失败

**可能原因**:
1. 业务方服务不可用
2. 任务参数错误
3. 执行超时

**排查步骤**:
```bash
# 查询任务日志
curl http://localhost:8080/api/v1/tasks/{task_id}/logs

# 查看 Worker 日志
tail -f logs/worker.log
```

---

## 下一步

- 阅读 [COMPONENTS.md](./COMPONENTS.md) 了解组件设计
- 阅读 [CORE_FLOWS.md](./CORE_FLOWS.md) 了解核心流程
- 阅读 [EXTENSIBILITY.md](./EXTENSIBILITY.md) 了解如何扩展
