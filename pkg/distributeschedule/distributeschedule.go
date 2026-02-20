package distributeschedule

import (
	"context"

	"bamboo/pkg/distributeschedule/config"
	"bamboo/pkg/distributeschedule/domain/service"
	"bamboo/pkg/distributeschedule/interfaces"
)

// DistributeSchedule 分布式调度框架（门面）
type DistributeSchedule struct {
	scheduler *interfaces.Scheduler
}

// New 创建分布式调度框架实例
func New(cfg *config.Config) (*DistributeSchedule, error) {
	scheduler, err := interfaces.NewScheduler(cfg)
	if err != nil {
		return nil, err
	}

	return &DistributeSchedule{
		scheduler: scheduler,
	}, nil
}

// NewWithDefaultConfig 使用默认配置创建实例
func NewWithDefaultConfig() (*DistributeSchedule, error) {
	return New(config.DefaultConfig())
}

// Start 启动调度框架
func (ds *DistributeSchedule) Start(ctx context.Context) error {
	return ds.scheduler.Start(ctx)
}

// RegisterExecutor 注册自定义执行器
func (ds *DistributeSchedule) RegisterExecutor(executor service.Executor) {
	ds.scheduler.RegisterExecutor(executor)
}

// Close 关闭调度框架
func (ds *DistributeSchedule) Close() error {
	return ds.scheduler.Close()
}
