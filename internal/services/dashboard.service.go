package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

func (s *Services) DashboardChartStats(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.ChartResponse, error) {
	var (
		res       []domain.ChartResponse
		truncFunc string
		startTime = (*params.StartDate).GetTime()
		endTime   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).GetTime()
	)

	if params.EndDate == nil {
		endTime = startTime
	}

	// Group type - determine the date_trunc function to use
	switch params.Type {
	case "HALF_HOURLY":
		// PostgreSQL doesn't have native 30-minute trunc, so we'll use a formula
		truncFunc = "date_trunc('hour', s.completed_at + INTERVAL '5 hours') + INTERVAL '30 minutes' * FLOOR(EXTRACT(MINUTE FROM s.completed_at) / 30)"
		startTime = startTime.Truncate(30 * time.Minute)

	case "HOURLY":
		truncFunc = "date_trunc('hour', s.completed_at + INTERVAL '5 hours')"
		startTime = startTime.Truncate(time.Hour)

	case "DAILY":
		truncFunc = "date_trunc('day', s.completed_at + INTERVAL '5 hours')"
		startTime = time.Date(
			startTime.Year(), startTime.Month(), startTime.Day(),
			0, 0, 0, 0, startTime.Location(),
		)

	case "WEEKLY":
		truncFunc = "date_trunc('week', s.completed_at + INTERVAL '1 day' + INTERVAL '5 hours')"
		// Move to start of the week (Monday)
		weekday := int(startTime.Weekday()) - 1
		startTime = time.Date(
			startTime.Year(), startTime.Month(), startTime.Day()-weekday,
			0, 0, 0, 0, startTime.Location(),
		)

	case "MONTHLY":
		truncFunc = "date_trunc('month', s.completed_at + INTERVAL '5 hours')"
		startTime = time.Date(
			startTime.Year(), startTime.Month(), 2,
			0, 0, 0, 0, startTime.Location(),
		)

	case "YEARLY":
		truncFunc = "date_trunc('year', s.completed_at + INTERVAL '5 hours')"
		startTime = time.Date(
			startTime.Year(), 1, 1,
			0, 0, 0, 0, startTime.Location(),
		)

	default:
		truncFunc = "date_trunc('hour', s.completed_at + INTERVAL '5 hours')"
		startTime = startTime.Truncate(time.Hour)
	}

	args := []any{startTime, endTime}

	// Additional filters
	storeFilter := ""
	if len(params.StoreIds) > 0 {
		storeFilter = " AND s.store_id IN (?)"
		args = append(args, params.StoreIds)
	}

	companyFilter := ""
	if len(params.CompanyIds) > 0 {
		companyFilter = " AND st.company_id IN (?)"
		args = append(args, params.CompanyIds)
	}

	// Join stores only if company filter is needed
	storeJoin := ""
	if companyFilter != "" {
		storeJoin = "LEFT JOIN stores st ON s.store_id = st.id"
	}

	// Optimized query without time_series generation
	query := fmt.Sprintf(`
	SELECT
		%s - INTERVAL '5 hours' AS id,
		%s - INTERVAL '5 hours' AS created_at,
		COUNT(s.id) AS count,
		COALESCE(SUM(s.total_amount), 0) AS total_amount
	FROM sales s
	%s
	WHERE (s.completed_at + INTERVAL '5 hours') >= ?::timestamp
	  AND (s.completed_at + INTERVAL '5 hours') < ?::timestamp
	  AND s.stage IN (9, 11)
	  %s
	  %s
	GROUP BY %s
	ORDER BY id
	`, truncFunc, truncFunc, storeJoin, storeFilter, companyFilter, truncFunc)

	// Execute
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get chart info: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

// get dashboard top stores
func (s *Services) DashboardTopStores(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopStores, error) {
	order := utils.BuildTopStoreOrderClauseForDashBoard(params.Order)
	qb := s.db.WithContext(ctx).
		Select(
			"st.id AS id",
			"st.name AS name",
			"COUNT(s.id) AS count",
			"SUM(s.total_amount) AS total_amount",
		).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	// Store filter
	if len(params.StoreIds) > 0 {
		qb.Where("s.store_id IN (?)", params.StoreIds)
	}

	// Company filter
	if len(params.CompanyIds) > 0 {
		qb.Where("st.company_id IN (?)", params.CompanyIds)
	}

	// Limit & Offset
	qb = qb.Group("st.id").Order(order).Limit(params.Limit).Offset(params.Offset)

	// Execute query
	var res []domain.TopStores
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("Failed to get top stores: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard top products
func (s *Services) DashboardTopProducts(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopProducts, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"p.id AS id",
			"p.name AS name",
			"p.unit_per_pack AS unit_per_pack",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0)) / p.unit_per_pack AS count",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0)) % p.unit_per_pack AS unit_quantity",
			"(COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.unit_quantity) FILTER (WHERE s.stage = 11), 0))::NUMERIC / p.unit_per_pack AS quantity",
			"COALESCE(SUM(ci.total_price) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.total_price) FILTER (WHERE s.stage = 11), 0) AS total_amount",
			"COALESCE(SUM(ci.total_price - ci.discount_amount) FILTER (WHERE s.stage = 9), 0) - COALESCE(SUM(ci.total_price - ci.discount_amount) FILTER (WHERE s.stage = 11), 0) AS net_amount",
		).
		Table("cart_items ci").
		Joins("JOIN sales s ON s.id = ci.sale_id").
		Joins("JOIN products p ON p.id = ci.product_id").
		Where("s.stage IN(?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	// Store filter
	if len(params.StoreIds) > 0 {
		qb.Where("s.store_id IN (?)", params.StoreIds)
	}

	// Sorting (replaced switch)
	order := utils.BuildTopProductOrderClause(params.Order)

	qb = qb.Group("p.id").Order(order).Limit(params.Limit).Offset(params.Offset)
	var res []domain.TopProducts
	// Execute query
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard top products: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard bonus products
func (s *Services) DashboardBonusProducts(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.BonusProducts, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"p.id AS id",
			"p.name AS name",
			"p.unit_per_pack AS unit_per_pack",
			"SUM(eb.quantity) + SUM(eb.unit_quantity) / p.unit_per_pack AS count",
			"SUM(eb.unit_quantity) % p.unit_per_pack AS unit_quantity",
			"SUM(eb.bonus_amount) AS bonus_amount",
		).
		Table("employee_bonus eb").
		Joins("JOIN products p ON p.id = eb.product_id")

	if params.Search != "" {
		qb = qb.Where("p.name ILIKE ?", "%"+params.Search+"%")
	}

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("eb.created_at <= ?", params.EndDate.UTC())
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("p.company_id IN (?)", params.CompanyIds)
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Joins("JOIN employees e ON eb.employee_id = e.id").
			Where("e.store_id IN (?)", params.StoreIds)
	}

	qb = qb.Group("p.id")

	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get dashboard bonus products: %v", err)
		return nil, domain.InternalServerError
	}

	// New flexible order logic
	order := utils.BuildBonusProductOrderClause(params.Order)

	var res []domain.BonusProducts
	// Execute query
	err := qb.
		Order(order).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard bonus products: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get dashboard top seller
func (s *Services) DashboardTopSeller(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.TopSeller, error) {
	var (
		res  []domain.TopSeller
		args []any

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		startTimeStr       = startTimeInUTC.GetString()
		endTimeStr         = endTimeInUTC.GetString()
		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	// Main query
	query := `
	SELECT
		curr.id,
		curr.full_name,
		curr.store_name,
		curr.count,
		curr.total_amount,
		prev.total_amount AS previous_total_amount,
		ROUND(
			CASE 
				WHEN COALESCE(prev.total_amount, 0) = 0 THEN 100
				ELSE ((curr.total_amount - prev.total_amount) * 100.0) / NULLIF(prev.total_amount, 0)
			END, 2
		) AS percent,
		COUNT(*) OVER() AS total_count
	FROM (
		SELECT
			e.id,
			e.full_name,
			st.name AS store_name,
			st.company_id,
			COUNT(s.id) AS count,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		INNER JOIN stores st ON s.store_id = st.id
		WHERE s.stage IN(9, 11)
		AND s.sale_type = 'SALE'
		AND s.completed_at BETWEEN ? AND ?
		GROUP BY e.id, e.full_name, st.name, st.company_id
	) AS curr
	LEFT JOIN (
		SELECT
			e.id,
			SUM(s.total_amount) AS total_amount
		FROM sales s
		INNER JOIN employees e ON s.employee_id = e.id
		WHERE s.stage IN(9, 11)
		AND s.sale_type = 'SALE'
		AND s.completed_at BETWEEN ? AND ?
		GROUP BY e.id
	) AS prev ON curr.id = prev.id
`

	// First 4 args: 2 for current, 2 for previous range
	args = append(args,
		startTimeStr, endTimeStr,
		beforeStartTimeStr, beforeEndTimeStr,
	)

	// Optional filters
	where := " WHERE 1 = 1"
	if params.Search != "" {
		where += " AND curr.full_name ILIKE ?"
		args = append(args, "%"+params.Search+"%")
	}
	if len(params.CompanyIds) > 0 {
		where += " AND curr.company_id IN (?) "
		args = append(args, params.CompanyIds)
	}
	// check store_ids
	if len(params.StoreIds) > 0 {
		where += " AND curr.store_name IN (SELECT name FROM stores WHERE id IN (?))"
		args = append(args, params.StoreIds)
	}

	// Apply flexible ordering
	order := utils.BuildTopSellerOrderClause(params.Order)

	// Pagination
	limitOffset := " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	finalQuery := query + where + order + limitOffset

	// Execute
	err := s.db.WithContext(ctx).Raw(finalQuery, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get top seller: %v", err)
		return nil, domain.InternalServerError
	}

	return res, nil
}

// get payment
func (s *Services) DashboardPayments(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardPaymentDto, error) {
	var (
		res domain.DashboardPaymentDto

		startTimeInUTC = (*params.StartDate).ToUTC()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC()

		beforeStartTimeStr = startTimeInUTC.PrevDay().GetString()
		beforeEndTimeStr   = endTimeInUTC.PrevDay().GetString()
	)

	qb := s.db.WithContext(ctx).
		Select(
			"SUM(s.cash) AS cash",
			"COUNT(1) FILTER (WHERE s.cash > 0) AS cash_count",
			"SUM(s.humo) AS humo",
			"COUNT(1) FILTER (WHERE s.humo > 0) AS humo_count",
			"SUM(s.uzcard) AS uzcard",
			"COUNT(1) FILTER (WHERE s.uzcard > 0) AS uzcard_count",
			"SUM(s.click) AS click",
			"COUNT(1) FILTER (WHERE s.click > 0) AS click_count",
			"SUM(s.payme) AS payme",
			"COUNT(1) FILTER (WHERE s.payme > 0) AS payme_count",
			"SUM(s.alif) AS alif",
			"COUNT(1) FILTER (WHERE s.alif > 0) AS alif_count",
			"SUM(s.uzum) AS uzum",
			"COUNT(1) FILTER (WHERE s.uzum > 0) AS uzum_count",
			"SUM(s.uzum_tez_kor) AS uzum_tez_kor",
			"COUNT(1) FILTER (WHERE s.uzum_tez_kor > 0) AS uzum_tez_kor_count",
		).
		Table("sales s").
		Where("s.stage IN(?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	// filters
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}
	if len(params.CompanyIds) > 0 {
		qb = qb.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyIds)
	}

	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get dashboard payment stats: %v", err)
		return &res, domain.InternalServerError
	}

	// previus data
	var tmpPreviues struct {
		CashPrevius   float64 `gorm:"cash_previus"`
		HumoPrevius   float64 `gorm:"humo_previus"`
		UzcardPrevius float64 `gorm:"uzcard_previus"`
		ClickPrevius  float64 `gorm:"click_previus"`
		PaymePrevius  float64 `gorm:"payme_previus"`
		AlifPrevius   float64 `gorm:"alif_previus"`
		UzumPrevius   float64 `gorm:"uzum_previus"`
		UzumTRPrevius float64 `gorm:"uzum_tez_kor_previus"`
	}

	qbPrev := s.db.WithContext(ctx).
		Select(
			"SUM(s.cash) AS cash_previus",
			"SUM(s.humo) AS humo_previus",
			"SUM(s.uzcard) AS uzcard_previus",
			"SUM(s.click) AS click_previus",
			"SUM(s.payme) AS payme_previus",
			"SUM(s.alif) AS alif_previus",
			"SUM(s.uzum) AS uzum_previus",
			"SUM(s.uzum_tez_kor) AS uzum_tez_kor_previus",
		).
		Table("sales s").
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at between ? and ? ", beforeStartTimeStr, beforeEndTimeStr)

	// previus filter
	if len(params.StoreIds) > 0 {
		qbPrev = qbPrev.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qbPrev = qbPrev.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyIds)
	}

	err = qbPrev.Take(&tmpPreviues).Error
	if err != nil {
		s.log.Errorf("could not get dashboard payment stats for previus: %v", err)
		return &res, domain.InternalServerError
	}
	// cash
	if tmpPreviues.CashPrevius != 0 {
		res.CashPercent = math.Round((((res.Cash - tmpPreviues.CashPrevius) * 100) / tmpPreviues.CashPrevius) * 100)
	}
	// humo
	if tmpPreviues.HumoPrevius != 0 {
		res.HumoPercent = math.Round((((res.Humo - tmpPreviues.HumoPrevius) * 100) / tmpPreviues.HumoPrevius) * 100)
	}
	// uzcard
	if tmpPreviues.UzcardPrevius != 0 {
		res.UzcardPercent = math.Round((((res.Uzcard - tmpPreviues.UzcardPrevius) * 100) / tmpPreviues.UzcardPrevius) * 100)
	}
	// click
	if tmpPreviues.ClickPrevius != 0 {
		res.ClickPercent = math.Round((((res.Cash - tmpPreviues.ClickPrevius) * 100) / tmpPreviues.ClickPrevius) * 100)
	}
	// payme
	if tmpPreviues.PaymePrevius != 0 {
		res.PaymePercent = math.Round((((res.Cash - tmpPreviues.PaymePrevius) * 100) / tmpPreviues.PaymePrevius) * 100)
	}
	// alif
	if tmpPreviues.AlifPrevius != 0 {
		res.AlifPercent = math.Round((((res.Cash - tmpPreviues.AlifPrevius) * 100) / tmpPreviues.AlifPrevius) * 100)
	}
	// uzum
	if tmpPreviues.UzumPrevius != 0 {
		res.UzumPercent = math.Round((((res.Uzum - tmpPreviues.UzumPrevius) * 100) / tmpPreviues.UzumPrevius) * 100)
	}
	// uzum_tezkor
	if tmpPreviues.UzumTRPrevius != 0 {
		res.UzumTezkorPercent = math.Round((((res.UzumTezkor - tmpPreviues.UzumTRPrevius) * 100) / tmpPreviues.UzumTRPrevius) * 100)
	}

	return &res, nil
}

func (s *Services) DashboardTransaction(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.DashboardTransaction, error) {

	qb := s.db.WithContext(ctx).
		Select(
			"CASE s.stage WHEN 9 THEN 'Товары' WHEN 11 THEN 'Возвраты' END AS name",
			"SUM(ci.total_price) AS amount",
			"SUM(ci.unit_quantity / p.unit_per_pack) AS count",
		).
		Table("cart_items ci").
		Joins("JOIN sales s ON s.id = ci.sale_id").
		Joins("JOIN products p ON ci.product_id = p.id").
		Where("s.stage IN (?)", constants.FinishedSaleStages)

	if params.StartDate != nil && !params.StartDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at >= ?", params.StartDate.UTC())
	}

	if params.EndDate != nil && !params.EndDate.GetTime().IsZero() {
		qb = qb.Where("s.completed_at <= ?", params.EndDate.UTC())
	}

	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Joins("JOIN stores st ON s.store_id = st.id AND st.company_id IN(?)", params.CompanyIds)
	}
	qb = qb.Group("s.stage")
	var res = []domain.DashboardTransaction{}
	// Execute
	err := qb.Find(&res).Error
	if err != nil {
		s.log.Errorf("Error fetching dashboard transaction stats: %v", err)
		return res, domain.InternalServerError
	}

	return res, nil
}

func (s *Services) DashboardOldImports(ctx context.Context, params *domain.DashboardQueryParam) ([]domain.Import, int64, error) {
	var tmpImport []struct {
		Id                string     `gorm:"id"`
		PublicId          int        `gorm:"public_id"`
		StoreId           string     `gorm:"store_id"`
		CreatedBy         string     `gorm:"created_by"`
		AcceptedBy        string     `gorm:"accepted_by"`
		DocumentNumber    string     `gorm:"document_number"`
		DocumentYear      int        `gorm:"document_year"`
		Status            string     `gorm:"status"`
		ImportDate        *time.Time `gorm:"import_date"`
		AcceptedAmount    float64    `gorm:"accepted_amount"`
		ReceivedAmount    float64    `gorm:"received_amount"`
		ReceivedCount     float64    `gorm:"received_count"`
		AcceptedCount     float64    `gorm:"accepted_count"`
		AcceptedAmountVat float64    `gorm:"accepted_amount_vat"`
		ReceivedAmountVat float64    `gorm:"received_amount_vat"`
		CreatedAt         *time.Time `gorm:"created_at"`
		UpdatedAt         *time.Time `gorm:"updated_at"`
		StoreName         string     `gorm:"store_name"`
		CreatedByName     string     `gorm:"created_by_name"`
		AcceptedByName    string     `gorm:"accepted_by_name"`
	}

	// Fetch imports with detailed data
	qb := s.db.
		WithContext(ctx).
		Table("imports im").
		Joins("JOIN stores st ON st.id = im.store_id").
		Where("im.entry_type = ?", constants.ProductMovementImport).
		Where("im.created_at <= NOW() - interval '24 hours'")

	qb = qb.Where("im.status = ?", constants.GeneralStatusNew)

	if params.Search != "" {
		params.Search = fmt.Sprintf("%%%s%%", params.Search)
		qb = qb.Where("im.document_number ILIKE ? OR im.public_id::text LIKE ?", params.Search, params.Search)
	}
	if len(params.StoreIds) > 0 {
		qb = qb.Where("im.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN (?) ", params.CompanyIds)
	}
	var totalCount int64
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not get imports total_count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	err := qb.
		Select(
			"im.id",
			"im.public_id",
			"im.store_id",
			"im.name",
			"im.document_number",
			"im.document_year",
			"im.status",
			"im.import_date",
			"im.received_count AS received_count",
			"im.received_sum AS received_amount_vat",
			"im.scanned_count AS accepted_count",
			"im.scanned_sum AS accepted_amount_vat",
			"im.created_by",
			"im.accepted_by",
			"im.created_at",
			"im.updated_at",

			"st.name AS store_name",
		).
		Order("im.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&tmpImport).Error
	if err != nil {
		s.log.Errorf("could not get imports: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res = []domain.Import{}
	for _, item := range tmpImport {
		res = append(res, domain.Import{
			Id:                item.Id,
			PublicId:          item.PublicId,
			StoreId:           item.StoreId,
			DocumentNumber:    item.DocumentNumber,
			DocumentYear:      item.DocumentYear,
			Status:            item.Status,
			ReceivedCount:     item.ReceivedCount,
			ReceivedAmountVat: item.ReceivedAmountVat,
			AcceptedCount:     item.AcceptedCount,
			AcceptedAmountVat: item.AcceptedAmountVat,
			CreatedBy:         item.CreatedBy,
			AcceptedBy:        item.AcceptedBy,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
			ImportDate:        item.ImportDate,
			Store: domain.NewNullStruct(domain.ImportStore{
				Id:   item.StoreId,
				Name: item.StoreName,
			}, item.StoreId != ""),
			Sender: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.CreatedBy,
				FullName: item.CreatedByName,
			}, item.CreatedBy != ""),
			Receiver: domain.NewNullStruct(domain.ImportEmployee{
				Id:       item.AcceptedBy,
				FullName: item.AcceptedByName,
			}, item.AcceptedBy != ""),
		})
	}

	return res, totalCount, nil
}

// get dashboard count and amount data
func (s *Services) DashboardSaleStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardSaleStatistic, error) {
	var (
		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	qb := s.db.
		WithContext(ctx).
		Select(
			"COUNT(distinct s.id) filter ( where s.sale_type = 'SALE' ) AS sale_count",
			"SUM(CASE WHEN s.sale_type = 'SALE' THEN ci.total_price - ci.discount_amount ELSE (ci.total_price - ci.discount_amount) * (-1) END) AS sale_amount",
		).
		Table("sales s").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN stores st ON s.store_id = st.id").
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)

	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardSaleStatistic
	err := qb.Debug().Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get total sale amounts: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardNetProfitStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardCountStatsIncome, error) {
	var (
		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	qb := s.db.WithContext(ctx).
		Select(
			`ROUND(
                 SUM(
                     case
                         when s.sale_type = 'SALE'
                         then ((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity - ci.discount_amount
                         else (((ci.unit_price - sp.supply_price) / p.unit_per_pack) * ci.unit_quantity - ci.discount_amount)*(-1)
                     end
                 ), 2) AS income_amount`,
			`ROUND(
                 SUM(
                     case
                         when s.sale_type = 'SALE'
                         then (sp.supply_price / p.unit_per_pack) * ci.unit_quantity
                         else ((sp.supply_price / p.unit_per_pack) * ci.unit_quantity) * (-1)
                     end
                 ), 2) AS production_cost`,
		).
		Table("sales s").
		Joins("JOIN stores st ON s.store_id = st.id").
		Joins("JOIN cart_items ci ON s.id = ci.sale_id").
		Joins("JOIN store_products sp ON ci.store_product_id = sp.id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Where("s.stage IN(?)", constants.FinishedSaleStages).
		Where("s.completed_at BETWEEN ? AND ?", startTimeInUTC, endTimeInUTC)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		qb = qb.Where("s.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardCountStatsIncome
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get total income: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardImportStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardImportStatistic, error) {
	qb := s.db.WithContext(ctx).
		Select(
			"SUM(im.received_sum) AS import_amount",
			"SUM(CASE WHEN im.created_at < NOW() - interval '24 hour' THEN im.received_sum ELSE 0 END) AS expired_import_amount",
		).
		Table("imports im").
		Joins("JOIN stores st ON im.store_id = st.id").
		Where("im.entry_type = ?", constants.ProductMovementImport).
		Where("im.status = ?", constants.GeneralStatusNew)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		qb = qb.Where("im.store_id IN(?)", params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		qb = qb.Where("st.company_id IN(?)", params.CompanyIds)
	}

	var res domain.DashboardImportStatistic
	err := qb.Take(&res).Error
	if err != nil {
		s.log.Errorf("could not get import_count for_24: %v", err)
		return nil, domain.InternalServerError
	}

	res.NotLast24HImportAmount = res.ExpiredImportAmount

	return &res, nil
}

// get dashboard count and amount data
func (s *Services) DashboardProductStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardProductStatistic, error) {
	// declarations
	var res domain.DashboardProductStatistic

	// queries
	var (
		args []any
		// get sale stats information
		query = `
		SELECT
			ROUND(SUM(sp.unit_quantity), 2) AS total_product_count,
			ROUND(SUM((retail_price / p.unit_per_pack) * sp.unit_quantity), 2) AS stock_total_amount,
			ROUND(SUM(CASE WHEN expire_date BETWEEN NOW() AND NOW() + INTERVAL '3 month' THEN sp.unit_quantity ELSE 0 END), 2) AS expiring_soon_count,
			ROUND(SUM(CASE WHEN expire_date BETWEEN NOW() AND NOW() + INTERVAL '3 month' THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END), 2) AS expiring_soon_amount,
			ROUND(SUM(CASE WHEN expire_date < NOW() THEN sp.unit_quantity ELSE 0 END), 2) AS expired_soon_count,
			ROUND(SUM(CASE WHEN expire_date < NOW() THEN ((retail_price/p.unit_per_pack) * sp.unit_quantity) ELSE 0 END),2) AS expired_soon_amount
		FROM store_products sp
			JOIN products p ON sp.product_id = p.id
			JOIN stores st ON sp.store_id = st.id
		WHERE sp.unit_quantity > 0
		`
		filter = ""
	)

	// filter by several store ids
	if len(params.StoreIds) > 0 {
		filter += " AND sp.store_id IN (?)"
		args = append(args, params.StoreIds)
	}

	if len(params.CompanyIds) > 0 {
		filter += " AND st.company_id IN(?)"
		args = append(args, params.CompanyIds)
	}

	// Execute queries
	// get total product count
	query += filter
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get total product_amounts: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) LoyaltyCardStatistic(ctx context.Context, params *domain.DashboardQueryParam) (*domain.DashboardLoyaltyCardStatistic, error) {
	var (
		res domain.DashboardLoyaltyCardStatistic

		startTimeInUTC = (*params.StartDate).ToUTC().GetString()
		endTimeInUTC   = domain.AddDefaultDuration(*params.StartDate, params.EndDate).ToUTC().GetString()
	)

	var query = `
	select
		sum(case when loyalty_card_barcode is not null THEN 1 ELSE 0 END) as total_loyalty_card_count,
		sum(balance) as total_loyalty_card_balance,
		sum(case when loyalty_card_created_at between ? and ? then 1 else 0 end) as today_created_loyalty_card_count
	from
		customers
	`

	err := s.db.WithContext(ctx).Raw(query, startTimeInUTC, endTimeInUTC).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not get loyalty card stats: %v", err)
		return nil, domain.InternalServerError
	}

	return &res, nil
}
