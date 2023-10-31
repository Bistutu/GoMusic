package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"regexp"
	"strings"

	"GoMusic/common/models"
	"GoMusic/httputil"
	"GoMusic/repo/cache"
)

const (
	netEasyLongRegex  = "https://(y\\.)?music.163.com/.*playlist\\?.*id=\\d{9,10}.*"
	netEasyShortRegex = "http://163cn.tv/\\w{6}"
	netEasyRedis      = "net_easy:%s"
)

var (
	netEasyLongPattern, _  = regexp.Compile(netEasyLongRegex)
	netEasyShortPattern, _ = regexp.Compile(netEasyShortRegex)
)

func NetEasyDiscover(link string) (*models.SongList, error) {
	// 判断链接是否为网易云歌单链接并标准化
	link, err := IsNetEasyDiscover(link)
	if err != nil {
		log.Printf("无效的链接格式：%s", link)
		return nil, err
	}
	id, err := GetSongsId(link)
	if err != nil {
		log.Printf("fail to parse url: %v", err)
		return nil, err
	}
	// 检查缓存
	key, err := cache.GetKey(fmt.Sprintf(netEasyRedis, id))
	if err != nil {
		log.Printf("fail to get key: %v", err)
	}
	// 1、如果缓存中存在的话
	if key != "" {
		log.Printf("neteasy 命中缓存：%v", id)
		songs := &models.SongList{}
		err := json.Unmarshal([]byte(key), &songs)
		if err != nil {
			log.Printf("fail to unmarshal: %v", err)
			return nil, err
		}
		return songs, nil
	}

	// 2、若缓存中不存在，取数据、缓存
	res, err := httputil.Post("https://music.163.com/api/v6/playlist/detail", strings.NewReader("id="+id))
	if err != nil {
		log.Printf("fail to post: %v", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("fail to read res body: %v", err)
		return nil, err
	}
	netEasySongId := &models.NetEasySongId{}
	err = json.Unmarshal(body, &netEasySongId)
	if err != nil {
		log.Printf("fail to unmarshal: %v", err)
		return nil, err
	}
	// 无权限访问
	if netEasySongId.Code == 401 {
		cache.SetKey(fmt.Sprintf(netEasyRedis, id), "") // redis 防击穿
		log.Printf("无权限访问, link: %v", link)
		return nil, fmt.Errorf("无权限访问")
	}
	SongsName := netEasySongId.Playlist.Name            // 歌单名称
	trackIds := netEasySongId.Playlist.TrackIds         // 歌单歌曲 ID 列表
	songsId := make([]*models.SongId, 0, len(trackIds)) // 歌曲 ID To []Uint
	for _, v := range trackIds {
		songsId = append(songsId, &models.SongId{Id: v.Id})
	}
	marshal, _ := json.Marshal(songsId)

	reader := strings.NewReader("c=" + string(marshal))
	post, err := httputil.Post("https://music.163.com/api/v3/song/detail", reader)
	if err != nil {
		log.Printf("fail to post: %v", err)
		return nil, err
	}
	defer post.Body.Close()
	bytes, _ := io.ReadAll(post.Body)
	songs := &models.Songs{}
	err = json.Unmarshal(bytes, &songs)
	if err != nil {
		log.Printf("fail to unmarshal: %v", err)
		return nil, err
	}
	songsString := make([]string, 0, len(songs.Songs))
	for _, v := range songs.Songs {
		builder := strings.Builder{}
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
		Name:  SongsName,
		Songs: songsString,
	}
	bytes, err = json.Marshal(songList)
	if err != nil {
		log.Printf("fail to marshal: %v", err)
		return nil, err
	}
	// 3、设置缓存
	err = cache.SetKey(fmt.Sprintf(netEasyRedis, id), string(bytes))
	if err != nil {
		log.Printf(err.Error())
	}
	return songList, nil
}

func GetSongsId(link string) (string, error) {
	parse, err := url.ParseRequestURI(link)
	if err != nil {
		log.Printf("fail to parse url: %v", err)
		return "", err
	}
	query, err := url.ParseQuery(parse.RawQuery)
	return query.Get("id"), nil
}

func IsNetEasyDiscover(link string) (string, error) {

	if netEasyShortPattern.MatchString(link) {
		short, err := short2Long(link)
		if err != nil {
			log.Printf("短链转换失败: %v", link)
			return "", errors.New("短链转换失败！")
		}
		link = short
	}
	if !netEasyLongPattern.MatchString(link) {
		return "", errors.New("无效的网易云歌单链接！")
	}
	// http://163cn.tv/zoIxm3
	// https://music.163.com/#/playlist?app_version=8.10.81&id=8725919816&dlt=0846&creatorId=341246998"
	// https://music.163.com/playlist?id=477577176&userid=341246998
	return link, nil
}

func short2Long(link string) (string, error) {
	redirectionURL, err := httputil.GetRedirectionURL(link)
	if err != nil {
		return "", err
	}
	return redirectionURL, nil
}
