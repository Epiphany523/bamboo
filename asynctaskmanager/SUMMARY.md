# 设计文档总结

## 文档导航

本项目包含以下设计文档：

1. **[readme.md](./readme.md)** - 项目概述和基础设计
2. **[COMPONENTS.md](./COMPONENTS.md)** - 组件职责与依赖关系
3. **[CORE_FLOWS.md](./CORE_FLOWS.md)** - 核心业务流程
4. **[REDIS_DESIGN.md](./REDIS_DESIGN.md)** - Redis 数据结构设计
5. **[EXTENSIBILITY.md](./EXTENSIBILITY.md)** - 可扩展性设计
6. **[SUMMARY.md](./SUMMARY.md)** - 本文档

---

## 快速索引

### 1. 架构设计

**组件关系**:
```
API Gateway → Scheduler (Leader) → Worker → Executor → 业务方服务
                ↓                    ↓         ↓
              Redis              Redis     MySQL
                ↓                    ↓
              MySQL              task_logs
```

**核心组件**:
- **Scheduler**: 调度器，负责 Leader 选举和任务分配
- **Worker**: 工作节点，负责任务执行
- **Executor**: 执行器，负责调用业务方接口

详见：[COMPONENTS.md](./COMPONENTS.md)

### 2. 任务状态流转

```
PENDING → PROCESSING → SUCCESS
                    → FAILED → PENDING (重试)
                    → TIMEOUT → PENDING (重试)
                    → CANCELLED
```

详见：[readme.md - 任务状态流转图](./readme.md#任务状态流转图)

### 3. 数据库设计

**核心表**:
- `task`: 任务主表
- `task_logs`: 任务日志表
- `task_config`: 任务配置表
- `worker_registry`: Worker 注册表

详见：[readme.md - 数据库表设计](./readme.md#数据库表设计)

### 4. Redis 数据结构

**核心结构**:
- 任务队列：`queue:high`, `queue:normal`
- Worker 队列：`worker:{worker_id}:queue`
- Leader 锁：`scheduler:leader`
- Worker 注册：`worker:{worker_id}`
- 取消标记：`task:cancel:{task_id}`

详见：[REDIS_DESIGN.md](./REDIS_DESIGN.md)

### 5. 核心流程

**主要流程**:
1. 创建任务流程
2. 任务调度流程
3. Worker 执行任务流程
4. 查询任务流程
5. 取消任务流程
6. 任务重试流程

详见：[CORE_FLOWS.md](./CORE_FLOWS.md)

### 6. 扩展点

**可扩展组件**:
- 自定义 Executor（执行器）
- 自定义 LoadBalancer（负载均衡策略）
- 自定义存储后端
- 自定义监控指标
- 插件系统

详见：[EXTENSIBILITY.md](./EXTENSIBILITY.md)

---

## 关键设计决策

### 1. 为什么使用 Redis？

**优势**:
- 高性能：内存存储，低延迟
- 丰富的数据结构：List、Hash、Set、Sorted Set
- 分布式锁：支持 Leader 选举
- Pub/Sub：支持事件通知
- 持久化：AOF/RDB 保证数据安全

**使用场景**:
- 任务队列（List）
- Worker 注册（Hash）
- Leader 选举（分布式锁）
- 取消标记（String）
- 延迟队列（Sorted Set）

### 2. 为什么分离 Scheduler 和 Worker？

**职责分离**:
- Scheduler：专注于任务调度和分配
- Worker：专注于任务执行

**优势**:
- 独立扩展：可以独立增加 Scheduler 或 Worker 实例
- 故障隔离：Scheduler 故障不影响 Worker 执行
- 灵活部署：可以部署在不同的机器上

### 3. 为什么使用 Leader 选举？

**避免重复调度**:
- 多个 Scheduler 实例只有一个 Leader
- Leader 负责从队列拉取任务
- 避免同一任务被多次分配

**高可用**:
- Leader 故障后自动重新选举
- 新 Leader 接管调度工作
- 不影响正在执行的任务

### 4. 为什么使用优先级队列？

**业务需求**:
- 紧急任务需要优先处理
- 普通任务可以延后处理

**实现方式**:
- 高优先级队列：`queue:high`
- 普通优先级队列：`queue:normal`
- Scheduler 优先消费高优先级队列

### 5. 为什么任务详情存储在 MySQL？

**持久化需求**:
- 任务数据需要长期保存
- 支持复杂查询和统计
- 事务保证数据一致性

**Redis 只存 ID**:
- 减少内存占用
- 队列操作更快
- 任务详情按需加载

---

## 性能指标

### 预期性能

| 指标 | 目标值 |
|------|--------|
| 任务创建 TPS | 1000+ |
| 任务调度延迟 | < 100ms |
| 任务执行并发 | 1000+ |
| Leader 选举时间 | < 1s |
| Worker 心跳间隔 | 10s |
| 队列消费速率 | 10000+ tasks/s |

### 容量规划

**单 Scheduler 实例**:
- 支持 10000+ 任务/秒的调度
- 管理 100+ Worker 节点

**单 Worker 实例**:
- 默认并发：10 个任务
- 可配置：1-100 个任务

**Redis**:
- 队列长度：建议 < 100000
- 内存使用：取决于队列长度
- 连接数：Scheduler + Worker 数量

**MySQL**:
- 任务表：支持千万级数据
- 日志表：建议定期归档
- 连接池：20-50 连接

---

## 部署建议

### 最小部署（开发环境）

```
1 Scheduler + 1 Worker + 1 Redis + 1 MySQL
```

### 推荐部署（生产环境）

```
3 Scheduler + 10 Worker + Redis Sentinel (3节点) + MySQL 主从
```

### 高可用部署

```
5 Scheduler + 50 Worker + Redis Cluster (6节点) + MySQL 集群
```

---

## 监控指标

### Scheduler 指标

- Leader 状态
- 任务调度速率
- 队列长度
- Worker 在线数量
- 调度失败率

### Worker 指标

- 当前负载
- 任务执行成功率
- 任务执行耗时（P50/P95/P99）
- 心跳延迟

### 任务指标

- 任务总数（按状态）
- 任务创建速率
- 任务完成速率
- 任务失败率
- 任务重试率
- 任务平均执行时间

### 系统指标

- Redis 连接数
- Redis 内存使用
- MySQL 连接数
- MySQL 慢查询
- CPU 使用率
- 内存使用率

---

## 故障处理

### Scheduler 故障

**现象**: Leader 不再续约

**处理**:
1. 其他 Scheduler 自动竞争成为 Leader
2. 新 Leader 接管任务调度
3. 正在执行的任务不受影响

**恢复时间**: < 10s

### Worker 故障

**现象**: 心跳超时

**处理**:
1. Scheduler 将 Worker 标记为离线
2. 正在执行的任务超时后重新调度
3. 队列中的任务分配给其他 Worker

**恢复时间**: < 30s

### Redis 故障

**现象**: 连接失败

**处理**:
1. 使用 Redis Sentinel 自动故障转移
2. 应用自动重连到新的 Master
3. 队列数据通过 AOF 恢复

**恢复时间**: < 30s

### MySQL 故障

**现象**: 连接失败

**处理**:
1. 使用主从切换
2. 应用重连到新的 Master
3. 任务数据不丢失

**恢复时间**: < 60s

---

## 最佳实践

### 1. 任务设计

- 任务应该是幂等的
- 任务执行时间应该可控
- 避免长时间运行的任务
- 合理设置超时时间

### 2. 重试策略

- 区分可重试和不可重试错误
- 使用指数退避避免雪崩
- 设置合理的最大重试次数
- 记录详细的错误信息

### 3. 监控告警

- 监控队列堆积
- 监控任务失败率
- 监控 Worker 健康状态
- 设置合理的告警阈值

### 4. 性能优化

- 使用连接池
- 批量操作
- 异步处理
- 合理的索引

### 5. 安全考虑

- 任务参数验证
- 敏感信息加密
- 访问控制
- 审计日志

---

## 后续优化方向

### 短期（1-2 月）

- [ ] 支持任务优先级动态调整
- [ ] 支持任务依赖关系
- [ ] 完善监控和告警
- [ ] 性能压测和优化

### 中期（3-6 月）

- [ ] Web 管理界面
- [ ] 任务执行可视化
- [ ] 支持定时任务（Cron）
- [ ] 支持任务编排（DAG）

### 长期（6-12 月）

- [ ] 支持分片任务
- [ ] 支持流式任务
- [ ] 多租户隔离
- [ ] 国际化支持

---

## 参考资料

- [Redis 官方文档](https://redis.io/documentation)
- [MySQL 官方文档](https://dev.mysql.com/doc/)
- [分布式系统设计模式](https://martinfowler.com/articles/patterns-of-distributed-systems/)
- [任务队列设计](https://www.cloudamqp.com/blog/part1-rabbitmq-best-practice.html)
