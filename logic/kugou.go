package logic

import (
	"bytes"
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

	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"github.com/PuerkitoBio/goquery"
)

const kugouMobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 Version/16.5 Mobile/15E148 Safari/604.1"

// Public protocol signing constant, not a user credential. Sources: docs/REFERENCES.md.
const kugouSigningKey = "NVPh5oo715z5DIWAeQlhMDsWXXQV4hwt"

type kugouSong struct {
	Hash       string `json:"hash"`
	Name       string `json:"name"`
	Songname   string `json:"songname"`
	Filename   string `json:"filename"`
	Singername string `json:"singername"`
	Singers    []struct {
		Name string `json:"name"`
	} `json:"singerinfo"`
}

func (song kugouSong) track() models.Track {
	track := models.Track{ID: song.Hash, Title: song.Songname, Artists: []string{}}
	if track.Title == "" {
		track.Title = song.Name
	}
	if track.Title == "" {
		track.Title = song.Filename
	}
	for _, artist := range song.Singers {
		if artist.Name != "" {
			track.Artists = append(track.Artists, artist.Name)
		}
	}
	if len(track.Artists) == 0 && song.Singername != "" {
		track.Artists = append(track.Artists, song.Singername)
	}
	if len(track.Artists) > 0 {
		for _, separator := range []string{"、", " / ", ", ", "&"} {
			prefix := strings.Join(track.Artists, separator) + " - "
			if strings.HasPrefix(track.Title, prefix) {
				track.Title = strings.TrimPrefix(track.Title, prefix)
				break
			}
		}
	} else if artist, title, ok := strings.Cut(track.Title, " - "); ok {
		track.Title = title
		track.Artists = strings.Split(artist, "、")
	}
	return track
}

func (s *Service) kugou(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	if link.ID == "" {
		final, err := s.HTTP.Resolve(ctx, link.URL.String(), map[string]string{"User-Agent": kugouMobileUA})
		if err != nil {
			return nil, err
		}
		resolved, err := musicurl.Parse(final.String())
		if err != nil || resolved.ID == "" {
			return nil, fmt.Errorf("%w: 酷狗分享链接没有可读取的歌单 ID", ErrInvalidLink)
		}
		link = resolved
	}
	switch link.Kind {
	case "gcid":
		return s.kugouShare(ctx, link.ID)
	case "collection":
		return s.kugouPages(ctx, link.ID, true)
	case "special":
		return s.kugouPages(ctx, link.ID, false)
	default:
		return nil, fmt.Errorf("%w: 请使用酷狗歌单分享链接", ErrInvalidLink)
	}
}

func (s *Service) kugouShare(ctx context.Context, id string) (*models.SongList, error) {
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, "https://m.kugou.com/songlist/"+id+"/", nil, map[string]string{"User-Agent": kugouMobileUA})
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var list *models.SongList
	var collection string
	var special string
	var owner int64
	doc.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := script.Text()
		start := strings.Index(text, "window.$output")
		if start < 0 {
			return true
		}
		text = strings.TrimSpace(text[start+len("window.$output"):])
		if !strings.HasPrefix(text, "=") {
			return true
		}
		var payload struct {
			Collection string `json:"global_collection_id"`
			Info       *struct {
				Listinfo struct {
					Name       string      `json:"name"`
					Count      *int        `json:"count"`
					Collection string      `json:"global_collection_id"`
					SpecialID  json.Number `json:"specialid"`
					Owner      json.Number `json:"list_create_userid"`
				} `json:"listinfo"`
				Songs []kugouSong `json:"songs"`
			} `json:"info"`
		}
		if json.NewDecoder(strings.NewReader(strings.TrimSpace(text[1:]))).Decode(&payload) != nil || payload.Info == nil {
			return true
		}
		info := payload.Info
		list = &models.SongList{Name: info.Listinfo.Name}
		if info.Listinfo.Count != nil && *info.Listinfo.Count >= 0 {
			list.TotalCount = *info.Listinfo.Count
		} else {
			list.Warnings = append(list.Warnings, "酷狗未提供歌单总数，请核对导出结果")
		}
		collection = info.Listinfo.Collection
		if collection == "" {
			collection = payload.Collection
		}
		special = info.Listinfo.SpecialID.String()
		owner, _ = info.Listinfo.Owner.Int64()
		for _, song := range info.Songs {
			if track := song.track(); track.Title != "" {
				list.Tracks = append(list.Tracks, track)
			}
		}
		return false
	})
	if list == nil || (list.Name == "" && len(list.Tracks) == 0) {
		return nil, fmt.Errorf("酷狗未返回可读取的公开歌单")
	}
	if list.TotalCount < len(list.Tracks) && len(list.Warnings) == 0 {
		list.Warnings = append(list.Warnings, "酷狗分享页的歌曲总数与预览不一致，请核对导出结果")
	}
	preview := list
	if list.TotalCount > len(list.Tracks) || len(list.Warnings) > 0 {
		link, err := musicurl.Parse("https://activity.kugou.com/share/index.html?global_specialid=" + url.QueryEscape(collection))
		if err == nil && link.Kind == "collection" && link.ID != "" {
			candidate, err := s.kugouPages(ctx, link.ID, true, list.TotalCount)
			if err == nil {
				name := list.Name
				list = preferPlaylist(list, candidate)
				if name != "" {
					list.Name = name
				}
			}
		}
		if list.TotalCount > len(list.Tracks) || len(list.Warnings) > 0 {
			if id, err := strconv.ParseUint(special, 10, 64); err == nil && id > 0 {
				candidate, err := s.kugouPages(ctx, special, false, list.TotalCount)
				if err == nil {
					name := list.Name
					list = preferPlaylist(list, candidate)
					if name != "" {
						list.Name = name
					}
				}
			}
		}
	}
	if list.TotalCount > len(list.Tracks) || len(list.Warnings) > 0 {
		if derived := kugouCollectionFromShare(id, owner); derived != "" && derived != collection {
			candidate, err := s.kugouPages(ctx, derived, true, list.TotalCount)
			// The share trailer is not an authenticated identity. Require the
			// entire visible prefix to match before accepting decoded metadata.
			if err == nil && kugouMatchesPreview(preview, candidate) {
				name := list.Name
				list = preferPlaylist(list, candidate)
				if name != "" {
					list.Name = name
				}
			}
		}
	}
	if list == preview && (list.TotalCount > len(list.Tracks) || len(list.Warnings) > 0) {
		list.Preview = true
		list.Warnings = append(list.Warnings, "酷狗普通版分享页仅提供部分预览，未获取的歌曲不会计入导出数量；可尝试概念版的歌单分享链接")
	}
	return list, nil
}

// The observed v3 GCID format encodes the owner/list fields in base 35,
// separated by z. Only decode its known zero-tag shape, corroborate the
// owner against official share metadata, then verify the visible songs.
func kugouCollectionFromShare(id string, owner int64) string {
	if owner <= 0 || !strings.HasPrefix(id, "gcid_") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(id, "gcid_"), "z")
	if len(parts) != 4 || parts[0] != "3" || len(parts[3]) != 3 || parts[3][0] != '0' {
		return ""
	}
	var fields [3]uint64
	for i := range fields {
		value, err := strconv.ParseUint(parts[i], 35, 63)
		if err != nil || strconv.FormatUint(value, 35) != parts[i] {
			return ""
		}
		fields[i] = value
	}
	if _, err := strconv.ParseUint(parts[3][1:], 35, 16); err != nil || fields[1] != uint64(owner) || fields[2] == 0 {
		return ""
	}
	return fmt.Sprintf("collection_3_%d_%d_0", fields[1], fields[2])
}

func kugouMatchesPreview(preview, candidate *models.SongList) bool {
	if candidate == nil || len(preview.Tracks) == 0 || len(candidate.Tracks) < len(preview.Tracks) {
		return false
	}
	for i, track := range preview.Tracks {
		if track.ID == "" || !strings.EqualFold(track.ID, candidate.Tracks[i].ID) {
			return false
		}
	}
	return true
}

// kugouSignedParams implements the public H5 parameter signature using Go's standard MD5.
func kugouSignedParams(values url.Values) url.Values {
	// url.Values.Encode sorts keys but escapes values; the signature uses their original values.
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "signature" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var raw strings.Builder
	raw.WriteString(kugouSigningKey)
	for _, key := range keys {
		raw.WriteString(key)
		raw.WriteByte('=')
		raw.WriteString(values.Get(key))
	}
	raw.WriteString(kugouSigningKey)
	digest := md5.Sum([]byte(raw.String()))
	values.Set("signature", hex.EncodeToString(digest[:]))
	return values
}
func kugouCommonParams() url.Values {
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return url.Values{"srcappid": {"2919"}, "clientver": {"20000"}, "clienttime": {now}, "mid": {now}, "uuid": {now}, "dfid": {"-"}}
}
func (s *Service) kugouPages(ctx context.Context, id string, collection bool, knownTotal ...int) (*models.SongList, error) {
	list := &models.SongList{Name: "酷狗歌单"}
	if len(knownTotal) > 0 {
		list.TotalCount = max(0, knownTotal[0])
	}
	headers := map[string]string{"User-Agent": kugouMobileUA, "Referer": "https://activity.kugou.com/"}
	var infoURL string
	if collection {
		values := kugouCommonParams()
		values.Set("global_specialid", id)
		values.Set("specialid", "0")
		values.Set("sign", "h5")
		infoURL = "https://mobiles.kugou.com/v5/special/info_v2?" + kugouSignedParams(values).Encode()
	} else {
		infoURL = "https://mobilecdn.kugou.com/api/v3/special/info?specialid=" + id
	}
	if body, _, err := s.HTTP.Do(ctx, http.MethodGet, infoURL, nil, headers); err == nil {
		var metadata struct {
			Status int `json:"status"`
			Data   struct {
				Name  string `json:"specialname"`
				Count int    `json:"songcount"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &metadata) == nil && metadata.Status == 1 {
			if metadata.Data.Name != "" {
				list.Name = metadata.Data.Name
			}
			list.TotalCount = max(list.TotalCount, metadata.Data.Count)
		}
	}
	seen := map[string]bool{}
	seenTracks := map[string]bool{}
	for page := 1; len(list.Tracks) < s.MaxSongs; page++ {
		values := url.Values{"page": {strconv.Itoa(page)}, "pagesize": {"100"}}
		endpoint := "https://mobilecdn.kugou.com/api/v3/special/song"
		if collection {
			for key, value := range kugouCommonParams() {
				values[key] = value
			}
			values.Set("uid", "0")
			values.Set("appid", "1058")
			values.Set("token", "")
			values.Set("type", "0")
			values.Set("module", "playlist")
			values.Set("global_collection_id", id)
			values = kugouSignedParams(values)
			endpoint = "https://pubsongscdn.kugou.com/v2/get_other_list_file"
		} else {
			values.Set("specialid", id)
			values.Set("version", "9108")
			values.Set("area_code", "1")
		}
		body, _, err := s.HTTP.Do(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil, headers)
		if err != nil {
			if len(list.Tracks) > 0 {
				list.Warnings = append(list.Warnings, "酷狗分页未完成，请稍后重试")
				return list, nil
			}
			return nil, err
		}
		var data struct {
			Status int `json:"status"`
			Data   *struct {
				Count *int        `json:"count"`
				Info  []kugouSong `json:"info"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &data) != nil || data.Status != 1 || data.Data == nil || data.Data.Info == nil {
			if len(list.Tracks) > 0 {
				list.Warnings = append(list.Warnings, "酷狗未返回后续歌曲")
				return list, nil
			}
			return nil, fmt.Errorf("酷狗歌单不存在、未公开或接口暂不可用")
		}
		if data.Data.Count != nil && *data.Data.Count > list.TotalCount {
			list.TotalCount = *data.Data.Count
		}
		if data.Data.Count == nil && page == 1 && list.TotalCount == 0 {
			list.Warnings = append(list.Warnings, "酷狗未提供歌单总数，请核对导出结果")
		}
		if len(data.Data.Info) == 0 {
			break
		}
		fingerprint, _ := json.Marshal(data.Data.Info)
		if seen[string(fingerprint)] {
			list.Warnings = append(list.Warnings, "酷狗返回重复分页，已停止获取")
			break
		}
		seen[string(fingerprint)] = true
		before := len(list.Tracks)
		for _, song := range data.Data.Info {
			appendDistinctTrack(list, song.track(), seenTracks)
		}
		if len(list.Tracks) == before {
			list.Warnings = append(list.Warnings, "酷狗分页没有返回新的有效歌曲，已停止获取")
			break
		}
		if (list.TotalCount > 0 && len(list.Tracks) >= list.TotalCount) || (list.TotalCount == 0 && len(data.Data.Info) < 100) {
			break
		}
	}
	return list, nil
}
