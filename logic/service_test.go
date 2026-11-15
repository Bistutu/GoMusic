package logic

import (
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(r *http.Request, data any) *http.Response {
	body, _ := json.Marshal(data)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header), Request: r}
}
func testService(f roundTripFunc) *Service {
	return NewService(httputil.NewClient(&http.Client{Transport: f}), nil, nil, 10000, 3)
}

type failingRepo struct{}

func (failingRepo) Get(context.Context, string, []string) (map[string]models.Track, error) {
	return nil, errors.New("storage is offline")
}
func (failingRepo) Put(context.Context, string, []models.Track) error {
	return errors.New("storage is offline")
}

type memoryRepo struct {
	values map[string]models.Track
	writes int
}

func (m *memoryRepo) Get(_ context.Context, _ string, ids []string) (map[string]models.Track, error) {
	return m.values, nil
}
func (m *memoryRepo) Put(_ context.Context, _ string, tracks []models.Track) error {
	m.writes += len(tracks)
	return nil
}

func TestNetEaseCacheOutageDoesNotLoseSongs(t *testing.T) {
	var calls atomic.Int32
	s := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "playlist/detail") {
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "Long playlist", "trackCount": 1005, "trackIds": func() []map[string]int {
				ids := make([]map[string]int, 1005)
				for i := range ids {
					ids[i] = map[string]int{"id": i + 1}
				}
				return ids
			}()}}), nil
		}
		calls.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		var ids []map[string]string
		if err := json.Unmarshal([]byte(r.Form.Get("c")), &ids); err != nil {
			t.Fatal(err)
		}
		songs := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			n, _ := strconv.Atoi(id["id"])
			songs = append(songs, map[string]any{"id": n, "name": fmt.Sprintf("Song %d - Version", n), "ar": []map[string]string{{"name": "Artist"}}})
		}
		return response(r, map[string]any{"code": 200, "songs": songs}), nil
	})
	s.Cache = failingRepo{}
	s.Store = failingRepo{}
	list, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{Format: "singer-song"})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 1005 || list.TotalCount != 1005 || !list.Complete || calls.Load() != 3 {
		t.Fatalf("count=%d total=%d complete=%v calls=%d", list.SongsCount, list.TotalCount, list.Complete, calls.Load())
	}
	if list.Songs[0] != "Artist - Song 1 - Version" || list.Songs[1004] != "Artist - Song 1005 - Version" {
		t.Fatal("song order/title was corrupted")
	}
}
func TestDatabaseHitBackfillsRedisAndPreservesOriginal(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, "playlist/detail") {
			t.Fatal("unnecessary song detail request")
		}
		return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "test", "trackCount": 1, "trackIds": []map[string]int{{"id": 42}}}}), nil
	})
	cache := &memoryRepo{}
	s.Cache = cache
	s.Store = &memoryRepo{values: map[string]models.Track{"42": {ID: "42", Title: "Song【tag】", Artists: []string{"A"}}}}
	list, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if cache.writes != 1 || list.Songs[0] != "Song【tag】 - A" {
		t.Fatalf("cache writes=%d list=%v", cache.writes, list)
	}
}
func TestNetEaseMissingSongIsPartial(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "playlist/detail") {
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "test", "trackCount": 2, "trackIds": []map[string]int{{"id": 1}, {"id": 2}}}}), nil
		}
		return response(r, map[string]any{"code": 200, "songs": []map[string]any{{"id": 1, "name": "Song"}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Complete || list.SongsCount != 1 || list.TotalCount != 2 || len(list.Warnings) == 0 {
		t.Fatalf("missing song hidden: %#v", list)
	}
}
func TestQQPaginationIncludesLastPage(t *testing.T) {
	var begins []int
	s := testService(func(r *http.Request) (*http.Response, error) {
		var request models.QQMusicReq
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		p := request.Req0.Param
		begins = append(begins, p.SongBegin)
		songs := make([]map[string]any, 0, p.SongNum)
		for i := p.SongBegin; i < p.SongBegin+p.SongNum && i < 61; i++ {
			songs = append(songs, map[string]any{"id": i + 1, "name": fmt.Sprintf("Song %d", i+1), "singer": []map[string]string{{"name": "Artist"}}})
		}
		return response(r, map[string]any{"code": 0, "req_0": map[string]any{"code": 0, "data": map[string]any{"dirinfo": map[string]any{"title": "QQ test", "songnum": 61}, "songlist": songs}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://y.qq.com/n/ryqq_v2/playlist/1?share=1", Options{Order: "reverse"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(begins, []int{0, 30, 60}) || list.SongsCount != 61 || !list.Complete || list.Songs[0] != "Song 61 - Artist" {
		t.Fatalf("offsets=%v list=%#v", begins, list)
	}
}
func TestQQFallbackDoesNotHideTruncation(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			return response(r, map[string]int{"code": 2000}), nil
		}
		return response(r, map[string]any{"code": 0, "cdlist": []map[string]any{{"dissname": "Partial", "songnum": 400, "songlist": []map[string]any{{"songname": "Song", "singer": []map[string]string{{"name": "Artist"}}}}}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://y.qq.com/n/ryqq/playlist/1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 1 || list.TotalCount != 400 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("truncation hidden: %#v", list)
	}
}
func TestQQRepeatedPageFallsBack(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		songs := []map[string]any{{"id": 1, "name": "Song"}}
		if r.Method == http.MethodPost {
			return response(r, map[string]any{"code": 0, "req_0": map[string]any{"code": 0, "data": map[string]any{"dirinfo": map[string]any{"title": "test", "songnum": 2}, "songlist": songs}}}), nil
		}
		return response(r, map[string]any{"code": 0, "cdlist": []map[string]any{{"dissname": "Fallback", "songnum": 2, "songlist": []map[string]string{{"songname": "One"}, {"songname": "Two"}}}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://y.qq.com/n/ryqq/playlist/1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Fallback" || list.SongsCount != 2 || !list.Complete {
		t.Fatalf("bad fallback: %#v", list)
	}
}
func TestQishuiCursorPagination(t *testing.T) {
	var cursors []string
	s := testService(func(r *http.Request) (*http.Response, error) {
		var input struct {
			Cursor string `json:"cursor"`
		}
		json.NewDecoder(r.Body).Decode(&input)
		cursors = append(cursors, input.Cursor)
		next := "2"
		name := "One"
		if input.Cursor == "2" {
			next = "3"
			name = "Two"
		}
		return response(r, map[string]any{"playlist": map[string]any{"title": "汽水", "count_tracks": 2}, "next_cursor": next, "media_resources": []map[string]any{{"type": "track", "entity": map[string]any{"track": map[string]any{"id": input.Cursor, "name": name, "artists": []map[string]string{{"name": "Artist"}}}}}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://music.douyin.com/qishui/share/playlist?playlist_id=1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cursors, []string{"0", "2"}) || list.SongsCount != 2 || !list.Complete {
		t.Fatalf("cursors=%v list=%#v", cursors, list)
	}
}
func TestQishuiSSRAndMalformedPage(t *testing.T) {
	html := `<script>_ROUTER_DATA = {"loaderData":{"playlist_page":{"playlistInfo":{"title":"test","count_tracks":2},"medias":[{"type":"track","entity":{"track":{"id":"1","name":"歌 - 名","artists":[{"name":"歌手"}]}}}]}}}; window.untrustedCode();</script>`
	list, err := parseQishui(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tracks) != 1 || list.TotalCount != 2 || list.Tracks[0].Title != "歌 - 名" {
		t.Fatal(list)
	}
	if _, err := parseQishui(strings.NewReader(`<html>Login required</html>`)); err == nil {
		t.Fatal("blank page treated as empty playlist")
	}
}
func TestKugouPreviewAndArtistParsing(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		body := `<script>window.$output = {"info":{"listinfo":{"name":"Kugou","count":50},"songs":[{"hash":"x","name":"Artist - Song - Version","singerinfo":[{"name":"Artist"}]}]}};other();</script>`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})
	list, err := s.Fetch(context.Background(), "https://www.kugou.com/songlist/gcid_example12345/", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Songs[0] != "Song - Version - Artist" || list.TotalCount != 50 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("bad Kugou preview: %#v", list)
	}
}
func TestKugouCollectionPaginates(t *testing.T) {
	var pages []string
	s := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "info_v2") {
			return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Collection", "songcount": 101}}), nil
		}
		q := r.URL.Query()
		if q.Get("signature") == "" {
			t.Fatal("missing signature")
		}
		pages = append(pages, q.Get("page"))
		songs := make([]map[string]string, 0)
		begin := 0
		end := 100
		if q.Get("page") == "2" {
			begin = 100
			end = 101
		}
		for i := begin; i < end; i++ {
			songs = append(songs, map[string]string{"hash": strconv.Itoa(i), "name": fmt.Sprintf("Artist - Song %d", i)})
		}
		return response(r, map[string]any{"status": 1, "data": map[string]any{"count": 101, "info": songs}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://activity.kugou.com/share/index.html?global_specialid=collection_3_123_2_0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.SongsCount != 101 || !list.Complete {
		t.Fatalf("pages=%v count=%d complete=%v", pages, list.SongsCount, list.Complete)
	}
}
func TestNetEaseRequestsAreBoundedAndCancelable(t *testing.T) {
	var active, maximum atomic.Int32
	var mu sync.Mutex
	s := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "playlist/detail") {
			ids := make([]map[string]int, 2401)
			for i := range ids {
				ids[i] = map[string]int{"id": i + 1}
			}
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "test", "trackCount": len(ids), "trackIds": ids}}), nil
		}
		n := active.Add(1)
		defer active.Add(-1)
		mu.Lock()
		if n > maximum.Load() {
			maximum.Store(n)
		}
		mu.Unlock()
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	s.Concurrency = 2
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := s.Fetch(ctx, "https://music.163.com/playlist?id=1", Options{})
	if !errors.Is(err, context.DeadlineExceeded) || maximum.Load() > 2 {
		t.Fatalf("err=%v concurrent=%d", err, maximum.Load())
	}
}

func TestNetEaseReusesMetadataTracks(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, "playlist/detail") {
			t.Fatal("fetched a song already included in playlist metadata")
		}
		return response(r, map[string]any{"code": 200, "playlist": map[string]any{"name": "test", "trackCount": 1, "trackIds": []map[string]int{{"id": 1}}, "tracks": []map[string]any{{"id": 1, "name": "Fresh title", "ar": []map[string]string{{"name": "Artist"}}}}}}), nil
	})
	list, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{})
	if err != nil || list.SongsCount != 1 || !list.Complete {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func TestAlbumSupportsCurrentAndLegacyShapes(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		calls := 0
		s := testService(func(r *http.Request) (*http.Response, error) {
			calls++
			if legacy && strings.Contains(r.URL.Path, "/v1/") {
				return response(r, map[string]int{"code": -462}), nil
			}
			song := map[string]any{"id": 1, "name": "Album song", "artists": []map[string]string{{"name": "Artist"}}}
			album := map[string]any{"name": "Album", "size": 1}
			data := map[string]any{"code": 200, "album": album}
			if legacy {
				album["songs"] = []any{song}
			} else {
				data["songs"] = []any{song}
			}
			return response(r, data), nil
		})
		list, err := s.Fetch(context.Background(), "https://music.163.com/album?id=1", Options{})
		if err != nil || list == nil || !list.Complete || list.Songs[0] != "Album song - Artist" {
			t.Fatalf("legacy=%v list=%v err=%v", legacy, list, err)
		}
		if legacy && calls != 2 {
			t.Fatal("album fallback not used")
		}
	}
}
func TestMalformedProviderObjectsNeverBecomeCompleteEmptyPlaylists(t *testing.T) {
	for _, url := range []string{"https://music.163.com/playlist?id=1", "https://y.qq.com/n/ryqq/playlist/1", "https://music.douyin.com/qishui/share/playlist?playlist_id=1", "https://activity.kugou.com/share/index.html?global_specialid=collection_3_1_2_0"} {
		s := testService(func(r *http.Request) (*http.Response, error) {
			return response(r, map[string]any{"code": 200, "status": 1, "playlist": map[string]any{}, "data": map[string]any{}, "req_0": map[string]any{"code": 0, "data": map[string]any{}}}), nil
		})
		list, err := s.Fetch(context.Background(), url, Options{})
		if err == nil {
			t.Fatalf("malformed response accepted for %s: %#v", url, list)
		}
	}
}

func TestQishuiFallbackPreservesTimeoutCause(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "beta-luna.douyin.com" {
			return nil, context.DeadlineExceeded
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("<html>unavailable</html>")), Header: make(http.Header), Request: r}, nil
	})
	_, err := s.Fetch(context.Background(), "https://music.douyin.com/qishui/share/playlist?playlist_id=1", Options{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout cause lost through fallback: %v", err)
	}
}
