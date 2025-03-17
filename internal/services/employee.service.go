package services

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
