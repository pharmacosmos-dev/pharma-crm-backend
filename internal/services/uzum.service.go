package services

import (
	"context"
	"crypto/sha1"
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

// ProductWithStore represents a product with store-specific data
type ProductWithStore struct {
	Id           string            `gorm:"id"`
	Name         string            `gorm:"name"`
	Barcode      string            `gorm:"barcode"`
	Description  string            `gorm:"description"`
	Vat          int               `gorm:"vat"`
	Photos       utils.StringArray `gorm:"type:text[]"`
	UnitPerPack  int               `gorm:"unit_per_pack"`
	IsMarking    bool              `gorm:"is_marking"`
	RetailPrice  float64           `gorm:"retail_price"`
	UnitQuantity int               `gorm:"unit_quantity"`
	Mxik         string            `gorm:"mxik"`
	UnitCode     string            `gorm:"unit_code"`
	StoreId      string            `gorm:"store_id"`
	CategoryId   *string           `gorm:"category_id"`
	CategoryName *string           `gorm:"category_name"`
}

func (s *Services) GetNomenclature(ctx context.Context, storeId string, page, limit int) (*domain.NomenclatureResponse, error) {
	// Query products with store-specific data
	var products []ProductWithStore

	query := `
		SELECT 
			p.id, 
			p.name, 
			p.barcode, 
			p.description, 
			p.photos, 
			p.unit_per_pack, 
			p.is_marking,
			p.mxik,
			p.unit_code,
			sp.retail_price, 
			sp.unit_quantity, 
			sp.vat, 
			sp.store_id,
			c.id as category_id, 
			c.name as category_name
		FROM products p
		INNER JOIN store_products sp ON p.id = sp.product_id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE sp.store_id = ? AND sp.unit_quantity > 0
	`
	query = query + fmt.Sprintf("LIMIT %d OFFSET %d", limit, (page-1)*limit)
	err := s.db.WithContext(ctx).Raw(query, storeId).Scan(&products).Error
	if err != nil {
		s.log.Errorf("failed to fetch nomenclature for store %s: %v", storeId, err)
		return nil, err
	}

	// Fetch all categories for this store
	categoryMap := make(map[string]*domain.NomenclatureCategory)
	var categories []domain.Category

	err = s.db.WithContext(ctx).Find(&categories).Error
	if err != nil {
		s.log.Errorf("failed to fetch categories: %v", err)
		return nil, err
	}

	// Build category map
	for _, cat := range categories {
		categoryMap[cat.Id] = &domain.NomenclatureCategory{
			Id:        cat.Id,
			Name:      cat.Name,
			Images:    buildCategoryImages(s.cfg.FileBaseURL + cat.Photo),
			ParentId:  cat.CategoryId,
			SortOrder: 100, // Default sort order
		}
	}

	// Build nomenclature items
	items := make([]domain.NomenclatureItem, 0, len(products))

	for _, p := range products {
		// Build images
		images := buildProductImages(s.cfg.FileBaseURL, p.Photos)

		// Determine category ID
		categoryId := ""
		if p.CategoryId != nil {
			categoryId = *p.CategoryId
		}

		// Build item
		item := domain.NomenclatureItem{
			Id:         p.Id,
			Name:       p.Name,
			CategoryId: categoryId,
			Barcode: domain.NomenclatureBarcode{
				Type:           "ean13",
				Value:          p.Barcode,
				WeightEncoding: "none",
			},
			Price:  p.RetailPrice,
			Vat:    p.Vat,
			Images: images,
			Description: domain.NomenclatureDescription{
				General: p.Description,
			},
			Measure: domain.NomenclatureMeasure{
				Unit:    "GRM",
				Value:   1000,
				Quantum: 1.0,
			},
			IsCatchWeight: false,
			VendorCode:    p.Id,
			SortOrder:     100,
			ServiceCodesUz: &domain.NomenclatureServiceCodes{
				MxikCodeUz:    p.Mxik,
				PackageCodeUz: p.UnitCode,
			},
		}

		items = append(items, item)
	}

	// Convert category map to slice
	categorySlice := make([]domain.NomenclatureCategory, 0, len(categoryMap))
	for _, cat := range categoryMap {
		categorySlice = append(categorySlice, *cat)
	}

	return &domain.NomenclatureResponse{
		Categories: categorySlice,
		Items:      items,
	}, nil
}

func (s *Services) GetAvailability(ctx context.Context, storeId string, page, limit int) (*domain.AvailabilityResponse, error) {
	var items []struct {
		ProductId    string
		UnitQuantity float64
	}

	query := `
		SELECT product_id, unit_quantity
		FROM store_products
		WHERE store_id = ? AND unit_quantity > 0
	`

	query = query + fmt.Sprintf("LIMIT %d OFFSET %d", limit, (page-1)*limit)
	err := s.db.WithContext(ctx).Raw(query, storeId).Scan(&items).Error
	if err != nil {
		s.log.Errorf("failed to fetch availability for store %s: %v", storeId, err)
		return nil, err
	}

	availabilityItems := make([]domain.AvailabilityItem, 0, len(items))
	for _, item := range items {
		availabilityItems = append(availabilityItems, domain.AvailabilityItem{
			Id:    item.ProductId,
			Stock: item.UnitQuantity,
		})
	}

	return &domain.AvailabilityResponse{
		Items: availabilityItems,
	}, nil
}

// Helper functions
func buildProductImages(baseUrl string, photos []string) []domain.NomenclatureImage {
	images := make([]domain.NomenclatureImage, 0, len(photos))
	for _, photo := range photos {
		if photo != "" {
			images = append(images, domain.NomenclatureImage{
				Hash: generateSHA1(photo),
				Url:  photo,
			})
		}
	}
	return images
}

func buildCategoryImages(photo string) []domain.NomenclatureImage {
	if photo == "" {
		return []domain.NomenclatureImage{}
	}
	return []domain.NomenclatureImage{
		{
			Hash: generateSHA1(photo),
			Url:  photo,
		},
	}
}

func generateSHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
