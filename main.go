package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"GoMusic/logic"
	"GoMusic/models"
)

func main() {
	r := gin.Default()
	r.Use(cors.Default())         // 允许所有跨域请求
	r.StaticFile("/", "./static") // 加载静态资源

	r.POST("/neteasy", func(c *gin.Context) {
		// 判断链接是否为网易云歌单链接并标准化
		link, err := logic.IsNetEasyDiscover(c.PostForm("url"))
		if err != nil {
			log.Printf("无效的链接格式：%s", link)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
			return
		}
		netEasyDiscover, err := logic.NetEasyDiscover(link)
		if err != nil {
			log.Printf("fail to get net easy discover: %v", err)
			c.JSON(http.StatusBadRequest, &models.Result{Code: -1, Msg: err.Error(), Data: nil})
			return
		}
		c.JSON(200, &models.Result{
			Code: 1,
			Msg:  "success",
			Data: netEasyDiscover,
		})
	})
	r.Run(":8081")
}
