package logic

import (
	"GoMusic/internal/musicurl"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const miguSongsPerPage = 50

type miguMetadataResponse struct {
	Code json.RawMessage `json:"code"`
	Info string          `json:"info"`
	Data *struct {
		Title    string          `json:"title"`
		MusicNum json.RawMessage `json:"musicNum"`
	} `json:"data"`
}

type miguSongsResponse struct {
	Code json.RawMessage `json:"code"`
	Info string          `json:"info"`
	Data *miguSongPage   `json:"data"`
}

type miguSongPage struct {
	TotalCount json.RawMessage `json:"totalCount"`
	SongList   []miguSong      `json:"songList"`
	total      int
	totalKnown bool
}

// miguAlternateSongsResponse is the flat shape returned by the official
// queryMusicListSongs.do endpoint, kept as a second song-list channel.
type miguAlternateSongsResponse struct {
	Code       json.RawMessage `json:"code"`
	Info       string          `json:"info"`
	TotalCount json.RawMessage `json:"totalCount"`
	List       []struct {
		ContentID json.RawMessage `json:"contentId"`
		SongID    json.RawMessage `json:"songId"`
		Name      string          `json:"songName"`
		Singer    string          `json:"singer"`
	} `json:"list"`
}

type miguSinger struct {
	Name string `json:"name"`
}

type miguSong struct {
	ContentID json.RawMessage `json:"contentId"`
	SongID    json.RawMessage `json:"songId"`
	Name      string          `json:"songName"`
	Singers   []miguSinger    `json:"singerList"`
}

type miguPlaylistMetadata struct {
	Name       string
	TotalCount int
	TotalKnown bool
}

func (s *Service) migu(ctx context.Context, link musicurl.Link) (*models.SongList, error) {
	link, err := s.resolve(ctx, link)
	if err != nil {
		return nil, err
	}

	metadata, metadataErr := s.miguMetadata(ctx, link.ID)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	list := &models.SongList{Tracks: []models.Track{}, Warnings: []string{}}
	totalKnown := false
	referenceTotal := 0
	totalConflict := false
	if metadata != nil {
		list.Name = metadata.Name
		list.TotalCount = metadata.TotalCount
		totalKnown = metadata.TotalKnown
		if totalKnown {
			referenceTotal = metadata.TotalCount
		}
	}

	seen := map[string]bool{}
	var pageErr error
	maxPages := (s.MaxSongs + miguSongsPerPage - 1) / miguSongsPerPage
	for pageNo := 1; pageNo <= maxPages; pageNo++ {
		if len(list.Tracks) >= s.MaxSongs || (list.TotalCount > 0 && len(list.Tracks) >= list.TotalCount) {
			break
		}

		page, err := s.miguSongsPage(ctx, link.ID, pageNo, list.TotalCount)
		if err != nil {
			pageErr = fmt.Errorf("咪咕歌单第 %d 页获取失败: %w", pageNo, err)
			break
		}
		pageTotal := page.total
		if page.totalKnown {
			if totalKnown && pageTotal != referenceTotal {
				totalConflict = true
			} else if !totalKnown {
				referenceTotal = pageTotal
			}
			list.TotalCount = max(list.TotalCount, pageTotal)
			totalKnown = true
		}

		before := len(list.Tracks)
		for _, song := range page.SongList {
			track := song.track()
			appendDistinctTrack(list, track, seen)
			if len(list.Tracks) >= s.MaxSongs {
				break
			}
		}
		if len(page.SongList) > 0 && len(list.Tracks) == before {
			pageErr = fmt.Errorf("咪咕音乐分页没有返回新的有效歌曲")
			break
		}
		if len(page.SongList) == 0 {
			if totalKnown && len(list.Tracks) < list.TotalCount {
				pageErr = fmt.Errorf("咪咕音乐提前结束分页，平台显示 %d 首，当前获取 %d 首", list.TotalCount, len(list.Tracks))
			}
			break
		}
		if len(page.SongList) < miguSongsPerPage && (!totalKnown || len(list.Tracks) >= list.TotalCount) {
			break
		}
	}

	if len(list.Tracks) == 0 {
		if list.Name != "" && totalKnown && list.TotalCount == 0 && pageErr == nil {
			return list, nil
		}
		return nil, fmt.Errorf("咪咕音乐未返回可读取的歌单: %w", errors.Join(metadataErr, pageErr))
	}
	if list.Name == "" {
		list.Name = "咪咕歌单"
		list.Warnings = append(list.Warnings, "咪咕歌单名称未能读取，已使用默认名称")
	}
	if !totalKnown {
		list.TotalCount = max(list.TotalCount, len(list.Tracks))
		list.Warnings = append(list.Warnings, "咪咕音乐未返回可确认的歌单总数，无法确认结果是否完整")
	}
	if list.TotalCount < len(list.Tracks) {
		totalConflict = true
		list.TotalCount = len(list.Tracks)
	}
	if totalConflict {
		list.Warnings = append(list.Warnings, "咪咕接口报告的歌曲总数与实际返回数不一致，已按较大值处理")
	}
	if pageErr != nil {
		list.Warnings = append(list.Warnings, pageErr.Error())
	}
	if list.TotalCount > len(list.Tracks) && pageErr == nil {
		list.Warnings = append(list.Warnings, fmt.Sprintf("咪咕音乐显示 %d 首，当前获取 %d 首，结果可能不完整", list.TotalCount, len(list.Tracks)))
	}
	return list, nil
}

func (s *Service) miguMetadata(ctx context.Context, id string) (*miguPlaylistMetadata, error) {
	query := url.Values{}
	query.Set("playlistId", id)
	address := "https://app.c.nf.migu.cn/resource/playlist/v2.0?" + query.Encode()
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, address, nil, map[string]string{
		"Referer": "https://music.migu.cn/",
	})
	if err != nil {
		return nil, err
	}

	var response miguMetadataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("咪咕歌单元数据格式无效: %w", err)
	}
	if code := miguCode(response.Code); code != "000000" {
		return nil, fmt.Errorf("咪咕歌单元数据请求失败 (code=%s): %s", code, strings.TrimSpace(response.Info))
	}
	if response.Data == nil {
		return nil, fmt.Errorf("咪咕音乐未返回歌单元数据")
	}
	name := strings.TrimSpace(response.Data.Title)
	if name == "" {
		return nil, fmt.Errorf("咪咕歌单元数据缺少名称")
	}
	total, known, err := miguCount(response.Data.MusicNum)
	if err != nil {
		return nil, err
	}
	return &miguPlaylistMetadata{Name: name, TotalCount: total, TotalKnown: known}, nil
}

func (s *Service) miguSongsPage(ctx context.Context, id string, pageNo, knownTotal int) (*miguSongPage, error) {
	query := url.Values{
		"pageNo": {strconv.Itoa(pageNo)}, "pageSize": {strconv.Itoa(miguSongsPerPage)}, "playlistId": {id},
	}
	const base = "https://app.c.nf.migu.cn/MIGUM3.0"
	headers := map[string]string{"Referer": "https://music.migu.cn/"}
	body, _, err := s.HTTP.Do(ctx, http.MethodGet, base+"/resource/playlist/song/v2.0?"+query.Encode(), nil, headers)
	var primary miguSongsResponse
	if err == nil {
		err = json.Unmarshal(body, &primary)
		if err == nil && miguCode(primary.Code) != "000000" {
			err = fmt.Errorf("咪咕歌曲列表请求失败 (code=%s): %s", miguCode(primary.Code), strings.TrimSpace(primary.Info))
			primary.Data = nil
		}
		if err == nil {
			err = primary.Data.validate(pageNo, knownTotal)
		}
		if err == nil {
			return primary.Data, nil
		}
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	// Both official routes accept the same 1-based page number and page size.
	query.Del("playlistId")
	query.Set("musicListId", id)
	body, _, alternateErr := s.HTTP.Do(ctx, http.MethodGet, base+"/v1.0/user/queryMusicListSongs.do?"+query.Encode(), nil, headers)
	if alternateErr == nil {
		var alternate miguAlternateSongsResponse
		alternateErr = json.Unmarshal(body, &alternate)
		if alternateErr == nil && miguCode(alternate.Code) != "000000" {
			alternateErr = fmt.Errorf("咪咕备用歌曲列表请求失败 (code=%s): %s", miguCode(alternate.Code), strings.TrimSpace(alternate.Info))
		}
		if alternateErr == nil {
			candidate := &miguSongPage{TotalCount: alternate.TotalCount}
			if alternate.List != nil {
				candidate.SongList = make([]miguSong, 0, len(alternate.List))
			}
			for _, item := range alternate.List {
				song := miguSong{ContentID: item.ContentID, SongID: item.SongID, Name: item.Name}
				if name := strings.TrimSpace(item.Singer); name != "" {
					song.Singers = []miguSinger{{Name: name}}
				}
				candidate.SongList = append(candidate.SongList, song)
			}
			if primary.Data != nil && primary.Data.totalKnown {
				knownTotal = max(knownTotal, primary.Data.total)
			}
			alternateErr = candidate.validate(pageNo, knownTotal)
			if alternateErr == nil {
				if primary.Data != nil && primary.Data.totalKnown && primary.Data.total > candidate.total {
					candidate.total, candidate.totalKnown = primary.Data.total, true
				}
				return candidate, nil
			}
		}
	}
	return nil, errors.Join(err, fmt.Errorf("咪咕备用歌曲列表不可用: %w", alternateErr))
}

func (page *miguSongPage) validate(pageNo, knownTotal int) error {
	if page == nil {
		return fmt.Errorf("咪咕音乐未返回歌单歌曲列表")
	}
	total, known, err := miguCount(page.TotalCount)
	if err != nil {
		return err
	}
	page.total, page.totalKnown = total, known
	for _, song := range page.SongList {
		if strings.TrimSpace(song.Name) != "" {
			return nil
		}
	}
	if page.SongList != nil && len(page.SongList) == 0 && max(knownTotal, total) <= (pageNo-1)*miguSongsPerPage {
		return nil
	}
	return fmt.Errorf("咪咕音乐这一页未返回可读取的歌曲")
}

func miguCount(raw json.RawMessage) (int, bool, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, false, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil {
		if count < 0 {
			return 0, false, fmt.Errorf("咪咕歌单总数无效")
		}
		return count, true, nil
	}
	var countText string
	if err := json.Unmarshal(raw, &countText); err != nil {
		return 0, false, fmt.Errorf("咪咕歌单总数格式无效: %w", err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(countText))
	if err != nil || count < 0 {
		return 0, false, fmt.Errorf("咪咕歌单总数无效")
	}
	return count, true, nil
}

func miguCode(raw json.RawMessage) string {
	var code string
	if err := json.Unmarshal(raw, &code); err == nil {
		return strings.TrimSpace(code)
	}
	return strings.Trim(strings.TrimSpace(string(raw)), `"`)
}

func (song miguSong) track() models.Track {
	id := miguString(song.ContentID)
	if id == "" {
		id = miguString(song.SongID)
	}
	track := models.Track{ID: id, Title: strings.TrimSpace(song.Name), Artists: []string{}}
	for _, singer := range song.Singers {
		if name := strings.TrimSpace(singer.Name); name != "" {
			track.Artists = append(track.Artists, name)
		}
	}
	return track
}

func miguString(raw json.RawMessage) string {
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
