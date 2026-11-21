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
	"regexp"
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

	// The signed API is the primary channel. When it is blocked or returns an
	// incomplete first page, the official share page embeds the same trackList
	// in window.__NUXT__, giving an independent read-only recovery path.
	if len(list.Tracks) == 0 || pageErr != nil || (totalKnown && len(list.Tracks) < list.TotalCount) {
		if ctx.Err() == nil {
			page, _, pageRequestErr := s.HTTP.Do(ctx, http.MethodGet, link.URL.String(), nil,
				map[string]string{"Accept": "text/html", "Referer": "https://music.91q.com/"})
			if pageRequestErr == nil {
				candidate, parseErr := parseQianqianSharePage(page)
				if parseErr == nil && candidate != nil && len(candidate.Tracks) > 0 {
					name := list.Name
					list = preferPlaylist(list, candidate)
					if name != "" {
						list.Name = name
					}
					list.Warnings = append(list.Warnings, "千千音乐签名接口不可用，已从歌单分享页恢复曲目，完整性无法确认")
					pageErr = nil
				}
			}
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

var qianqianPlaylistName = regexp.MustCompile(`cateList:\s*\[[^\]]*\]\s*,\s*title:\s*"((?:\\.|[^"\\])*)"`)

// parseQianqianSharePage reads the public share page's embedded Nuxt payload.
// It deliberately extracts only the track list and plain text metadata; no
// script is evaluated and no playback fields are used.
func parseQianqianSharePage(body []byte) (*models.SongList, error) {
	text := string(body)
	start := strings.Index(text, "window.__NUXT__")
	if start < 0 {
		return nil, fmt.Errorf("千千分享页未包含内嵌歌单数据")
	}
	state := text[start:]
	marker := "trackList:"
	offset := strings.Index(state, marker)
	if offset < 0 {
		return nil, fmt.Errorf("千千分享页未包含曲目列表")
	}
	array, ok := qianqianBalanced(state[offset+len(marker):], '[', ']')
	if !ok {
		return nil, fmt.Errorf("千千分享页曲目列表格式无效")
	}
	vars := qianqianVariables(state)
	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	if match := qianqianPlaylistName.FindStringSubmatch(state); len(match) > 1 {
		name, err := strconv.Unquote(`"` + match[1] + `"`)
		if err != nil {
			name = match[1]
		}
		list.Name = strings.TrimSpace(name)
	}
	seen := map[string]bool{}
	rest := array
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			break
		}
		object, ok := qianqianBalanced(rest[open:], '{', '}')
		if !ok {
			break
		}
		track := models.Track{ID: qianqianJSField(object, "TSID", vars), Title: qianqianJSField(object, "title", vars), Artists: []string{}}
		if track.ID == "" {
			track.ID = qianqianJSField(object, "assetId", vars)
		}
		if artistBlock, ok := qianqianJSBlock(object, "artist", '[', ']'); ok {
			for remaining := artistBlock; ; {
				offset := qianqianFindKey(remaining, "name")
				if offset < 0 {
					break
				}
				if name := qianqianJSField(remaining[offset:], "name", vars); name != "" {
					track.Artists = append(track.Artists, name)
				}
				remaining = remaining[offset+len("name:"):]
			}
		}
		appendDistinctTrack(list, track, seen)
		rest = rest[open+len(object):]
	}
	if len(list.Tracks) == 0 {
		return nil, fmt.Errorf("千千分享页未返回可读取的歌曲")
	}
	list.TotalCount = len(list.Tracks)
	return list, nil
}

// qianqianBalanced returns the balanced region starting at the first open
// bracket, ignoring brackets inside JavaScript string literals.
func qianqianBalanced(text string, open, close byte) (string, bool) {
	start := strings.IndexByte(text, open)
	if start < 0 {
		return "", false
	}
	depth := 0
	var quote byte
	escaped := false
	for i := start; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '"' || char == '\'' || char == '`' {
			quote = char
			continue
		}
		switch char {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return text[start : i+1], true
			}
		}
	}
	return "", false
}

func qianqianJSBlock(object, key string, open, close byte) (string, bool) {
	offset := strings.Index(object, key+":")
	if offset < 0 {
		return "", false
	}
	return qianqianBalanced(object[offset+len(key)+1:], open, close)
}

// qianqianVariables maps the IIFE parameter names of window.__NUXT__ to their
// literal values. Nuxt minifies repeated strings into these parameters, so a
// track's title is often a bare identifier such as "V" rather than a literal.
func qianqianVariables(state string) map[string]string {
	vars := map[string]string{}
	if end := strings.Index(state, "</script>"); end >= 0 {
		state = state[:end]
	}
	start := strings.Index(state, "(function(")
	if start < 0 {
		return vars
	}
	paramsStart := start + len("(function(")
	paramsEnd := strings.Index(state[paramsStart:], "){")
	if paramsEnd < 0 {
		return vars
	}
	invoke := strings.LastIndex(state, "}(")
	if invoke < paramsStart+paramsEnd {
		return vars
	}
	args, ok := qianqianBalanced(state[invoke+1:], '(', ')')
	if !ok {
		return vars
	}
	params := strings.Split(state[paramsStart:paramsStart+paramsEnd], ",")
	values := qianqianSplitArguments(args[1 : len(args)-1])
	for i, param := range params {
		if i >= len(values) {
			break
		}
		vars[strings.TrimSpace(param)] = values[i]
	}
	return vars
}

func qianqianSplitArguments(args string) []string {
	values := []string{}
	depth := 0
	var quote byte
	escaped := false
	start := 0
	for i := 0; i < len(args); i++ {
		char := args[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '"', '\'', '`':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				values = append(values, strings.TrimSpace(args[start:i]))
				start = i + 1
			}
		}
	}
	values = append(values, strings.TrimSpace(args[start:]))
	for i, value := range values {
		if value == "null" || value == "undefined" {
			values[i] = ""
			continue
		}
		if value != "" && (value[0] == '"' || value[0] == '\'') {
			decoded, err := qianqianUnquote(value)
			if err == nil {
				values[i] = decoded
			}
		}
	}
	return values
}

func qianqianFindKey(object, key string) int {
	needle := key + ":"
	for offset := 0; ; {
		index := strings.Index(object[offset:], needle)
		if index < 0 {
			return -1
		}
		position := offset + index
		if position == 0 || !isQianqianKeyChar(object[position-1]) {
			return position
		}
		offset = position + 1
	}
}

func isQianqianKeyChar(char byte) bool {
	return char == '_' || char == '$' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

func qianqianJSField(object, key string, vars map[string]string) string {
	offset := qianqianFindKey(object, key)
	if offset < 0 {
		return ""
	}
	text := strings.TrimSpace(object[offset+len(key)+1:])
	if text == "" {
		return ""
	}
	if text[0] == '"' || text[0] == '\'' {
		quote := text[0]
		for i := 1; i < len(text); i++ {
			if text[i] == '\\' {
				i++
				continue
			}
			if text[i] == quote {
				value, err := qianqianUnquote(text[:i+1])
				if err != nil {
					return ""
				}
				return strings.TrimSpace(value)
			}
		}
		return ""
	}
	end := 0
	for end < len(text) && isQianqianKeyChar(text[end]) {
		end++
	}
	return strings.TrimSpace(vars[text[:end]])
}

func qianqianUnquote(literal string) (string, error) {
	if strings.HasPrefix(literal, "'") {
		inner := strings.TrimSuffix(strings.TrimPrefix(literal, "'"), "'")
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\'`, `'`)
		return inner, nil
	}
	return strconv.Unquote(literal)
}
