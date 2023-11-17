package main

import (
	"fmt"
	"strconv"
	"testing"
)

const (
	netEasyRedis = "net:%v"
)

func BenchmarkFmt(b *testing.B) {

	b.Run("1", func(b *testing.B) {
		songCacheKey := make([]any, 0)
		for i := 0; i < b.N; i++ {
			songCacheKey = append(songCacheKey, fmt.Sprintf(netEasyRedis, i))
		}
	})
	b.Run("2", func(b *testing.B) {
		songCacheKey := make([]any, 0)
		for i := 0; i < b.N; i++ {
			songCacheKey = append(songCacheKey, strconv.FormatInt(int64(i), 10))

		}
	})
}
