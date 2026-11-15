package handler

import (
	"GoMusic/logic"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Fetcher interface {
	Fetch(context.Context, string, logic.Options) (*models.SongList, error)
}
type musicHandler struct {
	fetcher Fetcher
	timeout time.Duration
}

func (h musicHandler) serve(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var input struct {
		URL      string `json:"url"`
		Detailed bool   `json:"detailed"`
		Format   string `json:"format"`
		Order    string `json:"order"`
	}
	contentType := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if contentType == "application/json" {
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			requestError(c, err)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			replyError(c, http.StatusBadRequest, "请求必须是单个 JSON 对象")
			return
		}
	} else if contentType == "application/x-www-form-urlencoded" || contentType == "" {
		if err := c.Request.ParseForm(); err != nil {
			requestError(c, err)
			return
		}
		input.URL = c.Request.PostForm.Get("url")
	} else {
		replyError(c, http.StatusUnsupportedMediaType, "请使用 JSON 或表单提交歌单链接或酷狗码")
		return
	}
	if value, ok := c.GetQuery("detailed"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			replyError(c, http.StatusBadRequest, "detailed 必须为 true 或 false")
			return
		}
		input.Detailed = parsed
	}
	if value, ok := c.GetQuery("format"); ok {
		input.Format = value
	}
	if value, ok := c.GetQuery("order"); ok {
		input.Order = value
	}
	if input.Format != "" && input.Format != "song-singer" && input.Format != "singer-song" && input.Format != "song" {
		replyError(c, http.StatusBadRequest, "不支持的歌曲格式")
		return
	}
	if input.Order != "" && input.Order != "normal" && input.Order != "reverse" {
		replyError(c, http.StatusBadRequest, "不支持的歌曲顺序")
		return
	}
	if strings.TrimSpace(input.URL) == "" || len(input.URL) > 4096 {
		replyError(c, http.StatusBadRequest, "请输入有效的歌单链接或酷狗码（最多 4096 字节）")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	list, err := h.fetcher.Fetch(ctx, input.URL, logic.Options{Detailed: input.Detailed, Format: input.Format, Order: input.Order})
	if err != nil {
		if errors.Is(err, logic.ErrPrivatePlaylist) {
			replyError(c, http.StatusForbidden, "该歌单未公开，请先将歌单设置为公开后重试")
			return
		}
		if errors.Is(err, logic.ErrInvalidLink) {
			replyError(c, http.StatusBadRequest, strings.TrimPrefix(err.Error(), logic.ErrInvalidLink.Error()+": "))
			return
		}
		var timeout net.Error
		if (errors.As(err, &timeout) && timeout.Timeout()) || errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			replyError(c, http.StatusGatewayTimeout, "获取歌单超时，请稍后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			return
		}
		log.Warnf("获取歌单失败: %v", err)
		// Keep transport and database details out of the public API.
		replyError(c, http.StatusBadGateway, "音乐平台暂时未返回可读取的歌单，请检查链接和公开权限后重试")
		return
	}
	c.Set(usageOutcomeKey, usageOutcome{Succeeded: true, Partial: !list.Complete, Songs: int64(len(list.Tracks))})
	c.JSON(http.StatusOK, models.Result{Code: models.SuccessCode, Msg: "success", Data: list})
}
func requestError(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		replyError(c, http.StatusRequestEntityTooLarge, "请求内容过长")
		return
	}
	replyError(c, http.StatusBadRequest, "请求内容无效，请检查 JSON 或表单格式")
}
func replyError(c *gin.Context, status int, message string) {
	c.JSON(status, models.Result{Code: models.FailureCode, Msg: message, Data: nil})
}
