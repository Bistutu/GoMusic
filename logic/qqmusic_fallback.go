package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"GoMusic/misc/models"
)

// 备用歌单接口。主签名接口 u6.y.qq.com 一旦签名失效或域名不可用，
// 这些公开只读接口依次接棒；均无需登录态。
const (
	qqMusicPlainAPIURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	qqMusicV8APIURL    = "https://c.y.qq.com/v8/fcg-bin/fcg_v8_playlist_cp.fcg"
	qqMusicV8AltURL    = "https://i.y.qq.com/v8/fcg-bin/fcg_v8_playlist_cp.fcg"
	qqMusicQzoneAPIURL = "https://i.y.qq.com/qzone-music/fcg-bin/fcg_ucc_getcdinfo_byids_cp.fcg"
)

// qqFallback 是一种备用歌单接口，返回统一的 SongList。
type qqFallback func(ctx context.Context, id string) (*models.SongList, error)

// qqFallbacks 按优先级排列。v8/qzone 一次请求即可返回整份歌单，远快于需要
// 逐页拉取的明文 musicu，因此放在分页接口之前；只有当前者返回的数据不完整时
// 才会继续尝试后续接口补全。legacy 保留在最前以维持原有的错误语义。
func (s *Service) qqFallbacks() []struct {
	name  string
	fetch qqFallback
} {
	return []struct {
		name  string
		fetch qqFallback
	}{
		{"legacy", s.qqLegacy},
		{"v8-playlist", s.qqV8Playlist},
		{"qzone-music", s.qqQzoneMusic},
		{"plain-musicu", s.qqPlainPages},
	}
}

func (s *Service) qqPlainPages(ctx context.Context, idText string) (*models.SongList, error) {
	id, err := strconv.Atoi(idText)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("歌单 ID 无效: %q", idText)
	}
	return s.qqPages(ctx, id, s.qqPlainPage)
}

func (s *Service) qqPlainPage(ctx context.Context, id, begin, count int) (*models.QQMusicResp, error) {
	payload := models.GetQQMusicReqStringWithPagination(id, "h5", begin, count)
	address := qqMusicPlainAPIURL + "?data=" + url.QueryEscape(payload)
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return nil, err
	}
	return decodeQQMusicPage(body)
}

// Try the alternate host if the first host fails or returns a shorter list.
func (s *Service) qqV8Playlist(ctx context.Context, idText string) (*models.SongList, error) {
	params := url.Values{
		"id": {idText}, "format": {"json"}, "inCharset": {"utf-8"}, "outCharset": {"utf-8"},
	}
	var best *models.SongList
	var failures []error
	for _, endpoint := range []string{qqMusicV8APIURL, qqMusicV8AltURL} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil,
			map[string]string{"Referer": "https://y.qq.com/"})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		list, err := decodeLegacyPlaylist(bytes.NewReader(body))
		if err != nil {
			failures = append(failures, err)
			continue
		}
		best = preferPlaylist(best, list)
		if qqPlaylistComplete(best) || len(best.Tracks) >= s.MaxSongs {
			return best, nil
		}
	}
	if best != nil {
		return best, nil
	}
	return nil, fmt.Errorf("v8 歌单接口全部域名失败: %w", errors.Join(failures...))
}

// qqQzoneMusic 使用 qzone-music 的 JSONP 接口获取完整歌单。
func (s *Service) qqQzoneMusic(ctx context.Context, idText string) (*models.SongList, error) {
	params := url.Values{}
	params.Set("type", "1")
	params.Set("utf8", "1")
	params.Set("onlysong", "0")
	params.Set("nosign", "1")
	params.Set("jsonpCallback", "callback")
	params.Set("disstid", idText)
	params.Set("format", "json")
	query := params.Encode()

	body, _, err := s.HTTP.Do(ctx, http.MethodGet, qqMusicQzoneAPIURL+"?"+query, nil,
		map[string]string{"Referer": "https://y.qq.com/"})
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		body, err = stripQQJSONP(body)
		if err != nil {
			return nil, err
		}
	}
	return decodeLegacyPlaylist(bytes.NewReader(body))
}

func stripQQJSONP(data []byte) ([]byte, error) {
	data = bytes.TrimSpace(bytes.TrimSuffix(bytes.TrimSpace(data), []byte(";")))
	start := bytes.IndexByte(data, '(')
	if start < 0 || !bytes.Equal(bytes.TrimSpace(data[:start]), []byte("callback")) || !bytes.HasSuffix(data, []byte(")")) {
		return nil, fmt.Errorf("无法识别 QQ 音乐 JSONP 响应")
	}
	payload := data[start+1 : len(data)-1]
	if !json.Valid(payload) {
		return nil, fmt.Errorf("QQ 音乐 JSONP 不包含有效 JSON")
	}
	return payload, nil
}
