package services

import (
	"context"

	"github.com/pharma-crm-backend/domain"
)

// GetExistingProductMaterialCodes — berilgan material_code'lardan products'da
// (o'chirilmagan) borlarini qaytaradi.
func (s *Services) GetExistingProductMaterialCodes(ctx context.Context, codes []int) (map[int]bool, error) {
	res := make(map[int]bool, len(codes))
	if len(codes) == 0 {
		return res, nil
	}

	var found []int
	err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT material_code FROM products
		WHERE material_code IN (?) AND deleted_at IS NULL`, codes).
		Scan(&found).Error
	if err != nil {
		s.log.Errorf("could not get products by material_code: %v", err)
		return nil, domain.InternalServerError
	}

	for _, code := range found {
		res[code] = true
	}
	return res, nil
}

// AttachProductPhotos — har bir material_code'ga tegishli products.photos'ga
// rasm nomlarini yozadi. overwrite=true bo'lsa eski rasmlar o'rniga yoziladi,
// aks holda oxiriga qo'shiladi.
func (s *Services) AttachProductPhotos(ctx context.Context, photos map[int][]string, overwrite bool) error {
	tx := s.db.WithContext(ctx).Begin()
	defer recoverTransaction(tx, s.log)

	for code, names := range photos {
		if len(names) == 0 {
			continue
		}

		query := `
			UPDATE products
			SET photos = COALESCE(photos, '{}') || ARRAY[?]::varchar[],
				updated_at = NOW()
			WHERE material_code = ? AND deleted_at IS NULL`
		if overwrite {
			query = `
			UPDATE products
			SET photos = ARRAY[?]::varchar[],
				updated_at = NOW()
			WHERE material_code = ? AND deleted_at IS NULL`
		}

		if err := tx.Exec(query, names, code).Error; err != nil {
			tx.Rollback()
			s.log.Errorf("could not update photos for material_code %d: %v", code, err)
			return domain.InternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("could not commit product photos: %v", err)
		return domain.InternalServerError
	}
	return nil
}
