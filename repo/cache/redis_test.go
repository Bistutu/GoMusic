package cache

import (
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
