package handler

import (
	"GoMusic/internal/config"
	"GoMusic/logic"
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKugouCodeThroughJSONAndLegacyContracts(t *testing.T) {
	for _, tc := range []struct {
		name, path, body, contentType string
		expired                       bool
	}{
		{"json", "/api/v1/songlists", `{"url":"40725013","format":"singer-song","order":"reverse"}`, "application/json", false},
		{"form", "/songlist?format=singer-song&order=reverse", "url=40725013", "application/x-www-form-urlencoded", false},
		{"label", "/api/v1/songlists", `{"url":"酷狗码：40725013","format":"singer-song","order":"reverse"}`, "application/json", false},
		{"expired-json", "/api/v1/songlists", `{"url":"40725013"}`, "application/json", true},
		{"expired-form", "/songlist", "url=40725013", "application/x-www-form-urlencoded", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := playlistTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "t.kugou.com" || r.URL.Path != "/command/" {
					t.Fatalf("unexpected request %s", r.URL)
				}
				var payload any = map[string]any{"status": 1, "data": map[string]any{"info": map[string]any{"type": 4, "name": "Code playlist", "count": 2}, "list": []map[string]string{
					{"hash": "1", "songname": "Song 1", "singername": "Artist"}, {"hash": "2", "songname": "Song 2", "singername": "Artist"},
				}}}
				if tc.expired {
					payload = map[string]int{"status": 0, "err_code": 10001}
				}
				body, _ := json.Marshal(payload)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header), Request: r}, nil
			})
			cfg, _ := config.FromEnv(func(string) string { return "" })
			service := logic.NewService(httputil.NewClient(&http.Client{Transport: transport}), nil, nil, cfg.MaxSongs, cfg.Concurrency)
			router := NewRouterWithService(service, cfg, nil)
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			out := httptest.NewRecorder()
			router.ServeHTTP(out, req)
			if tc.expired {
				if out.Code != 400 || !strings.Contains(out.Body.String(), "酷狗码") {
					t.Fatalf("status=%d body=%s", out.Code, out.Body)
				}
				return
			}
			var result struct {
				Code int             `json:"code"`
				Data models.SongList `json:"data"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if out.Code != 200 || result.Code != 1 || result.Data.SongsCount != 2 || !result.Data.Complete || result.Data.Songs[0] != "Artist - Song 2" {
				t.Fatalf("status=%d body=%s", out.Code, out.Body)
			}
		})
	}
}
