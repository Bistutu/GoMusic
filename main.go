package main

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gin-contrib/cors"
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

func main() {
	r := gin.Default()
	r.Use(cors.Default())         // 允许所有跨域请求
	r.StaticFile("/", "./static") // 加载静态资源

	r.POST("/songlist", func(c *gin.Context) {
		// 判断为网易云、QQ音乐（后续还可能扩展）
		form := c.PostForm("url")
		switch {
		case netEasyRegx.MatchString(form):
			songList, err := logic.NetEasyDiscover(form)
			if err != nil {
				log.Printf("fail to get net easy discover: %v", err)
				c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
				return
			}
			c.JSON(200, &models.Result{
				Code: 1,
				Msg:  "success",
				Data: songList,
			})
		case qqMusicRegx.MatchString(form):
			songList, err := logic.QQMusicDiscover(form)
			if err != nil {
				log.Printf("fail to get qq music discover: %v", err)
				c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
				return
			}
			c.JSON(200, &models.Result{
				Code: 1,
				Msg:  "success",
				Data: songList,
			})
			return
		default:
			c.JSON(http.StatusBadRequest, nil)
		}
	})
	r.Run(":8081")
}
