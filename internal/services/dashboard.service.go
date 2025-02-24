package services

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/helper"
)

// get dashboard data
func (s *Storage) DashboardTotalCountStats(storeId string, startDate, endDate string) (*domain.TotalCountStats, error) {
	// declarations
	var (
		res    domain.TotalCountStats
		params = make(map[string]interface{})
	)

	// queries
	var (
		arr     []interface{}
		querys  = `SELECT COUNT(*) AS total_sale_count, SUM(total_amount) AS total_sale_amount FROM sales`
		queryp  = `SELECT COUNT(pack_quantity) AS total_product_count FROM store_products`
		filters = " WHERE 1=1"
		filterp = " WHERE expire_date::date >= current_date "
	)
	fmt.Println("--->>> ", storeId, startDate, endDate)
	// if store id is not empty
	if storeId != "" {
		params["store_id"] = storeId
		filters += " AND store_id = :store_id"
		filterp += " AND store_id = :store_id"
	}
	// if start date is not empty
	if startDate != "" && endDate == "" {
		params["start_date"] = startDate
		filters += " AND completed_at >= :start_date"
		filterp += " AND created_at >= :start_date"
	}
	// if end date is not empty
	if startDate != "" && endDate != "" {
		params["start_date"] = startDate
		params["end_date"] = endDate
		filters += " AND completed_at >= :start_date AND completed_at <= :end_date"
		filterp += " AND created_at >= :start_date AND created_at <= :end_date"
	}

	// get total sale count and amount
	var q = querys + filters
	q, arr = helper.ReplaceQueryParams(q, params)
	err := s.db.Raw(q, arr...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	// get total product count
	var qp = queryp + filterp
	qp, arr = helper.ReplaceQueryParams(qp, params)
	err = s.db.Debug().Raw(qp, arr...).Scan(&res).Error
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
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
