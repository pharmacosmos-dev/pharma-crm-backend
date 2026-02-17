package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain/constants"
)

type StoreProductStock struct {
	Id           int       `gorm:"id" json:"id"`
	StoreId      string    `gorm:"store_id" json:"store_id"`
	ProductId    string    `gorm:"product_id" json:"product_id"`
	UnitQuantity int       `gorm:"unit_quantity" json:"unit_quantity"`
	MinPrice     float64   `gorm:"min_price" json:"min_price"`
	MaxPrice     float64   `gorm:"max_price" json:"max_price"`
	CreatedAt    time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"updated_at" json:"updated_at"`
}

func (s *StoreProductStock) TableName() string {
	return "store_product_stocks"
}

// region Import

func (s *Services) createOrUpdateStocksAfterImportConfirm(importId string, storeId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	details, err := s.getConfirmedImportDetails(ctx, importId)
	if err != nil {
		return err
	}

	if len(details) == 0 {
		return nil
	}

	const batchSize = 500

	for i := 0; i < len(details); i += batchSize {
		end := i + batchSize
		if end > len(details) {
			end = len(details)
		}
		batch := details[i:end]

		valueStrings := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*5)

		for _, detail := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?)")
			args = append(args, storeId, detail.ProductId, detail.UnitQuantity, detail.RetailPrice, detail.RetailPrice)
		}

		query := fmt.Sprintf(`
		INSERT INTO store_product_stocks (store_id, product_id, unit_quantity, min_price, max_price)
		VALUES %s
		ON CONFLICT (store_id, product_id) DO UPDATE SET
			unit_quantity = store_product_stocks.unit_quantity + EXCLUDED.unit_quantity,
			min_price     = LEAST(store_product_stocks.min_price, EXCLUDED.min_price),
			max_price     = GREATEST(store_product_stocks.max_price, EXCLUDED.max_price),
			updated_at    = NOW();
		`, strings.Join(valueStrings, ","))

		if err := s.db.WithContext(ctx).Exec(query, args...).Error; err != nil {
			return fmt.Errorf("failed to batch upsert store_product_stocks: %w", err)
		}
	}

	return nil
}

type confirmedDetails struct {
	ProductId    string  `json:"product_id"`
	UnitQuantity int     `json:"unit_quantity"`
	RetailPrice  float64 `json:"retail_price"`
}

// get confirmed import details for create or update store_product_stocks data
func (s *Services) getConfirmedImportDetails(ctx context.Context, importId string) ([]confirmedDetails, error) {
	var details []confirmedDetails
	query := `
	SELECT
		imd.product_id,
		ROUND(imd.scanned_count * p.unit_per_pack) AS unit_quantity,
		imd.retail_price_vat AS retail_price
	FROM import_details imd
	JOIN products p ON imd.product_id = p.id
	WHERE imd.import_id = ?;
	`
	err := s.db.WithContext(ctx).Raw(query, importId).Scan(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

// end region

// region Sale
func (s *Services) updateStocksAfterSaleFinished(saleId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	query := `
	UPDATE store_product_stocks sps
	SET
		unit_quantity = sps.unit_quantity - ci_agg.total_qty,
		updated_at    = NOW()
	FROM (
		SELECT ci.product_id, SUM(ci.unit_quantity) AS total_qty
		FROM cart_items ci
		WHERE ci.sale_id = ?
		GROUP BY ci.product_id
	) ci_agg
	JOIN sales s ON s.id = ?
	WHERE sps.store_id = s.store_id
	  AND sps.product_id = ci_agg.product_id;
	`

	if err := s.db.WithContext(ctx).Exec(query, saleId, saleId).Error; err != nil {
		return fmt.Errorf("failed to update store_product_stocks after sale %s: %w", saleId, err)
	}

	return nil
}

func (s *Services) updateStocksAfterReturnSaleFinished(saleId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	query := `
	UPDATE store_product_stocks sps
	SET
		unit_quantity = sps.unit_quantity + ci_agg.total_qty,
		updated_at    = NOW()
	FROM (
		SELECT ci.product_id, SUM(ci.unit_quantity) AS total_qty
		FROM cart_items ci
		WHERE ci.sale_id = ?
		GROUP BY ci.product_id
	) ci_agg
	JOIN sales s ON s.id = ?
	WHERE sps.store_id = s.store_id
	  AND sps.product_id = ci_agg.product_id;
	`

	if err := s.db.WithContext(ctx).Exec(query, saleId, saleId).Error; err != nil {
		return fmt.Errorf("failed to update store_product_stocks after return sale %s: %w", saleId, err)
	}

	return nil
}

// end region

// region Vozvrat
func (s *Services) updateStocksAfterVozvratFinished(vozvratId string, storeId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	var details []struct {
		ProductId    string `json:"product_id"`
		UnitQuantity int    `json:"unit_quantity"`
	}
	query := `
	SELECT
		td.product_id,
		td.accepted_count * p.unit_per_pack AS unit_quantity
	FROM transfer_details td
	JOIN products p ON td.product_id = p.id
	WHERE td.transfer_id = ?;
	`
	err := s.db.WithContext(ctx).Raw(query, vozvratId).Scan(&details).Error
	if err != nil {
		return fmt.Errorf("failed to get confirmed vozvrat details: %w", err)
	}

	if len(details) == 0 {
		return nil
	}

	const batchSize = 500

	for i := 0; i < len(details); i += batchSize {
		end := i + batchSize
		if end > len(details) {
			end = len(details)
		}
		batch := details[i:end]

		valueStrings := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*2)

		for _, detail := range batch {
			valueStrings = append(valueStrings, "(?::uuid, ?::bigint)")
			args = append(args, detail.ProductId, detail.UnitQuantity)
		}

		updateQuery := fmt.Sprintf(`
		UPDATE store_product_stocks sps
		SET
			unit_quantity = sps.unit_quantity - v.qty,
			updated_at    = NOW()
		FROM (VALUES %s) AS v(product_id, qty)
		WHERE sps.store_id = ?
		  AND sps.product_id = v.product_id;
		`, strings.Join(valueStrings, ","))

		args = append(args, storeId)

		if err := s.db.WithContext(ctx).Exec(updateQuery, args...).Error; err != nil {
			return fmt.Errorf("failed to batch update store_product_stocks after vozvrat: %w", err)
		}
	}

	return nil
}
