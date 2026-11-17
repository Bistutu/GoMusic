package logic

import (
	"GoMusic/misc/httputil"
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestLiveKuwoAlternateFallbackPreservesKnownTotal(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "www.kuwo.cn" && request.URL.Path == "/newh5/playlist/content" {
			return nil, errors.New("simulate an unreadable public H5 page")
		}
		return http.DefaultTransport.RoundTrip(request)
	})
	service := NewService(httputil.NewClient(&http.Client{Transport: transport}), nil, nil, 10000, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list, err := service.Fetch(ctx, "https://www.kuwo.cn/playlist_detail/2267788529", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("fetched=%d reported=%d complete=%v warnings=%v", list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
	if list.SongsCount == 0 || list.SongsCount != len(list.Tracks) || list.TotalCount < list.SongsCount {
		t.Fatalf("invalid fallback counts: %#v", list)
	}
	if list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("fallback result must remain explicitly partial: %#v", list)
	}
}

func TestLiveKuwoLargeShareFetchesContinuation(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, "https://www.kuwo.cn/newh5/playlist/content?pid=2267788529", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
	if list.SongsCount < 50 || list.SongsCount != list.TotalCount {
		t.Fatalf("invalid Kuwo counts: fetched=%d reported=%d", list.SongsCount, list.TotalCount)
	}
	if !list.Complete || len(list.Warnings) != 0 {
		t.Fatalf("Kuwo public continuation did not produce a complete result: %#v", list)
	}
}

func TestLiveKuwoMultiPagePlaylistOver60(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	var pages []string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "www.kuwo.cn" && request.URL.Path == "/newh5/playlist/playlistByPage" {
			pages = append(pages, request.URL.Query().Get("pn"))
		}
		return http.DefaultTransport.RoundTrip(request)
	})
	service := NewService(httputil.NewClient(&http.Client{Transport: transport}), nil, nil, 10000, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list, err := service.Fetch(ctx, "https://www.kuwo.cn/playlist_detail/3135508477", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name=%q fetched=%d reported=%d complete=%v continuation_pages=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, pages, list.Warnings)
	if list.SongsCount <= 60 || list.SongsCount != list.TotalCount || !list.Complete || len(pages) < 2 {
		t.Fatalf("Kuwo multi-page playlist is incomplete: tracks=%d total=%d complete=%v pages=%v warnings=%v", list.SongsCount, list.TotalCount, list.Complete, pages, list.Warnings)
	}
	for index, page := range pages {
		if page != strconv.Itoa(index+1) {
			t.Fatalf("Kuwo continuation pages are not sequential: %v", pages)
		}
	}
}

func TestLiveKuwoOfficialMobileShareURL(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	shareURL := "https://m.kuwo.cn/newh5app/playlist_detail/3527541442?t=plantform&from=ar"
	list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, shareURL, Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
	if list.Name == "" || list.SongsCount == 0 || list.SongsCount != len(list.Tracks) || list.TotalCount < list.SongsCount {
		t.Fatalf("Kuwo mobile share did not produce valid playlist rows: %#v", list)
	}
}

func TestLiveKuwoOfficialWebShareURL(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	shareURL := "https://www.kuwo.cn/playlist_detail/3527541442"
	list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, shareURL, Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
	if list.Name == "" || list.SongsCount == 0 || list.SongsCount != len(list.Tracks) || list.TotalCount < list.SongsCount {
		t.Fatalf("Kuwo official web share did not produce valid playlist rows: %#v", list)
	}
	if !list.Complete {
		t.Fatalf("Kuwo official web share did not produce a complete list: %#v", list)
	}
}
