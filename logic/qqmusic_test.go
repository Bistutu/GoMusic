package logic

import (
	"context"
	"os"
	"testing"
	"time"
)

// Live tests are opt-in; deterministic provider tests never require external services.
func TestLiveProviders(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	tests := []struct {
		name, url string
		min       int
	}{
		{"netease", "https://music.163.com/playlist?id=19723756", 1},
		{"netease-original-example", "http://163cn.tv/zoIxm3", 1},
		{"netease-album", "https://music.163.com/album?id=35150843", 1},
		{"qq", "https://c6.y.qq.com/base/fcgi-bin/u?__=GNSf48IjarL6", 31},
		{"qishui", "https://qishui.douyin.com/s/iAA9MFsL/", 1},
		{"kugou", "https://www.kugou.com/songlist/gcid_3z17seve8z2z051/", 1},
		{"kugou-user-share", "https://m.kugou.com/songlist/gcid_3zbivkavz4z072/?src_cid=3zbivkavz4z072&uid=606105881&chl=wechat&iszlist=1", 12},
		{"kugou-code", "40725013", 12},
		{"kugou-collection", "https://activity.kugou.com/share/index.html?global_specialid=collection_3_2248574973_2_0", 1},
	}
	t.Run("netease-song-details", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		tracks, err := NewService(nil, nil, nil, 10000, 4).netEaseSongs(ctx, []string{"186016"}, nil)
		if err != nil || len(tracks) != 1 || tracks["186016"].Title == "" {
			t.Fatalf("detail result=%v error=%v", tracks, err)
		}
		t.Logf("detail id=186016 title=%q artists=%v", tracks["186016"].Title, tracks["186016"].Artists)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, tt.url, Options{Detailed: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
			if list.SongsCount < tt.min || list.SongsCount != len(list.Songs) {
				t.Fatalf("unexpected count %d", list.SongsCount)
			}
		})
	}
}
