package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Historical upstream responses have silently capped larger detail batches.
// Keep below that boundary; see issue #18 and the capped-response regression.
const chunkSize = 200

type netSong struct {
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
	AR   []struct {
		Name string `json:"name"`
	} `json:"ar"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

func (song netSong) track() models.Track {
	t := models.Track{ID: song.ID.String(), Title: song.Name, Artists: []string{}}
	for _, a := range song.AR {
		if a.Name != "" {
			t.Artists = append(t.Artists, a.Name)
		}
	}
	if len(t.Artists) == 0 {
		for _, a := range song.Artists {
			if a.Name != "" {
				t.Artists = append(t.Artists, a.Name)
			}
		}
	}
	return t
}

func (s *Service) netEase(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	link, err := s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}
	if link.Kind == "album" {
		return s.netEaseAlbum(ctx, link.ID)
	}
	body, _, err := s.HTTP.Do(ctx, http.MethodPost, "https://music.163.com/api/v6/playlist/detail",
		strings.NewReader(url.Values{"id": {link.ID}, "n": {"100000"}}.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("请求网易云歌单失败: %w", err)
	}
	var metadata struct {
		Code     int `json:"code"`
		Playlist *struct {
			Name       string    `json:"name"`
			TrackCount int       `json:"trackCount"`
			Tracks     []netSong `json:"tracks"`
			TrackIDs   []struct {
				ID json.Number `json:"id"`
			} `json:"trackIds"`
		} `json:"playlist"`
	}
	if err = json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("网易云返回了无效数据: %w", err)
	}
	if metadata.Code == 401 || metadata.Code == 403 {
		return nil, fmt.Errorf("%w: 该网易云歌单未公开，请先设置为公开", ErrPrivatePlaylist)
	}
	if metadata.Code != 200 || metadata.Playlist == nil || metadata.Playlist.Name == "" {
		return nil, fmt.Errorf("网易云歌单不存在、未公开或暂时不可访问 (code=%d)", metadata.Code)
	}
	p := metadata.Playlist
	if p.TrackCount > 0 && len(p.TrackIDs) == 0 {
		return nil, fmt.Errorf("网易云未返回歌曲列表，请检查歌单访问权限")
	}
	ids := make([]string, 0, len(p.TrackIDs))
	for _, item := range p.TrackIDs {
		if item.ID.String() != "" {
			ids = append(ids, item.ID.String())
		}
		if len(ids) >= s.MaxSongs {
			break
		}
	}
	seeds := make(map[string]models.Track, len(p.Tracks))
	for _, song := range p.Tracks {
		if song.Name != "" {
			seeds[song.ID.String()] = song.track()
		}
	}
	tracks, err := s.netEaseSongs(ctx, ids, seeds)
	if err != nil {
		return nil, err
	}
	list := &models.SongList{Name: p.Name, TotalCount: p.TrackCount, Tracks: make([]models.Track, 0, len(ids))}
	if len(p.TrackIDs) > list.TotalCount {
		list.TotalCount = len(p.TrackIDs)
	}
	for _, id := range ids {
		if track, ok := tracks[id]; ok {
			list.Tracks = append(list.Tracks, track)
		}
	}
	return list, nil
}

func (s *Service) netEaseAlbum(ctx context.Context, id string) (*models.SongList, error) {
	var last error
	for _, prefix := range []string{"https://music.163.com/api/v1/album/", "https://music.163.com/api/album/"} {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, prefix+id, nil, map[string]string{"Referer": "https://music.163.com/"})
		if err != nil {
			last = err
			continue
		}
		var data struct {
			Code  int `json:"code"`
			Album *struct {
				Name  string    `json:"name"`
				Size  int       `json:"size"`
				Songs []netSong `json:"songs"`
			} `json:"album"`
			Songs []netSong `json:"songs"`
		}
		if err = json.Unmarshal(body, &data); err != nil {
			last = err
			continue
		}
		if data.Code != 200 || data.Album == nil {
			last = fmt.Errorf("网易云专辑不存在或暂时不可访问 (code=%d)", data.Code)
			continue
		}
		songs := data.Songs
		if songs == nil {
			songs = data.Album.Songs
		}
		if data.Album.Name == "" || (data.Album.Size > 0 && songs == nil) {
			last = fmt.Errorf("网易云未返回有效专辑数据")
			continue
		}
		list := &models.SongList{Name: data.Album.Name, TotalCount: data.Album.Size}
		for _, song := range songs {
			if song.Name != "" {
				list.Tracks = append(list.Tracks, song.track())
			}
		}
		return list, nil
	}
	return nil, last
}

func (s *Service) netEaseSongs(ctx context.Context, ids []string, seeds map[string]models.Track) (map[string]models.Track, error) {
	found := make(map[string]models.Track, len(ids))
	for _, id := range ids {
		if track, ok := seeds[id]; ok {
			found[id] = track
		}
	}
	// Iterate the requested IDs, including when MGET failed or returned no values.
	for _, repo := range []Repository{s.Cache, s.Store} {
		missing := missingIDs(ids, found)
		if len(missing) == 0 {
			return found, nil
		}
		if repo == nil {
			continue
		}
		values, err := repo.Get(ctx, musicurl.NetEase, missing)
		if err != nil {
			log.Warnf("读取歌曲缓存失败，继续向上游获取: %v", err)
			continue
		}
		for _, id := range missing {
			if track, ok := values[id]; ok && track.ID == id && track.Title != "" {
				found[id] = track
			}
		}
	}
	missing := missingIDs(ids, found)
	var mu sync.Mutex
	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.Concurrency)
	for start := 0; start < len(missing); start += chunkSize {
		end := start + chunkSize
		if end > len(missing) {
			end = len(missing)
		}
		chunk := missing[start:end]
		group.Go(func() error {
			payload := make([]map[string]string, 0, len(chunk))
			for _, id := range chunk {
				payload = append(payload, map[string]string{"id": id})
			}
			encoded, _ := json.Marshal(payload)
			body, _, err := s.HTTP.Do(gctx, http.MethodPost, "https://music.163.com/api/v3/song/detail",
				strings.NewReader(url.Values{"c": {string(encoded)}}.Encode()), nil)
			if err != nil {
				return fmt.Errorf("获取网易云歌曲详情失败: %w", err)
			}
			var result struct {
				Code  int       `json:"code"`
				Songs []netSong `json:"songs"`
			}
			if err = json.Unmarshal(body, &result); err != nil {
				return err
			}
			if result.Code != 200 {
				return fmt.Errorf("网易云歌曲详情暂时不可访问 (code=%d)", result.Code)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, song := range result.Songs {
				if song.Name != "" {
					found[song.ID.String()] = song.track()
				}
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	// Backfill Redis even when all missing metadata came from MySQL.
	values := make([]models.Track, 0, len(found))
	for _, track := range found {
		values = append(values, track)
	}
	if s.Cache != nil && len(values) > 0 {
		if err := s.Cache.Put(ctx, musicurl.NetEase, values); err != nil {
			log.Warnf("写入 Redis 缓存失败: %v", err)
		}
	}
	fresh := make([]models.Track, 0, len(missing))
	for _, id := range missing {
		if track, ok := found[id]; ok {
			fresh = append(fresh, track)
		}
	}
	if s.Store != nil && len(fresh) > 0 {
		if err := s.Store.Put(ctx, musicurl.NetEase, fresh); err != nil {
			log.Warnf("写入 MySQL 缓存失败: %v", err)
		}
	}
	return found, nil
}

func missingIDs(ids []string, found map[string]models.Track) []string {
	missing := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if _, ok := found[id]; !ok && !seen[id] {
			missing = append(missing, id)
			seen[id] = true
		}
	}
	return missing
}

// NetEasyDiscover remains available for existing Go callers; HTTP handlers use the injected service.
func NetEasyDiscover(link string, detailed bool) (*models.SongList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return NewService(nil, nil, nil, 10000, 4).Fetch(ctx, link, Options{Detailed: detailed})
}
