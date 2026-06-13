package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type TelegramErrorParams struct {
	BotToken   string
	ChannelID  string
	Status     int
	Method     string
	RequestURL string
	Body       string
	IP         string
	Token      string
	UserAgent  string
	ServerName string
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func SendTelegramError(p TelegramErrorParams) {
	if p.BotToken == "" || p.ChannelID == "" {
		return
	}

	emoji := "🔵"
	if p.Status == 500 {
		emoji = "🔴"
	}

	text := fmt.Sprintf(
		"<b>%s STATUS:</b> %d\n"+
			"<b>🖥 SERVER:</b> %s\n"+
			"<b>📬 URL:</b> (%s) %s\n"+
			"<b>🌐 IP:</b> %s\n"+
			"<b>👤 TOKEN:</b> %s\n"+
			"<b>🔎 USER-AGENT:</b> %s\n"+
			"<b>🚧 ERROR:</b> %s",
		emoji, p.Status,
		escapeHTML(p.ServerName),
		p.Method, escapeHTML(p.RequestURL),
		p.IP,
		escapeHTML(p.Token),
		escapeHTML(p.UserAgent),
		escapeHTML(truncate(p.Body, 500)),
	)

	// Telegram max 4096 char
	if len(text) > 4096 {
		text = text[:4093] + "..."
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.BotToken)

	payload, _ := json.Marshal(map[string]string{
		"chat_id":    p.ChannelID,
		"text":       text,
		"parse_mode": "HTML",
	})

	go func() {
		resp, err := http.Post(apiURL, "application/json", bytes.NewReader(payload)) //nolint:gosec
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
}
