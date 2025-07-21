package redisclient

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sort"
	"strings"
	"time"
)

var (
	Ctx = context.Background()
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisClient{Client: rdb}
}

func (r *RedisClient) Set(key string, value string, ttl time.Duration) error {
	return r.Client.Set(Ctx, key, value, ttl).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.Client.Get(Ctx, key).Result()
}

func (r *RedisClient) Del(key string) error {
	return r.Client.Del(Ctx, key).Err()
}

func MakeCacheKey(startDate, endDate string, storeIds []string) string {
	if len(storeIds) == 0 {
		return fmt.Sprintf("count_stats:%s:%s:all", startDate, endDate)
	}
	sort.Strings(storeIds)
	return fmt.Sprintf("count_stats:%s:%s:%s", startDate, endDate, strings.Join(storeIds, ","))
}
