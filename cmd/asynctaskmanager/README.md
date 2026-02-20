# Async Task Manager - gRPC Service

基于 gRPC 的异步任务管理服务，支持分布式部署和高可用。

## 功能特性

- gRPC API 接口
- 任务创建、查询、取消
- 任务日志查询
- 优先级队列（Normal/High）
- 分布式调度（基于 Redis）
- 高可用 Worker 集群
- MySQL 持久化存储

## 快速开始

### 1. 安装依赖

```bash
# 安装 protobuf 编译器
brew install protobuf  # macOS
# 或
apt-get install protobuf-compiler  # Ubuntu

# 安装 Go protobuf 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 2. 生成 protobuf 代码

```bash
cd cmd/asynctaskmanager
make proto
```

### 3. 启动 Redis

```bash
docker run -d -p 6379:6379 redis:latest
```

### 4. 启动 MySQL 并初始化数据库

```bash
# 启动 MySQL
docker run -d -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=asynctask \
  mysql:8.0

# 等待 MySQL 启动
sleep 10

# 初始化数据库表（使用 asynctaskmanager/infrastructure/mysql 中的 SQL）
```

### 5. 启动服务端

单实例模式：
```bash
make run-server
```

集群模式（3个实例）：
```bash
make run-cluster
```

或手动启动多个实例：
```bash
# 终端 1
go run main.go -id=server-1 -grpc-port=9091 -worker-port=8081

# 终端 2
go run main.go -id=server-2 -grpc-port=9092 -worker-port=8082

# 终端 3
go run main.go -id=server-3 -grpc-port=9093 -worker-port=8083
```

### 6. 运行客户端测试

```bash
# 连接到 server-1
make run-client

# 或连接到其他服务器
go run client/main.go -server=localhost:9092
```

## API 接口

### CreateTask - 创建任务

```protobuf
rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);

message CreateTaskRequest {
  string task_type = 1;
  int32 priority = 2;  // 0=Normal, 1=High
  map<string, string> payload = 3;
}
```

### GetTask - 查询任务

```protobuf
rpc GetTask(GetTaskRequest) returns (GetTaskResponse);

message GetTaskRequest {
  string task_id = 1;
}
```

### CancelTask - 取消任务

```protobuf
rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);

message CancelTaskRequest {
  string task_id = 1;
}
```

### GetTaskLogs - 获取任务日志

```protobuf
rpc GetTaskLogs(GetTaskLogsRequest) returns (GetTaskLogsResponse);

message GetTaskLogsRequest {
  string task_id = 1;
}
```

## 客户端使用示例

```go
package main

import (
    "context"
    "log"
    
    "bamboo/cmd/asynctaskmanager/client"
)

func main() {
    // 创建客户端
    grpcClient, err := client.NewGRPCClient("localhost:9090")
    if err != nil {
        log.Fatal(err)
    }
    defer grpcClient.Close()
    
    ctx := context.Background()
    
    // 创建高优先级任务
    task, err := grpcClient.CreateTask(ctx, "example_task", 1, map[string]interface{}{
        "message": "Hello World",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Task created: %s", task.TaskId)
    
    // 等待任务完成
    completedTask, err := grpcClient.WaitForTask(ctx, task.TaskId, 30*time.Second)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Task completed: %s", completedTask.Status)
}
```

## 配置说明

### 命令行参数

- `-id`: 服务器 ID（默认：server-1）
- `-grpc-port`: gRPC 端口（默认：9090）
- `-worker-port`: Worker 端口（默认：8080）
- `-redis`: Redis 地址（默认：localhost:6379）
- `-mysql`: MySQL DSN（默认：root:password@tcp(localhost:3306)/asynctask）

### 环境变量

可以通过环境变量覆盖默认配置：

```bash
export REDIS_ADDR=localhost:6379
export MYSQL_DSN=root:password@tcp(localhost:3306)/asynctask
```

## 架构说明

### 组件职责

1. **gRPC Server**: 提供 API 接口，处理客户端请求
2. **Scheduler**: 调度任务到 Worker，支持主从选举
3. **Worker**: 执行任务，支持多种执行器
4. **Redis**: 任务队列和分布式协调
5. **MySQL**: 任务和日志持久化存储

### 高可用设计

- 多个服务实例共享 Redis 和 MySQL
- Scheduler 通过 Redis 实现主从选举
- Worker 自动注册和心跳检测
- 任务失败自动重试

## 测试

客户端测试程序包含以下测试用例：

1. 创建普通优先级任务
2. 创建高优先级任务
3. 批量创建任务
4. 查询任务状态
5. 创建并取消任务
6. 查询任务日志
7. 等待任务完成
8. 统计信息

## 故障排查

### gRPC 连接失败

检查服务端是否启动：
```bash
netstat -an | grep 9090
```

### Redis 连接失败

检查 Redis 是否运行：
```bash
redis-cli ping
```

### MySQL 连接失败

检查 MySQL 是否运行：
```bash
mysql -h localhost -u root -p -e "SELECT 1"
```

### 任务不执行

1. 检查 Worker 是否在线
2. 检查 Scheduler 是否选举成功
3. 查看服务端日志

## 开发

### 修改 protobuf 定义

1. 编辑 `proto/task_service.proto`
2. 运行 `make proto` 重新生成代码
3. 更新服务端和客户端实现

### 添加新的任务类型

在 `server/main.go` 中注册新的任务处理器：

```go
localExecutor.RegisterHandler("new_task_type", func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
    // 实现任务逻辑
    return result, nil
})
```

## License

MIT
