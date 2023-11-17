package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"GoMusic/common/utils"
	"GoMusic/initialize/log"
	"GoMusic/repo/db"

	"GoMusic/common/models"
	"GoMusic/httputil"
	"GoMusic/repo/cache"
)

const (
	netEasyRedis = "net:%v"
	netEasyUrlV6 = "https://music.163.com/api/v6/playlist/detail"
	netEasyUrlV3 = "https://music.163.com/api/v3/song/detail"
	chunkSize    = 500
)

// NetEasyDiscover 需转发 2~3 次请求
func NetEasyDiscover(link string) (*models.SongList, error) {
	// 批量获取歌单信息：歌单名、歌曲ids、歌曲总数
	SongIdsResp, err := batchGetSongsId(link)
	if err != nil {
		return nil, err
	}
	SongsListName := SongIdsResp.Playlist.Name     // 歌单名
	trackIds := SongIdsResp.Playlist.TrackIds      // 歌曲列表
	tracksCount := SongIdsResp.Playlist.TrackCount // 歌曲列表

	songCacheKey := make([]string, 0, len(trackIds))
	for _, v := range trackIds {
		songCacheKey = append(songCacheKey, fmt.Sprintf(netEasyRedis, v.Id))
	}

	resultMap := sync.Map{} // 结果

	// 1、尝试获取缓存，失败不退出
	cacheResult, _ := cache.MGet(songCacheKey...)

	missCacheKey := make([]uint, 0)
	for k, v := range cacheResult {
		if v != nil {
			resultMap.Store(trackIds[k].Id, v.(string))
			continue
		}
		missCacheKey = append(missCacheKey, trackIds[k].Id)
	}
	if len(missCacheKey) == 0 { // 缓存全部命中，直接返回
		log.Infof("全部命中缓存（网易云）: %v", link)
		return &models.SongList{
			Name:       SongsListName,
			Songs:      utils.SyncMapToSortedSlice(trackIds, resultMap),
			SongsCount: tracksCount,
		}, nil
	}

	// 2、查询数据库，失败不退出
	dbResultMap, _ := db.BatchGetSongById(missCacheKey)

	missDBKey := make([]uint, 0)

	for _, v := range missCacheKey {
		if val, ok := dbResultMap[v]; ok {
			resultMap.Store(v, val)
			continue
		}
		missDBKey = append(missDBKey, v)
	}
	if len(dbResultMap) == len(missCacheKey) { // 数据库全部命中
		return &models.SongList{
			Name:       SongsListName,
			Songs:      utils.SyncMapToSortedSlice(trackIds, resultMap),
			SongsCount: tracksCount,
		}, nil
	}

	missKeyCacheMap, err := batchGetSongs(missDBKey, resultMap)
	if err != nil {
		return nil, err
	}

	// 写数据库
	missDbData := make([]*models.NetEasySong, 0, len(missDBKey))
	for _, v := range missDBKey {
		value, _ := missKeyCacheMap.Load(fmt.Sprintf(netEasyRedis, v))
		missDbData = append(missDbData, &models.NetEasySong{Id: v, Name: value.(string)})
	}
	_ = db.BatchInsertSong(missDbData)

	// 写缓存
	_ = cache.MSet(missKeyCacheMap)

	return &models.SongList{
		Name:       SongsListName,
		Songs:      utils.SyncMapToSortedSlice(trackIds, resultMap),
		SongsCount: tracksCount,
	}, nil
}

func batchGetSongsId(link string) (*models.NetEasySongId, error) {
	songListId, err := utils.GetSongsId(link)
	if err != nil {
		return nil, err
	}
	resp, err := httputil.Post(netEasyUrlV6, strings.NewReader("id="+songListId))
	if err != nil {
		log.Errorf("fail to result: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	SongIdsResp := &models.NetEasySongId{}
	err = json.Unmarshal(body, SongIdsResp)
	switch {
	case err != nil:
		log.Errorf("fail to unmarshal: %v", err)
		return nil, err
	case SongIdsResp.Code == 401:
		log.Errorf("无权限访问, songList id: %v", songListId)
		return nil, errors.New("抱歉，您无权限访问该歌单")
	}
	return SongIdsResp, nil
}

// 批量从网易云音乐查询歌曲数据
func batchGetSongs(missKey []uint, resultMap sync.Map) (sync.Map, error) {
	missSongIds := make([]*models.SongId, 0, len(missKey))
	for _, v := range missKey {
		missSongIds = append(missSongIds, &models.SongId{Id: v})
	}
	missSize := len(missSongIds)
	// errgroup 并发编程
	errgroup := errgroup.Group{}
	chunks := make([][]*models.SongId, 0, missSize/500+1)
	missKeyCacheMap := sync.Map{}

	for i := 0; i < missSize; i += chunkSize {
		end := i + chunkSize
		if end > missSize {
			end = missSize
		}
		chunks = append(chunks, missSongIds[i:end])
	}
	for _, v := range chunks {
		chunk := v
		errgroup.Go(func() error {
			marshal, _ := json.Marshal(chunk)
			resp, err := httputil.Post(netEasyUrlV3, strings.NewReader("c="+string(marshal)))
			if err != nil {
				log.Errorf("fail to result: %v", err)
				return err
			}
			defer resp.Body.Close()
			bytes, _ := io.ReadAll(resp.Body)
			songs := &models.Songs{}
			err = json.Unmarshal(bytes, &songs)
			if err != nil {
				log.Errorf("fail to unmarshal: %v", err)
				return err
			}

			builder := strings.Builder{}
			for _, v := range songs.Songs {
				builder.Reset()
				// 去除多余符号
				builder.WriteString(utils.StandardSongName(v.Name))
				builder.WriteString(" - ")

				authors := make([]string, 0, len(v.Ar))
				for _, v := range v.Ar {
					authors = append(authors, v.Name)
				}
				authorsString := strings.Join(authors, " / ")
				builder.WriteString(authorsString)
				song := builder.String()
				missKeyCacheMap.Store(fmt.Sprintf(netEasyRedis, v.Id), song)
				resultMap.Store(v.Id, song)
			}
			return nil
		})
	}
	// 等待所有 goroutine 完成
	if err := errgroup.Wait(); err != nil {
		log.Errorf("fail to wait: %v", err)
		return sync.Map{}, err
	}
	return missKeyCacheMap, nil
}
