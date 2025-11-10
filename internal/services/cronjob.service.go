package services

import "fmt"

func (s *Services) syncUnitCodes() error {
	// store_products update
	if err := s.db.Exec(`
		WITH correct_units AS (
			SELECT
				mxik,
				array_agg(DISTINCT unit_code) AS correct_unit_codes
			FROM tax_products
			GROUP BY mxik
		)
		UPDATE store_products sp
		SET unit_code = cu.correct_unit_codes[1], last_updated_at = NOW()
		FROM correct_units cu
		WHERE sp.mxik = cu.mxik
		  AND (sp.unit_code IS NULL OR sp.unit_code <> ALL (cu.correct_unit_codes));
    `).Error; err != nil {
		return fmt.Errorf("failed to update store_products: %w", err)
	}

	// products update
	if err := s.db.Exec(`
		WITH correct_units AS (
			SELECT
				mxik,
				array_agg(DISTINCT unit_code) AS correct_unit_codes
			FROM tax_products
			GROUP BY mxik
		)
		UPDATE products p
		SET unit_code = cu.correct_unit_codes[1], last_updated_at = NOW()
		FROM correct_units cu
		WHERE p.mxik = cu.mxik
		  AND (p.unit_code IS NULL OR p.unit_code <> ALL (cu.correct_unit_codes));

    `).Error; err != nil {
		return fmt.Errorf("failed to update products: %w", err)
	}
	return nil
}

func (s *Services) SendRemainingQuantityToOsonApteka() error {
	return nil
}
