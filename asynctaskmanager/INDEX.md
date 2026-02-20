# 异步任务管理服务 - 文档索引

## 📚 文档导航

### 核心文档

1. **[readme.md](./readme.md)** - 项目概述、任务模型、数据库设计
   - 项目概述和核心功能
   - 任务状态定义和流转图
   - 完整的数据库表设计（task、task_logs、task_config、worker_registry）
   - 基础业务流程

2. **[COMPONENTS.md](./COMPONENTS.md)** - 组件职责与依赖关系 ⭐
   - Scheduler（调度器）详细设计
   - Worker（工作节点）详细设计
   - Executor（执行器）详细设计
   - 组件间的交互流程
   - 接口定义和配置示例

3. **[CORE_FLOWS.md](./CORE_FLOWS.md)** - 核心业务流程 ⭐
   - 创建任务流程（含代码实现）
   - 任务调度流程（含代码实现）
   - Worker 执行任务流程（含代码实现）
   - 查询任务流程
   - 取消任务流程
   - 任务重试流程

4. **[REDIS_DESIGN.md](./REDIS_DESIGN.md)** - Redis 数据结构设计 ⭐
   - 任务队列设计（优先级队列）
   - 分布式锁实现（Leader 选举）
   - Worker 注册表设计
   - 取消标记机制
   - 任务缓存策略
   - 延迟队列实现
   - Redis 高可用方案

5. **[EXTENSIBILITY.md](./EXTENSIBILITY.md)** - 可扩展性设计 ⭐
   - 自定义 Executor（含示例）
   - 自定义负载均衡策略
   - 自定义存储后端
   - 自定义监控指标
   - 事件系统
   - 插件系统
   - 配置驱动

6. **[STORAGE.md](./STORAGE.md)** - 存储实现说明 ⭐
   - 内存存储实现（用于测试和开发）
   - MySQL 存储实现（用于生产环境）
   - 数据库表结构详解
   - 存储切换方法
   - 性能优化建议
   - 扩展其他存储后端

### 辅助文档

7. **[QUICKSTART.md](./QUICKSTART.md)** - 快速开始指南
   - 5 分钟快速体验
   - API 文档
   - 代码示例（Go、Python）
   - 自定义 Executor 示例
   - 监控配置
   - 故障排查

8. **[SUMMARY.md](./SUMMARY.md)** - 设计文档总结
   - 快速索引
   - 关键设计决策
   - 性能指标
   - 部署建议
   - 监控指标
   - 故障处理
   - 最佳实践

9. **[PRIORITY_DESIGN.md](./PRIORITY_DESIGN.md)** - 优先级设计说明
   - TaskPriority 值对象设计
   - 优先级常量定义
   - 使用示例和最佳实践

10. **[PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md)** - 项目结构说明
    - DDD 分层架构
    - 目录结构详解
    - 模块职责说明

11. **[IMPLEMENTATION.md](./IMPLEMENTATION.md)** - 实现细节说明
    - 代码实现要点
    - 关键技术选型
    - 最佳实践

12. **[INDEX.md](./INDEX.md)** - 本文档

---

## 🎯 按需阅读指南

### 我是产品经理/项目经理

**推荐阅读顺序**:
1. [readme.md](./readme.md) - 了解项目概述和核心功能
2. [SUMMARY.md](./SUMMARY.md) - 了解关键设计决策和性能指标
3. [QUICKSTART.md](./QUICKSTART.md) - 了解 API 和使用方式

**关注重点**:
- 任务状态流转
- 核心功能
- 性能指标
- 部署建议

### 我是架构师

**推荐阅读顺序**:
1. [readme.md](./readme.md) - 了解整体设计
2. [COMPONENTS.md](./COMPONENTS.md) - 了解组件职责和依赖
3. [REDIS_DESIGN.md](./REDIS_DESIGN.md) - 了解 Redis 设计
4. [EXTENSIBILITY.md](./EXTENSIBILITY.md) - 了解扩展性设计
5. [SUMMARY.md](./SUMMARY.md) - 了解关键决策

**关注重点**:
- 架构设计
- 组件职责
- 依赖关系
- 高可用方案
- 扩展点设计

### 我是后端开发

**推荐阅读顺序**:
1. [QUICKSTART.md](./QUICKSTART.md) - 快速上手
2. [COMPONENTS.md](./COMPONENTS.md) - 了解组件设计
3. [CORE_FLOWS.md](./CORE_FLOWS.md) - 了解核心流程
4. [REDIS_DESIGN.md](./REDIS_DESIGN.md) - 了解 Redis 使用
5. [EXTENSIBILITY.md](./EXTENSIBILITY.md) - 了解如何扩展

**关注重点**:
- 接口定义
- 代码实现
- 数据库设计
- Redis 数据结构
- 扩展示例

### 我是运维工程师

**推荐阅读顺序**:
1. [readme.md](./readme.md) - 了解项目概述
2. [SUMMARY.md](./SUMMARY.md) - 了解部署和监控
3. [QUICKSTART.md](./QUICKSTART.md) - 了解部署步骤
4. [REDIS_DESIGN.md](./REDIS_DESIGN.md) - 了解 Redis 配置

**关注重点**:
- 部署架构
- 监控指标
- 故障处理
- 性能优化
- 高可用配置

### 我是测试工程师

**推荐阅读顺序**:
1. [readme.md](./readme.md) - 了解功能和流程
2. [CORE_FLOWS.md](./CORE_FLOWS.md) - 了解业务流程
3. [QUICKSTART.md](./QUICKSTART.md) - 了解 API
4. [SUMMARY.md](./SUMMARY.md) - 了解性能指标

**关注重点**:
- 任务状态流转
- 业务流程
- API 接口
- 异常场景
- 性能指标

---

## 🔍 快速查找

### 任务相关

- **任务状态**: [readme.md - 任务状态定义](./readme.md#任务状态定义)
- **状态流转**: [readme.md - 任务状态流转图](./readme.md#任务状态流转图)
- **创建任务**: [CORE_FLOWS.md - 创建任务流程](./CORE_FLOWS.md#1-创建任务流程)
- **查询任务**: [CORE_FLOWS.md - 查询任务流程](./CORE_FLOWS.md#4-查询任务流程)
- **取消任务**: [CORE_FLOWS.md - 取消任务流程](./CORE_FLOWS.md#5-取消任务流程)
- **任务重试**: [CORE_FLOWS.md - 任务重试流程](./CORE_FLOWS.md#6-任务重试流程)

### 组件相关

- **Scheduler 设计**: [COMPONENTS.md - Scheduler](./COMPONENTS.md#1-scheduler调度器)
- **Worker 设计**: [COMPONENTS.md - Worker](./COMPONENTS.md#2-worker工作节点)
- **Executor 设计**: [COMPONENTS.md - Executor](./COMPONENTS.md#3-executor执行器)
- **组件交互**: [COMPONENTS.md - 组件交互流程](./COMPONENTS.md#组件交互流程)

### 数据库相关

- **task 表**: [readme.md - task 表](./readme.md#1-task-表任务主表)
- **task_logs 表**: [readme.md - task_logs 表](./readme.md#2-task_logs-表任务日志表)
- **task_config 表**: [readme.md - task_config 表](./readme.md#3-task_config-表任务配置表)
- **worker_registry 表**: [readme.md - worker_registry 表](./readme.md#4-worker_registry-表worker-注册表)

### 存储相关

- **存储架构**: [STORAGE.md - 存储架构](./STORAGE.md#存储架构)
- **内存存储**: [STORAGE.md - 内存存储](./STORAGE.md#内存存储)
- **MySQL 存储**: [STORAGE.md - MySQL 存储](./STORAGE.md#mysql-存储)
- **表结构设计**: [STORAGE.md - 数据库表结构](./STORAGE.md#数据库表结构)
- **存储切换**: [STORAGE.md - 存储切换](./STORAGE.md#存储切换)
- **性能优化**: [STORAGE.md - 性能优化建议](./STORAGE.md#性能优化建议)
- **扩展其他存储**: [STORAGE.md - 扩展其他存储](./STORAGE.md#扩展其他存储)

### Redis 相关

- **任务队列**: [REDIS_DESIGN.md - 任务队列](./REDIS_DESIGN.md#1-任务队列)
- **Leader 选举**: [REDIS_DESIGN.md - 分布式锁](./REDIS_DESIGN.md#2-分布式锁leader-选举)
- **Worker 注册**: [REDIS_DESIGN.md - Worker 注册表](./REDIS_DESIGN.md#3-worker-注册表)
- **取消标记**: [REDIS_DESIGN.md - 取消标记](./REDIS_DESIGN.md#4-取消标记)
- **延迟队列**: [REDIS_DESIGN.md - 延迟队列](./REDIS_DESIGN.md#9-延迟队列重试)
- **高可用方案**: [REDIS_DESIGN.md - Redis 高可用方案](./REDIS_DESIGN.md#redis-高可用方案)

### 扩展相关

- **自定义 Executor**: [EXTENSIBILITY.md - 自定义 Executor](./EXTENSIBILITY.md#1-自定义-executor执行器)
- **自定义负载均衡**: [EXTENSIBILITY.md - 自定义负载均衡策略](./EXTENSIBILITY.md#2-自定义负载均衡策略)
- **自定义存储**: [EXTENSIBILITY.md - 自定义存储后端](./EXTENSIBILITY.md#3-自定义存储后端)
- **自定义监控**: [EXTENSIBILITY.md - 自定义监控指标](./EXTENSIBILITY.md#4-自定义监控指标)
- **事件系统**: [EXTENSIBILITY.md - 事件系统](./EXTENSIBILITY.md#5-事件系统)
- **插件系统**: [EXTENSIBILITY.md - 插件系统](./EXTENSIBILITY.md#6-插件系统)

---

## 📊 设计亮点

### 1. 高可用设计

- **Leader 选举**: 基于 Redis 分布式锁，自动故障转移
- **Worker 注册**: 心跳机制，自动检测和移除故障节点
- **任务恢复**: 超时任务自动重新调度
- **数据持久化**: MySQL + Redis AOF 双重保障

### 2. 高性能设计

- **优先级队列**: 高优先级任务优先处理
- **批量操作**: 减少网络开销
- **连接池**: 复用数据库和 Redis 连接
- **异步处理**: 非阻塞任务执行

### 3. 可扩展设计

- **插件化 Executor**: 支持自定义任务执行方式
- **策略模式**: 支持自定义负载均衡策略
- **事件驱动**: 通过事件解耦组件
- **配置驱动**: 通过配置控制行为

### 4. 可观测性

- **完整日志**: 记录任务完整生命周期
- **监控指标**: Prometheus 指标导出
- **链路追踪**: 支持分布式追踪
- **告警机制**: 及时发现和处理问题

---

## 🚀 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/yourusername/async-task-manager.git

# 2. 启动依赖服务
docker-compose up -d

# 3. 初始化数据库
mysql -h 127.0.0.1 -u root -proot task_manager < schema.sql

# 4. 启动服务
go run cmd/server/main.go

# 5. 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"task_type":"send_email","priority":1,"payload":{"to":"user@example.com"}}'
```

详见：[QUICKSTART.md](./QUICKSTART.md)

---

## 📈 性能指标

| 指标 | 目标值 |
|------|--------|
| 任务创建 TPS | 1000+ |
| 任务调度延迟 | < 100ms |
| 任务执行并发 | 1000+ |
| Leader 选举时间 | < 1s |
| 队列消费速率 | 10000+ tasks/s |

详见：[SUMMARY.md - 性能指标](./SUMMARY.md#性能指标)

---

## 🤝 贡献指南

欢迎贡献代码、文档或提出建议！

1. Fork 项目
2. 创建特性分支
3. 提交代码
4. 发起 Pull Request

---

## 📝 更新日志

- **2024-02-16**: 完成完整设计文档
  - 组件职责与依赖关系
  - 核心业务流程
  - Redis 数据结构设计
  - 可扩展性设计
  - 快速开始指南

---

## 📧 联系方式

- 项目地址: https://github.com/yourusername/async-task-manager
- 问题反馈: 提交 Issue
- 技术讨论: 欢迎 PR

---
