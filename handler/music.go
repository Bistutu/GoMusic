package handler

import (
	"net/http"
	"regexp"

	"GoMusic/initialize/log"

	"github.com/gin-gonic/gin"

	"GoMusic/common/models"
	"GoMusic/logic"
)

const (
	netEasy = `(163cn)|(.163.)`
	qqMusic = `.qq.`
)

var (
	netEasyRegx, _ = regexp.Compile(netEasy)
	qqMusicRegx, _ = regexp.Compile(qqMusic)
)

func MusicHandler(c *gin.Context) {
	// 获取前端传过来的 url，判断是网易云还是 qq 音乐
	link := c.PostForm("url")
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
			Msg:  "success",
			Data: songList,
		})
		return
	// 2、QQ 音乐
	case qqMusicRegx.MatchString(link):
		songList, err := logic.QQMusicDiscover(link)
		if err != nil {
			log.Errorf("fail to get qqmusic discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
		}
		c.JSON(200, &models.Result{
			Code: 1,
			Msg:  "success",
			Data: songList,
		})
		return
	// 3、都不是
	default:
		c.JSON(http.StatusBadRequest, nil)
	}
}
