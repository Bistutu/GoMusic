package logic

import (
	"GoMusic/repo/cache"
	"GoMusic/repo/db"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This uses real Redis/MySQL with a deterministic upstream, so each fallback can be proven.
func TestPersistentCacheFallbackIntegration(t *testing.T) {
	dsn := os.Getenv("GOMUSIC_INTEGRATION_MYSQL_DSN")
	addr := os.Getenv("GOMUSIC_INTEGRATION_REDIS_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("set both GOMUSIC_INTEGRATION_* storage addresses")
	}
	ctx := context.Background()
	sqlStore, err := db.Open(ctx, dsn, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlStore.Close()
	redisStore, err := cache.Open(ctx, addr, os.Getenv("GOMUSIC_INTEGRATION_REDIS_PASSWORD"), 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer redisStore.Close()
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	cleanup, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close()
	defer cleanup.Exec("DELETE FROM song_metadata WHERE provider = ? AND id = ?", "netease", id)
	detailRequests := 0
	service := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "playlist/detail") {
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "Cache test", "trackCount": 1, "trackIds": []map[string]string{{"id": id}}}}), nil
		}
		detailRequests++
		return response(r, map[string]any{"code": 200, "songs": []map[string]any{{"id": id, "name": "Original【tag】", "ar": []map[string]string{{"name": "Artist - Name"}}}}}), nil
	})
	service.Cache = redisStore
	service.Store = sqlStore
	fetch := func() {
		t.Helper()
		list, err := service.Fetch(ctx, "https://music.163.com/playlist?id=1", Options{Detailed: true})
		if err != nil || list == nil || len(list.Songs) != 1 || list.Songs[0] != "Original【tag】 - Artist - Name" {
			t.Fatalf("result=%v error=%v", list, err)
		}
	}
	fetch()
	if detailRequests != 1 {
		t.Fatal("initial request did not populate stores")
	}
	fetch()
	if detailRequests != 1 {
		t.Fatal("Redis cache miss after initial write")
	}
	redisStore.Close()
	fetch()
	if detailRequests != 1 {
		t.Fatal("MySQL did not serve the Redis failure")
	}
	sqlStore.Close()
	fetch()
	if detailRequests != 2 {
		t.Fatal("upstream did not serve both cache failures")
	}
	t.Log(fmt.Sprintf("real storage fallback verified; upstream detail requests=%d", detailRequests))
}
