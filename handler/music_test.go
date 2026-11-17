package handler

import (
	"GoMusic/internal/config"
	"GoMusic/logic"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fetchFunc func(context.Context, string, logic.Options) (*models.SongList, error)

func (f fetchFunc) Fetch(ctx context.Context, url string, o logic.Options) (*models.SongList, error) {
	return f(ctx, url, o)
}
func TestLegacyAndJSONContracts(t *testing.T) {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	router := NewRouterWithService(fetchFunc(func(_ context.Context, url string, o logic.Options) (*models.SongList, error) {
		if url != "https://y.qq.com/n/ryqq/playlist/1" || !o.Detailed || o.Format != "singer-song" || o.Order != "reverse" {
			t.Fatalf("input/options lost: %q %#v", url, o)
		}
		return &models.SongList{Name: "test", Songs: []string{"Artist - Song"}, SongsCount: 1, TotalCount: 1, Complete: true}, nil
	}), cfg, nil)
	for _, tt := range []struct{ path, body, kind string }{
		{"/songlist?detailed=true&format=singer-song&order=reverse", "url=https%3A%2F%2Fy.qq.com%2Fn%2Fryqq%2Fplaylist%2F1", "application/x-www-form-urlencoded"},
		{"/api/v1/songlists", `{"url":"https://y.qq.com/n/ryqq/playlist/1","detailed":true,"format":"singer-song","order":"reverse"}`, "application/json"},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", tt.path, strings.NewReader(tt.body))
		request.Header.Set("Content-Type", tt.kind)
		router.ServeHTTP(recorder, request)
		var body models.Result
		json.Unmarshal(recorder.Body.Bytes(), &body)
		if recorder.Code != 200 || body.Code != 1 || body.Msg != "success" {
			t.Fatalf("contract failed: %d %s", recorder.Code, recorder.Body.String())
		}
	}
}
func TestRequestValidation(t *testing.T) {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	router := NewRouterWithService(fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) {
		t.Fatal("invalid input reached upstream")
		return nil, nil
	}), cfg, nil)
	for _, tt := range []struct {
		body, kind, query string
		status            int
	}{
		{`{"url":"x","format":"invalid"}`, "application/json", "", 400}, {`{"url":"x"}`, "application/json", "?detailed=wat", 400},
		{`{"url":"x","unexpected":1}`, "application/json", "", 400}, {`{"url":"x"} {}`, "application/json", "", 400},
		{`{`, "application/json", "", 400}, {`{}`, "application/json", "", 400}, {`{"url":"` + strings.Repeat("x", 9000) + `"}`, "application/json", "", 413},
		{`url=x`, "text/plain", "", 415},
	} {
		request := httptest.NewRequest("POST", "/api/v1/songlists"+tt.query, strings.NewReader(tt.body))
		request.Header.Set("Content-Type", tt.kind)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != tt.status {
			t.Errorf("status=%d want=%d body=%s", recorder.Code, tt.status, recorder.Body.String())
		}
	}
}

func TestRetiredDeezerShareHintReachesAPI(t *testing.T) {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	router := NewRouterWithService(logic.NewService(nil, nil, nil, 10000, 4), cfg, nil)
	request := httptest.NewRequest("POST", "/api/v1/songlists", strings.NewReader(`{"url":"https://deezer.page.link/abc123"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var result models.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusBadRequest || !strings.Contains(result.Msg, "旧版 page.link 短链已停止跳转") || !strings.Contains(result.Msg, "link.deezer.com/s/") {
		t.Fatalf("retired Deezer link recovery hint was lost: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTimeoutAndUpstreamErrors(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{context.DeadlineExceeded, 504}, {errors.New("internal DSN secret"), 502}, {logic.ErrInvalidLink, 400}, {logic.ErrPrivatePlaylist, 403}} {
		cfg, _ := config.FromEnv(func(string) string { return "" })
		router := NewRouterWithService(fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) { return nil, tt.err }), cfg, nil)
		req := httptest.NewRequest("POST", "/songlist", strings.NewReader("url=https://y.qq.com/n/ryqq/playlist/1"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != tt.status || strings.Contains(out.Body.String(), "secret") {
			t.Fatalf("status or information leak: %d %s", out.Code, out.Body.String())
		}
	}
}
func TestHealthAndFrontend(t *testing.T) {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	cfg.StaticDir = t.TempDir()
	os.Mkdir(filepath.Join(cfg.StaticDir, "assets"), 0755)
	os.WriteFile(filepath.Join(cfg.StaticDir, "index.html"), []byte("<html>GoMusic</html>"), 0644)
	os.WriteFile(filepath.Join(cfg.StaticDir, "assets", "app.js"), []byte("hello"), 0644)
	router := NewRouterWithService(nil, cfg, map[string]HealthCheck{"redis": func(context.Context) error { return errors.New("offline") }})
	for path, status := range map[string]int{"/": 200, "/assets/app.js": 200, "/healthz": 200, "/readyz": 503, "/missing": 404} {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, path, nil))
		if out.Code != status {
			t.Errorf("%s: %d want %d", path, out.Code, status)
		}
	}
	cfg.RequestTimeout = time.Second
}
