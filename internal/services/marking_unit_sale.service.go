package services

import (
	"context"
	"fmt"

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
	ProductName  string
	UnitPerPack  int
	UnitQuantity int
	Markings     utils.StringArray `gorm:"type:text[]"`
}

// getMarkingAvailableUnits: har bir markirovkadan yana nechta dona sotish mumkin.
// Jadvalda yo'q markirovka hali ochilmagan pachka — unit_per_pack dona.
func (s *Services) getMarkingAvailableUnits(ctx context.Context, tx *gorm.DB, markings []string, unitPerPack int) (map[string]int, error) {
	res := make(map[string]int, len(markings))
	if len(markings) == 0 {
		return res, nil
	}

	var rows []struct {
		Marking       string
		StockQuantity int
	}
	err := tx.WithContext(ctx).
		Raw(`SELECT marking, stock_quantity FROM marking_unit_sales WHERE marking IN ?`, markings).
		Scan(&rows).Error
	if err != nil {
		s.log.Errorf("could not get marking_unit_sales stock: %v", err)
		return nil, domain.InternalServerError
	}

	for _, m := range markings {
		res[m] = max(unitPerPack, 1)
	}
	for _, r := range rows {
		res[r.Marking] = r.StockQuantity
	}

	return res, nil
}

// Xabar: "marking.not.enough:<qoldiq>:<mahsulot nomi>" — nom oxirida, chunki
// uning ichida ham ":" bo'lishi mumkin.
func markingNotEnoughError(available int, productName string) error {
	return domain.NewError(
		domain.MarkingNotEnoughError.Code,
		fmt.Sprintf("%s:%d:%s", domain.MarkingNotEnoughError.Message, available, productName),
	)
}

// checkMarkingCanCover: cart_item'ga yangi markirovka qo'shishdan oldin tekshiradi.
//   - qoldig'i 0 — markirovka to'liq sotilgan;
//   - qisman sotilgan markirovka (qoldiq < unit_per_pack) cart_item'ning hali
//     markirovka bilan qoplanmagan donalaridan kam bo'lsa — yetmaydi.
//
// To'liq pachka yetmasa xato berilmaydi: kassir keyingi pachka markirovkasini skanerlaydi.
func (s *Services) checkMarkingCanCover(ctx context.Context, tx *gorm.DB, cartItem *domain.CartItem, marking string) error {
	if marking == "" {
		return nil
	}

	var product struct {
		Name        string
		UnitPerPack int
	}
	err := tx.WithContext(ctx).
		Raw(`SELECT p.name, p.unit_per_pack FROM store_products sp JOIN products p ON p.id = sp.product_id WHERE sp.id = ?`, cartItem.StoreProductId).
		Scan(&product).Error
	if err != nil {
		s.log.Errorf("could not get unit_per_pack of store_product(%s): %v", cartItem.StoreProductId, err)
		return domain.InternalServerError
	}
	unitPerPack := max(product.UnitPerPack, 1)

	available, err := s.getMarkingAvailableUnits(ctx, tx, append([]string{marking}, cartItem.Markings...), unitPerPack)
	if err != nil {
		return err
	}

	if available[marking] <= 0 {
		return domain.MarkingAlreadySoldError
	}

	needed := cartItem.UnitQuantity
	for _, m := range cartItem.Markings {
		needed -= available[m]
	}
	if needed > available[marking] && available[marking] < unitPerPack {
		return markingNotEnoughError(available[marking], product.Name)
	}

	return nil
}

// validateSaleMarkingStock: FinalSale'da (fiskallashdan oldin) savdodagi markirovkalar
// donalarni qoplay olishini tekshiradi — ApplyMarkingUnitSale bilan bir xil taqsimlash.
// Bir markirovka savdoning bir nechta cart_item'ida bo'lsa ham qoldig'i umumiy hisoblanadi.
// Xato faqat qoplanmay qolgan cart_item'da qisman sotilgan markirovka bo'lsa beriladi,
// aks holda markirovkasi to'liq skanerlanmagan oddiy savdolar to'xtab qolardi.
func (s *Services) validateSaleMarkingStock(ctx context.Context, tx *gorm.DB, saleId string) error {
	items, err := s.getMarkingCartItems(ctx, tx, saleId)
	if err != nil {
		return err
	}

	available := map[string]int{}
	for _, item := range items {
		unitPerPack := max(item.UnitPerPack, 1)

		var unknown []string
		for _, m := range item.Markings {
			if _, ok := available[m]; !ok {
				unknown = append(unknown, m)
			}
		}
		fetched, err := s.getMarkingAvailableUnits(ctx, tx, unknown, unitPerPack)
		if err != nil {
			return err
		}
		for m, v := range fetched {
			available[m] = v
		}

		remaining := item.UnitQuantity
		partial := ""
		for _, m := range item.Markings {
			if available[m] <= 0 {
				return domain.MarkingAlreadySoldError
			}
			if available[m] < unitPerPack && partial == "" {
				partial = m
			}
			take := min(available[m], remaining)
			available[m] -= take
			remaining -= take
		}

		if remaining > 0 && partial != "" {
			return markingNotEnoughError(item.UnitQuantity-remaining, item.ProductName)
		}
	}

	return nil
}

func (s *Services) getMarkingCartItems(ctx context.Context, tx *gorm.DB, saleId string) ([]markingCartItem, error) {
	var items []markingCartItem
	err := tx.WithContext(ctx).
		Table("cart_items ci").
		Select("ci.id", "p.id AS product_id", "p.name AS product_name", "p.unit_per_pack", "ci.unit_quantity", "ci.markings").
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

// GetMarkingUnitSales: markirovkalar bo'yicha sotilgan/qolgan donalar ro'yxati.
// marking parametri markirovka yoki mahsulot nomi bo'yicha qidiradi.
func (s *Services) GetMarkingUnitSales(
	ctx context.Context, params *domain.MarkingUnitSaleQueryParams,
) ([]domain.MarkingUnitSale, int64, error) {
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).
			Table("marking_unit_sales mus").
			Joins("LEFT JOIN products p ON p.id = mus.product_id")
		if params.Marking != "" {
			search := fmt.Sprintf("%%%s%%", params.Marking)
			q = q.Where("mus.marking ILIKE ? OR p.name ILIKE ?", search, search)
		}
		return q
	}

	var totalCount int64
	if err := newQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count marking_unit_sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.MarkingUnitSale
	err := newQuery().
		Select("mus.*", "p.name AS product_name").
		Order("mus.updated_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("could not get marking_unit_sales: %v", err)
		return nil, 0, domain.InternalServerError
	}

	return res, totalCount, nil
}
