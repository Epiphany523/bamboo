package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client Redis 客户端封装
type Client struct {
	client *redis.Client
}

// NewClient 创建 Redis 客户端
func NewClient(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 100,
	})

	return &Client{client: rdb}
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}

// GetClient 获取原生客户端
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// SetNX 设置键值（不存在时）
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// Set 设置键值
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Keys 查找键
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// LPush 左侧推入列表
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPop 右侧弹出列表
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.client.RPop(ctx, key).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, key).Result()
}

// HSet 设置哈希字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

// HGetAll 获取哈希所有字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HIncrBy 哈希字段增加
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) error {
	return c.client.HIncrBy(ctx, key, field, incr).Err()
}

// SAdd 添加到集合
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// SRem 从集合移除
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, key, members...).Err()
}

// ZAdd 添加到有序集合
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRangeByScore 根据分数范围获取有序集合成员
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, opt).Result()
}

// ZRem 从有序集合移除
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.ZRem(ctx, key, members...).Err()
}

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅频道
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}
