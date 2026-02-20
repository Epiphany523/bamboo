# MySQL 存储实现优化文档

## 优化概述

根据 `domain/model` 中的领域模型，对 MySQL 存储实现进行了全面优化，确保数据库字段与领域模型完全匹配。

## 主要变更

### 1. Task 表优化

#### 领域模型字段
```go
type Task struct {
    ID          int64                    // 自增主键
    TaskID      string                   // 任务唯一标识
    TaskType    string                   // 任务类型
    Priority    TaskPriority             // 优先级（0=Normal, 1=High）
    Status      TaskStatus               // 任务状态
    Payload     map[string]interface{}   // 任务参数
    Result      map[string]interface{}   // 执行结果
    ErrorMsg    string                   // 错误信息
    RetryCount  int                      // 重试次数
    MaxRetry    int                      // 最大重试次数
    Timeout     int                      // 超时时间（秒）
    WorkerID    string                   // Worker ID
    ScheduledAt time.Time                // 调度时间
    StartedAt   *time.Time               // 开始时间
    CompletedAt *time.Time               // 完成时间
    CreatedAt   time.Time                // 创建时间
    UpdatedAt   time.Time                // 更新时间
}
```

#### 修复的问题
- ✅ 添加 `id` 自增主键字段
- ✅ 修正 `error_message` 字段映射到 `ErrorMsg`
- ✅ 修正 `scheduled_at` 字段类型从 `*time.Time` 改为 `time.Time`
- ✅ 添加 `updated_at` 字段的处理
- ✅ 统一所有查询语句包含完整字段列表

#### 表结构
```sql
CREATE TABLE IF NOT EXISTS task (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(64) UNIQUE NOT NULL,
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
    scheduled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    INDEX idx_task_id (task_id),
    INDEX idx_status (status),
    INDEX idx_task_type (task_type),
    INDEX idx_priority (priority),
    INDEX idx_scheduled_at (scheduled_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2. TaskLog 表优化

#### 领域模型字段
```go
type TaskLog struct {
    ID          int64      // 自增主键
    TaskID      string     // 任务ID
    LogType     LogType    // 日志类型
    FromStatus  TaskStatus // 原状态
    ToStatus    TaskStatus // 新状态
    Message     string     // 日志消息
    WorkerID    string     // Worker ID
    RetryCount  int        // 重试次数
    ErrorDetail string     // 错误详情
    CreatedAt   time.Time  // 创建时间
}
```

#### 修复的问题
- ✅ 移除 `log_id` 字符串字段，使用 `id` 自增主键
- ✅ 移除 `details` JSON 字段
- ✅ 添加 `worker_id` 字段
- ✅ 添加 `retry_count` 字段
- ✅ 添加 `error_detail` 字段
- ✅ 字段完全匹配领域模型

#### 表结构
```sql
CREATE TABLE IF NOT EXISTS task_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    log_type VARCHAR(32) NOT NULL,
    from_status VARCHAR(32),
    to_status VARCHAR(32),
    message TEXT,
    worker_id VARCHAR(64),
    retry_count INT DEFAULT 0,
    error_detail TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_log_type (log_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. Worker 表优化

#### 领域模型字段
```go
type Worker struct {
    ID             int64        // 自增主键
    WorkerID       string       // Worker 唯一标识
    WorkerName     string       // Worker 名称
    Address        string       // Worker 地址
    Status         WorkerStatus // Worker 状态
    Capacity       int          // 容量
    CurrentLoad    int          // 当前负载
    SupportedTypes []string     // 支持的任务类型
    LastHeartbeat  time.Time    // 最后心跳时间
    CreatedAt      time.Time    // 创建时间
    UpdatedAt      time.Time    // 更新时间
}
```

#### 修复的问题
- ✅ 添加 `id` 自增主键字段
- ✅ 添加 `created_at` 字段的处理
- ✅ 添加 `updated_at` 字段的处理
- ✅ 所有查询和更新操作包含完整字段

#### 表结构
```sql
CREATE TABLE IF NOT EXISTS worker (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    worker_id VARCHAR(64) UNIQUE NOT NULL,
    worker_name VARCHAR(128) NOT NULL,
    address VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL,
    capacity INT NOT NULL DEFAULT 10,
    current_load INT NOT NULL DEFAULT 0,
    supported_types JSON,
    last_heartbeat TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_worker_id (worker_id),
    INDEX idx_status (status),
    INDEX idx_last_heartbeat (last_heartbeat)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 4. TaskConfig 表优化

#### 领域模型字段
```go
type TaskConfig struct {
    ID              int64                    // 自增主键
    TaskType        string                   // 任务类型
    TaskName        string                   // 任务名称
    Description     string                   // 描述
    ExecutorType    ExecutorType             // 执行器类型
    ExecutorConfig  map[string]interface{}   // 执行器配置
    DefaultTimeout  int                      // 默认超时时间
    DefaultMaxRetry int                      // 默认最大重试次数
    RetryStrategy   RetryStrategy            // 重试策略
    RetryDelay      int                      // 重试延迟
    BackoffRate     float64                  // 退避率
    MaxConcurrent   int                      // 最大并发数
    Enabled         bool                     // 是否启用
    CreatedAt       time.Time                // 创建时间
    UpdatedAt       time.Time                // 更新时间
}
```

#### 表结构
```sql
CREATE TABLE IF NOT EXISTS task_config (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_type VARCHAR(64) UNIQUE NOT NULL,
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
    INDEX idx_task_type (task_type),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 优化后的优势

### 1. 数据一致性
- 数据库字段与领域模型完全匹配
- 避免字段映射错误
- 类型安全

### 2. 性能优化
- 使用自增主键提高查询性能
- 添加必要的索引
- 优化查询语句

### 3. 可维护性
- 代码结构清晰
- 字段命名统一
- 易于扩展

### 4. 数据完整性
- 添加 `created_at` 和 `updated_at` 时间戳
- 支持审计和追踪
- 便于数据分析


## 测试验证

### 1. 单元测试
```bash
cd asynctaskmanager
go test ./infrastructure/mysql/... -v
```

### 2. 集成测试
```bash
# 启动 MySQL
docker run -d -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=asynctask \
  mysql:8.0

# 初始化数据库
mysql -h localhost -u root -ppassword asynctask < cmd/asynctaskmanager/scripts/init_db.sql

# 运行测试
cd cmd/asynctaskmanager
go run main.go -id=test-server -grpc-port=9090
```

## 注意事项

1. **时间字段处理**
   - `scheduled_at` 是必填字段，默认为当前时间
   - `started_at` 和 `completed_at` 是可空字段
   - 使用 `sql.NullTime` 处理可空时间字段

2. **JSON 字段**
   - `payload`, `result`, `executor_config`, `supported_types` 使用 JSON 类型
   - 需要序列化/反序列化处理
   - 注意 JSON 字段的大小限制

3. **索引优化**
   - 根据查询模式添加合适的索引
   - 避免过多索引影响写入性能
   - 定期分析慢查询并优化

4. **字符集**
   - 统一使用 `utf8mb4` 字符集
   - 支持完整的 Unicode 字符
   - 包括 emoji 等特殊字符

## 相关文件

- `asynctaskmanager/domain/model/` - 领域模型定义
- `asynctaskmanager/infrastructure/mysql/` - MySQL 实现
- `cmd/asynctaskmanager/scripts/init_db.sql` - 数据库初始化脚本
- `asynctaskmanager/infrastructure/mysql/mysql_client.go` - 表结构定义

