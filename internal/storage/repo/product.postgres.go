package repo

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/shopspring/decimal"
)

type ProductRepo struct {
	db  *sqlx.DB
	log *logger.Logger
}

func NewProductRepository(db *sqlx.DB, log *logger.Logger) *ProductRepo {
	return &ProductRepo{db: db, log: log}
}

func (r *ProductRepo) Create(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	Id := uuid.New()
	p := &domain.Product{}
	supplyPrice, err := decimal.NewFromString(req.SupplyPrice)
	if err != nil {
		return nil, err
	}
	retailPrice, err := decimal.NewFromString(req.RetailPrice)
	if err != nil {
		return nil, err
	}
	sum, err := decimal.NewFromString(req.Sum)
	if err != nil {
		return nil, err
	}
	query := `INSERT INTO products(
					id, 
					name, 
-- 					store_id, 
-- 					category_id, 
-- 					brand_id, 
-- 					supplier_id, 
-- 					unit_id, 
					product_type, 
					product_variability, 
					sku, 
					barcode, 
					main_photo, 
					photos, 
					supply_price, 
					markup, 
                    retail_price, 
                    quantity, 
                    sum, 
                    description, status, manufacturer, expire_date) VALUES (
                        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17 
--                                                                             $18, $19, $20, $21, $22                                                  
                    )
                    RETURNING 
                    id, 
					name, 
-- 					store_id, 
-- 					category_id, 
-- 					brand_id, 
-- 					supplier_id, 
-- 					unit_id, 
					product_type, 
					product_variability, 
					sku, 
					barcode, 
					main_photo, 
					photos, 
					supply_price, 
					markup, 
                    retail_price, 
                    quantity, 
                    sum, 
                    description, status, 
                    manufacturer, expire_date, 
                    created_at, updated_at`
	if err := r.db.QueryRowxContext(ctx,
		query,
		Id,
		&req.Name,
		//&req.StoreId,
		//&req.CategoryId,
		//&req.BrandId,
		//&req.SupplierId,
		//&req.UnitId,
		&req.ProductType,
		&req.ProductVariability,
		&req.Sku,
		&req.BarCode,
		&req.MainPhoto,
		pq.Array(req.Photos),
		&supplyPrice,
		&req.Markup,
		&retailPrice,
		&req.Quantity,
		&sum,
		&req.Description,
		&req.Status,
		&req.Manufacturer,
		&req.ExpireDate,
	).Scan(
		&p.Id,
		&p.Name,
		&p.ProductType,
		&p.ProductVariability,
		&p.Sku,
		&p.BarCode,
		&p.MainPhoto,
		pq.Array(&p.Photos), // Scan photos as a PostgreSQL array
		&p.SupplyPrice,
		&p.Markup,
		&p.RetailPrice,
		&p.Quantity,
		&p.Sum,
		&p.Description,
		&p.Status,
		&p.Manufacturer,
		&p.ExpireDate,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) Get(ctx context.Context, Id string) (*domain.Product, error) {
	p := &domain.Product{}
	query := `
SELECT 
		id, 
		name, 
		store_id, 
		category_id, 
		brand_id, 
		supplier_id, 
		unit_id, 
		product_type, 
		product_variability, 
		sku, 
		barcode, 
		main_photo, 
		photos, 
		supply_price, 
		markup, 
		retail_price, 
		quantity, 
		sum, 
		description, status, 
		manufacturer, expire_date, 
		created_at, updated_at
FROM products WHERE id=$1`
	if err := r.db.QueryRowxContext(ctx,
		query,
		Id).Scan(
		&p.Id,
		&p.Name,
		&p.ProductType,
		&p.ProductVariability,
		&p.Sku,
		&p.BarCode,
		&p.MainPhoto,
		pq.Array(&p.Photos), // Scan photos as a PostgreSQL array
		&p.SupplyPrice,
		&p.Markup,
		&p.RetailPrice,
		&p.Quantity,
		&p.Sum,
		&p.Description,
		&p.Status,
		&p.Manufacturer,
		&p.ExpireDate,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) GetList(ctx context.Context, param *domain.Params) ([]*domain.Product, error) {
	var products []*domain.Product
	query := `
SELECT 
		id, 
		name, 
		product_type, 
		product_variability, 
		sku, 
		barcode, 
		main_photo, 
		photos, 
		supply_price, 
		markup, 
		retail_price, 
		quantity, 
		sum, 
		description, status, 
		manufacturer, expire_date, 
		created_at, updated_at
FROM products LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryxContext(ctx, query, &param.Limit, &param.Offset)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(
			&p.Id,
			&p.Name,
			&p.ProductType,
			&p.ProductVariability,
			&p.Sku,
			&p.BarCode,
			&p.MainPhoto,
			pq.Array(&p.Photos), // Scan photos as a PostgreSQL array
			&p.SupplyPrice,
			&p.Markup,
			&p.RetailPrice,
			&p.Quantity,
			&p.Sum,
			&p.Description,
			&p.Status,
			&p.Manufacturer,
			&p.ExpireDate,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepo) Update(ctx context.Context, req *domain.Product) (*domain.Product, error) {
	p := &domain.Product{}
	query := "UPDATE products SET "
	var args []interface{}
	argCounter := 1

	// Check and append each field if it's not empty or zero
	if req.Name != "" {
		query += "name=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Name)
		argCounter++
	}
	if req.StoreId != "" {
		query += "store_id=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.StoreId)
		argCounter++
	}
	if req.CategoryId != "" {
		query += "category_id=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.CategoryId)
		argCounter++
	}
	if req.BrandId != "" {
		query += "brand_id=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.BrandId)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "supplier_id=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.SupplierId)
		argCounter++
	}

	if req.SupplierId != "" {
		query += "unit_id=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.UnitId)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "product_type=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.ProductType)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "product_variability=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.ProductVariability)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "sku=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Sku)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "barcode=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.BarCode)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "main_photo=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.MainPhoto)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "photos=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Photos)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "supply_price=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.SupplyPrice)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "markup=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Markup)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "retail_price=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.RetailPrice)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "quantity=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Quantity)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "sum=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Sum)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "description=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Description)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "status=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Status)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "manufacturer=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.Manufacturer)
		argCounter++
	}
	if req.SupplierId != "" {
		query += "expire_date=$" + strconv.Itoa(argCounter) + ", "
		args = append(args, req.ExpireDate)
		argCounter++
	}

	// Remove the trailing comma and space
	query = strings.TrimSuffix(query, ", ")

	// Add the WHERE clause to update the specific product by id
	query += " WHERE id=$" + strconv.Itoa(argCounter)
	args = append(args, req.Id)

	// Execute the query with QueryRowxContext
	if err := r.db.QueryRowxContext(ctx, query, args...).StructScan(p); err != nil {
		return nil, err
	}

	return p, nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id=$1`, id); err != nil {
		return err
	}
	return nil
}
