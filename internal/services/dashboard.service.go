package services

import (
	"github.com/pharma-crm-backend/domain"
)

// get dashboard data
func (s *Storage) DashboardTotalCountStats(storeId string, startDate, endDate string) (*domain.TotalCountStats, error) {
	// declarations
	var (
		res       domain.TotalCountStats
		totalSale struct {
			TotalSaleCount  float64 `gorm:"total_sale_count" json:"total_sale_count"`
			TotalSaleAmount float64 `gorm:"total_sale_amount" json:"total_sale_amount"`
		}
		productCount int64
	)
	// queries
	var (
		args    []interface{}
		querys  = `SELECT COUNT(*) AS total_sale_count, SUM(total_amount) AS total_sale_amount FROM sales`
		queryp  = `SELECT SUM(pack_quantity) AS total_product_count FROM store_products`
		filters = " WHERE 1=1"
		filterp = " WHERE expire_date::date >= current_date "
	)
	// if store id is not empty
	if storeId != "" {
		filters += " AND store_id = ?"
		filterp += " AND store_id = ?"
		args = append(args, storeId)
	}

	// if start date is not empty
	if startDate != "" && endDate == "" {
		filters += " AND completed_at >= ?"
		filterp += " AND created_at >= ?"
		args = append(args, startDate)
	}

	// if end date is not empty
	if startDate != "" && endDate != "" {
		filters += " AND completed_at >= ? AND completed_at <= ?"
		filterp += " AND created_at >= ? AND created_at <= ?"
		args = append(args, startDate, endDate)
	}

	// get total sale count and amount
	var q = querys + filters
	err := s.db.Raw(q, args...).Scan(&totalSale).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	// get total product count
	var qp = queryp + filterp
	err = s.db.Debug().Raw(qp, args...).Scan(&productCount).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}

	res.TotalSaleCount = totalSale.TotalSaleCount
	res.TotalSaleAmount = totalSale.TotalSaleAmount
	res.TotalProductCount = productCount

	// get store count by checking store_id is not emply
	if storeId != "" {
		res.TotalStoreCount = 1
	} else {
		err = s.db.Model(&domain.Store{}).Count(&res.TotalStoreCount).Error
		if err != nil {
			s.log.Error(err)
			return nil, err
		}
	}

	return &res, nil
}
