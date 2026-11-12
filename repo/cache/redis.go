package cache

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"GoMusic/misc/log"
)

const (
	month = 30 * 24 * time.Hour
)

var (
	ctx      = context.Background()
	rdb      *redis.Client
	disabled = os.Getenv("GOMUSIC_DISABLE_REDIS") == "1"
)

func init() {
	if disabled {
		log.Warn("GOMUSIC_DISABLE_REDIS=1，本地模式不启用 Redis 缓存")
		return
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:16379",  // redis 服务端地址
		Password: "SzW7fh2Fs5d2ypwT", // redis 密码
		DB:       0,
	})
}

func SetKey(key string, value string) error {
	if disabled {
		return nil
	}
	return rdb.Set(ctx, key, value, 30*time.Second).Err() // 缓存 30 秒
}

func GetKey(key string) (string, error) {
	if disabled {
		return "", nil
	}
	val, err := rdb.Get(ctx, key).Result()
	if err != redis.Nil && err != nil {
		return "", err
	}
	return val, nil
}

func MGet(keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, errors.New("keys is empty")
	}
	if disabled {
		return make([]interface{}, len(keys)), nil
	}
	result, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		log.Errorf("MGet error: %v", err)
	}
	return result, err
}

func MSet(kv sync.Map) error {
	if disabled {
		return nil
	}
	pipeline := rdb.Pipeline()
	kv.Range(func(k, v any) bool {
		// 缓存 30 天
		pipeline.Set(ctx, k.(string), v, month)
		return true
	})
	// 不关注单个命令的执行结果，只关注 pipeline 执行的结果
	if _, err := pipeline.Exec(ctx); err != nil {
		log.Error("MSet error: ", err)
		return err
	}
	return nil
}
