package logic

import (
	"GoMusic/misc/httputil"
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestLiveDeezerPublicPlaylistPagination(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	var indexes []string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "api.deezer.com" && request.URL.Path == "/playlist/6399367984/tracks" {
			indexes = append(indexes, request.URL.Query().Get("index"))
		}
		return http.DefaultTransport.RoundTrip(request)
	})
	service := NewService(httputil.NewClient(&http.Client{Transport: transport}), nil, nil, 10000, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list, err := service.Fetch(ctx, "https://www.deezer.com/playlist/6399367984", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name=%q fetched=%d reported=%d complete=%v page_indexes=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, indexes, list.Warnings)
	if list.SongsCount <= 100 || list.SongsCount != list.TotalCount || !list.Complete || len(indexes) < 2 || len(list.Warnings) != 0 {
		t.Fatalf("Deezer public playlist was not fully retrieved: %#v indexes=%v", list, indexes)
	}
	for _, track := range list.Tracks {
		if track.ID == "" {
			t.Fatalf("Deezer track is missing a stable ID: %#v", track)
		}
	}
	for index, pageIndex := range indexes {
		if pageIndex != strconv.Itoa(index*deezerPageSize) {
			t.Fatalf("Deezer next links were not sequential: %v", indexes)
		}
	}
}
