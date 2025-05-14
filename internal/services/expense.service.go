package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pharma-crm-backend/domain"
)

var mu sync.Mutex

// send expense products to 1C
func (s *Services) SendExpenseTo1C(sendDate string, storeID string) error {
	// get cashbox operation info with store
	var store domain.Store
	err := s.db.First(&store, "id = ?", storeID).Error
	if err != nil {
		s.log.Warn("ERROR on getting store info with operation: %v", err)
		return err
	}
	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name
	// get expense docs number
	docNumberQuery := `
	SELECT 'NP-' || LPAD(store_code::TEXT, 5, '0') || '-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MI') AS docs_number
	FROM stores
	WHERE id = ?;`
	err = s.db.Raw(docNumberQuery, storeID).Scan(&expenseData.Document.NumberDok).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense docs number: %v", err)
		return err
	}

	// create new shift expense
	err = s.CreateNewExpense(storeID, expenseData.Document.NumberDok, sendDate)
	if err != nil {
		s.log.Warn("ERROR on creating shift expense: %v", err)
		return err
	}

	// get expense dok time with adding 5 hours
	expenseData.Document.DocumentDate = sendDate
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
	JOIN cart_items ci ON s.id = ci.sale_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	AND s.status = 'completed' AND s.sale_type = 'SALE' AND (s.completed_at+interval '5 hours')::date BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id
	`
	// complete get expense product list
	err = s.db.Raw(expenseProductQuery, storeID, sendDate, sendDate).Scan(&expenseData.Товары).Error
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

func (s *Services) CreateNewExpense(storeID string, docsNumber string, sendDate string) error {
	query := `INSERT INTO shift_expenses(store_id, docs_number, sent_at) VALUES(?, ?, ?)`
	err := s.db.Exec(query, storeID, docsNumber, sendDate).Error
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
	now := time.Now()

	for _, store := range stores {
		fmt.Printf("Sending report for %s...\n", store.Name)
		if err = s.sendReportTo1C(&store, now.Format(time.DateOnly), now); err != nil {
			log.Printf("Failed to send report for %s: %v\n", store.Name, err)
			// You can choose to retry here or log for manual retry
			continue
		}

		fmt.Printf("Successfully sent report for %s\n", store.Name)
		time.Sleep(5 * time.Second)
	}

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
			if err = s.sendReportTo1C(&store, currentDate.Format("2006-01-02"), currentDate); err != nil {
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

// send expense products to 1C
func (s *Services) sendReportTo1C(store *domain.Store, date string, docDate time.Time) error {

	var expenseData domain.SendExpense
	expenseData.Store.StoreCode = store.StoreCode
	expenseData.Store.Name = store.Name
	// get expense docs number
	docNumberQuery := `
	SELECT 'NP-' || LPAD(store_code::TEXT, 5, '0') || '-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MI') AS docs_number
	FROM stores
	WHERE id = ?;`
	err := s.db.Raw(docNumberQuery, store.Id).Scan(&expenseData.Document.NumberDok).Error
	if err != nil {
		s.log.Warn("ERROR on getting expense docs number: %v", err)
		return err
	}

	// create new shift expense
	err = s.CreateNewExpense(store.Id, expenseData.Document.NumberDok, date)
	if err != nil {
		s.log.Warn("ERROR on creating shift expense: %v", err)
		return err
	}
	// "2006-01-01T00:00:00Z"
	// get expense dok time with adding 5 hours
	expenseData.Document.DocumentDate = docDate.Add(time.Minute * 1430).Format("2006-01-02T15:04:05Z07:00")

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
	JOIN cart_items ci ON s.id = ci.sale_id
	JOIN store_products sp ON ci.store_product_id = sp.id
	JOIN products p ON sp.product_id = p.id
	LEFT JOIN producers pr ON p.producer_id = pr.id
	LEFT JOIN import_details id ON sp.import_detail_id = id.id
	WHERE s.store_id = ?
	AND s.status = 'completed' AND s.sale_type = 'SALE' AND (s.completed_at + interval '5 hours')::date BETWEEN ? AND ?
	GROUP BY
		p.id, pr.id, sp.id, id.id
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
