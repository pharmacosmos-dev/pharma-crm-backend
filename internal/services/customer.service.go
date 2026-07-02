package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// region Create

func (s *Services) CreateCustomer(ctx context.Context, req *domain.CustomerRequest) (*domain.Customer, error) {
	var (
		res domain.Customer

		loyaltyCardBarcode sql.NullString
		loyaltyCardType    sql.NullString // "physical" // virtual

		loyaltyCardPersent      sql.NullInt64
		loyaltyCardLevelID      sql.NullString
		loyaltyCardShouldCreate bool = req.VirtualLoyaltyCardNeeded || *req.LoyaltyCardBarcode != ""
		loyaltyCardCreatedBy    sql.NullString
		loyaltyCardCreatedAt    sql.NullTime
	)

	// generate virtual loyalty card
	if req.VirtualLoyaltyCardNeeded {
		loyaltyCardBarcode = sql.NullString{String: utils.GenerateBarcode(), Valid: true}
		loyaltyCardType = sql.NullString{String: "virtual", Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	} else if *req.LoyaltyCardBarcode != "" {
		var count int64
		err := s.db.WithContext(ctx).Table("customers").Where("loyalty_card_barcode = ?", *req.LoyaltyCardBarcode).Count(&count).Error
		if err != nil {
			return &res, fmt.Errorf("error on checking loyalty card barcode: %s", err.Error())
		}
		if count > 0 {
			return &res, domain.DuplicateLoyaltyCardError
		}
		loyaltyCardBarcode = sql.NullString{String: *req.LoyaltyCardBarcode, Valid: true}
		loyaltyCardType = sql.NullString{String: "physical", Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// check if phone belongs to a cashier employee
	var cashierCount int64
	if err := s.db.WithContext(ctx).Table("employees e").
		Joins("JOIN employee_roles er ON er.employee_id = e.id").
		Joins("JOIN roles r ON r.id = er.role_id").
		Where("e.phone = ? AND r.name IN ?", req.Phone, []string{"Кассир", "Кассир Франшиза"}).
		Count(&cashierCount).Error; err != nil {
		return &res, domain.InternalServerError
	}
	if cashierCount > 0 {
		return &res, domain.CanNotCreateYourselfError
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// getting loyalty level
	if loyaltyCardShouldCreate {
		var loyaltyLevel domain.LoyaltyCardLevel
		err := tx.Order("position ASC").First(&loyaltyLevel).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = tx.Rollback()
				s.log.Error("could not find loyalty level for new customer")
				return &res, fmt.Errorf("could not find loyalty level for new customer: %s", err.Error())
			}
			_ = tx.Rollback()
			s.log.Errorf("error on getting loyalty card level in db: %s", err.Error())
			return &res, fmt.Errorf("error on getting loyalty card level in db: %s", err.Error())
		}

		loyaltyCardLevelID = sql.NullString{String: loyaltyLevel.Id, Valid: true}
		loyaltyCardPersent = sql.NullInt64{Int64: int64(loyaltyLevel.CashbackPercent), Valid: true}
		loyaltyCardCreatedBy = sql.NullString{String: req.CreatedBy, Valid: true}
	}

	query := `
	INSERT INTO customers (
		id, 
		store_id, 
		tag_id, 
		first_name, 
		last_name, 
		full_name, 
		phone, 
		gender, 
		birthday, 
		created_by,
		discount_card,
		discount_percent,
		loyalty_card_barcode,
		loyalty_card_percent,
		loyalty_card_level_id,
		loyalty_card_type,
		loyalty_card_created_by,
		loyalty_card_created_at
		)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
	RETURNING *
	`
	// insert customer
	err := tx.WithContext(ctx).
		Raw(query,
			uuid.New().String(),
			req.StoreId,
			req.TagId,
			req.FirstName,
			req.LastName,
			req.FirstName+" "+req.LastName,
			req.Phone,
			req.Gender,
			req.Birthday,
			req.CreatedBy,
			req.DiscountCard,
			req.DiscountPercent,
			loyaltyCardBarcode,
			loyaltyCardPersent,
			loyaltyCardLevelID,
			loyaltyCardType,
			loyaltyCardCreatedBy,
			loyaltyCardCreatedAt,
		).Scan(&res).Error
	if err != nil {
		var pgErr *pgconn.PgError
		// Try to unwrap GORM's error wrapper
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			_ = tx.Rollback()
			return &res, domain.DuplicatePhoneError
		}

		// Alternative: Check error string as fallback
		if strings.Contains(err.Error(), "23505") ||
			strings.Contains(err.Error(), "duplicate key") {
			_ = tx.Rollback()
			return &res, domain.DuplicatePhoneError
		}

		_ = tx.Rollback()
		s.log.Errorf("could not create customer: %v", err)
		return &res, domain.InternalServerError
	}

	// writing loyalty card history
	if loyaltyCardShouldCreate {
		err = tx.Exec(`insert into loyalty_card_levelup_history(
			customer_id, loyalty_card_level_id, total_spent
		) values (
				?, ?, ?
		)`, res.Id, loyaltyCardLevelID, 0).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("error on creating loyalty card levelup history: %s", err.Error())
			return &res, fmt.Errorf("error on creating loyalty card levelup history: %s", err.Error())
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("error on commit transaction: %s", err.Error())
		return nil, domain.InternalServerError
	}

	return &res, nil
}

// create new customer with phone and name
func (s *Services) CreateCustomerWithPhone(ctx context.Context, req *domain.NoorClientInfo) (*domain.Customer, error) {
	var res domain.Customer
	query := `
	INSERT INTO customers(
		first_name,
		full_name,
		phone
	)
	VALUES (?, ?, ?)
	RETURNING *
	`
	err := s.db.WithContext(ctx).Raw(query, req.Name, req.Name, req.Phone).Scan(&res).Error
	if err != nil {
		s.log.Errorf("could not create customer with phone: %v", err)
		return &res, domain.InternalServerError
	}
	return &res, nil
}

// region Get

// get customer list data
func (s *Services) GetCustomers(ctx context.Context, params *domain.QueryParam, usedInSalePage bool) ([]domain.Customer, int64, error) {
	type row struct {
		Id                   string     `gorm:"column:id"`
		PublicId             int        `gorm:"column:public_id"`
		StoreId              string     `gorm:"column:store_id"`
		TagId                string     `gorm:"column:tag_id"`
		FirstName            string     `gorm:"column:first_name"`
		LastName             string     `gorm:"column:last_name"`
		FullName             string     `gorm:"column:full_name"`
		Phone                string     `gorm:"column:phone"`
		Birthday             string     `gorm:"column:birthday"`
		Gender               string     `gorm:"column:gender"`
		Balance              float64    `gorm:"column:balance"`
		SpendingFromBalance  float64    `gorm:"column:spending_from_balance"`
		DiscountCard         string     `gorm:"column:discount_card"`
		DiscountPercent      int        `gorm:"column:discount_percent"`
		LoyaltyCardBarcode   string     `gorm:"column:loyalty_card_barcode"`
		LoyaltyCardPercent   int        `gorm:"column:loyalty_card_percent"`
		LoyaltyCardLevelId   string     `gorm:"column:loyalty_card_level_id"`
		LoyaltyCardType      string     `gorm:"column:loyalty_card_type"`
		LoyaltyCardCreatedBy string     `gorm:"column:loyalty_card_created_by"`
		LoyaltyCardCreatedAt *time.Time `gorm:"column:loyalty_card_created_at"`
		TelegramChatId       int64      `gorm:"column:telegram_chat_id"`
		CreatedAt            *time.Time `gorm:"column:created_at"`
		UpdatedAt            *time.Time `gorm:"column:updated_at"`
		IsActive             bool       `gorm:"column:is_active"`
		IsBlocked            bool       `gorm:"column:is_blocked"`
		SId                  string     `gorm:"column:s_id"`
		SName                string     `gorm:"column:s_name"`
		TId                  string     `gorm:"column:t_id"`
		TName                string     `gorm:"column:t_name"`
		SalesCount24h        int64      `gorm:"column:sales_count_24h"`
		MonthlySalesSum      float64    `gorm:"column:monthly_sales_sum"`
		MonthlySalesCount    int64      `gorm:"column:monthly_sales_count"`
	}

	// where shartlarini to'plovchi qism
	whereClauses := []string{"c.is_active = true"}
	var args []interface{}

	if params.Search != "" {
		if usedInSalePage {
			whereClauses = append(whereClauses, "(c.discount_card = ? OR c.loyalty_card_barcode = ?)")
			args = append(args, params.Search, params.Search)
		} else {
			whereClauses = append(whereClauses, "(c.public_id::text ILIKE ? OR c.phone::text ILIKE ? OR c.full_name ILIKE ?)")
			args = append(args, "%"+params.Search+"%", "%"+params.Search+"%", "%"+params.Search+"%")
		}
	}
	if params.StoreID != "" {
		whereClauses = append(whereClauses, "c.store_id = ?")
		args = append(args, params.StoreID)
	}
	if params.CompanyId != "" {
		whereClauses = append(whereClauses, "s.company_id = ?")
		args = append(args, params.CompanyId)
	}
	if params.IsBlocked != nil {
		whereClauses = append(whereClauses, "c.is_blocked = ?")
		args = append(args, *params.IsBlocked)
	}

	where := strings.Join(whereClauses, " AND ")

	// count
	var totalCount int64
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM customers c
		LEFT JOIN stores s ON c.store_id = s.id
		LEFT JOIN tags t ON c.tag_id = t.id
		WHERE %s
	`, where)
	if err := s.db.WithContext(ctx).Raw(countSQL, args...).Scan(&totalCount).Error; err != nil {
		s.log.Errorf("could not count customers: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// data
	var salesJoin string
	var salesCountField string
	var salesSumtMonth string
	var monthlySumField string
	if params.Search != "" {
		salesJoin = `LEFT JOIN (
			SELECT customer_id, COUNT(*) AS sales_count_24h
			FROM sales
			WHERE stage = 9 AND created_at >= CURRENT_DATE
			GROUP BY customer_id
		) sc ON sc.customer_id = c.id`
		salesCountField = "COALESCE(sc.sales_count_24h, 0) AS sales_count_24h"
		salesSumtMonth = `LEFT JOIN (
			SELECT customer_id, SUM(total_amount) AS monthly_sales_sum, COUNT(*) AS monthly_sales_count
			FROM sales
			WHERE stage = 9 AND completed_at >= NOW() - INTERVAL '1 month'
			GROUP BY customer_id
		) ms ON ms.customer_id = c.id`
		monthlySumField = "COALESCE(ms.monthly_sales_sum, 0) AS monthly_sales_sum, COALESCE(ms.monthly_sales_count, 0) AS monthly_sales_count"
	} else {
		salesCountField = "0 AS sales_count_24h"
		monthlySumField = "0 AS monthly_sales_sum, 0 AS monthly_sales_count"
	}

	dataArgs := append(args, params.Limit, params.Offset)
	dataSQL := fmt.Sprintf(`
		SELECT
			c.id, c.public_id, c.store_id, c.tag_id,
			c.first_name, c.last_name, c.full_name, c.phone,
			c.birthday, c.gender, c.balance, c.spending_from_balance,
			c.discount_card, c.discount_percent,
			c.loyalty_card_barcode, c.loyalty_card_percent,
			c.loyalty_card_level_id, c.loyalty_card_type,
			c.loyalty_card_created_by, c.loyalty_card_created_at,
			c.telegram_chat_id, c.created_at, c.updated_at, c.is_active, c.is_blocked,
			s.id AS s_id, s.name AS s_name,
			t.id AS t_id, t.name AS t_name,
			%s,
			%s
		FROM customers c
		LEFT JOIN stores s ON c.store_id = s.id
		LEFT JOIN tags t ON c.tag_id = t.id
		%s
		%s
		WHERE %s
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`, salesCountField, monthlySumField, salesJoin, salesSumtMonth, where)

	s.log.Infof("GetCustomers dataSQL: %s | args: %v", dataSQL, dataArgs)

	var rows []row
	if err := s.db.WithContext(ctx).Raw(dataSQL, dataArgs...).Scan(&rows).Error; err != nil {
		s.log.Errorf("could not get customers: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if len(rows) > 0 {
		s.log.Infof("GetCustomers first row sales_count_24h: %d", rows[0].SalesCount24h)
	}

	customers := make([]domain.Customer, 0, len(rows))
	for _, r := range rows {
		customers = append(customers, domain.Customer{
			Id:                   r.Id,
			PublicId:             r.PublicId,
			StoreId:              r.StoreId,
			TagId:                r.TagId,
			FirstName:            r.FirstName,
			LastName:             r.LastName,
			FullName:             r.FullName,
			Phone:                r.Phone,
			Birthday:             r.Birthday,
			Gender:               r.Gender,
			Balance:              r.Balance,
			SpendingFromBalance:  r.SpendingFromBalance,
			DiscountCard:         r.DiscountCard,
			DiscountPercent:      r.DiscountPercent,
			LoyaltyCardBarcode:   r.LoyaltyCardBarcode,
			LoyaltyCardPercent:   r.LoyaltyCardPercent,
			LoyaltyCardLevelId:   r.LoyaltyCardLevelId,
			LoyaltyCardType:      r.LoyaltyCardType,
			LoyaltyCardCreatedBy: r.LoyaltyCardCreatedBy,
			LoyaltyCardCreatedAt: r.LoyaltyCardCreatedAt,
			TelegramChatId:       r.TelegramChatId,
			CreatedAt:            r.CreatedAt,
			UpdatedAt:            r.UpdatedAt,
			IsActive:             r.IsActive,
			IsBlocked:            r.IsBlocked,
			SalesCount24h:        r.SalesCount24h,
			MonthlySalesSum:      r.MonthlySalesSum,
			MonthlySalesCount:    r.MonthlySalesCount,
			Store: &domain.Store{
				Id:   r.SId,
				Name: r.SName,
			},
			Tag: &domain.Tag{
				Id:   r.TId,
				Name: r.TName,
			},
		})
	}

	return customers, totalCount, nil
}

func (s *Services) GetCustomerById(ctx context.Context, tx *gorm.DB, id string) (*domain.Customer, error) {
	var res domain.Customer

	err := tx.WithContext(ctx).
		Where("id = ? AND is_active = true", id).
		First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError
		}
		s.log.Errorf("could not get customer: %v", err)
		return nil, domain.InternalServerError
	}

	var tag domain.Tag
	err = s.db.WithContext(ctx).First(&tag, "id = ?", res.TagId).Error
	if err != nil {
		s.log.Errorf("could not get customer tag: %v", err)
	}

	res.Tag = &tag

	return &res, nil
}

func (s *Services) ListDiscountCards(ctx context.Context, params *domain.QueryParam) ([]domain.DiscountCardWithCustomer, int64, error) {
	var (
		res        []domain.DiscountCardWithCustomer
		totalCount int64
	)

	// Start building the query
	query := s.db.
		Select(
			"c.id",
			"c.store_id",
			"c.tag_id",
			"c.full_name",
			"c.phone",
			"c.birthday",
			"c.gender",
			"c.balance",
			"c.discount_card AS barcode",
			"c.discount_percent AS percent",
			"c.created_at",

			"s.name AS store_name",
			"t.name AS tag_name",
		).Table("customers c").
		Joins("LEFT JOIN stores s ON c.store_id = s.id").
		Joins("LEFT JOIN tags t ON c.tag_id = t.id")

	if params.Search != "" {
		search := fmt.Sprintf("%%%s%%", params.Search)
		query = query.Where("c.discount_card LIKE ? OR c.full_name ILIKE ? OR c.phone LIKE ?", search, search, search)
	}

	if params.StoreID != "" {
		query = query.Where("c.store_id = ?", params.StoreID)
	}

	if params.CompanyId != "" {
		query = query.Where("s.company_id = ? ", params.CompanyId)
	}

	err := query.WithContext(ctx).
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("c.created_at DESC").
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not create new customer: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}

// get or create customer by existing phone
func (s *Services) GetOrCreateCustomerByPhone(ctx context.Context, req *domain.NoorClientInfo) (*domain.Customer, error) {
	var customer *domain.Customer
	req.Phone = utils.NormalizePhoneNumber(req.Phone)

	err := s.db.WithContext(ctx).Take(&customer, "phone = ?", req.Phone).Error

	// If record found, return it
	if err == nil {
		return customer, nil
	}

	// If record not found, create new customer
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.CreateCustomerWithPhone(ctx, req)
	}

	// Any other database error
	s.log.Errorf("could not get customer on creating online sale: %v", err)

	return nil, domain.InternalServerError
}

func (s *Services) CreateCustomerDiscountCard(ctx context.Context, req *domain.CreateDiscountCardRequest) (*domain.DiscountCard, error) {
	// check discount card is exists
	if s.checkDiscountCardExists(ctx, req.Barcode) {
		return nil, domain.DuplicateError
	}
	// check customer is exists
	var count int64
	err := s.db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("id = ?", req.CustomerId).
		Count(&count).Error
	if err != nil {
		s.log.Errorf("could not count customers: %v", err)
		return nil, domain.InternalServerError
	}
	if count == 0 {
		return nil, domain.NotFoundError
	}

	var customer domain.Customer
	err = s.db.WithContext(ctx).
		Raw("UPDATE customers SET discount_card = ?, discount_percent = ? WHERE id = ? RETURNING *",
			req.Barcode, req.Percent, req.CustomerId).Scan(&customer).Error
	if err != nil {
		s.log.Errorf("could not attach discount_card to customer: %v", err)
		return nil, domain.InternalServerError
	}

	return &domain.DiscountCard{
		Id:         customer.Id,
		CustomerId: customer.Id,
		Barcode:    customer.DiscountCard,
		Percent:    customer.DiscountPercent,
	}, nil
}

func (s *Services) checkDiscountCardExists(ctx context.Context, discountCard string) bool {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&domain.Customer{}).
		Where("discount_card = ?", discountCard).
		Count(&count).Error
	if err != nil {
		s.log.Errorf("could not check discount card existence: %v", err)
		return false
	}
	return count > 0
}

// region Update

func (s *Services) UpdateCustomer(ctx context.Context, req *domain.CustomerRequest) (*domain.Customer, error) {
	var (
		res domain.Customer

		loyaltyCardBarcode sql.NullString
		loyaltyCardType    sql.NullString // "physical" // virtual

		loyaltyCardPersent      sql.NullInt64
		loyaltyCardLevelID      sql.NullString
		loyaltyCardShouldCreate bool = req.VirtualLoyaltyCardNeeded || *req.LoyaltyCardBarcode != ""
		loyaltyCardCreatedBy    sql.NullString
		loyaltyCardCreatedAt    sql.NullTime
	)

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// getting existing customer for checking loyalty card exists
	var existingCustomer domain.Customer
	err := tx.WithContext(ctx).
		Where("id = ?", req.Id).
		First(&existingCustomer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = tx.Rollback()
			s.log.Error("could not find customer to update")
			return &res, domain.NotFoundError
		}
		_ = tx.Rollback()
		s.log.Errorf("error on getting existing customer in db: %s", err.Error())
		return &res, domain.InternalServerError
	}

	if existingCustomer.LoyaltyCardBarcode != "" {
		loyaltyCardShouldCreate = false
	}

	if req.LoyaltyCardBarcode == &existingCustomer.LoyaltyCardBarcode {
		return nil, domain.DuplicateLoyaltyCardError
	}

	// generate virtual loyalty card
	if req.VirtualLoyaltyCardNeeded && loyaltyCardShouldCreate {
		loyaltyCardBarcode = sql.NullString{String: utils.GenerateBarcode(), Valid: true}
		loyaltyCardType = sql.NullString{String: "virtual", Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	} else if *req.LoyaltyCardBarcode != "" && loyaltyCardShouldCreate {
		loyaltyCardBarcode = sql.NullString{String: *req.LoyaltyCardBarcode, Valid: true}
		loyaltyCardType = sql.NullString{String: "physical", Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	} else if !loyaltyCardShouldCreate {
		loyaltyCardBarcode = sql.NullString{String: existingCustomer.LoyaltyCardBarcode, Valid: true}
		loyaltyCardType = sql.NullString{String: existingCustomer.LoyaltyCardType, Valid: true}
		loyaltyCardPersent = sql.NullInt64{Int64: int64(existingCustomer.LoyaltyCardPercent), Valid: true}
		loyaltyCardLevelID = sql.NullString{String: existingCustomer.LoyaltyCardLevelId, Valid: true}
		loyaltyCardCreatedBy = sql.NullString{String: existingCustomer.LoyaltyCardCreatedBy, Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: *existingCustomer.LoyaltyCardCreatedAt, Valid: true}
	}

	// getting loyalty level
	if loyaltyCardShouldCreate {
		var loyaltyLevel domain.LoyaltyCardLevel
		err := tx.Order("position ASC").First(&loyaltyLevel).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = tx.Rollback()
				s.log.Error("could not find loyalty level for new customer")
				return &res, domain.InternalServerError
			}
			_ = tx.Rollback()
			s.log.Errorf("error on getting loyalty card level in db: %s", err.Error())
			return &res, domain.InternalServerError
		}

		loyaltyCardLevelID = sql.NullString{String: loyaltyLevel.Id, Valid: true}
		loyaltyCardPersent = sql.NullInt64{Int64: int64(loyaltyLevel.CashbackPercent), Valid: true}
		loyaltyCardCreatedBy = sql.NullString{String: req.CreatedBy, Valid: true}
	}

	query := `
	UPDATE customers
	SET
		store_id = ?,
		tag_id = ?,
		first_name = ?,
		last_name = ?,
		full_name = ?,
		phone = ?,
		gender = ?,
		birthday = ?,
		created_by = ?,
		discount_card = ?,
		discount_percent = ?,
		loyalty_card_barcode = ?,
		loyalty_card_percent = ?,
		loyalty_card_level_id = ?,
		loyalty_card_type = ?,
		loyalty_card_created_by = ?,
		loyalty_card_created_at = ?,
		updated_at = now()
	WHERE
		id = ?
	RETURNING *
	`
	// insert customer
	err = tx.WithContext(ctx).
		Raw(query,
			req.StoreId,
			req.TagId,
			req.FirstName,
			req.LastName,
			req.FirstName+" "+req.LastName,
			req.Phone,
			req.Gender,
			req.Birthday,
			req.CreatedBy,
			req.DiscountCard,
			req.DiscountPercent,
			loyaltyCardBarcode,
			loyaltyCardPersent,
			loyaltyCardLevelID,
			loyaltyCardType,
			loyaltyCardCreatedBy,
			loyaltyCardCreatedAt,
			req.Id,
		).Scan(&res).Error
	if err != nil {
		var pgErr *pgconn.PgError
		// Try to unwrap GORM's error wrapper
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			_ = tx.Rollback()
			return &res, domain.DuplicatePhoneError
		}

		// Alternative: Check error string as fallback
		if strings.Contains(err.Error(), "23505") ||
			strings.Contains(err.Error(), "duplicate key") {
			_ = tx.Rollback()
			return &res, domain.DuplicatePhoneError
		}

		_ = tx.Rollback()
		s.log.Errorf("could not update customer: %v", err)
		return &res, domain.InternalServerError
	}

	// writing loyalty card history
	if loyaltyCardShouldCreate {
		err = tx.WithContext(ctx).
			Exec(`insert into loyalty_card_levelup_history(
			customer_id, loyalty_card_level_id, total_spent
		) values (
				?, ?, ?
		)`, res.Id, loyaltyCardLevelID, 0).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("error on creating loyalty card levelup history: %s", err.Error())
			return &res, domain.InternalServerError
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("error on commit transaction: %s", err.Error())
		return nil, domain.InternalServerError
	}

	return &res, nil
}

func (s *Services) UpdateCustomerIsBlocked(ctx context.Context, customerID string, isBlocked bool) error {
	err := s.db.WithContext(ctx).Exec(
		`UPDATE customers SET is_blocked = ?, updated_at = NOW() WHERE id = ?`,
		isBlocked, customerID,
	).Error
	if err != nil {
		s.log.Errorf("could not update customer is_blocked(%s): %v", customerID, err)
		return domain.InternalServerError
	}
	return nil
}
