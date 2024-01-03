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

const (
	qqMusicRedis   = "qq_music:%d"
	qqMusicPattern = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"
	qqMusicV1      = `fcgi-bin`
	qqMusicV2      = `details`
	qqMusicV3      = `playlist`
	qqMusicV4      = `id=[89]\d{9}`
	qqMusicV5      = `.*playlist/7\d{9}$`
)

var (
	qqMusicV1Regx = regexp.MustCompile(qqMusicV1)
	qqMusicV2Regx = regexp.MustCompile(qqMusicV2)
	qqMusicV3Regx = regexp.MustCompile(qqMusicV3)
	qqMusicV4Regx = regexp.MustCompile(qqMusicV4)
	qqMusicV5Regx = regexp.MustCompile(qqMusicV5)
	platforms     = []string{"-1", "android", "iphone", "h5"}
)

// QQMusicDiscover 获取qq音乐歌单
func QQMusicDiscover(link string) (*models.SongList, error) {
	tid, err := getParams(link)
	// platform 写死为-1
	if err != nil {
		return nil, err
	}

	bytes, err := getQQMusicResponse(tid)
	if err != nil {
		log.Errorf("fail to get qqmusic response: %v", err)
		return nil, err
	}

	qqmusicResponse := &models.QQMusicResp{}
	err = json.Unmarshal(bytes, qqmusicResponse)
	if err != nil {
		log.Errorf("fail to unmarshal qqmusic: %v", err)
		return nil, err
	}
	songsString := make([]string, 0, len(qqmusicResponse.Req0.Data.Songlist))
	builder := strings.Builder{}
	for _, v := range qqmusicResponse.Req0.Data.Songlist {
		builder.Reset()
		// 去除多余符号
		builder.WriteString(utils.StandardSongName(v.Name))
		builder.WriteString(" - ")

		authors := make([]string, 0, len(v.Singer))
		for _, v := range v.Singer {
			authors = append(authors, v.Name)
		}
		authorsString := strings.Join(authors, " / ")
		builder.WriteString(authorsString)
		songsString = append(songsString, builder.String())
	}
	return &models.SongList{
		Name:       qqmusicResponse.Req0.Data.Dirinfo.Title,
		Songs:      songsString,
		SongsCount: qqmusicResponse.Req0.Data.Dirinfo.Songnum,
	}, nil
}

// 适配不同的平台
func getQQMusicResponse(tid int) (bytes []byte, err error) {
	platforms := []string{"-1", "android", "iphone", "h5", "wxfshare", "iphone_wx", "windows"}

	var resp *http.Response

	for _, platform := range platforms {
		paramString := models.GetQQMusicReqString(tid, platform)
		sign := utils.Encrypt(paramString)
		link := fmt.Sprintf(qqMusicPattern, sign, time.Now().UnixMilli())

		resp, err = httputil.Post(link, strings.NewReader(paramString))
		if err != nil {
			log.Errorf("http error: %+v", err)
			continue
		}

		bytes, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		// 108 代表返回了错误的信息，并没有获取到歌曲
		if len(bytes) != 108 { // Check for a valid response
			return bytes, nil
		}
	}
	return nil, fmt.Errorf("failed to get qqmusic after trying all platforms: %v", err)
}

// GetNetEasyParam 获取歌单id
func getParams(link string) (tid int, err error) {
	if qqMusicV5Regx.MatchString(link) {
		index := strings.Index(link, "playlist")
		if index < 0 || index+19 > len(link) {
			log.Errorf("fail to get tid: %v", err)
			return
		}
		tid, err = strconv.Atoi(link[index+9 : index+19])
		return tid, nil
	}

	if qqMusicV4Regx.MatchString(link) {
		index := strings.Index(link, "id=")
		if index < 0 || index+3 > len(link) {
			log.Errorf("fail to get tid: %v", err)
			return
		}
		tid, err = strconv.Atoi(link[index+3 : index+13])
		if err != nil {
			log.Errorf("fail to convert tid: %v", err)
			return
		}
		return tid, nil
	}
	if qqMusicV1Regx.MatchString(link) {
		link, err = httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("fail to get redirection url: %v", err)
			return
		}
	}
	if qqMusicV2Regx.MatchString(link) {
		var tidString string
		tidString, err = utils.GetQQMusicParam(link)
		if err != nil {
			log.Errorf("fail to get songs id: %v", err)
			return
		}
		tid, err = strconv.Atoi(tidString)
		if err != nil {
			log.Errorf("fail to convert tid: %v", err)
			return
		}
		return tid, nil
	}
	if qqMusicV3Regx.MatchString(link) {
		index := strings.Index(link, "playlist")
		if index < 0 || index+19 > len(link) {
			log.Errorf("fail to get tid: %v", err)
			return
		}
		tid, err = strconv.Atoi(link[index+9 : index+19])
		return tid, nil
	}
	return 0, errors.New("无效的歌单链接")
}
