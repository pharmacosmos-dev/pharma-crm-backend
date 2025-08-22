package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pharma-crm-backend/domain"
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

	for _, store := range stores {
		fmt.Printf("Sending report for %s...\n", store.Name)
		if err = s.sendReportTo1C(&store, sendDate); err != nil {
			log.Printf("Failed to send report for %s: %v\n", store.Name, err)
			// You can choose to retry here or log for manual retry
			continue
		}

		fmt.Printf("Successfully sent report for %s\n", store.Name)
		time.Sleep(5 * time.Second)
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
	    sp.expire_date::date,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN ci.quantity + (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.quantity + (ci.unit_quantity::NUMERIC / p.unit_per_pack)) * -1
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
	                            THEN (sp.vat_price * ci.quantity) + ((sp.vat_price / p.unit_per_pack) * ci.unit_quantity)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN -1 * ((sp.vat_price * ci.quantity) + ((sp.vat_price / p.unit_per_pack) * ci.unit_quantity))
	                        ELSE 0
	                        END
	            )
	        , 2) AS vat_sum,
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN (id.retail_price * ci.quantity) + ((id.retail_price / p.unit_per_pack) * ci.unit_quantity)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN -1 * ((id.retail_price * ci.quantity) + ((id.retail_price / p.unit_per_pack) * ci.unit_quantity))
	                        ELSE 0
	                        END
	            )
	        , 2) AS sum,
	    SUM(
	            CASE
	                WHEN s.sale_type = 'SALE'
	                    THEN ci.total_price
	                WHEN s.sale_type = 'RETURN'
	                    THEN -1 * ci.total_price
	                ELSE 0
	                END
	    ) AS sum_vat
	FROM sales s
	         LEFT JOIN sales s_return
	                   ON s_return.parent_id = s.id
	                       AND s_return.sale_type = 'RETURN'
	                       AND s_return.status = 'completed'
	         JOIN cart_items ci ON s.id = ci.sale_id
	         JOIN store_products sp ON ci.store_product_id = sp.id
	         JOIN products p ON sp.product_id = p.id
	         LEFT JOIN producers pr ON p.producer_id = pr.id
	         LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	  AND s.status = 'completed'
	  AND (s.completed_at + interval '5 hours')::date
	    BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id
	HAVING
	    ROUND(
	            SUM(
	                    CASE
	                        WHEN s.sale_type = 'SALE'
	                            THEN ci.quantity + (ci.unit_quantity::NUMERIC / p.unit_per_pack)
	                        WHEN s.sale_type = 'RETURN'
	                            THEN (ci.quantity + (ci.unit_quantity::NUMERIC / p.unit_per_pack)) * -1
	                        ELSE 0
	                        END
	            )::NUMERIC
	        , 4) != 0;
	`
	// complete get expense product list
	err = s.db.Raw(expenseProductQuery, store.Id, date, date).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense products: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}

	// send fakt to 1C
	err = s.DoRequest(context.Background(), expenseData, "/rasxod")
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
		sp.expire_date::date,
		ROUND(SUM(ci.quantity)::NUMERIC + (SUM(ci.unit_quantity)::NUMERIC / p.unit_per_pack), 4) AS quantity,
		sp.supply_price AS supply_price_vat,
		sp.retail_price AS retail_price_vat,
		id.supply_price,
		id.retail_price,
		sp.vat,
		ROUND((sp.vat_price*SUM(ci.quantity)) + ((sp.vat_price/p.unit_per_pack)*SUM(ci.unit_quantity)), 2) AS vat_sum,
		ROUND((id.retail_price*SUM(ci.quantity)) + ((id.retail_price/p.unit_per_pack)*SUM(ci.unit_quantity)), 2) AS sum,
		SUM(ci.total_price) AS sum_vat
	FROM sales s
	LEFT JOIN sales s_return ON s_return.parent_id = s.id AND s_return.sale_type = 'RETURN' AND s_return.status = 'completed'
	JOIN cart_items ci ON s.id = ci.sale_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	AND s.status = 'completed' AND s.sale_type = 'SALE' AND 
		s_return.id IS NULL AND (s.completed_at+interval '5 hours')::date BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id;
	`
	// complete get expense product list
	err = s.db.Raw(expenseProductQuery, store.Id, date, date).Scan(&expenseData.Товары).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense products: %v", err)
		return err
	}
	// check expense product length
	if len(expenseData.Товары) < 1 {
		return nil
	}

	// send fakt to 1C
	err = s.DoRequest(context.Background(), expenseData, "/rasxod")
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
