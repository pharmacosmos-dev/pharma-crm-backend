package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

func (s *Services) SaveAsilBelgiToken(db *gorm.DB, req *domain.AsilBelgiTokenRequest) error {
	// deactivate old tokens
	if err := db.Model(&domain.AsilBelgiToken{}).
		Where("is_active = true").
		Update("is_active", false).Error; err != nil {
		return err
	}

	// parse expires_at if given
	var expiresAt time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = t
		}
	} else {
		expiresAt = time.Now().Add(time.Hour * 10)
	}

	// insert new token
	token := domain.AsilBelgiToken{
		Token:     req.Token,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}
	return db.Create(&token).Error
}

// FetchCisInfo asl belgi API dan productName va gtin ni olib keladi.
// Agar 429 qaytsa 20 soniya kutib qaytadan yuboradi.
func (s *Services) FetchCisInfo(markingCode string) (*domain.CisInfo, error) {
	url := "https://aslbelgisi.uz/api/v3/true-api/cises/info?pg=pharma"
	var token domain.AsilBelgiToken
	result := s.db.
		Where("is_active = ?", true).
		First(&token)
	if result.Error != nil {
		s.log.Error("ERROR on fetching asil belgi token: %v", result.Error)
		return nil, result.Error
	}
	reqBody, _ := json.Marshal([]string{markingCode})
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		s.log.Error("ERROR on creating request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Token)

	client := &http.Client{Timeout: 15 * time.Second}

	for i := 0; i < 2; i++ { // 2 marta urinib ko‘ramiz
		resp, err := client.Do(req)
		if err != nil {
			s.log.Error("ERROR on doing request: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(60 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			s.log.Error("ERROR on doing request: %v", fmt.Errorf("unexpected status: %d", resp.StatusCode))
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}

		var data []struct {
			CisInfo domain.CisInfo `json:"cisInfo"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			s.log.Error("ERROR on decoding response: %v", err)
			return nil, err
		}
		if len(data) == 0 {
			s.log.Error("ERROR on decoding response: %v", fmt.Errorf("empty response"))
			return nil, fmt.Errorf("empty response")
		}
		return &data[0].CisInfo, nil
	}

	return nil, fmt.Errorf("too many requests, retry failed")
}
