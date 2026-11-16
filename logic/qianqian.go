package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	qianqianAppID        = "16073360"
	qianqianPageSize     = 50
	qianqianSigningKey   = "0b50b02fd0d73a9c4c8c3a781c30845f"
	qianqianTracklistAPI = "https://music.91q.com/v1/tracklist/info"
)

type qianqianResponse struct {
	State  bool   `json:"state"`
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   *struct {
		Title      string          `json:"title"`
		TrackCount json.RawMessage `json:"trackCount"`
		HaveMore   json.RawMessage `json:"haveMore"`
		TrackList  []qianqianSong  `json:"trackList"`
	} `json:"data"`
}

type qianqianSong struct {
	TSID    string          `json:"TSID"`
	ID      json.RawMessage `json:"id"`
	AssetID string          `json:"assetId"`
	Title   string          `json:"title"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artist"`
}

func (s *Service) qianqian(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	link, err := s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}

	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	seen := map[string]bool{}
	totalKnown := false
	referenceTotal := 0
	totalConflict := false
	var pageErr error
	maxPages := (s.MaxSongs + qianqianPageSize - 1) / qianqianPageSize

	for pageNo := 1; pageNo <= maxPages; pageNo++ {
		if len(list.Tracks) >= s.MaxSongs || (list.TotalCount > 0 && len(list.Tracks) >= list.TotalCount) {
			break
		}

		page, err := s.qianqianPage(ctx, link.ID, pageNo)
		if err != nil {
			pageErr = fmt.Errorf("千千音乐第 %d 页获取失败: %w", pageNo, err)
			break
		}
		if list.Name == "" {
			list.Name = strings.TrimSpace(page.Data.Title)
		}
		pageTotal, pageTotalKnown, err := qianqianCount(page.Data.TrackCount)
		if err != nil {
			pageErr = err
			break
		}
		if pageTotalKnown {
			if totalKnown && pageTotal != referenceTotal {
				totalConflict = true
			} else if !totalKnown {
				referenceTotal = pageTotal
			}
			list.TotalCount = max(list.TotalCount, pageTotal)
			totalKnown = true
		}

		before := len(list.Tracks)
		for _, song := range page.Data.TrackList {
			appendDistinctTrack(list, song.track(), seen)
			if len(list.Tracks) >= s.MaxSongs {
				break
			}
		}
		if len(page.Data.TrackList) > 0 && len(list.Tracks) == before {
			pageErr = fmt.Errorf("千千音乐分页没有返回新的有效歌曲")
			break
		}
		if len(page.Data.TrackList) == 0 {
			if totalKnown && len(list.Tracks) < list.TotalCount {
				pageErr = fmt.Errorf("千千音乐提前结束分页，平台显示 %d 首，当前获取 %d 首", list.TotalCount, len(list.Tracks))
			}
			break
		}

		hasMore := qianqianHasMore(page.Data.HaveMore)
		if totalKnown && len(list.Tracks) < list.TotalCount {
			hasMore = true
		}
		if !hasMore && len(page.Data.TrackList) == qianqianPageSize && (!totalKnown || len(list.Tracks) < list.TotalCount) {
			hasMore = true
		}
		if !hasMore {
			if totalKnown && len(list.Tracks) < list.TotalCount {
				pageErr = fmt.Errorf("千千音乐提前结束分页，平台显示 %d 首，当前获取 %d 首", list.TotalCount, len(list.Tracks))
			}
			break
		}
	}

	if len(list.Tracks) == 0 {
		if list.Name != "" && totalKnown && list.TotalCount == 0 && pageErr == nil {
			return list, nil
		}
		if pageErr != nil {
			return nil, fmt.Errorf("千千音乐未返回可读取的歌单: %w", pageErr)
		}
		return nil, fmt.Errorf("千千音乐未返回可读取的歌曲")
	}
	if list.Name == "" {
		list.Name = "千千歌单 " + link.ID
		list.Warnings = append(list.Warnings, "千千歌单名称未能读取，已使用默认名称")
	}
	if !totalKnown {
		list.TotalCount = max(list.TotalCount, len(list.Tracks))
		list.Warnings = append(list.Warnings, "千千音乐未返回可确认的歌单总数，无法确认结果是否完整")
	}
	if list.TotalCount < len(list.Tracks) {
		totalConflict = true
		list.TotalCount = len(list.Tracks)
	}
	if totalConflict {
		list.Warnings = append(list.Warnings, "千千音乐报告的歌曲总数与实际返回数不一致，已按较大值处理")
	}
	if pageErr != nil {
		list.Warnings = append(list.Warnings, pageErr.Error())
	}
	if list.TotalCount > len(list.Tracks) && pageErr == nil {
		list.Warnings = append(list.Warnings, fmt.Sprintf("千千音乐显示 %d 首，当前获取 %d 首，结果可能不完整", list.TotalCount, len(list.Tracks)))
	}
	return list, nil
}

func (s *Service) qianqianPage(ctx context.Context, id string, pageNo int) (*qianqianResponse, error) {
	query := url.Values{}
	query.Set("pageNo", strconv.Itoa(pageNo))
	query.Set("pageSize", strconv.Itoa(qianqianPageSize))
	query.Set("appid", qianqianAppID)
	query.Set("id", id)
	query.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	query.Set("sign", qianqianSign(query))
	address := qianqianTracklistAPI + "?" + query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
		"Accept":  "application/json",
		"Referer": "https://music.91q.com/player",
		"from":    "web",
	})
	if err != nil {
		return nil, err
	}

	var response qianqianResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("千千歌单接口响应格式无效: %w", err)
	}
	if !response.State || response.Data == nil {
		return nil, fmt.Errorf("千千歌单接口拒绝请求 (errno=%d): %s", response.Errno, strings.TrimSpace(response.Errmsg))
	}
	return &response, nil
}

func qianqianSign(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "sign" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + qianqianSigningKey))
	return hex.EncodeToString(sum[:])
}

func qianqianCount(raw json.RawMessage) (int, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, false, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil {
		if count < 0 {
			return 0, false, fmt.Errorf("千千歌单总数无效")
		}
		return count, true, nil
	}
	var countText string
	if err := json.Unmarshal(raw, &countText); err != nil {
		return 0, false, fmt.Errorf("千千歌单总数格式无效: %w", err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(countText))
	if err != nil || count < 0 {
		return 0, false, fmt.Errorf("千千歌单总数无效")
	}
	return count, true, nil
}

func qianqianHasMore(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" || value == "false" || value == "0" || value == `"0"` {
		return false
	}
	if value == "true" || value == `"true"` {
		return true
	}
	more, err := strconv.Atoi(strings.Trim(value, `"`))
	return err == nil && more > 0
}

func (song qianqianSong) track() models.Track {
	id := strings.TrimSpace(song.TSID)
	if id == "" {
		id = strings.TrimSpace(song.AssetID)
	}
	if id == "" {
		id = qianqianString(song.ID)
	}
	track := models.Track{ID: id, Title: strings.TrimSpace(song.Title), Artists: []string{}}
	for _, artist := range song.Artists {
		if name := strings.TrimSpace(artist.Name); name != "" {
			track.Artists = append(track.Artists, name)
		}
	}
	return track
}

func qianqianString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	number := strings.TrimSpace(string(raw))
	if number != "" && strings.IndexFunc(number, func(r rune) bool { return r < '0' || r > '9' }) == -1 {
		return number
	}
	return ""
}
