package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

// qqPlainPages 使用无需签名的 musicu.fcg 分页拉取歌单。
// 该接口与主接口同构，但单次最多返回 30 首，需逐页请求。
func (s *Service) qqPlainPages(ctx context.Context, idText string) (*models.SongList, error) {
	id, err := strconv.Atoi(idText)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("歌单 ID 无效: %q", idText)
	}

	var list *models.SongList
	seenPages := map[string]bool{}
	seenTracks := map[string]bool{}

	for begin := 0; begin < s.MaxSongs; {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
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

		payload := models.GetQQMusicReqStringWithPagination(id, "h5", begin, count)
		address := qqMusicPlainAPIURL + "?data=" + url.QueryEscape(payload)
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{"Referer": "https://y.qq.com/"})
		if err != nil {
			return list, err
		}

		var page models.QQMusicResp
		if err = json.Unmarshal(body, &page); err != nil {
			return list, err
		}
		if page.Code != 0 || page.Req0 == nil || page.Req0.Code != 0 || page.Req0.Data == nil {
			return list, fmt.Errorf("明文接口拒绝访问 (code=%d)", page.Code)
		}
		data := page.Req0.Data
		if list == nil {
			list = &models.SongList{Name: data.Dirinfo.Title, TotalCount: data.Dirinfo.Songnum}
		}
		list.TotalCount = max(list.TotalCount, data.Dirinfo.Songnum)
		if len(data.Songlist) == 0 {
			if begin < list.TotalCount {
				return list, fmt.Errorf("明文接口第 %d 页未返回歌曲", begin/maxSongsPerPage+1)
			}
			break
		}

		fingerprint, _ := json.Marshal(data.Songlist)
		if seenPages[string(fingerprint)] {
			return list, fmt.Errorf("明文接口返回重复分页")
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
			return list, fmt.Errorf("明文接口分页没有返回新的有效歌曲")
		}
		begin += len(data.Songlist)
		if begin >= list.TotalCount {
			break
		}
	}
	if list == nil {
		return nil, fmt.Errorf("明文接口未返回有效歌单")
	}
	return list, nil
}

// qqV8Playlist 使用旧版 v8 接口一次性获取完整歌单，主域名失败时切换备用域名。
func (s *Service) qqV8Playlist(ctx context.Context, idText string) (*models.SongList, error) {
	params := url.Values{}
	params.Set("id", idText)
	params.Set("format", "json")
	params.Set("inCharset", "utf-8")
	params.Set("outCharset", "utf-8")
	query := params.Encode()

	var last error
	for _, endpoint := range []string{qqMusicV8APIURL, qqMusicV8AltURL} {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, endpoint+"?"+query, nil,
			map[string]string{"Referer": "https://y.qq.com/"})
		if err != nil {
			last = err
			continue
		}
		list, err := decodeQQV8Playlist(body)
		if err != nil {
			last = err
			continue
		}
		return list, nil
	}
	return nil, fmt.Errorf("v8 歌单接口全部域名失败: %w", last)
}

// decodeQQV8Playlist 解析 v8 接口的 data.cdlist 结构。
func decodeQQV8Playlist(body []byte) (*models.SongList, error) {
	var response struct {
		Code    int    `json:"code"`
		Subcode int    `json:"subcode"`
		Message string `json:"msg"`
		Data    struct {
			Cdlist []struct {
				Dissname string `json:"dissname"`
				Songnum  int    `json:"songnum"`
				Songlist []struct {
					Songid   json.Number `json:"songid"`
					Songmid  string      `json:"songmid"`
					Songname string      `json:"songname"`
					Singer   []struct {
						Name string `json:"name"`
					} `json:"singer"`
				} `json:"songlist"`
			} `json:"cdlist"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if response.Code != 0 || response.Subcode != 0 {
		return nil, fmt.Errorf("v8 接口业务错误(code:%d, subcode:%d): %s", response.Code, response.Subcode, response.Message)
	}
	if len(response.Data.Cdlist) == 0 {
		return nil, fmt.Errorf("v8 接口未返回歌单")
	}
	data := response.Data.Cdlist[0]
	if data.Dissname == "" && len(data.Songlist) == 0 {
		return nil, fmt.Errorf("v8 接口未返回有效歌单信息")
	}
	list := &models.SongList{Name: data.Dissname, TotalCount: data.Songnum}
	for _, song := range data.Songlist {
		track := models.Track{ID: song.Songid.String(), Title: song.Songname, Artists: []string{}}
		if track.ID == "" || track.ID == "0" {
			track.ID = song.Songmid
		}
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
	jsonBody, err := stripQQJSONP(body)
	if err != nil {
		return nil, err
	}
	// qzone 与现有 legacy 接口同构（顶层 cdlist），共用解析器。
	return decodeLegacyPlaylist(bytes.NewReader(jsonBody))
}

// stripQQJSONP 从 callback({...}) 中提取 JSON 载荷。
func stripQQJSONP(data []byte) ([]byte, error) {
	text := strings.TrimSpace(string(data))
	start := strings.Index(text, "(")
	end := strings.LastIndex(text, ")")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("无法识别 JSONP 响应: %.80s", text)
	}
	return []byte(text[start+1 : end]), nil
}
