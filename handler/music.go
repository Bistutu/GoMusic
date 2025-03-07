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
	qishuiMusic = `qishui`
	SUCCESS     = "success"
)

var (
	netEasyRegx, _     = regexp.Compile(netEasy)
	qqMusicRegx, _     = regexp.Compile(qqMusic)
	qishuiMusicRegx, _ = regexp.Compile(qishuiMusic)
	counter            atomic.Int64 // request counter
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
		handleQiShuiMusic(c, link, detailed)
	default:
		// 尝试从文本中提取汽水音乐链接
		if qishuiRegx := regexp.MustCompile(`https?://qishui\.douyin\.com/s/[a-zA-Z0-9]+/?`); qishuiRegx.MatchString(link) {
			handleQiShuiMusic(c, link, detailed)
			return
		}

		log.Warnf("不支持的音乐链接格式: %s", link)
		c.JSON(http.StatusBadRequest, &models.Result{Code: models.FailureCode, Msg: "不支持的音乐链接格式", Data: nil})
	}
}

// handle net easy music
func handleNetEasyMusic(c *gin.Context, link string, detailed bool) {
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

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: SUCCESS, Data: songList})
}

// 处理QQ音乐链接
func handleQQMusic(c *gin.Context, link string, detailed bool) {
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

	c.JSON(http.StatusOK, &models.Result{Code: models.SuccessCode, Msg: SUCCESS, Data: songList})
}

// handleQiShuiMusic 处理汽水音乐歌单
func handleQiShuiMusic(c *gin.Context, link string, detailed bool) {
	songList, err := logic.QiShuiMusicDiscover(link, detailed)
	if err != nil {
		log.Errorf("fail to get qishui music discover: %v", err)
		c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
	} else {
		c.JSON(200, &models.Result{Code: 1, Msg: SUCCESS, Data: songList})
	}
}
