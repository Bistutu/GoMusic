package logic

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"GoMusic/internal/musicurl"
	"GoMusic/misc/httputil"
)

func TestDecodeQQV8Playlist(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"cdlist": [{
				"dissname": "测试歌单",
				"songnum": 2,
				"songlist": [
					{"songid": 123, "songmid": "abc", "songname": "第一首", "singer": [{"name": "歌手甲"}, {"name": "歌手乙"}]},
					{"songid": 456, "songmid": "def", "songname": "第二首", "singer": [{"name": "歌手丙"}]}
				]
			}]
		}
	}`

	list, err := decodeQQV8Playlist([]byte(body))
	if err != nil {
		t.Fatalf("decodeQQV8Playlist() error = %v", err)
	}
	if list.Name != "测试歌单" || list.TotalCount != 2 {
		t.Fatalf("name=%q total=%d", list.Name, list.TotalCount)
	}
	if len(list.Tracks) != 2 || list.Tracks[0].ID != "123" || list.Tracks[0].Title != "第一首" {
		t.Fatalf("unexpected tracks: %#v", list.Tracks)
	}
	if got := strings.Join(list.Tracks[0].Artists, " / "); got != "歌手甲 / 歌手乙" {
		t.Fatalf("artists = %q", got)
	}
}

func TestDecodeQQV8PlaylistError(t *testing.T) {
	for _, body := range []string{
		`{"code":0,"subcode":4000,"msg":"歌单不存在"}`,
		`{"code":0,"data":{"cdlist":[]}}`,
		`{not json`,
	} {
		if _, err := decodeQQV8Playlist([]byte(body)); err == nil {
			t.Fatalf("expected error for %s", body)
		}
	}
}

func TestStripQQJSONP(t *testing.T) {
	got, err := stripQQJSONP([]byte(`  callback({"code":0,"cdlist":[]})  `))
	if err != nil {
		t.Fatalf("stripQQJSONP() error = %v", err)
	}
	if string(got) != `{"code":0,"cdlist":[]}` {
		t.Fatalf("payload = %q", string(got))
	}
	if _, err = stripQQJSONP([]byte(`{"code":0}`)); err == nil {
		t.Fatal("expected error for plain JSON")
	}
}

func TestQQFallbackOrder(t *testing.T) {
	fallbacks := NewService(nil, nil, nil, 10000, 4).qqFallbacks()
	want := []string{"legacy", "plain-musicu", "v8-playlist", "qzone-music"}
	if len(fallbacks) != len(want) {
		t.Fatalf("fallback count = %d, want %d", len(fallbacks), len(want))
	}
	for i, name := range want {
		if fallbacks[i].name != name {
			t.Fatalf("fallback[%d] = %q, want %q", i, fallbacks[i].name, name)
		}
	}
}

// TestQQFallbackProvidersLive 逐个验证备用接口能返回同一份公开歌单。
// 运行：GOMUSIC_LIVE_TEST=1 go test ./logic -run TestQQFallbackProvidersLive -v
func TestQQFallbackProvidersLive(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	const id = "7364061065"
	service := NewService(nil, nil, nil, 10000, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// legacy 是原有接口，对部分公开歌单会返回 check privacy error；
	// 这里只验证本次新增的三个备用接口均可独立取到完整歌单。
	for _, fallback := range service.qqFallbacks()[1:] {
		t.Run(fallback.name, func(t *testing.T) {
			list, err := fallback.fetch(ctx, id)
			if err != nil {
				t.Fatalf("%s error = %v", fallback.name, err)
			}
			if list.Name == "" || len(list.Tracks) == 0 {
				t.Fatalf("%s returned empty list: %#v", fallback.name, list)
			}
			t.Logf("name=%q reported=%d fetched=%d first=%q", list.Name, list.TotalCount, len(list.Tracks), list.Tracks[0].Title)
			if len(list.Tracks) < 30 {
				t.Fatalf("%s returned only %d tracks", fallback.name, len(list.Tracks))
			}
		})
	}
}

// blockingTransport 模拟主接口域名故障，用于验证 fallback 是否真正生效。
type blockingTransport struct {
	blockHost string
	base      http.RoundTripper
}

func (t *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == t.blockHost {
		return nil, fmt.Errorf("simulated outage for %s", t.blockHost)
	}
	return t.base.RoundTrip(req)
}

// TestQQMusicPrimaryOutageFallback 模拟 u6 签名接口整体不可用，
// 验证仍能从备用接口拿到完整歌单。
// 运行：GOMUSIC_LIVE_TEST=1 go test ./logic -run TestQQMusicPrimaryOutageFallback -v
func TestQQMusicPrimaryOutageFallback(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	client := httputil.NewClient(&http.Client{
		Transport: &blockingTransport{blockHost: "u6.y.qq.com", base: http.DefaultTransport},
	})
	service := NewService(client, nil, nil, 10000, 4)
	link, err := musicurl.Parse("https://y.qq.com/n/ryqq/playlist/7364061065")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	list, err := service.qqMusic(ctx, link)
	if err != nil {
		t.Fatalf("primary outage fallback error = %v", err)
	}
	if list.Name == "" || len(list.Tracks) == 0 {
		t.Fatalf("fallback returned empty list: %#v", list)
	}
	t.Logf("fallback name=%q fetched=%d reported=%d first=%q", list.Name, len(list.Tracks), list.TotalCount, list.Tracks[0].Title)
	if len(list.Tracks) < 100 {
		t.Fatalf("fallback returned only %d tracks, want a complete playlist", len(list.Tracks))
	}
}
