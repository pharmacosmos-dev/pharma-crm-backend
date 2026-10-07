package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain/constants"
)

const (
	incassoReportDays   = 10
	telegramMessageMax  = 4096
	incassoTimeLayout   = "2006-01-02 15:04:05"
	incassoTashkentDiff = 5 * time.Hour
)

// incassoRecipients — naqd savdo hisobotini oluvchi inkassatorlar.
// Hisobot FAQAT shu ro'yxatdagi Telegram chat ID'larga yuboriladi; botga /start bosgan
// boshqa hech kim hech narsa olmaydi. Har bir oluvchi EmployeeId'dagi xodimning
// store_ids'idagi aptekalar bo'yicha hisobot oladi.
// Yangi inkassator qo'shish: u botga /start bossin, keyin uning Telegram ID'sini shu yerga yozing.
var incassoRecipients = []struct {
	Name       string
	ChatId     string
	EmployeeId string
}{
	{Name: "Lazizbek", ChatId: "6027354863", EmployeeId: "997d7b8e-48c1-476a-9487-a7c40f786f39"},
}

type incassoStore struct {
	Id   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

type incassoDayCash struct {
	StoreId string    `gorm:"column:store_id"`
	Day     time.Time `gorm:"column:day"`
	Cash    float64   `gorm:"column:cash"`
}

// SendIncassoCashReport — har bir inkassatorga o'z xodimining store_ids'idagi aptekalar bo'yicha
// oxirgi 10 kunlik (bugundan oldingi to'liq Toshkent kunlari) naqd savdo summasini yuboradi.
func (s *Services) SendIncassoCashReport() {
	botToken := s.cfg.Telegram.IncassoBotToken
	if botToken == "" {
		s.log.Errorf("incasso report: TELEGRAM_INCASSO_BOT_TOKEN is not configured")
		return
	}

	for _, r := range incassoRecipients {
		messages, err := s.buildIncassoCashReport(r.EmployeeId)
		if err != nil {
			s.log.Errorf("incasso report: %s (%s): %v", r.Name, r.ChatId, err)
			continue
		}
		for _, msg := range messages {
			if err := sendTelegramMessage(botToken, r.ChatId, msg); err != nil {
				s.log.Errorf("incasso report: could not send to %s (%s): %v", r.Name, r.ChatId, err)
				break
			}
		}
	}
}

// buildIncassoCashReport xodimning store_ids'i bo'yicha hisobot xabarlarini tuzadi.
// Qaytarim cheklarida cash manfiy saqlanadi, shuning uchun SUM sof naqd summani beradi.
func (s *Services) buildIncassoCashReport(employeeId string) ([]string, error) {
	var stores []incassoStore
	err := s.db.Raw(`
		SELECT st.id, st.name
		FROM stores st
		WHERE st.id::text IN (SELECT unnest(e.store_ids) FROM employees e WHERE e.id = ?)
		ORDER BY st.name`, employeeId).Scan(&stores).Error
	if err != nil {
		return nil, fmt.Errorf("could not get employee stores: %w", err)
	}
	if len(stores) == 0 {
		return nil, fmt.Errorf("employee %s has no store_ids", employeeId)
	}

	// sales.completed_at — UTC wall-clock (timestamp without time zone),
	// shuning uchun Toshkent kun chegaralarini UTC ga o'girib, satr sifatida beramiz.
	tashkent := time.FixedZone("Asia/Tashkent", int(incassoTashkentDiff.Seconds()))
	now := time.Now().In(tashkent)
	endDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tashkent)
	startDay := endDay.AddDate(0, 0, -incassoReportDays)

	storeIds := make([]string, 0, len(stores))
	for _, st := range stores {
		storeIds = append(storeIds, st.Id)
	}

	var rows []incassoDayCash
	err = s.db.Raw(`
		SELECT
			s.store_id::text AS store_id,
			(s.completed_at + INTERVAL '5 hours')::date AS day,
			COALESCE(SUM(s.cash), 0) AS cash
		FROM sales s
		WHERE s.store_id IN (?)
		  AND s.stage IN (?)
		  AND s.completed_at >= ?::timestamp
		  AND s.completed_at < ?::timestamp
		GROUP BY 1, 2`,
		storeIds, constants.FinishedSaleStages,
		startDay.UTC().Format(incassoTimeLayout), endDay.UTC().Format(incassoTimeLayout),
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("could not get cash sales: %w", err)
	}

	cashByStoreDay := make(map[string]float64, len(rows))
	for _, r := range rows {
		cashByStoreDay[r.StoreId+"|"+r.Day.Format("2006-01-02")] = r.Cash
	}

	header := fmt.Sprintf("💵 <b>Naqd savdo hisoboti</b>\n%s — %s (%d kun)",
		startDay.Format("02.01.2006"), endDay.AddDate(0, 0, -1).Format("02.01.2006"), incassoReportDays)

	blocks := []string{header}
	var grandTotal float64
	for _, st := range stores {
		var b strings.Builder
		var storeTotal float64
		fmt.Fprintf(&b, "🏪 <b>%s</b>\n", html.EscapeString(st.Name))
		for d := startDay; d.Before(endDay); d = d.AddDate(0, 0, 1) {
			cash := cashByStoreDay[st.Id+"|"+d.Format("2006-01-02")]
			storeTotal += cash
			fmt.Fprintf(&b, "%s — %s\n", d.Format("02.01"), formatSum(cash))
		}
		fmt.Fprintf(&b, "<b>Jami: %s</b>", formatSum(storeTotal))
		grandTotal += storeTotal
		blocks = append(blocks, b.String())
	}
	if len(stores) > 1 {
		blocks = append(blocks, fmt.Sprintf("━━━━━━━━━━\n<b>Umumiy jami: %s</b>", formatSum(grandTotal)))
	}

	return joinTelegramBlocks(blocks, "\n\n"), nil
}

// joinTelegramBlocks bloklarni Telegram limitidan oshmaydigan xabarlarga bo'ladi.
func joinTelegramBlocks(blocks []string, sep string) []string {
	var (
		messages []string
		current  strings.Builder
	)
	for _, block := range blocks {
		if current.Len() > 0 && current.Len()+len(sep)+len(block) > telegramMessageMax {
			messages = append(messages, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(sep)
		}
		current.WriteString(block)
	}
	if current.Len() > 0 {
		messages = append(messages, current.String())
	}
	return messages
}

func sendTelegramMessage(botToken, chatID, text string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken), "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// formatSum 1234567.8 -> "1 234 568"
func formatSum(v float64) string {
	n := int64(math.Round(v))
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	digits := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	return sign + b.String()
}
