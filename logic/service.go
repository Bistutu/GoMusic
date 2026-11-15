package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
	"GoMusic/misc/utils"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidLink = errors.New("invalid playlist link")
var ErrPrivatePlaylist = errors.New("playlist is private")

// Repository stores raw metadata. Storage failures are cache misses, never empty playlists.
type Repository interface {
	Get(context.Context, string, []string) (map[string]models.Track, error)
	Put(context.Context, string, []models.Track) error
}

type Service struct {
	HTTP                  *httputil.Client
	Cache, Store          Repository
	MaxSongs, Concurrency int
}

type Options struct {
	Detailed      bool
	Format, Order string
}

func NewService(client *httputil.Client, cache, store Repository, maxSongs, concurrency int) *Service {
	if client == nil {
		client = httputil.NewClient(nil)
	}
	if maxSongs <= 0 {
		maxSongs = 10000
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Service{HTTP: client, Cache: cache, Store: store, MaxSongs: maxSongs, Concurrency: concurrency}
}

// A preview or legacy endpoint can report a smaller total than the primary
// endpoint. Keep the best result without forgetting songs already known to exist.
func preferPlaylist(current, candidate *models.SongList) *models.SongList {
	if current == nil {
		return candidate
	}
	if candidate == nil {
		return current
	}
	total := max(current.TotalCount, candidate.TotalCount)
	if len(candidate.Tracks) > len(current.Tracks) || (len(candidate.Tracks) == len(current.Tracks) && len(candidate.Warnings) < len(current.Warnings)) {
		current = candidate
	}
	current.TotalCount = total
	return current
}

// Pagination can overlap even when page bodies differ. Use stable IDs, never
// titles: different recordings can have the same name. Unknown IDs stay intact.
func appendDistinctTrack(list *models.SongList, track models.Track, seen map[string]bool) bool {
	if track.Title == "" {
		return false
	}
	if track.ID != "" && track.ID != "0" {
		if seen[track.ID] {
			return false
		}
		seen[track.ID] = true
	}
	list.Tracks = append(list.Tracks, track)
	return true
}

func (s *Service) Fetch(ctx context.Context, input string, options Options) (*models.SongList, error) {
	link, err := musicurl.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLink, err)
	}
	var list *models.SongList
	switch link.Provider {
	case musicurl.NetEase:
		list, err = s.netEase(ctx, link)
	case musicurl.QQ:
		list, err = s.qqMusic(ctx, link)
	case musicurl.Kugou:
		list, err = s.kugou(ctx, link)
	case musicurl.Qishui:
		list, err = s.qishui(ctx, link)
	}
	if err != nil {
		return nil, err
	}
	list.Source = link.Provider
	if len(list.Tracks) > s.MaxSongs {
		list.Tracks = list.Tracks[:s.MaxSongs]
		list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首", s.MaxSongs))
	}
	list.SongsCount = len(list.Tracks)
	if list.TotalCount < list.SongsCount {
		list.TotalCount = list.SongsCount
	}
	list.Complete = list.SongsCount == list.TotalCount && len(list.Warnings) == 0
	if !list.Complete && len(list.Warnings) == 0 {
		list.Warnings = append(list.Warnings, fmt.Sprintf("平台显示 %d 首，实际获取 %d 首，部分歌曲可能已下架或受访问限制", list.TotalCount, list.SongsCount))
	}
	if list.Tracks == nil {
		list.Tracks = []models.Track{}
	}
	if list.Warnings == nil {
		list.Warnings = []string{}
	}
	list.Songs = make([]string, 0, len(list.Tracks))
	for _, track := range list.Tracks {
		list.Songs = append(list.Songs, FormatTrack(track, options))
	}
	if options.Order == "reverse" {
		for i, j := 0, len(list.Songs)-1; i < j; i, j = i+1, j-1 {
			list.Songs[i], list.Songs[j] = list.Songs[j], list.Songs[i]
			list.Tracks[i], list.Tracks[j] = list.Tracks[j], list.Tracks[i]
		}
	}
	return list, nil
}

func FormatTrack(track models.Track, options Options) string {
	title := track.Title
	if !options.Detailed {
		title = utils.StandardSongName(title)
	}
	artist := strings.Join(track.Artists, " / ")
	if options.Format == "song" || artist == "" {
		return title
	}
	if options.Format == "singer-song" {
		return artist + " - " + title
	}
	return title + " - " + artist
}

func (s *Service) resolve(ctx context.Context, link musicurl.Link) (musicurl.Link, error) {
	if link.ID != "" {
		return link, nil
	}
	final, err := s.HTTP.Resolve(ctx, link.URL.String(), nil)
	if err != nil {
		return link, err
	}
	resolved, err := musicurl.Parse(final.String())
	if err != nil || resolved.ID == "" {
		return link, fmt.Errorf("%w: 分享链接没有有效歌单 ID", ErrInvalidLink)
	}
	return resolved, nil
}
