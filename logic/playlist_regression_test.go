package logic

import (
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The upstream can return HTTP 200 while silently capping a song-detail batch.
// These cases reproduce the historical large-playlist reports without needing
// a mutable public playlist or relying on the upstream accepting oversized batches.
func TestNetEaseLongPlaylistWithCappedDetails(t *testing.T) {
	for _, tc := range []struct{ total, embedded, limit int }{
		{199, 0, 10000}, {200, 0, 10000}, {201, 0, 10000},
		{999, 5, 10000}, {1000, 5, 10000}, {1001, 5, 10000},
		{2800, 5, 10000}, {2800, 5, 1000},
	} {
		t.Run(fmt.Sprintf("songs_%d_limit_%d", tc.total, tc.limit), func(t *testing.T) {
			service := testService(func(r *http.Request) (*http.Response, error) {
				if strings.Contains(r.URL.Path, "playlist/detail") {
					ids := make([]map[string]int, tc.total)
					for i := range ids {
						ids[i] = map[string]int{"id": i + 1}
					}
					return response(r, map[string]any{"code": 200, "playlist": map[string]any{
						"name": "Long playlist", "trackCount": tc.total, "trackIds": ids,
						"tracks": regressionSongs("netease", 1, tc.embedded),
					}}), nil
				}
				if err := r.ParseForm(); err != nil {
					return nil, err
				}
				var ids []map[string]string
				if err := json.Unmarshal([]byte(r.Form.Get("c")), &ids); err != nil {
					return nil, err
				}
				if len(ids) > 200 {
					ids = ids[:200]
				}
				var songs []map[string]any
				for _, id := range ids {
					n, _ := strconv.Atoi(id["id"])
					songs = append(songs, regressionSongs("netease", n, 1)...)
				}
				return response(r, map[string]any{"code": 200, "songs": songs}), nil
			})
			service.MaxSongs = tc.limit
			list, err := service.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{Detailed: true})
			if err != nil {
				t.Fatal(err)
			}
			want := min(tc.total, tc.limit)
			if list.SongsCount != want || len(list.Songs) != want || list.TotalCount != tc.total || list.Complete != (want == tc.total) {
				t.Fatalf("got %d/%d complete=%v; want %d/%d", list.SongsCount, list.TotalCount, list.Complete, want, tc.total)
			}
			for i, track := range list.Tracks {
				if track.ID != strconv.Itoa(i+1) {
					t.Fatalf("lost order at %d: ID=%s", i, track.ID)
				}
			}
			if !list.Complete && len(list.Warnings) == 0 {
				t.Fatal("export limit was not disclosed")
			}
		})
	}
}

func TestQQLongPlaylistUsesActualPageLength(t *testing.T) {
	for _, total := range []int{30, 31, 60, 61, 200, 201, 999, 1000, 1001, 2800} {
		t.Run(strconv.Itoa(total), func(t *testing.T) {
			service := testService(func(r *http.Request) (*http.Response, error) {
				var input models.QQMusicReq
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					return nil, err
				}
				p := input.Req0.Param
				// A short page is not the end of the playlist when the reported
				// total is larger. Offsets must use records actually returned.
				count := min(p.SongNum, 17, total-p.SongBegin)
				return regressionQQPage(r, total, regressionSongs("qq", p.SongBegin+1, count)), nil
			})
			list, err := service.Fetch(context.Background(), "https://y.qq.com/n/ryqq_v2/playlist/1", Options{Order: "reverse"})
			if err != nil {
				t.Fatal(err)
			}
			if !list.Complete || list.SongsCount != total || list.TotalCount != total || len(list.Songs) != total {
				t.Fatalf("got %d/%d complete=%v; want %d", list.SongsCount, list.TotalCount, list.Complete, total)
			}
			for i, track := range list.Tracks {
				if track.ID != strconv.Itoa(total-i) {
					t.Fatalf("wrong reverse order at %d: %s", i, track.ID)
				}
			}
		})
	}
}

func TestFallbackDoesNotReduceKnownPlaylistTotal(t *testing.T) {
	for _, provider := range []string{"qq", "qishui", "kugou"} {
		for _, tc := range []struct{ primary, primaryTotal, fallback, fallbackTotal int }{
			{30, 2800, 60, 60},
			{60, 2800, 30, 5000},
			{30, 2800, 30, 5000},
			{30, 60, 60, 60},
		} {
			t.Run(fmt.Sprintf("%s/%d_of_%d_then_%d_of_%d", provider, tc.primary, tc.primaryTotal, tc.fallback, tc.fallbackTotal), func(t *testing.T) {
				service := testService(func(r *http.Request) (*http.Response, error) {
					switch provider {
					case "qq":
						if r.Method == http.MethodGet {
							return regressionQQLegacy(r, tc.fallbackTotal, regressionSongs("qq", 1, tc.fallback)), nil
						}
						var input models.QQMusicReq
						if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
							return nil, err
						}
						p := input.Req0.Param
						if p.SongBegin >= tc.primary {
							return response(r, map[string]int{"code": 2000}), nil
						}
						return regressionQQPage(r, tc.primaryTotal, regressionSongs("qq", p.SongBegin+1, min(p.SongNum, tc.primary-p.SongBegin))), nil
					case "qishui":
						if r.Method == http.MethodGet {
							return regressionHTML(r, "_ROUTER_DATA", map[string]any{"loaderData": map[string]any{"playlist_page": map[string]any{
								"playlistInfo": map[string]any{"title": "Preview", "count_tracks": tc.fallbackTotal},
								"medias":       regressionSongs("qishui", 1, tc.fallback),
							}}}), nil
						}
						var input struct {
							Cursor string `json:"cursor"`
						}
						if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
							return nil, err
						}
						if input.Cursor != "0" {
							return nil, fmt.Errorf("second page unavailable")
						}
						return regressionQishuiPage(r, tc.primaryTotal, regressionSongs("qishui", 1, tc.primary), "next", true), nil
					case "kugou":
						if strings.Contains(r.URL.Path, "/songlist/") {
							return regressionHTML(r, "window.$output", map[string]any{"info": map[string]any{
								"listinfo": map[string]any{"name": "Share", "count": tc.primaryTotal, "global_collection_id": "collection_3_123_2_0"},
								"songs":    regressionSongs("kugou", 1, tc.primary),
							}}), nil
						}
						if strings.Contains(r.URL.Path, "info_v2") {
							return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Collection", "songcount": tc.fallbackTotal}}), nil
						}
						if r.URL.Query().Get("page") != "1" {
							return nil, fmt.Errorf("second page unavailable")
						}
						return regressionKugouPage(r, tc.fallbackTotal, regressionSongs("kugou", 1, tc.fallback)), nil
					}
					return nil, fmt.Errorf("unexpected provider")
				})
				list, err := service.Fetch(context.Background(), regressionURL(provider), Options{})
				if err != nil {
					t.Fatal(err)
				}
				wantCount, wantTotal := max(tc.primary, tc.fallback), max(tc.primaryTotal, tc.fallbackTotal)
				if list.SongsCount != wantCount || list.TotalCount != wantTotal || list.Complete != (wantCount == wantTotal) {
					t.Fatalf("got %d/%d complete=%v; want %d/%d", list.SongsCount, list.TotalCount, list.Complete, wantCount, wantTotal)
				}
				if !list.Complete && len(list.Warnings) == 0 {
					t.Fatal("incomplete fallback has no warning")
				}
			})
		}
	}
}

func TestOverlappingPagesPreserveDistinctSameTitleSongs(t *testing.T) {
	for _, provider := range []string{"qq", "qishui", "kugou"} {
		t.Run(provider, func(t *testing.T) {
			service := testService(func(r *http.Request) (*http.Response, error) {
				page := 1
				switch provider {
				case "qq":
					if r.Method == http.MethodGet {
						return regressionQQLegacy(r, 4, regressionSongs("qq", 1, 4)), nil
					}
					var input models.QQMusicReq
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						return nil, err
					}
					page = input.Req0.Param.SongBegin/2 + 1
				case "qishui":
					var input struct {
						Cursor string `json:"cursor"`
					}
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						return nil, err
					}
					cursor, _ := strconv.Atoi(input.Cursor)
					page = cursor + 1
				case "kugou":
					if strings.Contains(r.URL.Path, "info_v2") {
						return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Collection", "songcount": 4}}), nil
					}
					page, _ = strconv.Atoi(r.URL.Query().Get("page"))
				}
				start, count := 1, 2
				if page == 2 {
					start = 2 // The second page overlaps the first by one stable song ID.
				} else if page == 3 {
					start, count = 4, 1
				} else if page > 3 {
					return nil, fmt.Errorf("pagination did not stop")
				}
				songs := regressionSongs(provider, start, count)
				// Identical titles with different IDs are separate valid songs.
				for _, song := range songs {
					switch provider {
					case "qq", "kugou":
						song["name"] = "同名歌曲"
					case "qishui":
						song["entity"].(map[string]any)["track"].(map[string]any)["name"] = "同名歌曲"
					}
				}
				switch provider {
				case "qq":
					return regressionQQPage(r, 4, songs), nil
				case "qishui":
					return regressionQishuiPage(r, 4, songs, strconv.Itoa(page), page < 3), nil
				default:
					return regressionKugouPage(r, 4, songs), nil
				}
			})
			playlistURL := regressionURL(provider)
			if provider == "kugou" {
				playlistURL = "https://activity.kugou.com/share/index.html?global_specialid=collection_3_123_2_0"
			}
			list, err := service.Fetch(context.Background(), playlistURL, Options{})
			if err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, track := range list.Tracks {
				ids = append(ids, track.ID)
			}
			if !list.Complete || list.TotalCount != 4 || !reflect.DeepEqual(ids, []string{"1", "2", "3", "4"}) {
				t.Fatalf("overlap counted as songs: ids=%v count=%d total=%d complete=%v", ids, list.SongsCount, list.TotalCount, list.Complete)
			}
		})
	}
}

func TestChangingRepeatedPagesDoNotInflateCounts(t *testing.T) {
	for _, provider := range []string{"qq", "qishui", "kugou"} {
		t.Run(provider, func(t *testing.T) {
			requests := 0
			service := testService(func(r *http.Request) (*http.Response, error) {
				if (provider == "qq" || provider == "qishui") && r.Method == http.MethodGet {
					return nil, fmt.Errorf("fallback unavailable")
				}
				if provider == "kugou" && strings.Contains(r.URL.Path, "info_v2") {
					return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Collection", "songcount": 5}}), nil
				}
				requests++
				if requests > 3 {
					return nil, fmt.Errorf("pagination kept requesting the same IDs")
				}
				songs := regressionSongs(provider, 1, 2)
				for _, song := range songs {
					if provider == "qishui" {
						song = song["entity"].(map[string]any)["track"].(map[string]any)
					}
					song["name"] = fmt.Sprintf("Updated title %d", requests)
				}
				switch provider {
				case "qq":
					return regressionQQPage(r, 5, songs), nil
				case "qishui":
					return regressionQishuiPage(r, 5, songs, strconv.Itoa(requests), true), nil
				default:
					return regressionKugouPage(r, 5, songs), nil
				}
			})
			playlistURL := regressionURL(provider)
			if provider == "kugou" {
				playlistURL = "https://activity.kugou.com/share/index.html?global_specialid=collection_3_123_2_0"
			}
			list, err := service.Fetch(context.Background(), playlistURL, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if list.SongsCount != 2 || list.TotalCount != 5 || list.Complete || len(list.Warnings) == 0 || requests > 2 {
				t.Fatalf("stalled pages inflated the result: %d/%d complete=%v requests=%d warnings=%v", list.SongsCount, list.TotalCount, list.Complete, requests, list.Warnings)
			}
		})
	}
}

func regressionURL(provider string) string {
	switch provider {
	case "qq":
		return "https://y.qq.com/n/ryqq_v2/playlist/1"
	case "qishui":
		return "https://music.douyin.com/qishui/share/playlist?playlist_id=1"
	default:
		return "https://www.kugou.com/songlist/gcid_example12345/"
	}
}

func regressionSongs(provider string, start, count int) []map[string]any {
	songs := make([]map[string]any, 0, count)
	for i := start; i < start+count; i++ {
		song := map[string]any{"id": i, "name": fmt.Sprintf("Song %d", i)}
		switch provider {
		case "netease":
			song["ar"] = []map[string]string{{"name": "Artist"}}
		case "qq":
			song["songid"], song["songname"] = i, song["name"]
			song["singer"] = []map[string]string{{"name": "Artist"}}
		case "qishui":
			song["id"] = strconv.Itoa(i)
			song["artists"] = []map[string]string{{"name": "Artist"}}
			song = map[string]any{"type": "track", "entity": map[string]any{"track": song}}
		case "kugou":
			song["hash"], song["singername"] = strconv.Itoa(i), "Artist"
		}
		songs = append(songs, song)
	}
	return songs
}

func regressionQQPage(r *http.Request, total int, songs []map[string]any) *http.Response {
	return response(r, map[string]any{"code": 0, "req_0": map[string]any{"code": 0, "data": map[string]any{
		"dirinfo": map[string]any{"title": "QQ playlist", "songnum": total}, "songlist": songs,
	}}})
}
func regressionQQLegacy(r *http.Request, total int, songs []map[string]any) *http.Response {
	return response(r, map[string]any{"code": 0, "cdlist": []map[string]any{{"dissname": "QQ fallback", "songnum": total, "songlist": songs}}})
}
func regressionQishuiPage(r *http.Request, total int, songs []map[string]any, next string, more bool) *http.Response {
	return response(r, map[string]any{"playlist": map[string]any{"title": "Qishui playlist", "count_tracks": total}, "media_resources": songs, "next_cursor": next, "has_more": more})
}
func regressionKugouPage(r *http.Request, total int, songs []map[string]any) *http.Response {
	return response(r, map[string]any{"status": 1, "data": map[string]any{"count": total, "info": songs}})
}
func regressionHTML(r *http.Request, variable string, value any) *http.Response {
	data, _ := json.Marshal(value)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("<script>" + variable + "=" + string(data) + ";</script>")), Header: make(http.Header), Request: r}
}
