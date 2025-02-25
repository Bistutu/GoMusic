package handler

import (
	"net/http"
	"regexp"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"GoMusic/logic"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
)

const (
	// 正则表达式模式
	netEasyPattern = `(163cn)|(\.163\.)`
	qqMusicPattern = `.qq.`

	// 响应消息
	successMsg = "success"
)

var (
	netEasyRegx, _ = regexp.Compile(netEasyPattern)
	qqMusicRegx, _ = regexp.Compile(qqMusicPattern)
	counter        atomic.Int64 // request counter
)

// MusicHandler handler for music request
func MusicHandler(c *gin.Context) {
	link := c.PostForm("url")
	currentCount := counter.Add(1)

	log.Infof("第 %v 次歌单请求：%v", currentCount, link)

	// router to different music service
	switch {
	case netEasyRegx.MatchString(link):
		handleNetEasyMusic(c, link)
	case qqMusicRegx.MatchString(link):
		handleQQMusic(c, link)
	default:
		log.Warnf("不支持的音乐链接格式: %s", link)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: "不支持的音乐链接格式", Data: nil})
	}
}

// handle net easy music
func handleNetEasyMusic(c *gin.Context, link string) {
	songList, err := logic.NetEasyDiscover(link)
	if err != nil {
		log.Errorf("获取网易云音乐歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: successMsg, Data: songList})
}

// 处理QQ音乐链接
func handleQQMusic(c *gin.Context, link string) {
	songList, err := logic.QQMusicDiscover(link)
	if err != nil {
		log.Errorf("获取QQ音乐歌单失败: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: successMsg, Data: songList})
}
