package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain/constants"

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

// get one
func (s *Services) GetEmployeeById(ctx context.Context, tx *gorm.DB, id string) (*domain.Employee, error) {
	var employee domain.Employee
	err := tx.WithContext(ctx).Take(&employee, "id = ?", id).Error
	if err != nil {
		s.log.Errorf("could not get employee by id: %v", err)
		return nil, domain.InternalServerError
	}
	return &employee, nil
}

// get employee list data
func (s *Services) GetEmployees(ctx context.Context, params *domain.EmployeeQueryParams) ([]domain.Employee, int64, error) {
	var (
		res        []domain.Employee
		totalCount int64
	)

	query := s.db.
		Model(&domain.Employee{}).
		Preload("Store").Preload("Roles").Where("status != ?", constants.GeneralStatusDeleted)
	if params.RoleId != "" {
		query = query.
			Joins("JOIN employee_roles ON employee_roles.employee_id = employees.id").
			Where("role_id = ?", params.RoleId)
	}
	if params.StoreId != "" {
		query = query.Where("store_id = ?", params.StoreId)
	}
	if params.CompanyId != "" {
		query = query.Where("company_id = ?", params.CompanyId)
	}

	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where(`
		full_name ILIKE ? OR
		phone LIKE ? OR 
		CAST(public_id AS TEXT) LIKE ?`,
			params.Search, params.Search, params.Search)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	err := query.WithContext(ctx).
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("created_at DESC").
		Find(&res).Error

	if err != nil {
		s.log.Errorf("could not employees: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get employee bonus amount
func (s *Services) GetEmployeeBonusAmount(ctx context.Context, param *domain.DashboardQueryParam, id string) (domain.DashboardCountStatsBonus, error) {
	var (
		bonus domain.DashboardCountStatsBonus

		startTimeInUTC = (*param.StartDate).ToUTC()
		endTimeInUTC   = domain.AddDefaultDuration(*param.StartDate, param.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	fmt.Println("startTimeStr", startTimeStr)
	fmt.Println("endTimeStr", endTimeStr)

	query := `
	SELECT
		SUM(CASE WHEN created_at BETWEEN ? AND ? THEN bonus_amount END) AS bonus_amount,
		SUM(CASE WHEN created_at BETWEEN ? AND ? THEN bonus_amount END) AS before_bonus_amount
	FROM employee_bonus  WHERE employee_id = ?;`

	err := s.db.WithContext(ctx).Raw(query, startTimeStr, endTimeStr, beforeStartTimeStr, beforeEndTimeStr, id).Scan(&bonus).Error
	if err != nil {
		s.log.Errorf("could not get employee bonus amount: %v", err)
		return bonus, domain.InternalServerError
	}
	return bonus, nil
}

// add employee bonus
func (s *Services) AddEmployeeBonus(ctx context.Context, tx *gorm.DB, req *domain.EmployeeBonusRequest) error {
	err := tx.Exec(`
	INSERT INTO employee_bonus (
		employee_id, 
		sale_id, 
		product_id, 
		quantity, 
		unit_quantity, 
		bonus_amount,
		cashbox_operation_id
		) 
	VALUES(
		?, ?, ?, ?, ?, ?, ?
		)`,
		req.EmployeeId,
		req.SaleId,
		req.ProductId,
		req.Quantity,
		req.UnitQuantity,
		req.BonusAmount,
		req.CashboxOperationId,
	).Error
	if err != nil {
		s.log.Errorf("could not add bonus to employee: %v", err)
		return domain.InternalServerError
	}

	return nil
}
