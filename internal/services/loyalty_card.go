package services

import (
	"errors"
	"fmt"

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
	fmt.Println(req.CustomerID, loyaltyCardLevelID)

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
		s.log.Error("error on updating loyalty leveling up for customers: ", err)
		_ = tx.Rollback()
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
			s.log.Error("error on creating loyalty card leveling up history: ", err)
			_ = tx.Rollback()
			return
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Error("error on commit transaction: ", err)
		_ = tx.Rollback()
		return
	}

	s.log.Infof("loyalty card leveling up process completed successfully for: %d customers\n", len(customers))
}
