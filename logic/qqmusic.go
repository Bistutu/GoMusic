package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"GoMusic/common/models"
	"GoMusic/common/utils"
	"GoMusic/httputil"
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
	qqMusicV1Regx, _ = regexp.Compile(qqMusicV1)
	qqMusicV2Regx, _ = regexp.Compile(qqMusicV2)
	qqMusicV3Regx, _ = regexp.Compile(qqMusicV3)
)

func QQMusicDiscover(link string) (*models.SongList, error) {
	tid, err := getDissTid(link)
	if err != nil {
		log.Printf("fail to get tid: %v", err)
		return nil, err
	}

	key, err := cache.GetKey(fmt.Sprintf(qqMusicRedis, tid))
	if err != nil {
		log.Printf("fail to get key: %v", err)
	}
	// 1、如果缓存中存在的话
	if key != "" {
		log.Printf("qqmusic 命中缓存：%v", tid)
		songs := &models.SongList{}
		err := json.Unmarshal([]byte(key), &songs)
		if err != nil {
			log.Printf("fail to unmarshal: %v", err)
			return nil, err
		}
		return songs, nil
	}

	// 2、若缓存中不存在，取数据、缓存
	// 获取参数
	param := models.NewQQMusicReq(tid)
	marshal, _ := json.Marshal(param)
	// 获取签名
	data := string(marshal)
	sign, err := utils.GetSign(data)
	if err != nil {
		log.Printf("fail to get sign: %v", err)
		return nil, err
	}
	// 构建并发送请求
	link = fmt.Sprintf(qqMusicPattern, sign, time.Now().UnixMilli())
	payload := strings.NewReader(string(marshal))
	resp, err := httputil.Post(link, payload)
	if err != nil {
		log.Printf("fail to get qqmusic: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)
	m := &models.QQMusicResp{}
	err = json.Unmarshal(bytes, m)
	if err != nil {
		log.Printf("fail to unmarshal qqmusic: %v", err)
		return nil, err
	}
	songsString := make([]string, 0, len(m.Req0.Data.Songlist))
	for _, v := range m.Req0.Data.Songlist {
		builder := strings.Builder{}
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
	// 3、设置缓存
	err = cache.SetKey(fmt.Sprintf(qqMusicRedis, tid), string(bytes))
	if err != nil {
		log.Printf(err.Error())
	}
	return songList, nil
}

func getDissTid(link string) (tid int, err error) {
	// https://c6.y.qq.com/base/fcgi-bin/u?__=4V33zWKDE3tI
	// https://y.qq.com/n/ryqq/playlist/7364061065
	// https://i.y.qq.com/n2/m/share/details/taoge.html?hosteuin=oKE57evqoiEPoz**&id=1596010000&appversion=120801&ADTAG=wxfshare&appshare=iphone_wx
	if qqMusicV1Regx.MatchString(link) {
		link, err = httputil.GetRedirectionURL(link)
		if err != nil {
			log.Printf("fail to get redirection url: %v", err)
			return 0, err
		}
	}
	if qqMusicV2Regx.MatchString(link) {
		tidString, err := GetSongsId(link)
		if err != nil {
			log.Printf("fail to get songs id: %v", err)
			return 0, err
		}
		tid, err = strconv.Atoi(tidString)
		if err != nil {
			log.Printf("fail to convert string to int: %v", err)
			return 0, err
		}
		return tid, nil
	}
	if qqMusicV3Regx.MatchString(link) {
		index := strings.Index(link, "playlist")
		if index < 0 || index+19 > len(link) {
			log.Printf("fail to get tid: %v", err)
			return 0, err
		}
		tid, err = strconv.Atoi(link[index+9 : index+19])
		return tid, nil
	}
	return 0, errors.New("invalid link")
}
