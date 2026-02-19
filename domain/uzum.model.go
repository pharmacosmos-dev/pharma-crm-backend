package domain

// Uzum Tezkor API Models

// NomenclatureResponse represents the top-level response for nomenclature composition
type NomenclatureResponse struct {
	Categories []NomenclatureCategory `json:"categories"`
	Items      []NomenclatureItem     `json:"items"`
}

// NomenclatureCategory represents a product category
type NomenclatureCategory struct {
	Id        string              `json:"id"`
	Name      string              `json:"name"`
	Images    []NomenclatureImage `json:"images,omitempty"`
	ParentId  *string             `json:"parentId,omitempty"`
	SortOrder int                 `json:"sortOrder,omitempty"`
}

// NomenclatureItem represents a product in the nomenclature
type NomenclatureItem struct {
	Id             string                    `json:"id"`
	Name           string                    `json:"name"`
	CategoryId     string                    `json:"categoryId"`
	Barcode        NomenclatureBarcode       `json:"barcode"`
	Price          float64                   `json:"price"`
	OldPrice       *float64                  `json:"oldPrice,omitempty"`
	Vat            int                       `json:"vat,omitempty"`
	Images         []NomenclatureImage       `json:"images"`
	Description    NomenclatureDescription   `json:"description"`
	Measure        NomenclatureMeasure       `json:"measure"`
	IsCatchWeight  bool                      `json:"isCatchWeight"`
	VendorCode     string                    `json:"vendorCode"`
	SortOrder      int                       `json:"sortOrder,omitempty"`
	Location       string                    `json:"location,omitempty"`
	ServiceCodesUz *NomenclatureServiceCodes `json:"serviceCodesUz,omitempty"`
	Volume         *NomenclatureVolume       `json:"volume,omitempty"`
}

// NomenclatureBarcode represents barcode information
type NomenclatureBarcode struct {
	Type           string   `json:"type,omitempty"`
	Value          string   `json:"value"`
	Values         []string `json:"values,omitempty"`
	WeightEncoding string   `json:"weightEncoding"`
}

// NomenclatureMeasure represents measurement information
type NomenclatureMeasure struct {
	Unit    string  `json:"unit"`
	Value   int     `json:"value"`
	Quantum float64 `json:"quantum,omitempty"`
}

// NomenclatureImage represents an image with hash
type NomenclatureImage struct {
	Hash string `json:"hash"`
	Url  string `json:"url"`
}

// NomenclatureDescription represents detailed product description
type NomenclatureDescription struct {
	General             string `json:"general,omitempty"`
	VendorName          string `json:"vendorName,omitempty"`
	VendorCountry       string `json:"vendorCountry,omitempty"`
	Composition         string `json:"composition,omitempty"`
	NutritionalValue    string `json:"nutritionalValue,omitempty"`
	ExpiresIn           string `json:"expiresIn,omitempty"`
	StorageRequirements string `json:"storageRequirements,omitempty"`
	PackageInfo         string `json:"packageInfo,omitempty"`
	Purpose             string `json:"purpose,omitempty"`
}

// NomenclatureServiceCodes represents Uzbekistan-specific codes
type NomenclatureServiceCodes struct {
	MxikCodeUz    string `json:"mxikCodeUz"`
	PackageCodeUz string `json:"packageCodeUz,omitempty"`
}

// NomenclatureVolume represents product volume
type NomenclatureVolume struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

// AvailabilityResponse represents the product availability response
type AvailabilityResponse struct {
	Items []AvailabilityItem `json:"items"`
}

// AvailabilityItem represents a product's stock availability
type AvailabilityItem struct {
	Id    string  `json:"id"`
	Stock float64 `json:"stock"`
}

// UzumError represents an error in Uzum API format
type UzumError struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

// UzumErrorList is an array of errors
type UzumErrorList []UzumError

// ===== ORDER REQUEST MODELS (from Uzum Tezkor) =====

// UzumCreateOrderRequest represents the incoming order request (YGroceryOrderV2)
type UzumCreateOrderRequest struct {
	Discriminator string                 `json:"discriminator"`
	Comment       string                 `json:"comment"`
	EatsId        string                 `json:"eatsId" binding:"required"`
	Items         []UzumOrderItemRequest `json:"items" binding:"required"`
	PaymentInfo   *UzumOrderPaymentInfo  `json:"paymentInfo"`
	DeliveryInfo  *UzumOrderDeliveryInfo `json:"deliveryInfo"`
	Persons       int                    `json:"persons"`
	Promos        []UzumOrderPromo       `json:"promos"`
	RestaurantId  string                 `json:"restaurantId"`
}

// UzumOrderItemRequest represents an order item in the request
type UzumOrderItemRequest struct {
	Id            string                  `json:"id" binding:"required"`
	Name          string                  `json:"name"`
	Price         float64                 `json:"price" binding:"required"`
	Quantity      float64                 `json:"quantity" binding:"required"`
	Modifications []UzumOrderModification `json:"modifications"`
	Promos        []UzumOrderPromo        `json:"promos"`
	LabelCodes    []string                `json:"labelCodes"`
}

// UzumOrderModification represents an item modification
type UzumOrderModification struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

// UzumOrderPromo represents a promo applied to order or item
type UzumOrderPromo struct {
	Discount float64 `json:"discount"`
	Type     string  `json:"type"` // GIFT, PERCENTAGE, FIXED
}

// UzumOrderPaymentInfo represents payment information
type UzumOrderPaymentInfo struct {
	ItemsCost   float64 `json:"itemsCost"`
	PaymentType string  `json:"paymentType"` // CARD, CASH
}

// UzumOrderDeliveryInfo represents delivery information
type UzumOrderDeliveryInfo struct {
	ClientName            string `json:"clientName"`
	CourierArrivementDate string `json:"courierArrivementDate"`
	PhoneNumber           string `json:"phoneNumber"`
	ClientPhoneNumber     string `json:"clientPhoneNumber"`
}

// ===== ORDER RESPONSE MODELS =====

// UzumCreateOrderResponse represents the response after creating an order
type UzumCreateOrderResponse struct {
	OrderId string `json:"orderId"`
	Result  string `json:"result"`
}

// UzumGetOrderResponse represents the GET order response (YGroceryOrderV2 schema)
type UzumGetOrderResponse struct {
	Discriminator string                  `json:"discriminator"`
	Comment       string                  `json:"comment"`
	EatsId        string                  `json:"eatsId"`
	Items         []UzumOrderItemResponse `json:"items"`
	PaymentInfo   *UzumOrderPaymentInfo   `json:"paymentInfo,omitempty"`
	DeliveryInfo  *UzumOrderDeliveryInfo  `json:"deliveryInfo,omitempty"`
	Persons       int                     `json:"persons"`
	Promos        []UzumOrderPromo        `json:"promos"`
	RestaurantId  string                  `json:"restaurantId"`
}

// UzumOrderItemResponse represents an order item in the response
type UzumOrderItemResponse struct {
	Id            string                  `json:"id"`
	Name          string                  `json:"name"`
	Price         float64                 `json:"price"`
	Quantity      float64                 `json:"quantity"`
	Modifications []UzumOrderModification `json:"modifications"`
	Promos        []UzumOrderPromo        `json:"promos"`
	LabelCodes    []string                `json:"labelCodes"`
}

// UzumOrderStatusResponse represents order status response (GET /order/:orderId/status)
type UzumOrderStatusResponse struct {
	Status    string `json:"status"`
	Comment   string `json:"comment,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// UzumCancelOrderRequest represents a cancel order request (DELETE /order/:orderId)
type UzumCancelOrderRequest struct {
	EatsId  string `json:"eatsId" binding:"required"`
	Comment string `json:"comment"`
}

// Uzum order status constants (mapped to sale online_status)
const (
	UzumOrderStatusNew                  = "NEW"
	UzumOrderStatusAcceptedByRestaurant = "ACCEPTED_BY_RESTAURANT"
	UzumOrderStatusPostponed            = "POSTPONED"
	UzumOrderStatusCooking              = "COOKING"
	UzumOrderStatusReady                = "READY"
	UzumOrderStatusTakenByCourier       = "TAKEN_BY_COURIER"
	UzumOrderStatusDelivered            = "DELIVERED"
	UzumOrderStatusCancelled            = "CANCELLED"
)

// MapOnlineStatusToUzum maps sale online_status int to Uzum status string
func MapOnlineStatusToUzum(onlineStatus int) string {
	switch onlineStatus {
	case 1: // SaleOnlineStageNew
		return UzumOrderStatusNew
	case 2: // SaleOnlineStagePending
		return UzumOrderStatusAcceptedByRestaurant
	case 3: // SaleOnlineStageCompleted
		return UzumOrderStatusReady
	case 4: // SaleOnlineStageWaiting
		return UzumOrderStatusCooking
	case -1: // SaleOnlineStageCanceled
		return UzumOrderStatusCancelled
	default:
		return UzumOrderStatusNew
	}
}

type Restaurant struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Coordinates Point  `gorm:"column:coordinates" json:"coordinates"`
	WorkHours   string `json:"workHours"`
	IsFullday   bool   `json:"isFullday"`
}
