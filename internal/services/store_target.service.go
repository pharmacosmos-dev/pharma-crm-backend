package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pharma-crm-backend/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// 1. FEATURE: Store target yaratish + darhol employee larga bo'lish
// ============================================================
func (s *Services) CreateStoreTarget(ctx context.Context, req *domain.StoreTargetRequest) (*domain.StoreTarget, error) {
	// Avval mavjud emasligini tekshirish
	var existing domain.StoreTarget
	err := s.db.WithContext(ctx).
		Where("store_id = ? AND year = ? AND month = ?", req.StoreId, req.Year, req.Month).
		First(&existing).Error

	if err == nil {
		return nil, fmt.Errorf("target already exists for this store and month, use update instead")
	}

	if err != gorm.ErrRecordNotFound {
		s.log.Errorf("could not check existing store target: %v", err)
		return nil, domain.InternalServerError
	}

	// Yangi target yaratish
	target := domain.StoreTarget{
		Id:        uuid.New().String(),
		StoreId:   req.StoreId,
		CompanyId: req.CompanyId,
		Amount:    req.Amount,
		Year:      req.Year,
		Month:     req.Month,
	}

	if err := s.db.WithContext(ctx).Create(&target).Error; err != nil {
		s.log.Errorf("could not create store target: %v", err)
		return nil, domain.InternalServerError
	}

	// Darhol employee larga bo'lish
	if err := s.distributeToEmployees(ctx, &target); err != nil {
		s.log.Errorf("could not distribute target to employees: %v", err)
		// Target yaratildi lekin bo'linmadi — rollback qilish
		s.db.WithContext(ctx).Delete(&target)
		return nil, domain.InternalServerError
	}

	return &target, nil
}

// Store dagi barcha aktiv xodimlar orasida teng bo'lish
func (s *Services) distributeToEmployees(ctx context.Context, target *domain.StoreTarget) error {
	var employees []domain.Employee
	err := s.db.WithContext(ctx).
		Where("store_id = ? AND status = ?", target.StoreId, "active").
		Find(&employees).Error

	if err != nil {
		s.log.Errorf("could not get employees for store %s: %v", target.StoreId, err)
		return err
	}

	if len(employees) == 0 {
		return nil
	}

	// Avvalgi bo'linmalarni tozalash
	s.db.WithContext(ctx).
		Where("store_target_id = ?", target.Id).
		Delete(&domain.EmployeeTarget{})

	perEmployee := target.Amount / float64(len(employees))

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
		})
	}

	if err := s.db.WithContext(ctx).Create(&employeeTargets).Error; err != nil {
		s.log.Errorf("could not insert employee targets: %v", err)
		return err
	}

	return nil
}

// ============================================================
// 2. FEATURE: Store target yangilash — FAQAT keyingi oy uchun
// ============================================================
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

	// Joriy oy yoki o'tgan oy bo'lsa — error
	if existing.Year < currentYear ||
		(existing.Year == currentYear && existing.Month <= currentMonth) {
		return nil, fmt.Errorf("permission denied: can only update next month or future targets")
	}

	// Summani yangilash
	existing.Amount = req.Amount
	nowTime := time.Now()
	existing.UpdatedAt = &nowTime

	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		s.log.Errorf("could not update store target: %v", err)
		return nil, domain.InternalServerError
	}

	// Employee target larni ham qayta hisoblash
	if err := s.distributeToEmployees(ctx, &existing); err != nil {
		s.log.Errorf("could not redistribute target to employees: %v", err)
		return nil, domain.InternalServerError
	}

	return &existing, nil
}

// ============================================================
// 3. FEATURE: Store history — store_id bo'yicha + sales LEFT JOIN
// month va year bo'yicha filter, haqiqiy daromad bilan
// ============================================================
func (s *Services) GetStoreTargetHistory(ctx context.Context, storeId string, year, month int) ([]domain.StoreTargetHistoryItem, error) {
	query := s.db.WithContext(ctx).
		Table("store_targets st").
		Select(`
			st.id,
			st.store_id,
			st.amount,
			st.year,
			st.month,
			COALESCE(SUM(s.total_amount), 0) AS actual_amount
		`).
		Joins(`
			LEFT JOIN sales s ON s.store_id = st.store_id
				AND EXTRACT(YEAR FROM s.created_at) = st.year
				AND EXTRACT(MONTH FROM s.created_at) = st.month
				AND s.status = 'completed'
				AND s.is_returned = false
		`).
		Where("st.store_id = ?", storeId).
		Group("st.id, st.store_id, st.amount, st.year, st.month")

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

	return results, nil
}

// ============================================================
// 4. FEATURE: Barcha store target list — month+year filter
// Joriy oy bo'lsa bugungi kungacha sotuv, o'tgan oy bo'lsa faqat amount
// ============================================================
func (s *Services) GetStoreTargetList(ctx context.Context, params *domain.StoreTargetQueryParams) ([]domain.StoreTargetListItem, int64, error) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	isCurrentMonth := params.Year == currentYear && params.Month == currentMonth

	var salesJoin string
	if isCurrentMonth {
		// Bugungi kungacha
		salesJoin = `
			LEFT JOIN sales s ON s.store_id = st.store_id
				AND EXTRACT(YEAR FROM s.created_at) = st.year
				AND EXTRACT(MONTH FROM s.created_at) = st.month
				AND s.status = 'completed'
				AND s.is_returned = false
				AND DATE(s.created_at) <= CURRENT_DATE
		`
	} else {
		// O'tgan oy — sales join qilmasdan, actual_amount = 0
		salesJoin = `LEFT JOIN sales s ON false`
	}

	query := s.db.WithContext(ctx).
		Table("store_targets st").
		Select(`
			st.id,
			st.store_id,
			stores.name AS store_name,
			st.amount,
			st.year,
			st.month,
			COALESCE(SUM(s.total_amount), 0) AS actual_amount
		`).
		Joins("LEFT JOIN stores ON stores.id = st.store_id").
		Joins(salesJoin).
		Group("st.id, st.store_id, stores.name, st.amount, st.year, st.month")

	if params.CompanyId != "" {
		query = query.Where("st.company_id = ?", params.CompanyId)
	}
	if params.StoreId != "" {
		query = query.Where("st.store_id = ?", params.StoreId)
	}
	if params.Year > 0 {
		query = query.Where("st.year = ?", params.Year)
	}
	if params.Month > 0 {
		query = query.Where("st.month = ?", params.Month)
	}

	var count int64
	if err := s.db.WithContext(ctx).
		Table("store_targets").
		Where("company_id = ?", params.CompanyId).
		Count(&count).Error; err != nil {
		s.log.Errorf("could not count store targets: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.StoreTargetListItem
	if err := query.Order("st.year DESC, st.month DESC").Scan(&results).Error; err != nil {
		s.log.Errorf("could not get store target list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return results, count, nil
}

// ============================================================
// 5. FEATURE: Employee joriy oy target + kunlik/oylik haqiqiy sotuvlar
// ============================================================
func (s *Services) GetEmployeeTargetWithSales(ctx context.Context, employeeId string) (*domain.EmployeeTargetWithSales, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Employee target ni olish
	var target domain.EmployeeTarget
	err := s.db.WithContext(ctx).
		Where("employee_id = ? AND year = ? AND month = ?", employeeId, year, month).
		First(&target).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		s.log.Errorf("could not get employee target: %v", err)
		return nil, domain.InternalServerError
	}

	// Employee ismini olish
	var employee domain.Employee
	s.db.WithContext(ctx).
		Select("full_name").
		Take(&employee, "id = ?", employeeId)

	// Oylik haqiqiy sotuvlar
	var monthlySales float64
	s.db.WithContext(ctx).
		Table("sales").
		Select("COALESCE(SUM(total_amount), 0)").
		Where("store_id = ? AND employee_id = ? AND status = ? AND is_returned = ? AND EXTRACT(YEAR FROM created_at) = ? AND EXTRACT(MONTH FROM created_at) = ?",
			target.StoreId, employeeId, "completed", false, year, month).
		Scan(&monthlySales)

	// Kunlik haqiqiy sotuvlar (bugun)
	var dailySales float64
	s.db.WithContext(ctx).
		Table("sales").
		Select("COALESCE(SUM(total_amount), 0)").
		Where("store_id = ? AND employee_id = ? AND status = ? AND is_returned = ? AND DATE(created_at) = CURRENT_DATE",
			target.StoreId, employeeId, "completed", false).
		Scan(&dailySales)

	days := daysIn(year, month)
	dailyTarget := target.Amount / float64(days)

	return &domain.EmployeeTargetWithSales{
		Id:                 target.Id,
		EmployeeId:         employeeId,
		EmployeeName:       employee.FullName,
		MonthlyTarget:      target.Amount,
		DailyTarget:        dailyTarget,
		ActualMonthlySales: monthlySales,
		ActualDailySales:   dailySales,
		Year:               year,
		Month:              month,
		DaysInMonth:        days,
	}, nil
}

// ============================================================
// 6. FEATURE: Store bo'yicha barcha employee lar tarixi
// store_id bo'yicha barcha employee_targets, employee ismi bilan
// ============================================================
func (s *Services) GetEmployeeTargetHistoryByStore(ctx context.Context, params *domain.EmployeeTargetQueryParams) ([]domain.EmployeeTargetHistoryItem, int64, error) {
	query := s.db.WithContext(ctx).
		Table("employee_targets et").
		Select(`
			et.employee_id,
			employees.full_name AS employee_name,
			et.amount,
			et.year,
			et.month
		`).
		Joins("LEFT JOIN employees ON employees.id = et.employee_id").
		Where("et.store_id = ?", params.StoreId)

	if params.Year > 0 {
		query = query.Where("et.year = ?", params.Year)
	}
	if params.Month > 0 {
		query = query.Where("et.month = ?", params.Month)
	}
	if params.EmployeeId != "" {
		query = query.Where("et.employee_id = ?", params.EmployeeId)
	}

	var count int64
	countQuery := s.db.WithContext(ctx).
		Table("employee_targets").
		Where("store_id = ?", params.StoreId)
	if params.Year > 0 {
		countQuery = countQuery.Where("year = ?", params.Year)
	}
	if params.Month > 0 {
		countQuery = countQuery.Where("month = ?", params.Month)
	}
	if err := countQuery.Count(&count).Error; err != nil {
		s.log.Errorf("could not count employee targets: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.EmployeeTargetHistoryItem
	if err := query.Order("et.year DESC, et.month DESC, employees.full_name ASC").Scan(&results).Error; err != nil {
		s.log.Errorf("could not get employee target history by store: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return results, count, nil
}

// ============================================================
// HELPER: Oydagi kunlar soni
// ============================================================
func daysIn(year, month int) int {
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}

// ============================================================
// CRON: Har oy 1-kuni — eski versiya (optional, faqat auto uchun)
// ============================================================
func (s *Services) DistributeMonthlyTargets() {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	var storeTargets []domain.StoreTarget
	err := s.db.
		Where("year = ? AND month = ?", year, month).
		Find(&storeTargets).Error

	if err != nil {
		s.log.Errorf("could not get store targets for distribution: %v", err)
		return
	}

	for _, target := range storeTargets {
		if err := s.distributeToEmployees(context.Background(), &target); err != nil {
			s.log.Errorf("cron: could not distribute for store %s: %v", target.StoreId, err)
		}
	}

	log.Printf("Monthly targets distributed for %d-%02d: %d stores", year, month, len(storeTargets))
}
