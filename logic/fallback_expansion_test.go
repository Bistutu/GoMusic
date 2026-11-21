package logic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestQishuiFallbackHostRotatesOnFailure(t *testing.T) {
	attempts := []string{}
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/luna/playlist/detail" {
			return nil, fmt.Errorf("unexpected Qishui path: %s", r.URL.Path)
		}
		attempts = append(attempts, r.URL.Host)
		if r.URL.Host == "beta-luna.douyin.com" {
			return nil, fmt.Errorf("primary host down")
		}
		return response(r, map[string]any{
			"playlist": map[string]any{"title": "Rotated list", "count_tracks": 1},
			"media_resources": []any{map[string]any{
				"type": "track",
				"entity": map[string]any{"track": map[string]any{
					"id": "1", "name": "Song", "artists": []any{map[string]string{"name": "Artist"}},
				}},
			}},
			"has_more": false,
		}), nil
	})
	list, err := service.Fetch(context.Background(), "https://music.douyin.com/qishui/share/playlist?playlist_id=1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Rotated list" || list.SongsCount != 1 || list.Tracks[0].Artists[0] != "Artist" {
		t.Fatalf("unexpected Qishui fallback result: %#v", list)
	}
	if len(attempts) < 2 || attempts[0] != "beta-luna.douyin.com" || attempts[1] != "api.qishui.com" {
		t.Fatalf("Qishui host rotation did not happen: %v", attempts)
	}
}

func TestMiguAlternateSongsEndpointRecoversList(t *testing.T) {
	alternateUsed := false
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/resource/playlist/v2.0":
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"title": "Migu list", "musicNum": 1}}), nil
		case "/MIGUM3.0/resource/playlist/song/v2.0":
			return nil, fmt.Errorf("v2 song route down")
		case "/MIGUM3.0/v1.0/user/queryMusicListSongs.do":
			alternateUsed = true
			return response(r, map[string]any{
				"code": "000000", "totalCount": 1,
				"list": []any{map[string]any{"contentId": "c1", "songName": "Migu song", "singer": "Migu artist"}},
			}), nil
		default:
			return nil, fmt.Errorf("unexpected Migu request: %s", r.URL)
		}
	})
	list, err := service.Fetch(context.Background(), "https://music.migu.cn/v3/music/playlist/1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !alternateUsed || list.Name != "Migu list" || list.SongsCount != 1 || list.TotalCount != 1 {
		t.Fatalf("Migu alternate route was not used correctly: %#v", list)
	}
	if list.Tracks[0].ID != "c1" || list.Tracks[0].Title != "Migu song" || list.Tracks[0].Artists[0] != "Migu artist" {
		t.Fatalf("Migu alternate row parsed incorrectly: %#v", list.Tracks[0])
	}
}

func TestQianqianSharePageRecoversWhenAPIRejects(t *testing.T) {
	page := `<script>window.__NUXT__=(function(a,b,c){return {layout:"default",data:[{pageData:{trackList:[` +
		`{artist:[{name:a}],title:b,assetId:"id-1",TSID:"id-1"},` +
		`{artist:[{name:c}],title:"Second song",assetId:"id-2",TSID:"id-2"}` +
		`],cateList:[1],title:"Nuxt list",trackCount:2}}]}}("Artist A","First song","Artist B"));</script>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/v1/tracklist/info" {
			return response(r, map[string]any{"state": false, "errno": 403, "errmsg": "blocked"}), nil
		}
		if r.URL.Host == "music.91q.com" && r.URL.Path == "/songlist/1" {
			return htmlResponse(r, page), nil
		}
		return nil, fmt.Errorf("unexpected Qianqian request: %s", r.URL)
	})
	list, err := service.Fetch(context.Background(), "https://music.91q.com/songlist/1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Nuxt list" || list.SongsCount != 2 || list.TotalCount != 2 {
		t.Fatalf("unexpected Qianqian share-page result: %#v", list)
	}
	if list.Tracks[0].Title != "First song" || list.Tracks[0].Artists[0] != "Artist A" || list.Tracks[1].Title != "Second song" || list.Tracks[1].Artists[0] != "Artist B" {
		t.Fatalf("Qianqian Nuxt variables were not resolved: %#v", list.Tracks)
	}
}

func TestBodianFallsBackToBrowserParameterProfile(t *testing.T) {
	var seenQueries []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "bd-api.kuwo.cn" {
			return nil, fmt.Errorf("unexpected Bodian host: %s", r.URL.Host)
		}
		switch {
		case strings.Contains(r.URL.Path, "/info/"):
			if r.URL.Query().Has("uid") {
				return &http.Response{StatusCode: http.StatusPaymentRequired, Body: io.NopCloser(strings.NewReader(`{"code":402}`)), Header: make(http.Header), Request: r}, nil
			}
			return response(r, map[string]any{"code": 200, "data": map[string]any{"name": "Bodian list", "musicCount": 1, "isPrivate": 0}}), nil
		case strings.HasSuffix(r.URL.Path, "/musicList"):
			seenQueries = append(seenQueries, r.URL.RawQuery)
			if r.URL.Query().Has("uid") {
				return &http.Response{StatusCode: http.StatusPaymentRequired, Body: io.NopCloser(strings.NewReader(`{"code":402}`)), Header: make(http.Header), Request: r}, nil
			}
			return response(r, map[string]any{"code": 200, "data": map[string]any{
				"total": 1, "list": []any{map[string]any{"id": "1", "name": "Bodian song", "artist": "Bodian artist"}},
			}}), nil
		default:
			return nil, fmt.Errorf("unexpected Bodian path: %s", r.URL.Path)
		}
	})
	list, err := service.Fetch(context.Background(), "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=1&source=5", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Bodian list" || list.SongsCount != 1 || list.Tracks[0].Title != "Bodian song" {
		t.Fatalf("unexpected Bodian fallback result: %#v", list)
	}
	if len(seenQueries) != 2 || strings.Contains(seenQueries[1], "uid=") {
		t.Fatalf("Bodian browser profile was not used: %v", seenQueries)
	}
}

func TestNetEaseSongDetailFallsBackToLegacyRoute(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v6/playlist/detail":
			return response(r, map[string]any{"code": 200, "playlist": map[string]any{
				"name": "NetEase list", "trackCount": 1, "trackIds": []any{map[string]any{"id": 1}},
			}}), nil
		case "/api/v3/song/detail":
			return response(r, map[string]any{"code": 500}), nil
		case "/api/song/detail":
			return response(r, map[string]any{"code": 200, "songs": []any{map[string]any{
				"id": 1, "name": "Legacy song", "ar": []any{map[string]string{"name": "Legacy artist"}},
			}}}), nil
		default:
			return nil, fmt.Errorf("unexpected NetEase request: %s", r.URL)
		}
	})
	list, err := service.Fetch(context.Background(), "https://music.163.com/playlist?id=1", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "NetEase list" || list.SongsCount != 1 || list.Tracks[0].Title != "Legacy song" || list.Tracks[0].Artists[0] != "Legacy artist" {
		t.Fatalf("unexpected NetEase song-detail fallback result: %#v", list)
	}
}
