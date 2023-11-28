package cache

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	msg := []string{"test1", "value1"}
	err := SetKey(msg[0], msg[1])
	assert.NoError(t, err)
	rs, err := GetKey(msg[0])
	assert.NoError(t, err)
	assert.Equal(t, msg[1], rs)
}

// 测试布隆过滤器性能
func BenchmarkBloom(b *testing.B) {

	b.Run("common", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			SetKey(strconv.Itoa(i), strconv.Itoa(i))
			rs, err := GetKey(strconv.Itoa(i))
			assert.NoError(b, err)
			assert.Equal(b, strconv.Itoa(i), rs)
		}
	})
	b.Run("bloom", func(b *testing.B) {
		for i := 0; i < b.N; i++ {

		}
	})
}
