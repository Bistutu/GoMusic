package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())             // allow all origins
	router.StaticFile("/", "./static")     // load static files
	router.POST("/songlist", MusicHandler) // bind route to handler
	return router
}
