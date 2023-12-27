package db

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"GoMusic/misc/models"
)

func TestBatchDelAndSet(t *testing.T) {
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
	err := BatchDelSong([]int{5241457, 1935948203})
	assert.NoError(t, err)

	// Set
	err = BatchInsertSong(songs)
	assert.NoError(t, err)
}

func TestBatchGet(t *testing.T) {
	songs, err := BatchGetSongById([]uint{5241457, 1935948203})
	assert.NoError(t, err)
	assert.Equal(t, "小酒窝(Live) - 蔡卓妍 / 林俊杰", songs[5241457])
	assert.Equal(t, "星河万里 - 王大毛", songs[1935948203])
}
