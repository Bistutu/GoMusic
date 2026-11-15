package logic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNetEaseOriginalExampleOfficialRedirectChain(t *testing.T) {
	for _, input := range []string{"http://163cn.tv/zoIxm3", "https://163cn.tv/zoIxm3", "分享歌单 http://163cn.tv/zoIxm3 (@网易云音乐)"} {
		t.Run(input, func(t *testing.T) {
			calls := 0
			service := testService(func(r *http.Request) (*http.Response, error) {
				calls++
				location := ""
				switch r.URL.Host {
				case "163cn.tv":
					location = "https://y.music.163.com/m/playlist?app_version=8.10.81&id=8725919816&userid=341246998"
				case "y.music.163.com":
					location = "https://music.163.com/playlist?id=8725919816"
				case "music.163.com":
					if strings.Contains(r.URL.Path, "/api/") {
						if err := r.ParseForm(); err != nil {
							return nil, err
						}
						if r.Form.Get("id") != "8725919816" {
							t.Errorf("wrong playlist ID: %s", r.Form.Get("id"))
						}
						return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "Original example", "trackCount": 2, "trackIds": []map[string]int{{"id": 1}, {"id": 2}}, "tracks": regressionSongs("netease", 1, 2)}}), nil
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("share HTML")), Header: make(http.Header), Request: r}, nil
				}
				return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": []string{location}}, Request: r}, nil
			})
			list, err := service.Fetch(context.Background(), input, Options{})
			if err != nil || list == nil || !list.Complete || list.SongsCount != 2 || calls != 4 {
				t.Fatalf("list=%v err=%v calls=%d", list, err, calls)
			}
		})
	}
}
