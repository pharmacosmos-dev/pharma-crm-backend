package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	// ✅ Agar mavjud bo'lsa — update qilamiz
	if err == nil {
		now := time.Now()
		currentYear := now.Year()
		currentMonth := int(now.Month())

		if existing.Year < currentYear ||
			(existing.Year == currentYear && existing.Month < currentMonth) {
			return nil, fmt.Errorf("permission denied: can only update current or future month targets")
		}

		tx := s.db.WithContext(ctx).Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		if err := tx.Model(&existing).Select("amount").Updates(map[string]interface{}{
			"amount": req.Amount,
		}).Error; err != nil {
			tx.Rollback()
			s.log.Errorf("could not update store target amount: %v", err)
			return nil, domain.InternalServerError
		}
		existing.Amount = req.Amount

		if err := s.redistributeToEmployees(tx, &existing); err != nil {
			tx.Rollback()
			return nil, domain.InternalServerError
		}

		if err := tx.Commit().Error; err != nil {
			s.log.Errorf("could not commit upsert store target transaction: %v", err)
			return nil, domain.InternalServerError
		}

		return &existing, nil
	}

	// ❌ DB xatolik bo'lsa
	if err != gorm.ErrRecordNotFound {
		s.log.Errorf("could not check existing store target: %v", err)
		return nil, domain.InternalServerError
	}

	// ✅ Yangi yaratamiz
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
		SyncedAt:  &startOfMonth,
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


// distributeToEmployees - yangi store_target yaratilganda xodim targetlarini CREATE qiladi.
// Faqat birinchi marta (target yo'q bo'lsa) ishlatiladi.
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

	// synced_at is set to the beginning of the month — cron will calculate all sales for that month
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
			SyncedAt:      &startOfMonth,
		})
	}

	if err := tx.Create(&employeeTargets).Error; err != nil {
		s.log.Errorf("could not insert employee targets: %v", err)
		return err
	}

	return nil
}

// redistributeToEmployees - store_target amount'i o'zgarganda xodim targetlarini qayta hisoblaydi.
// Mavjud employee target bo'lsa amount'ini UPDATE qiladi (sales saqlanadi),
// yangi xodim bo'lsa CREATE qiladi.
func (s *Services) redistributeToEmployees(tx *gorm.DB, target *domain.StoreTarget) error {
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
	startOfMonth := time.Date(target.Year, time.Month(target.Month), 1, 0, 0, 0, 0, time.Local)

	for _, emp := range employees {
		var existing domain.EmployeeTarget
		err := tx.Where("store_id = ? AND employee_id = ? AND year = ? AND month = ?",
			target.StoreId, emp.Id, target.Year, target.Month).
			First(&existing).Error

		if err == nil {
			//Existing — we only update amount and store_target_id (sales and synced_at do not change)
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"amount":          perEmployee,
				"store_target_id": target.Id,
			}).Error; err != nil {
				s.log.Errorf("could not update employee target for employee %s: %v", emp.Id, err)
				return err
			}
		} else if err == gorm.ErrRecordNotFound {
			// New employee — we will create
			newTarget := domain.EmployeeTarget{
				Id:            uuid.New().String(),
				StoreTargetId: target.Id,
				EmployeeId:    emp.Id,
				StoreId:       target.StoreId,
				CompanyId:     target.CompanyId,
				Amount:        perEmployee,
				Year:          target.Year,
				Month:         target.Month,
				SyncedAt:      &startOfMonth,
			}
			if err := tx.Create(&newTarget).Error; err != nil {
				s.log.Errorf("could not create employee target for employee %s: %v", emp.Id, err)
				return err
			}
		} else {
			s.log.Errorf("could not query employee target for employee %s: %v", emp.Id, err)
			return err
		}
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
		(existing.Year == currentYear && existing.Month < currentMonth) {
		return nil, fmt.Errorf("permission denied: can only update current or future month targets")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&existing).Select("amount").Updates(map[string]interface{}{
		"amount": req.Amount,
	}).Error; err != nil {
		tx.Rollback()
		s.log.Errorf("could not update store target amount: %v", err)
		return nil, domain.InternalServerError
	}
	existing.Amount = req.Amount

	if err := s.redistributeToEmployees(tx, &existing); err != nil {
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
			if cid == storeCheck.CompanyId {
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

	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}


	// ==========================================================
	// ==================== COUNT QUERY =========================
	// ==========================================================

	countQuery := s.db.WithContext(ctx).
		Table("store_targets st").
		Joins("LEFT JOIN stores ON stores.id = st.store_id").
		Where("st.year = ? AND st.month = ?", year, month)

	if len(params.CompanyIds) > 0 {
		countQuery = countQuery.Where("st.company_id IN ?", params.CompanyIds)
	}

	if params.StoreId != "" {
		countQuery = countQuery.Where("st.store_id = ?", params.StoreId)
	}

	if params.SearchField != "" {
		countQuery = countQuery.Where("stores.name ILIKE ?", "%"+params.SearchField+"%")
	}

	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count store targets: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if totalCount == 0 {
		return []domain.StoreTargetListItem{}, 0, nil
	}

	// ==========================================================
	// ==================== MAIN QUERY ==========================
	// ==========================================================

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

	if params.SearchField != "" {
		query = query.Where("stores.name ILIKE ?", "%"+params.SearchField+"%")
	}

	// ==========================================================
	// ====================== ORDER LOGIC =======================
	// ==========================================================


	switch params.Order {
	case "+store_name":
		query = query.Order("stores.name ASC")
	case "-store_name":
		query = query.Order("stores.name DESC")
	case "+target":
		query = query.Order("st.amount ASC")
	case "-target":
		query = query.Order("st.amount DESC")
	case "+sales":
		query = query.Order("COALESCE(st.sales,0) ASC")
	case "-sales":
		query = query.Order("COALESCE(st.sales,0) DESC")
	default:
		query = query.Order("st.updated_at DESC")
	}


	// ==========================================================
	// ====================== PAGINATION ========================
	// ==========================================================

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	var results []domain.StoreTargetListItem
	if err := query.Scan(&results).Error; err != nil {
		s.log.Errorf("could not get store target list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if results == nil {
		results = []domain.StoreTargetListItem{}
	}

	return results, totalCount, nil
}


func (s *Services) GetEmployeeTargetHistoryByStore(
	ctx context.Context,
	params *domain.EmployeeTargetQueryParams,
) ([]domain.EmployeeTargetHistoryItem, int64, error) {

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

	
	if params.EmployeeId != "" {
		query = query.Where("et.employee_id = ?", params.EmployeeId)
	}

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

	// days := daysIn(year, month)

	// for i := range results {
	// 	raw := results[i].Amount / float64(days)
	// 	results[i].DailyTarget = math.Round(raw*100) / 100
	// }

	return results, count, nil
}


// func daysIn(year, month int) int {
// 	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)
// 	return t.Day()
// }


// UpdateStoreTargetSales - called every hour (cron)
// updates sales by adding new sales that arrive after store_targets.updated_at
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
				AND s.created_at > st.synced_at
			), 0),
			synced_at = NOW()
		WHERE st.year = ? AND st.month = ?
		AND st.synced_at IS NOT NULL
	`, year, month).Error

	if err != nil {
		s.log.Errorf("cron UpdateStoreTargetSales: %v", err)
		return
	}
	s.log.Infof("UpdateStoreTargetSales completed for %d-%02d", year, month)
}


// UpdateEmployeeTargetSales - called every hour (cron)
// updates sales by adding new sales that arrive after employee_targets.updated_at
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
				AND s.created_at > et.synced_at
			), 0),
			synced_at = NOW()
		WHERE et.year = ? AND et.month = ?
		AND et.synced_at IS NOT NULL
	`, year, month).Error

	if err != nil {
		s.log.Errorf("cron UpdateEmployeeTargetSales: %v", err)
		return
	}
	s.log.Infof("UpdateEmployeeTargetSales completed for %d-%02d", year, month)
}

// AutoCreateMonthlyStoreTargets - har oyning 1-kuni 00:00 da cron tomonidan chaqiriladi.
// Oldingi oyda store_target bo'lgan har bir do'kon uchun yangi oy target mavjud bo'lmasa,
// oldingi oyning amount'i bilan yangi store_target va employee_target'larni yaratadi.
func (s *Services) AutoCreateMonthlyStoreTargets() {
	ctx := context.Background()
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	prevTime := now.AddDate(0, -1, 0)
	prevYear := prevTime.Year()
	prevMonth := int(prevTime.Month())

	// Get all store_targets available in the previous month
	var prevTargets []domain.StoreTarget
	if err := s.db.WithContext(ctx).
		Where("year = ? AND month = ?", prevYear, prevMonth).
		Find(&prevTargets).Error; err != nil {
		s.log.Errorf("AutoCreateMonthlyStoreTargets: Error retrieving previous month's targets: %v", err)
		return
	}

	// log.Printf("AutoCreateMonthlyStoreTargets: Found %d stores for %d-%02d",
    // 	len(prevTargets), prevYear, prevMonth)

	created := 0
	for _, prev := range prevTargets {
		var existing domain.StoreTarget
		err := s.db.WithContext(ctx).
			Where("store_id = ? AND year = ? AND month = ?", prev.StoreId, currentYear, currentMonth).
			First(&existing).Error

		if err == nil {
			continue
		}

		if err != gorm.ErrRecordNotFound {
			s.log.Errorf("AutoCreateMonthlyStoreTargets: Error checking existence for store %s: %v",
				prev.StoreId, err)
			continue
		}

		// Yengi store_target yaratish (oldingi oyning amount'i bilan)
		startOfMonth := time.Date(currentYear, time.Month(currentMonth), 1, 0, 0, 0, 0, time.Local)
		target := domain.StoreTarget{
			Id:        uuid.New().String(),
			StoreId:   prev.StoreId,
			CompanyId: prev.CompanyId,
			Amount:    prev.Amount,
			Year:      currentYear,
			Month:     currentMonth,
			SyncedAt:  &startOfMonth,
		}

		tx := s.db.WithContext(ctx).Begin()
		if err := tx.Create(&target).Error; err != nil {
			tx.Rollback()
			s.log.Errorf("AutoCreateMonthlyStoreTargets: Error creating target for store %s: %v", 
				prev.StoreId, err)
			continue
	}

		if err := s.distributeToEmployees(tx, &target); err != nil {
			tx.Rollback()
			s.log.Errorf("AutoCreateMonthlyStoreTargets: Error creating employee targets for store %s: %v",
				prev.StoreId, err)
			continue
		}

		if err := tx.Commit().Error; err != nil {
			s.log.Errorf("AutoCreateMonthlyStoreTargets: Commit error for store %s: %v",
				prev.StoreId, err)
			continue
		}

		created++
		log.Printf("AutoCreateMonthlyStoreTargets: Created target %d - %02d for store %s (amount: %.2f)",
			currentYear, currentMonth, prev.StoreId, prev.Amount)
	}

	log.Printf("AutoCreateMonthlyStoreTargets: %d-%02d done — Target created for store %d/%d",
		currentYear, currentMonth, created, len(prevTargets))
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

func (s *Services) GetDailySalesStoreTargetEmployee(ctx context.Context, employeeId string, storeId string) (*domain.EmployeeTargetHistoryItem, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	
	var target domain.EmployeeTarget
	err := s.db.WithContext(ctx).
		Select("id, employee_id, store_id, amount, sales, year, month").
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
	// var todaySales float64
	// s.db.WithContext(ctx).
	// 	Table("sales").
	// 	Select("COALESCE(SUM(total_amount), 0)").
	// 	Where(
	// 		"store_id = ? AND employee_id = ? AND stage = '9' AND is_returned = false AND DATE(created_at) = CURRENT_DATE",
	// 		storeId, employeeId,
	// 	).
	// 	Scan(&todaySales)

	// days := daysIn(year, month)
	// dailyTarget := math.Round((target.Amount/float64(days))*100) / 100

	return &domain.EmployeeTargetHistoryItem{
		EmployeeId:   employeeId,
		EmployeeName: employee.FullName,
		Amount:       target.Amount,
		//DailyTarget:  dailyTarget,
		Sales:        target.Sales,
		Year:         year,
		Month:        month,
	}, nil
}

// UpdateEmployeeTargetAmount - bitta xodimning target summasini o'zgartiradi.
// Qolgan xodimlar uchun qolgan summa teng taqsimlanadi.
// Store target summasiga ta'sir qilmaydi.
func (s *Services) UpdateEmployeeTargetAmount(ctx context.Context, storeTargetId string, employeeId string, newAmount float64) error {

	var storeTarget domain.StoreTarget
	if err := s.db.WithContext(ctx).First(&storeTarget, "id = ?", storeTargetId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.NotFoundError
		}
		s.log.Errorf("UpdateEmployeeTargetAmount: could not get store target: %v", err)
		return domain.InternalServerError
	}

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())
	if storeTarget.Year < currentYear ||
		(storeTarget.Year == currentYear && storeTarget.Month < currentMonth) {
		return fmt.Errorf("permission denied: can only update current or future month targets")
	}

	if newAmount >= storeTarget.Amount {
		return fmt.Errorf("employee target amount cannot be greater than or equal to store target amount")
	}

	var empTarget domain.EmployeeTarget
	if err := s.db.WithContext(ctx).
		Where("store_target_id = ? AND employee_id = ?", storeTargetId, employeeId).
		First(&empTarget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.NotFoundError
		}
		s.log.Errorf("UpdateEmployeeTargetAmount: could not get employee target: %v", err)
		return domain.InternalServerError
	}

	var otherTargets []domain.EmployeeTarget
	if err := s.db.WithContext(ctx).
		Where("store_target_id = ? AND employee_id != ?", storeTargetId, employeeId).
		Find(&otherTargets).Error; err != nil {
		s.log.Errorf("UpdateEmployeeTargetAmount: could not get other employee targets: %v", err)
		return domain.InternalServerError
	}

	
	remaining := storeTarget.Amount - newAmount
	var perOther float64
	if len(otherTargets) > 0 {
		perOther = remaining / float64(len(otherTargets))
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&empTarget).Update("amount", newAmount).Error; err != nil {
		tx.Rollback()
		s.log.Errorf("UpdateEmployeeTargetAmount: could not update employee target: %v", err)
		return domain.InternalServerError
	}


	for _, other := range otherTargets {
		if err := tx.Model(&other).Update("amount", perOther).Error; err != nil {
			tx.Rollback()
			s.log.Errorf("UpdateEmployeeTargetAmount: could not update other employee target %s: %v", other.EmployeeId, err)
			return domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("UpdateEmployeeTargetAmount: could not commit transaction: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// HandleEmployeeStoreChange - xodim bir do'kondan ikkinchisiga ko'chirilganda
// joriy oy uchun employee_target larni qayta taqsimlaydi.
// Goroutine da chaqiriladi — response ga ta'sir qilmaydi.
func (s *Services) HandleEmployeeStoreChange(oldStoreId, newStoreId string) {
	ctx := context.Background()
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// --- STORE A (eski do'kon): qolgan xodimlar orasida qayta taqsimlash ---
	if oldStoreId != "" {
		var oldTarget domain.StoreTarget
		err := s.db.WithContext(ctx).
			Where("store_id = ? AND year = ? AND month = ?", oldStoreId, year, month).
			First(&oldTarget).Error
		if err == nil {
			tx := s.db.WithContext(ctx).Begin()
			if err := s.redistributeToEmployees(tx, &oldTarget); err != nil {
				tx.Rollback()
				s.log.Errorf("HandleEmployeeStoreChange: redistribute old store %s failed: %v", oldStoreId, err)
			} else if err := tx.Commit().Error; err != nil {
				s.log.Errorf("HandleEmployeeStoreChange: commit old store %s failed: %v", oldStoreId, err)
			}
		} else if err != gorm.ErrRecordNotFound {
			s.log.Errorf("HandleEmployeeStoreChange: get old store target failed: %v", err)
		}
	}

	// --- STORE B (yangi do'kon): yangi employee uchun create + qayta taqsimlash ---
	if newStoreId != "" {
		var newTarget domain.StoreTarget
		err := s.db.WithContext(ctx).
			Where("store_id = ? AND year = ? AND month = ?", newStoreId, year, month).
			First(&newTarget).Error
		if err == nil {
			tx := s.db.WithContext(ctx).Begin()
			if err := s.redistributeToEmployees(tx, &newTarget); err != nil {
				tx.Rollback()
				s.log.Errorf("HandleEmployeeStoreChange: redistribute new store %s failed: %v", newStoreId, err)
			} else if err := tx.Commit().Error; err != nil {
				s.log.Errorf("HandleEmployeeStoreChange: commit new store %s failed: %v", newStoreId, err)
			}
		} else if err != gorm.ErrRecordNotFound {
			s.log.Errorf("HandleEmployeeStoreChange: get new store target failed: %v", err)
		}
	}
}

// UpsertStoreTargetsFromExcel - Upserts store_id, amount, month, and year
// data coming from an Excel file into the store_target table within a single transaction.
// If the record exists → update the amount + redistribute employees
// If the record does not exist → create a new record + distribute employees
func (s *Services) UpsertStoreTargetsFromExcel(ctx context.Context, rows []domain.StoreTargetExcelRow) (*domain.StoreTargetUpsertResult, error) {
	result := &domain.StoreTargetUpsertResult{
		Total: len(rows),
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, row := range rows {
		if row.StoreId == "" || row.Amount <= 0 || row.Month < 1 || row.Month > 12 || row.Year < 2000 {
			result.Skipped++
			continue
		}

		var existing domain.StoreTarget
		err := tx.Where("store_id = ? AND year = ? AND month = ?", row.StoreId, row.Year, row.Month).
			First(&existing).Error

		if err == nil {
			if err := tx.Model(&existing).Update("amount", row.Amount).Error; err != nil {
				tx.Rollback()
				s.log.Errorf("UpsertStoreTargetsFromExcel: update failed store_id=%s year=%d month=%d: %v",
					row.StoreId, row.Year, row.Month, err)
				return nil, domain.InternalServerError
			}
			existing.Amount = row.Amount

			if err := s.redistributeToEmployees(tx, &existing); err != nil {
				tx.Rollback()
				return nil, domain.InternalServerError
			}
			result.Updated++

		} else if err == gorm.ErrRecordNotFound {
			var store domain.Store
			if err := tx.Select("id, company_id").First(&store, "id = ?", row.StoreId).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					s.log.Warnf("UpsertStoreTargetsFromExcel: store not found store_id=%s, skip", row.StoreId)
					result.Skipped++
					continue
				}
				tx.Rollback()
				s.log.Errorf("UpsertStoreTargetsFromExcel: get store failed: %v", err)
				return nil, domain.InternalServerError
			}

			startOfMonth := time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.Local)
			target := domain.StoreTarget{
				Id:        uuid.New().String(),
				StoreId:   row.StoreId,
				CompanyId: store.CompanyId,
				Amount:    row.Amount,
				Year:      row.Year,
				Month:     row.Month,
				SyncedAt:  &startOfMonth,
			}

			if err := tx.Create(&target).Error; err != nil {
				tx.Rollback()
				s.log.Errorf("UpsertStoreTargetsFromExcel: create failed store_id=%s: %v", row.StoreId, err)
				return nil, domain.InternalServerError
			}

			if err := s.distributeToEmployees(tx, &target); err != nil {
				tx.Rollback()
				return nil, domain.InternalServerError
			}
			result.Created++

		} else {
			tx.Rollback()
			s.log.Errorf("UpsertStoreTargetsFromExcel: db error store_id=%s: %v", row.StoreId, err)
			return nil, domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("UpsertStoreTargetsFromExcel: commit failed: %v", err)
		return nil, domain.InternalServerError
	}

	return result, nil
}