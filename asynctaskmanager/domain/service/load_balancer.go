package service

import (
	"errors"
	"hash/crc32"
	"sync/atomic"

	"bamboo/asynctaskmanager/domain/model"
)

var (
	ErrNoAvailableWorker = errors.New("no available worker")
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy string

const (
	StrategyLeastTask      LoadBalanceStrategy = "least_task"      // 最少任务优先
	StrategyRoundRobin     LoadBalanceStrategy = "round_robin"     // 轮询
	StrategyConsistentHash LoadBalanceStrategy = "consistent_hash" // 一致性哈希
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(workers []*model.Worker, taskID string) (*model.Worker, error)
}

// LeastTaskLoadBalancer 最少任务优先负载均衡器
type LeastTaskLoadBalancer struct{}

func NewLeastTaskLoadBalancer() *LeastTaskLoadBalancer {
	return &LeastTaskLoadBalancer{}
}

func (lb *LeastTaskLoadBalancer) Select(workers []*model.Worker, taskID string) (*model.Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoAvailableWorker
	}

	var selected *model.Worker
	minLoad := int(^uint(0) >> 1) // max int

	for _, worker := range workers {
		if worker.CanAcceptTask() && worker.CurrentLoad < minLoad {
			selected = worker
			minLoad = worker.CurrentLoad
		}
	}

	if selected == nil {
		return nil, ErrNoAvailableWorker
	}

	return selected, nil
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	counter uint64
}

func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

func (lb *RoundRobinLoadBalancer) Select(workers []*model.Worker, taskID string) (*model.Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoAvailableWorker
	}

	// 过滤可用的 worker
	available := make([]*model.Worker, 0)
	for _, worker := range workers {
		if worker.CanAcceptTask() {
			available = append(available, worker)
		}
	}

	if len(available) == 0 {
		return nil, ErrNoAvailableWorker
	}

	index := atomic.AddUint64(&lb.counter, 1) % uint64(len(available))
	return available[index], nil
}

// ConsistentHashLoadBalancer 一致性哈希负载均衡器
type ConsistentHashLoadBalancer struct{}

func NewConsistentHashLoadBalancer() *ConsistentHashLoadBalancer {
	return &ConsistentHashLoadBalancer{}
}

func (lb *ConsistentHashLoadBalancer) Select(workers []*model.Worker, taskID string) (*model.Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoAvailableWorker
	}

	// 过滤可用的 worker
	available := make([]*model.Worker, 0)
	for _, worker := range workers {
		if worker.CanAcceptTask() {
			available = append(available, worker)
		}
	}

	if len(available) == 0 {
		return nil, ErrNoAvailableWorker
	}

	hash := crc32.ChecksumIEEE([]byte(taskID))
	index := int(hash) % len(available)
	return available[index], nil
}

// LoadBalancerFactory 负载均衡器工厂
func LoadBalancerFactory(strategy LoadBalanceStrategy) LoadBalancer {
	switch strategy {
	case StrategyRoundRobin:
		return NewRoundRobinLoadBalancer()
	case StrategyConsistentHash:
		return NewConsistentHashLoadBalancer()
	default:
		return NewLeastTaskLoadBalancer()
	}
}
