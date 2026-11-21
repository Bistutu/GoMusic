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
	html, _, pageErr := s.HTTP.Do(ctx, http.MethodGet, link.URL.String(), nil, nil)
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

// track 兼容汽水 API 的两种响应结构：早期为 entity.track，
// 现在改为 entity.track_wrapper.track。优先取新结构，缺失时回退旧结构。
func (m qishuiMedia) track() *qishuiTrack {
	if m.Entity.TrackWrapper != nil && m.Entity.TrackWrapper.Track != nil {
		return m.Entity.TrackWrapper.Track
	}
	return m.Entity.Track
}

// qishuiAPIEndpoints are official Qishui/Luna hosts observed serving the same
// public playlist detail protocol. Rotating them keeps pagination alive when a
// single CDN or DNS name is unavailable; the share-page parser remains a
// separate, independent fallback.
var qishuiAPIEndpoints = []string{
	"https://beta-luna.douyin.com/luna/playlist/detail",
	"https://api.qishui.com/luna/playlist/detail",
	"https://api5-lf.qishui.com/luna/playlist/detail",
}

// qishuiPageBody tries every official host for one cursor page. A fresh payload
// reader is required for each attempt because the HTTP client consumes it.
func (s *Service) qishuiPageBody(ctx context.Context, payload []byte) ([]byte, error) {
	failures := make([]error, 0, len(qishuiAPIEndpoints))
	for _, endpoint := range qishuiAPIEndpoints {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		body, _, err := s.HTTP.Do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload),
			map[string]string{"Content-Type": "application/json", "User-Agent": "Luna/19.1.0 Android"})
		if err == nil && len(body) > 0 {
			return body, nil
		}
		if err != nil {
			failures = append(failures, err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, err
			}
		} else {
			failures = append(failures, fmt.Errorf("汽水备用主机返回空响应: %s", endpoint))
		}
	}
	if len(failures) == 0 {
		return nil, fmt.Errorf("汽水音乐所有备用主机均不可用")
	}
	return nil, errors.Join(failures...)
}

func (s *Service) qishuiPages(ctx context.Context, id string) (*models.SongList, error) {
	cursor := "0"
	seen := map[string]bool{}
	seenTracks := map[string]bool{}
	var list *models.SongList
	for len(seen) < 100 && !seen[cursor] {
		seen[cursor] = true
		payload, _ := json.Marshal(map[string]any{"playlist_id": id, "cursor": cursor, "count": 200})
		body, err := s.qishuiPageBody(ctx, payload)
		if err != nil {
			return list, err
		}
		var page struct {
			Playlist *struct {
				Title string `json:"title"`
				Count int    `json:"count_tracks"`
			} `json:"playlist"`
			Medias  []qishuiMedia   `json:"media_resources"`
			Cursor  json.RawMessage `json:"next_cursor"`
			HasMore bool            `json:"has_more"`
		}
		if err = json.Unmarshal(body, &page); err != nil {
			return list, err
		}
		if page.Playlist == nil || page.Playlist.Title == "" {
			return list, fmt.Errorf("汽水歌单不存在或未公开")
		}
		if list == nil {
			list = &models.SongList{Name: page.Playlist.Title, TotalCount: page.Playlist.Count}
		}
		list.TotalCount = max(list.TotalCount, page.Playlist.Count)
		before := len(list.Tracks)
		hasTracks := false
		for _, media := range page.Medias {
			if media.Type != "track" {
				continue
			}
			song := media.track()
			if song == nil || song.Name == "" {
				continue
			}
			hasTracks = true
			track := models.Track{ID: song.ID, Title: song.Name, Artists: []string{}}
			for _, artist := range song.Artists {
				if artist.Name != "" {
					track.Artists = append(track.Artists, artist.Name)
				}
			}
			appendDistinctTrack(list, track, seenTracks)
		}
		if hasTracks && len(list.Tracks) == before {
			return list, fmt.Errorf("汽水音乐分页没有返回新的歌曲")
		}
		if len(list.Tracks) >= s.MaxSongs || (!page.HasMore && len(list.Tracks) >= list.TotalCount) {
			return list, nil
		}
		next := strings.Trim(string(page.Cursor), "\"")
		if next == "" || next == "null" || seen[next] || len(page.Medias) == 0 {
			return list, fmt.Errorf("汽水音乐分页未继续")
		}
		cursor = next
	}
	return list, fmt.Errorf("汽水音乐分页次数超过限制")
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
		for _, media := range page.Medias {
			if media.Type != "track" {
				continue
			}
			song := media.track()
			if song == nil || song.Name == "" {
				continue
			}
			track := models.Track{ID: song.ID, Title: song.Name, Artists: []string{}}
			for _, artist := range song.Artists {
				if artist.Name != "" {
					track.Artists = append(track.Artists, artist.Name)
				}
			}
			list.Tracks = append(list.Tracks, track)
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
