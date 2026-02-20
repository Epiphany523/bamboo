#!/bin/bash

# 启动 3 个服务实例的集群

echo "Starting Async Task Manager Cluster..."

# 检查 Redis 是否运行
if ! redis-cli ping > /dev/null 2>&1; then
    echo "Error: Redis is not running. Please start Redis first."
    echo "  docker run -d -p 6379:6379 redis:latest"
    exit 1
fi

# 检查 MySQL 是否运行
#if ! mysql -h localhost -u root -p -e "SELECT 1" > /dev/null 2>&1; then
#    echo "Error: MySQL is not running or credentials are incorrect."
#    echo "  docker run -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=asynctask mysql:8.0"
#    exit 1
#fi

# 初始化数据库
#echo "Initializing database..."
#mysql -h localhost -u root -ppassword < scripts/init_db.sql

# 启动服务实例
echo "Starting server instances..."


go run main.go -id=server-1 -grpc-port=9091 -worker-port=8081 > logs/server-1.log 2>&1 &
SERVER1_PID=$!
echo "  Server 1 started (PID: $SERVER1_PID, gRPC: 9091)"

go run main.go -id=server-2 -grpc-port=9092 -worker-port=8082 > logs/server-2.log 2>&1 &
SERVER2_PID=$!
echo "  Server 2 started (PID: $SERVER2_PID, gRPC: 9092)"

go run main.go -id=server-3 -grpc-port=9093 -worker-port=8083 > logs/server-3.log 2>&1 &
SERVER3_PID=$!
echo "  Server 3 started (PID: $SERVER3_PID, gRPC: 9093)"

# 保存 PID
echo "$SERVER1_PID" > logs/server-1.pid
echo "$SERVER2_PID" > logs/server-2.pid
echo "$SERVER3_PID" > logs/server-3.pid

echo ""
echo "Cluster started successfully!"
echo ""
echo "Server endpoints:"
echo "  Server 1: localhost:9091"
echo "  Server 2: localhost:9092"
echo "  Server 3: localhost:9093"
echo ""
echo "To test the cluster, run:"
echo "  go run client/main.go -server=localhost:9091"
echo ""
echo "To stop the cluster, run:"
echo "  ./scripts/stop_cluster.sh"
