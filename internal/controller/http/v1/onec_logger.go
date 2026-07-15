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
	logInsertTimeout = 3 * time.Second
	logUpdateTimeout = 5 * time.Second
	logMaxResponseKB = 512 * 1024
)

type requestLogWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	truncated bool
}

func (w *requestLogWriter) Write(b []byte) (int, error) {
	if w.body.Len() < logMaxResponseKB {
		w.body.Write(b)
	} else {
		w.truncated = true
	}
	return w.ResponseWriter.Write(b)
}

// newRequestLogger — generic middleware, istalgan tablega log yozadi
func newRequestLogger(db *gorm.DB, log *logger.Logger, table string) gin.HandlerFunc {
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

		var logID string
		insertCtx, insertCancel := context.WithTimeout(context.Background(), logInsertTimeout)
		defer insertCancel()
		//nolint:gosec
		if err := db.WithContext(insertCtx).Raw(
			`INSERT INTO `+table+` (method, payload, token, ip_address) VALUES (?, ?, ?, ?) RETURNING id`,
			method, payload, token, ipAddress,
		).Scan(&logID).Error; err != nil {
			log.Errorf("%s logger: could not insert log: %v", table, err)
		}

		blw := &requestLogWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer, truncated: false}
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

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), logUpdateTimeout)
			defer cancel()

			var responsePayload json.RawMessage
			if truncated {
				responsePayload = json.RawMessage(`{"truncated": true, "reason": "response exceeded 512KB"}`)
			} else if json.Valid(responseBytes) && len(responseBytes) > 0 {
				responsePayload = json.RawMessage(responseBytes)
			} else {
				responsePayload = json.RawMessage(`{}`)
			}
			//nolint:gosec
			if err := db.WithContext(ctx).Exec(
				`UPDATE `+table+` SET response = ?, status_code = ?, duration_ms = ?, updated_at = NOW() WHERE id = ?`,
				responsePayload, statusCode, durationMs, logID,
			).Error; err != nil {
				log.Errorf("%s logger: could not update log: %v", table, err)
			}
		}()
	}
}

func OnecRequestLogger(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return newRequestLogger(db, log, "onec_requests")
}

func UzumOrderLogger(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return newRequestLogger(db, log, "uzum_order_logs")
}

// UzumRequestCounter — chaqiruv sonini hisoblash va so'rov parametrlarini (path + query) yozish uchun
func UzumRequestCounter(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method + " " + c.FullPath()
		ipAddress := c.ClientIP()

		params := make(map[string]string, len(c.Params)+2)
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
		var payload json.RawMessage
		if b, err := json.Marshal(params); err == nil {
			payload = json.RawMessage(b)
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), logInsertTimeout)
			defer cancel()
			if err := db.WithContext(ctx).Exec(
				`INSERT INTO uzum_order_logs (method, payload, ip_address) VALUES (?, ?, ?)`,
				method, payload, ipAddress,
			).Error; err != nil {
				log.Errorf("uzum request counter: could not insert log: %v", err)
			}
		}()

		c.Next()
	}
}
