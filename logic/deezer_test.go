package logic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func deezerTrackPageResponse(r *http.Request, total, first, count int, next string) *http.Response {
	data := map[string]any{"total": total, "data": deezerRows(first, count)}
	if next != "" {
		data["next"] = next
	}
	return response(r, data)
}

func deezerRows(first, count int) []any {
	rows := make([]any, 0, count)
	for id := first; id < first+count; id++ {
		rows = append(rows, map[string]any{
			"id":    id,
			"title": fmt.Sprintf("Deezer Song %d", id),
			"artist": map[string]any{
				"name": fmt.Sprintf("Deezer Artist %d", id),
			},
		})
	}
	return rows
}

func TestDeezerPlaylistFollowsNextPagesPreservingOrder(t *testing.T) {
	var indexes []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.deezer.com" {
			return nil, fmt.Errorf("unexpected Deezer host: %s", r.URL.Host)
		}
		switch r.URL.Path {
		case "/playlist/123":
			return response(r, map[string]any{"id": 123, "title": "Test Deezer playlist", "public": true, "nb_tracks": 201}), nil
		case "/playlist/123/tracks":
			index := r.URL.Query().Get("index")
			indexes = append(indexes, index)
			switch index {
			case "0":
				return deezerTrackPageResponse(r, 201, 1, 100, "https://api.deezer.com/playlist/123/tracks?limit=100&index=100"), nil
			case "100":
				return deezerTrackPageResponse(r, 201, 101, 100, "https://api.deezer.com/playlist/123/tracks?limit=100&index=200"), nil
			case "200":
				return deezerTrackPageResponse(r, 201, 201, 1, ""), nil
			default:
				return nil, fmt.Errorf("unexpected Deezer index: %s", index)
			}
		default:
			return nil, fmt.Errorf("unexpected Deezer path: %s", r.URL.Path)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.deezer.com/en/playlist/123", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexes, []string{"0", "100", "200"}) {
		t.Fatalf("unexpected Deezer page indexes: %v", indexes)
	}
	if list.Name != "Test Deezer playlist" || list.SongsCount != 201 || list.TotalCount != 201 || !list.Complete || len(list.Warnings) != 0 {
		t.Fatalf("unexpected Deezer result: %#v", list)
	}
	if list.Tracks[0].ID != "1" || list.Tracks[99].ID != "100" || list.Tracks[100].ID != "101" || list.Tracks[200].ID != "201" {
		t.Fatalf("Deezer pagination lost source order: %#v %#v %#v", list.Tracks[0], list.Tracks[100], list.Tracks[200])
	}
}

func TestDeezerPreservesRepeatedPlaylistEntries(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/playlist/124" {
			return response(r, map[string]any{"id": 124, "title": "Repeated songs", "public": true, "nb_tracks": 3}), nil
		}
		if r.URL.Path == "/playlist/124/tracks" && r.URL.Query().Get("index") == "0" {
			rows := []any{
				map[string]any{"id": 7, "title": "Repeat", "artist": map[string]any{"name": "Artist"}},
				map[string]any{"id": 7, "title": "Repeat", "artist": map[string]any{"name": "Artist"}},
				map[string]any{"id": 8, "title": "Next", "artist": map[string]any{"name": "Artist"}},
			}
			return response(r, map[string]any{"total": 3, "data": rows}), nil
		}
		return nil, fmt.Errorf("unexpected Deezer request: %s", r.URL)
	})

	list, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/124", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 3 || list.TotalCount != 3 || !list.Complete || list.Tracks[0].ID != "7" || list.Tracks[1].ID != "7" || list.Tracks[2].ID != "8" {
		t.Fatalf("Deezer repeat entries were changed: %#v", list)
	}
}

func TestDeezerRejectsPrivatePlaylist(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/playlist/125" {
			return nil, fmt.Errorf("unexpected Deezer request: %s", r.URL)
		}
		return response(r, map[string]any{"id": 125, "title": "Private playlist", "public": false, "nb_tracks": 1}), nil
	})

	_, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/125", Options{})
	if !errors.Is(err, ErrPrivatePlaylist) {
		t.Fatalf("private Deezer playlist was not rejected: %v", err)
	}
}

func TestDeezerRejectsUntrustedNextLinkAndKeepsPartialRows(t *testing.T) {
	var trackPages int
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.deezer.com" {
			return nil, fmt.Errorf("unexpected Deezer host: %s", r.URL.Host)
		}
		if r.URL.Path == "/playlist/126" {
			return response(r, map[string]any{"id": 126, "title": "Partial playlist", "public": true, "nb_tracks": 3}), nil
		}
		if r.URL.Path == "/playlist/126/tracks" {
			trackPages++
			return response(r, map[string]any{
				"total": 3,
				"next":  "https://example.test/playlist/126/tracks?limit=100&index=2",
				"data":  deezerRows(1, 2),
			}), nil
		}
		return nil, fmt.Errorf("unexpected Deezer path: %s", r.URL.Path)
	})

	list, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/126", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if trackPages != 1 || list.SongsCount != 2 || list.TotalCount != 3 || list.Complete {
		t.Fatalf("untrusted Deezer next link was followed or misreported: pages=%d list=%#v", trackPages, list)
	}
	if !strings.Contains(strings.Join(list.Warnings, " "), "分页链接无效") {
		t.Fatalf("invalid next-link warning is missing: %#v", list.Warnings)
	}
}

func TestDeezerExportLimitRemainsPartial(t *testing.T) {
	var indexes []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/playlist/127" {
			return response(r, map[string]any{"id": 127, "title": "Large Deezer playlist", "public": true, "nb_tracks": 490}), nil
		}
		if r.URL.Path != "/playlist/127/tracks" {
			return nil, fmt.Errorf("unexpected Deezer request: %s", r.URL)
		}
		index := r.URL.Query().Get("index")
		indexes = append(indexes, index)
		start, _ := strconv.Atoi(index)
		return deezerTrackPageResponse(r, 490, start+1, 100, fmt.Sprintf("https://api.deezer.com/playlist/127/tracks?limit=100&index=%d", start+100)), nil
	})
	service.MaxSongs = 150

	list, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/127", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexes, []string{"0", "100"}) || list.SongsCount != 150 || list.TotalCount != 490 || list.Complete {
		t.Fatalf("Deezer export limit was not preserved: pages=%v list=%#v", indexes, list)
	}
	if !strings.Contains(strings.Join(list.Warnings, " "), "达到本次导出上限 150 首") {
		t.Fatalf("Deezer export cap warning is missing: %#v", list.Warnings)
	}
}
