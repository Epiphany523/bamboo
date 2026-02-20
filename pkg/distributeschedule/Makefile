.PHONY: help test build run clean

help: ## 显示帮助信息
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

test: ## 运行测试
	go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

build: ## 编译示例程序
	go build -o bin/scheduler ./example/main.go

run: ## 运行示例程序
	go run ./example/main.go

clean: ## 清理编译产物
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## 下载依赖
	go mod download
	go mod tidy

fmt: ## 格式化代码
	go fmt ./...

lint: ## 代码检查
	golangci-lint run

docker-redis: ## 启动 Redis 容器
	docker run -d --name redis-scheduler -p 6379:6379 redis:latest

docker-redis-stop: ## 停止 Redis 容器
	docker stop redis-scheduler
	docker rm redis-scheduler
