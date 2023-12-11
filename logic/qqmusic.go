package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"GoMusic/common/models"
	"GoMusic/common/utils"
	"GoMusic/httputil"
	"GoMusic/initialize/log"
)

const (
	qqMusicRedis   = "qq_music:%d"
	qqMusicPattern = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"
	qqMusicV1      = `fcgi-bin`
	qqMusicV2      = `details`
	qqMusicV3      = `playlist`
	qqMusicV4      = `id=[89]\d{9}`
)

var (
	qqMusicV1Regx = regexp.MustCompile(qqMusicV1)
	qqMusicV2Regx = regexp.MustCompile(qqMusicV2)
	qqMusicV3Regx = regexp.MustCompile(qqMusicV3)
	qqMusicV4Regx = regexp.MustCompile(qqMusicV4)
)

// QQMusicDiscover 获取qq音乐歌单
func QQMusicDiscover(link string) (*models.SongList, error) {
	tid, platform, err := getParams(link)
	if err != nil {
		return nil, err
	}

	// 获取请求参数与验证签名
	paramString := models.GetQQMusicReqString(tid, platform)
	sign := utils.Encrypt(paramString)

	// 构建并发送请求
	link = fmt.Sprintf(qqMusicPattern, sign, time.Now().UnixMilli())
	resp, err := httputil.Post(link, strings.NewReader(paramString))
	if err != nil {
		log.Errorf("fail to get qqmusic: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)

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

// GetNetEasyParam 获取歌单id
func getParams(link string) (tid int, platform string, err error) {
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
		return tid, "android", nil
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
		tidString, platform, err = utils.GetQQMusicParam(link)
		if err != nil {
			log.Errorf("fail to get songs id: %v", err)
			return
		}
		tid, err = strconv.Atoi(tidString)
		if err != nil {
			log.Errorf("fail to convert tid: %v", err)
			return
		}
		return tid, platform, nil
	}
	if qqMusicV3Regx.MatchString(link) {
		index := strings.Index(link, "playlist")
		if index < 0 || index+19 > len(link) {
			log.Errorf("fail to get tid: %v", err)
			return
		}
		tid, err = strconv.Atoi(link[index+9 : index+19])
		return tid, "h5", nil
	}
	return 0, "", errors.New("invalid link")
}
