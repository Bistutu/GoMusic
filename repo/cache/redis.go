package cache

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"GoMusic/initialize/log"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",   // redis 服务端地址
		Password: "SzW7fh2Fs5d2ypwT", // redis 密码
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

func MGet(keys ...string) ([]interface{}, error) {
	result, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func MSet(kv sync.Map) error {
	pipeline := rdb.Pipeline()
	kv.Range(func(k, v any) bool {
		// 缓存 24 小时
		pipeline.Set(ctx, k.(string), v, 24*time.Hour)
		return true
	})
	// 不关注单个命令的执行结果，只关注 pipeline 执行的结果
	_, err := pipeline.Exec(ctx)
	if err != nil {
		log.Error("MSet error: ", err)
		return err
	}
	return nil
}
