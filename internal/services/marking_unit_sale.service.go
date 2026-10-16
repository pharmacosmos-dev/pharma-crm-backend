package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// marking_unit_sales: har bir markirovka bo'yicha nechta dona sotilgan (sold_quantity)
// va nechta qolgan (stock_quantity). Bir markirovka = bitta pachka = unit_per_pack dona.
// Qator markirovka birinchi marta yakunlangan savdoda uchraganda yaratiladi.

type markingUnitSale struct {
	Id            string
	SoldQuantity  int
	StockQuantity int
}

type markingCartItem struct {
	Id           string
	ProductId    string
	UnitPerPack  int
	UnitQuantity int
	Markings     utils.StringArray `gorm:"type:text[]"`
}

// checkMarkingHasStock: markirovka jadvalda bo'lsa va qoldig'i 0 bo'lsa — to'liq sotilgan.
// Jadvalda yo'q markirovka hali sotilmagan hisoblanadi.
func (s *Services) checkMarkingHasStock(ctx context.Context, tx *gorm.DB, marking string) error {
	if marking == "" {
		return nil
	}

	var res markingUnitSale
	result := tx.WithContext(ctx).
		Raw(`SELECT id, sold_quantity, stock_quantity FROM marking_unit_sales WHERE marking = ?`, marking).
		Scan(&res)
	if result.Error != nil {
		s.log.Errorf("could not get marking_unit_sale(%s): %v", marking, result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected > 0 && res.StockQuantity <= 0 {
		return domain.MarkingAlreadySoldError
	}

	return nil
}

func (s *Services) getMarkingCartItems(ctx context.Context, tx *gorm.DB, saleId string) ([]markingCartItem, error) {
	var items []markingCartItem
	err := tx.WithContext(ctx).
		Table("cart_items ci").
		Select("ci.id", "p.id AS product_id", "p.unit_per_pack", "ci.unit_quantity", "ci.markings").
		Joins("JOIN store_products sp ON sp.id = ci.store_product_id").
		Joins("JOIN products p ON sp.product_id = p.id").
		Where("ci.sale_id = ? AND COALESCE(cardinality(ci.markings), 0) > 0", saleId).
		Find(&items).Error
	if err != nil {
		s.log.Errorf("could not get marking cart_items of sale(%s): %v", saleId, err)
		return nil, domain.InternalServerError
	}

	return items, nil
}

// lockMarkingUnitSale: markirovka qatorini FOR UPDATE bilan oladi (parallel savdolar uchun).
func (s *Services) lockMarkingUnitSale(ctx context.Context, tx *gorm.DB, marking string) (*markingUnitSale, bool, error) {
	var res markingUnitSale
	result := tx.WithContext(ctx).
		Raw(`SELECT id, sold_quantity, stock_quantity FROM marking_unit_sales WHERE marking = ? FOR UPDATE`, marking).
		Scan(&res)
	if result.Error != nil {
		s.log.Errorf("could not lock marking_unit_sale(%s): %v", marking, result.Error)
		return nil, false, domain.InternalServerError
	}

	return &res, result.RowsAffected > 0, nil
}

func (s *Services) moveMarkingUnits(ctx context.Context, tx *gorm.DB, id string, soldDiff int) error {
	err := tx.WithContext(ctx).Exec(`
	UPDATE marking_unit_sales
	SET sold_quantity = sold_quantity + ?, stock_quantity = stock_quantity - ?, updated_at = NOW()
	WHERE id = ?`, soldDiff, soldDiff, id).Error
	if err != nil {
		s.log.Errorf("could not update marking_unit_sale(%s): %v", id, err)
		return domain.InternalServerError
	}

	return nil
}

// ApplyMarkingUnitSale: savdo yakunlanganda (stage 9) cart_item.unit_quantity markirovkalar
// bo'yicha ketma-ket taqsimlanadi: har bir markirovkadan qoldig'i yetguncha olinadi,
// sold_quantity ga qo'shiladi va stock_quantity dan ayriladi.
func (s *Services) ApplyMarkingUnitSale(ctx context.Context, tx *gorm.DB, saleId string) error {
	items, err := s.getMarkingCartItems(ctx, tx, saleId)
	if err != nil {
		return err
	}

	for _, item := range items {
		unitPerPack := max(item.UnitPerPack, 1)
		remaining := item.UnitQuantity

		for _, marking := range item.Markings {
			if remaining <= 0 {
				break
			}

			err = tx.WithContext(ctx).Exec(`
			INSERT INTO marking_unit_sales (marking, product_id, unit_per_pack, sold_quantity, stock_quantity)
			VALUES (?, ?, ?, 0, ?)
			ON CONFLICT (marking) DO NOTHING`, marking, item.ProductId, unitPerPack, unitPerPack).Error
			if err != nil {
				s.log.Errorf("could not insert marking_unit_sale(%s): %v", marking, err)
				return domain.InternalServerError
			}

			row, _, err := s.lockMarkingUnitSale(ctx, tx, marking)
			if err != nil {
				return err
			}

			take := min(row.StockQuantity, remaining)
			if take <= 0 {
				continue
			}
			if err = s.moveMarkingUnits(ctx, tx, row.Id, take); err != nil {
				return err
			}
			remaining -= take
		}

		// chek allaqachon fiskallashgan, shuning uchun savdo to'xtatilmaydi — faqat log
		if remaining > 0 {
			s.log.Warnf("markings of cart_item(%s) do not cover sold units: %d left unassigned", item.Id, remaining)
		}
	}

	return nil
}

// ReverseMarkingUnitSale: vozvrat yakunlanganda (stage 11) qaytarilgan dona markirovkalar
// bo'yicha sold_quantity dan ayriladi va stock_quantity ga qo'shiladi.
func (s *Services) ReverseMarkingUnitSale(ctx context.Context, tx *gorm.DB, saleId string) error {
	items, err := s.getMarkingCartItems(ctx, tx, saleId)
	if err != nil {
		return err
	}

	for _, item := range items {
		remaining := item.UnitQuantity

		for _, marking := range item.Markings {
			if remaining <= 0 {
				break
			}

			row, found, err := s.lockMarkingUnitSale(ctx, tx, marking)
			if err != nil {
				return err
			}
			if !found {
				continue
			}

			take := min(row.SoldQuantity, remaining)
			if take <= 0 {
				continue
			}
			if err = s.moveMarkingUnits(ctx, tx, row.Id, -take); err != nil {
				return err
			}
			remaining -= take
		}
	}

	return nil
}
