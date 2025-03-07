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
	qqMusicAPIURL = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"

	// 错误响应长度标识
	qqMusicErrorResponseLength = 108
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
func fetchPlaylistData(tid int) ([]byte, error) {
	// 支持的平台列表
	platforms := []string{"-1", "android", "iphone", "h5", "wxfshare", "iphone_wx", "windows"}

	var lastErr error
	var resp *http.Response

	// 尝试不同平台参数
	for _, platform := range platforms {
		// 1. 构建请求参数
		paramString := models.GetQQMusicReqString(tid, platform)
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
