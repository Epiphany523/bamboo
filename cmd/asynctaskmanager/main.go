package main

import (
	"flag"
	"log"

	"bamboo/cmd/asynctaskmanager/server"
)

func main() {
	// 解析命令行参数
	serverID := flag.String("id", "server-1", "Server ID")
	grpcPort := flag.Int("grpc-port", 9090, "gRPC port")
	workerPort := flag.Int("worker-port", 8080, "Worker port")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	mysqlDSN := flag.String("mysql", "root:a123456@tcp(localhost:3306)/asynctask?charset=utf8mb4&parseTime=True&loc=Local", "MySQL DSN")
	flag.Parse()

	log.Printf("Starting Async Task Manager Server")
	log.Printf("  Server ID: %s", *serverID)
	log.Printf("  gRPC Port: %d", *grpcPort)
	log.Printf("  Worker Port: %d", *workerPort)
	log.Printf("  Redis: %s", *redisAddr)

	// 创建服务器配置
	cfg := &server.ServerConfig{
		ServerID:   *serverID,
		GRPCPort:   *grpcPort,
		RedisAddr:  *redisAddr,
		MysqlDSN:   server.MysqlDSN(*mysqlDSN),
		WorkerPort: *workerPort,
	}

	// 运行服务器
	if err := server.Run(cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
