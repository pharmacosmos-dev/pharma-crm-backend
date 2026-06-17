package services

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

// ProductWithStore represents a product with store-specific data
type ProductWithStore struct {
	Id            string            `gorm:"id"`
	ProductId     string            `gorm:"product_id"`
	Name          string            `gorm:"name"`
	Barcode       string            `gorm:"barcode"`
	Description   string            `gorm:"description"`
	RequiresPrescription bool       `gorm:"requires_prescription"`
	ExpiredDate   string            `gorm:"expired_date"`
	Country       string            `gorm:"vendor_country"`
	Vat           int               `gorm:"vat"`
	Photos        utils.StringArray `gorm:"type:text[]"`
	UnitPerPack   int               `gorm:"unit_per_pack"`
	IsMarking     bool              `gorm:"is_marking"`
	RetailPrice   float64           `gorm:"retail_price"`
	UnitQuantity  int               `gorm:"unit_quantity"`
	Mxik          string            `gorm:"mxik"`
	UnitCode      string            `gorm:"unit_code"`
	StoreId       string            `gorm:"store_id"`
	CategoryId    *string           `gorm:"category_id"`
	CategoryName  *string           `gorm:"category_name"`
}

func (s *Services) GetNomenclature(ctx context.Context, storeId string, page, limit int) (*domain.NomenclatureResponse, error) {
	// Query products with store-specific data
	var products []ProductWithStore

	query := `
		SELECT
			p.id AS product_id,
			p.name,
			COALESCE(NULLIF(p.barcode, ''), pb.barcode, '') AS barcode,
			p.description,
			p.photos,
			p.unit_per_pack,
			p.is_marking,
			p.mxik,
			p.unit_code,
			p.requires_prescription,
			COALESCE(sp.id::text, '') AS id,
			opp.retail_price AS retail_price,
			COALESCE(sp.unit_quantity, 0) AS unit_quantity,
			COALESCE(sp.vat, 0) AS vat,
			COALESCE(sp.expire_date::text, '') AS expired_date,
			COALESCE(sp.store_id::text, '') AS store_id,
			c.id as category_id,
			c.name as category_name,
			COALESCE(cnt.name, '') AS country
		FROM (
			SELECT DISTINCT ON (product_id) *
			FROM online_products_price
			WHERE type = 'uzum'
			ORDER BY product_id, created_at DESC
		) opp
		INNER JOIN products p ON p.id = opp.product_id
		LEFT JOIN (
			SELECT DISTINCT ON (product_id) *
			FROM store_products
			WHERE store_id = ?
			ORDER BY product_id, unit_quantity DESC
		) sp ON sp.product_id = p.id
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN countries cnt ON p.country_id = cnt.id
		LEFT JOIN (
			SELECT DISTINCT ON (product_id) product_id, barcode
			FROM product_barcodes
			ORDER BY product_id, created_at DESC
		) pb ON pb.product_id = p.id
		WHERE p.requires_prescription = false
	`
	if limit > 0 && page > 0 {
		query = query + fmt.Sprintf("LIMIT %d OFFSET %d", limit, (page-1)*limit)
	}

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

		count := 0
		if p.UnitPerPack > 0 {
			count = p.UnitQuantity / p.UnitPerPack
		}

		// Build item
		item := domain.NomenclatureItem{
			Id:         p.ProductId,
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
				General:       p.Description,
				ExpiresIn:     p.ExpiredDate,
				VendorCountry: p.Country,
			},
			Count:   count,
			Retsept: p.RequiresPrescription,
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
		StoreProductId string  `gorm:"column:store_product_id"`
		Quantity       float64 `gorm:"column:quantity"`
	}

	query := `
		SELECT
			p.id AS store_product_id,
			SUM(sp.unit_quantity) / p.unit_per_pack AS quantity
		FROM (
			SELECT DISTINCT ON (product_id) *
			FROM store_products
			WHERE store_id = ?
			ORDER BY product_id, unit_quantity DESC
		) sp
		JOIN products p ON sp.product_id = p.id
		JOIN LATERAL (
			SELECT retail_price
			FROM online_products_price
			WHERE product_id = p.id
			  AND type = 'uzum'
			ORDER BY created_at DESC
			LIMIT 1
		) osp ON true
		WHERE sp.unit_quantity > 0 AND p.requires_prescription = false
		GROUP BY p.id, p.unit_per_pack
	`
	if limit > 0 && page > 0 {
		query = query + fmt.Sprintf("LIMIT %d OFFSET %d", limit, (page-1)*limit)
	}
	err := s.db.WithContext(ctx).Raw(query, storeId).Scan(&items).Error
	if err != nil {
		s.log.Errorf("failed to fetch availability for store %s: %v", storeId, err)
		return nil, err
	}

	availabilityItems := make([]domain.AvailabilityItem, 0, len(items))
	for _, item := range items {
		availabilityItems = append(availabilityItems, domain.AvailabilityItem{
			Id:    item.StoreProductId,
			Stock: item.Quantity,
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
				Url:  baseUrl + photo,
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

// ===== UZUM ORDER SERVICE METHODS =====

// CreateUzumOrder creates a new order using the existing sales and cart_items tables
func (s *Services) CreateUzumOrder(ctx context.Context, req *domain.UzumCreateOrderRequest) (*domain.UzumCreateOrderResponse, error) {
	storeId := req.RestaurantId

	// Check if order with this EatsId already exists (idempotency)
	var existingSale domain.Sale
	err := s.db.WithContext(ctx).
		Where("vendor_order_id = ? AND service_type = ?", req.EatsId, constants.ServiceTypeUzum).
		First(&existingSale).Error
	if err == nil {
		// Order already exists, return existing order info
		return &domain.UzumCreateOrderResponse{
			OrderId: existingSale.Id,
			Result:  "OK",
		}, nil
	}

	saleId := uuid.New().String()

	// Validate stock using existing method
	cartItems, err := s.getAndCheckUzumOrderItems(ctx, storeId, saleId, req.Items)
	if err != nil {
		s.log.Errorf("uzum order stock validation failed: %v", err)
		return nil, err
	}

	// create or get customer
	customer, err := s.GetOrCreateCustomerByPhone(ctx, &domain.NoorClientInfo{
		Name:  req.DeliveryInfo.ClientName,
		Phone: req.DeliveryInfo.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}

	// Create sale using existing CreateOnlineSale
	sale, err := s.CreateOnlineSale(ctx, &domain.OnlineSaleCreate{
		Id:            saleId,
		StoreId:       storeId,
		VendorOrderId: req.EatsId,
		ServiceType:   constants.ServiceTypeUzum,
		ClientComment: req.Comment,
		CustomerId:    customer.Id,
		Items:         cartItems,
	})

	if err != nil {
		s.log.Errorf("failed to create uzum sale: %v", err)
		return nil, err
	}

	go s.NotifyOnlineOrder(storeId, sale.SaleNumber)

	return &domain.UzumCreateOrderResponse{
		OrderId: sale.Id,
		Result:  "OK",
	}, nil
}

func (s *Services) getAndCheckUzumOrderItems(
	ctx context.Context,
	storeId string,
	saleId string,
	orderItems []domain.UzumOrderItemRequest,
) ([]domain.CartItemOnlineRequest, error) {
	type tmp struct {
		SaleId         string  `gorm:"sale_id" json:"sale_id"`
		StoreProductId string  `gorm:"store_product_id" json:"store_product_id"`
		Quantity       int     `gorm:"quantity" json:"-"`
		UnitQuantity   int     `gorm:"unit_quantity" json:"-"`
		UnitPrice      float64 `gorm:"unit_price" json:"-"`
		TotalPrice     float64 `gorm:"total_price" json:"-"`
		ProductId      string  `gorm:"product_id" json:"-"`
		UnitPerPack    int     `gorm:"unit_per_pack" json:"-"`
	}
	var items []tmp
	var ids []string
	for _, item := range orderItems {
		ids = append(ids, item.Id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no items provided")
	}

	query := `
	SELECT
		? AS sale_id,
		sp.id AS store_product_id,
		sp.product_id,
		sp.unit_quantity,
		sp.unit_quantity / p.unit_per_pack AS quantity,
		COALESCE(osp.retail_price, sp.retail_price) AS unit_price,
		COALESCE(osp.retail_price, sp.retail_price) * (sp.unit_quantity / p.unit_per_pack) AS total_price,
		p.name,
		p.unit_per_pack
	FROM (
		SELECT DISTINCT ON (product_id) *
		FROM store_products
		WHERE store_id = ?
		ORDER BY product_id, unit_quantity DESC
	) sp
		JOIN products p ON sp.product_id = p.id
		LEFT JOIN LATERAL (
			SELECT retail_price
			FROM online_products_price
			WHERE product_id = sp.product_id
			  AND type = 'uzum'
			ORDER BY created_at DESC
			LIMIT 1
		) osp ON true
	WHERE sp.product_id IN (?);`

	err := s.db.WithContext(ctx).Raw(query, saleId, storeId, ids).Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get uzum order items: %v", err)
	}

	itemsMap := make(map[string]*tmp)
	for i := range items {
		itemsMap[items[i].ProductId] = &items[i]
	}

	var requestItems []domain.CartItemOnlineRequest
	for _, item := range orderItems {
		if _, ok := itemsMap[item.Id]; !ok {
			return nil, &domain.NotAdditionError{
				Data: fmt.Sprintf("item %s not found", item.Id),
			}
		}
		if float64(itemsMap[item.Id].Quantity) < item.Quantity {
			return nil, &domain.NotAdditionError{
				Data: fmt.Sprintf("not enough stock for item %s", item.Id),
			}
		}

		requestItems = append(requestItems, domain.CartItemOnlineRequest{
			StoreProductId: itemsMap[item.Id].StoreProductId,
			SaleId:         saleId,
			Quantity:       0,
			UnitQuantity:   int(item.Quantity) * itemsMap[item.Id].UnitPerPack,
			UnitPrice:      itemsMap[item.Id].UnitPrice,
			TotalPrice:     itemsMap[item.Id].UnitPrice * item.Quantity,
			ProductId:      itemsMap[item.Id].ProductId,
		})
	}
	return requestItems, nil
}

// GetUzumOrder gets an order by sale ID and returns it in YGroceryOrderV2 format
func (s *Services) GetUzumOrder(ctx context.Context, orderId string) (*domain.UzumGetOrderResponse, error) {
	var sale domain.Sale
	err := s.db.WithContext(ctx).
		Where("id = ? AND service_type = ?", orderId, constants.ServiceTypeUzum).
		Take(&sale).Error
	if err != nil {
		s.log.Errorf("failed to get uzum order %s: %v", orderId, err)
		return nil, fmt.Errorf("order not found")
	}

	// Get customer info for delivery info
	var customer domain.Customer
	if sale.CustomerId != "" {
		_ = s.db.WithContext(ctx).Where("id = ?", sale.CustomerId).Take(&customer).Error
	}

	// Get cart items with product names in a single query
	type orderItemRow struct {
		ProductId      string  `gorm:"product_id"`
		ProductName    string  `gorm:"product_name"`
		UnitPrice      float64 `gorm:"unit_price"`
		Quantity       float64 `gorm:"quantity"`
		UnitCode       string  `gorm:"unit_code"`
	}

	var rows []orderItemRow
	err = s.db.WithContext(ctx).Raw(`
		SELECT 
			ci.product_id,
			p.name AS product_name,
			ci.unit_price,
			ci.unit_quantity / p.unit_per_pack AS quantity,
			p.unit_code
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.sale_id = ?`, orderId).Scan(&rows).Error
	if err != nil {
		s.log.Errorf("failed to get uzum order items for %s: %v", orderId, err)
		return nil, fmt.Errorf("failed to get order items")
	}

	// Map to Uzum order items
	items := make([]domain.UzumOrderItemResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.UzumOrderItemResponse{
			Id:            row.ProductId,
			Name:          row.ProductName,
			Price:         row.UnitPrice,
			Quantity:      row.Quantity,
			Modifications: []domain.UzumOrderModification{},
			Promos:        []domain.UzumOrderPromo{},
			LabelCodes:    []string{row.UnitCode},
		})
	}

	resp := &domain.UzumGetOrderResponse{
		Discriminator: "uzum",
		Comment:       sale.ClientComment,
		EatsId:        sale.VendorOrderId,
		Items:         items,
		Persons:       1,
		Promos:        []domain.UzumOrderPromo{},
		RestaurantId:  sale.StoreId,
	}

	// Build delivery info from customer
	if customer.Id != "" {
		resp.DeliveryInfo = &domain.UzumOrderDeliveryInfo{
			ClientName:  customer.FullName,
			PhoneNumber: customer.Phone,
		}
	}

	// Build payment info
	paymentType := "CARD"
	if sale.Cash > 0 {
		paymentType = "CASH"
	}
	resp.PaymentInfo = &domain.UzumOrderPaymentInfo{
		ItemsCost:   sale.TotalAmount,
		PaymentType: paymentType,
	}

	return resp, nil
}

// GetUzumOrderStatus returns the Uzum-formatted status of an order
func (s *Services) GetUzumOrderStatus(ctx context.Context, orderId string) (*domain.UzumOrderStatusResponse, error) {
	var sale domain.Sale
	err := s.db.WithContext(ctx).
		Select("online_status, updated_at").
		Where("id = ? AND service_type = ?", orderId, constants.ServiceTypeUzum).
		Take(&sale).Error
	if err != nil {
		s.log.Errorf("failed to get uzum order status %s: %v", orderId, err)
		return nil, fmt.Errorf("order not found")
	}

	updatedAt := ""
	if sale.UpdatedAt != nil {
		updatedAt = sale.UpdatedAt.Format("2006-01-02T15:04:05.000Z07:00")
	}

	return &domain.UzumOrderStatusResponse{
		Status:    domain.MapOnlineStatusToUzum(sale.OnlineStatus),
		UpdatedAt: updatedAt,
	}, nil
}

// UpdateUzumOrder updates an existing Uzum order
func (s *Services) UpdateUzumOrder(ctx context.Context, orderId string, req *domain.UzumCreateOrderRequest) error {
	var sale domain.Sale
	err := s.db.WithContext(ctx).
		Where("id = ? AND service_type = ?", orderId, constants.ServiceTypeUzum).
		Take(&sale).Error
	if err != nil {
		s.log.Errorf("failed to find uzum order %s for update: %v", orderId, err)
		return fmt.Errorf("order not found")
	}

	// Cannot update cancelled or completed orders
	if sale.OnlineStatus == constants.SaleOnlineStageCanceled || sale.OnlineStatus == constants.SaleOnlineStageCompleted {
		return fmt.Errorf("order cannot be updated in current status")
	}

	updates := map[string]interface{}{}
	if req.Comment != "" {
		updates["client_comment"] = req.Comment
	}

	if len(updates) > 0 {
		err = s.db.WithContext(ctx).Model(&domain.Sale{}).Where("id = ?", orderId).Updates(updates).Error
		if err != nil {
			s.log.Errorf("failed to update uzum order %s: %v", orderId, err)
			return fmt.Errorf("failed to update order")
		}
	}

	go s.NotifyOnlineOrderUpdatedStatus(sale.StoreId, sale.SaleNumber)

	return nil
}

// CancelUzumOrder cancels an order by setting its online_status to canceled
func (s *Services) CancelUzumOrder(ctx context.Context, orderId string, req *domain.UzumCancelOrderRequest) error {
	// get order
	var sale domain.Sale
	err := s.db.WithContext(ctx).
		Where("id = ? AND service_type = ?", orderId, constants.ServiceTypeUzum).
		Take(&sale).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("order not found")
		}
		s.log.Errorf("failed to find uzum order %s for cancel: %v", orderId, err)
		return fmt.Errorf("could not get order")
	}
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	if sale.Stage == constants.SaleStageFinished {
		err = s.ReverseInventoryUpdate(ctx, tx, &sale)
		if err != nil {
			_ = tx.Rollback()
			s.log.Errorf("could not reserve inventory for uzum order %s: %v", orderId, err)
			return fmt.Errorf("failed to reverse inventory update")
		}
	}

	err = tx.WithContext(ctx).
		Exec("UPDATE sales SET online_status = ? WHERE id = ? AND service_type = ?;",
			constants.SaleOnlineStageCanceled, orderId, constants.ServiceTypeUzum).Error
	if err != nil {
		_ = tx.Rollback()
		s.log.Errorf("failed to cancel uzum order %s: %v", orderId, err)
		return fmt.Errorf("order not found or already cancelled")
	}

	if err := tx.Commit().Error; err != nil {
		s.log.Errorf("failed to commit cancel uzum order %s: %v", orderId, err)
		return fmt.Errorf("failed to cancel order")
	}

	go s.NotifyOnlineOrderCancel(sale.StoreId, sale.SaleNumber)

	return nil
}

func (s *Services) GetRestaurants(ctx context.Context, limit, page int) ([]domain.Restaurant, error) {
	qb := s.db.WithContext(ctx).
		Model(&domain.Store{}).
		Select(
			"id",
			"name",
			"phone",
			"address",
			"ST_AsText(coordinates) AS coordinates",
			"work_hours",
			"is_fullday",
			"created_at",
			"updated_at",
		).
		Where("is_active = true AND is_online_order = true")

	totalCount := int64(0)
	if err := qb.Count(&totalCount).Error; err != nil {
		s.log.Errorf("could not count stores: %v", err)
		return nil, domain.InternalServerError
	}

	if limit > 0 {
		qb = qb.Limit(limit)
	}

	if page > 0 {
		qb = qb.Offset((page - 1) * limit)
	}
	var stores []domain.Restaurant
	err := qb.Order("created_at DESC").Find(&stores).Error
	if err != nil {
		s.log.Errorf("could not get stores: %v", err)
		return nil, domain.InternalServerError
	}

	return stores, nil
}
