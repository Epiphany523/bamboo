package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"bamboo/asynctaskmanager/application"
	"bamboo/asynctaskmanager/domain/model"
	pb "bamboo/cmd/asynctaskmanager/proto"
)

// GRPCServer gRPC 服务器
type GRPCServer struct {
	pb.UnimplementedTaskServiceServer
	taskService *application.TaskService
	grpcServer  *grpc.Server
	port        int
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(taskService *application.TaskService, port int) *GRPCServer {
	return &GRPCServer{
		taskService: taskService,
		port:        port,
	}
}

// Start 启动 gRPC 服务器
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterTaskServiceServer(s.grpcServer, s)

	log.Printf("gRPC server listening on port %d", s.port)
	return s.grpcServer.Serve(lis)
}

// Stop 停止 gRPC 服务器
func (s *GRPCServer) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// CreateTask 创建任务
func (s *GRPCServer) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	// 转换优先级
	priority := model.PriorityNormal
	if req.Priority == 1 {
		priority = model.PriorityHigh
	}

	// 转换 payload
	payload := make(map[string]interface{})
	for k, v := range req.Payload {
		payload[k] = v
	}

	// 创建任务
	task, err := s.taskService.CreateTask(ctx, req.TaskType, priority, payload)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTaskResponse{
		Task: convertTaskToProto(task),
	}, nil
}

// GetTask 查询任务
func (s *GRPCServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	task, err := s.taskService.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	return &pb.GetTaskResponse{
		Task: convertTaskToProto(task),
	}, nil
}

// CancelTask 取消任务
func (s *GRPCServer) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.CancelTaskResponse, error) {
	err := s.taskService.CancelTask(ctx, req.TaskId)
	if err != nil {
		return &pb.CancelTaskResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.CancelTaskResponse{
		Success: true,
		Message: "Task cancelled successfully",
	}, nil
}

// GetTaskLogs 获取任务日志
func (s *GRPCServer) GetTaskLogs(ctx context.Context, req *pb.GetTaskLogsRequest) (*pb.GetTaskLogsResponse, error) {
	logs, err := s.taskService.GetTaskLogs(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	pbLogs := make([]*pb.TaskLog, len(logs))
	for i, log := range logs {
		pbLogs[i] = convertTaskLogToProto(log)
	}

	return &pb.GetTaskLogsResponse{
		Logs: pbLogs,
	}, nil
}

// ListTasks 列出任务
func (s *GRPCServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	// 这里简化实现，实际应该在 TaskService 中添加 ListTasks 方法
	return &pb.ListTasksResponse{
		Tasks: []*pb.Task{},
		Total: 0,
	}, nil
}

// convertTaskToProto 转换任务为 protobuf 格式
func convertTaskToProto(task *model.Task) *pb.Task {
	pbTask := &pb.Task{
		TaskId:       task.TaskID,
		TaskType:     task.TaskType,
		Status:       string(task.Status),
		Priority:     int32(task.Priority.Value()),
		WorkerId:     task.WorkerID,
		RetryCount:   int32(task.RetryCount),
		MaxRetry:     int32(task.MaxRetry),
		ErrorMessage: task.ErrorMsg,
		CreatedAt:    timestamppb.New(task.CreatedAt),
	}

	// 转换 payload
	if task.Payload != nil {
		pbTask.Payload = make(map[string]string)
		for k, v := range task.Payload {
			if str, ok := v.(string); ok {
				pbTask.Payload[k] = str
			} else {
				// 将非字符串类型转为 JSON
				if jsonBytes, err := json.Marshal(v); err == nil {
					pbTask.Payload[k] = string(jsonBytes)
				}
			}
		}
	}

	// 转换 result
	if task.Result != nil {
		pbTask.Result = make(map[string]string)
		for k, v := range task.Result {
			if str, ok := v.(string); ok {
				pbTask.Result[k] = str
			} else {
				if jsonBytes, err := json.Marshal(v); err == nil {
					pbTask.Result[k] = string(jsonBytes)
				}
			}
		}
	}

	if task.StartedAt != nil {
		pbTask.StartedAt = timestamppb.New(*task.StartedAt)
	}

	if task.CompletedAt != nil {
		pbTask.CompletedAt = timestamppb.New(*task.CompletedAt)
	}

	return pbTask
}

// convertTaskLogToProto 转换任务日志为 protobuf 格式
func convertTaskLogToProto(log *model.TaskLog) *pb.TaskLog {
	return &pb.TaskLog{
		LogId:      fmt.Sprintf("%d", log.ID),
		TaskId:     log.TaskID,
		LogType:    string(log.LogType),
		FromStatus: string(log.FromStatus),
		ToStatus:   string(log.ToStatus),
		Message:    log.Message,
		WorkerId:   log.WorkerID,
		CreatedAt:  timestamppb.New(log.CreatedAt),
	}
}
