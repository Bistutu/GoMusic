package utils

import (
	"net/url"
	"regexp"
	"sync"

	"GoMusic/common/models"
	"GoMusic/httputil"
	"GoMusic/initialize/log"
)

const (
	bracketsPattern = `（|）`     // 去除特殊符号
	miscPattern     = `\s?【.*】` // 去除特殊符号
	netEasyV2       = "163cn"   // 短链接
)

var (
	bracketsRegex  = regexp.MustCompile(bracketsPattern)
	miscRegex      = regexp.MustCompile(miscPattern)
	netEasyV2Regex = regexp.MustCompile(netEasyV2)
)

func GetSongsId(link string) (string, error) {
	link, err := standardUrl(link)
	if err != nil {
		log.Errorf("fail to standard url: %v", err)
		return "", err
	}
	parse, err := url.ParseRequestURI(link)
	if err != nil {
		log.Errorf("fail to parse url: %v", err)
		return "", err
	}
	query, err := url.ParseQuery(parse.RawQuery)
	return query.Get("id"), nil
}

func standardUrl(link string) (string, error) {
	if netEasyV2Regex.MatchString(link) {
		return httputil.GetRedirectLocation(link)
	}
	return link, nil
}

// StandardSongName 获取标准化歌名
func StandardSongName(songName string) string {
	return miscRegex.ReplaceAllString(replaceCNBrackets(songName), "")
}

// 将中文括号替换为英文括号
func replaceCNBrackets(s string) string {
	return bracketsRegex.ReplaceAllStringFunc(s, func(m string) string {
		if m == "（" {
			return " (" // 左括号前面追加空格
		}
		return ")"
	})
}

func SyncMapToSortedSlice(trackIds []models.TrackId, sm sync.Map) []string {
	strings := make([]string, 0, len(trackIds))
	for _, v := range trackIds {
		value, _ := sm.Load(v.Id)
		strings = append(strings, value.(string))
	}
	return strings
}
