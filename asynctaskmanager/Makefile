.PHONY: help build run test clean deps

help: ## 显示帮助信息
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## 下载依赖
	go mod download
	go mod tidy

build: ## 编译程序
	go build -o bin/server cmd/server/main.go

run: ## 运行程序
	go run cmd/server/main.go

test: ## 运行测试
	go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## 清理编译产物
	rm -rf bin/
	rm -f coverage.out coverage.html

fmt: ## 格式化代码
	go fmt ./...

lint: ## 代码检查
	golangci-lint run

docker-redis: ## 启动 Redis 容器
	docker run -d --name redis-task-manager -p 6379:6379 redis:latest

docker-redis-stop: ## 停止 Redis 容器
	docker stop redis-task-manager
	docker rm redis-task-manager

docker-mysql: ## 启动 MySQL 容器
	docker run -d --name mysql-task-manager \
		-e MYSQL_ROOT_PASSWORD=root \
		-e MYSQL_DATABASE=task_manager \
		-p 3306:3306 \
		mysql:8.0

docker-mysql-stop: ## 停止 MySQL 容器
	docker stop mysql-task-manager
	docker rm mysql-task-manager
