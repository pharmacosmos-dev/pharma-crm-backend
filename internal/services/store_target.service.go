package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/pharma-crm-backend/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)


func (s *Services) CreateStoreTarget(ctx context.Context, req *domain.StoreTargetRequest) (*domain.StoreTarget, error) {
	var existing domain.StoreTarget
	err := s.db.WithContext(ctx).
		Where("store_id = ? AND year = ? AND month = ?", req.StoreId, req.Year, req.Month).
		First(&existing).Error

	if err == nil {
		return nil, domain.AlreadyExistsError
	}

	if err != gorm.ErrRecordNotFound {
		s.log.Errorf("could not check existing store target: %v", err)
		return nil, domain.InternalServerError
	}

	// Store orqali company_id ni olamiz
	var store domain.Store
	if err := s.db.WithContext(ctx).Select("id, company_id").First(&store, "id = ?", req.StoreId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get store: %v", err)
		return nil, domain.InternalServerError
	}

	startOfMonth := time.Date(req.Year, time.Month(req.Month), 1, 0, 0, 0, 0, time.Local)

	target := domain.StoreTarget{
		Id:        uuid.New().String(),
		StoreId:   req.StoreId,
		CompanyId: store.CompanyId,
		Amount:    req.Amount,
		Year:      req.Year,
		Month:     req.Month,
		UpdatedAt: &startOfMonth,
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&target).Error; err != nil {
		tx.Rollback()
		s.log.Errorf("could not create store target: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.distributeToEmployees(tx, &target); err != nil {
		tx.Rollback()
		return nil, domain.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit store target transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return &target, nil
}


func (s *Services) distributeToEmployees(tx *gorm.DB, target *domain.StoreTarget) error {
	var employees []domain.Employee
	if err := tx.Where("store_id = ? AND status = ?", target.StoreId, "active").
		Find(&employees).Error; err != nil {
		s.log.Errorf("could not get employees for store %s: %v", target.StoreId, err)
		return err
	}

	if len(employees) == 0 {
		return nil
	}


	perEmployee := target.Amount / float64(len(employees))

	// updated_at ni oyning boshiga set qilamiz, shunda cron shu oyning barcha saleslarini to'playdi
	startOfMonth := time.Date(target.Year, time.Month(target.Month), 1, 0, 0, 0, 0, time.Local)

	var employeeTargets []domain.EmployeeTarget
	for _, emp := range employees {
		employeeTargets = append(employeeTargets, domain.EmployeeTarget{
			Id:            uuid.New().String(),
			StoreTargetId: target.Id,
			EmployeeId:    emp.Id,
			StoreId:       target.StoreId,
			CompanyId:     target.CompanyId,
			Amount:        perEmployee,
			Year:          target.Year,
			Month:         target.Month,
			UpdatedAt:     &startOfMonth,
		})
	}

	if err := tx.Create(&employeeTargets).Error; err != nil {
		s.log.Errorf("could not insert employee targets: %v", err)
		return err
	}

	return nil
}


func (s *Services) UpdateStoreTarget(ctx context.Context, id string, req *domain.StoreTargetUpdateRequest) (*domain.StoreTarget, error) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	var existing domain.StoreTarget
	err := s.db.WithContext(ctx).
		Where("id = ?", id).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("store target not found")
	}
	if err != nil {
		s.log.Errorf("could not get store target: %v", err)
		return nil, domain.InternalServerError
	}

	if existing.Year < currentYear ||
		(existing.Year == currentYear && existing.Month <= currentMonth) {
		return nil, fmt.Errorf("permission denied: can only update next month or future targets")
	}

	existing.Amount = req.Amount
	nowTime := time.Now()
	existing.UpdatedAt = &nowTime

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(&existing).Error; err != nil {
		tx.Rollback()
		s.log.Errorf("could not update store target: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.distributeToEmployees(tx, &existing); err != nil {
		tx.Rollback()
		return nil, domain.InternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit update store target transaction: %v", err)
		return nil, domain.InternalServerError
	}

	return &existing, nil
}


func (s *Services) GetStoreTargetHistory(ctx context.Context, storeId string, companyId string, year, month int) ([]domain.StoreTargetHistoryItem, error) {
	
	var storeCheck struct {
		CompanyId string `gorm:"column:company_id"`
	}
	if err := s.db.WithContext(ctx).
		Table("store_targets").
		Select("company_id").
		Where("store_id = ?", storeId).
		Take(&storeCheck).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not validate store: %v", err)
		return nil, domain.InternalServerError
	}

	if storeCheck.CompanyId != companyId {
		return nil, domain.NotFoundError
	}

	query := s.db.WithContext(ctx).
		Table("store_targets st").
		Select(`
			st.id,
			st.store_id,
			st.amount,
			COALESCE(st.sales, 0) AS sales,
			st.year,
			st.month
		`).
		Where("st.store_id = ? AND st.company_id = ?", storeId, companyId)

	if year > 0 {
		query = query.Where("st.year = ?", year)
	}
	if month > 0 {
		query = query.Where("st.month = ?", month)
	}

	var results []domain.StoreTargetHistoryItem
	if err := query.Order("st.year DESC, st.month DESC").Scan(&results).Error; err != nil {
		s.log.Errorf("could not get store target history: %v", err)
		return nil, domain.InternalServerError
	}

	if results == nil {
		results = []domain.StoreTargetHistoryItem{}
	}

	return results, nil
}


func (s *Services) GetStoreTargetList(ctx context.Context, params *domain.StoreTargetQueryParams) ([]domain.StoreTargetListItem, int64, error) {
	// store_id berilgan bo'lsa kompaniyaga tegishliligini tekshiramiz
	if params.StoreId != "" && len(params.CompanyIds) > 0 {
		var storeCheck struct {
			CompanyId string `gorm:"column:company_id"`
		}
		if err := s.db.WithContext(ctx).
			Table("stores").
			Select("company_id").
			Where("id = ?", params.StoreId).
			Take(&storeCheck).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, domain.NotFoundError
			}
			s.log.Errorf("could not validate store: %v", err)
			return nil, 0, domain.InternalServerError
		}
		found := false
		for _, cid := range params.CompanyIds {
			if storeCheck.CompanyId == cid {
				found = true
				break
			}
		}
		if !found {
			return nil, 0, domain.NotFoundError
		}
	}

	now := time.Now()

	year := params.Year
	month := params.Month
	if year == 0 || month == 0 {
		year = now.Year()
		month = int(now.Month())
	}

	var count int64
	countQuery := s.db.WithContext(ctx).Table("store_targets st").
		Where("st.year = ? AND st.month = ?", year, month)
		
	if len(params.CompanyIds) > 0 {
		countQuery = countQuery.Where("st.company_id IN ?", params.CompanyIds)
	}
	if params.StoreId != "" {
		countQuery = countQuery.Where("st.store_id = ?", params.StoreId)
	}
	if err := countQuery.Count(&count).Error; err != nil {
		s.log.Errorf("could not count store targets: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if count == 0 {
		return []domain.StoreTargetListItem{}, 0, nil
	}

	query := s.db.WithContext(ctx).
		Table("store_targets st").
		Select(`
			st.id,
			st.store_id,
			st.company_id,
			st.amount,
			st.year,
			st.month,
			stores.name AS store_name,
			COALESCE(st.sales, 0) AS sales
		`).
		Joins("LEFT JOIN stores ON stores.id = st.store_id").
		Where("st.year = ? AND st.month = ?", year, month)

	if len(params.CompanyIds) > 0 {
		query = query.Where("st.company_id IN ?", params.CompanyIds)
	}
	if params.StoreId != "" {
		query = query.Where("st.store_id = ?", params.StoreId)
	}

	// Pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.StoreTargetListItem
	if err := query.Order("stores.name ASC").Scan(&results).Error; err != nil {
		s.log.Errorf("could not get store target list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.StoreTargetListItem{}
	}

	return results, count, nil
}


func (s *Services) GetEmployeeTargetHistoryByStore(
	ctx context.Context,
	params *domain.EmployeeTargetQueryParams,
) ([]domain.EmployeeTargetHistoryItem, int64, error) {

	// Store kompaniyaga tegishliligini tekshiramiz
	if params.CompanyId != "" {

		var storeCheck struct {
			CompanyId string `gorm:"column:company_id"`
		}

		if err := s.db.WithContext(ctx).
			Table("store_targets").
			Select("company_id").
			Where("store_id = ?", params.StoreId).
			Take(&storeCheck).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, domain.NotFoundError
			}
			s.log.Errorf("could not validate store: %v", err)
			return nil, 0, domain.InternalServerError
		}
		if storeCheck.CompanyId != params.CompanyId {
			return nil, 0, domain.NotFoundError
		}
	}

	now := time.Now()
	
	year := params.Year
	if year == 0 {
		year = now.Year()
	}
	month := params.Month
	if month == 0 {
		month = int(now.Month())
	}

	query := s.db.WithContext(ctx).
		Table("employee_targets et").
		Select(`
			et.employee_id,
			e.full_name AS employee_name,
			et.amount,
			COALESCE(et.sales, 0) AS sales,
			et.year,
			et.month
		`).
		Joins(`LEFT JOIN employees e ON e.id = et.employee_id`).
		Where("et.store_id = ?", params.StoreId).
		Where("et.year = ?", year).
		Where("et.month = ?", month).
		Order("e.full_name ASC")

	// Filter employee_id agar berilgan bo’lsa
	if params.EmployeeId != "" {
		query = query.Where("et.employee_id = ?", params.EmployeeId)
	}

	// Count query
	var count int64
	countQuery := s.db.WithContext(ctx).
		Table("employee_targets").
		Where("store_id = ?", params.StoreId).
		Where("year = ?", year).
		Where("month = ?", month)

	if params.EmployeeId != "" {
		countQuery = countQuery.Where("employee_id = ?", params.EmployeeId)
	}

	if err := countQuery.Count(&count).Error; err != nil {
		s.log.Errorf("could not count employee targets: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// Pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	

	var results []domain.EmployeeTargetHistoryItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get employee target history by store: %v", err)
		return nil, 0, domain.InternalServerError
	}

	days := daysIn(year, month)

	for i := range results {
		raw := results[i].Amount / float64(days)
		results[i].DailyTarget = math.Round(raw*100) / 100
	}

	return results, count, nil
}


func daysIn(year, month int) int {
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)
	return t.Day()
}

// UpdateStoreTargetSales - her soatda chaqiriladi (cron)
// store_targets.updated_at dan keyin kelgan yangi saleslarni sales ga qo'shib update qiladi
func (s *Services) UpdateStoreTargetSales() {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	err := s.db.Exec(`
		UPDATE store_targets st
		SET
			sales = st.sales + COALESCE((
				SELECT SUM(s.total_amount)
				FROM sales s
				WHERE s.store_id = st.store_id
				AND s.stage = '9'
				AND s.is_returned = false
				AND s.created_at > st.updated_at
			), 0),
			updated_at = NOW()
		WHERE st.year = ? AND st.month = ?
	`, year, month).Error

	if err != nil {
		s.log.Errorf("cron UpdateStoreTargetSales: %v", err)
		return
	}
	log.Printf("UpdateStoreTargetSales completed for %d-%02d", year, month)
}


// UpdateEmployeeTargetSales - her soatda chaqiriladi (cron)
// employee_targets.updated_at dan keyin kelgan yangi saleslarni sales ga qo'shib update qiladi
func (s *Services) UpdateEmployeeTargetSales() {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	err := s.db.Exec(`
		UPDATE employee_targets et
		SET
			sales = et.sales + COALESCE((
				SELECT SUM(s.total_amount)
				FROM sales s
				WHERE s.store_id = et.store_id
				AND s.employee_id = et.employee_id
				AND s.stage = '9'
				AND s.is_returned = false
				AND s.created_at > et.updated_at
			), 0),
			updated_at = NOW()
		WHERE et.year = ? AND et.month = ?
	`, year, month).Error

	if err != nil {
		s.log.Errorf("cron UpdateEmployeeTargetSales: %v", err)
		return
	}
	log.Printf("UpdateEmployeeTargetSales completed for %d-%02d", year, month)
}


func (s *Services) GetCurrentMonthStoreTargetsSummary(ctx context.Context, companyIds []string, storeId string) (*domain.StoreTargetSummary, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	type result struct {
		TotalAmount float64 `json:"total_target_amount"`
		TotalSales  float64 `json:"total_target_sales"`
	}

	var res result

	query := s.db.WithContext(ctx).
		Table("store_targets st").
		Select(`
			COALESCE(SUM(st.amount), 0) AS total_amount,
			COALESCE(SUM(st.sales), 0) AS total_sales
		`).
		Where("st.company_id IN ?", companyIds).
		Where("st.year = ?", year).
		Where("st.month = ?", month)

	if storeId != "" {
		query = query.Where("st.store_id = ?", storeId)
	}

	if err := query.Scan(&res).Error; err != nil {
		s.log.Errorf("could not get store target summary: %v", err)
		return nil, domain.InternalServerError
	}

	return &domain.StoreTargetSummary{
		TotalAmount: res.TotalAmount,
		TotalSales:  res.TotalSales,
		Year:        year,
		Month:       month,
	}, nil
}


func (s *Services) GetDailySalesStoreTargetsEmployee(ctx context.Context, employeeId string, storeId string) (*domain.EmployeeTargetHistoryItem, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	var target domain.EmployeeTarget
	err := s.db.WithContext(ctx).
		Select("id, employee_id, store_id, amount, year, month").
		Where("employee_id = ? AND store_id = ? AND year = ? AND month = ?", employeeId, storeId, year, month).
		First(&target).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		s.log.Errorf("could not get employee target: %v", err)
		return nil, domain.InternalServerError
	}
	
	var employee domain.Employee
	s.db.WithContext(ctx).Select("full_name").Take(&employee, "id = ?", employeeId)

	// Faqat bugungi sotuv yig'indisini olish (DATE(created_at) = bugun)
	var todaySales float64
	s.db.WithContext(ctx).
		Table("sales").
		Select("COALESCE(SUM(total_amount), 0)").
		Where(
			"store_id = ? AND employee_id = ? AND stage = '9' AND is_returned = false AND DATE(created_at) = CURRENT_DATE",
			storeId, employeeId,
		).
		Scan(&todaySales)

	days := daysIn(year, month)
	dailyTarget := math.Round((target.Amount/float64(days))*100) / 100

	return &domain.EmployeeTargetHistoryItem{
		EmployeeId:   employeeId,
		EmployeeName: employee.FullName,
		Amount:       target.Amount,
		DailyTarget:  dailyTarget,
		Sales:        todaySales,
		Year:         year,
		Month:        month,
	}, nil
}
