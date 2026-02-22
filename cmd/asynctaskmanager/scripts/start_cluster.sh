#!/bin/bash
go run main.go -id=server-1 -grpc-port=9091 -worker-port=8081

go run main.go -id=server-2 -grpc-port=9092 -worker-port=8082

go run main.go -id=server-3 -grpc-port=9093 -worker-port
