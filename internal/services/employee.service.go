package services

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
func (s *Services) GetEmployeeBonusAmount(param *domain.DashboardQueryParam, id string) (float64, error) {
	var (
		bonus  float64
		args   []any
		query  = `SELECT COALESCE(SUM(bonus_amount), 0) AS bonus_amount FROM employee_bonus `
		filter = `WHERE employee_id = ?`
	)
	// add employee id
	args = append(args, id)
	if param.StartDate != "" && param.EndDate == "" {
		filter += ` AND created_at::date = ?`
		args = append(args, param.StartDate)
	}
	if param.StartDate != "" && param.EndDate != "" {
		filter += ` AND created_at::date BETWEEN ? AND ?`
		args = append(args, param.StartDate, param.EndDate)
	}
	query += filter
	err := s.db.Raw(query, args...).Scan(&bonus).Error
	if err != nil {
		s.log.Error(err)
		return 0, err
	}
	return bonus, nil
}

// add employee bonus
func (s *Services) AddEmployeeBonus(tx *gorm.DB, req *domain.EmployeeBonusRequest) error {
	err := tx.Exec(`
	INSERT INTO employee_bonus (
		employee_id, sale_id, product_id, quantity, unit_quantity, bonus_amount) 
	VALUES(?, ?, ?, ?, ?)`, req.EmployeeId, req.SaleId, req.ProductId, req.Quantity, req.UnitQuantity, req.BonusAmount).Error
	if err != nil {
		return err
	}

	return nil
}
