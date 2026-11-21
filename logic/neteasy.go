package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
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
	list, primaryErr := s.netEaseV6(ctx, link.ID)
	if primaryErr == nil {
		return list, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// v6 接口失败时降级到仍存活的老接口 /api/playlist/detail。
	// 老接口一次返回完整 tracks（含歌名与歌手），无需再逐批请求歌曲详情。
	if legacy, legacyErr := s.netEaseLegacy(ctx, link.ID); legacyErr == nil {
		return legacy, nil
	}
	return nil, primaryErr
}

// netEaseV6 使用 v6 详情接口取 trackIds，再按批次补齐歌曲元数据。
func (s *Service) netEaseV6(ctx context.Context, id string) (*models.SongList, error) {
	body, _, err := s.HTTP.Do(ctx, http.MethodPost, "https://music.163.com/api/v6/playlist/detail",
		strings.NewReader(url.Values{"id": {id}, "n": {"100000"}}.Encode()), nil)
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

// netEaseLegacy 使用仍存活的老接口 /api/playlist/detail 获取完整歌单。
// 该接口一次返回带歌名和歌手的 tracks，作为 v6 接口故障时的备用路径。
func (s *Service) netEaseLegacy(ctx context.Context, id string) (*models.SongList, error) {
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, "https://music.163.com/api/playlist/detail?id="+url.QueryEscape(id), nil,
		map[string]string{"Referer": "https://music.163.com/"})
	if err != nil {
		return nil, fmt.Errorf("请求网易云备用接口失败: %w", err)
	}
	var metadata struct {
		Code   int `json:"code"`
		Result *struct {
			Name       string    `json:"name"`
			TrackCount int       `json:"trackCount"`
			Tracks     []netSong `json:"tracks"`
		} `json:"result"`
	}
	if err = json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("网易云备用接口返回了无效数据: %w", err)
	}
	if metadata.Code == 401 || metadata.Code == 403 {
		return nil, fmt.Errorf("%w: 该网易云歌单未公开，请先设置为公开", ErrPrivatePlaylist)
	}
	if metadata.Code != 200 || metadata.Result == nil || metadata.Result.Name == "" {
		return nil, fmt.Errorf("网易云备用接口歌单不存在或暂时不可访问 (code=%d)", metadata.Code)
	}
	result := metadata.Result
	list := &models.SongList{Name: result.Name, TotalCount: result.TrackCount, Tracks: make([]models.Track, 0, len(result.Tracks))}
	seen := map[string]bool{}
	for _, song := range result.Tracks {
		if song.Name == "" {
			continue
		}
		track := song.track()
		if track.ID != "" && track.ID != "0" {
			if seen[track.ID] {
				continue
			}
			seen[track.ID] = true
		}
		list.Tracks = append(list.Tracks, track)
		if len(list.Tracks) >= s.MaxSongs {
			break
		}
	}
	if len(list.Tracks) == 0 {
		return nil, fmt.Errorf("网易云备用接口未返回有效歌曲")
	}
	if list.TotalCount < len(list.Tracks) {
		list.TotalCount = len(list.Tracks)
	}
	return list, nil
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
			songs, detailErr := s.netEaseSongChunk(gctx, chunk, string(encoded))
			if detailErr != nil {
				return detailErr
			}
			mu.Lock()
			defer mu.Unlock()
			for _, song := range songs {
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

// netEaseSongChunk tries the batch detail endpoint first and falls back to the
// older official /api/song/detail route. Both are read-only metadata APIs.
func (s *Service) netEaseSongChunk(ctx context.Context, chunk []string, encodedIDs string) ([]netSong, error) {
	body, _, err := s.HTTP.Do(ctx, http.MethodPost, "https://music.163.com/api/v3/song/detail",
		strings.NewReader(url.Values{"c": {encodedIDs}}.Encode()), nil)
	var result struct {
		Code  int       `json:"code"`
		Songs []netSong `json:"songs"`
	}
	if err == nil {
		if parseErr := json.Unmarshal(body, &result); parseErr == nil && result.Code == 200 {
			return result.Songs, nil
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	idList := make([]string, 0, len(chunk))
	for _, id := range chunk {
		if id != "" {
			idList = append(idList, id)
		}
	}
	idsJSON, _ := json.Marshal(idList)
	legacyBody, _, legacyErr := s.HTTP.Do(ctx, http.MethodGet, "https://music.163.com/api/song/detail?ids="+
		url.QueryEscape(string(idsJSON)), nil, map[string]string{"Referer": "https://music.163.com/"})
	if legacyErr != nil {
		return nil, fmt.Errorf("获取网易云歌曲详情失败: %w", errors.Join(err, legacyErr))
	}
	var legacy struct {
		Code  int       `json:"code"`
		Songs []netSong `json:"songs"`
	}
	if err := json.Unmarshal(legacyBody, &legacy); err != nil {
		return nil, fmt.Errorf("网易云备用歌曲详情格式无效: %w", err)
	}
	if legacy.Code != 200 {
		return nil, fmt.Errorf("网易云备用歌曲详情暂时不可访问 (code=%d)", legacy.Code)
	}
	return legacy.Songs, nil
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
