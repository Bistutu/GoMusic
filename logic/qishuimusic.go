package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *Service) qishui(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	var err error
	link, err = s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}
	var apiList *models.SongList
	var apiErr error
	if link.ID != "" {
		apiList, apiErr = s.qishuiPages(ctx, link.ID)
		if apiErr == nil {
			return apiList, nil
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	shareURL := "https://music.douyin.com/qishui/share/playlist?playlist_id=" + url.QueryEscape(link.ID)
	html, _, pageErr := s.HTTP.Do(ctx, http.MethodGet, shareURL, nil, nil)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	preview, previewErr := parseQishui(bytes.NewReader(html))
	if pageErr == nil && previewErr == nil {
		best := preferPlaylist(apiList, preview)
		if len(best.Tracks) < best.TotalCount {
			best.Warnings = append(best.Warnings, "汽水音乐分页未完成，请稍后重试")
		}
		return best, nil
	}
	if apiList != nil {
		apiList.Warnings = append(apiList.Warnings, "汽水音乐分页未完成，请稍后重试")
		return apiList, nil
	}
	return nil, fmt.Errorf("汽水音乐未返回可读取的歌单: %w", errors.Join(apiErr, pageErr, previewErr))
}

type qishuiTrack struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

type qishuiMedia struct {
	Type   string `json:"type"`
	Entity struct {
		Track        *qishuiTrack `json:"track"`
		TrackWrapper *struct {
			Track *qishuiTrack `json:"track"`
		} `json:"track_wrapper"`
	} `json:"entity"`
}

func (m qishuiMedia) track() models.Track {
	if m.Type != "track" {
		return models.Track{}
	}
	song := m.Entity.Track
	if wrapped := m.Entity.TrackWrapper; wrapped != nil && wrapped.Track != nil && wrapped.Track.Name != "" {
		song = wrapped.Track
	}
	if song == nil || song.Name == "" {
		return models.Track{}
	}
	track := models.Track{ID: song.ID, Title: song.Name, Artists: []string{}}
	for _, artist := range song.Artists {
		if artist.Name != "" {
			track.Artists = append(track.Artists, artist.Name)
		}
	}
	return track
}

var qishuiAPIEndpoints = [...]string{
	"https://beta-luna.douyin.com/luna/playlist/detail",
	"https://api.qishui.com/luna/playlist/detail",
	"https://api5-lf.qishui.com/luna/playlist/detail",
}

type qishuiAPIPage struct {
	Playlist *struct {
		Title string `json:"title"`
		Count *int   `json:"count_tracks"`
	} `json:"playlist"`
	Medias  []qishuiMedia   `json:"media_resources"`
	Cursor  json.RawMessage `json:"next_cursor"`
	HasMore bool            `json:"has_more"`
	Status  int             `json:"status_code"`
	Tracks  []models.Track  `json:"-"`
}

// Subsequent pages start with the last successful host to avoid repeating
// a slow or failed request for every cursor.
func (s *Service) qishuiPage(ctx context.Context, payload []byte, preferredHost, remaining int) (*qishuiAPIPage, int, error) {
	failures := make([]error, 0, len(qishuiAPIEndpoints))
	knownTotal := -1
	for attempt := range len(qishuiAPIEndpoints) {
		if err := ctx.Err(); err != nil {
			return nil, preferredHost, err
		}
		index := (preferredHost + attempt) % len(qishuiAPIEndpoints)
		endpoint := qishuiAPIEndpoints[index]
		body, _, err := s.HTTP.Do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload),
			map[string]string{"Content-Type": "application/json", "User-Agent": "Luna/19.1.0 Android"})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		var page qishuiAPIPage
		if err := json.Unmarshal(body, &page); err != nil {
			failures = append(failures, fmt.Errorf("%s 返回无效的歌单响应: %w", endpoint, err))
			continue
		}
		if page.Status != 0 || page.Playlist == nil || strings.TrimSpace(page.Playlist.Title) == "" {
			failures = append(failures, fmt.Errorf("%s 没有返回公开歌单信息", endpoint))
			continue
		}
		if page.Playlist.Count != nil && *page.Playlist.Count < 0 {
			failures = append(failures, fmt.Errorf("%s 返回无效的歌单总数", endpoint))
			continue
		}
		for _, media := range page.Medias {
			if track := media.track(); track.Title != "" {
				page.Tracks = append(page.Tracks, track)
			}
		}
		if page.Playlist.Count != nil {
			knownTotal = max(knownTotal, *page.Playlist.Count)
		}
		knownEmpty := page.Playlist.Count != nil && *page.Playlist.Count == 0 && knownTotal == 0 && remaining == 0
		if len(page.Tracks) == 0 && (!knownEmpty || page.HasMore) {
			failures = append(failures, fmt.Errorf("%s 未返回可读取的歌曲", endpoint))
			continue
		}
		if knownTotal >= 0 {
			page.Playlist.Count = &knownTotal
		}
		return &page, index, nil
	}
	return nil, preferredHost, errors.Join(failures...)
}

func (s *Service) qishuiPages(ctx context.Context, id string) (*models.SongList, error) {
	cursor := "0"
	seen := map[string]bool{}
	seenTracks := map[string]bool{}
	var list *models.SongList
	host, remaining := 0, 0
	totalKnown := false
	for len(seen) < 100 && !seen[cursor] {
		seen[cursor] = true
		payload, _ := json.Marshal(map[string]any{"playlist_id": id, "cursor": cursor, "count": 200})
		page, usedHost, err := s.qishuiPage(ctx, payload, host, remaining)
		if err != nil {
			return list, err
		}
		host = usedHost
		if list == nil {
			list = &models.SongList{Name: page.Playlist.Title}
		}
		if page.Playlist.Count != nil {
			totalKnown = true
			list.TotalCount = max(list.TotalCount, *page.Playlist.Count)
		}
		before := len(list.Tracks)
		for _, track := range page.Tracks {
			appendDistinctTrack(list, track, seenTracks)
		}
		if len(page.Tracks) > 0 && len(list.Tracks) == before {
			return list, fmt.Errorf("汽水音乐分页没有返回新的歌曲")
		}
		if len(list.Tracks) >= s.MaxSongs && page.HasMore && list.TotalCount <= len(list.Tracks) {
			list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首，汽水音乐仍有后续页", s.MaxSongs))
		}
		if len(list.Tracks) >= s.MaxSongs || (!page.HasMore && len(list.Tracks) >= list.TotalCount) {
			if !totalKnown {
				list.Warnings = append(list.Warnings, "汽水音乐未返回可确认的歌单总数")
			}
			return list, nil
		}
		remaining = max(0, list.TotalCount-len(list.Tracks))
		next := qishuiCursor(page.Cursor)
		if next == "" || seen[next] {
			return list, fmt.Errorf("汽水音乐分页未继续")
		}
		cursor = next
	}
	return list, fmt.Errorf("汽水音乐分页次数超过限制")
}

func qishuiCursor(raw json.RawMessage) string {
	var cursor string
	if json.Unmarshal(raw, &cursor) == nil {
		return cursor
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		return number.String()
	}
	return ""
}

// The current share page embeds playlist_page data in _ROUTER_DATA, independently of CSS layout.
func parseQishui(body io.Reader) (*models.SongList, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	var list *models.SongList
	doc.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := strings.TrimSpace(script.Text())
		start := strings.Index(text, "_ROUTER_DATA")
		if start < 0 {
			return true
		}
		text = text[start+len("_ROUTER_DATA"):]
		text = strings.TrimSpace(text)
		if !strings.HasPrefix(text, "=") {
			return true
		}
		text = strings.TrimSpace(strings.TrimPrefix(text, "="))
		var data struct {
			LoaderData struct {
				Page *struct {
					Info *struct {
						Title string `json:"title"`
						Count int    `json:"count_tracks"`
					} `json:"playlistInfo"`
					Medias []qishuiMedia `json:"medias"`
				} `json:"playlist_page"`
			} `json:"loaderData"`
		}
		// Decode exactly one JSON value; subsequent hydration JavaScript is never evaluated.
		if json.NewDecoder(strings.NewReader(text)).Decode(&data) != nil || data.LoaderData.Page == nil || data.LoaderData.Page.Info == nil || data.LoaderData.Page.Info.Title == "" {
			return true
		}
		page := data.LoaderData.Page
		list = &models.SongList{Name: page.Info.Title, TotalCount: page.Info.Count}
		seen := map[string]bool{}
		for _, media := range page.Medias {
			appendDistinctTrack(list, media.track(), seen)
		}
		return false
	})
	if list != nil {
		return list, nil
	}
	return nil, fmt.Errorf("汽水音乐未返回可读取的歌单，请确认歌单已公开，并使用汽水音乐的歌单分享链接")
}
func QiShuiMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return NewService(nil, nil, nil, 10000, 4).Fetch(ctx, link, Options{Detailed: detailed})
}
