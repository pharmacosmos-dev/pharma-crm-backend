package middleware

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/utils"
)

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func TelegramErrorLogger(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		bw := &bodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = bw

		c.Next()

		status := c.Writer.Status()
		if status != 400 && status != 500 {
			return
		}

		env := strings.ToUpper(cfg.App.Env)
		if env != "PROD" && env != "PRODUCTION" {
			return
		}

		utils.SendTelegramError(utils.TelegramErrorParams{
			BotToken:   cfg.Telegram.BotToken,
			ChannelID:  cfg.Telegram.ChannelID,
			Status:     status,
			Method:     c.Request.Method,
			RequestURL: c.Request.URL.String(),
			Body:       bw.body.String(),
			IP:         c.ClientIP(),
			Token:      c.GetHeader("Authorization"),
			UserAgent:  c.GetHeader("User-Agent"),
			ServerName: cfg.App.Name,
		})
	}
}
