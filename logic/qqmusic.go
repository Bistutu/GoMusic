package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"GoMusic/misc/httputil"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"GoMusic/misc/utils"
)

// QQ音乐相关常量
const (
	// API相关
	qqMusicRedis  = "qq_music:%d"
	qqMusicAPIURL = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"

	// 错误响应长度标识
	qqMusicErrorResponseLength = 108

	// 分页相关
	maxSongsPerPage = 1000  // 每页最大歌曲数
	maxTotalSongs   = 10000 // 最大支持的歌曲总数
)

// 链接类型正则表达式
var (
	// 短链接，需要重定向
	shortLinkRegex = regexp.MustCompile(`fcgi-bin`)

	// 详情页链接，包含details关键词
	detailsLinkRegex = regexp.MustCompile(`details`)

	// 包含id=数字的链接
	idParamLinkRegex = regexp.MustCompile(`id=\d+`)

	// 包含playlist/数字的链接
	playlistLinkRegex = regexp.MustCompile(`.*playlist/\d+$`)
)

// QQMusicDiscover 获取QQ音乐歌单信息
// link: 歌单链接
// detailed: 是否使用详细歌曲名（原始歌曲名，不去除括号等内容）
func QQMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	// 1. 从链接中提取歌单ID
	tid, err := extractPlaylistID(link)
	if err != nil || tid == 0 {
		return nil, errors.New("无效的歌单链接")
	}

	// 2. 获取歌单数据
	responseData, err := fetchPlaylistData(tid)
	if err != nil {
		log.Errorf("获取QQ音乐歌单数据失败: %v", err)
		return nil, fmt.Errorf("获取歌单数据失败: %w", err)
	}

	// 3. 解析响应数据
	qqMusicResponse := &models.QQMusicResp{}
	if err = json.Unmarshal(responseData, qqMusicResponse); err != nil {
		log.Errorf("解析QQ音乐响应数据失败: %v", err)
		return nil, fmt.Errorf("解析歌单数据失败: %w", err)
	}

	// 4. 构建歌曲列表
	songList := buildSongList(qqMusicResponse, detailed)

	return songList, nil
}

// fetchPlaylistData 获取QQ音乐歌单数据
// 尝试多个平台参数，直到获取有效响应
// 支持分页获取大型歌单的所有歌曲
func fetchPlaylistData(tid int) ([]byte, error) {
	// 1. 先获取歌单基本信息，了解总歌曲数
	basicInfo, err := fetchPlaylistBasicInfo(tid)
	if err != nil {
		return nil, fmt.Errorf("获取歌单基本信息失败: %w", err)
	}

	// 解析基本信息
	basicResp := &models.QQMusicResp{}
	if err = json.Unmarshal(basicInfo, basicResp); err != nil {
		return nil, fmt.Errorf("解析歌单基本信息失败: %w", err)
	}

	// 获取歌曲总数
	totalSongs := basicResp.Req0.Data.Dirinfo.Songnum
	if totalSongs <= maxSongsPerPage {
		// 如果歌曲数量不超过单页上限，直接返回基本信息
		return basicInfo, nil
	}

	// 2. 分页获取所有歌曲
	log.Infof("歌单包含%d首歌曲，需要分页获取", totalSongs)

	// 限制最大获取数量，防止请求过多
	if totalSongs > maxTotalSongs {
		log.Warnf("歌单歌曲数量(%d)超过最大支持数量(%d)，将只获取前%d首",
			totalSongs, maxTotalSongs, maxTotalSongs)
		totalSongs = maxTotalSongs
	}

	// 计算需要的页数
	pageCount := (totalSongs + maxSongsPerPage - 1) / maxSongsPerPage

	// 创建一个新的响应对象，用于合并所有页的数据
	mergedResp := models.QQMusicResp{
		Code: basicResp.Code,
		Req0: struct {
			Code int `json:"code"`
			Data struct {
				Dirinfo struct {
					Title   string `json:"title"`
					Songnum int    `json:"songnum"`
				} `json:"dirinfo"`
				Songlist []struct {
					Name   string `json:"name"`
					Singer []struct {
						Name string `json:"name"`
					} `json:"singer"`
				} `json:"songlist"`
			} `json:"data"`
		}{
			Code: basicResp.Req0.Code,
			Data: struct {
				Dirinfo struct {
					Title   string `json:"title"`
					Songnum int    `json:"songnum"`
				} `json:"dirinfo"`
				Songlist []struct {
					Name   string `json:"name"`
					Singer []struct {
						Name string `json:"name"`
					} `json:"singer"`
				} `json:"songlist"`
			}{
				Dirinfo: basicResp.Req0.Data.Dirinfo,
				Songlist: make([]struct {
					Name   string `json:"name"`
					Singer []struct {
						Name string `json:"name"`
					} `json:"singer"`
				}, 0, totalSongs),
			},
		},
	}

	// 添加第一页的歌曲
	mergedResp.Req0.Data.Songlist = append(mergedResp.Req0.Data.Songlist, basicResp.Req0.Data.Songlist...)

	// 获取剩余页的数据
	for page := 1; page < pageCount; page++ {
		songBegin := page * maxSongsPerPage
		songNum := maxSongsPerPage
		if songBegin+songNum > totalSongs {
			songNum = totalSongs - songBegin
		}

		log.Infof("获取第%d页歌曲，起始位置: %d，数量: %d", page+1, songBegin, songNum)

		// 获取当前页数据
		pageData, err := fetchPlaylistPage(tid, songBegin, songNum)
		if err != nil {
			log.Errorf("获取第%d页歌曲失败: %v", page+1, err)
			continue
		}

		// 解析当前页数据
		pageResp := &models.QQMusicResp{}
		if err = json.Unmarshal(pageData, pageResp); err != nil {
			log.Errorf("解析第%d页歌曲数据失败: %v", page+1, err)
			continue
		}

		// 添加当前页歌曲到合并的响应中
		mergedResp.Req0.Data.Songlist = append(mergedResp.Req0.Data.Songlist, pageResp.Req0.Data.Songlist...)
	}

	// 更新合并后的歌曲总数
	mergedResp.Req0.Data.Dirinfo.Songnum = len(mergedResp.Req0.Data.Songlist)

	// 将合并后的响应转换为JSON
	mergedData, err := json.Marshal(mergedResp)
	if err != nil {
		return nil, fmt.Errorf("合并歌单数据失败: %w", err)
	}

	log.Infof("成功获取全部%d首歌曲", len(mergedResp.Req0.Data.Songlist))
	return mergedData, nil
}

// fetchPlaylistBasicInfo 获取歌单基本信息（第一页数据）
func fetchPlaylistBasicInfo(tid int) ([]byte, error) {
	return fetchPlaylistPage(tid, 0, maxSongsPerPage)
}

// fetchPlaylistPage 获取歌单指定页的数据
func fetchPlaylistPage(tid int, songBegin, songNum int) ([]byte, error) {
	// 支持的平台列表
	platforms := []string{"-1", "android", "iphone", "h5", "wxfshare", "iphone_wx", "windows"}

	var lastErr error
	var resp *http.Response

	// 尝试不同平台参数
	for _, platform := range platforms {
		// 1. 构建请求参数
		paramString := models.GetQQMusicReqStringWithPagination(tid, platform, songBegin, songNum)
		sign := utils.Encrypt(paramString)
		requestURL := fmt.Sprintf(qqMusicAPIURL, sign, time.Now().UnixMilli())

		// 2. 发送请求
		resp, lastErr = httputil.Post(requestURL, strings.NewReader(paramString))
		if lastErr != nil {
			log.Errorf("HTTP请求失败(平台:%s): %v", platform, lastErr)
			continue
		}

		// 3. 读取响应
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 4. 检查响应是否有效
		// 108字节长度表示返回了错误信息，需要尝试其他平台
		if len(data) != qqMusicErrorResponseLength {
			return data, nil
		}
	}

	return nil, fmt.Errorf("尝试所有平台参数均失败: %w", lastErr)
}

// extractPlaylistID 从QQ音乐链接中提取歌单ID
func extractPlaylistID(link string) (int, error) {
	// 1. 处理playlist/数字格式的链接
	if playlistLinkRegex.MatchString(link) {
		return extractNumberAfterKeyword(link, "playlist/")
	}

	// 2. 处理id=数字格式的链接
	if idParamLinkRegex.MatchString(link) {
		return extractNumberAfterKeyword(link, "id=")
	}

	// 3. 处理需要重定向的短链接
	if shortLinkRegex.MatchString(link) {
		redirectedLink, err := httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("获取重定向链接失败: %v", err)
			return 0, fmt.Errorf("处理短链接失败: %w", err)
		}
		// 递归处理重定向后的链接
		return extractPlaylistID(redirectedLink)
	}

	// 4. 处理details页面链接
	if detailsLinkRegex.MatchString(link) {
		tidString, err := utils.GetQQMusicParam(link)
		if err != nil {
			log.Errorf("从details链接提取ID失败: %v", err)
			return 0, fmt.Errorf("提取歌单ID失败: %w", err)
		}

		tid, err := strconv.Atoi(tidString)
		if err != nil {
			log.Errorf("歌单ID转换为数字失败: %v", err)
			return 0, fmt.Errorf("歌单ID格式错误: %w", err)
		}

		return tid, nil
	}

	return 0, errors.New("无效的歌单链接格式")
}

// buildSongList 根据QQ音乐响应数据构建歌曲列表
func buildSongList(response *models.QQMusicResp, detailed bool) *models.SongList {
	songsCount := response.Req0.Data.Dirinfo.Songnum
	songList := response.Req0.Data.Songlist

	songs := make([]string, 0, len(songList))
	builder := strings.Builder{}

	for _, song := range songList {
		builder.Reset()

		// 根据detailed参数决定是否使用原始歌曲名
		if detailed {
			builder.WriteString(song.Name) // 使用原始歌曲名
		} else {
			builder.WriteString(utils.StandardSongName(song.Name)) // 去除多余符号
		}

		builder.WriteString(" - ")

		// 处理歌手信息
		singers := make([]string, 0, len(song.Singer))
		for _, singer := range song.Singer {
			singers = append(singers, singer.Name)
		}
		builder.WriteString(strings.Join(singers, " / "))

		songs = append(songs, builder.String())
	}

	return &models.SongList{
		Name:       response.Req0.Data.Dirinfo.Title,
		Songs:      songs,
		SongsCount: songsCount,
	}
}

// extractNumberAfterKeyword 从字符串中提取关键词后面的数字
func extractNumberAfterKeyword(s, keyword string) (int, error) {
	index := strings.Index(s, keyword)
	if index < 0 {
		return 0, fmt.Errorf("未找到关键词: %s", keyword)
	}

	// 提取关键词后面的所有数字
	startIndex := index + len(keyword)
	endIndex := len(s)

	// 找到数字结束的位置
	for i := startIndex; i < endIndex; i++ {
		if s[i] < '0' || s[i] > '9' {
			endIndex = i
			break
		}
	}

	// 提取并转换数字
	numStr := s[startIndex:endIndex]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("数字转换失败: %w", err)
	}

	return num, nil
}
