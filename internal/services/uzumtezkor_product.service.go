package services

import (
	"context"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
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

func (s *Services) GetOnlineProducts(ctx context.Context, params *domain.UzumTezkorProductQueryParam) ([]domain.OnlineProductsPrice, int64, error) {
	// sp_agg: is_online_order do'konlar bo'yicha ombor qoldig'i
	spJoin := `LEFT JOIN (
		SELECT sp.product_id, COALESCE(SUM(sp.unit_quantity), 0) AS store_quantity
		FROM store_products sp
		JOIN stores st ON sp.store_id = st.id
		WHERE st.is_online_order = true`
	if params.StoreId != "" {
		spJoin += fmt.Sprintf(" AND sp.store_id = '%s'", params.StoreId)
	}
	spJoin += " GROUP BY sp.product_id) sp_agg ON sp_agg.product_id = opp.product_id"

	// sl_agg: uzum_tez_kor orqali yakunlangan buyurtmalar (stage=9, online_status=3)
	slJoin := fmt.Sprintf(`LEFT JOIN (
		SELECT ci.product_id, COALESCE(SUM(ci.unit_quantity), 0) AS sold_quantity
		FROM cart_items ci
		JOIN sales s ON ci.sale_id = s.id
		WHERE s.uzum_tez_kor > 0
		  AND s.online_status = %d
		  AND s.stage         = %d`,
		constants.SaleOnlineStageCompleted,
		constants.SaleStageFinished,
	)
	if params.StoreId != "" {
		slJoin += fmt.Sprintf(" AND s.store_id = '%s'", params.StoreId)
	}
	if params.StartDate != "" {
		slJoin += fmt.Sprintf(" AND s.created_at >= '%s'", params.StartDate)
	}
	if params.EndDate != "" {
		slJoin += fmt.Sprintf(" AND s.created_at <= '%s'", params.EndDate)
	}
	slJoin += " GROUP BY ci.product_id) sl_agg ON sl_agg.product_id = opp.product_id"

	qb := s.db.WithContext(ctx).
		Table("online_products_price opp").
		Joins("LEFT JOIN products p ON p.id = opp.product_id").
		Joins(spJoin).
		Joins(slJoin)

    if params.Type != "" {
        qb = qb.Where("opp.type = ?", params.Type)
    }
    if params.ProductId != "" {
        qb = qb.Where("opp.product_id = ?", params.ProductId)
    }
	if params.Search != "" {
		search := fmt.Sprintf("%%%s%%", params.Search)
		switch utils.DefineProductSearchQuery(params.Search) {
		case "barcode":
			qb = qb.Where("p.barcode LIKE ?", search)
		case "material_code":
			qb = qb.Where("opp.material_code LIKE ?", search)
		default:
			qb = qb.Where("p.name ILIKE ?", search)
		}
	}

    var total int64
    if err := qb.Count(&total).Error; err != nil {
        s.log.Errorf("GetOnlineProducts count: %v", err)
        return nil, 0, domain.InternalServerError
    }

    var res []domain.OnlineProductsPrice
    err := qb.Select(
        "opp.id", 
		"opp.product_id", 
		"opp.material_code", 
		"opp.type",
        "opp.retail_price", 
		"opp.created_by", 
		"opp.created_at",
        "opp.updated_at", 
		"opp.updated_by",
        "p.name    AS product_name",
        "p.barcode AS product_barcode",
		"COALESCE(p.photos[1], '') AS product_photo",
        "p.unit_per_pack",
        "COALESCE(sp_agg.store_quantity, 0) AS store_quantity",
        "COALESCE(sl_agg.sold_quantity,  0) AS sold_quantity",
    ).
        Order("sold_quantity DESC, opp.created_at DESC").
        Limit(params.Limit).
        Offset(params.Offset).
        Find(&res).Error
    if err != nil {
        s.log.Errorf("GetOnlineProducts find: %v", err)
        return nil, 0, domain.InternalServerError
    }

    for i := range res {
        res[i].Product = domain.NewNullStruct(domain.OnlineProductSummary{
            Id:      res[i].ProductId,
            Name:    res[i].ProductName,
            Barcode: res[i].ProductBarcode,
			Photos:  res[i].ProductPhoto,
        }, res[i].ProductId != "")

        qty, upack := int(res[i].StoreQuantity), res[i].UnitPerPack
        if upack > 0 {
            if qty%upack > 0 {
                res[i].Units = fmt.Sprintf("%d (%d/%d)", qty/upack, qty%upack, upack)
            } else {
                res[i].Units = fmt.Sprintf("%d", qty/upack)
            }
        }
    }

    return res, total, nil
}
