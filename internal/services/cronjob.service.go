package services

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pharma-crm-backend/domain"
)

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

func (s *Services) performSendOsonApteka() {
	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		if err := s.SendRemainingQuantityToOsonApteka(); err != nil {
			s.log.Errorf("error sending remaining quantity to OsonApteka: %v", err)
		}
	}
}

func (s *Services) SendRemainingQuantityToOsonApteka() error {
	if s.cfg.Integration.OsonAptekaApiUrl == "test" {
		return nil
	}

	var query = `
WITH valid_stores AS (
    SELECT
        id,
        oson_apteka_login,
        oson_apteka_parol
    FROM
        stores
    WHERE
        oson_apteka_login IS NOT NULL AND oson_apteka_login != '' AND
        oson_apteka_parol IS NOT NULL AND oson_apteka_parol != ''
)
SELECT
    json_build_object(
            'store', vs.oson_apteka_login,
            'code', vs.oson_apteka_parol,
            'remain_count', COUNT(*),  -- Simplified: all rows already filtered
            'programm', 13,
            'drugs', json_agg(
                    json_build_object(
                            'id', sp.id,
                            'drug_id', p.unit_code,
                            'barcode', p.barcode,
                            'name', p.name,
                            'manufacturer', pr.name,
                            'price', sp.retail_price,
                            'qty', ROUND(sp.unit_quantity::numeric / p.unit_per_pack, 2),
                            'expiry_date', sp.expire_date
                    )
                     )
    ) AS result
FROM
    valid_stores AS vs
        INNER JOIN  -- Changed from LEFT JOIN since we need products
        store_products AS sp ON vs.id = sp.store_id AND sp.unit_quantity > 0
        INNER JOIN
    products AS p ON p.id = sp.product_id
        LEFT JOIN  -- Keep LEFT JOIN if producer can be null
        producers AS pr ON pr.id = p.producer_id
GROUP BY
    vs.id, vs.oson_apteka_login, vs.oson_apteka_parol
	`

	// Scan and process in one pass
	var stores []string
	if err := s.db.Raw(query).Scan(&stores).Error; err != nil {
		return fmt.Errorf("failed to fetch remaining quantities for OsonApteka: %w", err)
	}

	for _, jsonStr := range stores {
		var store domain.OsonAptekaRequest
		if err := json.Unmarshal([]byte(jsonStr), &store); err != nil {
			return fmt.Errorf("failed to unmarshal store data: %w", err)
		}

		// Process immediately - no second loop needed
		jsonData, err := json.MarshalIndent(store, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal store data: %w", err)
		}

		// 2️⃣ Compress via GZIP
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)
		gzipWriter.Write(jsonData)
		gzipWriter.Close()
		compressedData := buf.Bytes()

		// // 3️⃣ Encode []byte to Base64 so it can be in JSON
		jsonFromBytes, err := json.Marshal(compressedData)
		if err != nil {
			log.Fatal(err)
		}

		// Send to API
		req, err := http.NewRequest("POST", s.cfg.Integration.OsonAptekaApiUrl, bytes.NewBuffer(jsonFromBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// print response
		bodyByte, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		response := domain.OsonAptekaRemainingQuantityResponse{}
		if err := json.Unmarshal(bodyByte, &response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// check response
		if !response.Succeeded {
			return fmt.Errorf("oson apteka returned error: %+v", response)
		}

		s.log.Info("Sent remainings to oson apteka successfully")
	}

	return nil
}
