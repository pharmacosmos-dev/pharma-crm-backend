package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		if err := s.db.WithContext(ctx).Count(&count).Error; err != nil {
			return &res, fmt.Errorf("error on checking loyalty card barcode: %s", err.Error())
		}
		if count > 0 {
			return &res, domain.DuplicateLoyaltyCardError
		}
		loyaltyCardBarcode = sql.NullString{String: *req.LoyaltyCardBarcode, Valid: true}
		loyaltyCardType = sql.NullString{String: "physical", Valid: true}
		loyaltyCardCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}
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
				s.log.Error("could not find loyalty level for new customer")
				_ = tx.Rollback()
				return &res, fmt.Errorf("could not find loyalty level for new customer: %s", err.Error())
			}
			s.log.Errorf("error on getting loyalty card level in db: %s", err.Error())
			_ = tx.Rollback()
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
		if errors.As(err, &pgErr) {
			// 23505 = unique_violation
			if pgErr.Code == "23505" {
				_ = tx.Rollback()
				return &res, domain.DuplicatePhoneError
			}
		}

		s.log.Errorf("could not create customer: %v", err)
		_ = tx.Rollback()
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
			s.log.Errorf("error on creating loyalty card levelup history: %s", err.Error())
			_ = tx.Rollback()
			return &res, fmt.Errorf("error on creating loyalty card levelup history: %s", err.Error())
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("error on commit transaction: %s", err.Error())
		return nil, fmt.Errorf("error on commit transaction: %s", err.Error())
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
	var tmpCustomer []struct {
		Id                   string     `gorm:"id" json:"id"`
		PublicId             int        `gorm:"public_id" json:"public_id"`
		StoreId              string     `gorm:"store_id" json:"store_id"`
		TagId                string     `gorm:"tag_id" json:"tag_id"`
		FirstName            string     `gorm:"first_name" json:"first_name"`
		LastName             string     `gorm:"last_name" json:"last_name"`
		FullName             string     `gorm:"full_name" json:"full_name"`
		Phone                string     `gorm:"phone" json:"phone"`
		Birthday             string     `gorm:"birthday" json:"birthday" example:"2006-01-02"`
		Gender               string     `gorm:"gender" json:"gender" example:"male/female"`
		Balance              float64    `gorm:"balance" json:"balance"`
		SpendingFromBalance  float64    `gorm:"spending_from_balance" json:"spending_from_balance"`
		DiscountCard         string     `gorm:"discount_card" json:"discount_card"`
		DiscountPercent      int        `gorm:"discount_percent" json:"discount_percent"`
		LoyaltyCardBarcode   string     `gorm:"loyalty_card_barcode" json:"loyalty_card_barcode"`
		LoyaltyCardPercent   int        `gorm:"loyalty_card_percent" json:"loyalty_card_percent"`
		LoyaltyCardLevelId   string     `gorm:"loyalty_card_level_id" json:"loyalty_card_level_id"`
		LoyaltyCardType      string     `gorm:"loyalty_card_type" json:"loyalty_card_type"`
		LoyaltyCardCreatedBy string     `gorm:"loyalty_card_created_by" json:"loyalty_card_created_by"`
		LoyaltyCardCreatedAt *time.Time `gorm:"loyalty_card_created_at" json:"loyalty_card_created_at"`

		TelegramChatId int64      `gorm:"telegram_chat_id" json:"telegram_chat_id"`
		CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
		UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`

		TId   string `gorm:"t_id"`
		TName string `gorm:"t_name"`

		SId   string `gorm:"s_id"`
		SName string `gorm:"s_name"`
	}

	// Start building the query
	query := s.db.
		Select(
			"c.id",
			"c.public_id",
			"c.store_id",
			"c.tag_id",
			"c.first_name",
			"c.last_name",
			"c.full_name",
			"c.phone",
			"c.birthday",
			"c.gender",
			"c.balance",
			"c.spending_from_balance",
			"c.discount_card",
			"c.discount_percent",
			"c.loyalty_card_barcode",
			"c.loyalty_card_percent",
			"c.loyalty_card_level_id",
			"c.loyalty_card_type",
			"c.loyalty_card_created_by",
			"c.loyalty_card_created_at",
			"c.telegram_chat_id",
			"c.created_at",
			"c.updated_at",

			"s.id AS s_id",
			"s.name AS s_name",

			"t.id AS t_id",
			"t.name AS t_name",
		).Table("customers c").
		Joins("LEFT JOIN stores s ON c.store_id = s.id").
		Joins("LEFT JOIN tags t ON c.tag_id = t.id")

	if params.Search != "" {
		if usedInSalePage {
			query = query.Where("c.discount_card = ? or c.loyalty_card_barcode = ?", params.Search, params.Search)
		} else {
			query = query.Where("c.public_id::text ilike ? or c.phone::text ilike ? or c.full_name ilike ?", "%"+params.Search+"%", "%"+params.Search+"%", "%"+params.Search+"%")
		}
	}

	if params.StoreID != "" {
		query = query.Where("c.store_id = ?", params.StoreID)
	}

	if params.CompanyId != "" {
		query = query.Where("s.company_id = ? ", params.CompanyId)
	}

	var (
		customers  []domain.Customer
		totalCount int64
	)

	err := query.WithContext(ctx).
		Count(&totalCount).
		Limit(params.Limit).
		Offset(params.Offset).
		Order("c.created_at DESC").
		Find(&tmpCustomer).Error
	if err != nil {
		s.log.Errorf("could not get new customer: %v", err)
		return nil, 0, domain.InternalServerError
	}

	for _, row := range tmpCustomer {
		customers = append(customers, domain.Customer{
			Id:                   row.Id,
			PublicId:             row.PublicId,
			StoreId:              row.StoreId,
			TagId:                row.TagId,
			FirstName:            row.FirstName,
			LastName:             row.LastName,
			FullName:             row.FullName,
			Phone:                row.Phone,
			Birthday:             row.Birthday,
			Gender:               row.Gender,
			Balance:              row.Balance,
			SpendingFromBalance:  row.SpendingFromBalance,
			DiscountCard:         row.DiscountCard,
			DiscountPercent:      row.DiscountPercent,
			LoyaltyCardBarcode:   row.LoyaltyCardBarcode,
			LoyaltyCardPercent:   row.LoyaltyCardPercent,
			LoyaltyCardLevelId:   row.LoyaltyCardLevelId,
			LoyaltyCardType:      row.LoyaltyCardType,
			LoyaltyCardCreatedBy: row.LoyaltyCardCreatedBy,
			LoyaltyCardCreatedAt: row.LoyaltyCardCreatedAt,
			TelegramChatId:       row.TelegramChatId,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,

			Store: &domain.Store{
				Id:   row.SId,
				Name: row.SName,
			},

			Tag: &domain.Tag{
				Id:   row.TId,
				Name: row.TName,
			},
		})
	}

	return customers, totalCount, nil
}

func (s *Services) GetCustomerById(ctx context.Context, tx *gorm.DB, id string) (*domain.Customer, error) {
	var res domain.Customer

	err := tx.WithContext(ctx).
		Where("id = ?", id).
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
			s.log.Error("could not find customer to update")
			_ = tx.Rollback()
			return &res, domain.NotFoundError
		}
		s.log.Errorf("error on getting existing customer in db: %s", err.Error())
		_ = tx.Rollback()
		return &res, fmt.Errorf("error on getting existing customer in db: %s", err.Error())
	}

	if existingCustomer.LoyaltyCardBarcode != "" {
		loyaltyCardShouldCreate = false
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
				s.log.Error("could not find loyalty level for new customer")
				_ = tx.Rollback()
				return &res, fmt.Errorf("could not find loyalty level for new customer: %s", err.Error())
			}
			s.log.Errorf("error on getting loyalty card level in db: %s", err.Error())
			_ = tx.Rollback()
			return &res, fmt.Errorf("error on getting loyalty card level in db: %s", err.Error())
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
		s.log.Errorf("could not update customer: %v", err)
		_ = tx.Rollback()
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
			s.log.Errorf("error on creating loyalty card levelup history: %s", err.Error())
			_ = tx.Rollback()
			return &res, fmt.Errorf("error on creating loyalty card levelup history: %s", err.Error())
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("error on commit transaction: %s", err.Error())
		return nil, fmt.Errorf("error on commit transaction: %s", err.Error())
	}

	return &res, nil
}
