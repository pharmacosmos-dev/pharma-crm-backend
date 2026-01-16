package services

import (
	"strings"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

const (
	MovementStatusNew       = 0
	MovementStatusPending   = 1
	MovementStatusCancelled = -1
	MovementStatusSending   = 2
	MovementStatusCompleted = 3
)
const batchSize = 500

type MovementCreateDto struct {
	ProductId     string            `gorm:"product_id"`
	StoreId       string            `gorm:"store_id"`
	ToStoreId     domain.NullString `gorm:"to_store_id"`
	MovementType  string            `gorm:"movement_type"`
	MovementId    string            `gorm:"movement_id"`
	DisplayId     int64             `gorm:"display_id"`
	PrevQuantity  int               `gorm:"prev_quantity"`
	Quantity      int               `gorm:"quantity"`
	AfterQuantity int               `gorm:"after_quantity"`
	Price         float64           `gorm:"price"`
	TotalPrice    float64           `gorm:"total_price"`
	Status        int8              `gorm:"status"`
	MovementDate  time.Time         `gorm:"movement_date"`
}

type Details struct {
	ProductId     string  `gorm:"product_id"`
	PrevQuantity  int     `gorm:"prev_quantity"`
	Quantity      int     `gorm:"quantity"`
	AfterQuantity int     `gorm:"after_quantity"`
	Price         float64 `gorm:"price"`
	TotalPrice    float64 `gorm:"total_price"`
}

func (s *Services) CreateProductMovementsForImport(data *domain.Import) {

	var details []Details

	query := `
	SELECT
		imd.product_id AS product_id,

		SUM(sp.unit_quantity) - (imd.scanned_count * p.unit_per_pack) AS prev_quantity,

		(imd.scanned_count * p.unit_per_pack) AS quantity,

		SUM(sp.unit_quantity) AS after_quantity,

		imd.retail_price_vat AS price,

		(imd.retail_price_vat * imd.scanned_count) AS total_price

	FROM import_details imd
	JOIN products p
	ON p.id = imd.product_id
	LEFT JOIN store_products sp
	ON sp.product_id = p.id
	AND sp.store_id = ?

	WHERE imd.import_id = ?

	GROUP BY
		imd.product_id,
		imd.scanned_count,
		p.unit_per_pack,
		imd.retail_price_vat;
	`

	if err := s.db.Raw(query, data.StoreId, data.Id).Scan(&details).Error; err != nil {
		s.log.Errorf("could not get movements import details: %v", err)
		return
	}

	if len(details) == 0 {
		return
	}

	// 2️⃣ Transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		s.log.Errorf("could not begin tx: %v", tx.Error)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3️⃣ Batch insert
	for i := 0; i < len(details); i += batchSize {
		end := min(i+batchSize, len(details))

		valueStrings := make([]string, 0, end-i)
		valueArgs := make([]any, 0, (end-i)*12)

		for _, d := range details[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				d.ProductId,
				data.StoreId,
				constants.MovementTypeImport,
				data.Id,
				data.PublicId,
				d.PrevQuantity,
				d.Quantity,
				d.AfterQuantity,
				d.Price,
				d.TotalPrice,
				MovementStatusCompleted,
				data.CreatedAt,
			)
		}

		query := `
		INSERT INTO product_movements
			(
				product_id,
				store_id,
				movement_type,
				movement_id,
				display_id,
				prev_quantity,
				quantity,
				after_quantity,
				price,
				total_price,
				status,
				movement_date
			)
		VALUES ` + strings.Join(valueStrings, ",")

		if err := tx.Exec(query, valueArgs...).Error; err != nil {
			_ = tx.Rollback()
			s.log.Errorf("bulk insert failed: %v", err)
			return
		}
	}

	// 4️⃣ Commit
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("commit failed: %v", err)
	}

}

func (s *Services) CreateProductMovementsForSale(data *domain.Sale) {
	var details []Details
	query := `
	SELECT
		sp.product_id AS product_id,
		sp.unit_quantity + ci.unit_quantity AS prev_quantity,
		ci.unit_quantity AS quantity,
		sp.unit_quantity - ci.unit_quantity AS after_quantity,
		ci.unit_price AS price,
		ci.total_price AS total_price
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	WHERE ci.sale_id = ?;
	`
	if err := s.db.Raw(query, data.Id).Scan(&details).Error; err != nil {
		s.log.Errorf("could not get movements cart_items: %v", err)
		return
	}

	if len(details) == 0 {
		return
	}

	// 2️⃣ Transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		s.log.Errorf("could not begin tx: %v", tx.Error)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3️⃣ Batch insert
	for i := 0; i < len(details); i += batchSize {
		end := min(i+batchSize, len(details))

		valueStrings := make([]string, 0, end-i)
		valueArgs := make([]any, 0, (end-i)*12)

		for _, d := range details[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				d.ProductId,
				data.StoreId,
				constants.MovementTypeSale,
				data.Id,
				data.SaleNumber,
				d.PrevQuantity,
				d.Quantity,
				d.AfterQuantity,
				d.Price,
				d.TotalPrice,
				MovementStatusCompleted,
				time.Now(),
			)
		}

		query := `
		INSERT INTO product_movements
			(
				product_id,
				store_id,
				movement_type,
				movement_id,
				display_id,
				prev_quantity,
				quantity,
				after_quantity,
				price,
				total_price,
				status,
				movement_date
			)
		VALUES ` + strings.Join(valueStrings, ",")

		if err := tx.Exec(query, valueArgs...).Error; err != nil {
			_ = tx.Rollback()
			s.log.Errorf("bulk insert failed: %v", err)
			return
		}
	}

	// 4️⃣ Commit
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("commit failed: %v", err)
	}

}

func (s *Services) CreateProductMovementsForReturnSale(data *domain.Sale) {
	var details []Details
	query := `
	SELECT
		sp.product_id AS product_id,
		sp.unit_quantity - ci.unit_quantity AS prev_quantity,
		ci.unit_quantity AS quantity,
		sp.unit_quantity + ci.unit_quantity AS after_quantity,
		ci.unit_price AS price,
		ci.total_price AS total_price
	FROM cart_items ci
	JOIN store_products sp ON ci.store_product_id = sp.id
	WHERE ci.sale_id = ?;
	`
	if err := s.db.Raw(query, data.Id).Scan(&details).Error; err != nil {
		s.log.Errorf("could not get movements cart_items: %v", err)
		return
	}

	if len(details) == 0 {
		return
	}

	// 2️⃣ Transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		s.log.Errorf("could not begin tx: %v", tx.Error)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 3️⃣ Batch insert
	for i := 0; i < len(details); i += batchSize {
		end := min(i+batchSize, len(details))

		valueStrings := make([]string, 0, end-i)
		valueArgs := make([]any, 0, (end-i)*12)

		for _, d := range details[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				d.ProductId,
				data.StoreId,
				constants.MovementTypeSale,
				data.Id,
				data.SaleNumber,
				d.PrevQuantity,
				d.Quantity,
				d.AfterQuantity,
				d.Price,
				d.TotalPrice,
				MovementStatusCompleted,
				time.Now(),
			)
		}

		query := `
		INSERT INTO product_movements
			(
				product_id,
				store_id,
				movement_type,
				movement_id,
				display_id,
				prev_quantity,
				quantity,
				after_quantity,
				price,
				total_price,
				status,
				movement_date
			)
		VALUES ` + strings.Join(valueStrings, ",")

		if err := tx.Exec(query, valueArgs...).Error; err != nil {
			_ = tx.Rollback()
			s.log.Errorf("bulk insert failed: %v", err)
			return
		}
	}

	// 4️⃣ Commit
	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("commit failed: %v", err)
	}
}

func (s *Services) CreateProductMovementsForReturnSupplier(data *domain.Sale) {
	
}

func (s *Services) CreateProductMovementsForTransferOut(data *domain.Sale) {

}

func (s *Services) CreateProductMovementsForTransferIn(data *domain.Sale) {

}

func (s *Services) CreateProductMovementsForInventory(data *domain.Sale) {

}

func (s *Services) CreateProductMovementsForRepricing(data *domain.Sale) {

}
