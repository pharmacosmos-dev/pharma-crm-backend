package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// check field employee
func (s *Services) CheckFieldEmployee(field, value string) (bool, error) {
	var temp = 0
	err := s.db.Raw(`SELECT 1 FROM employees WHERE `+field+` = ?`, value).Scan(&temp).Error
	if err != nil {
		return false, err
	}
	return false, nil
}

// get employee list data
func (s *Services) ListEmployee(c *gin.Context, limit, offset int) ([]domain.Employee, int64, error) {
	var (
		res        []domain.Employee
		totalCount int64
		roleId     = c.Query("role_id")
		storeId    = c.Query("store_id")
		search     = c.Query("search")
		status     = c.Query("status")
	)

	query := s.db.
		Model(&domain.Employee{}).
		Preload("Store").Preload("Roles")
	if roleId != "" {
		query = query.
			Joins("JOIN employee_roles ON employee_roles.employee_id = employees.id").
			Where("role_id = ?", roleId)
	}
	if storeId != "" {
		query = query.Where("store_id = ?", storeId)
	}

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where(`
		full_name ILIKE ? OR
		phone LIKE ? OR 
		CAST(public_id AS TEXT) LIKE ?`,
			search, search, search)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error

	if err != nil {
		s.log.Error(query.Error)
		return nil, 0, err
	}

	return res, totalCount, nil
}

// get employee bonus amount
func (s *Services) GetEmployeeBonusAmount(param *domain.DashboardQueryParam, id string) (domain.DashboardCountStatsBonus, error) {
	var bonus domain.DashboardCountStatsBonus
	// Parse start and end dates
	startTime, err := time.Parse(time.RFC3339, param.StartDate)
	if err != nil {
		s.log.Error("invalid.start_date.format: %v", err)
		return bonus, err
	}
	endTime := startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	if param.EndDate != "" {
		endTime, err = time.Parse(time.RFC3339, param.EndDate)
		if err != nil {
			s.log.Error("invalid.end_date.format: %v", err)
			return bonus, err
		}
	}
	beforeStart, beforeEnd := utils.BeforeDatesTime(startTime, endTime)
	query := `
	SELECT
		SUM(CASE WHEN created_at BETWEEN ? AND ? THEN bonus_amount END) AS bonus_amount,
		SUM(CASE WHEN created_at BETWEEN ? AND ? THEN bonus_amount END) AS before_bonus_amount
	FROM employee_bonus  WHERE employee_id = ?;`
	err = s.db.Raw(query, startTime, endTime, beforeStart, beforeEnd, id).Scan(&bonus).Error
	if err != nil {
		s.log.Error(err)
		return bonus, err
	}
	return bonus, nil
}

// add employee bonus
func (s *Services) AddEmployeeBonus(ctx context.Context, tx *gorm.DB, req *domain.EmployeeBonusRequest) error {
	err := tx.Exec(`
	INSERT INTO employee_bonus (
		employee_id, 
		sale_id, 
		cashbox_operation_id, 
		product_id, 
		quantity, 
		unit_quantity, 
		bonus_amount
		) 
	VALUES(
		?, ?, ?, ?, ?, ?, ?
		)`,
		req.EmployeeId,
		req.SaleId,
		req.CashboxOperationId,
		req.ProductId,
		req.Quantity,
		req.UnitQuantity,
		req.BonusAmount,
	).Error
	if err != nil {
		s.log.Error("ERROR on adding bonus to employee: ", err)
		return err
	}

	return nil
}
