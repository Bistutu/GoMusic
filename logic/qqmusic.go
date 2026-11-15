package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"GoMusic/misc/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxSongsPerPage = 30
const qqMusicAPIURL = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"
const qqMusicLegacyAPIURL = "https://c.y.qq.com/qzone/fcg-bin/fcg_ucc_getcdinfo_byids_cp.fcg?type=1&json=1&utf8=1&onlysong=0&disstid=%s&format=json&g_tk=5381&loginUin=0&hostUin=0&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq&needNewCode=0"

func (s *Service) qqMusic(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	link, err := s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(link.ID)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("%w: 歌单 ID 无效", ErrInvalidLink)
	}
	primary, primaryErr := s.qqPages(ctx, id)
	if primaryErr == nil && len(primary.Tracks) >= primary.TotalCount {
		return primary, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	legacy, legacyErr := s.qqLegacy(ctx, link.ID)
	if legacyErr == nil && (primary == nil || len(legacy.Tracks) > len(primary.Tracks)) {
		return legacy, nil
	}
	if primary != nil {
		primary.Warnings = append(primary.Warnings, "QQ 音乐未返回完整歌单，请检查公开权限或稍后重试")
		return primary, nil
	}
	if legacyErr == nil {
		return legacy, nil
	}
	return nil, fmt.Errorf("QQ 音乐歌单暂时不可访问，请确认歌单已公开: %w", errors.Join(primaryErr, legacyErr))
}

func (s *Service) qqPages(ctx context.Context, id int) (*models.SongList, error) {
	var list *models.SongList
	seenPages := map[string]bool{}
	for begin := 0; begin < s.MaxSongs; {
		count := maxSongsPerPage
		if list != nil && count > list.TotalCount-begin {
			count = list.TotalCount - begin
		}
		if count > s.MaxSongs-begin {
			count = s.MaxSongs - begin
		}
		if count <= 0 {
			break
		}
		page, err := s.qqPage(ctx, id, begin, count)
		if err != nil {
			return list, err
		}
		data := page.Req0.Data
		if list == nil {
			list = &models.SongList{Name: data.Dirinfo.Title, TotalCount: data.Dirinfo.Songnum}
		}
		if len(data.Songlist) == 0 {
			if begin < list.TotalCount {
				return list, fmt.Errorf("QQ 音乐第 %d 页未返回歌曲", begin/maxSongsPerPage+1)
			}
			break
		}
		// Detect upstream endpoints ignoring song_begin rather than duplicating page one.
		fingerprint, _ := json.Marshal(data.Songlist)
		if seenPages[string(fingerprint)] {
			return list, fmt.Errorf("QQ 音乐返回重复分页")
		}
		seenPages[string(fingerprint)] = true
		for _, song := range data.Songlist {
			track := models.Track{ID: song.ID.String(), Title: song.Name, Artists: []string{}}
			for _, artist := range song.Singer {
				if artist.Name != "" {
					track.Artists = append(track.Artists, artist.Name)
				}
			}
			if track.Title != "" {
				list.Tracks = append(list.Tracks, track)
			}
		}
		begin += len(data.Songlist)
		if begin >= list.TotalCount {
			break
		}
	}
	if list == nil {
		return nil, fmt.Errorf("QQ 音乐未返回有效歌单")
	}
	return list, nil
}

func (s *Service) qqPage(ctx context.Context, id, begin, count int) (*models.QQMusicResp, error) {
	var last error
	for _, platform := range []string{"-1", "android", "iphone", "h5", "wxfshare", "iphone_wx", "windows"} {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		payload := models.GetQQMusicReqStringWithPagination(id, platform, begin, count)
		address := fmt.Sprintf(qqMusicAPIURL, utils.Encrypt(payload), time.Now().UnixMilli())
		body, _, err := s.HTTP.Do(ctx, http.MethodPost, address, strings.NewReader(payload), map[string]string{"Content-Type": "application/json", "Referer": "https://y.qq.com/"})
		if err != nil {
			last = err
			continue
		}
		var page models.QQMusicResp
		if err = json.Unmarshal(body, &page); err != nil {
			last = err
			continue
		}
		if page.Code != 0 || page.Req0 == nil || page.Req0.Code != 0 || page.Req0.Data == nil {
			last = fmt.Errorf("QQ 音乐拒绝歌单访问 (code=%d)", page.Code)
			continue
		}
		if page.Req0.Data.Dirinfo.Title == "" && len(page.Req0.Data.Songlist) == 0 {
			last = fmt.Errorf("QQ 音乐返回了无效的歌单数据")
			continue
		}
		return &page, nil
	}
	return nil, fmt.Errorf("QQ 音乐分页请求失败: %w", last)
}

func (s *Service) qqLegacy(ctx context.Context, id string) (*models.SongList, error) {
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, fmt.Sprintf(qqMusicLegacyAPIURL, id), nil, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return nil, err
	}
	return decodeLegacyPlaylist(bytes.NewReader(body))
}

func decodeLegacyPlaylist(body io.Reader) (*models.SongList, error) {
	var response models.QQMusicLegacyResp
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}
	if response.Code != 0 || response.Subcode != 0 {
		return nil, fmt.Errorf("旧版接口业务错误(code:%d, subcode:%d): %s", response.Code, response.Subcode, response.Message)
	}
	if len(response.Cdlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回歌单")
	}
	data := response.Cdlist[0]
	if data.Dissname == "" && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回有效歌单信息")
	}
	if data.Songnum > 0 && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回歌曲")
	}
	list := &models.SongList{Name: data.Dissname, TotalCount: data.Songnum}
	for _, song := range data.Songlist {
		track := models.Track{ID: song.Songid.String(), Title: song.Songname, Artists: []string{}}
		for _, artist := range song.Singer {
			if artist.Name != "" {
				track.Artists = append(track.Artists, artist.Name)
			}
		}
		if track.Title != "" {
			list.Tracks = append(list.Tracks, track)
		}
	}
	return list, nil
}

// Compatibility helpers for existing Go users and regression tests.
func QQMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return NewService(nil, nil, nil, 10000, 4).Fetch(ctx, link, Options{Detailed: detailed})
}
func extractPlaylistID(input string) (int, error) {
	link, err := musicurl.Parse(input)
	if err != nil {
		return 0, err
	}
	if link.Provider != musicurl.QQ {
		return 0, fmt.Errorf("不是 QQ 音乐歌单链接")
	}
	link, err = NewService(nil, nil, nil, 10000, 4).resolve(context.Background(), link)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(link.ID)
}
func parseLegacyPlaylist(body io.Reader, detailed bool) (*models.SongList, error) {
	list, err := decodeLegacyPlaylist(body)
	if err != nil {
		return nil, err
	}
	list.SongsCount = len(list.Tracks)
	for _, track := range list.Tracks {
		list.Songs = append(list.Songs, FormatTrack(track, Options{Detailed: detailed}))
	}
	return list, nil
}
