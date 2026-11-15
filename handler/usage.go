package handler

import (
	"GoMusic/internal/usage"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const visitorCookie = "gomusic_visitor"
const usageOutcomeKey = "gomusic_usage_outcome"

type usageOutcome struct {
	Succeeded, Partial bool
	Songs              int64
}

type usageHandler struct {
	repo         usage.Repository
	token        string
	cookieSecure bool
	now          func() time.Time
}

func (h usageHandler) record(c *gin.Context) {
	if h.repo == nil {
		c.Next()
		return
	}
	id, _ := c.Cookie(visitorCookie)
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 16 {
		var value [16]byte
		if _, err := rand.Read(value[:]); err != nil {
			log.Warnf("生成匿名使用标识失败: %v", err)
			c.Next()
			return
		}
		id = hex.EncodeToString(value[:])
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(visitorCookie, id, 365*24*60*60, "/", "", h.cookieSecure || c.Request.TLS != nil, true)
	}
	hash := sha256.Sum256([]byte(strings.ToLower(id)))
	defer func() {
		value, _ := c.Get(usageOutcomeKey)
		outcome, _ := value.(usageOutcome)
		// Cancelled requests and recovered panics still count as attempts.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 2*time.Second)
		defer cancel()
		if err := h.repo.RecordUsage(ctx, usage.Event{At: h.now(), VisitorHash: hex.EncodeToString(hash[:]), Succeeded: outcome.Succeeded, Partial: outcome.Partial, Songs: outcome.Songs}); err != nil {
			log.Warnf("保存使用统计失败: %v", err)
		}
	}()
	c.Next()
}

func (h usageHandler) stats(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if h.token == "" {
		replyError(c, http.StatusNotFound, "统计接口未启用")
		return
	}
	provided, bearer := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !bearer || subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
		replyError(c, http.StatusUnauthorized, "统计接口需要有效的访问令牌")
		return
	}
	if h.repo == nil {
		replyError(c, http.StatusServiceUnavailable, "统计持久化存储未配置")
		return
	}
	now := h.now()
	anchor := now
	if date := c.Query("date"); date != "" {
		var err error
		anchor, err = time.ParseInLocation("2006-01-02", date, usage.Shanghai)
		if err != nil {
			replyError(c, http.StatusBadRequest, "date 必须为 YYYY-MM-DD")
			return
		}
	}
	periods := []string{"day", "week", "month", "all"}
	if period := c.Query("period"); period != "" {
		periods = []string{period}
	}
	results := make(map[string]usage.Result, len(periods))
	for _, period := range periods {
		window, err := usage.Period(period, anchor)
		if err != nil {
			replyError(c, http.StatusBadRequest, err.Error())
			return
		}
		result, err := h.repo.UsageStats(c.Request.Context(), window)
		if err != nil {
			log.Warnf("读取使用统计失败: %v", err)
			replyError(c, http.StatusServiceUnavailable, "统计暂时不可读取，请稍后重试")
			return
		}
		results[period] = result
	}
	c.JSON(http.StatusOK, models.Result{Code: models.SuccessCode, Msg: "success", Data: gin.H{"timezone": "Asia/Shanghai", "generated_at": now.In(usage.Shanghai).Format(time.RFC3339), "periods": results}})
}
