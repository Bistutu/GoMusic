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
	primary, primaryErr := s.qqPages(ctx, id, s.qqPage)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if primaryErr == nil && (qqPlaylistComplete(primary) || len(primary.Tracks) >= s.MaxSongs) {
		return primary, nil
	}

	// 主接口失败或不完整时，依次尝试所有备用接口，保留数据最完整的结果。
	best := primary
	failures := make([]error, 0, 5)
	if primaryErr != nil {
		failures = append(failures, fmt.Errorf("签名接口: %w", primaryErr))
	}
	for _, fallback := range s.qqFallbacks() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		candidate, err := fallback.fetch(ctx, link.ID)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", fallback.name, err))
		}
		if isEmptyQQPlaylist(candidate) || (err != nil && len(candidate.Tracks) == 0) {
			if err == nil {
				failures = append(failures, fmt.Errorf("%s: 返回空的歌单数据", fallback.name))
			}
			continue
		}
		if err != nil {
			candidate.Warnings = append(candidate.Warnings, "QQ 音乐备用接口分页中断，结果可能不完整")
		}
		best = preferPlaylist(best, candidate)
		if qqPlaylistComplete(best) || len(best.Tracks) >= s.MaxSongs {
			return best, nil
		}
	}
	if best != nil && len(best.Tracks) > 0 {
		if len(best.Warnings) == 0 {
			best.Warnings = append(best.Warnings, "QQ 音乐未返回完整歌单，请检查公开权限或稍后重试")
		}
		return best, nil
	}
	return nil, fmt.Errorf("QQ 音乐歌单暂时不可访问，请确认歌单已公开: %w", errors.Join(failures...))
}

func qqPlaylistComplete(list *models.SongList) bool {
	return !isEmptyQQPlaylist(list) && len(list.Tracks) == list.TotalCount && len(list.Warnings) == 0
}

func (s *Service) qqPages(ctx context.Context, id int, fetchPage func(context.Context, int, int, int) (*models.QQMusicResp, error)) (*models.SongList, error) {
	var list *models.SongList
	seenPages := map[string]bool{}
	seenTracks := map[string]bool{}
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
		page, err := fetchPage(ctx, id, begin, count)
		if err != nil {
			return list, err
		}
		data := page.Req0.Data
		if list == nil {
			list = &models.SongList{Name: data.Dirinfo.Title, TotalCount: data.Dirinfo.Songnum}
		}
		list.TotalCount = max(list.TotalCount, data.Dirinfo.Songnum)
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
		before := len(list.Tracks)
		for _, song := range data.Songlist {
			track := models.Track{ID: song.ID.String(), Title: song.Name, Artists: []string{}}
			if track.ID == "" || track.ID == "0" {
				track.ID = song.Mid
			}
			for _, artist := range song.Singer {
				if artist.Name != "" {
					track.Artists = append(track.Artists, artist.Name)
				}
			}
			appendDistinctTrack(list, track, seenTracks)
		}
		if len(list.Tracks) == before {
			return list, fmt.Errorf("QQ 音乐分页没有返回新的有效歌曲")
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
			// Other platform parameters still use the same host. Leave time
			// for the alternate endpoints on transport/HTTP failures.
			return nil, fmt.Errorf("QQ 音乐分页请求失败: %w", err)
		}
		page, err := decodeQQMusicPage(body)
		if err == nil {
			return page, nil
		}
		last = err
	}
	return nil, fmt.Errorf("QQ 音乐分页请求失败: %w", last)
}

func decodeQQMusicPage(body []byte) (*models.QQMusicResp, error) {
	var page models.QQMusicResp
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}
	if page.Code != 0 || page.Req0 == nil || page.Req0.Code != 0 || page.Req0.Data == nil {
		return nil, fmt.Errorf("QQ 音乐拒绝歌单访问 (code=%d)", page.Code)
	}
	data := page.Req0.Data
	if strings.TrimSpace(data.Dirinfo.Title) == "" && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("QQ 音乐返回了无效的歌单数据")
	}
	return &page, nil
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
	cdlist := response.Cdlist
	if len(cdlist) == 0 && response.Data != nil {
		cdlist = response.Data.Cdlist
	}
	if len(cdlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回歌单")
	}
	data := cdlist[0]
	if strings.TrimSpace(data.Dissname) == "" && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回有效歌单信息")
	}
	if data.Songnum > 0 && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("旧版接口未返回歌曲")
	}
	list := &models.SongList{Name: data.Dissname, TotalCount: data.Songnum}
	seen := map[string]bool{}
	for _, song := range data.Songlist {
		title := song.Songorig
		if strings.TrimSpace(title) == "" {
			title = song.Songname
		}
		track := models.Track{ID: song.Songid.String(), Title: title, Artists: []string{}}
		if track.ID == "" || track.ID == "0" {
			track.ID = song.Songmid
		}
		for _, artist := range song.Singer {
			if artist.Name != "" {
				track.Artists = append(track.Artists, artist.Name)
			}
		}
		appendDistinctTrack(list, track, seen)
	}
	return list, nil
}

// Compatibility helpers for existing Go users and regression tests.
func QQMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return NewService(nil, nil, nil, 10000, 4).Fetch(ctx, link, Options{Detailed: detailed})
}

// isEmptyQQPlaylist 判断备用接口是否返回了无效的空壳歌单。
// 失效/私密歌单在明文接口上会返回 200 + 空数据（无名称、无歌曲），此时应视为失败，
// 保持与基线一致的「歌单不可访问」错误语义，避免返回空列表误导调用方。
func isEmptyQQPlaylist(list *models.SongList) bool {
	if list == nil {
		return true
	}
	return strings.TrimSpace(list.Name) == "" && len(list.Tracks) == 0
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
