package redis

import (
	"context"
	"fmt"
	"time"
)

const (
	leaderKey = "schedule:leader"
	leaderTTL = 10 * time.Second
)

// LeaderElection Leader 选举
type LeaderElection struct {
	client   *Client
	leaderID string
}

// NewLeaderElection 创建 Leader 选举
func NewLeaderElection(client *Client, leaderID string) *LeaderElection {
	return &LeaderElection{
		client:   client,
		leaderID: leaderID,
	}
}

// TryAcquire 尝试获取 Leader 锁
func (le *LeaderElection) TryAcquire(ctx context.Context) (bool, error) {
	acquired, err := le.client.SetNX(ctx, leaderKey, le.leaderID, leaderTTL)
	if err != nil {
		return false, fmt.Errorf("acquire leader lock failed: %w", err)
	}

	return acquired, nil
}

// Renew 续约 Leader 锁
func (le *LeaderElection) Renew(ctx context.Context) error {
	// 检查当前 leader 是否是自己
	currentLeader, err := le.client.Get(ctx, leaderKey)
	if err != nil {
		return fmt.Errorf("get current leader failed: %w", err)
	}

	if currentLeader != le.leaderID {
		return fmt.Errorf("not the current leader")
	}

	// 续约
	return le.client.Expire(ctx, leaderKey, leaderTTL)
}

// Release 释放 Leader 锁
func (le *LeaderElection) Release(ctx context.Context) error {
	// 检查当前 leader 是否是自己
	currentLeader, err := le.client.Get(ctx, leaderKey)
	if err != nil {
		return fmt.Errorf("get current leader failed: %w", err)
	}

	if currentLeader != le.leaderID {
		return fmt.Errorf("not the current leader")
	}

	return le.client.Del(ctx, leaderKey)
}

// IsLeader 判断是否是 Leader
func (le *LeaderElection) IsLeader(ctx context.Context) (bool, error) {
	currentLeader, err := le.client.Get(ctx, leaderKey)
	if err != nil {
		return false, nil
	}

	return currentLeader == le.leaderID, nil
}

// GetLeader 获取当前 Leader
func (le *LeaderElection) GetLeader(ctx context.Context) (string, error) {
	return le.client.Get(ctx, leaderKey)
}
