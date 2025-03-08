package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

// RedisClient 结构体封装 Redis 相关操作
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient 初始化 Redis 客户端
func NewRedisClient(config *Config) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	fmt.Println("Redis 连接成功")
	return &RedisClient{Client: client}
}

// PushMessage 将消息推入 Redis 队列
func (r *RedisClient) PushMessage(queue string, message map[string]string) error {
	ctx := context.Background()
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.Client.LPush(ctx, queue, jsonData).Err()
}

// PopMessage 从 Redis 队列取出消息
func (r *RedisClient) PopMessage(ctx context.Context, queue string) (map[string]string, error) {
	result, err := r.Client.BRPop(ctx, 1*time.Second, queue).Result()
	if err != nil || len(result) < 2 {
		return nil, err
	}
	var message map[string]string
	json.Unmarshal([]byte(result[1]), &message)
	return message, err
}

// IncrUserActivity 增加用户活跃度
func (r *RedisClient) IncrUserActivity(username string) {
	ctx := context.Background()
	r.Client.ZIncrBy(ctx, "chatRoom_user_activity", 1, username)
}

// RemoveUser 从活跃度排行榜删除用户
func (r *RedisClient) RemoveUser(username string) {
	ctx := context.Background()
	r.Client.ZRem(ctx, "chatRoom_user_activity", username)
}

// 根据键名删键
func (r *RedisClient) Del(queue string) {
	ctx := context.Background()
	r.Client.Del(ctx, queue)
}

// 关闭Redis
func (r *RedisClient) Close() {
	r.Client.Close()
}

func (r *RedisClient) ShowRange() ([]redis.Z, error) {
	ctx := context.Background()
	result, err := r.Client.ZRevRangeWithScores(ctx, "chatRoom_user_activity", 0, -1).Result()
	return result, err
}
