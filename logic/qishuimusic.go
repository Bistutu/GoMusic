package logic

import (
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
	"GoMusic/misc/utils"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// 歌曲链接正则
const (
	qishuiMusicV1 = `https?://[a-zA-Z0-9./?=&_-]+`
	qishuiMusicV2 = `playlist_id=`
	qishuiMusicV3 = `https?://qishui\.douyin\.com/s/[a-zA-Z0-9]+/?` // 匹配汽水音乐分享链接
)

var (
	qishuiMusicV1Regx, _ = regexp.Compile(qishuiMusicV1)
	qishuiMusicV2Regx, _ = regexp.Compile(qishuiMusicV2)
	qishuiMusicV3Regx, _ = regexp.Compile(qishuiMusicV3) // 专门匹配汽水音乐链接
)

// 歌曲信息列表#root > div > div > div > div > div:nth-child(2) > div > div > div > div > div 下的子元素nth-child
// 歌曲名称 div:nth-child(2) > div:nth-child(1) > p
// 歌曲作者 div:nth-child(2) > div:nth-child(2) > p

// QiShuiMusicDiscover 解析歌单
// link: 歌单链接
// detailed: 是否使用详细歌曲名（原始歌曲名，不去除括号等内容）
func QiShuiMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	// 从文本中提取汽水音乐链接
	extractedLink := qishuiMusicV3Regx.FindString(link)
	if extractedLink != "" {
		link = extractedLink
	}
	
	params, err := getQiShuiParams(link)
	if err != nil {
		return nil, err
	}
	resp, err := httputil.Get(link, strings.NewReader(params))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	songList, err := ParseSongList(resp.Body, detailed)
	if err != nil {
		return nil, err
	}
	return songList, nil
}

// getQiShuiParams 获取参数,但是都爬网页了好像没有必要
func getQiShuiParams(link string) (string, error) {
	extractLink := qishuiMusicV1Regx.FindString(link)
	if !qishuiMusicV2Regx.MatchString(extractLink) {
		var err error
		extractLink, err = httputil.GetRedirectLocation(extractLink)
		if err != nil {
			return "", err
		}
	}
	fmt.Println(extractLink)
	parsedURL, err := url.Parse(extractLink)
	if err != nil {
		return "", err
	}
	params := parsedURL.Query()
	return params.Encode(), nil
}

// ParseSongList 解析网页
// detailed: 是否使用详细歌曲名（原始歌曲名，不去除括号等内容）
func ParseSongList(body io.Reader, detailed bool) (*models.SongList, error) {
	docDetail, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}
	songListName := docDetail.Find("#root > div > div > div > div > div:nth-child(1) > div:nth-child(3) > h1 > p").Text()
	songListAuthor := docDetail.Find("#root > div > div > div > div > div:nth-child(1) > div:nth-child(3) > div > div > div:nth-child(2) > p").Text()
	songList := models.SongList{
		Name:       fmt.Sprintf("%s-%s", songListName, songListAuthor),
		SongsCount: 0,
	}
	docDetail.Find("#root > div > div > div > div > div:nth-child(2) > div > div > div > div > div").Each(
		func(i int, s *goquery.Selection) {
			title := s.Find("div:nth-child(2) > div:nth-child(1) > p").Text()
			artist := s.Find("div:nth-child(2) > div:nth-child(2) > p").Text()
			
			// 根据detailed参数决定是否使用原始歌曲名
			var songName string
			if detailed {
				songName = title // 使用原始歌曲名
			} else {
				songName = utils.StandardSongName(title) // 使用标准化的歌曲名
			}
			
			// 按照网易云音乐的格式化方式: "歌曲名 - 歌手"
			formattedSong := fmt.Sprintf("%s - %s", songName, artist)
			songList.Songs = append(songList.Songs, formattedSong)
			songList.SongsCount++
		},
	)
	return &songList, nil
}
