package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"GoMusic/misc/httputil"
	"GoMusic/misc/log"
	"GoMusic/misc/utils"
	"GoMusic/repo/db"

	"GoMusic/misc/models"
	"GoMusic/repo/cache"
)

const (
	netEasyRedis = "net:%v"
	netEasyUrlV6 = "https://music.163.com/api/v6/playlist/detail"
	netEasyUrlV3 = "https://music.163.com/api/v3/song/detail"
	chunkSize    = 500
)

// NetEasyDiscover 获取网易云音乐歌单信息
// link: 歌单链接
// detailed: 是否使用详细歌曲名（原始歌曲名，不去除括号等内容）
func NetEasyDiscover(link string, detailed bool) (*models.SongList, error) {
	// 1. 获取歌单基本信息
	songIdsResp, err := getSongsInfo(link)
	if err != nil {
		return nil, fmt.Errorf("获取歌单信息失败: %w", err)
	}

	playlistName := songIdsResp.Playlist.Name     // 歌单名
	trackIds := songIdsResp.Playlist.TrackIds      // 歌曲ID列表
	tracksCount := songIdsResp.Playlist.TrackCount // 歌曲总数

	// 如果歌单为空，直接返回
	if len(trackIds) == 0 {
		return &models.SongList{
			Name:       playlistName,
			Songs:      []string{},
			SongsCount: 0,
		}, nil
	}

	// 详细模式下，直接从API获取所有歌曲信息，不走缓存和数据库
	if detailed {
		log.Infof("详细模式：直接从网易云API获取歌曲信息: %v", link)
		// 收集所有歌曲ID
		allSongIds := make([]uint, len(trackIds))
		for i, track := range trackIds {
			allSongIds[i] = track.Id
		}
		
		// 存储歌曲信息的结果集
		resultMap := sync.Map{}
		
		// 直接从API获取所有歌曲信息
		// 注意：在详细模式下，我们不会将获取的数据写入缓存和数据库
		allSongIdsSlice := make([]*models.SongId, len(allSongIds))
		for i, id := range allSongIds {
			allSongIdsSlice[i] = &models.SongId{Id: id}
		}
		
		// 分块处理，避免请求过大
		missSize := len(allSongIdsSlice)
		chunkCount := (missSize + chunkSize - 1) / chunkSize
		chunks := make([][]*models.SongId, chunkCount)

		for i := 0; i < missSize; i += chunkSize {
			end := i + chunkSize
			if end > missSize {
				end = missSize
			}
			chunks[i/chunkSize] = allSongIdsSlice[i:end]
		}

		// 并发请求处理
		var eg errgroup.Group

		for _, chunk := range chunks {
			chunk := chunk // 创建副本避免闭包问题
			eg.Go(func() error {
				return processChunkDetailed(chunk, &resultMap)
			})
		}

		// 等待所有请求完成
		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("获取歌曲详情失败: %w", err)
		}
		
		// 返回最终结果
		return createSongList(playlistName, trackIds, resultMap, tracksCount), nil
	}

	// 非详细模式下，走正常的缓存和数据库流程
	// 2. 构建缓存键
	songCacheKeys := make([]string, len(trackIds))
	for i, track := range trackIds {
		songCacheKeys[i] = fmt.Sprintf(netEasyRedis, track.Id)
	}

	// 3. 存储歌曲信息的结果集
	resultMap := sync.Map{}

	// 4. 尝试从缓存获取歌曲信息
	cacheResults, _ := cache.MGet(songCacheKeys...)
	
	// 5. 收集缓存未命中的歌曲ID
	missCacheKeys := make([]uint, 0, len(trackIds))
	for i, result := range cacheResults {
		if result != nil {
			resultMap.Store(trackIds[i].Id, result.(string))
		} else {
			missCacheKeys = append(missCacheKeys, trackIds[i].Id)
		}
	}

	// 缓存全部命中，直接返回结果
	if len(missCacheKeys) == 0 {
		log.Infof("网易云歌单缓存全部命中: %v", link)
		return createSongList(playlistName, trackIds, resultMap, tracksCount), nil
	}

	// 6. 从数据库查询缓存未命中的歌曲
	dbResults, _ := db.BatchGetSongById(missCacheKeys)
	
	// 7. 收集数据库未命中的歌曲ID
	missDBKeys := make([]uint, 0, len(missCacheKeys))
	for _, id := range missCacheKeys {
		if val, ok := dbResults[id]; ok {
			resultMap.Store(id, val)
		} else {
			missDBKeys = append(missDBKeys, id)
		}
	}

	// 数据库全部命中，更新缓存并返回结果
	if len(missDBKeys) == 0 {
		log.Infof("网易云歌单数据库全部命中: %v", link)
		// 更新缓存
		missKeyCacheMap := sync.Map{}
		for k, v := range dbResults {
			missKeyCacheMap.Store(fmt.Sprintf(netEasyRedis, k), v)
		}
		_ = cache.MSet(missKeyCacheMap)

		return createSongList(playlistName, trackIds, resultMap, tracksCount), nil
	}

	// 8. 从网易云API获取未命中的歌曲信息
	missKeyCacheMap, err := batchGetSongs(missDBKeys, resultMap, detailed)
	if err != nil {
		return nil, fmt.Errorf("获取歌曲详情失败: %w", err)
	}

	// 9. 将新获取的歌曲信息写入数据库
	missDbData := make([]*models.NetEasySong, 0, len(missDBKeys))
	for _, id := range missDBKeys {
		if value, ok := missKeyCacheMap.Load(fmt.Sprintf(netEasyRedis, id)); ok {
			missDbData = append(missDbData, &models.NetEasySong{Id: id, Name: value.(string)})
		}
	}
	
	if len(missDbData) > 0 {
		if err := db.BatchInsertSong(missDbData); err != nil {
			log.Warnf("写入数据库失败: %v", err)
		}
	}

	// 10. 更新缓存
	if err := cache.MSet(missKeyCacheMap); err != nil {
		log.Warnf("更新缓存失败: %v", err)
	}

	// 11. 返回最终结果
	return createSongList(playlistName, trackIds, resultMap, tracksCount), nil
}

// createSongList 创建歌单结果
func createSongList(name string, trackIds []*models.TrackId, resultMap sync.Map, count int) *models.SongList {
	return &models.SongList{
		Name:       name,
		Songs:      utils.SyncMapToSortedSlice(trackIds, resultMap),
		SongsCount: count,
	}
}

// getSongsInfo 获取歌单基本信息
func getSongsInfo(link string) (*models.NetEasySongId, error) {
	songListId, err := utils.GetNetEasyParam(link)
	if err != nil {
		return nil, fmt.Errorf("解析歌单链接失败: %w", err)
	}
	
	resp, err := httputil.Post(netEasyUrlV6, strings.NewReader("id="+songListId))
	if err != nil {
		return nil, fmt.Errorf("请求网易云API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %w", err)
	}
	
	songIdsResp := &models.NetEasySongId{}
	if err = json.Unmarshal(body, songIdsResp); err != nil {
		return nil, fmt.Errorf("解析响应内容失败: %w", err)
	}
	
	if songIdsResp.Code == 401 {
		return nil, errors.New("无权限访问该歌单")
	}
	
	return songIdsResp, nil
}

// batchGetSongs 批量获取歌曲详情
func batchGetSongs(missKeys []uint, resultMap sync.Map, detailed bool) (sync.Map, error) {
	if len(missKeys) == 0 {
		return sync.Map{}, nil
	}
	
	// 1. 构建请求参数
	missSongIds := make([]*models.SongId, len(missKeys))
	for i, id := range missKeys {
		missSongIds[i] = &models.SongId{Id: id}
	}

	// 2. 分块处理，避免请求过大
	missSize := len(missSongIds)
	chunkCount := (missSize + chunkSize - 1) / chunkSize
	chunks := make([][]*models.SongId, chunkCount)

	for i := 0; i < missSize; i += chunkSize {
		end := i + chunkSize
		if end > missSize {
			end = missSize
		}
		chunks[i/chunkSize] = missSongIds[i:end]
	}

	// 3. 并发请求处理
	var eg errgroup.Group
	missKeyCacheMap := sync.Map{}

	for _, chunk := range chunks {
		chunk := chunk // 创建副本避免闭包问题
		eg.Go(func() error {
			return processChunk(chunk, &missKeyCacheMap, &resultMap, detailed)
		})
	}

	// 4. 等待所有请求完成
	if err := eg.Wait(); err != nil {
		return sync.Map{}, err
	}

	return missKeyCacheMap, nil
}

// processChunk 处理一个分块的歌曲ID
func processChunk(chunk []*models.SongId, missKeyCacheMap *sync.Map, resultMap *sync.Map, detailed bool) error {
	// 1. 序列化请求参数
	marshal, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("序列化请求参数失败: %w", err)
	}
	
	// 2. 发送请求
	resp, err := httputil.Post(netEasyUrlV3, strings.NewReader("c="+string(marshal)))
	if err != nil {
		return fmt.Errorf("请求歌曲详情失败: %w", err)
	}
	defer resp.Body.Close()

	// 3. 读取响应内容
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应内容失败: %w", err)
	}
	
	// 4. 解析响应内容
	songs := &models.Songs{}
	if err = json.Unmarshal(bytes, songs); err != nil {
		return fmt.Errorf("解析响应内容失败: %w", err)
	}

	// 5. 处理歌曲信息
	for _, song := range songs.Songs {
		// 根据detailed参数决定是否使用原始歌曲名
		var songName string
		if detailed {
			songName = song.Name // 使用原始歌曲名
		} else {
			songName = utils.StandardSongName(song.Name) // 使用标准化的歌曲名
		}

		// 构建作者信息
		authors := make([]string, len(song.Ar))
		for i, ar := range song.Ar {
			authors[i] = ar.Name
		}
		
		// 格式化歌曲信息
		songInfo := fmt.Sprintf("%s - %s", songName, strings.Join(authors, " / "))

		// 存储结果
		cacheKey := fmt.Sprintf(netEasyRedis, song.Id)
		missKeyCacheMap.Store(cacheKey, songInfo)
		resultMap.Store(song.Id, songInfo)
	}
	
	return nil
}

// processChunkDetailed 处理一个分块的歌曲ID（详细模式）
func processChunkDetailed(chunk []*models.SongId, resultMap *sync.Map) error {
	// 1. 序列化请求参数
	marshal, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("序列化请求参数失败: %w", err)
	}
	
	// 2. 发送请求
	resp, err := httputil.Post(netEasyUrlV3, strings.NewReader("c="+string(marshal)))
	if err != nil {
		return fmt.Errorf("请求歌曲详情失败: %w", err)
	}
	defer resp.Body.Close()

	// 3. 读取响应内容
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应内容失败: %w", err)
	}
	
	// 4. 解析响应内容
	songs := &models.Songs{}
	if err = json.Unmarshal(bytes, songs); err != nil {
		return fmt.Errorf("解析响应内容失败: %w", err)
	}

	// 5. 处理歌曲信息
	for _, song := range songs.Songs {
		// 使用原始歌曲名
		songName := song.Name

		// 构建作者信息
		authors := make([]string, len(song.Ar))
		for i, ar := range song.Ar {
			authors[i] = ar.Name
		}
		
		// 格式化歌曲信息
		songInfo := fmt.Sprintf("%s - %s", songName, strings.Join(authors, " / "))

		// 存储结果 - 注意这里直接使用song.Id作为key，而不是缓存键
		resultMap.Store(song.Id, songInfo)
	}
	
	return nil
}
