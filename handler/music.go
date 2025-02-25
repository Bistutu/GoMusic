package handler

import (
	"GoMusic/logic"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

const (
	netEasy     = `(163cn)|(\.163\.)`
	qqMusic     = `.qq.`
	qishuiMusic = `/qishui`
	SUCCESS     = "success"
)

var (
	netEasyRegx, _     = regexp.Compile(netEasy)
	qqMusicRegx, _     = regexp.Compile(qqMusic)
	qishuiMusicRegx, _ = regexp.Compile(qishuiMusic)
	counter        atomic.Int64 // request counter
)

// MusicHandler handler for music request
func MusicHandler(c *gin.Context) {
	link := c.PostForm("url")
	detailed := c.Query("detailed") == "true"
	currentCount := counter.Add(1)

	log.Infof("第 %v 次歌单请求：%v，详细歌曲名：%v", currentCount, link, detailed)

	// router to different music service
	switch {
	case netEasyRegx.MatchString(link):
		handleNetEasyMusic(c, link, detailed)
	case qqMusicRegx.MatchString(link):
		handleQQMusic(c, link, detailed)
	case qishuiMusicRegx.MatchString(link):
		songList, err := logic.QiShuiMusicDiscover(link)
		if err != nil {
			log.Errorf("fail to get qqmusic discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
		} else {
			c.JSON(200, &models.Result{Code: 1, Msg: SUCCESS, Data: songList})
		}
	default:
		log.Warnf("不支持的音乐链接格式: %s", link)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: "不支持的音乐链接格式", Data: nil})
	}
}

// handle net easy music
func handleNetEasyMusic(c *gin.Context, link string, detailed bool) {
	songList, err := logic.NetEasyDiscover(link, detailed)
	if err != nil {
		log.Errorf("获取网易云音乐歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: successMsg, Data: songList})
}

// 处理QQ音乐链接
func handleQQMusic(c *gin.Context, link string, detailed bool) {
	songList, err := logic.QQMusicDiscover(link, detailed)
	if err != nil {
		log.Errorf("获取QQ音乐歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: successMsg, Data: songList})
}
