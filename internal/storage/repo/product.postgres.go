package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ProductRepo struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewProductRepository(db *gorm.DB, log *logger.Logger) *ProductRepo {
	return &ProductRepo{db: db, log: log}
}

func (r *ProductRepo) Create(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	// Generate new UUID for the product
	req.Id = uuid.New().String()

	// Convert `SupplyPrice`, `RetailPrice`, and `Sum` to decimal types
	req.SupplyPrice = decimal.NewFromFloat(req.SupplyPrice).InexactFloat64()
	req.RetailPrice = decimal.NewFromFloat(req.RetailPrice).InexactFloat64()
	req.Sum = decimal.NewFromFloat(req.Sum).InexactFloat64()

	// Use GORM to create the product
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		r.log.Error("Failed to create product:", err)
		return nil, err
	}

	return req, nil
}

func (r *ProductRepo) Get(ctx context.Context, id string) (*domain.Product, error) {
	p := &domain.Product{}

	// Fetch the product by its ID
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(p).Error; err != nil {
		r.log.Error("Failed to get product: ", err)
		return nil, err
	}

	return p, nil
}

func (r *ProductRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Product, error) {
	var products []*domain.Product

	// Use GORM's Limit and Offset for pagination
	if err := r.db.WithContext(ctx).
		Limit(param.Limit).
		Offset(param.Offset).
		Find(&products).Error; err != nil {
		r.log.Error("Failed to get product list: ", err)
		return nil, err
	}

	return products, nil
}

func (r *ProductRepo) Update(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	// Fetch the existing product
	p := &domain.Product{}
	if err := r.db.WithContext(ctx).Where("id = ?", req.Id).First(p).Error; err != nil {
		r.log.Error("Product not found: ", err)
		return nil, err
	}

	// Update the fields that are not empty or zero
	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.StoreId != "" {
		updates["store_id"] = req.StoreId
	}
	if req.CategoryId != "" {
		updates["category_id"] = req.CategoryId
	}
	if req.BrandId != "" {
		updates["brand_id"] = req.BrandId
	}
	if req.SupplierId != "" {
		updates["supplier_id"] = req.SupplierId
	}
	if req.UnitId != "" {
		updates["unit_id"] = req.UnitId
	}
	if req.ProductType != "" {
		updates["product_type"] = req.ProductType
	}
	if req.ProductVariability != "" {
		updates["product_variability"] = req.ProductVariability
	}
	if req.Sku != "" {
		updates["sku"] = req.Sku
	}
	if req.BarCode != "" {
		updates["barcode"] = req.BarCode
	}
	if req.MainPhoto != "" {
		updates["main_photo"] = req.MainPhoto
	}
	if len(req.Photos) > 0 {
		updates["photos"] = req.Photos
	}
	if req.SupplyPrice != 0 {
		updates["supply_price"] = req.SupplyPrice
	}
	if req.Markup != 0 {
		updates["markup"] = req.Markup
	}
	if req.RetailPrice != 0 {
		updates["retail_price"] = req.RetailPrice
	}
	if req.Quantity != 0 {
		updates["quantity"] = req.Quantity
	}
	if req.Sum != 0 {
		updates["sum"] = req.Sum
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Manufacturer != "" {
		updates["manufacturer"] = req.Manufacturer
	}

	// Perform the update
	if err := r.db.WithContext(ctx).Model(p).Updates(updates).Error; err != nil {
		r.log.Error("Failed to update product: ", err)
		return nil, err
	}

	return p, nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Product{}).Error; err != nil {
		r.log.Error("Failed to delete product: ", err)
		return err
	}
	return nil
}

