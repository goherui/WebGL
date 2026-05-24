package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	useRedis bool
	codes    map[string]codeEntry
	mu       sync.RWMutex
}

type codeEntry struct {
	code      string
	expiresAt time.Time
}

func NewRedisClient(logger log.Logger) (*RedisClient, func(), error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "115.190.54.31:6379",
		Password: "redis_6jBRR2",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.NewHelper(logger).Warnf("Redis 连接失败，使用内存存储: %v", err)
		return &RedisClient{
			useRedis: false,
			codes:    make(map[string]codeEntry),
		}, nil, nil
	}

	log.NewHelper(logger).Info("Redis connected")

	return &RedisClient{
		client:  rdb,
		useRedis: true,
		codes:   make(map[string]codeEntry),
	}, nil, nil
}

func NewRedisClientWithConfig(logger log.Logger, addr, password string) (*RedisClient, func(), error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.NewHelper(logger).Warnf("Redis 连接失败，使用内存存储: %v", err)
		return &RedisClient{
			useRedis: false,
			codes:    make(map[string]codeEntry),
		}, nil, nil
	}

	log.NewHelper(logger).Info("Redis connected")

	return &RedisClient{
		client:   rdb,
		useRedis: true,
		codes:    make(map[string]codeEntry),
	}, nil, nil
}

func (r *RedisClient) Close() {
	if r.client != nil {
		r.client.Close()
	}
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.useRedis {
		return r.client.Set(ctx, key, value, expiration).Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	
	strValue, ok := value.(string)
	if !ok {
		strValue = fmt.Sprintf("%v", value)
	}
	
	r.codes[key] = codeEntry{
		code:      strValue,
		expiresAt: time.Now().Add(expiration),
	}
	
	go r.cleanup(key, expiration)
	return nil
}

func (r *RedisClient) cleanup(key string, delay time.Duration) {
	time.Sleep(delay)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.codes, key)
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r.useRedis {
		return r.client.Get(ctx, key).Result()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entry, exists := r.codes[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}
	
	if time.Now().After(entry.expiresAt) {
		return "", fmt.Errorf("key expired")
	}
	
	return entry.code, nil
}

func (r *RedisClient) Del(ctx context.Context, key string) error {
	if r.useRedis {
		return r.client.Del(ctx, key).Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.codes, key)
	return nil
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if r.useRedis {
		result, err := r.client.Exists(ctx, key).Result()
		return result > 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	
	entry, exists := r.codes[key]
	if !exists {
		return false, nil
	}
	
	if time.Now().After(entry.expiresAt) {
		return false, nil
	}
	
	return true, nil
}
