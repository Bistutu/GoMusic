package handler

import (
	"GoMusic/internal/config"
	"GoMusic/logic"
	"GoMusic/misc/models"
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type HealthCheck func(context.Context) error

func NewRouter() *gin.Engine {
	cfg, _ := config.FromEnv(func(string) string { return "" })
	return NewRouterWithService(logic.NewService(nil, nil, nil, 10000, 4), cfg, nil)
}
func NewRouterWithService(service Fetcher, cfg config.Config, checks map[string]HealthCheck) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.CustomRecovery(func(c *gin.Context, _ any) { replyError(c, http.StatusInternalServerError, "服务暂时不可用") }))
	_ = router.SetTrustedProxies(nil)
	if len(cfg.AllowedOrigins) > 0 {
		router.Use(cors.New(cors.Config{AllowOrigins: cfg.AllowedOrigins, AllowMethods: []string{"GET", "POST", "OPTIONS"}, AllowHeaders: []string{"Content-Type"}, MaxAge: 12 * time.Hour}))
	}
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		status := http.StatusOK
		details := map[string]string{}
		for name, check := range checks {
			if err := check(ctx); err != nil {
				details[name] = "unavailable"
				status = http.StatusServiceUnavailable
			} else {
				details[name] = "ok"
			}
		}
		state := "ok"
		if status != http.StatusOK {
			state = "degraded"
		}
		c.JSON(status, gin.H{"status": state, "dependencies": details})
	})
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	h := musicHandler{fetcher: service, timeout: timeout}
	router.POST("/songlist", h.serve)
	router.POST("/api/v1/songlists", h.serve)
	router.GET("/", func(c *gin.Context) {
		index := filepath.Join(cfg.StaticDir, "index.html")
		if _, err := os.Stat(index); err != nil {
			c.JSON(http.StatusOK, gin.H{"service": "GoMusic", "api": "/api/v1/songlists"})
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.File(index)
	})
	if info, err := os.Stat(filepath.Join(cfg.StaticDir, "assets")); err == nil && info.IsDir() {
		router.Static("/assets", filepath.Join(cfg.StaticDir, "assets"))
	}
	router.GET("/favicon.svg", func(c *gin.Context) { c.File(filepath.Join(cfg.StaticDir, "favicon.svg")) })
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, models.Result{Code: models.FailureCode, Msg: "接口不存在", Data: nil})
	})
	return router
}
