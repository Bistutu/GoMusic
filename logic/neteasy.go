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
	netEasyV1       = `.163.` // 正常链接
	netEasyV2       = "163cn" // 短链接
	netEasyRedis    = "net:%v"
	netEasyUrlV6    = "https://music.163.com/api/v6/playlist/detail"
	netEasyUrlV3    = "https://music.163.com/api/v3/song/detail"
	bracketsPattern = `\s\(.*\)|\s【.*】` // 去除（空格字符）括号及其中内容
)

var (
	netEasyV1Regex = regexp.MustCompile(netEasyV1)
	netEasyV2Regex = regexp.MustCompile(netEasyV2)
	bracketsRegex  = regexp.MustCompile(bracketsPattern)
)

func NetEasyDiscover(link string) (string, error) {
	var err error
	// 如果是短链接，则转换为长链接
	if netEasyV2Regex.MatchString(link) {
		link, err = httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("fail to convert short to long: %v", err)
			return "", err
		}
	}

	// 获取歌单 id
	id, err := utils.GetSongsId(link)
	if err != nil {
		log.Errorf("fail to parse url: %v", err)
		return "", err
	}

	// 第一次发请求，获取歌曲 id 列表
	res, err := httputil.Post(netEasyUrlV6, strings.NewReader("id="+id))
	if err != nil {
		log.Errorf("fail to result: %v", err)
		return "", err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	netEasySongId := &models.NetEasySongId{}
	err = json.Unmarshal(body, netEasySongId)
	if err != nil {
		log.Errorf("fail to unmarshal: %v", err)
		return "", err
	}
	// 无权限访问
	if netEasySongId.Code == 401 {
		log.Errorf("无权限访问, link: %v", link)
		return "", errors.New("无权限访问")
	}
	trackIds := netEasySongId.Playlist.TrackIds       // 歌单歌曲 ID 列表
	songsIdString := make([]string, 0, len(trackIds)) // 歌曲 ID To []String
	for _, v := range trackIds {
		songsIdString = append(songsIdString, fmt.Sprintf(netEasyRedis, v.Id))
	}

	// 尝试获取缓存
	cacheResult, err := cache.MGet(songsIdString...)
	if err != nil {
		// 缓存获取失败，不退出
		log.Errorf("fail to get key: %v", err)
	}

	missKey := make([]*models.SongId, 0)
	resultList := make([]string, 0, len(trackIds))
	for k, v := range cacheResult {
		if v != nil {
			resultList = append(resultList, v.(string))
			continue
		}
		missKey = append(missKey, &models.SongId{Id: trackIds[k].Id})
	}
	// 全部命中，直接返回
	if len(missKey) == 0 {
		log.Infof("全部命中缓存（网易云）: %v", link)
		songList := &models.SongList{
			Name:  netEasySongId.Playlist.Name,
			Songs: resultList,
		}
		bytes, _ := json.Marshal(songList)
		return string(bytes), nil
	}

	// TODO 2023.11.11 考虑 missKey > 500 的情况，需要分批请求（errgroup）

	// missKey 不为 0，第二次发请求，获取歌曲详情
	marshal, _ := json.Marshal(missKey)
	result, err := httputil.Post(netEasyUrlV3, strings.NewReader("c="+string(marshal)))
	if err != nil {
		log.Errorf("fail to result: %v", err)
		return "", err
	}
	defer result.Body.Close()

	bytes, _ := io.ReadAll(result.Body)
	songs := &models.Songs{}
	err = json.Unmarshal(bytes, &songs)
	if err != nil {
		log.Errorf("fail to unmarshal: %v", err)
		return "", err
	}

	missKeyCacheMap := make(map[string]interface{})

	builder := strings.Builder{}
	for _, v := range songs.Songs {
		builder.Reset()
		// 去除多余符号
		builder.WriteString(bracketsRegex.ReplaceAllString(v.Name, ""))
		builder.WriteString(" - ")

		authors := make([]string, 0, len(v.Ar))
		for _, v := range v.Ar {
			authors = append(authors, v.Name)
		}
		authorsString := strings.Join(authors, " / ")
		builder.WriteString(authorsString)
		song := builder.String()
		missKeyCacheMap[fmt.Sprintf(netEasyRedis, v.Id)] = song
		resultList = append(resultList, song)
	}
	// 写缓存
	err = cache.MSet(missKeyCacheMap)
	if err != nil {
		// 缓存写入失败，不退出
		log.Errorf("fail to set key: %v", err)
	}

	songList := &models.SongList{
		Name:  netEasySongId.Playlist.Name,
		Songs: resultList,
	}
	bytes, err = json.Marshal(songList)
	return string(bytes), nil
}
