package handler

import (
	"GoMusic/logic"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"net/http"
	"regexp"

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
	requestCount       = 1
)

func MusicHandler(c *gin.Context) {

	link := c.PostForm("url")

	log.Infof("第 %v 次歌单请求：%v", requestCount, link)
	requestCount++

	switch {
	// 1、网易云
	case netEasyRegx.MatchString(link):
		songList, err := logic.NetEasyDiscover(link)
		if err != nil {
			log.Errorf("fail to get neteasy discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
			return
		}
		c.JSON(200, &models.Result{
			Code: 1,
			Msg:  SUCCESS,
			Data: songList,
		})
		return
	// 2、QQ 音乐
	case qqMusicRegx.MatchString(link):
		songList, err := logic.QQMusicDiscover(link)
		if err != nil {
			log.Errorf("fail to get qqmusic discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
		} else {
			c.JSON(200, &models.Result{Code: 1, Msg: SUCCESS, Data: songList})
		}

		return
	// 3、汽水音乐
	case qishuiMusicRegx.MatchString(link):
		songList, err := logic.QiShuiMusicDiscover(link)
		if err != nil {
			log.Errorf("fail to get qqmusic discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
		} else {
			c.JSON(200, &models.Result{Code: 1, Msg: SUCCESS, Data: songList})
		}
	//4、都不是
	default:
		c.JSON(http.StatusBadRequest, nil)
	}
}
