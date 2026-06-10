package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pharma-crm-backend/domain"
)

// InsertOnlinePricesFromOnec — 1C dan material_code + price oladi, product_id topib insert qiladi.
// type = 'uzum', created_by handler dan uzatiladi.
func (s *Services) InsertOnlinePricesFromOnec(ctx context.Context, req *domain.UzumTezkorProductRepriceFromOnecRequest, createdBy string) error {
	if len(req.Items) == 0 {
		return fmt.Errorf("items list is empty")
	}

	var createdByVal interface{}
	if createdBy != "" {
		createdByVal = createdBy
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	for _, item := range req.Items {
		err := tx.Exec(`
			INSERT INTO online_products_price (product_id, material_code, type, retail_price, created_by)
			SELECT p.id, ?, 'uzum', ?, ?
			FROM products p
			WHERE p.material_code::text = ?
			LIMIT 1`,
			item.MaterialCode,
			item.RetailPrice,
			createdByVal,
			item.MaterialCode,
		).Error
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("failed to insert online price material_code=%s: %v", item.MaterialCode, err)
			return domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit InsertOnlinePricesFromOnec: %v", err)
		return domain.InternalServerError
	}

	return nil
}

// UpdateOnlinePriceByMaterialCode — CRM dan material_code bo'yicha mavjud narxni yangilaydi.
func (s *Services) UpdateOnlinePriceByMaterialCode(ctx context.Context, req *domain.UpdateOnlinePriceRequest, createdBy string) error {
	productType := req.Type
	if productType == "" {
		productType = "uzum"
	}

	result := s.db.WithContext(ctx).Exec(`
		UPDATE online_products_price
		SET retail_price = ?, updated_by = ?, updated_at = NOW()
		WHERE material_code = ? AND type = ?`,
		req.RetailPrice,
		createdBy,
		req.MaterialCode,
		productType,
	)
	if result.Error != nil {
		s.log.Errorf("failed to update online price material_code=%s: %v", req.MaterialCode, result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product with material_code=%s not found", req.MaterialCode)
	}

	return nil
}

// BulkUpdateOnlinePriceFromExcel — Excel dan kelgan items bo'yicha narxlarni yangilaydi.
// Topilmagan material_code lar notFound slice ga yig'iladi.
func (s *Services) BulkUpdateOnlinePriceFromExcel(ctx context.Context, items []domain.UzumTezKorProductRepriceItem, productType string, updatedBy string) (updated int64, notFound []string, err error) {
	if len(items) == 0 {
		return 0, nil, fmt.Errorf("items list is empty")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	for _, item := range items {
		result := tx.Exec(`
			UPDATE online_products_price
			SET retail_price = ?, updated_by = ?, updated_at = NOW()
			WHERE material_code = ? AND type = ?`,
			item.RetailPrice,
			updatedBy,
			item.MaterialCode,
			productType,
		)
		if result.Error != nil {
			_ = tx.Rollback()
			s.log.Errorf("failed to bulk update online price material_code=%s: %v", item.MaterialCode, result.Error)
			return 0, nil, domain.InternalServerError
		}
		if result.RowsAffected == 0 {
			notFound = append(notFound, item.MaterialCode)
		} else {
			updated += result.RowsAffected
		}
	}

	if err = tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit BulkUpdateOnlinePriceFromExcel: %v", err)
		return 0, nil, domain.InternalServerError
	}

	return updated, notFound, nil
}

// CreateOnlinePrice — CRM dan yangi online_products_price qatori qo'shadi.
// material_code orqali product_id topiladi; topilmasa xato qaytariladi.
func (s *Services) CreateOnlinePrice(ctx context.Context, req *domain.CreateOnlinePriceRequest, createdBy string) error {
	var createdByVal interface{}
	if createdBy != "" {
		createdByVal = createdBy
	}

	result := s.db.WithContext(ctx).Exec(`
		INSERT INTO online_products_price (product_id, material_code, type, retail_price, created_by)
		SELECT p.id, ?, ?, ?, ?
		FROM products p
		WHERE p.material_code::text = ?
		LIMIT 1`,
		req.MaterialCode,
		req.Type,
		req.RetailPrice,
		createdByVal,
		req.MaterialCode,
	)
	if result.Error != nil {
		s.log.Errorf("failed to create online price material_code=%s: %v", req.MaterialCode, result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product with material_code=%s not found", req.MaterialCode)
	}

	return nil
}

// GetOnlineProducts — CRM uchun narx tarixi (products bilan left join)
func (s *Services) GetOnlineProducts(ctx context.Context, params *domain.UzumTezkorProductQueryParam) ([]domain.OnlineProductsPrice, int64, error) {
	var tmp []struct {
		Id             string     `gorm:"id"`
		ProductId      string     `gorm:"product_id"`
		MaterialCode   string     `gorm:"material_code"`
		Type           string     `gorm:"type"`
		RetailPrice    float64    `gorm:"retail_price"`
		CreatedBy      *string    `gorm:"created_by"`
		CreatedAt      *time.Time `gorm:"created_at"`
		UpdatedAt      *time.Time `gorm:"updated_at"`
		UpdatedBy      *string    `gorm:"updated_by"`
		ProductName    string     `gorm:"product_name"`
		ProductBarcode string     `gorm:"product_barcode"`
	}
	var total int64

	q := s.db.WithContext(ctx).
		Table("online_products_price opp").
		Select(
			"opp.id",
			"opp.product_id",
			"opp.material_code",
			"opp.type",
			"opp.retail_price",
			"opp.created_by",
			"opp.created_at",
			"opp.updated_at",
			"opp.updated_by",
			"p.name AS product_name",
			"p.barcode AS product_barcode",
		).
		Joins("LEFT JOIN products p ON opp.product_id = p.id")

	if params.Type != "" {
		q = q.Where("opp.type = ?", params.Type)
	}
	if params.ProductId != "" {
		q = q.Where("opp.product_id = ?", params.ProductId)
	}
	if params.MaterialCode != "" {
		q = q.Where("opp.material_code = ?", params.MaterialCode)
	}

	if err := q.Count(&total).Error; err != nil {
		s.log.Errorf("failed to count online_products_price: %v", err)
		return nil, 0, domain.InternalServerError
	}

	if err := q.Order("opp.created_at DESC").
		Limit(params.Limit).Offset(params.Offset).
		Find(&tmp).Error; err != nil {
		s.log.Errorf("failed to get online_products_price: %v", err)
		return nil, 0, domain.InternalServerError
	}

	result := make([]domain.OnlineProductsPrice, 0, len(tmp))
	for _, item := range tmp {
		result = append(result, domain.OnlineProductsPrice{
			Id:           item.Id,
			ProductId:    item.ProductId,
			MaterialCode: item.MaterialCode,
			Type:         item.Type,
			RetailPrice:  item.RetailPrice,
			CreatedBy:    item.CreatedBy,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
			UpdatedBy:    item.UpdatedBy,
			Product: domain.NewNullStruct(domain.OnlineProductSummary{
				Id:      item.ProductId,
				Name:    item.ProductName,
				Barcode: item.ProductBarcode,
			}, item.ProductId != ""),
		})
	}

	return result, total, nil
}
