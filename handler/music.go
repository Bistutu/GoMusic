package handler

import (
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"

	"GoMusic/logic"
	"GoMusic/misc/log"
	"GoMusic/misc/models"

	"github.com/gin-gonic/gin"
)

const (
	netEasy     = `(163cn)|(\.163\.)`
	qqMusic     = `.qq.`
	qishuiMusic = `(qishui)|(douyin)`
	SUCCESS     = "success"
)

var (
	netEasyRegx, _     = regexp.Compile(netEasy)
	qqMusicRegx, _     = regexp.Compile(qqMusic)
	qishuiMusicRegx, _ = regexp.Compile(qishuiMusic)
	counter            atomic.Int64 // request counter
)

// MusicHandler 处理音乐请求的入口函数
func MusicHandler(c *gin.Context) {
	link := c.PostForm("url")
	detailed := c.Query("detailed") == "true"
	format := c.Query("format")
	order := c.Query("order")
	currentCount := counter.Add(1)

	log.Infof("第 %v 次歌单请求：%v，详细歌曲名：%v，歌曲格式：%v，歌曲顺序：%v", currentCount, link, detailed, format, order)

	// 路由到不同的音乐服务处理函数
	switch {
	case netEasyRegx.MatchString(link):
		handleNetEasyMusic(c, link, detailed, format, order)
	case qqMusicRegx.MatchString(link):
		handleQQMusic(c, link, detailed, format, order)
	case qishuiMusicRegx.MatchString(link):
		handleQiShuiMusic(c, link, detailed, format, order)
	default:
		log.Warnf("不支持的音乐链接格式: %s", link)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: "不支持的音乐链接格式", Data: nil})
	}
}

// handleNetEasyMusic 处理网易云音乐歌单
func handleNetEasyMusic(c *gin.Context, link string, detailed bool, format, order string) {
	songList, err := logic.NetEasyDiscover(link, detailed)
	if err != nil {
		if strings.Contains(err.Error(), "无权限访问该歌单") {
			log.Errorf("获取歌单失败，无权限访问: %v", link)
		} else {
			log.Errorf("获取歌单失败: %v", err)
		}
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	// 根据格式选项处理歌曲列表
	formatSongList(songList, format)
	
	// 根据顺序选项处理歌曲列表
	processSongOrder(songList, order)

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: SUCCESS, Data: songList})
}

// handleQQMusic 处理QQ音乐歌单
func handleQQMusic(c *gin.Context, link string, detailed bool, format, order string) {
	if link == "https://i.y.qq.com/v8/playsong.html" {
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: "无效歌单链接，请检查是否正确", Data: nil})
		return
	}

	songList, err := logic.QQMusicDiscover(link, detailed)
	if err != nil {
		log.Errorf("获取歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	// 根据格式选项处理歌曲列表
	formatSongList(songList, format)
	
	// 根据顺序选项处理歌曲列表
	processSongOrder(songList, order)

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: SUCCESS, Data: songList})
}

// handleQiShuiMusic 处理汽水音乐歌单
func handleQiShuiMusic(c *gin.Context, link string, detailed bool, format, order string) {
	songList, err := logic.QiShuiMusicDiscover(link, detailed)
	if err != nil {
		log.Errorf("获取汽水音乐歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	// 根据格式选项处理歌曲列表
	formatSongList(songList, format)
	
	// 根据顺序选项处理歌曲列表
	processSongOrder(songList, order)

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: SUCCESS, Data: songList})
}

// processSongOrder 根据指定的顺序处理歌曲列表
func processSongOrder(songList *models.SongList, order string) {
	if songList == nil || len(songList.Songs) == 0 {
		return
	}

	// 如果是倒序，则反转歌曲列表
	if order == "reverse" {
		songs := songList.Songs
		for i, j := 0, len(songs)-1; i < j; i, j = i+1, j-1 {
			songs[i], songs[j] = songs[j], songs[i]
		}
	}
}

// formatSongList 根据指定的格式处理歌曲列表
func formatSongList(songList *models.SongList, format string) {
	if songList == nil || len(songList.Songs) == 0 {
		return
	}

	// 如果没有指定格式或格式为默认的"歌名-歌手"，则不做处理
	if format == "" || format == "song-singer" {
		return
	}

	formattedSongs := make([]string, 0, len(songList.Songs))

	for _, song := range songList.Songs {
		switch format {
		case "singer-song":
			// 将"歌名 - 歌手"转换为"歌手 - 歌名"
			parts := strings.Split(song, " - ")
			if len(parts) == 2 {
				formattedSongs = append(formattedSongs, parts[1]+" - "+parts[0])
			} else {
				// 如果格式不符合预期，保持原样
				formattedSongs = append(formattedSongs, song)
			}
		case "song":
			// 只保留歌名
			parts := strings.Split(song, " - ")
			if len(parts) > 0 {
				formattedSongs = append(formattedSongs, parts[0])
			} else {
				formattedSongs = append(formattedSongs, song)
			}
		default:
			// 未知格式，保持原样
			formattedSongs = append(formattedSongs, song)
		}
	}

	// 更新歌曲列表
	songList.Songs = formattedSongs
}
