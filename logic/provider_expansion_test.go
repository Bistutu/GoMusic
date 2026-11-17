package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func htmlResponse(r *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    r,
	}
}

func kuwoShareRow(id int) string {
	return fmt.Sprintf(`<li class="singBox" mid="%d"><span class="singTexUp2">Song %d</span><span class="singName">Artist - Song %d</span></li>`, id, id, id)
}

func kuwoShareRows(first, count int) string {
	var rows strings.Builder
	for id := first; id < first+count; id++ {
		rows.WriteString(kuwoShareRow(id))
	}
	return rows.String()
}

func TestKuwoLoadMoreAndReportedTotalStayPartial(t *testing.T) {
	page := `<html><div class="titleCenter">Kuwo list</div><ul id="messDivId">
	<li class="singBox" mid="101"><span class="singTexUp2">First</span><span class="singName">Artist - First</span></li>
	<li class="singBox" mid="102"><span class="singTexUp2">Second</span><span class="singName">Artist - Second</span></li>
	</ul><button id="morebtnId" onclick="loadMore()">获取更多</button></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content" {
			return htmlResponse(r, page), nil
		}
		if r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/playlistByPage" {
			if r.URL.Query().Get("id") != "7" || r.URL.Query().Get("pn") != "1" || r.URL.Query().Get("rn") != "30" {
				return nil, fmt.Errorf("unexpected Kuwo share page query: %s", r.URL.RawQuery)
			}
			return htmlResponse(r, kuwoShareRow(103)), nil
		}
		if r.URL.Host == "nplserver.kuwo.cn" {
			return response(r, map[string]any{"result": "ok", "title": "Kuwo list", "total": 4, "musiclist": []any{}}), nil
		}
		return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
	})

	list, err := service.Fetch(context.Background(), "https://m.kuwo.cn/newh5/playlist/content?pid=7", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Kuwo list" || list.SongsCount != 3 || list.TotalCount != 4 || list.Complete {
		t.Fatalf("unexpected Kuwo result: %#v", list)
	}
	if list.Tracks[0].ID != "101" || list.Tracks[0].Artists[0] != "Artist" || list.Tracks[2].ID != "103" || len(list.Warnings) == 0 {
		t.Fatalf("missing order, artist, or partial warning: %#v", list)
	}
}

func TestKuwoSharePaginationPreservesOrderAndDeduplicates(t *testing.T) {
	var pages []string
	page := `<html><div class="titleCenter">Paged Kuwo list</div><ul id="messDivId">` + kuwoShareRows(100, 30) + `</ul><button id="morebtnId" onclick="loadMore()">获取更多</button></html>`
	pageOne := kuwoShareRow(129) + kuwoShareRows(130, 29)
	pageTwo := kuwoShareRows(159, 20)
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content":
			return htmlResponse(r, page), nil
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/playlistByPage":
			if r.Method != http.MethodGet || r.URL.Query().Get("id") != "7" || r.URL.Query().Get("rn") != "30" {
				return nil, fmt.Errorf("unexpected Kuwo share pagination request: %s", r.URL)
			}
			pageNo := r.URL.Query().Get("pn")
			pages = append(pages, pageNo)
			switch pageNo {
			case "1":
				return htmlResponse(r, pageOne), nil
			case "2":
				return htmlResponse(r, pageTwo), nil
			default:
				return nil, fmt.Errorf("unexpected Kuwo page number: %s", pageNo)
			}
		case r.URL.Host == "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Paged Kuwo list", "total": 79, "musiclist": []any{}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/7", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.Name != "Paged Kuwo list" || list.SongsCount != 79 || list.TotalCount != 79 || !list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
	if list.Tracks[29].ID != "129" || list.Tracks[30].ID != "130" || list.Tracks[78].ID != "178" {
		t.Fatalf("Kuwo continuation lost stable order or duplicate filtering: first=%#v next=%#v last=%#v", list.Tracks[29], list.Tracks[30], list.Tracks[78])
	}
}

func TestKuwoSharePaginationStopsAfterEmptyPage(t *testing.T) {
	var pages []string
	page := `<html><div class="titleCenter">Exact-page Kuwo list</div><ul id="messDivId">` + kuwoShareRows(100, 30) + `</ul><button id="morebtnId" onclick="loadMore()">获取更多</button></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content":
			return htmlResponse(r, page), nil
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/playlistByPage":
			pageNo := r.URL.Query().Get("pn")
			pages = append(pages, pageNo)
			if pageNo == "1" {
				return htmlResponse(r, kuwoShareRows(130, 30)), nil
			}
			if pageNo == "2" {
				return htmlResponse(r, ""), nil
			}
			return nil, fmt.Errorf("unexpected Kuwo page number: %s", pageNo)
		case r.URL.Host == "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Exact-page Kuwo list", "total": 60, "musiclist": []any{}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/7", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.SongsCount != 60 || list.TotalCount != 60 || !list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
}

func TestKuwoSharePaginationFailureKeepsInitialRowsPartial(t *testing.T) {
	page := `<html><div class="titleCenter">Partial Kuwo list</div><ul id="messDivId">` + kuwoShareRows(100, 30) + `</ul><button id="morebtnId" onclick="loadMore()">获取更多</button></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content":
			return htmlResponse(r, page), nil
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/playlistByPage":
			return nil, errors.New("simulated continuation failure")
		case r.URL.Host == "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Partial Kuwo list", "total": 60, "musiclist": []any{}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/7", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 30 || list.TotalCount != 60 || list.Complete || len(list.Warnings) == 0 || !strings.Contains(strings.Join(list.Warnings, " "), "后续歌曲加载失败") {
		t.Fatalf("failed Kuwo continuation was not reported partial: %#v", list)
	}
}

func TestKuwoSharePaginationRespectsExportLimit(t *testing.T) {
	page := `<html><div class="titleCenter">Limited Kuwo list</div><ul id="messDivId">` + kuwoShareRows(100, 30) + `</ul><button id="morebtnId" onclick="loadMore()">获取更多</button></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content":
			return htmlResponse(r, page), nil
		case r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/playlistByPage":
			return htmlResponse(r, kuwoShareRows(130, 30)), nil
		case r.URL.Host == "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Limited Kuwo list", "total": 80, "musiclist": []any{}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
		}
	})
	service.MaxSongs = 35

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/7", Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 35 || list.TotalCount != 80 || list.Complete || list.Tracks[34].ID != "134" {
		t.Fatalf("Kuwo continuation ignored the export limit: %#v", list)
	}
	if !strings.Contains(strings.Join(list.Warnings, " "), "达到本次导出上限 35 首") {
		t.Fatalf("Kuwo export limit was not disclosed: %#v", list.Warnings)
	}
}

func TestKuwoHiddenLoadMoreControlIsNotPending(t *testing.T) {
	for _, markup := range []string{
		`<button id="morebtnId" hidden onclick="loadMore()">获取更多</button>`,
		`<button id="morebtnId" aria-hidden="true" onclick="loadMore()">获取更多</button>`,
		`<button id="morebtnId" style="display:none" onclick="loadMore()">获取更多</button>`,
		`<button id="morebtnId" style="visibility:hidden !important" onclick="loadMore()">获取更多</button>`,
	} {
		page := `<html><div class="titleCenter">Small Kuwo list</div><ul id="messDivId">` + kuwoShareRow(1) + `</ul>` + markup + `</html>`
		_, hasMore, err := parseKuwoSharePage(strings.NewReader(page))
		if err != nil {
			t.Fatal(err)
		}
		if hasMore {
			t.Fatalf("hidden Kuwo control was treated as pending: %s", markup)
		}
	}
}

func TestKuwoVisibleTracksCanBeComplete(t *testing.T) {
	page := `<html><div class="titleCenter">Kuwo list</div><ul id="messDivId">
	<li class="singBox" mid="101"><span class="singTexUp2">First</span><span class="singName">Artist - First</span></li>
	<li class="singBox" mid="102"><span class="singTexUp2">Second</span><span class="singName">Artist - Second</span></li>
	</ul><button id="morebtnId" style="display:none" onclick="loadMore()">获取更多</button></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "www.kuwo.cn" {
			return htmlResponse(r, page), nil
		}
		if r.URL.Host == "nplserver.kuwo.cn" {
			return response(r, map[string]any{"result": "ok", "title": "Kuwo list", "total": "2", "musiclist": []any{}}), nil
		}
		return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/7", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 2 || list.TotalCount != 2 || !list.Complete || len(list.Warnings) != 0 {
		t.Fatalf("unexpected complete Kuwo result: %#v", list)
	}
}

func TestKuwoMobileShareUsesH5RowsWhenMetadataOnlyHasTotal(t *testing.T) {
	rows := []struct{ id, title string }{
		{"265180280", "忘れないでベイベー"},
		{"55017654", "メルヘル小惑星"},
		{"67526128", "シアワセシンドローム"},
		{"204687466", "ディスコミュ星人 (2022 ver.)"},
		{"78034438", "ダメレオンハート"},
		{"150255733", "Higher’s High"},
		{"158679496", "リセットセット"},
		{"55017671", "kidding"},
		{"92158896", "もしも信者"},
		{"157989278", "オトナのピーターパン"},
		{"142282310", "シアワセシンドローム"},
		{"55017673", "ダダダダ天使"},
		{"268117334", "奇縁ロマンス"},
		{"148354927", "完全放棄宣言"},
		{"52846213", "ワンルームシュガーライフ (TV size)"},
		{"225053586", "恋愛脳"},
	}
	var htmlRows strings.Builder
	wantIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		fmt.Fprintf(&htmlRows, `<li class="singBox" mid="%s"><span class="singTexUp2">%s</span><span class="singName">七音阿卡莉 (NANAOAKARI) - %s</span></li>`, row.id, row.title, row.title)
		wantIDs = append(wantIDs, row.id)
	}
	page := `<html><div class="titleCenter">NANAOAKARI热单 中国巡演在即！</div><ul id="messDivId">` + htmlRows.String() + `</ul></html>`
	newService := func() *Service {
		return testService(func(r *http.Request) (*http.Response, error) {
			if r.URL.Host == "www.kuwo.cn" && r.URL.Path == "/newh5/playlist/content" {
				return htmlResponse(r, page), nil
			}
			if r.URL.Host == "nplserver.kuwo.cn" {
				return response(r, map[string]any{"result": "ok", "title": "NANAOAKARI热单 中国巡演在即！", "total": 16, "musiclist": []any{}}), nil
			}
			return nil, fmt.Errorf("unexpected Kuwo request: %s", r.URL)
		})
	}
	for _, tc := range []struct{ name, url string }{
		{"official web share copy", "https://www.kuwo.cn/playlist_detail/3527541442"},
		{"mobile share detail", "https://m.kuwo.cn/newh5app/playlist_detail/3527541442?t=plantform&from=ar"},
		{"legacy playlist index", "https://www.kuwo.cn/playlist/index?pid=3527541442"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list, err := newService().Fetch(context.Background(), tc.url, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if list.Name != "NANAOAKARI热单 中国巡演在即！" || list.SongsCount != 16 || list.TotalCount != 16 || !list.Complete || len(list.Warnings) != 0 {
				t.Fatalf("Kuwo H5 rows were not accepted against the public total: %#v", list)
			}
			gotIDs := make([]string, len(list.Tracks))
			for index, track := range list.Tracks {
				gotIDs[index] = track.ID
			}
			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Fatalf("Kuwo H5 row order was not preserved: got=%v want=%v", gotIDs, wantIDs)
			}
			if list.Tracks[2].Title != list.Tracks[10].Title || list.Tracks[2].ID == list.Tracks[10].ID {
				t.Fatalf("same-title tracks with distinct IDs must both be preserved: %#v %#v", list.Tracks[2], list.Tracks[10])
			}
		})
	}
}

func TestKuwoShortLinkResolvesToPlaylist(t *testing.T) {
	const playlistID = "236289985"
	page := `<html><div class="titleCenter">Kuwo short-link list</div><ul id="messDivId">
	<li class="singBox" mid="101"><span class="singTexUp2">First</span><span class="singName">Artist - First</span></li>
	</ul></html>`
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "t.kuwo.cn":
			if r.Method != http.MethodGet || r.URL.Path != "/AbC123" {
				return nil, fmt.Errorf("unexpected Kuwo short link request: %s", r.URL)
			}
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://www.kuwo.cn/playlist_detail/" + playlistID}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		case "www.kuwo.cn":
			if r.URL.Path == "/playlist_detail/"+playlistID {
				return htmlResponse(r, "<html>resolved</html>"), nil
			}
			if r.URL.Path == "/newh5/playlist/content" && r.URL.Query().Get("pid") == playlistID {
				return htmlResponse(r, page), nil
			}
		case "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Kuwo short-link list", "total": 1, "musiclist": []any{}}), nil
		}
		return nil, fmt.Errorf("unexpected Kuwo short-link request: %s", r.URL)
	})

	list, err := service.Fetch(context.Background(), "酷我歌单 https://t.kuwo.cn/AbC123", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Kuwo short-link list" || list.SongsCount != 1 || list.TotalCount != 1 || !list.Complete || list.Tracks[0].ID != "101" {
		t.Fatalf("short link did not resolve to the Kuwo playlist adapter: %#v", list)
	}
}

func TestKuwoAlternateH5APIProvidesMorePartialRows(t *testing.T) {
	var pages []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "www.kuwo.cn":
			return htmlResponse(r, `<html>share page unavailable</html>`), nil
		case "nplserver.kuwo.cn":
			return response(r, map[string]any{"result": "ok", "title": "Kuwo list", "total": 4, "musiclist": []map[string]any{{"id": "legacy-1", "name": "Legacy row"}}}), nil
		case "m.kuwo.cn":
			pageNo := r.URL.Query().Get("pn")
			pages = append(pages, pageNo)
			rows := []map[string]any{
				{"musicrid": "MUSIC_1", "name": "API one", "artist": "Artist"},
				{"musicrid": "MUSIC_2", "name": "API two", "artist": "Artist"},
			}
			if pageNo == "2" {
				rows = []map[string]any{{"musicrid": "MUSIC_3", "name": "API three", "artist": "Artist"}}
			}
			return response(r, map[string]any{"code": 200, "msg": "success", "data": map[string]any{"name": "Kuwo API", "total": 3, "musicList": rows}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo fallback request: %s", r.URL)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/77", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.Name != "Kuwo API" || list.SongsCount != 3 || list.TotalCount != 4 || list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
	if list.Tracks[0].ID != "1" || list.Tracks[2].Title != "API three" || len(list.Warnings) == 0 {
		t.Fatalf("alternate Kuwo rows/order/partial warning missing: %#v", list)
	}
	if !strings.Contains(strings.Join(list.Warnings, " "), "总数不一致") {
		t.Fatalf("Kuwo fallback count disagreement was not disclosed: %#v", list.Warnings)
	}
}

func TestKuwoAlternateH5APIIsNeverDeclaredCompleteAlone(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "www.kuwo.cn", "nplserver.kuwo.cn":
			return htmlResponse(r, `<html>page unavailable</html>`), nil
		case "m.kuwo.cn":
			return response(r, map[string]any{"code": 200, "data": map[string]any{"name": "Kuwo API", "total": 1, "musicList": []map[string]any{{"musicrid": "MUSIC_1", "name": "Only API row"}}}}), nil
		default:
			return nil, fmt.Errorf("unexpected Kuwo fallback request: %s", r.URL)
		}
	})

	list, err := service.Fetch(context.Background(), "https://www.kuwo.cn/playlist_detail/77", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 1 || list.TotalCount != 1 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("fallback source was falsely declared complete: %#v", list)
	}
}

func TestKuwoAnonymousRequestSignatureFormat(t *testing.T) {
	const want = "1361b99125125e1ce61cc1f328ded44b00bc614e"
	if got := kuwoBuildSecret("0123456789abcdef", 12345678); got != want {
		t.Fatalf("Kuwo request signature changed: %q", got)
	}
	requestID, err := kuwoRequestID()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(requestID) {
		t.Fatalf("invalid Kuwo request ID: %q", requestID)
	}
}

func TestBodianPublicPlaylistUsesAnonymousPagesAndPreservesOrder(t *testing.T) {
	var pages []string
	var deviceID string
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "bd-api.kuwo.cn" {
			return nil, fmt.Errorf("unexpected Bodian request host: %s", r.URL.Host)
		}
		query := r.URL.Query()
		if query.Get("uid") != "-1" || query.Get("source") != "5" || query.Get("token") != "" {
			return nil, fmt.Errorf("unexpected Bodian auth/source query: %s", r.URL.RawQuery)
		}
		if query.Has("owerId") || strings.Contains(r.Header.Get("Referer"), "45780003") {
			return nil, fmt.Errorf("share owner identity was forwarded")
		}
		deviceID = r.Header.Get("devid")
		if deviceID == "" || r.Header.Get("qimei36") != deviceID {
			return nil, fmt.Errorf("missing Bodian anonymous device headers")
		}
		switch r.URL.Path {
		case "/api/service/playlist/info/123":
			return response(r, map[string]any{"code": 200, "msg": "success", "data": map[string]any{
				"name": "Bodian playlist", "musicCount": 3, "isPrivate": 0, "sourceType": 5,
			}}), nil
		case "/api/service/playlist/123/musicList":
			pageNo := query.Get("pn")
			pages = append(pages, pageNo)
			if pageNo == "1" {
				return response(r, map[string]any{"code": 200, "msg": "success", "data": map[string]any{
					"total": 3,
					"list": []map[string]any{
						{"id": 11, "name": "First", "artist": "Artist"},
						{"id": 12, "name": "Second", "artist": "Artist"},
					},
				}}), nil
			}
			if pageNo == "2" {
				return response(r, map[string]any{"code": 200, "msg": "success", "data": map[string]any{
					"total": 3,
					"list":  []map[string]any{{"id": 13, "name": "Third", "artist": "Artist"}},
				}}), nil
			}
		}
		return nil, fmt.Errorf("unexpected Bodian request path: %s", r.URL.Path)
	})

	input := "https://h5app.kuwo.cn/m/bodian/collection.html?uid=45780003&playlistId=123&source=5&owerId=owner"
	list, err := service.Fetch(context.Background(), input, Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.Name != "Bodian playlist" || list.SongsCount != 3 || list.TotalCount != 3 || !list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
	if got := []string{list.Tracks[0].ID, list.Tracks[1].ID, list.Tracks[2].ID}; !reflect.DeepEqual(got, []string{"11", "12", "13"}) {
		t.Fatalf("Bodian rows were not normalized or ordered: %v", got)
	}
}

func TestBodianPrivatePlaylistIsRejected(t *testing.T) {
	calls := 0
	service := testService(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/api/service/playlist/info/123" {
			return nil, fmt.Errorf("unexpected Bodian private-playlist call: %s", r.URL.Path)
		}
		return response(r, map[string]any{"code": 200, "msg": "success", "data": map[string]any{
			"name": "Private playlist", "musicCount": 1, "isPrivate": 1,
		}}), nil
	})
	_, err := service.Fetch(context.Background(), "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=123&source=5", Options{})
	if !errors.Is(err, ErrPrivatePlaylist) || calls != 1 {
		t.Fatalf("private Bodian playlist was not rejected before track fetch: calls=%d err=%v", calls, err)
	}
}

func TestBodianLaterPageFailureStaysPartial(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/service/playlist/info/123":
			return response(r, map[string]any{"code": 200, "data": map[string]any{"name": "Bodian partial", "musicCount": 3, "isPrivate": 0}}), nil
		case "/api/service/playlist/123/musicList":
			if r.URL.Query().Get("pn") == "1" {
				return response(r, map[string]any{"code": 200, "data": map[string]any{"total": 3, "list": []map[string]any{
					{"id": 11, "name": "First", "artist": "Artist"},
					{"id": 12, "name": "Second", "artist": "Artist"},
				}}}), nil
			}
			return nil, fmt.Errorf("simulated Bodian page failure")
		default:
			return nil, fmt.Errorf("unexpected Bodian request: %s", r.URL)
		}
	})

	input := "https://h5app.kuwo.cn/m/bodian/collection.html?playlistId=123&source=5"
	list, err := service.Fetch(context.Background(), input, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 2 || list.TotalCount != 3 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("Bodian later-page failure was not reported partial: %#v", list)
	}
}

func TestMiguContinuesAfterShortPageWhenTotalShowsMore(t *testing.T) {
	var pages []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/resource/playlist/v2.0") {
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"title": "Migu list", "musicNum": 3}}), nil
		}
		if strings.Contains(r.URL.Path, "/resource/playlist/song/v2.0") {
			pageNo := r.URL.Query().Get("pageNo")
			pages = append(pages, pageNo)
			first := 1
			last := 2
			if pageNo == "2" {
				first, last = 3, 3
			}
			rows := make([]map[string]any, 0, last-first+1)
			for id := first; id <= last; id++ {
				rows = append(rows, map[string]any{"contentId": id, "songName": fmt.Sprintf("Song %d", id), "singerList": []map[string]string{{"name": "Artist"}}})
			}
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"totalCount": 3, "songList": rows}}), nil
		}
		return nil, fmt.Errorf("unexpected Migu request: %s", r.URL)
	})

	list, err := service.Fetch(context.Background(), "https://music.migu.cn/v5/#/playlist?playlistId=9", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.Name != "Migu list" || list.SongsCount != 3 || list.TotalCount != 3 || !list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
}

func TestMiguCountBelowReturnedRowsStaysPartial(t *testing.T) {
	service := testService(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/resource/playlist/v2.0") {
			return response(r, map[string]any{"code": "000000", "data": map[string]any{"title": "Migu list", "musicNum": 2}}), nil
		}
		rows := []map[string]any{
			{"contentId": 1, "songName": "One"},
			{"contentId": 2, "songName": "Two"},
			{"contentId": 3, "songName": "Three"},
		}
		return response(r, map[string]any{"code": "000000", "data": map[string]any{"totalCount": 2, "songList": rows}}), nil
	})

	list, err := service.Fetch(context.Background(), "https://music.migu.cn/v5/#/playlist?playlistId=9", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.SongsCount != 3 || list.TotalCount != 3 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("contradictory Migu total hidden: %#v", list)
	}
}

func TestQianqianContinuesWhenHaveMoreConflictsWithTotal(t *testing.T) {
	var pages []string
	service := testService(func(r *http.Request) (*http.Response, error) {
		pageNo := r.URL.Query().Get("pageNo")
		pages = append(pages, pageNo)
		if r.URL.Query().Get("sign") != qianqianSign(r.URL.Query()) {
			return nil, fmt.Errorf("invalid Qianqian request signature")
		}
		first, last := 1, 2
		if pageNo == "2" {
			first, last = 3, 3
		}
		rows := make([]map[string]any, 0, last-first+1)
		for id := first; id <= last; id++ {
			rows = append(rows, map[string]any{"TSID": fmt.Sprintf("T%d", id), "title": fmt.Sprintf("Song %d", id), "artist": []map[string]string{{"name": "Artist"}}})
		}
		return response(r, map[string]any{"state": true, "data": map[string]any{"title": "Qianqian list", "trackCount": 3, "haveMore": false, "trackList": rows}}), nil
	})

	list, err := service.Fetch(context.Background(), "https://music.91q.com/songlist/13", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pages, []string{"1", "2"}) || list.SongsCount != 3 || list.TotalCount != 3 || !list.Complete {
		t.Fatalf("pages=%v list=%#v", pages, list)
	}
}
