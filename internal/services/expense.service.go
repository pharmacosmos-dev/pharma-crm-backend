package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/xuri/excelize/v2"
)

var mu sync.Mutex

// send expense products to 1C
func (s *Services) SendExpenseTo1C(sendDate string) error {
	mu.Lock()
	defer mu.Unlock()

	var stores []domain.Store
	// get store list
	err := s.db.Raw(`
	SELECT
		DISTINCT s.*
	FROM stores s
	JOIN sales sl ON s.id = sl.store_id
	WHERE DATE(sl.completed_at) = ?;
	`, sendDate).Scan(&stores).Error
	if err != nil {
		s.log.Warn("ERROR on getting store list: %v", err)
		return errors.New("error on getting store")
	}
	fmt.Println("--->>> ", len(stores))
	for _, store := range stores {
		fmt.Printf("Sending report for %s...\n", store.Name)
		if err = s.sendReportTo1C(&store, sendDate); err != nil {
			log.Printf("Failed to send report for %s: %v\n", store.Name, err)
			// You can choose to retry here or log for manual retry
			continue
		}

		fmt.Printf("Successfully sent report for %s\n", store.Name)
		time.Sleep(10 * time.Second)
	}

	return nil
}

// send expense products to 1C with dock number
// This function is used to send expense reports with a specific document number
func (s *Services) SendExpenseWithNumberTo1C(sendDate, storeID, dockNumber string) error {
	var store domain.Store
	// get store info
	err := s.db.First(&store, "id = ?", storeID).Error
	if err != nil {
		s.log.Warn("ERROR on getting store list: %v", err)
		return errors.New("error on getting store")
	}
	// send expense with dock number
	err = s.sendReportWithNumberTo1C(&store, sendDate, dockNumber)
	if err != nil {
		log.Printf("Failed to send report for %s: %v\n", store.Name, err)
		return errors.New("error on sending report with number")
	}
	fmt.Printf("Successfully sent report for %s\n", store.Name)
	return nil
}

func (s *Services) CreateNewExpense(storeID string, dockNumer, sendAt string) error {
	query := `INSERT INTO shift_expenses(store_id, docs_number, sent_at) VALUES(?, ?, ?)`
	err := s.db.Exec(query, storeID, dockNumer, sendAt).Error
	if err != nil {
		s.log.Warn("ERROR on creating shift_expenses: %v", err)
		return err
	}

	return nil
}

func (s *Services) UpdateExpenseStatusByDocNumber(status int, docsNumber string) error {
	query := `UPDATE shift_expenses SET status = ? WHERE docs_number = ?`
	err := s.db.Exec(query, status, docsNumber).Error
	if err != nil {
		s.log.Warn("ERROR on updating shift_expenses status: %v", err)
		return err
	}
	return nil
}

func (s *Services) CheckShiftExpense(sendDate, storeID string) bool {
	query := `SELECT COUNT(*) FROM shift_expenses WHERE store_id = ? AND sent_at = ?`
	var count int
	err := s.db.Raw(query, storeID, sendDate).Scan(&count).Error
	if err != nil {
		s.log.Warn("ERROR on checking shift expense: %v", err)
		return false
	}
	return count > 0
}

// SendReportsSequentially sends today's reports for each store
func (s *Services) SendReportsSequentially() {
	mu.Lock()
	defer mu.Unlock()

	var stores []domain.Store
	// get store list
	err := s.db.Find(&stores).Error
	if err != nil {
		s.log.Warn("ERROR on getting store list: %v", err)
		return
	}
	now := time.Now().UTC()

	for _, store := range stores {
		fmt.Printf("Sending report for %s...\n", store.Name)
		if err = s.sendReportTo1C(&store, now.Format(time.DateOnly)); err != nil {
			log.Printf("Failed to send report for %s: %v\n", store.Name, err)
			// You can choose to retry here or log for manual retry
			continue
		}

		fmt.Printf("Successfully sent report for %s\n", store.Name)
		time.Sleep(5 * time.Second)
	}

}

// send expense products to 1C
func (s *Services) sendReportTo1C(store *domain.Store, date string) error {
	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name

	// get expense docs number
	docNumberQuery := `
	SELECT 
		'NP-' || LPAD(store_code::TEXT, 5, '0') || '-' || TO_CHAR(?::DATE, 'YYYYMMDD') || '0000' AS nomer_dok
	FROM stores
	WHERE id = ?;`
	err := s.db.Raw(docNumberQuery, date, store.Id).Scan(&expenseData.Document.NumberDok).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense docs number: %v", err)
		return err
	}

	dokTime, err := time.Parse(time.DateOnly, date)
	if err != nil {
		s.log.Warn("ERROR on parsing date: %v", err)
		return errors.New("error on parsing date")
	}

	expenseData.Document.DocumentDate = dokTime.Format(time.RFC3339) // set document date
	// create new shift expense
	err = s.CreateNewExpense(store.Id, expenseData.Document.NumberDok, expenseData.Document.DocumentDate)
	if err != nil {
		s.log.Warn("ERROR on creating shift expense: %v", err)
	}

	// get expense products query
	expenseProductQuery := `
	SELECT
	    sp.product_id,
	    p.material_code,
	    p.name,
	    p.barcode,
	    p.mxik AS ikpu,
	    COALESCE(pr.name, '') AS manufacturer,
	    COALESCE(sp.serial_number, '') AS product_series_number,
	    sp.expire_date,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
	                        ELSE 0
	                        END
	            )::NUMERIC
	        , 4) AS quantity,
        sp.supply_price AS supply_price_vat,
        sp.retail_price AS retail_price_vat,
	    id.supply_price,
	    id.retail_price,
	    sp.vat,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (sp.vat_price / p.unit_per_pack) * ci.unit_quantity
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (-1) * ((sp.vat_price / p.unit_per_pack) * ci.unit_quantity)
	                        ELSE 0
	                        END
	            )
	        , 2) AS vat_sum,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (id.retail_price / p.unit_per_pack) * ci.unit_quantity
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (-1) *  ((id.retail_price / p.unit_per_pack) * ci.unit_quantity)
	                        ELSE 0
	                        END
	            )
	        , 2) AS sum,
	    SUM(
	            CASE
	                WHEN s.sale_type = 'SALE'
	                    THEN ci.total_price
	                WHEN s.sale_type = 'RETURN'
	                    THEN (-1) * ci.total_price
	                ELSE 0
	                END
	    ) AS sum_vat
	FROM sales s
	         LEFT JOIN sales s_return
	                   ON s_return.parent_id = s.id
	                       AND s_return.sale_type = 'RETURN'
	                       AND s_return.stage IN (9, 11)
	         JOIN cart_items ci ON s.id = ci.sale_id
	         JOIN store_products sp ON ci.store_product_id = sp.id
	         JOIN products p ON sp.product_id = p.id
	         LEFT JOIN producers pr ON p.producer_id = pr.id
	         LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	  AND s.stage IN (9, 11)
	  AND s.completed_at BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id
	HAVING
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
	                        ELSE 0
	                        END
	            )::NUMERIC
	        , 4) != 0;
	`
	startTime := dokTime.Add(-5 * time.Hour)
	endTime := dokTime.Add(19 * time.Hour)

	// complete get expense product list
	err = s.db.Raw(
		expenseProductQuery,
		store.Id,
		startTime,
		endTime,
	).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Errorf("could not get expense products: %v", err)
		return domain.InternalServerError
	}

	// get total discount
	discountQuery := `
    SELECT
        COALESCE(SUM(s.total_discount), 0) AS discount_sum
    FROM sales s
    WHERE s.store_id = ?
      AND s.stage = 9
      AND s.completed_at BETWEEN ? AND ?;
`

	err = s.db.Raw(discountQuery, store.Id, startTime, endTime).Scan(&expenseData.Document.DiscountSum).Error
	if err != nil {
		s.log.Errorf("could not get discount sum: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}

	// send fakt to 1C
	err = s.DoRequestOnec(context.Background(), expenseData, "/rasxod")
	if err != nil {
		s.log.Errorf("could not send rasxod request: %v", err)
		return err
	}
	// update expense status to 1 after successfully sent
	err = s.UpdateExpenseStatusByDocNumber(1, expenseData.Document.NumberDok)
	if err != nil {
		return err
	}
	return nil
}

// send expense with docs number products to 1C
func (s *Services) sendReportWithNumberTo1C(store *domain.Store, date, dockNumber string) error {
	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name

	dokTime, err := time.Parse(time.DateOnly, date)
	if err != nil {
		s.log.Warn("ERROR on parsing date: %v", err)
		return errors.New("error on parsing date")
	}

	expenseData.Document.DocumentDate = dokTime.Format(time.RFC3339) // set document date
	expenseData.Document.NumberDok = dockNumber

	// get expense products query
	expenseProductQuery := `
	SELECT
	    sp.product_id,
	    p.material_code,
	    p.name,
	    p.barcode,
	    p.mxik AS ikpu,
	    COALESCE(pr.name, '') AS manufacturer,
	    COALESCE(sp.serial_number, '') AS product_series_number,
	    sp.expire_date,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
	                        ELSE 0
	                        END
	            )::NUMERIC
	        , 4) AS quantity,
        sp.supply_price AS supply_price_vat,
        sp.retail_price AS retail_price_vat,
	    id.supply_price,
	    id.retail_price,
	    sp.vat,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (sp.vat_price / p.unit_per_pack) * ci.unit_quantity
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (-1) * ((sp.vat_price / p.unit_per_pack) * ci.unit_quantity)
	                        ELSE 0
	                        END
	            )
	        , 2) AS vat_sum,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (id.retail_price / p.unit_per_pack) * ci.unit_quantity
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (-1) *  ((id.retail_price / p.unit_per_pack) * ci.unit_quantity)
	                        ELSE 0
	                        END
	            )
	        , 2) AS sum,
	    SUM(
	            CASE
	                WHEN s.sale_type = 'SALE'
	                    THEN ci.total_price
	                WHEN s.sale_type = 'RETURN'
	                    THEN (-1) * ci.total_price
	                ELSE 0
	                END
	    ) AS sum_vat
	FROM sales s
	         LEFT JOIN sales s_return
	                   ON s_return.parent_id = s.id
	                       AND s_return.sale_type = 'RETURN'
	                       AND s_return.stage IN (9, 11)
	         JOIN cart_items ci ON s.id = ci.sale_id
	         JOIN store_products sp ON ci.store_product_id = sp.id
	         JOIN products p ON sp.product_id = p.id
	         LEFT JOIN producers pr ON p.producer_id = pr.id
	         LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	  AND s.stage IN (9, 11)
	  AND s.completed_at BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id
	HAVING
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
	                        ELSE 0
	                        END
	            )::NUMERIC
	        , 4) != 0;
	`
	startTime := dokTime.Add(-5 * time.Hour)
	endTime := dokTime.Add(19 * time.Hour)

	// complete get expense product list
	err = s.db.Raw(
		expenseProductQuery,
		store.Id,
		startTime,
		endTime,
	).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Errorf("could not get expense products: %v", err)
		return domain.InternalServerError
	}

	// get total discount
	discountQuery := `
    SELECT
        COALESCE(SUM(s.total_discount), 0) AS discount_sum
    FROM sales s
    WHERE s.store_id = ?
      AND s.stage = 9
      AND s.completed_at BETWEEN ? AND ?;
`

	err = s.db.Raw(discountQuery, store.Id, startTime, endTime).Scan(&expenseData.Document.DiscountSum).Error
	if err != nil {
		s.log.Errorf("could not get discount sum: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}
	t, _ := json.Marshal(expenseData)

	fmt.Println("--->> ", string(t))
	// send fakt to 1C
	err = s.DoRequestOnec(context.Background(), expenseData, "/rasxod")
	if err != nil {
		s.log.Warn("ERROR on send rasxod request: %v", err)
		return err
	}
	// update expense status to 1 after successfully sent
	err = s.UpdateExpenseStatusByDocNumber(1, expenseData.Document.NumberDok)
	if err != nil {
		return err
	}
	return nil
}

// SendBacklogReportsSequentially sends reports for each store for the backlog period
func (s *Services) SendBacklogReportsSequentially(start, end time.Time) {
	mu.Lock()
	defer mu.Unlock()

	var stores []domain.Store
	// get store list
	err := s.db.Find(&stores).Error
	if err != nil {
		s.log.Warn("ERROR on getting store list: %v", err)
		return
	}

	for currentDate := start; currentDate.Before(end) || currentDate.Equal(end); currentDate = currentDate.AddDate(0, 0, 1) {
		fmt.Printf("Sending reports for date: %s\n", currentDate.Format("2006-01-02"))

		for _, store := range stores {
			fmt.Printf("Sending report for store %s on %s...\n", store.Name, currentDate.Format("2006-01-02"))
			if err = s.sendReportTo1C(&store, currentDate.Format("2006-01-02")); err != nil {
				log.Printf("Failed to send report for %s on %s: %v\n", store.Name, currentDate.Format("2006-01-02"), err)
				continue
			}
			fmt.Printf("Successfully sent report for %s on %s\n", store.Name, currentDate.Format("2006-01-02"))
			time.Sleep(5 * time.Second) // Wait 5 seconds before next store
		}
		fmt.Printf("Completed reports for %s. Waiting 10 minutes...\n", currentDate.Format("2006-01-02"))
		time.Sleep(10 * time.Minute) // Wait 10 minutes before next day
	}

}

// Excel fayldan olib yuborish
func (s *Services) SendExpenseTo1CFromExcel(filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot open excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("List")
	if err != nil {
		return fmt.Errorf("cannot read sheet: %v", err)
	}

	type key struct {
		StoreCode string
		Date      string
	}
	unique := make(map[key]struct{}) // set

	for i, row := range rows {
		if i == 0 { // skip header
			continue
		}
		if len(row) < 3 {
			continue
		}
		//if len(row) < 15 {
		//	continue
		//} else {
		//	r := strings.TrimSpace(row[17])
		//	if r == "" || r == "0" || r == "0.0" {
		//		continue
		//	}
		//}

		storeCode := row[0] // ID
		date := row[2]      // Дата (2025-03-10T00:00:00Z)
		parsedDate, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return fmt.Errorf("invalid date format at row %d: %v", i+1, err)
		}
		sendDate := parsedDate.Format("2006-01-02")
		k := key{StoreCode: storeCode, Date: sendDate}
		unique[k] = struct{}{} // faqat unikallar qoladi
	}

	// Endi faqat unikal kombinatsiyalar bo‘yicha yuboramiz
	for k := range unique {
		var store domain.Store
		err = s.db.Raw("SELECT * FROM stores WHERE store_code = ?", k.StoreCode).Scan(&store).Error
		if err != nil {
			return fmt.Errorf("cannot find store with code %s: %v", k.StoreCode, err)
		}
		fmt.Printf("Sending report for store=%s date=%s...\n", store.Name, k.Date)
		if err = s.sendReportTo1C(&store, k.Date); err != nil {
			fmt.Printf("Failed for store=%s date=%s: %v\n", store.Name, k.Date, err)
			continue
		}
		fmt.Printf("Successfully sent report for store=%s date=%s\n", store.Name, k.Date)
		time.Sleep(10 * time.Second)
	}

	return nil
}

func (s *Services) SendChequesTemporary(sendDate string) {

	mu.Lock()
	defer mu.Unlock()

	var stores []domain.Store
	// get store list
	err := s.db.Find(&stores).Error
	if err != nil {
		s.log.Errorf("could not get store list: %v", err)
		return
	}

	for _, store := range stores {
		fmt.Printf("Sending report for %s...\n", store.Name)
		if err = s.sendReportToTemporary(&store, sendDate); err != nil {
			log.Printf("Failed to send report for %s: %v\n", store.Name, err)
			// You can choose to retry here or log for manual retry
			continue
		}

		fmt.Printf("Successfully sent report for %s\n", store.Name)
		time.Sleep(10 * time.Second)
	}
}

// send expense products to 1C
func (s *Services) sendReportToTemporary(store *domain.Store, date string) error {
	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name

	// get expense docs number
	docNumberQuery := `
	SELECT 
		'NP-' || LPAD(store_code::TEXT, 5, '0') || '-' || TO_CHAR(?::DATE, 'YYYYMMDD') || '0000' AS nomer_dok
	FROM stores
	WHERE id = ?;`
	err := s.db.Raw(docNumberQuery, date, store.Id).Scan(&expenseData.Document.NumberDok).Error
	if err != nil {
		s.log.Errorf("could not get expense docs number: %v", err)
		return err
	}

	dokTime, err := time.Parse(time.DateOnly, date)
	if err != nil {
		s.log.Errorf("could not parse date: %v", err)
		return domain.InvalidTimeFormatError
	}

	expenseData.Document.DocumentDate = dokTime.Format(time.RFC3339) // set document date
	// create new shift expense
	err = s.CreateNewExpense(store.Id, expenseData.Document.NumberDok, expenseData.Document.DocumentDate)
	if err != nil {
		s.log.Errorf("could not create shift expense: %v", err)
	}

	// get expense products query
	expenseProductQuery := `
	WITH valid_sales AS (
		SELECT
			s.id AS sale_id
		FROM sales s
		JOIN cart_items ci ON ci.sale_id = s.id
		WHERE s.stage IN (9, 11) AND
			(s.completed_at BETWEEN ? AND ?) AND
			s.store_id = ?
		GROUP BY s.id, s.total_amount, s.total_discount
		HAVING ROUND(SUM(ci.total_price), 2) = ROUND((s.total_amount + s.total_discount), 2)
	)
	SELECT
		sp.product_id,
		p.material_code,
		p.name,
		p.barcode,
		p.mxik AS ikpu,
		COALESCE(pr.name, '') AS manufacturer,
		COALESCE(sp.serial_number, '') AS product_series_number,
		sp.expire_date,
		ROUND(
			SUM(
				CASE
					WHEN s.sale_type = 'SALE'
						THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
					WHEN s.sale_type = 'RETURN'
						THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
					ELSE 0
				END
			)::NUMERIC, 4
		) AS quantity,
		sp.supply_price AS supply_price_vat,
		sp.retail_price AS retail_price_vat,
		id.supply_price,
		id.retail_price,
		sp.vat,
		ROUND(
			SUM(
				CASE
					WHEN s.sale_type = 'SALE'
						THEN (sp.vat_price / p.unit_per_pack) * ci.unit_quantity
					WHEN s.sale_type = 'RETURN'
						THEN (-1) * ((sp.vat_price / p.unit_per_pack) * ci.unit_quantity)
					ELSE 0
				END
			), 2
		) AS vat_sum,
		ROUND(
			SUM(
				CASE
					WHEN s.sale_type = 'SALE'
						THEN (id.retail_price / p.unit_per_pack) * ci.unit_quantity
					WHEN s.sale_type = 'RETURN'
						THEN (-1) * ((id.retail_price / p.unit_per_pack) * ci.unit_quantity)
					ELSE 0
				END
			), 2
		) AS sum,
		SUM(
			CASE
				WHEN s.sale_type = 'SALE' THEN ci.total_price
				WHEN s.sale_type = 'RETURN' THEN (-1) * ci.total_price
				ELSE 0
			END
		) AS sum_vat
	FROM sales s
	JOIN valid_sales vs ON vs.sale_id = s.id
	LEFT JOIN sales s_return
		ON s_return.parent_id = s.id
		AND s_return.sale_type = 'RETURN'
		AND s_return.stage IN (9, 11)
	JOIN cart_items ci ON s.id = ci.sale_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	AND s.stage IN (9, 11)
	AND s.completed_at BETWEEN ? AND ?
	GROUP BY p.id, pr.id, sp.id, id.id
	HAVING ROUND(
		SUM(
			CASE
				WHEN s.sale_type = 'SALE'
					THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack)
				WHEN s.sale_type = 'RETURN'
					THEN (ci.unit_quantity::NUMERIC / p.unit_per_pack) * (-1)
				ELSE 0
			END
		)::NUMERIC, 4
	) != 0;
	`
	startTime := dokTime.Add(-5 * time.Hour)
	endTime := dokTime.Add(19 * time.Hour)

	// complete get expense product list
	err = s.db.Raw(
		expenseProductQuery,
		startTime,
		endTime,
		store.Id,
		store.Id,
		startTime,
		endTime,
	).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Errorf("could not get expense products: %v", err)
		return err
	}

	// get total discount
	discountQuery := `
    SELECT
        COALESCE(SUM(s.total_discount), 0) AS discount_sum
    FROM sales s
    WHERE s.store_id = ?
      AND s.stage = 9
      AND s.completed_at BETWEEN ? AND ?;
`

	err = s.db.Raw(
		discountQuery,
		store.Id,
		startTime,
		endTime,
	).Scan(&expenseData.Document.DiscountSum).Error
	if err != nil {
		s.log.Errorf("could not get discount sum: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}

	// send fakt to 1C
	err = s.DoRequestOnec(context.Background(), expenseData, constants.OnecPathRasxod)
	if err != nil {
		s.log.Errorf("could not send rasxod request: %v", err)
		return err
	}
	// update expense status to 1 after successfully sent
	err = s.UpdateExpenseStatusByDocNumber(1, expenseData.Document.NumberDok)
	if err != nil {
		return err
	}
	return nil
}
