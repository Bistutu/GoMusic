package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"GoMusic/common/models"
)

func TestBatchDelAndSet(t *testing.T) {
	ctx := context.Background()

	var songs []*models.NetEasySong
	songs = append(songs, &models.NetEasySong{
		Id:   5241457,
		Name: "小酒窝(Live) - 蔡卓妍 / 林俊杰",
	})
	songs = append(songs, &models.NetEasySong{
		Id:   1935948203,
		Name: "星河万里 - 王大毛",
	})

	// Del
	err := BatchDelSong(ctx, []int{5241457, 1935948203})
	assert.NoError(t, err)

	// Set
	err = BatchInsertSong(ctx, songs)
	assert.NoError(t, err)
}

func TestBatchGet(t *testing.T) {
	ctx := context.Background()
	songs, err := BatchGetSongById(ctx, []int{5241457, 1935948203})
	assert.NoError(t, err)
	assert.Equal(t, "小酒窝(Live) - 蔡卓妍 / 林俊杰", songs[0].Name)
	assert.Equal(t, "星河万里 - 王大毛", songs[1].Name)
}
