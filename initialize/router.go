package initialize

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"GoMusic/handler"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	// 允许所有跨域请求
	router.Use(cors.Default())
	// 加载静态资源
	router.StaticFile("/", "./static")
	// 绑定路由
	router.POST("/songlist", handler.MusicHandler)
	return router
}
