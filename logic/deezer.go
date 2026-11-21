package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const deezerPageSize = 100
const deezerAPIURL = "https://api.deezer.com"

type deezerAPIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type deezerCollectionInfo struct {
	Error    *deezerAPIError `json:"error"`
	ID       int64           `json:"id"`
	Title    string          `json:"title"`
	Public   *bool           `json:"public"`
	NbTracks json.RawMessage `json:"nb_tracks"`
	Tracks   json.RawMessage `json:"tracks"`
}

type deezerTrackRow struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
}

type deezerTrackPage struct {
	Error      *deezerAPIError  `json:"error"`
	Total      json.RawMessage  `json:"total"`
	Next       string           `json:"next"`
	Data       []deezerTrackRow `json:"data"`
	TotalCount int              `json:"-"`
	TotalKnown bool             `json:"-"`
}

func (apiErr *deezerAPIError) Error() string {
	if apiErr == nil {
		return ""
	}
	return fmt.Sprintf("Deezer API 请求失败 (code=%d): %s", apiErr.Code, apiErr.Message)
}

func (s *Service) deezer(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	if link.ID == "" {
		resolved, err := s.resolve(ctx, link)
		if err != nil {
			return nil, fmt.Errorf("Deezer 分享链接无法解析: %w", err)
		}
		if resolved.Provider != musicurl.Deezer || resolved.ID == "" || resolved.Kind != "playlist" {
			return nil, fmt.Errorf("%w: Deezer 分享链接没有有效公开歌单 ID", ErrInvalidLink)
		}
		link = resolved
	}
	if link.Kind != "playlist" {
		return nil, fmt.Errorf("%w: Deezer 当前仅支持公开歌单", ErrInvalidLink)
	}
	const kind = "playlist"
	collectionID, err := strconv.ParseInt(link.ID, 10, 64)
	if err != nil || collectionID <= 0 {
		return nil, fmt.Errorf("Deezer 歌单 ID 无效")
	}

	infoURL := deezerAPIURL + "/" + kind + "/" + url.PathEscape(link.ID)
	infoBody, _, err := s.HTTP.Do(ctx, "GET", infoURL, nil, map[string]string{
		"Accept":  "application/json",
		"Referer": "https://www.deezer.com/",
	})
	if err != nil {
		return nil, fmt.Errorf("Deezer 歌单详情暂时不可访问: %w", err)
	}
	var info deezerCollectionInfo
	if err := json.Unmarshal(infoBody, &info); err != nil {
		return nil, fmt.Errorf("Deezer 歌单信息格式无效: %w", err)
	}
	if info.Error != nil {
		return nil, info.Error
	}
	if info.ID != collectionID {
		return nil, fmt.Errorf("Deezer 返回了不同的歌单")
	}
	// The tracks endpoint has no public field. Require explicit visibility
	// from the detail endpoint before trying either source of tracks.
	if info.Public == nil {
		return nil, fmt.Errorf("Deezer 未能确认歌单为公开状态")
	}
	if !*info.Public {
		return nil, ErrPrivatePlaylist
	}

	list := &models.SongList{Name: strings.TrimSpace(info.Title)}
	total, totalKnown, totalErr := deezerCount(info.NbTracks)
	if totalErr != nil {
		list.Warnings = append(list.Warnings, "Deezer 歌单总数格式无效")
	} else if totalKnown {
		list.TotalCount = total
	}
	totalConflictWarned := false
	recordTotal := func(page *deezerTrackPage) {
		if page == nil || !page.TotalKnown {
			return
		}
		if totalKnown && page.TotalCount != list.TotalCount && !totalConflictWarned {
			list.Warnings = append(list.Warnings, "Deezer 歌单信息与歌曲接口报告的总数不一致")
			totalConflictWarned = true
		}
		list.TotalCount = max(list.TotalCount, page.TotalCount)
		totalKnown = true
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(deezerPageSize))
	query.Set("index", "0")
	pageURL := deezerAPIURL + "/" + kind + "/" + url.PathEscape(link.ID) + "/tracks?" + query.Encode()
	nextIndex := 0
	seenURLs := map[string]bool{}
	missingIDs := false
	maxPages := (s.MaxSongs + deezerPageSize - 1) / deezerPageSize
	if maxPages < 1 {
		maxPages = 1
	}

	for pageNo := 0; pageNo < maxPages; pageNo++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if seenURLs[pageURL] {
			list.Warnings = append(list.Warnings, "Deezer 分页链接重复，结果可能不完整")
			break
		}
		seenURLs[pageURL] = true
		body, _, requestErr := s.HTTP.Do(ctx, "GET", pageURL, nil, map[string]string{
			"Accept":  "application/json",
			"Referer": "https://www.deezer.com/",
		})
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var page *deezerTrackPage
		if requestErr == nil {
			page, requestErr = decodeDeezerPage(body)
		}
		recordTotal(page)
		if requestErr == nil && len(page.Data) == 0 && (!totalKnown || len(list.Tracks) < list.TotalCount || page.Next != "") {
			requestErr = fmt.Errorf("Deezer 没有返回可读取的歌曲")
		}
		if requestErr != nil && pageNo == 0 && len(info.Tracks) > 0 {
			embedded, embeddedErr := decodeDeezerPage(info.Tracks)
			recordTotal(embedded)
			if embeddedErr == nil && (len(embedded.Data) > 0 || (totalKnown && list.TotalCount == 0 && embedded.Next == "")) {
				appendDeezerRows(list, embedded.Data, s.MaxSongs, &missingIDs)
				if embedded.Next != "" {
					list.Warnings = append(list.Warnings, "Deezer 已从详情恢复首屏曲目，后续分页不可用")
				}
				break
			}
		}
		if requestErr != nil {
			if len(list.Tracks) == 0 {
				return nil, fmt.Errorf("Deezer 歌曲第 %d 页获取失败: %w", pageNo+1, requestErr)
			}
			list.Warnings = append(list.Warnings, fmt.Sprintf("Deezer 歌曲第 %d 页获取失败，结果可能不完整", pageNo+1))
			break
		}
		if len(page.Data) == 0 {
			break
		}
		appendDeezerRows(list, page.Data, s.MaxSongs, &missingIDs)
		if len(list.Tracks) >= s.MaxSongs {
			if page.Next != "" && totalKnown && list.TotalCount <= len(list.Tracks) {
				list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首，Deezer 仍有后续页", s.MaxSongs))
			}
			break
		}
		if page.Next == "" {
			break
		}

		nextURL, nextOffset, err := validateDeezerNext(page.Next, link.ID, nextIndex+len(page.Data))
		if err != nil {
			list.Warnings = append(list.Warnings, "Deezer 分页链接无效，结果可能不完整")
			break
		}
		pageURL = nextURL
		nextIndex = nextOffset
	}

	if list.Name == "" {
		list.Name = "Deezer " + kind + " " + link.ID
	}
	if missingIDs {
		list.Warnings = append(list.Warnings, "Deezer 部分歌曲缺少稳定 ID，完整性无法确认")
	}
	if !totalKnown {
		list.TotalCount = len(list.Tracks)
		list.Warnings = append(list.Warnings, "Deezer 未返回可确认的歌曲总数")
	} else if list.TotalCount < len(list.Tracks) {
		list.Warnings = append(list.Warnings, "Deezer 报告的歌曲总数小于实际返回数")
		list.TotalCount = len(list.Tracks)
	}
	if len(list.Tracks) >= s.MaxSongs && (!totalKnown || list.TotalCount > len(list.Tracks)) {
		list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首", s.MaxSongs))
	}
	if totalKnown && len(list.Tracks) < list.TotalCount {
		list.Warnings = append(list.Warnings, fmt.Sprintf("Deezer 已获取 %d 首，平台显示 %d 首，结果不完整", len(list.Tracks), list.TotalCount))
	}
	return list, nil
}

func decodeDeezerPage(body []byte) (*deezerTrackPage, error) {
	var page deezerTrackPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("Deezer 歌曲分页响应无效: %w", err)
	}
	if page.Error != nil {
		return nil, page.Error
	}
	total, known, err := deezerCount(page.Total)
	if err != nil {
		return nil, fmt.Errorf("Deezer 歌曲总数格式无效: %w", err)
	}
	page.TotalCount, page.TotalKnown = total, known
	if page.Data != nil && len(page.Data) == 0 {
		return &page, nil
	}
	for _, row := range page.Data {
		if strings.TrimSpace(row.Title) != "" {
			return &page, nil
		}
	}
	return &page, fmt.Errorf("Deezer 没有返回可读取的歌曲")
}

func appendDeezerRows(list *models.SongList, rows []deezerTrackRow, maxSongs int, missingIDs *bool) {
	for _, item := range rows {
		if len(list.Tracks) >= maxSongs {
			return
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		id := ""
		if item.ID > 0 {
			id = strconv.FormatInt(item.ID, 10)
		} else {
			*missingIDs = true
		}
		artists := []string{}
		if name := strings.TrimSpace(item.Artist.Name); name != "" {
			artists = append(artists, name)
		}
		list.Tracks = append(list.Tracks, models.Track{ID: id, Title: title, Artists: artists})
	}
}

func deezerCount(raw json.RawMessage) (int, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, false, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil {
		if count < 0 {
			return 0, false, fmt.Errorf("negative total")
		}
		return count, true, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false, err
	}
	count, err := strconv.Atoi(text)
	if err != nil || count < 0 {
		return 0, false, fmt.Errorf("invalid total %q", text)
	}
	return count, true, nil
}

func validateDeezerNext(raw, id string, expectedIndex int) (string, int, error) {
	nextURL, err := url.Parse(raw)
	if err != nil {
		return "", 0, err
	}
	if !strings.EqualFold(nextURL.Scheme, "https") || !strings.EqualFold(nextURL.Hostname(), "api.deezer.com") || nextURL.User != nil || nextURL.Port() != "" || nextURL.Fragment != "" {
		return "", 0, errors.New("Deezer 分页链接离开了官方 API 域名")
	}
	expectedPath := "/playlist/" + id + "/tracks"
	if nextURL.Path != expectedPath {
		return "", 0, errors.New("Deezer 分页链接切换了歌单")
	}
	query := nextURL.Query()
	limit, limitErr := strconv.Atoi(query.Get("limit"))
	index, indexErr := strconv.Atoi(query.Get("index"))
	if limitErr != nil || indexErr != nil || len(query) != 2 || limit != deezerPageSize || index != expectedIndex {
		return "", 0, errors.New("Deezer 分页链接的页码参数异常")
	}
	return nextURL.String(), index, nil
}
