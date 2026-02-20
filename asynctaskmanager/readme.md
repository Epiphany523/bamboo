## 分布式异步任务管理

### 概述

该服务是一个高可用的分布式异步任务管理系统，专注于任务的调度和生命周期管理，不负责具体的业务执行。业务执行通过调用业务方的 RPC 接口实现，保证了系统的解耦和灵活性。

### 核心功能

- 任务创建：支持优先级、超时时间、重试策略等配置
- 任务查询：实时查询任务状态和执行结果
- 任务取消：支持取消队列中和执行中的任务
- 任务重试：支持自动重试和指数退避策略
- 任务监控：提供完整的监控指标和告警机制

### 技术栈选择

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 开发语言 | Go 1.23+ | 高性能、并发友好 |
| 数据库 | MySQL 8.0 | 持久化存储任务数据 |
| 消息队列 | Redis Stream | 任务队列和消息传递 |
| 服务发现 | Redis | Worker 注册和发现 |
| 分布式协调 | Redis 分布式锁 | Leader 选举和任务分配 |
| 监控 | Prometheus / 自定义 | 面向接口设计，易于替换 |

---


## 任务模型设计

### 任务状态定义

```go
const (
    StatusPending    TaskStatus = "PENDING"    // 待处理
    StatusProcessing TaskStatus = "PROCESSING" // 处理中
    StatusSuccess    TaskStatus = "SUCCESS"    // 成功
    StatusFailed     TaskStatus = "FAILED"     // 失败
    StatusCancelled  TaskStatus = "CANCELLED"  // 已取消
    StatusTimeout    TaskStatus = "TIMEOUT"    // 超时
)
```

| 状态 | 说明 | 可转换状态 |
|------|------|-----------|
| PENDING | 任务已提交，等待 Worker 处理 | PROCESSING, CANCELLED |
| PROCESSING | 任务正在执行中 | SUCCESS, FAILED, TIMEOUT, CANCELLED |
| SUCCESS | 任务执行成功 | - |
| FAILED | 任务执行失败 | PENDING (重试) |
| TIMEOUT | 任务执行超时 | PENDING (重试) |
| CANCELLED | 任务已取消 | - |

### 任务状态流转图

```
                    ┌─────────────┐
                    │   PENDING   │ ◄──────┐
                    └──────┬──────┘        │
                           │               │
                    用户取消│  调度器分配    │ 重试
                           │               │
                    ┌──────▼──────┐        │
                    │  CANCELLED  │        │
                    └─────────────┘        │
                           │               │
                           │               │
                    ┌──────▼──────┐        │
                    │ PROCESSING  │────────┤
                    └──────┬──────┘        │
                           │               │
              ┌────────────┼────────────┐  │
              │            │            │  │
         执行成功│       执行失败│      超时│  │
              │            │            │  │
        ┌─────▼────┐ ┌────▼─────┐ ┌───▼──▼───┐
        │ SUCCESS  │ │  FAILED  │ │ TIMEOUT  │
        └──────────┘ └──────────┘ └──────────┘
```

### 状态流转说明

#### 1. 正常流程
- **PENDING → PROCESSING**: 调度器将任务分配给空闲的 Worker
- **PROCESSING → SUCCESS**: Worker 成功执行任务
- **PROCESSING → FAILED**: Worker 执行任务失败
- **PROCESSING → TIMEOUT**: 任务执行超过预设时限

#### 2. 取消流程
- **PENDING → CANCELLED**: 用户取消队列中的任务
- **PROCESSING → CANCELLED**: 用户取消执行中的任务（Worker 检测到取消标记后终止）

#### 3. 重试流程
- **FAILED → PENDING**: 根据重试策略重新进入队列
- **TIMEOUT → PENDING**: 超时后根据重试策略重新进入队列

#### 4. 终态
- **SUCCESS**: 任务成功完成，不再变化
- **CANCELLED**: 任务已取消，不再变化
- **FAILED**: 达到最大重试次数后的最终失败状态
- **TIMEOUT**: 达到最大重试次数后的最终超时状态

### 任务存储说明

1. **MySQL**: 持久化存储任务完整信息
2. **Redis 队列**: 只存储 taskID，减少内存占用
3. **优先级队列**:
   - `queue:high`: 高优先级任务队列
   - `queue:normal`: 普通优先级任务队列
   - 调度器优先消费 high 队列

---

## 数据库表设计

### 1. task 表（任务主表）

```sql
CREATE TABLE `task` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务唯一标识(UUID)',
  `task_type` VARCHAR(64) NOT NULL COMMENT '任务类型',
  `priority` TINYINT NOT NULL DEFAULT 0 COMMENT '优先级: 0-普通, 1-高',
  `status` VARCHAR(20) NOT NULL DEFAULT 'PENDING' COMMENT '任务状态',
  `payload` JSON NOT NULL COMMENT '任务参数(JSON格式)',
  `result` JSON DEFAULT NULL COMMENT '任务执行结果',
  `error_msg` TEXT DEFAULT NULL COMMENT '错误信息',
  `retry_count` INT NOT NULL DEFAULT 0 COMMENT '已重试次数',
  `max_retry` INT NOT NULL DEFAULT 3 COMMENT '最大重试次数',
  `timeout` INT NOT NULL DEFAULT 300 COMMENT '超时时间(秒)',
  `worker_id` VARCHAR(64) DEFAULT NULL COMMENT '执行该任务的Worker ID',
  `scheduled_at` DATETIME NOT NULL COMMENT '计划执行时间',
  `started_at` DATETIME DEFAULT NULL COMMENT '开始执行时间',
  `completed_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_id` (`task_id`),
  KEY `idx_status_priority` (`status`, `priority`),
  KEY `idx_task_type` (`task_type`),
  KEY `idx_worker_id` (`worker_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_scheduled_at` (`scheduled_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务主表';
```

**字段说明**:
- `task_id`: 全局唯一的任务标识，使用 UUID 或雪花算法生成
- `task_type`: 任务类型，用于路由到不同的 Executor
- `priority`: 优先级，决定任务进入哪个队列
- `status`: 任务当前状态
- `payload`: 任务参数，JSON 格式存储，灵活扩展
- `result`: 任务执行结果，JSON 格式
- `retry_count`: 当前已重试次数，用于重试控制
- `worker_id`: 记录执行该任务的 Worker，便于追踪和调试

### 2. task_logs 表（任务日志表）

```sql
CREATE TABLE `task_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务ID',
  `log_type` VARCHAR(20) NOT NULL COMMENT '日志类型: STATE_CHANGE, RETRY, ERROR, INFO',
  `from_status` VARCHAR(20) DEFAULT NULL COMMENT '原状态',
  `to_status` VARCHAR(20) DEFAULT NULL COMMENT '新状态',
  `message` TEXT NOT NULL COMMENT '日志内容',
  `worker_id` VARCHAR(64) DEFAULT NULL COMMENT 'Worker ID',
  `retry_count` INT DEFAULT NULL COMMENT '重试次数',
  `error_detail` TEXT DEFAULT NULL COMMENT '错误详情',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_log_type` (`log_type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务日志表';
```

**日志类型**:
- `STATE_CHANGE`: 状态变更日志
- `RETRY`: 重试日志
- `ERROR`: 错误日志
- `INFO`: 信息日志

**用途**:
- 追踪任务完整的生命周期
- 问题排查和调试
- 审计和统计分析

### 3. task_config 表（任务配置表）

```sql
CREATE TABLE `task_config` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_type` VARCHAR(64) NOT NULL COMMENT '任务类型',
  `task_name` VARCHAR(128) NOT NULL COMMENT '任务名称',
  `description` TEXT DEFAULT NULL COMMENT '任务描述',
  `executor_type` VARCHAR(64) NOT NULL COMMENT '执行器类型: RPC, HTTP, LOCAL',
  `executor_config` JSON NOT NULL COMMENT '执行器配置(RPC地址、HTTP URL等)',
  `default_timeout` INT NOT NULL DEFAULT 300 COMMENT '默认超时时间(秒)',
  `default_max_retry` INT NOT NULL DEFAULT 3 COMMENT '默认最大重试次数',
  `retry_strategy` VARCHAR(20) NOT NULL DEFAULT 'FIXED' COMMENT '重试策略: FIXED, EXPONENTIAL',
  `retry_delay` INT NOT NULL DEFAULT 10 COMMENT '重试延迟(秒)',
  `backoff_rate` DECIMAL(3,2) NOT NULL DEFAULT 2.0 COMMENT '退避倍率(指数退避)',
  `max_concurrent` INT NOT NULL DEFAULT 10 COMMENT '最大并发数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_type` (`task_type`),
  KEY `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务配置表';
```

**字段说明**:
- `task_type`: 任务类型，与 task 表关联
- `executor_type`: 执行器类型，决定如何调用业务方
- `executor_config`: 执行器配置，JSON 格式，如 RPC 地址、HTTP URL 等
- `retry_strategy`: 重试策略
  - `FIXED`: 固定间隔重试
  - `EXPONENTIAL`: 指数退避重试
- `max_concurrent`: 该类型任务的最大并发数，用于限流

**配置示例**:
```json
{
  "task_type": "send_email",
  "executor_type": "RPC",
  "executor_config": {
    "service": "email-service",
    "method": "SendEmail",
    "address": "email-service:8080"
  },
  "default_timeout": 60,
  "default_max_retry": 3,
  "retry_strategy": "EXPONENTIAL",
  "retry_delay": 10,
  "backoff_rate": 2.0
}
```

### 4. worker_registry 表（Worker 注册表）

```sql
CREATE TABLE `worker_registry` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `worker_id` VARCHAR(64) NOT NULL COMMENT 'Worker唯一标识',
  `worker_name` VARCHAR(128) NOT NULL COMMENT 'Worker名称',
  `address` VARCHAR(256) NOT NULL COMMENT 'Worker地址',
  `status` VARCHAR(20) NOT NULL DEFAULT 'ONLINE' COMMENT '状态: ONLINE, OFFLINE',
  `capacity` INT NOT NULL DEFAULT 10 COMMENT '最大并发任务数',
  `current_load` INT NOT NULL DEFAULT 0 COMMENT '当前负载',
  `supported_types` JSON NOT NULL COMMENT '支持的任务类型列表',
  `last_heartbeat` DATETIME NOT NULL COMMENT '最后心跳时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_worker_id` (`worker_id`),
  KEY `idx_status` (`status`),
  KEY `idx_last_heartbeat` (`last_heartbeat`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Worker注册表';
```

**用途**:
- 持久化 Worker 信息（Redis 作为主要服务发现，MySQL 作为备份）
- Worker 历史记录和统计
- 故障分析和容量规划

---
 ```text
1. 用户调用CreateTask API
2. 验证参数，生成任务ID(UUID v4或雪花算法)
3. 插入任务记录到MySQL (状态为PENDING)
4. 根据优先级推送到Redis队列
5. 返回任务ID
6. 触发Worker消费队列
 ```

查询任务状态流程：
```text
1. 根据taskID查询Redis缓存
2. 缓存命中：直接返回结果
3. 缓存未命中：查询MySQL数据库
4. 查询到结果：缓存到Redis并返回
5. 未查询到：返回任务不存在
```
取消任务：
```text
1. 用户调用CancelTask API
2. 检查任务状态是否为PENDING或PROCESSING
3. 如果是PENDING：直接从队列删除，更新状态为CANCELLED
4. 如果是PROCESSING：标记为取消状态，Worker检测到后终止任务
5. 如果是其他状态：返回无法取消
```

### 核心模块设计
调度器：
1、基于redis分布式锁，实现一主多从
2、消费redis中的队列（优先级高、低）并调用worker
3、调用worker时，支持不同的策略选择不同的worker
4、其他的一些调度器行为，比如监控、限流等

worker：
1、自动服务发现与注册
2、心跳机制保持连接
3、优雅关闭处理
4、负载均衡策略
5、worker本身有最大的并发处理限制

Executor：（用于执行具体的任务）
1、根据不同的任务类型选择不同的Executor
2、灵活的业务handler处理，支持拓展

### 存储设计

系统支持多种存储实现，基于 DDD 的 Repository 模式，可灵活切换存储后端。

#### 支持的存储类型

1. **内存存储** (`infrastructure/memory/`)
   - 适用于：单元测试、本地开发、原型验证
   - 特点：无需外部依赖、启动快速、数据不持久化
   - 实现：使用 Go map + sync.RWMutex

2. **MySQL 存储** (`infrastructure/mysql/`)
   - 适用于：生产环境、需要持久化、分布式部署
   - 特点：数据持久化、支持事务、支持复杂查询
   - 实现：使用 database/sql + MySQL 驱动

#### 存储表设计

1. **task 表**：存放任务信息
2. **task_log 表**：存放任务的执行过程、状态变化
3. **task_config 表**：任务配置信息（类型、超时时间、重试策略等）
4. **worker 表**：Worker 注册信息（仅 MySQL）

详细的存储实现说明请参考 [STORAGE.md](./STORAGE.md)

监控与告警
    任务堆积监控
    Worker节点健康检查
    失败率统计
    处理时长统计
    自动告警机制
可扩展性设计
    支持插件化任务处理器
    支持自定义队列策略
    支持多种存储后端
    支持任务依赖关系