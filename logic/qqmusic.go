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
	"GoMusic/repo/cache"
)

const (
	qqMusicRedis   = "qq_music:%d"
	qqMusicPattern = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"
	qqMusicV1      = `fcgi-bin`
	qqMusicV2      = `details`
	qqMusicV3      = `playlist`
)

var (
	qqMusicV1Regx = regexp.MustCompile(qqMusicV1)
	qqMusicV2Regx = regexp.MustCompile(qqMusicV2)
	qqMusicV3Regx = regexp.MustCompile(qqMusicV3)
)

// QQMusicDiscover 获取qq音乐歌单
func QQMusicDiscover(link string) (string, error) {
	tid, err := getDissTid(link)
	if err != nil {
		log.Errorf("fail to get tid: %v", err)
		return "", err
	}

	redisCache, err := cache.GetKey(fmt.Sprintf(qqMusicRedis, tid))
	if err != nil {
		log.Errorf("redis connect fail: %v", err)
	}
	// 1、如果缓存中存在的话
	if redisCache != "" {
		log.Infof("qqmusic 命中缓存：%v", tid)
		return redisCache, nil
	}

	// 2、若缓存没命中，取数据
	// 获取请求参数与验证签名
	paramString := models.GetQQMusicReqString(tid)
	sign, err := utils.GetSign(paramString)
	if err != nil {
		log.Errorf("fail to get sign: %v", err)
		return "", err
	}
	// 构建并发送请求
	link = fmt.Sprintf(qqMusicPattern, sign, time.Now().UnixMilli())
	resp, err := httputil.Post(link, strings.NewReader(paramString))
	if err != nil {
		log.Errorf("fail to get qqmusic: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)
	m := &models.QQMusicResp{}
	err = json.Unmarshal(bytes, m)
	if err != nil {
		log.Errorf("fail to unmarshal qqmusic: %v", err)
		return "", err
	}
	songsString := make([]string, 0, len(m.Req0.Data.Songlist))
	builder := strings.Builder{}
	for _, v := range m.Req0.Data.Songlist {
		builder.Reset()
		builder.WriteString(v.Name)
		builder.WriteString(" - ")

		authors := make([]string, 0, len(v.Singer))
		for _, v := range v.Singer {
			authors = append(authors, v.Name)
		}
		authorsString := strings.Join(authors, " / ")
		builder.WriteString(authorsString)
		songsString = append(songsString, builder.String())
	}
	songList := &models.SongList{
		Name:  m.Req0.Data.Dirinfo.Title,
		Songs: songsString,
	}
	bytes, _ = json.Marshal(songList)
	data := string(bytes)
	// 3、设置缓存
	err = cache.SetKey(fmt.Sprintf(qqMusicRedis, tid), data)
	if err != nil {
		log.Errorf(err.Error())
	}
	return data, nil
}

// GetSongsId 获取歌单id
func getDissTid(link string) (tid int, err error) {
	if qqMusicV1Regx.MatchString(link) {
		link, err = httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("fail to get redirection url: %v", err)
			return
		}
	}
	if qqMusicV2Regx.MatchString(link) {
		var tidString string
		tidString, err = utils.GetSongsId(link)
		if err != nil {
			log.Errorf("fail to get songs id: %v", err)
			return
		}
		return strconv.Atoi(tidString)
	}
	if qqMusicV3Regx.MatchString(link) {
		index := strings.Index(link, "playlist")
		if index < 0 || index+19 > len(link) {
			log.Errorf("fail to get tid: %v", err)
			return
		}
		return strconv.Atoi(link[index+9 : index+19])
	}
	return 0, errors.New("invalid link")
}
