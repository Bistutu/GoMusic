package handler

import (
	"GoMusic/internal/config"
	"GoMusic/internal/usage"
	"GoMusic/logic"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type usageMemory struct {
	events            []usage.Event
	windows           []usage.Range
	writeErr, readErr error
	cancelledWrite    bool
}

func (m *usageMemory) RecordUsage(ctx context.Context, event usage.Event) error {
	if ctx.Err() != nil {
		m.cancelledWrite = true
	}
	m.events = append(m.events, event)
	return m.writeErr
}
func (m *usageMemory) UsageStats(_ context.Context, window usage.Range) (usage.Result, error) {
	m.windows = append(m.windows, window)
	return usage.Result{Period: window.Period}, m.readErr
}

func TestUsageCountsActualSongsAndReusesAnonymousVisitor(t *testing.T) {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	store := &usageMemory{}
	fetch := fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) {
		return &models.SongList{Tracks: []models.Track{{ID: "1", Title: "Song"}, {ID: "2", Title: "Other"}}, SongsCount: 2, TotalCount: 2800, Complete: false}, nil
	})
	router := NewRouterWithUsage(fetch, cfg, nil, store)
	var cookie *http.Cookie
	for _, path := range []string{"/api/v1/songlists", "/songlist"} {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"url":"https://y.qq.com/n/ryqq/playlist/1"}`))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
		}
		if cookie == nil {
			cookies := w.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != visitorCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
				t.Fatalf("invalid anonymous cookie: %v", cookies)
			}
			cookie = cookies[0]
		}
	}
	if len(store.events) != 2 || store.events[0].VisitorHash != store.events[1].VisitorHash {
		t.Fatal("same browser was counted as different visitors")
	}
	for _, event := range store.events {
		if event.Songs != 2 || !event.Succeeded || !event.Partial || len(event.VisitorHash) != 64 || event.VisitorHash == cookie.Value {
			t.Fatalf("wrong or identifying event: %#v", event)
		}
	}
}

func TestUsageStorageFailureDoesNotBreakPlaylistAndInvalidRequestsAreFailures(t *testing.T) {
	for _, kind := range []string{"success", "invalid", "upstream", "cancel", "panic"} {
		t.Run(kind, func(t *testing.T) {
			cfg, _ := config.FromEnv(func(string) string { return "" })
			store := &usageMemory{writeErr: errors.New("storage unavailable")}
			fetch := fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) {
				if kind == "panic" {
					panic("test recovery")
				}
				if kind == "upstream" {
					return nil, errors.New("upstream failed")
				}
				if kind == "cancel" {
					return nil, context.Canceled
				}
				return &models.SongList{Tracks: []models.Track{{ID: "1", Title: "Song"}}, Complete: true}, nil
			})
			body := `{"url":"https://y.qq.com/n/ryqq/playlist/1"}`
			if kind == "invalid" {
				body = `{`
			}
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/v1/songlists", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			if kind == "cancel" {
				ctx, cancel := context.WithCancel(r.Context())
				cancel()
				r = r.WithContext(ctx)
			}
			r.AddCookie(&http.Cookie{Name: visitorCookie, Value: "invalid-cookie"})
			NewRouterWithUsage(fetch, cfg, nil, store).ServeHTTP(w, r)
			if len(store.events) != 1 || store.cancelledWrite {
				t.Fatalf("events=%v", store.events)
			}
			event := store.events[0]
			if event.Succeeded != (kind == "success") || (kind != "success" && event.Songs != 0) || event.Partial {
				t.Fatalf("wrong outcome: %#v", event)
			}
			if kind == "success" && w.Code != 200 {
				t.Fatalf("metrics broke successful response: %d", w.Code)
			}
		})
	}
}

func TestStatsAuthenticationWindowsAndErrors(t *testing.T) {
	for _, tc := range []struct {
		token, auth, query string
		enabled, broken    bool
		status, windows    int
	}{
		{"", "", "", true, false, 404, 0},
		{"secret", "", "", true, false, 401, 0},
		{"secret", "secret", "", true, false, 401, 0},
		{"secret", "Bearer wrong", "", true, false, 401, 0},
		{"secret", "Bearer secret", "", false, false, 503, 0},
		{"secret", "Bearer secret", "?period=invalid", true, false, 400, 0},
		{"secret", "Bearer secret", "?date=2026-02-30", true, false, 400, 0},
		{"secret", "Bearer secret", "?date=2026-11-16", true, false, 200, 4},
		{"secret", "Bearer secret", "?period=week&date=2026-11-16", true, false, 200, 1},
		{"secret", "Bearer secret", "?period=month", true, true, 503, 1},
	} {
		t.Run(tc.auth+tc.query, func(t *testing.T) {
			cfg, _ := config.FromEnv(func(string) string { return "" })
			cfg.StatsToken = tc.token
			store := &usageMemory{}
			if tc.broken {
				store.readErr = errors.New("private DSN secret")
			}
			var repository usage.Repository
			if tc.enabled {
				repository = store
			}
			router := NewRouterWithUsage(nil, cfg, nil, repository)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/v1/stats"+tc.query, nil)
			r.Header.Set("Authorization", tc.auth)
			router.ServeHTTP(w, r)
			if w.Code != tc.status || len(store.windows) != tc.windows || len(store.events) != 0 || strings.Contains(w.Body.String(), "DSN") || w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("status=%d queries=%d body=%s", w.Code, len(store.windows), w.Body.String())
			}
			if tc.status == 200 {
				var body struct {
					Data struct {
						Timezone string `json:"timezone"`
					}
				}
				if json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Data.Timezone != "Asia/Shanghai" {
					t.Fatal(w.Body.String())
				}
				if strings.Contains(tc.query, "period=week") && store.windows[0].Start.Format("2006-01-02") != "2026-11-16" {
					t.Fatal("week did not begin on Monday")
				}
			}
		})
	}
}

func TestUsageCookieSecurityAndStatelessMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg, _ := config.FromEnv(func(string) string { return "" })
		cfg.StatsCookieSecure = true
		store := &usageMemory{}
		var repository usage.Repository
		if enabled {
			repository = store
		}
		fetch := fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) {
			return &models.SongList{Complete: true}, nil
		})
		router := NewRouterWithUsage(fetch, cfg, nil, repository)
		var previous string
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/v1/songlists", strings.NewReader(`{"url":"https://music.163.com/playlist?id=1"}`))
			r.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, r)
			cookies := w.Result().Cookies()
			if enabled {
				if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Value == previous {
					t.Fatalf("anonymous browsers were not isolated: %v", cookies)
				}
				previous = cookies[0].Value
			} else if len(cookies) != 0 || len(store.events) != 0 {
				t.Fatal("stateless mode unexpectedly recorded visitors")
			}
		}
	}
}

func TestUsageCORSAllowsCredentialsOnlyForConfiguredOrigins(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, origin, authorization string
		status, events                            int
	}{
		{"allowed-preflight", "OPTIONS", "/api/v1/songlists", "https://music.example.test", "", 204, 0},
		{"rejected-preflight", "OPTIONS", "/api/v1/songlists", "https://other.example.test", "", 403, 0},
		{"allowed-playlist", "POST", "/api/v1/songlists", "https://music.example.test", "", 200, 1},
		{"rejected-playlist", "POST", "/api/v1/songlists", "https://other.example.test", "", 403, 0},
		{"private-stats", "GET", "/api/v1/stats", "https://music.example.test", "", 401, 0},
		{"authorized-stats", "GET", "/api/v1/stats", "https://music.example.test", "Bearer secret", 200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, _ := config.FromEnv(func(string) string { return "" })
			cfg.AllowedOrigins = []string{"https://music.example.test"}
			cfg.StatsToken = "secret"
			store := &usageMemory{}
			fetch := fetchFunc(func(context.Context, string, logic.Options) (*models.SongList, error) {
				return &models.SongList{Tracks: []models.Track{{ID: "1", Title: "Song"}}, Complete: true}, nil
			})
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"url":"https://music.163.com/playlist?id=1"}`))
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Authorization", tc.authorization)
			if tc.method == "OPTIONS" {
				r.Header.Set("Access-Control-Request-Method", "POST")
				r.Header.Set("Access-Control-Request-Headers", "content-type,authorization")
			}
			w := httptest.NewRecorder()
			NewRouterWithUsage(fetch, cfg, nil, store).ServeHTTP(w, r)
			if w.Code != tc.status || len(store.events) != tc.events {
				t.Fatalf("status=%d events=%d body=%s", w.Code, len(store.events), w.Body.String())
			}
			if tc.status != 403 {
				if w.Header().Get("Access-Control-Allow-Origin") != tc.origin || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Fatalf("browser cannot retain its anonymous cookie: %v", w.Header())
				}
			} else if w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("unconfigured origin received CORS access")
			}
		})
	}
}
