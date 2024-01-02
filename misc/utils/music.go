package utils

import (
	"net/url"
	"regexp"
	"sync"

	"GoMusic/misc/httputil"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
)

const (
	bracketsPattern = `（|）`     // 去除特殊符号
	miscPattern     = `\s?【.*】` // 去除特殊符号
	netEasyV2       = "163cn"   // 短链接
	shardModel      = `http[s]?://[^ ]+`
	restfulModel    = `playlist/(\d+)`
)

var (
	bracketsRegex    = regexp.MustCompile(bracketsPattern)
	miscRegex        = regexp.MustCompile(miscPattern)
	netEasyV2Regex   = regexp.MustCompile(netEasyV2)
	shardModelRegex  = regexp.MustCompile(shardModel)
	restfulModeRegex = regexp.MustCompile(restfulModel)
)

func GetQQMusicParam(link string) (string, error) {
	parse, err := url.ParseRequestURI(link)
	if err != nil {
		log.Errorf("fail to parse url: %v", err)
		return "", err
	}
	query, err := url.ParseQuery(parse.RawQuery)
	if err != nil {
		log.Errorf("fail to parse query: %v", err)
		return "", err
	}
	id := query.Get("id")
	return id, nil
}

func GetNetEasyParam(link string) (string, error) {
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
	if err != nil {
		log.Errorf("fail to parse query: %v", err)
		return "", err
	}
	return query.Get("id"), nil
}

func standardUrl(link string) (string, error) {
	// 格式化带中文的分享链接
	link = shardModelRegex.FindString(link)
	// 短链转换
	if netEasyV2Regex.MatchString(link) {
		return httputil.GetRedirectLocation(link)
	}
	// 匹配 restful 链接
	if match := restfulModeRegex.FindStringSubmatch(link); len(match) > 1 {
		link = "https://music.163.com/playlist?id=" + match[1]
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

func SyncMapToSortedSlice(trackIds []*models.TrackId, sm sync.Map) []string {
	strings := make([]string, 0, len(trackIds))
	for _, v := range trackIds {
		value, _ := sm.Load(v.Id)
		if elems, ok := value.(string); ok {
			strings = append(strings, elems)
		}
	}
	return strings
}
