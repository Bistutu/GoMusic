package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"context"
	"crypto/md5"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const bodianPageSize = 100

const bodianAPIBase = "https://bd-api.kuwo.cn/api/service/playlist"

type bodianInfoResponse struct {
	Code json.RawMessage `json:"code"`
	Msg  string          `json:"msg"`
	Data *struct {
		Name       string          `json:"name"`
		MusicCount json.RawMessage `json:"musicCount"`
		IsPrivate  json.RawMessage `json:"isPrivate"`
	} `json:"data"`
}

type bodianPageResponse struct {
	Code json.RawMessage `json:"code"`
	Msg  string          `json:"msg"`
	Data *struct {
		Total json.RawMessage `json:"total"`
		List  []bodianSong    `json:"list"`
	} `json:"data"`
}

type bodianSong struct {
	ID     json.RawMessage `json:"id"`
	RID    json.RawMessage `json:"rid"`
	Name   string          `json:"name"`
	Title  string          `json:"songName"`
	Artist string          `json:"artist"`
}

type bodianPlaylistInfo struct {
	Name       string
	TotalCount int
	TotalKnown bool
	IsPrivate  json.RawMessage
}

func (song bodianSong) track() models.Track {
	id := bodianScalar(song.ID)
	if id == "" {
		id = bodianScalar(song.RID)
	}
	title := strings.TrimSpace(song.Name)
	if title == "" {
		title = strings.TrimSpace(song.Title)
	}
	artist := strings.TrimSpace(song.Artist)
	artists := []string{}
	if artist != "" {
		artists = append(artists, artist)
	}
	return models.Track{ID: id, Title: title, Artists: artists}
}

func bodianScalar(raw json.RawMessage) string {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return ""
	}
	if len(value) > 1 && value[0] == '"' {
		var decoded string
		if json.Unmarshal(raw, &decoded) == nil {
			return strings.TrimSpace(decoded)
		}
	}
	return value
}

func (s *Service) bodian(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	source := strings.TrimSpace(link.URL.Query().Get("source"))
	if source == "" {
		source = strings.TrimSpace(link.URL.Query().Get("sourceType"))
	}
	if source == "" {
		source = "5"
	}
	deviceID, err := bodianDeviceID()
	if err != nil {
		return nil, fmt.Errorf("生成波点音乐请求标识失败: %w", err)
	}
	refererQuery := url.Values{"playlistId": {link.ID}, "source": {source}}
	referer := "https://h5app.kuwo.cn/m/bodian/collection.html?" + refererQuery.Encode()
	headers := map[string]string{
		"User-Agent":   "Dart/3.3 (dart:io)",
		"plat":         "win",
		"api-ver":      "application/json",
		"channel":      "W1",
		"brand":        "Windows 11 Pro for Workstations",
		"net":          "wifi",
		"Content-Type": "application/json",
		"ver":          "1.1.5",
		"svrver":       "13",
		"devid":        deviceID,
		"qimei36":      deviceID,
		"Referer":      referer,
	}

	info, infoErr := s.bodianInfo(ctx, link.ID, source, headers)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if info != nil {
		private, known, err := bodianPrivate(info.IsPrivate)
		if err != nil {
			infoErr = errors.Join(infoErr, err)
		} else if known && private {
			return nil, ErrPrivatePlaylist
		}
	}

	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	totalKnown := false
	referenceTotal := 0
	if info != nil {
		list.Name = strings.TrimSpace(info.Name)
		list.TotalCount = info.TotalCount
		totalKnown = info.TotalKnown
		if totalKnown {
			referenceTotal = info.TotalCount
		}
	}

	seenTracks := map[string]bool{}
	seenPages := map[string]bool{}
	totalConflict := false
	var pageErr error
	maxPages := (s.MaxSongs + bodianPageSize - 1) / bodianPageSize
	for pageNo := 1; pageNo <= maxPages; pageNo++ {
		if len(list.Tracks) >= s.MaxSongs || (totalKnown && list.TotalCount > 0 && len(list.Tracks) >= list.TotalCount) {
			break
		}
		count := min(bodianPageSize, s.MaxSongs-len(list.Tracks))
		query := url.Values{}
		query.Set("source", source)
		query.Set("pn", strconv.Itoa(pageNo))
		query.Set("rn", strconv.Itoa(count))
		query.Set("uid", "-1")
		query.Set("token", "")
		address := bodianAPIBase + "/" + url.PathEscape(link.ID) + "/musicList?" + query.Encode()
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, headers)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			pageErr = fmt.Errorf("波点音乐第 %d 页获取失败: %w", pageNo, err)
			break
		}

		var page bodianPageResponse
		if err := json.Unmarshal(body, &page); err != nil {
			pageErr = fmt.Errorf("波点音乐分页响应格式无效: %w", err)
			break
		}
		if err := bodianStatus(page.Code, page.Msg); err != nil {
			pageErr = err
			break
		}
		if page.Data == nil {
			pageErr = fmt.Errorf("波点音乐未返回歌单曲目")
			break
		}
		pageTotal, pageTotalKnown, err := kuwoTotal(page.Data.Total)
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
		if len(page.Data.List) == 0 {
			if totalKnown && len(list.Tracks) < list.TotalCount {
				pageErr = fmt.Errorf("波点音乐提前结束分页，平台显示 %d 首，当前获取 %d 首", list.TotalCount, len(list.Tracks))
			}
			break
		}
		fingerprint, _ := json.Marshal(page.Data.List)
		if seenPages[string(fingerprint)] {
			pageErr = fmt.Errorf("波点音乐返回重复分页")
			break
		}
		seenPages[string(fingerprint)] = true
		before := len(list.Tracks)
		for _, song := range page.Data.List {
			appendDistinctTrack(list, song.track(), seenTracks)
			if len(list.Tracks) >= s.MaxSongs {
				break
			}
		}
		if len(list.Tracks) == before {
			pageErr = fmt.Errorf("波点音乐分页没有返回新的有效歌曲")
			break
		}
		if totalKnown && len(list.Tracks) >= list.TotalCount {
			break
		}
		if !totalKnown && len(page.Data.List) < count {
			break
		}
	}

	if len(list.Tracks) == 0 {
		if list.Name != "" && totalKnown && list.TotalCount == 0 && pageErr == nil {
			return list, nil
		}
		return nil, fmt.Errorf("波点音乐未返回可读取的公开歌单: %w", errors.Join(infoErr, pageErr))
	}
	if list.Name == "" {
		list.Name = "波点歌单"
		list.Warnings = append(list.Warnings, "波点歌单名称未能读取，已使用默认名称")
	}
	if infoErr != nil {
		list.Warnings = append(list.Warnings, "波点歌单元信息未能读取，完整性以曲目接口为准")
	}
	if !totalKnown {
		list.TotalCount = max(list.TotalCount, len(list.Tracks))
		list.Warnings = append(list.Warnings, "波点音乐未返回可确认的歌单总数")
	}
	if list.TotalCount < len(list.Tracks) {
		list.TotalCount = len(list.Tracks)
		totalConflict = true
	}
	if totalConflict {
		list.Warnings = append(list.Warnings, "波点音乐接口报告的歌曲总数不一致，已按较大值保留")
	}
	if pageErr != nil {
		list.Warnings = append(list.Warnings, pageErr.Error())
	}
	if len(list.Tracks) < list.TotalCount && pageErr == nil {
		list.Warnings = append(list.Warnings, fmt.Sprintf("波点音乐显示 %d 首，实际获取 %d 首，结果可能不完整", list.TotalCount, len(list.Tracks)))
	}
	if len(list.Tracks) >= s.MaxSongs && list.TotalCount > len(list.Tracks) {
		list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首", s.MaxSongs))
	}
	return list, nil
}

func (s *Service) bodianInfo(ctx context.Context, id, source string, headers map[string]string) (*bodianPlaylistInfo, error) {
	query := url.Values{}
	query.Set("source", source)
	query.Set("uid", "-1")
	query.Set("token", "")
	address := bodianAPIBase + "/info/" + url.PathEscape(id) + "?" + query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, headers)
	if err != nil {
		return nil, err
	}
	var response bodianInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("波点歌单元信息格式无效: %w", err)
	}
	if err := bodianStatus(response.Code, response.Msg); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, fmt.Errorf("波点歌单元信息为空")
	}
	total, known, err := kuwoTotal(response.Data.MusicCount)
	if err != nil {
		return nil, err
	}
	return &bodianPlaylistInfo{Name: strings.TrimSpace(response.Data.Name), TotalCount: total, TotalKnown: known, IsPrivate: response.Data.IsPrivate}, nil
}

func bodianStatus(raw json.RawMessage, message string) error {
	code, known, err := kuwoTotal(raw)
	if err != nil {
		return fmt.Errorf("波点音乐接口状态格式无效: %w", err)
	}
	if known && code != 200 {
		return fmt.Errorf("波点音乐接口请求失败 (code=%d): %s", code, strings.TrimSpace(message))
	}
	return nil
}

func bodianPrivate(raw json.RawMessage) (bool, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return false, false, nil
	}
	var boolean bool
	if err := json.Unmarshal(raw, &boolean); err == nil {
		return boolean, true, nil
	}
	if len(value) > 1 && value[0] == '"' {
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return false, false, err
		}
		value = strings.TrimSpace(decoded)
	}
	private, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, false, fmt.Errorf("波点歌单公开状态格式无效")
	}
	return private != 0, true, nil
}

func bodianDeviceID() (string, error) {
	var random [16]byte
	if _, err := cryptorand.Read(random[:]); err != nil {
		return "", err
	}
	device := md5.Sum(random[:])
	return hex.EncodeToString(device[:]), nil
}
