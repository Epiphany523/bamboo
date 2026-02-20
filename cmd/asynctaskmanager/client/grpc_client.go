package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "bamboo/cmd/asynctaskmanager/proto"
)

// GRPCClient gRPC 客户端
type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.TaskServiceClient
}

// NewGRPCClient 创建 gRPC 客户端
func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := pb.NewTaskServiceClient(conn)

	return &GRPCClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close 关闭连接
func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

// CreateTask 创建任务
func (c *GRPCClient) CreateTask(ctx context.Context, taskType string, priority int32, payload map[string]interface{}) (*pb.Task, error) {
	// 转换 payload
	pbPayload := make(map[string]string)
	for k, v := range payload {
		if str, ok := v.(string); ok {
			pbPayload[k] = str
		} else {
			// 将非字符串类型转为 JSON
			if jsonBytes, err := json.Marshal(v); err == nil {
				pbPayload[k] = string(jsonBytes)
			}
		}
	}

	req := &pb.CreateTaskRequest{
		TaskType: taskType,
		Priority: priority,
		Payload:  pbPayload,
	}

	resp, err := c.client.CreateTask(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Task, nil
}

// GetTask 查询任务
func (c *GRPCClient) GetTask(ctx context.Context, taskID string) (*pb.Task, error) {
	req := &pb.GetTaskRequest{
		TaskId: taskID,
	}

	resp, err := c.client.GetTask(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Task, nil
}

// CancelTask 取消任务
func (c *GRPCClient) CancelTask(ctx context.Context, taskID string) (bool, string, error) {
	req := &pb.CancelTaskRequest{
		TaskId: taskID,
	}

	resp, err := c.client.CancelTask(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

// GetTaskLogs 获取任务日志
func (c *GRPCClient) GetTaskLogs(ctx context.Context, taskID string) ([]*pb.TaskLog, error) {
	req := &pb.GetTaskLogsRequest{
		TaskId: taskID,
	}

	resp, err := c.client.GetTaskLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Logs, nil
}

// ListTasks 列出任务
func (c *GRPCClient) ListTasks(ctx context.Context, status string, priority int32, page, pageSize int32) ([]*pb.Task, int32, error) {
	req := &pb.ListTasksRequest{
		Status:   status,
		Priority: priority,
		Page:     page,
		PageSize: pageSize,
	}

	resp, err := c.client.ListTasks(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	return resp.Tasks, resp.Total, nil
}

// WaitForTask 等待任务完成
func (c *GRPCClient) WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*pb.Task, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		task, err := c.GetTask(ctx, taskID)
		if err != nil {
			return nil, err
		}

		// 检查任务是否完成
		if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
			return task, nil
		}

		// 等待一段时间后重试
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("task %s timeout after %v", taskID, timeout)
}
