package services

import (
	"context"
	"errors"

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

func (s *Services) GetLoyaltyCardDashboard(ctx context.Context, req *domain.LoyaltyCardDashboardRequest) (*domain.LoyaltyCardDashboard, error) {
	var result domain.LoyaltyCardDashboard

	var totalCashback float64
	err := s.db.Raw(`
		SELECT COALESCE(SUM(s.total_amount * c.loyalty_card_percent / 100.0), 0) as total_cashback
		FROM sales s
		JOIN customers c ON c.id = s.customer_id
		WHERE c.loyalty_card_barcode IS NOT NULL
			AND c.loyalty_card_barcode != ''
	`).Scan(&totalCashback).Error
	if err != nil {
		s.log.Errorf("error getting total cashback: %s", err.Error())
		return nil, domain.InternalServerError
	}
	result.TotalCashbackGiven = totalCashback

	var totalCards int64
	err = s.db.Raw(`
		SELECT COUNT(*)
		FROM customers
		WHERE loyalty_card_barcode IS NOT NULL
			AND loyalty_card_barcode != ''
	`).Scan(&totalCards).Error
	if err != nil {
		s.log.Errorf("error getting total cards: %s", err.Error())
		return nil, domain.InternalServerError
	}
	result.TotalCards = totalCards

	var newCards int64
	if req.StartDate != nil && req.EndDate != nil {
		err = s.db.Raw(`
			SELECT COUNT(*)
			FROM customers
			WHERE loyalty_card_barcode IS NOT NULL
				AND loyalty_card_barcode != ''
				AND loyalty_card_created_at >= ?
				AND loyalty_card_created_at <= ?
		`, *req.StartDate, *req.EndDate).Scan(&newCards).Error
		if err != nil {
			s.log.Errorf("error getting new cards in period: %s", err.Error())
			return nil, domain.InternalServerError
		}
	}
	result.NewCardsInPeriod = newCards

	var cardsByLevel []domain.LoyaltyCardByLevel
	err = s.db.Raw(`
		SELECT
			l.id as level_id,
			l.name as level_name,
			l.cashback_percent as percent,
			COUNT(c.id) as count
		FROM loyalty_card_levels l
		LEFT JOIN customers c ON c.loyalty_card_level_id = l.id
			AND c.loyalty_card_barcode IS NOT NULL
			AND c.loyalty_card_barcode != ''
		GROUP BY l.id, l.name, l.cashback_percent
		ORDER BY l.position
	`).Scan(&cardsByLevel).Error
	if err != nil {
		s.log.Errorf("error getting cards by level: %s", err.Error())
		return nil, domain.InternalServerError
	}
	result.CardsByLevel = cardsByLevel

	if req.IsLoyalty != nil && *req.IsLoyalty {
		if req.Limit <= 0 {
			req.Limit = 10
		}

		var customers []domain.Customer
		var count int64

		err = s.db.Raw(`
			SELECT COUNT(*) FROM customers
			WHERE loyalty_card_barcode IS NOT NULL AND loyalty_card_barcode != ''
		`).Scan(&count).Error
		if err != nil {
			s.log.Errorf("error getting loyalty customers count: %s", err.Error())
			return nil, domain.InternalServerError
		}

		err = s.db.Raw(`
			SELECT * FROM customers
			WHERE loyalty_card_barcode IS NOT NULL AND loyalty_card_barcode != ''
			ORDER BY loyalty_card_created_at DESC
			LIMIT ? OFFSET ?
		`, req.Limit, req.Offset).Scan(&customers).Error
		if err != nil {
			s.log.Errorf("error getting loyalty customers: %s", err.Error())
			return nil, domain.InternalServerError
		}

		result.Customers = utils.ListResponse(customers, count, req.Limit, req.Offset)
	}

	return &result, nil
}

func (s *Services) GetLoyaltyCardTopCustomers(ctx context.Context, req *domain.LoyaltyCardTopRequest) ([]domain.LoyaltyCardTopCustomer, int64, error) {
	var customers []domain.LoyaltyCardTopCustomer
	var count int64

	if req.Limit <= 0 {
		req.Limit = 10
	}

	dateFilter := ""
	dateParams := []interface{}{}
	if req.StartDate != nil && req.EndDate != nil {
		dateFilter = "AND c.loyalty_card_created_at >= ? AND c.loyalty_card_created_at <= ?"
		dateParams = append(dateParams, *req.StartDate, *req.EndDate)
	}

	countQuery := `
		SELECT COUNT(*)
		FROM customers c
		WHERE c.loyalty_card_barcode IS NOT NULL
			AND c.loyalty_card_barcode != ''
			` + dateFilter + `
	`

	err := s.db.Raw(countQuery, dateParams...).Scan(&count).Error
	if err != nil {
		s.log.Errorf("error getting top loyalty card customers count: %s", err.Error())
		return nil, 0, domain.InternalServerError
	}

	queryParams := append(dateParams, req.Limit, req.Offset)

	query := `
		WITH sales_agg AS (
			SELECT
				customer_id,
				COALESCE(SUM(total_amount), 0) AS total_spent
			FROM sales
			GROUP BY customer_id
		)
		SELECT
			c.id AS customer_id,
			c.public_id,
			c.full_name,
			c.phone,
			c.loyalty_card_barcode,
			l.name AS loyalty_card_level_name,
			c.loyalty_card_percent,
			COALESCE(sa.total_spent, 0) AS total_spent,
			COALESCE(sa.total_spent * c.loyalty_card_percent / 100.0, 0) AS total_cashback_earned,
			c.loyalty_card_created_at
		FROM customers c
		LEFT JOIN sales_agg sa ON sa.customer_id = c.id
		LEFT JOIN loyalty_card_levels l ON l.id = c.loyalty_card_level_id
		WHERE c.loyalty_card_barcode IS NOT NULL
			AND c.loyalty_card_barcode != ''
			` + dateFilter + `
		ORDER BY total_cashback_earned DESC
		LIMIT ? OFFSET ?
	`

	err = s.db.Raw(query, queryParams...).Scan(&customers).Error
	if err != nil {
		s.log.Errorf("error getting top loyalty card customers: %s", err.Error())
		return nil, 0, domain.InternalServerError
	}

	return customers, count, nil
}

