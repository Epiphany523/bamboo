# 快速开始指南

## 前置要求

1. Go 1.21+
2. Redis
3. MySQL 8.0+
4. protoc (Protocol Buffers 编译器)

## 安装步骤

### 1. 安装 protobuf 编译器和插件

```bash
# macOS
brew install protobuf

# Ubuntu/Debian
apt-get install protobuf-compiler

# 安装 Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 2. 启动 Redis

```bash
# 使用 Docker
docker run -d --name redis -p 6379:6379 redis:latest

# 或使用本地安装
redis-server
```

### 3. 启动 MySQL 并初始化数据库

```bash
# 使用 Docker
docker run -d --name mysql \
  -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=asynctask \
  mysql:8.0

# 等待 MySQL 启动
sleep 15

# 初始化数据库表
mysql -h localhost -u root -ppassword asynctask < scripts/init_db.sql
```

### 4. 生成 protobuf 代码

```bash
cd cmd/asynctaskmanager
make proto
```

### 5. 安装 Go 依赖

```bash
cd ../..  # 回到项目根目录
go mod tidy
```

## 运行服务

### 单实例模式

```bash
cd cmd/asynctaskmanager

# 启动服务器
go run main.go \
  -id=server-1 \
  -grpc-port=9090 \
  -worker-port=8080 \
  -redis=localhost:6379 \
  -mysql="root:password@tcp(localhost:3306)/asynctask?charset=utf8mb4&parseTime=True&loc=Local"
```

### 集群模式（3个实例）

在不同的终端窗口中运行：

```bash
# 终端 1 - Server 1
cd cmd/asynctaskmanager
go run main.go -id=server-1 -grpc-port=9091 -worker-port=8081

# 终端 2 - Server 2
cd cmd/asynctaskmanager
go run main.go -id=server-2 -grpc-port=9092 -worker-port=8082

# 终端 3 - Server 3
cd cmd/asynctaskmanager
go run main.go -id=server-3 -grpc-port=9093 -worker-port=8083
```

或使用脚本：

```bash
cd cmd/asynctaskmanager
chmod +x scripts/start_cluster.sh
./scripts/start_cluster.sh
```

## 测试客户端

### 运行测试客户端

```bash
cd cmd/asynctaskmanager

# 连接到单实例
go run client/main.go -server=localhost:9090

# 连接到集群中的某个实例
go run client/main.go -server=localhost:9091
```

### 客户端测试内容

测试客户端会自动执行以下测试：

1. ✓ 创建普通优先级任务
2. ✓ 创建高优先级任务
3. ✓ 批量创建任务（5个）
4. ✓ 查询任务状态
5. ✓ 创建并取消任务
6. ✓ 查询任务日志
7. ✓ 等待任务完成
8. ✓ 统计信息

## 使用 gRPC 客户端

### Go 客户端示例

```go
package main

import (
    "context"
    "log"
    "time"
    
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
    
    // 创建任务
    task, err := grpcClient.CreateTask(ctx, "example_task", 1, map[string]interface{}{
        "message": "Hello World",
        "number":  42,
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
    
    log.Printf("Task status: %s", completedTask.Status)
    log.Printf("Task result: %v", completedTask.Result)
}
```

## MySQL DSN 配置说明

DSN (Data Source Name) 格式：

```
user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

示例：

```bash
# 本地 MySQL
-mysql="root:password@tcp(localhost:3306)/asynctask?charset=utf8mb4&parseTime=True&loc=Local"

# 远程 MySQL
-mysql="admin:secret@tcp(192.168.1.100:3306)/production?charset=utf8mb4&parseTime=True&loc=Local"

# 无密码
-mysql="root:@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
```

DSN 解析会自动提取：
- 用户名 (user)
- 密码 (password)
- 主机 (host)
- 端口 (port)
- 数据库名 (database)

## 验证服务运行

### 检查服务端口

```bash
# 检查 gRPC 端口
netstat -an | grep 9090

# 检查 Redis 连接
redis-cli ping

# 检查 MySQL 连接
mysql -h localhost -u root -ppassword -e "USE asynctask; SHOW TABLES;"
```

### 查看日志

服务端日志会显示：

```
Starting Async Task Manager Server
  Server ID: server-1
  gRPC Port: 9090
  Worker Port: 8080
  Redis: localhost:6379
[server-1] Starting server...
[server-1] Server started successfully (gRPC port: 9090)
gRPC server listening on port 9090
```

## 常见问题

### 1. protoc 命令找不到

确保已安装 protobuf 编译器：

```bash
protoc --version
```

### 2. Redis 连接失败

检查 Redis 是否运行：

```bash
redis-cli ping
# 应该返回 PONG
```

### 3. MySQL 连接失败

检查 MySQL 是否运行并且密码正确：

```bash
mysql -h localhost -u root -ppassword -e "SELECT 1"
```

### 4. gRPC 连接失败

确保服务端已启动并监听正确的端口：

```bash
netstat -an | grep 9090
```

### 5. 任务不执行

1. 检查 Worker 是否在线
2. 检查 Scheduler 是否选举成功（查看日志）
3. 检查 Redis 队列是否有任务

```bash
redis-cli
> LLEN task:queue:normal
> LLEN task:queue:high
```

## 停止服务

### 单实例模式

按 `Ctrl+C` 停止服务

### 集群模式

```bash
cd cmd/asynctaskmanager
./scripts/stop_cluster.sh
```

或手动停止每个进程：

```bash
# 查找进程
ps aux | grep "go run main.go"

# 停止进程
kill <PID>
```

## 下一步

- 查看 [README.md](README.md) 了解更多功能
- 查看 [proto/task_service.proto](proto/task_service.proto) 了解 API 定义
- 查看 [client/grpc_client.go](client/grpc_client.go) 了解客户端实现
- 查看 [server/grpc_server.go](server/grpc_server.go) 了解服务端实现
