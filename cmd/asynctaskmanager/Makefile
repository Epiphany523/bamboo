.PHONY: proto build run-server run-client clean

# 生成 protobuf 代码
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/task_service.proto

# 编译服务端和客户端
build: proto
	@echo "Building server..."
	go build -o bin/server main.go
	@echo "Building client..."
	go build -o bin/client client/main.go

# 运行服务端（单实例）
run-server:
	go run main.go -id=server-1 -grpc-port=9090 -worker-port=8080

# 运行服务端集群（3个实例）
run-cluster:
	@echo "Starting 3-server cluster..."
	@go run main.go -id=server-1 -grpc-port=9091 -worker-port=8081 & \
	go run main.go -id=server-2 -grpc-port=9092 -worker-port=8082 & \
	go run main.go -id=server-3 -grpc-port=9093 -worker-port=8083 & \
	wait

# 运行客户端
run-client:
	go run client/main.go -server=localhost:9090

# 清理
clean:
	rm -rf bin/
	rm -f proto/*.pb.go
