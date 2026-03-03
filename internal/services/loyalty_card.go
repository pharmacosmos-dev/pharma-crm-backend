package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

func (s *Services) CreateLoyaltyCard(req *domain.LoyaltyCardCreateRequest) (*domain.Customer, error) {
	var (
		res                domain.Customer
		loyaltyCardBarcode string
		loyaltyCardType    string // "physical" // virtual

		loyaltyCardPersent int64
		loyaltyCardLevelID string
	)

	// generate virtual loyalty card
	if req.VirtualLoyaltyCardNeeded {
		loyaltyCardBarcode = utils.GenerateBarcode()
		loyaltyCardType = "virtual"
	} else if *req.LoyaltyCardBarcode != "" {
		loyaltyCardBarcode = *req.LoyaltyCardBarcode
		loyaltyCardType = "physical"
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// getting loyalty level
	var loyaltyLevel domain.LoyaltyCardLevel
	err := tx.Order("position ASC").First(&loyaltyLevel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = tx.Rollback()
			s.log.Error("could not find loyalty level for new customer")
			return &res, domain.NotFoundError
		}
		_ = tx.Rollback()
		s.log.Errorf("error on getting loyalty card level in db: %s", err.Error())
		return &res, domain.InternalServerError
	}

	loyaltyCardLevelID = loyaltyLevel.Id
	loyaltyCardPersent = int64(loyaltyLevel.CashbackPercent)

	// writing loyalty card history
	err = tx.Exec(`insert into loyalty_card_levelup_history(
		customer_id, loyalty_card_level_id, total_spent
	) values (
			?, ?, ?
	)`, req.CustomerID, loyaltyCardLevelID, 0).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("error on creating loyalty card levelup history: %s", err.Error())
		return &res, domain.InternalServerError
	}

	// add to loyalty card to customer
	err = tx.Raw(`
	UPDATE customers
	SET
		loyalty_card_barcode = ?,
		loyalty_card_percent = ?,
		loyalty_card_level_id = ?,
		loyalty_card_type = ?,
		loyalty_card_created_by = ?,
		loyalty_card_created_at = NOW()
	WHERE
		id = ?
	RETURNING *`,
		loyaltyCardBarcode,
		loyaltyCardPersent,
		loyaltyCardLevelID,
		loyaltyCardType,
		req.LoyaltyCardCreatedBy,
		req.CustomerID,
	).Scan(&res).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("error on adding loyalty card to customer: %s", err.Error())
		return &res, domain.InternalServerError
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("error on commit transaction: %s", err.Error())
		return nil, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) LoyaltyCardLevelingUp() {
	var customers []domain.Customer

	// create a transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// update loyalty leveling up for customers
	err := tx.Raw(`
		UPDATE customers c
		SET
			loyalty_card_level_id = sub.level_id,
			loyalty_card_percent = sub.percent
		FROM (
			SELECT
				c.id AS customer_id,
				l.id AS level_id,
				l.cashback_percent AS percent
			FROM
				customers c
			JOIN LATERAL (
				SELECT
					l.id,
					l.cashback_percent
				FROM
					loyalty_card_levels l
				WHERE l.min_spent <= (
					SELECT COALESCE(SUM(s.total_amount), 0)
					FROM sales s
					WHERE s.customer_id = c.id
				)
				ORDER BY l.min_spent DESC
				LIMIT 1
			) l ON TRUE
		) AS sub
		WHERE
			c.id = sub.customer_id
			AND (c.loyalty_card_barcode IS NOT NULL AND c.loyalty_card_barcode != '')
			AND c.loyalty_card_level_id != sub.level_id
		RETURNING
		c.id,
		c.loyalty_card_level_id
	`).Scan(&customers).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Error("error on updating loyalty leveling up for customers: ", err)
		return
	}

	// add leveling up logs to leveling up history table
	for _, customer := range customers {
		err = tx.Model(&domain.LoyaltyCardLevelupHistory{}).Create(map[string]interface{}{
			"customer_id":           customer.Id,
			"loyalty_card_level_id": customer.LoyaltyCardLevelId,
			"total_spent":           gorm.Expr("(SELECT COALESCE(SUM(s.total_amount), 0) FROM sales s WHERE s.customer_id = ?)", customer.Id),
		}).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Error("error on creating loyalty card leveling up history: ", err)
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
		_ = tx.Rollback()
		s.log.Errorf("error on commit transaction: %v", err)
		return
	}

	s.log.Infof("loyalty card leveling up process completed successfully for: %d customers\n", len(customers))
}

func (s *Services) GetLoyaltyCardStats(
	ctx context.Context,
	req *domain.LoyaltyCardDashboardRequest,
) (*domain.LoyaltyCardDashboard, error) {

	var result domain.LoyaltyCardDashboard

	// =========================
	// Common date helpers
	// =========================
	var startDate, endDate *time.Time

	if req.StartDate != nil && !req.StartDate.GetTime().IsZero() {
		t := req.StartDate.UTC()
		startDate = &t
	}
	if req.EndDate != nil && !req.EndDate.GetTime().IsZero() {
		t := req.EndDate.UTC()
		endDate = &t
	}

	// ============================================================
	// 1️⃣ TOTAL CASHBACK (sales.completed_at bo‘yicha)
	// ============================================================
	cashbackQB := s.db.WithContext(ctx).
		Table("sales s").
		Joins("JOIN customers c ON c.id = s.customer_id").
		Where("c.loyalty_card_barcode IS NOT NULL AND c.loyalty_card_barcode != ''")

	if startDate != nil {
		cashbackQB = cashbackQB.Where("s.completed_at >= ?", *startDate)
	}
	if endDate != nil {
		cashbackQB = cashbackQB.Where("s.completed_at <= ?", *endDate)
	}

	if err := cashbackQB.
		Select("COALESCE(SUM(s.total_amount * c.loyalty_card_percent / 100.0), 0)").
		Scan(&result.TotalCashbackGiven).Error; err != nil {

		s.log.Errorf("GetLoyaltyCardStats: cashback error: %v", err)
		return nil, domain.InternalServerError
	}

	// ============================================================
	// 2️⃣ TOTAL CARDS (umumiy)
	// ============================================================
	if err := s.db.WithContext(ctx).
		Table("customers c").
		Where("c.loyalty_card_barcode IS NOT NULL AND c.loyalty_card_barcode != ''").
		Count(&result.TotalCards).Error; err != nil {

		s.log.Errorf("GetLoyaltyCardStats: total cards error: %v", err)
		return nil, domain.InternalServerError
	}

	// ============================================================
	// 3️⃣ NEW CARDS IN PERIOD (loyalty_card_created_at)
	// ============================================================
	newCardsQB := s.db.WithContext(ctx).
		Table("customers c").
		Where("c.loyalty_card_barcode IS NOT NULL AND c.loyalty_card_barcode != ''")

	if startDate != nil {
		newCardsQB = newCardsQB.Where("c.loyalty_card_created_at >= ?", *startDate)
	}
	if endDate != nil {
		newCardsQB = newCardsQB.Where("c.loyalty_card_created_at <= ?", *endDate)
	}

	if err := newCardsQB.Count(&result.NewCardsInPeriod).Error; err != nil {
		s.log.Errorf("GetLoyaltyCardStats: new cards error: %v", err)
		return nil, domain.InternalServerError
	}

	// ============================================================
	// 4️⃣ CARDS BY LEVEL (period bo‘yicha)
	// ============================================================
	levelQuery := `
		SELECT
			l.id   AS level_id,
			l.name AS level_name,
			l.cashback_percent AS percent,
			COUNT(c.id) AS count
		FROM loyalty_card_levels l
		LEFT JOIN customers c
			ON c.loyalty_card_level_id = l.id
			AND c.loyalty_card_barcode IS NOT NULL
			AND c.loyalty_card_barcode != ''
	`

	var args []interface{}
	var joinConditions []string

	if startDate != nil {
		joinConditions = append(joinConditions, "c.loyalty_card_created_at >= ?")
		args = append(args, *startDate)
	}
	if endDate != nil {
		joinConditions = append(joinConditions, "c.loyalty_card_created_at <= ?")
		args = append(args, *endDate)
	}

	if len(joinConditions) > 0 {
		levelQuery += " AND " + strings.Join(joinConditions, " AND ")
	}

	levelQuery += `
		GROUP BY l.id, l.name, l.cashback_percent, l.position
		ORDER BY l.position
	`

	var cardsByLevel []domain.LoyaltyCardByLevel

	if err := s.db.WithContext(ctx).
		Raw(levelQuery, args...).
		Scan(&cardsByLevel).Error; err != nil {

		s.log.Errorf("GetLoyaltyCardStats: levels error: %v", err)
		return nil, domain.InternalServerError
	}

	result.CardsByLevel = cardsByLevel

	return &result, nil
}

func (s *Services) GetLoyaltyCardTopCustomers(
	ctx context.Context,
	req *domain.LoyaltyCardTopRequest,
) ([]domain.LoyaltyCardTopCustomer, int64, error) {

	var customers []domain.LoyaltyCardTopCustomer
	var count int64


	// =========================
	// Prepare date filters for sales
	// =========================
	var salesFilters []string
	var queryParams []interface{}

	if req.StartDate != nil && !req.StartDate.GetTime().IsZero() {
		salesFilters = append(salesFilters, "s.completed_at >= ?")
		queryParams = append(queryParams, req.StartDate.UTC())
	}
	if req.EndDate != nil && !req.EndDate.GetTime().IsZero() {
		salesFilters = append(salesFilters, "s.completed_at <= ?")
		queryParams = append(queryParams, req.EndDate.UTC())
	}

	salesWhere := ""
	if len(salesFilters) > 0 {
		salesWhere = "WHERE " + strings.Join(salesFilters, " AND ")
	}

	// =========================
	// Count total customers in period
	// =========================
	countQuery := fmt.Sprintf(`
		WITH sales_agg AS (
			SELECT customer_id
			FROM sales s
			%s
			GROUP BY customer_id
		)
		SELECT COUNT(*)
		FROM sales_agg sa
		JOIN customers c ON c.id = sa.customer_id
		WHERE c.loyalty_card_barcode IS NOT NULL
		  AND c.loyalty_card_barcode != ''
	`, salesWhere)

	if err := s.db.Raw(countQuery, queryParams...).Scan(&count).Error; err != nil {
		s.log.Errorf("GetTopCustomers count error: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// =========================
	// Get top customers with total cashback
	// =========================
	queryParamsWithLimit := append(queryParams, req.Limit, req.Offset)

	mainQuery := fmt.Sprintf(`
		WITH sales_agg AS (
			SELECT
				s.customer_id,
				COALESCE(SUM(s.total_amount), 0) AS total_spent
			FROM sales s
			%s
			GROUP BY s.customer_id
		)
		SELECT
			c.id AS customer_id,
			c.public_id,
			c.full_name,
			c.phone,
			c.loyalty_card_barcode,
			l.name AS loyalty_card_level_name,
			c.loyalty_card_percent,
			sa.total_spent,
			COALESCE(sa.total_spent * c.loyalty_card_percent / 100.0, 0) AS total_cashback_earned,
			c.loyalty_card_created_at
		FROM sales_agg sa
		JOIN customers c ON c.id = sa.customer_id
		LEFT JOIN loyalty_card_levels l ON l.id = c.loyalty_card_level_id
		WHERE c.loyalty_card_barcode IS NOT NULL
		  AND c.loyalty_card_barcode != ''
		ORDER BY total_cashback_earned DESC
		LIMIT ? OFFSET ?
	`, salesWhere)

	if err := s.db.Raw(mainQuery, queryParamsWithLimit...).Scan(&customers).Error; err != nil {
		s.log.Errorf("GetTopCustomers data error: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return customers, count, nil
}

