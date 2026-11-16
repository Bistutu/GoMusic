package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const kuwoMetadataPageSize = 1000
const kuwoH5PageSize = 100
const kuwoSharePageSize = 30
const kuwoHomeURL = "https://www.kuwo.cn/"
const kuwoSharePageAPIURL = "https://www.kuwo.cn/newh5/playlist/playlistByPage"

type kuwoPage struct {
	Result    string          `json:"result"`
	Title     string          `json:"title"`
	Total     json.RawMessage `json:"total"`
	Musiclist []kuwoSong      `json:"musiclist"`
}

type kuwoH5Page struct {
	Code json.RawMessage `json:"code"`
	Msg  string          `json:"msg"`
	Data *struct {
		Name      string          `json:"name"`
		Total     json.RawMessage `json:"total"`
		MusicList []kuwoSong      `json:"musicList"`
	} `json:"data"`
}

type kuwoSigningCookie struct {
	Name  string
	Value string
}

type kuwoSong struct {
	ID           string `json:"id"`
	MusicRID     string `json:"musicrid"`
	Name         string `json:"name"`
	SourceName   string `json:"FSONGNAME"`
	Artist       string `json:"artist"`
	SourceArtist string `json:"FARTIST"`
}

type kuwoMetadata struct {
	List       *models.SongList
	TotalKnown bool
}

func (song kuwoSong) track() models.Track {
	id := strings.TrimSpace(song.ID)
	if id == "" {
		id = strings.TrimSpace(song.MusicRID)
	}
	if strings.HasPrefix(strings.ToUpper(id), "MUSIC_") {
		musicID := id[len("MUSIC_"):]
		if musicID != "" && allASCIIDigits(musicID) {
			id = musicID
		}
	}
	title := strings.TrimSpace(song.Name)
	if title == "" {
		title = strings.TrimSpace(song.SourceName)
	}
	artist := strings.TrimSpace(song.Artist)
	if artist == "" {
		artist = strings.TrimSpace(song.SourceArtist)
	}
	artists := []string{}
	if artist != "" {
		artists = append(artists, artist)
	}
	return models.Track{ID: id, Title: title, Artists: artists}
}

func (s *Service) kuwo(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	link, err := s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}
	page, hasMore, pageErr := s.kuwoSharePage(ctx, link.ID)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	metadata, metadataErr := s.kuwoMetadata(ctx, link.ID)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if pageErr == nil && page != nil && len(page.Tracks) > 0 {
		if metadata != nil && metadata.List != nil {
			page.Warnings = append(page.Warnings, metadata.List.Warnings...)
			if page.Name == "" {
				page.Name = metadata.List.Name
			}
			page.TotalCount = max(page.TotalCount, metadata.List.TotalCount)
		}
		page.TotalCount = max(page.TotalCount, len(page.Tracks))
		if metadata == nil || !metadata.TotalKnown {
			page.Warnings = append(page.Warnings, "酷我音乐未返回可确认的歌单总数，当前页面内容可能只是预览")
		} else if len(page.Tracks) != metadata.List.TotalCount {
			page.Warnings = append(page.Warnings, fmt.Sprintf("酷我网页显示 %d 首，接口总数为 %d 首，当前结果可能不完整", len(page.Tracks), metadata.List.TotalCount))
		}
		if len(page.Tracks) >= s.MaxSongs && page.TotalCount > len(page.Tracks) {
			page.Warnings = append(page.Warnings, fmt.Sprintf("达到本次导出上限 %d 首", s.MaxSongs))
		}
		if hasMore && (metadata == nil || !metadata.TotalKnown) {
			page.Warnings = append(page.Warnings, "酷我分享页还有后续歌曲未载入")
		}
		return page, nil
	}

	h5Fallback, h5FallbackErr := s.kuwoH5Pages(ctx, link.ID)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if h5Fallback != nil && len(h5Fallback.Tracks) > 0 {
		totalConflict := metadata != nil && metadata.TotalKnown && metadata.List != nil && metadata.List.TotalCount > 0 && h5Fallback.TotalCount > 0 && metadata.List.TotalCount != h5Fallback.TotalCount
		if metadata != nil && metadata.List != nil {
			h5Fallback = preferPlaylist(h5Fallback, metadata.List)
		}
		h5Fallback.Warnings = append(h5Fallback.Warnings, "酷我分享页不可读，备用接口可能未包含全部歌曲")
		if totalConflict {
			h5Fallback.Warnings = append(h5Fallback.Warnings, "酷我接口报告的歌单总数不一致，已按较大值保留")
		}
		if h5FallbackErr != nil {
			h5Fallback.Warnings = append(h5Fallback.Warnings, "酷我备用接口分页中断，结果可能不完整")
		}
		return h5Fallback, nil
	}

	if metadata != nil && metadata.List != nil && len(metadata.List.Tracks) > 0 {
		metadata.List.Warnings = append(metadata.List.Warnings, "酷我官方分享页不可读；接口只返回了当前可见的歌曲")
		if !metadata.TotalKnown || len(metadata.List.Tracks) < metadata.List.TotalCount {
			metadata.List.Warnings = append(metadata.List.Warnings, "酷我接口结果可能不完整，请核对歌曲数量")
		}
		return metadata.List, nil
	}

	return nil, fmt.Errorf("酷我音乐未返回可读取的歌单: %w", errors.Join(pageErr, metadataErr))
}

func (s *Service) kuwoSigningCookie(ctx context.Context) (kuwoSigningCookie, error) {
	_, cookies, _, err := s.HTTP.DoWithCookies(ctx, http.MethodGet, kuwoHomeURL, nil, nil)
	if err != nil {
		return kuwoSigningCookie{}, err
	}
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, "Hm_Iuvt_") && validKuwoCookieValue(cookie.Value) {
			return kuwoSigningCookie{Name: cookie.Name, Value: cookie.Value}, nil
		}
	}
	return kuwoSigningCookie{}, fmt.Errorf("酷我首页未返回有效匿名会话")
}

func (s *Service) kuwoSignedGet(ctx context.Context, address, referer string, session kuwoSigningCookie) ([]byte, error) {
	nonce, err := cryptorand.Int(cryptorand.Reader, big.NewInt(90000000))
	if err != nil {
		return nil, fmt.Errorf("生成酷我请求签名失败: %w", err)
	}
	nonceValue := nonce.Int64() + 10000000
	reqID, err := kuwoRequestID()
	if err != nil {
		return nil, fmt.Errorf("生成酷我请求 ID 失败: %w", err)
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("reqId", reqID)
	parsed.RawQuery = query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, parsed.String(), nil, map[string]string{
		"Accept":     "application/json",
		"Cookie":     session.Name + "=" + session.Value,
		"Referer":    referer,
		"Secret":     kuwoBuildSecret(session.Value, nonceValue),
		"User-Agent": "Mozilla/5.0",
	})
	return body, err
}

func validKuwoCookieValue(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	return allASCIIAlphaNumeric(value)
}

func allASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func allASCIIAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')) {
			return false
		}
	}
	return true
}

func kuwoBuildSecret(cookieValue string, nonce int64) string {
	state := int64(59910100)
	encoded := make([]byte, len(cookieValue))
	for i := range cookieValue {
		state = (state*9253 + 23) % 2147483647
		encoded[i] = cookieValue[i] ^ byte(state*255/2147483647)
	}
	return hex.EncodeToString(encoded) + fmt.Sprintf("%08x", nonce)
}

func kuwoRequestID() (string, error) {
	var raw [16]byte
	if _, err := cryptorand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]), nil
}

func (s *Service) kuwoSharePage(ctx context.Context, id string) (*models.SongList, bool, error) {
	query := url.Values{}
	query.Set("pid", id)
	address := "https://www.kuwo.cn/newh5/playlist/content?" + query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
		"Referer": "https://www.kuwo.cn/",
	})
	if err != nil {
		return nil, false, err
	}
	list, hasMore, err := parseKuwoSharePage(strings.NewReader(string(body)))
	if err != nil || !hasMore {
		return list, hasMore, err
	}
	stillMore, pageErr := s.kuwoSharePages(ctx, id, list)
	if ctx.Err() != nil {
		return nil, false, ctx.Err()
	}
	if pageErr != nil {
		list.Warnings = append(list.Warnings, "酷我后续歌曲加载失败，结果可能不完整")
		return list, true, nil
	}
	return list, stillMore, nil
}

func parseKuwoSharePage(body io.Reader) (*models.SongList, bool, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, false, err
	}

	name := strings.TrimSpace(doc.Find(".titleCenter").First().Text())
	if name == "" {
		name = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	}
	if name == "" {
		name = strings.TrimSpace(doc.Find("title").Text())
		name = strings.TrimSuffix(name, "_酷我音乐")
	}
	if name == "" {
		return nil, false, fmt.Errorf("酷我分享页缺少歌单名称")
	}

	list := &models.SongList{Name: name, Tracks: []models.Track{}, Warnings: []string{}}
	list.Tracks = parseKuwoShareRows(doc.Find("#messDivId li.singBox"))
	if len(list.Tracks) == 0 {
		return nil, false, fmt.Errorf("酷我分享页未返回歌曲行")
	}
	list.TotalCount = len(list.Tracks)
	more := doc.Find("#morebtnId")
	hasMore := kuwoMoreControlVisible(more)
	return list, hasMore, nil
}

func parseKuwoShareRows(rows *goquery.Selection) []models.Track {
	tracks := make([]models.Track, 0, rows.Length())
	rows.Each(func(_ int, row *goquery.Selection) {
		title := strings.TrimSpace(row.Find(".singTexUp2").First().Text())
		if title == "" {
			return
		}
		combined := strings.TrimSpace(row.Find(".singName").First().Text())
		artist := strings.TrimSuffix(combined, "-"+title)
		if artist == combined {
			artist = strings.TrimSuffix(combined, " - "+title)
		}
		artist = strings.TrimSpace(artist)
		artists := []string{}
		if artist != "" {
			artists = append(artists, artist)
		}
		id, _ := row.Attr("mid")
		tracks = append(tracks, models.Track{ID: strings.TrimSpace(id), Title: title, Artists: artists})
	})
	return tracks
}

func (s *Service) kuwoSharePages(ctx context.Context, id string, list *models.SongList) (bool, error) {
	if len(list.Tracks) >= s.MaxSongs {
		return true, nil
	}
	seenTracks := make(map[string]bool, len(list.Tracks))
	for _, track := range list.Tracks {
		if track.ID != "" && track.ID != "0" {
			seenTracks[track.ID] = true
		}
	}
	seenPages := map[string]bool{}
	maxPages := (s.MaxSongs - len(list.Tracks) + kuwoSharePageSize - 1) / kuwoSharePageSize
	for pageNo := 1; pageNo <= maxPages; pageNo++ {
		if err := ctx.Err(); err != nil {
			return true, err
		}
		query := url.Values{}
		query.Set("id", id)
		query.Set("pn", strconv.Itoa(pageNo))
		query.Set("rn", strconv.Itoa(kuwoSharePageSize))
		address := kuwoSharePageAPIURL + "?" + query.Encode()
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
			"Accept":  "*/*",
			"Referer": "https://www.kuwo.cn/newh5/playlist/content?pid=" + id,
		})
		if err != nil {
			return true, fmt.Errorf("酷我分享页第 %d 页请求失败: %w", pageNo, err)
		}
		if err := ctx.Err(); err != nil {
			return true, err
		}
		if strings.TrimSpace(string(body)) == "" {
			return false, nil
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			return true, fmt.Errorf("酷我分享页第 %d 页响应无效: %w", pageNo, err)
		}
		tracks := parseKuwoShareRows(doc.Find("li.singBox"))
		if len(tracks) == 0 {
			return true, fmt.Errorf("酷我分享页第 %d 页未返回歌曲行", pageNo)
		}
		fingerprint, _ := json.Marshal(tracks)
		if seenPages[string(fingerprint)] {
			return true, fmt.Errorf("酷我分享页重复返回第 %d 页", pageNo)
		}
		seenPages[string(fingerprint)] = true
		before := len(list.Tracks)
		for index, track := range tracks {
			if len(list.Tracks) >= s.MaxSongs {
				return true, nil
			}
			appendDistinctTrack(list, track, seenTracks)
			if len(list.Tracks) >= s.MaxSongs && index < len(tracks)-1 {
				return true, nil
			}
		}
		if len(list.Tracks) == before {
			return true, fmt.Errorf("酷我分享页第 %d 页未返回新歌曲", pageNo)
		}
		if len(tracks) < kuwoSharePageSize {
			return false, nil
		}
		if len(list.Tracks) >= s.MaxSongs {
			return true, nil
		}
	}
	return true, nil
}

func kuwoMoreControlVisible(control *goquery.Selection) bool {
	if control.Length() == 0 || strings.TrimSpace(control.AttrOr("onclick", "")) == "" {
		return false
	}
	if _, hidden := control.Attr("hidden"); hidden || strings.EqualFold(strings.TrimSpace(control.AttrOr("aria-hidden", "")), "true") {
		return false
	}
	for _, declaration := range strings.Split(control.AttrOr("style", ""), ";") {
		property, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		property = strings.ToLower(strings.TrimSpace(property))
		value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "!important"))
		if (property == "display" && value == "none") || (property == "visibility" && (value == "hidden" || value == "collapse")) {
			return false
		}
	}
	return true
}

func (s *Service) kuwoMetadata(ctx context.Context, id string) (*kuwoMetadata, error) {
	query := url.Values{}
	query.Set("encode", "utf-8")
	query.Set("identity", "kuwo")
	query.Set("keyset", "pl2012")
	query.Set("op", "getlistinfo")
	query.Set("pid", id)
	query.Set("pn", "0")
	query.Set("rn", strconv.Itoa(kuwoMetadataPageSize))
	query.Set("ttime", "2006012011")
	address := "https://nplserver.kuwo.cn/pl.svc?" + query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
		"Referer": "https://www.kuwo.cn/playlist_detail/" + id,
	})
	if err != nil {
		return nil, err
	}

	var page kuwoPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("酷我歌单接口响应格式无效: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(page.Result), "ok") {
		return nil, fmt.Errorf("酷我歌单不存在、未公开或暂不可用")
	}
	if strings.TrimSpace(page.Title) == "" {
		return nil, fmt.Errorf("酷我歌单接口缺少名称")
	}

	list := &models.SongList{Name: strings.TrimSpace(page.Title), Tracks: []models.Track{}, Warnings: []string{}}
	total, totalKnown, err := kuwoTotal(page.Total)
	if err != nil {
		return nil, err
	}
	if totalKnown && (total > 0 || len(page.Musiclist) == 0) {
		list.TotalCount = total
	} else {
		totalKnown = false
	}
	seen := map[string]bool{}
	for _, song := range page.Musiclist {
		appendDistinctTrack(list, song.track(), seen)
	}
	if totalKnown && list.TotalCount < len(list.Tracks) {
		list.Warnings = append(list.Warnings, "酷我接口报告的歌曲总数小于实际返回数，完整性无法确认")
	}
	list.TotalCount = max(list.TotalCount, len(list.Tracks))
	return &kuwoMetadata{List: list, TotalKnown: totalKnown}, nil
}

func (s *Service) kuwoH5Pages(ctx context.Context, id string) (*models.SongList, error) {
	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	seenTracks := map[string]bool{}
	seenPages := map[string]bool{}
	totalKnown := false
	referenceTotal := 0
	totalConflictWarned := false
	maxPages := (s.MaxSongs + kuwoH5PageSize - 1) / kuwoH5PageSize
	for pageNo := 1; pageNo <= maxPages; pageNo++ {
		count := min(kuwoH5PageSize, s.MaxSongs-len(list.Tracks))
		if totalKnown && count > list.TotalCount-len(list.Tracks) {
			count = list.TotalCount - len(list.Tracks)
		}
		if count <= 0 {
			break
		}
		query := url.Values{}
		query.Set("pid", id)
		query.Set("pn", strconv.Itoa(pageNo))
		query.Set("rn", strconv.Itoa(count))
		address := "https://m.kuwo.cn/newh5app/wapi/api/www/playlist/playListInfo?" + query.Encode()
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
			"Accept":  "application/json",
			"Referer": "https://m.kuwo.cn/newh5app/playlist_detail/" + id,
		})
		if err != nil {
			return list, fmt.Errorf("酷我备用接口第 %d 页请求失败: %w", pageNo, err)
		}
		var page kuwoH5Page
		if err := json.Unmarshal(body, &page); err != nil {
			return list, fmt.Errorf("酷我备用接口响应格式无效: %w", err)
		}
		code, codeKnown, codeErr := kuwoTotal(page.Code)
		if codeErr != nil || (codeKnown && code != 200) {
			return list, fmt.Errorf("酷我备用接口请求失败 (code=%s): %s", strings.TrimSpace(string(page.Code)), strings.TrimSpace(page.Msg))
		}
		if page.Data == nil {
			return list, fmt.Errorf("酷我备用接口未返回歌单信息")
		}
		data := page.Data
		if list.Name == "" {
			list.Name = strings.TrimSpace(data.Name)
		}
		pageTotal, pageTotalKnown, err := kuwoTotal(data.Total)
		if err != nil {
			return list, err
		}
		if !totalConflictWarned && totalKnown && pageTotalKnown && pageTotal != referenceTotal {
			list.Warnings = append(list.Warnings, "酷我备用接口报告的歌曲总数发生变化")
			totalConflictWarned = true
		}
		if pageTotalKnown {
			if !totalKnown {
				referenceTotal = pageTotal
			}
			totalKnown = true
			list.TotalCount = max(list.TotalCount, pageTotal)
		}
		if len(data.MusicList) == 0 {
			if totalKnown && len(list.Tracks) < list.TotalCount {
				return list, fmt.Errorf("酷我备用接口提前结束分页，平台显示 %d 首，当前获取 %d 首", list.TotalCount, len(list.Tracks))
			}
			break
		}
		fingerprint, _ := json.Marshal(data.MusicList)
		if seenPages[string(fingerprint)] {
			return list, fmt.Errorf("酷我备用接口返回重复分页")
		}
		seenPages[string(fingerprint)] = true
		before := len(list.Tracks)
		for _, song := range data.MusicList {
			appendDistinctTrack(list, song.track(), seenTracks)
			if len(list.Tracks) >= s.MaxSongs {
				break
			}
		}
		if len(list.Tracks) == before {
			return list, fmt.Errorf("酷我备用接口分页没有返回新的有效歌曲")
		}
		if totalKnown && len(list.Tracks) >= list.TotalCount {
			break
		}
		if !totalKnown && len(data.MusicList) < count {
			break
		}
	}
	if len(list.Tracks) == 0 {
		return nil, fmt.Errorf("酷我备用接口未返回可读取的歌曲")
	}
	if list.Name == "" {
		list.Name = "酷我歌单"
	}
	if !totalKnown {
		list.TotalCount = max(list.TotalCount, len(list.Tracks))
		list.Warnings = append(list.Warnings, "酷我备用接口未返回可确认的歌单总数")
	}
	if list.TotalCount < len(list.Tracks) {
		list.TotalCount = len(list.Tracks)
		list.Warnings = append(list.Warnings, "酷我备用接口总数小于实际返回歌曲数")
	}
	if len(list.Tracks) >= s.MaxSongs && list.TotalCount > len(list.Tracks) {
		list.Warnings = append(list.Warnings, fmt.Sprintf("达到本次导出上限 %d 首", s.MaxSongs))
	}
	return list, nil
}

func kuwoTotal(raw json.RawMessage) (int, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, false, nil
	}
	var total int
	if err := json.Unmarshal(raw, &total); err == nil {
		if total < 0 {
			return 0, false, fmt.Errorf("酷我歌单总数无效")
		}
		return total, true, nil
	}
	var countText string
	if err := json.Unmarshal(raw, &countText); err != nil {
		return 0, false, fmt.Errorf("酷我歌单总数格式无效: %w", err)
	}
	total, err := strconv.Atoi(strings.TrimSpace(countText))
	if err != nil || total < 0 {
		return 0, false, fmt.Errorf("酷我歌单总数无效")
	}
	return total, true, nil
}
