package logic

import (
	"GoMusic/misc/httputil"
	"GoMusic/misc/models"
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
)

var (
	qishuiMusicV1Regx, _ = regexp.Compile(qishuiMusicV1)
	qishuiMusicV2Regx, _ = regexp.Compile(qishuiMusicV2)
)

// 歌曲信息列表#root > div > div > div > div > div:nth-child(2) > div > div > div > div > div 下的子元素nth-child
// 歌曲名称 div:nth-child(2) > div:nth-child(1) > p
// 歌曲作者 div:nth-child(2) > div:nth-child(2) > p

// QiShuiMusicDiscover 解析歌单
func QiShuiMusicDiscover(link string) (*models.SongList, error) {

	params, err := getQiShuiParams(link)
	if err != nil {
		return nil, err
	}
	resp, err := httputil.Get(link, strings.NewReader(params))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	songList, err := ParseSongList(resp.Body)
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
func ParseSongList(body io.Reader) (*models.SongList, error) {
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
			songName := fmt.Sprintf("%s-%s", title, artist)
			songList.Songs = append(songList.Songs, songName)
			songList.SongsCount++
		},
	)
	return &songList, nil
}
