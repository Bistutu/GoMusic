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
		{"qq", "https://c6.y.qq.com/base/fcgi-bin/u?__=GNSf48IjarL6", 31},
		{"qishui", "https://qishui.douyin.com/s/iAA9MFsL/", 1},
		{"qishui-long-share-with-tracking", "https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&playlist_id=7203983736064622592", 1},
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

// This opt-in smoke test reads only public playlists. Keep the requests serial
// so the checked platforms do not receive a burst.
func TestLiveExpandedProviders(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	tests := []struct {
		name, url string
	}{
		{"kuwo", "https://www.kuwo.cn/newh5/playlist/content?pid=236289985"},
		{"migu", "https://music.migu.cn/v5/#/playlist?playlistId=231760782&playlistType=ordinary"},
		{"qianqian", "https://music.91q.com/songlist/295893"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, tt.url, Options{Detailed: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
			if list.SongsCount == 0 || list.SongsCount != len(list.Tracks) || list.SongsCount != len(list.Songs) {
				t.Fatalf("unexpected result counts: songs=%d tracks=%d formatted=%d", list.SongsCount, len(list.Tracks), len(list.Songs))
			}
			if !list.Complete {
				t.Fatalf("known public sample returned a partial result: %d/%d: %v", list.SongsCount, list.TotalCount, list.Warnings)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	kuwoFallback, err := NewService(nil, nil, nil, 10000, 4).kuwoH5Pages(ctx, "236289985")
	if err != nil || kuwoFallback == nil || len(kuwoFallback.Tracks) == 0 {
		t.Fatalf("Kuwo public H5 JSON fallback returned no rows: list=%#v error=%v", kuwoFallback, err)
	}
	t.Logf("kuwo-h5-json-fallback name=%q fetched=%d api_total=%d", kuwoFallback.Name, len(kuwoFallback.Tracks), kuwoFallback.TotalCount)
}
