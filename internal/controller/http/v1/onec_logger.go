package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	onecLogInsertTimeout  = 3 * time.Second
	onecLogUpdateTimeout  = 5 * time.Second
	onecLogMaxResponseKB  = 512 * 1024 // 512 KB dan katta response saqlanmaydi
)

type onecResponseWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	truncated bool
}

func (w *onecResponseWriter) Write(b []byte) (int, error) {
	if w.body.Len() < onecLogMaxResponseKB {
		w.body.Write(b)
	} else {
		w.truncated = true
	}
	return w.ResponseWriter.Write(b)
}

func OnecRequestLogger(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload json.RawMessage

		if c.Request.Method == "GET" {
			params := make(map[string]string)
			for k, v := range c.Request.URL.Query() {
				if len(v) > 0 {
					params[k] = v[0]
				}
			}
			if b, err := json.Marshal(params); err == nil {
				payload = json.RawMessage(b)
			}
		} else {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if json.Valid(bodyBytes) && len(bodyBytes) > 0 {
				payload = json.RawMessage(bodyBytes)
			} else {
				payload = json.RawMessage(`{}`)
			}
		}

		token := c.GetHeader("Authorization")
		if len(token) > 30 {
			token = token[:30] + "..."
		}

		method := c.Request.Method + " " + c.FullPath()
		ipAddress := c.ClientIP()

		// INSERT with timeout — DB ishlamasa handler baribir ishlaydi
		var logID string
		insertCtx, insertCancel := context.WithTimeout(context.Background(), onecLogInsertTimeout)
		defer insertCancel()
		if err := db.WithContext(insertCtx).Raw(
			`INSERT INTO onec_requests (method, payload, token, ip_address) VALUES (?, ?, ?, ?) RETURNING id`,
			method, payload, token, ipAddress,
		).Scan(&logID).Error; err != nil {
			log.Errorf("onec_logger: could not insert log: %v", err)
		}

		blw := &onecResponseWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer, truncated: false}
		c.Writer = blw

		start := time.Now()
		c.Next()
		durationMs := time.Since(start).Milliseconds()

		if logID == "" {
			return
		}

		responseBytes := blw.body.Bytes()
		truncated := blw.truncated
		statusCode := c.Writer.Status()

		// async: client already got response, update log in background
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), onecLogUpdateTimeout)
			defer cancel()

			var responsePayload json.RawMessage
			if truncated {
				responsePayload = json.RawMessage(`{"truncated": true, "reason": "response exceeded 512KB"}`)
			} else if json.Valid(responseBytes) && len(responseBytes) > 0 {
				responsePayload = json.RawMessage(responseBytes)
			} else {
				responsePayload = json.RawMessage(`{}`)
			}
			if err := db.WithContext(ctx).Exec(
				`UPDATE onec_requests SET response = ?, status_code = ?, duration_ms = ?, updated_at = NOW() WHERE id = ?`,
				responsePayload, statusCode, durationMs, logID,
			).Error; err != nil {
				log.Errorf("onec_logger: could not update log: %v", err)
			}
		}()
	}
}
