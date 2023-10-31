package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "", // redis 服务端地址
		Password: "", // redis 密码
		DB:       0,
	})
}

func SetKey(key string, value string) error {
	err := rdb.Set(ctx, key, value, 30*time.Second).Err() // 缓存 30 秒
	return err
}

func GetKey(key string) (string, error) {
	val, err := rdb.Get(ctx, key).Result()
	if err != redis.Nil && err != nil {
		return "", err
	}
	return val, nil
}
