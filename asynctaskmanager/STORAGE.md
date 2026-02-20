# 存储实现说明

asynctaskmanager 支持多种存储实现，包括内存存储和 MySQL 存储。

## 存储架构

基于 DDD 模式，存储层实现了 Repository 接口，支持灵活切换不同的存储后端。

```
domain/repository/          # 仓储接口定义
  ├── task_repository.go
  ├── task_config_repository.go
  ├── task_log_repository.go
  └── worker_repository.go

infrastructure/
  ├── memory/               # 内存实现（用于测试和开发）
  │   ├── task_repository_impl.go
  │   ├── task_config_repository_impl.go
  │   └── task_log_repository_impl.go
  └── mysql/                # MySQL 实现（用于生产环境）
      ├── mysql_client.go
      ├── task_repository_impl.go
      ├── task_config_repository_impl.go
      ├── task_log_repository_impl.go
      └── worker_repository_impl.go
```

## 内存存储

内存存储使用 Go 的 map 和 sync.RWMutex 实现，适用于：
- 单元测试
- 本地开发
- 原型验证
- 不需要持久化的场景

### 使用示例

```go
import "bamboo/asynctaskmanager/infrastructure/memory"

// 创建内存仓储
taskRepo := memory.NewTaskRepository()
taskConfigRepo := memory.NewTaskConfigRepository()
taskLogRepo := memory.NewTaskLogRepository()
```

### 特点

- 无需外部依赖
- 启动快速
- 数据不持久化（重启后丢失）
- 不支持分布式部署

## MySQL 存储

MySQL 存储使用标准的 database/sql 包实现，适用于：
- 生产环境
- 需要数据持久化
- 分布式部署
- 高可用场景

### 数据库表结构

#### 1. task 表

存储任务信息：

```sql
CREATE TABLE task (
    task_id VARCHAR(64) PRIMARY KEY,
    task_type VARCHAR(64) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    payload JSON,
    result JSON,
    error_message TEXT,
    worker_id VARCHAR(64),
    retry_count INT NOT NULL DEFAULT 0,
    max_retry INT NOT NULL DEFAULT 3,
    timeout INT NOT NULL DEFAULT 30,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    scheduled_at TIMESTAMP NULL,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    INDEX idx_status (status),
    INDEX idx_task_type (task_type),
    INDEX idx_worker_id (worker_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 2. task_config 表

存储任务配置：

```sql
CREATE TABLE task_config (
    task_type VARCHAR(64) PRIMARY KEY,
    task_name VARCHAR(128) NOT NULL,
    description TEXT,
    executor_type VARCHAR(32) NOT NULL,
    executor_config JSON,
    default_timeout INT NOT NULL DEFAULT 30,
    default_max_retry INT NOT NULL DEFAULT 3,
    retry_strategy VARCHAR(32) NOT NULL,
    retry_delay INT NOT NULL DEFAULT 5,
    backoff_rate DECIMAL(10,2) NOT NULL DEFAULT 2.0,
    max_concurrent INT NOT NULL DEFAULT 10,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 3. task_log 表

存储任务日志：

```sql
CREATE TABLE task_log (
    log_id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    log_type VARCHAR(32) NOT NULL,
    from_status VARCHAR(32),
    to_status VARCHAR(32),
    message TEXT,
    details JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_log_type (log_type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 4. worker 表

存储 Worker 信息：

```sql
CREATE TABLE worker (
    worker_id VARCHAR(64) PRIMARY KEY,
    worker_name VARCHAR(128) NOT NULL,
    address VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL,
    capacity INT NOT NULL DEFAULT 10,
    current_load INT NOT NULL DEFAULT 0,
    supported_types JSON,
    last_heartbeat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_last_heartbeat (last_heartbeat)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 使用示例

```go
import (
    "bamboo/asynctaskmanager/infrastructure/mysql"
    "time"
)

// 1. 创建 MySQL 客户端
cfg := mysql.Config{
    Host:     "localhost",
    Port:     3306,
    User:     "root",
    Password: "password",
    Database: "asynctask",
    MaxOpen:  10,
    MaxIdle:  5,
    MaxLife:  time.Hour,
}

client, err := mysql.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 2. 初始化数据库表结构
if err := client.InitSchema(); err != nil {
    log.Fatal(err)
}

// 3. 创建仓储实例
taskRepo := mysql.NewTaskRepository(client)
taskConfigRepo := mysql.NewTaskConfigRepository(client)
taskLogRepo := mysql.NewTaskLogRepository(client)
workerRepo := mysql.NewWorkerRepository(client)

// 4. 使用仓储（与内存实现接口相同）
task := &model.Task{
    TaskID:   "task-001",
    TaskType: "email_task",
    Priority: model.PriorityHigh,
    Status:   model.TaskPending,
    // ...
}
err = taskRepo.Create(ctx, task)
```

### 配置说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| Host | MySQL 主机地址 | localhost |
| Port | MySQL 端口 | 3306 |
| User | 数据库用户名 | - |
| Password | 数据库密码 | - |
| Database | 数据库名称 | - |
| MaxOpen | 最大打开连接数 | 0（无限制） |
| MaxIdle | 最大空闲连接数 | 0（无限制） |
| MaxLife | 连接最大生命周期 | 0（永久） |

### 性能优化建议

1. **索引优化**
   - task 表按 status、task_type、created_at 建立索引
   - worker 表按 status、last_heartbeat 建立索引
   - task_log 表按 task_id、created_at 建立索引

2. **连接池配置**
   ```go
   cfg.MaxOpen = 20  // 根据并发量调整
   cfg.MaxIdle = 10  // 保持一定空闲连接
   cfg.MaxLife = time.Hour  // 定期回收连接
   ```

3. **JSON 字段优化**
   - payload、result、details 使用 JSON 类型存储
   - MySQL 5.7+ 支持 JSON 函数查询
   - 避免存储过大的 JSON 数据

4. **分区表**（可选）
   ```sql
   -- 按月分区 task 表
   ALTER TABLE task PARTITION BY RANGE (YEAR(created_at) * 100 + MONTH(created_at)) (
       PARTITION p202401 VALUES LESS THAN (202402),
       PARTITION p202402 VALUES LESS THAN (202403),
       ...
   );
   ```

## 存储切换

由于使用了 Repository 接口，切换存储实现非常简单：

```go
// 开发环境：使用内存存储
if env == "development" {
    taskRepo = memory.NewTaskRepository()
}

// 生产环境：使用 MySQL 存储
if env == "production" {
    client, _ := mysql.NewClient(mysqlConfig)
    taskRepo = mysql.NewTaskRepository(client)
}

// 应用层代码无需修改
taskService := application.NewTaskService(
    taskRepo,
    taskLogRepo,
    taskConfigRepo,
    queueManager,
)
```

## 扩展其他存储

如需支持其他存储（如 PostgreSQL、MongoDB），只需：

1. 在 `infrastructure/` 下创建新目录
2. 实现 `domain/repository/` 中定义的接口
3. 在应用启动时选择对应的实现

示例：

```go
// infrastructure/postgres/task_repository_impl.go
type TaskRepositoryImpl struct {
    client *PostgresClient
}

func (r *TaskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
    // PostgreSQL 实现
}

// 其他方法...
```

## 完整示例

参考 `examples/mysql_usage.go` 查看完整的 MySQL 存储使用示例。

运行示例：

```bash
# 确保 MySQL 已启动并创建数据库
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS asynctask"

# 运行示例
go run asynctaskmanager/examples/mysql_usage.go
```
