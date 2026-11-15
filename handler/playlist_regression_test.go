package handler

import (
	"GoMusic/internal/config"
	"GoMusic/logic"
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type playlistTransport func(*http.Request) (*http.Response, error)

func (f playlistTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLargePlaylistPartialResultSurvivesBothHTTPContracts(t *testing.T) {
	for _, contract := range []struct{ path, body, contentType string }{
		{"/api/v1/songlists", `{"url":"https://y.qq.com/n/ryqq_v2/playlist/1","format":"singer-song","order":"reverse"}`, "application/json"},
		{"/songlist?format=singer-song&order=reverse", "url=https%3A%2F%2Fy.qq.com%2Fn%2Fryqq_v2%2Fplaylist%2F1", "application/x-www-form-urlencoded"},
	} {
		t.Run(contract.path, func(t *testing.T) {
			transport := playlistTransport(func(r *http.Request) (*http.Response, error) {
				var payload any
				if r.Method == http.MethodPost {
					var input models.QQMusicReq
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						return nil, err
					}
					payload = map[string]int{"code": 2000}
					if input.Req0.Param.SongBegin == 0 {
						songs := make([]map[string]any, 30)
						for i := range songs {
							songs[i] = map[string]any{"id": i + 1, "name": fmt.Sprintf("Song %d", i+1), "singer": []map[string]string{{"name": "Artist"}}}
						}
						payload = map[string]any{"code": 0, "req_0": map[string]any{"code": 0, "data": map[string]any{
							"dirinfo": map[string]any{"title": "Large playlist", "songnum": 2800}, "songlist": songs,
						}}}
					}
				} else {
					songs := make([]map[string]any, 60)
					for i := range songs {
						songs[i] = map[string]any{"songid": i + 1, "songname": fmt.Sprintf("Song %d", i+1), "singer": []map[string]string{{"name": "Artist"}}}
					}
					payload = map[string]any{"code": 0, "cdlist": []map[string]any{{"dissname": "Preview", "songnum": 60, "songlist": songs}}}
				}
				body, _ := json.Marshal(payload)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header), Request: r}, nil
			})
			cfg, _ := config.FromEnv(func(string) string { return "" })
			service := logic.NewService(httputil.NewClient(&http.Client{Transport: transport}), nil, nil, cfg.MaxSongs, cfg.Concurrency)
			router := NewRouterWithService(service, cfg, nil)
			request := httptest.NewRequest(http.MethodPost, contract.path, strings.NewReader(contract.body))
			request.Header.Set("Content-Type", contract.contentType)
			out := httptest.NewRecorder()
			router.ServeHTTP(out, request)
			var result struct {
				Code int             `json:"code"`
				Data models.SongList `json:"data"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			list := result.Data
			if out.Code != 200 || result.Code != 1 || list.SongsCount != 60 || list.TotalCount != 2800 || list.Complete || len(list.Warnings) == 0 {
				t.Fatalf("partial result lost through HTTP: status=%d body=%s", out.Code, out.Body.String())
			}
			if len(list.Songs) != 60 || list.Songs[0] != "Artist - Song 60" || list.Songs[59] != "Artist - Song 1" {
				t.Fatalf("legacy format/order changed: %v", list.Songs)
			}
		})
	}
}
