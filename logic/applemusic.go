package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var appleTrackCountText = regexp.MustCompile(`(?i)([0-9]{1,5})\s*(?:songs?|tracks?)\b|([0-9]{1,5})\s*(?:首歌曲|首曲目)`)
var appleTrackIDFromLockup = regexp.MustCompile(`([0-9]{6,})$`)

func (s *Service) appleMusicPlaylist(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	list, err := s.appleMusicPublicPage(ctx, link)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("Apple Music 未返回可读取的公开歌单: %w", err)
	}
	if list == nil || len(list.Tracks) == 0 {
		return nil, fmt.Errorf("Apple Music 未返回可读取的公开歌单")
	}
	list.Warnings = append(list.Warnings, "Apple Music 使用公开歌单页读取；页面结构不稳定，结果可能不完整")
	return list, nil
}

func (s *Service) appleMusicPublicPage(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	resourceName := "歌单"
	address := link.URL.String()
	body, finalURL, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": appleAcceptLanguage(link.URL),
		"Referer":         "https://music.apple.com/",
	})
	if err != nil {
		return nil, err
	}
	if finalURL == nil {
		return nil, fmt.Errorf("Apple Music %s页没有返回最终地址", resourceName)
	}
	resolved, err := musicurl.Parse(finalURL.String())
	if err != nil || resolved.Provider != musicurl.Apple || resolved.Kind != link.Kind || resolved.ID != link.ID {
		return nil, fmt.Errorf("Apple Music %s链接跳转到不同的内容", resourceName)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Apple Music %s页格式无效: %w", resourceName, err)
	}
	var serialized string
	doc.Find("script[type='application/json']").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		id, _ := script.Attr("id")
		text := script.Text()
		if id == "serialized-server-data" || strings.Contains(text, "\"track-list") || strings.Contains(text, "\"track-lockup") {
			serialized = strings.TrimSpace(text)
			return false
		}
		return true
	})
	if serialized == "" {
		return nil, fmt.Errorf("Apple Music %s页未包含可读取的曲目数据", resourceName)
	}
	var root any
	decoder := json.NewDecoder(strings.NewReader(serialized))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("Apple Music %s页数据格式无效: %w", resourceName, err)
	}

	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	sections := applePageSections(root)
	var trackSection map[string]any
	trackSectionScore := 0
	for _, section := range sections {
		sectionID := strings.ToLower(appleMapString(section, "id"))
		if strings.Contains(sectionID, "detail-header") && list.Name == "" {
			if items, ok := section["items"].([]any); ok && len(items) > 0 {
				if item, ok := items[0].(map[string]any); ok {
					list.Name = strings.TrimSpace(appleMapString(item, "title"))
				}
			}
		}
		trackItems := applePageTrackItems(section)
		score := len(trackItems)
		if score > 0 && strings.Contains(sectionID, "track-list") {
			score += 10000
		}
		if score > trackSectionScore {
			trackSection = section
			trackSectionScore = score
		}
	}
	if trackSection == nil || trackSectionScore == 0 {
		return nil, fmt.Errorf("Apple Music %s页未返回曲目列表区块", resourceName)
	}
	seen := map[string]bool{}
	for _, item := range applePageTrackItems(trackSection) {
		track := models.Track{ID: applePageTrackID(item), Title: strings.TrimSpace(appleMapString(item, "title")), Artists: applePageArtists(item)}
		appendDistinctTrack(list, track, seen)
	}
	if len(list.Tracks) == 0 {
		return nil, fmt.Errorf("Apple Music %s页未返回可读取的曲目", resourceName)
	}
	if list.Name == "" {
		if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
			list.Name = applePageTitle(title)
		}
	}
	if list.Name == "" {
		list.Name = "Apple Music " + resourceName
		list.Warnings = append(list.Warnings, "Apple Music 名称未能读取，已使用默认名称")
	}
	if count := applePageCount(trackSection); count > 0 {
		list.TotalCount = count
	} else if count := appleCountFromText(doc.Find("meta[name='description']").AttrOr("content", "")); count > 0 {
		list.TotalCount = count
	} else {
		list.TotalCount = len(list.Tracks)
		list.Warnings = append(list.Warnings, "Apple Music 公开页面未提供可确认的曲目总数")
	}
	if list.TotalCount < len(list.Tracks) {
		list.TotalCount = len(list.Tracks)
		list.Warnings = append(list.Warnings, "Apple Music 总数小于页面返回曲目数，完整性无法确认")
	}
	if list.TotalCount > len(list.Tracks) {
		list.Warnings = append(list.Warnings, fmt.Sprintf("Apple Music 显示 %d 首，页面当前返回 %d 首，结果可能不完整", list.TotalCount, len(list.Tracks)))
	}
	return list, nil
}

func applePageTrackItems(section map[string]any) []map[string]any {
	var items []map[string]any
	var walk func(any)
	walk = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			id := strings.ToLower(appleMapString(node, "id"))
			if strings.HasPrefix(id, "track-lockup") || applePageTrackCandidate(node) {
				items = append(items, node)
				return
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	if sectionItems, ok := section["items"]; ok {
		walk(sectionItems)
	} else {
		walk(section)
	}
	return items
}

func applePageTrackCandidate(item map[string]any) bool {
	if strings.TrimSpace(appleMapString(item, "title")) == "" || applePageTrackID(item) == "" {
		return false
	}
	return len(applePageArtists(item)) > 0 || applePageInt(item["trackNumber"]) > 0
}

func applePageSections(root any) []map[string]any {
	var sections []map[string]any
	var walk func(any)
	walk = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			if rawSections, ok := node["sections"].([]any); ok {
				for _, rawSection := range rawSections {
					if section, ok := rawSection.(map[string]any); ok {
						sections = append(sections, section)
					}
				}
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(root)
	return sections
}

func applePageCount(section map[string]any) int {
	for _, key := range []string{"trackCount", "totalCount", "total"} {
		if count := applePageInt(section[key]); count > 0 {
			return count
		}
	}
	return 0
}

func applePageTrackID(item map[string]any) string {
	contentDescriptor, _ := item["contentDescriptor"].(map[string]any)
	identifiers, _ := contentDescriptor["identifiers"].(map[string]any)
	if id := applePageString(identifiers["storeAdamID"]); id != "" {
		return id
	}
	for _, key := range []string{"trackId", "songId", "adamId"} {
		if id := applePageString(item[key]); id != "" {
			return id
		}
	}
	if match := appleTrackIDFromLockup.FindStringSubmatch(appleMapString(item, "id")); len(match) > 1 {
		return match[1]
	}
	return ""
}

func applePageArtists(item map[string]any) []string {
	if artist := strings.TrimSpace(appleMapString(item, "artistName")); artist != "" {
		return []string{artist}
	}
	if raw, ok := item["artistNames"].([]any); ok {
		artists := make([]string, 0, len(raw))
		for _, value := range raw {
			if artist := strings.TrimSpace(applePageString(value)); artist != "" {
				artists = append(artists, artist)
			}
		}
		return artists
	}
	if artist := strings.TrimSpace(appleMapString(item, "artist")); artist != "" {
		return []string{artist}
	}
	if raw, ok := item["subtitleLinks"].([]any); ok {
		var names []string
		for _, value := range raw {
			if link, ok := value.(map[string]any); ok {
				if name := strings.TrimSpace(appleMapString(link, "title")); name != "" {
					names = append(names, name)
				}
			}
		}
		if len(names) > 0 {
			return []string{strings.Join(names, ", ")}
		}
	}
	return []string{}
}

func appleMapString(values map[string]any, key string) string {
	return applePageString(values[key])
}

func applePageString(value any) string {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item)
	case json.Number:
		return item.String()
	case float64:
		return fmt.Sprintf("%.0f", item)
	default:
		return ""
	}
}

func applePageInt(value any) int {
	text := applePageString(value)
	count, err := strconv.Atoi(text)
	if err != nil || count < 0 {
		return 0
	}
	return count
}

func appleCountFromText(text string) int {
	matches := appleTrackCountText.FindStringSubmatch(text)
	if len(matches) < 3 {
		return 0
	}
	value := matches[1]
	if value == "" {
		value = matches[2]
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 0 {
		return 0
	}
	return count
}

func appleAcceptLanguage(address *url.URL) string {
	if strings.HasPrefix(strings.ToLower(strings.Trim(address.Path, "/")), "cn/") {
		return "zh-CN,zh;q=0.9,en;q=0.8"
	}
	return "en-US,en;q=0.9"
}

func applePageTitle(title string) string {
	title = strings.TrimSuffix(strings.TrimSpace(title), " | Apple Music")
	if name, _, ok := strings.Cut(title, " - Playlist by "); ok {
		return strings.TrimSpace(name)
	}
	for _, suffix := range []string{" - Apple Music", " - Playlist", " - 歌单"} {
		title = strings.TrimSuffix(strings.TrimSpace(title), suffix)
	}
	return strings.TrimSpace(title)
}
