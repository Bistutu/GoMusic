package logic

import (
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestQishuiFallbackRetriesUnusableAPIPage(t *testing.T) {
	for _, failure := range []string{"host timeout", "invalid JSON", "missing playlist", "missing rows"} {
		t.Run(failure, func(t *testing.T) {
			var attempts []string
			service := testService(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.Path != "/luna/playlist/detail" {
					return nil, fmt.Errorf("unexpected share-page request")
				}
				var input struct {
					ID     string `json:"playlist_id"`
					Cursor string `json:"cursor"`
				}
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ID != "1" {
					t.Fatalf("bad request payload: %#v, %v", input, err)
				}
				attempts = append(attempts, r.URL.Host+"/"+input.Cursor)
				if r.URL.Host == "beta-luna.douyin.com" {
					switch failure {
					case "host timeout":
						return nil, context.DeadlineExceeded
					case "invalid JSON":
						return htmlResponse(r, "<html>temporary error</html>"), nil
					case "missing playlist":
						return response(r, map[string]int{"status_code": 1}), nil
					default:
						return regressionQishuiPage(r, 2, nil, "", false), nil
					}
				}
				if input.Cursor == "0" {
					return regressionQishuiPage(r, 2, regressionSongs("qishui", 1, 1), "next", true), nil
				}
				return regressionQishuiPage(r, 2, regressionSongs("qishui", 2, 1), "", false), nil
			})
			list, err := service.Fetch(context.Background(), regressionURL("qishui"), Options{})
			wantAttempts := []string{"beta-luna.douyin.com/0", "api.qishui.com/0", "api.qishui.com/next"}
			if err != nil || !list.Complete || list.SongsCount != 2 || !reflect.DeepEqual(attempts, wantAttempts) {
				t.Fatalf("list=%#v err=%v attempts=%v", list, err, attempts)
			}
		})
	}
}

func TestNetEaseHonorsExplicitPlaylistDenial(t *testing.T) {
	for _, tt := range []struct {
		name      string
		transport bool
		legacy    bool
	}{
		{name: "v6 business denial"},
		{name: "v6 HTTP denial", transport: true},
		{name: "legacy business denial", legacy: true},
		{name: "legacy HTTP denial", transport: true, legacy: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var attempts []string
			service := testService(func(r *http.Request) (*http.Response, error) {
				attempts = append(attempts, r.URL.Path)
				if tt.legacy && r.URL.Path == "/api/v6/playlist/detail" {
					return nil, fmt.Errorf("v6 temporarily offline")
				}
				res := response(r, map[string]int{"code": 403})
				if tt.transport {
					res.StatusCode = http.StatusForbidden
				}
				return res, nil
			})
			list, err := service.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{})
			wantAttempts := []string{"/api/v6/playlist/detail"}
			if tt.legacy {
				wantAttempts = append(wantAttempts, "/api/playlist/detail")
			}
			if list != nil || !errors.Is(err, ErrPrivatePlaylist) || !reflect.DeepEqual(attempts, wantAttempts) {
				t.Fatalf("list=%#v err=%v attempts=%v", list, err, attempts)
			}
		})
	}
}

func TestNetEasePartialSongMetadataIsCompletedWithoutReplacingPrimaryRows(t *testing.T) {
	var legacyIDs string
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v6/playlist/detail":
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{
				"name": "NetEase list", "trackCount": 2, "trackIds": []map[string]int{{"id": 1}, {"id": 2}},
			}}), nil
		case "/api/v3/song/detail":
			return response(r, map[string]any{"code": 200, "songs": []map[string]any{{"id": 1, "name": "Primary title"}}}), nil
		case "/api/song/detail":
			legacyIDs = r.URL.Query().Get("ids")
			return response(r, map[string]any{"code": 200, "songs": []map[string]any{
				{"id": 1, "name": "Stale title"}, {"id": 2, "name": "Recovered title"},
			}}), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s", r.URL.Path)
		}
	})
	list, err := service.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{})
	if err != nil || !list.Complete || !reflect.DeepEqual(list.Songs, []string{"Primary title", "Recovered title"}) || legacyIDs != `["2"]` {
		t.Fatalf("list=%#v err=%v legacy ids=%q", list, err, legacyIDs)
	}
}

func TestQQMusicKeepsPartialPlainFallback(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "u.y.qq.com" {
			return nil, fmt.Errorf("other endpoint unavailable")
		}
		var req models.QQMusicReq
		if err := json.Unmarshal([]byte(r.URL.Query().Get("data")), &req); err != nil {
			t.Fatal(err)
		}
		if req.Req0.Param.SongBegin > 0 {
			return nil, fmt.Errorf("second page unavailable")
		}
		return regressionQQPage(r, 4, regressionSongs("qq", 1, 2)), nil
	})
	list, err := service.Fetch(context.Background(), regressionURL("qq"), Options{})
	if err != nil || list.SongsCount != 2 || list.TotalCount != 4 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestQQV8RetriesPartialFirstHostWithoutInflatingTrackCount(t *testing.T) {
	var hosts []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		hosts = append(hosts, r.URL.Host)
		rows := []map[string]any{{"songid": 1, "songname": "Same name"}, {"songid": 1, "songname": "Same name"}}
		if r.URL.Host == "i.y.qq.com" {
			rows = append(rows, map[string]any{"songid": 2, "songname": "Same name"})
		}
		return response(r, map[string]any{"data": map[string]any{"cdlist": []map[string]any{
			{"dissname": "QQ list", "songnum": 2, "songlist": rows},
		}}}), nil
	})
	list, err := service.qqV8Playlist(context.Background(), "1")
	if err != nil || len(list.Tracks) != 2 || list.Tracks[0].ID != "1" || list.Tracks[1].ID != "2" ||
		!reflect.DeepEqual(hosts, []string{"c.y.qq.com", "i.y.qq.com"}) {
		t.Fatalf("list=%#v err=%v hosts=%v", list, err, hosts)
	}
}

func TestQQQzoneAcceptsPlainJSON(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		return regressionQQLegacy(r, 1, regressionSongs("qq", 1, 1)), nil
	})
	list, err := service.qqQzoneMusic(context.Background(), "1")
	if err != nil || len(list.Tracks) != 1 || list.Tracks[0].ID != "1" {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestMiguFallbackRetriesUnusablePage(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/resource/playlist/v2.0":
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"title": "Migu list", "musicNum": 1}}), nil
		case "/MIGUM3.0/resource/playlist/song/v2.0":
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"totalCount": 1}}), nil
		case "/MIGUM3.0/v1.0/user/queryMusicListSongs.do":
			return response(r, map[string]any{"code": "000000", "totalCount": 1,
				"list": []map[string]string{{"contentId": "track-1", "songName": "Recovered"}},
			}), nil
		default:
			return nil, fmt.Errorf("unexpected path: %s", r.URL.Path)
		}
	})
	list, err := service.Fetch(context.Background(), "https://music.migu.cn/v3/music/playlist/1", Options{})
	if err != nil || !list.Complete || list.SongsCount != 1 || list.Tracks[0].ID != "track-1" {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestBodianFallbackRetriesUnusableData(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Has("uid") {
			return response(r, map[string]any{"code": 200, "data": nil}), nil
		}
		if strings.Contains(r.URL.Path, "/info/") {
			return response(r, map[string]any{"code": 200, "data": map[string]any{"name": "Bodian list", "musicCount": 1, "isPrivate": 0}}), nil
		}
		return response(r, map[string]any{"code": 200, "data": map[string]any{
			"total": 1, "list": []map[string]string{{"id": "track-1", "name": "Recovered"}},
		}}), nil
	})
	list, err := service.Fetch(context.Background(), "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=1&source=5", Options{})
	if err != nil || !list.Complete || list.Name != "Bodian list" || list.SongsCount != 1 || list.Tracks[0].ID != "track-1" {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestBodianKeepsPageSizeAtExportLimit(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/info/") {
			return response(r, map[string]any{"code": 200, "data": map[string]any{"name": "Bodian list", "musicCount": 200, "isPrivate": 0}}), nil
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("pn"))
		size, _ := strconv.Atoi(r.URL.Query().Get("rn"))
		start := (page - 1) * size
		rows := make([]map[string]string, 0, size)
		for i := start + 1; i <= start+size; i++ {
			rows = append(rows, map[string]string{"id": strconv.Itoa(i), "name": "Track"})
		}
		return response(r, map[string]any{"code": 200, "data": map[string]any{"total": 200, "list": rows}}), nil
	})
	service.MaxSongs = 150
	list, err := service.Fetch(context.Background(), "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=1&source=5", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 150 || list.TotalCount != 200 || list.Complete || list.Tracks[149].ID != "150" {
		t.Fatalf("fetched=%d total=%d complete=%v", len(list.Tracks), list.TotalCount, list.Complete)
	}
}

func TestQianqianSharePageReadsOnlyPlaylistFields(t *testing.T) {
	page := `<script>window.__NUXT__=(function(a,b){return {data:[{
		sidebar:{trackList:[{id:8,title:"Recommendation"}]},
		pageData:{id:1,title:a,trackCount:b,trackList:[{
			album:{title:"Album"}, artist:[{name:"A, [name]"}], title:"Track \uD83D\uDE00", id:9
		}]}
	}]}}("Playlist",4));</script>`
	list, err := parseQianqianSharePage([]byte(page))
	if err != nil || list.Name != "Playlist" || list.TotalCount != 4 || len(list.Tracks) != 1 ||
		list.Tracks[0].ID != "9" || list.Tracks[0].Title != "Track 😀" || !reflect.DeepEqual(list.Tracks[0].Artists, []string{"A, [name]"}) {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestQianqianKeepsBetterSignedResult(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Query().Get("pageNo") {
		case "1":
			return response(r, map[string]any{"state": true, "data": map[string]any{
				"title": "Playlist", "trackCount": 5, "haveMore": true,
				"trackList": []map[string]string{{"TSID": "1", "title": "One"}, {"TSID": "2", "title": "Two"}},
			}}), nil
		case "2":
			return nil, fmt.Errorf("page unavailable")
		default:
			return htmlResponse(r, `<script>window.__NUXT__={data:[{pageData:{title:"Playlist",trackCount:8,trackList:[{id:1,title:"One"}]}}]};</script>`), nil
		}
	})
	list, err := service.Fetch(context.Background(), "https://music.91q.com/songlist/1", Options{})
	if err != nil || list.SongsCount != 2 || list.TotalCount != 8 || list.Complete ||
		strings.Contains(strings.Join(list.Warnings, " "), "从千千歌单分享页恢复曲目") {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestDeezerMalformedEmbeddedTracksDoNotBlockNormalPlaylist(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/playlist/1" {
			return response(r, map[string]any{"id": 1, "public": true, "nb_tracks": 1, "tracks": "unexpected format"}), nil
		}
		return deezerTrackPageResponse(r, 1, 7, 1, ""), nil
	})
	list, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/1", Options{})
	if err != nil || list.SongsCount != 1 || !list.Complete || list.Tracks[0].ID != "7" {
		t.Fatalf("list=%#v err=%v", list, err)
	}
}

func TestDeezerRecoversFromEmbeddedPageWhenDedicatedPageIsUnusable(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "API error", "empty data", "invalid total"} {
		t.Run(failure, func(t *testing.T) {
			service := testService(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/playlist/1" {
					return response(r, map[string]any{
						"id": 1, "title": "Deezer list", "public": true, "nb_tracks": 4,
						"tracks": map[string]any{"total": 5, "data": deezerRows(7, 2)},
					}), nil
				}
				switch failure {
				case "invalid JSON":
					return htmlResponse(r, "<html>temporarily unavailable</html>"), nil
				case "API error":
					return response(r, map[string]any{"error": map[string]int{"code": 700}}), nil
				case "empty data":
					return response(r, map[string]any{"total": 4, "data": []any{}}), nil
				default:
					return response(r, map[string]any{"total": "unknown", "data": deezerRows(7, 2)}), nil
				}
			})
			list, err := service.Fetch(context.Background(), "https://www.deezer.com/playlist/1", Options{})
			if err != nil || list.Complete || list.TotalCount != 5 || list.SongsCount != 2 || len(list.Warnings) == 0 {
				t.Fatalf("list=%#v err=%v", list, err)
			}
		})
	}
}
