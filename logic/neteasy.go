package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"GoMusic/common/utils"
	"GoMusic/initialize/log"

	"GoMusic/common/models"
	"GoMusic/httputil"
	"GoMusic/repo/cache"
)

const (
	netEasyV1    = `.163.` // 正常链接
	netEasyV2    = "163cn" // 短链接
	netEasyRedis = "net_easy:%s"
	netEasyUrlV6 = "https://music.163.com/api/v6/playlist/detail"
	netEasyUrlV3 = "https://music.163.com/api/v3/song/detail"
)

var (
	netEasyV1Regx = regexp.MustCompile(netEasyV1)
	netEasyV2Regx = regexp.MustCompile(netEasyV2)
)

func NetEasyDiscover(link string) (string, error) {
	var err error
	// 如果是短链接，则转换为长链接
	if netEasyV2Regx.MatchString(link) {
		link, err = httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("fail to convert short to long: %v", err)
			return "", err
		}
	}

	id, err := utils.GetSongsId(link)
	if err != nil {
		log.Errorf("fail to parse url: %v", err)
		return "", err
	}
	// 检查缓存
	redisCache, err := cache.GetKey(fmt.Sprintf(netEasyRedis, id))
	if err != nil {
		log.Errorf("redis connect fail: %v", err)
	}
	// 1、如果缓存中存在的话
	if redisCache != "" {
		log.Infof("neteasy 命中缓存：%v", id)
		return redisCache, nil
	}

	// 2、若缓存中不存在，取数据、缓存
	res, err := httputil.Post(netEasyUrlV6, strings.NewReader("id="+id))
	if err != nil {
		log.Errorf("fail to post: %v", err)
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("fail to read res body: %v", err)
		return "", err
	}
	netEasySongId := &models.NetEasySongId{}
	err = json.Unmarshal(body, netEasySongId)
	if err != nil {
		log.Errorf("fail to unmarshal: %v", err)
		return "", err
	}
	// 无权限访问
	if netEasySongId.Code == 401 {
		cache.SetKey(fmt.Sprintf(netEasyRedis, id), "") // redis 防击穿
		log.Errorf("无权限访问, link: %v", link)
		return "", errors.New("无权限访问")
	}
	trackIds := netEasySongId.Playlist.TrackIds         // 歌单歌曲 ID 列表
	songsId := make([]*models.SongId, 0, len(trackIds)) // 歌曲 ID To []Uint
	for _, v := range trackIds {
		songsId = append(songsId, &models.SongId{Id: v.Id})
	}
	marshal, _ := json.Marshal(songsId)

	post, err := httputil.Post(netEasyUrlV3, strings.NewReader("c="+string(marshal)))
	if err != nil {
		log.Errorf("fail to post: %v", err)
		return "", err
	}
	defer post.Body.Close()

	bytes, _ := io.ReadAll(post.Body)
	songs := &models.Songs{}
	err = json.Unmarshal(bytes, &songs)
	if err != nil {
		log.Errorf("fail to unmarshal: %v", err)
		return "", err
	}
	songsString := make([]string, 0, len(songs.Songs))
	builder := strings.Builder{}
	for _, v := range songs.Songs {
		builder.Reset()
		builder.WriteString(v.Name)
		builder.WriteString(" - ")

		authors := make([]string, 0, len(v.Ar))
		for _, v := range v.Ar {
			authors = append(authors, v.Name)
		}
		authorsString := strings.Join(authors, " / ")
		builder.WriteString(authorsString)
		songsString = append(songsString, builder.String())
	}
	songList := &models.SongList{
		Name:  netEasySongId.Playlist.Name,
		Songs: songsString,
	}
	bytes, err = json.Marshal(songList)
	data := string(bytes)
	if err != nil {
		log.Errorf("fail to marshal: %v", err)
		return "", err
	}
	// 3、设置缓存
	err = cache.SetKey(fmt.Sprintf(netEasyRedis, id), data)
	if err != nil {
		log.Errorf("fail to set key: %v", err)
	}
	return data, nil
}
