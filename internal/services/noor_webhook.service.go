package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

const (
	noorWebhookAttempts       = 3
	noorWebhookAttemptTimeout = 8 * time.Second
)

// SendNoorOrderStatus sends the order status to the noor webhook.
// 429, 5xx and network errors are retried: noor says a repeat is safe.
// 400/401/404/413 are not retried, the request itself has to be fixed.
func (s *Services) SendNoorOrderStatus(ctx context.Context, req *domain.NoorOrderStatusRequest) error {
	if s.cfg.NoorWebhookUrl == "" || s.cfg.NoorWebhookKey == "" {
		s.log.Errorf("noor webhook is not configured (NOOR_WEBHOOK_URL, NOOR_WEBHOOK_KEY), %s status of order %d is not sent",
			req.Status, req.OrderId)
		return domain.NoorNotOperationalError
	}

	// noor finds the order by its own id
	if req.ExternalId == "" {
		s.log.Errorf("noor order %d has no external_id, %s status is not sent", req.OrderId, req.Status)
		return domain.NoorNotOperationalError
	}

	data, err := json.Marshal(req)
	if err != nil {
		s.log.Errorf("could not marshal noor order status request: %v", err)
		return domain.InternalServerError
	}

	var lastErr error
	attempts:
	for attempt := 1; attempt <= noorWebhookAttempts; attempt++ {
		if attempt > 1 {
			// pause: 1s, then 2s
			select {
			case <-ctx.Done():
				lastErr = fmt.Errorf("%v (last error: %v)", ctx.Err(), lastErr)
				break attempts
			case <-time.After(time.Duration(attempt-1) * time.Second):
			}
		}

		statusCode, body, err := s.doNoorWebhookRequest(ctx, data)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			return nil
		}

		lastErr = fmt.Errorf("status %d, body: %s", statusCode, string(body))
		if statusCode != http.StatusTooManyRequests && statusCode < http.StatusInternalServerError {
			break
		}
	}

	s.log.Errorf("could not send %s status of noor order %d (external_id %s): %v",
		req.Status, req.OrderId, req.ExternalId, lastErr)
	return domain.NoorNotOperationalError
}

// SendNoorOrderCompleted sends the completed status with every pack of the finished noor sale.
// noor issues the customer's receipt from these packs.
func (s *Services) SendNoorOrderCompleted(saleId string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var sale struct {
		SaleNumber    int    `gorm:"column:sale_number"`
		VendorOrderId string `gorm:"column:vendor_order_id"`
	}
	err := s.db.WithContext(ctx).Table("sales").
		Select("sale_number, COALESCE(vendor_order_id, '') AS vendor_order_id").
		Where("id = ?", saleId).
		Take(&sale).Error
	if err != nil {
		s.log.Errorf("could not get noor sale %s for completed status: %v", saleId, err)
		return
	}

	var rows []struct {
		ProductId    string            `gorm:"column:product_id"`
		UnitQuantity int               `gorm:"column:unit_quantity"`
		UnitPerPack  int               `gorm:"column:unit_per_pack"`
		Markings     utils.StringArray `gorm:"column:markings;type:text[]"`
		Barcode      string            `gorm:"column:barcode"`
		ClassCode    string            `gorm:"column:class_code"`
		PackageCode  string            `gorm:"column:package_code"`
		VatPercent   int               `gorm:"column:vat_percent"`
	}
	err = s.db.WithContext(ctx).Raw(`
	SELECT
		sp.product_id,
		ci.unit_quantity,
		p.unit_per_pack,
		ci.markings,
		COALESCE(ci.barcode, '') AS barcode,
		COALESCE(pb.mxik, sp.mxik, p.mxik, '') AS class_code,
		COALESCE(pb.unit_code, sp.unit_code, p.unit_code, '') AS package_code,
		COALESCE(sp.vat, 0) AS vat_percent
	FROM cart_items ci
	JOIN store_products sp ON sp.id = ci.store_product_id
	JOIN products p ON p.id = sp.product_id
	LEFT JOIN LATERAL (
		SELECT mxik, unit_code
		FROM product_barcodes
		WHERE product_id = p.id
		AND barcode = ci.barcode
		LIMIT 1
	) pb ON true
	WHERE ci.sale_id = ?
	ORDER BY ci.created_at`, saleId).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("could not get cart_items of noor sale %d for completed status: %v", sale.SaleNumber, err)
		return
	}

	items := make([]domain.NoorOrderStatusItem, 0, len(rows))
	for _, r := range rows {
		// noor counts whole packs; a part of a pack can't be sent
		if r.UnitPerPack <= 0 || r.UnitQuantity%r.UnitPerPack != 0 {
			s.log.Errorf("noor order %d: product %s has %d units, not whole packs of %d, completed status is not sent",
				sale.SaleNumber, r.ProductId, r.UnitQuantity, r.UnitPerPack)
			return
		}
		packs := r.UnitQuantity / r.UnitPerPack

		// all packs of a product have a label, or none does
		markings := make([]string, 0, len(r.Markings))
		for _, m := range r.Markings {
			if m = strings.TrimSpace(m); m != "" {
				markings = append(markings, m)
			}
		}
		if len(markings) > 0 && len(markings) != packs {
			s.log.Errorf("noor order %d: product %s has %d markings for %d packs, completed status is not sent",
				sale.SaleNumber, r.ProductId, len(markings), packs)
			return
		}

		// noor accepts an 8-14 digit barcode only
		var barcode *string
		if b := strings.TrimSpace(r.Barcode); len(b) >= 8 && len(b) <= 14 && isDigits(b) {
			barcode = &b
		}

		for i := 0; i < packs; i++ {
			label := ""
			if len(markings) > 0 {
				label = markings[i]
			}
			items = append(items, domain.NoorOrderStatusItem{
				ProductId:   r.ProductId,
				Label:       label,
				Barcode:     barcode,
				ClassCode:   r.ClassCode,
				PackageCode: r.PackageCode,
				VatPercent:  r.VatPercent,
			})
		}
	}

	if len(items) == 0 {
		s.log.Errorf("noor order %d has no packs, completed status is not sent", sale.SaleNumber)
		return
	}

	_ = s.SendNoorOrderStatus(ctx, &domain.NoorOrderStatusRequest{
		ExternalId: sale.VendorOrderId,
		OrderId:    sale.SaleNumber,
		Status:     constants.NoorOrderStatusCompleted,
		Items:      items,
	})
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// one webhook call: returns status code and (limited) response body
func (s *Services) doNoorWebhookRequest(ctx context.Context, data []byte) (int, []byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, noorWebhookAttemptTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, s.cfg.NoorWebhookUrl, bytes.NewReader(data))
	if err != nil {
		return 0, nil, fmt.Errorf("could not construct request: %w", err)
	}

	// server env'da qiymat qo'shtirnoq bilan yozilib qolsa ham kalit buzilmasin
	key := strings.Trim(s.cfg.NoorWebhookKey, `"'`)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJson)
	req.Header.Set("Accept", constants.ContentTypeJson)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, body, nil
}
