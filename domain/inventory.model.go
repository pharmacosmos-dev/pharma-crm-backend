package domain

import "time"

// InventoryParam structure
type InventoryParam struct {
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	InventoryId string `form:"inventory_id"`
	StoreId     string `form:"store_id"`
	CompanyId   string `form:"company_id"`
	Type        string `form:"type"`
	Status      string `form:"status"`
	Search      string `form:"search"`
	ProductId   string `form:"product_id"`
	Order       string `form:"order"`
}

// Inventory structure
type Inventory struct {
	Id              string                        `gorm:"id" json:"id"`
	PublicId        string                        `gorm:"public_id" json:"public_id"`
	StoreId         string                        `gorm:"store_id" json:"store_id"`
	Name            string                        `gorm:"name" json:"name"`
	InventoryType   string                        `gorm:"inventory_type" json:"type"`
	Status          string                        `gorm:"status" json:"status"`
	CreatedById     string                        `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById     string                        `gorm:"column:accepted_by" json:"updated_by_id"`
	CurrentCount    float64                       `gorm:"current_count" json:"current_count"`
	CurrentSum      float64                       `gorm:"current_sum" json:"current_sum"`
	FactCount       float64                       `gorm:"fact_count" json:"fact_count"`
	FactSum         float64                       `gorm:"fact_sum" json:"fact_sum"`
	DifferenceCount float64                       `gorm:"difference_count" json:"difference_count"`
	DifferenceSum   float64                       `gorm:"difference_sum" json:"difference_sum"`
	CreatedAt       *time.Time                    `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time                    `gorm:"updated_at" json:"updated_at"`
	Store           NullStruct[InventoryStore]    `gorm:"-" json:"store"`
	CreatedBy       NullStruct[InventoryEmployee] `gorm:"-" json:"created_by"`
	UpdatedBy       NullStruct[InventoryEmployee] `gorm:"-" json:"updated_by"`
}

type InventoryStore struct {
	Id   string `gorm:"id" json:"id"`
	Name string `gorm:"name" json:"name"`
}

type InventoryEmployee struct {
	Id       string `gorm:"id" json:"id"`
	FullName string `gorm:"full_name" json:"full_name"`
}

type InventoryStatusSummary struct {
	CurrentSum      float64 `json:"current_sum"`
	FactSum         float64 `json:"fact_sum"`
	DifferenceSum   float64 `json:"difference_sum"`
	CurrentCount    float64 `json:"current_count"`
	FactCount       float64 `json:"fact_count"`
	DifferenceCount float64 `json:"difference_count"`
}

// InventoryRequest structure
type InventoryRequest struct {
	PublicId  string `gorm:"public_id" json:"public_id"`
	StoreId   string `gorm:"store_id" json:"store_id"`
	Name      string `gorm:"name" json:"name"`
	Type      string `gorm:"type" json:"type"` // FULL || PARTIAL || IMPORT
	CreatedBy string `gorm:"created_by" json:"created_by"`
}

// InventoryRequest structure
type InventoryDetail struct {
	Id                 string     `gorm:"id" json:"id"`
	InventoryId        string     `gorm:"inventory_id" json:"inventory_id"`
	ProductId          string     `gorm:"product_id" json:"product_id"`
	MaterialCode       int        `gorm:"material_code" json:"material_code"`
	UnitPerPack        int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	Name               string     `gorm:"name" json:"name"`
	ProducerName       string     `gorm:"producer_name" json:"producer_name"`
	Barcode            string     `gorm:"barcode" json:"barcode"`
	CurrentQuantity    float64    `gorm:"current_quantity" json:"current_quantity"`
	CurrentUnit        float64    `gorm:"current_unit" json:"current_unit"`
	FactQuantity       float64    `gorm:"fact_quantity" json:"fact_quantity"`
	FactUnit           float64    `gorm:"fact_unit" json:"fact_unit"`
	DifferenceQuantity float64    `gorm:"difference_quantity" json:"difference_quantity"`
	DifferenceUnit     float64    `gorm:"difference_unit" json:"difference_unit"`
	CurrentSum         float64    `gorm:"current_sum" json:"current_sum"`
	FactSum            float64    `gorm:"fact_sum" json:"fact_sum"`
	DifferenceSum      float64    `gorm:"difference_sum" json:"difference_sum"`
	SupplyPrice        float64    `gorm:"supply_price" json:"supply_price"`
	RetailPrice        float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate         *time.Time `gorm:"expire_date" json:"expire_date"`
	TotalCount         int64      `gorm:"total_count" json:"-"`
}

// InventoryDetailRequest structure
type InventoryDetailRequest struct {
	InventoryId string `gorm:"inventory_id" json:"inventory_id"`
	ProductId   string `gorm:"product_id" json:"product_id"`
}

type InventoryDetailStatus struct {
	Scanned           float64 `gorm:"scanned" json:"scanned"`
	Shortage          float64 `gorm:"shortage" json:"shortage"`
	Surplus           float64 `gorm:"surplus" json:"surplus"`
	All               float64 `gorm:"all" json:"all"`
	New               float64 `gorm:"new" json:"new"`
	Accepted          float64 `gorm:"accepted" json:"accepted"`
	ShortageSupplySum float64 `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64 `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64 `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64 `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
}

type InventoryAddProduct struct {
	FactQuantity float64 `gorm:"fact_quantity" json:"fact_quantity"`
	FactUnit     float64 `gorm:"fact_unit" json:"fact_unit"`
	Barcode      string  `gorm:"barcode" json:"barcode"`
	ExpireDate   string  `gorm:"expire_date" json:"expire_date"`
	RetailPrice  float64 `gorm:"retail_price" json:"retail_price"`
	Id           string  `gorm:"id" json:"id"`
}

type InventoryDetailSum struct {
	TotalFactCount     float64 `gorm:"total_fact_count" json:"total_fact_count"`
	TotalFactSum       float64 `gorm:"total_fact_sum" json:"total_fact_sum"`
	TotalCurrentCount  float64 `gorm:"total_current_count" json:"total_current_count"`
	TotalCurrentSum    float64 `gorm:"total_current_sum" json:"total_current_sum"`
	TotalDifferenceSum float64 `gorm:"total_difference_sum" json:"total_difference_sum"`
	Scanned            int     `gorm:"scanned" json:"scanned"`
	Shortage           int     `gorm:"shortage" json:"shortage"`
	All                int     `gorm:"all" json:"all"`
	Surplus            int     `gorm:"surplus" json:"surplus"`
	Accepted           int     `gorm:"accepted" json:"accepted"`
}

type InventoryDetailStats struct {
	Scanned  int `gorm:"scanned" json:"scanned"`
	Shortage int `gorm:"shortage" json:"shortage"`
	All      int `gorm:"all" json:"all"`
	Surplus  int `gorm:"surplus" json:"surplus"`
	Accepted int `gorm:"accepted" json:"accepted"`
}

type InventoryDetailTotalStats struct {
	TotalCurrentSum          float64 `gorm:"total_current_sum" json:"total_current_sum"`
	TotalFactSum             float64 `gorm:"total_fact_sum" json:"total_fact_sum"`
	TotalDifferenceSum       float64 `gorm:"total_difference_sum" json:"total_difference_sum"`
	TotalImportCurrentSum    float64 `gorm:"total_import_current_sum" json:"total_import_current_sum"`
	TotalImportFactSum       float64 `gorm:"total_import_fact_sum" json:"total_import_fact_sum"`
	TotalImportDifferenceSum float64 `gorm:"total_import_difference_sum" json:"total_import_difference_sum"`
	Scanned                  int     `gorm:"scanned" json:"scanned"`
	Shortage                 int     `gorm:"shortage" json:"shortage"`
	All                      int     `gorm:"all" json:"all"`
	Surplus                  int     `gorm:"surplus" json:"surplus"`
	Accepted                 int     `gorm:"accepted" json:"accepted"`
}

// 1C request Structure
type InventoryProduct1C struct {
	MaterialCode        int        `gorm:"material_code" json:"material_code"`
	Name                string     `gorm:"name" json:"name"`
	Barcode             string     `gorm:"barcode" json:"barcode"`
	Manufacturer        string     `gorm:"manufacturer" json:"manufacturer"`
	ProductSeriesNumber string     `gorm:"product_series_number" json:"product_series_number"`
	ExpireDate          *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity            float64    `gorm:"quantity" json:"quantity"`
	QuantityInventar    float64    `gorm:"quantity_inventar" json:"quantity_inventar"`
	RetailPrice         float64    `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat      float64    `gorm:"retail_price_vat" json:"retail_price_vat"`
	SupplyPrice         float64    `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat      float64    `gorm:"supply_price_vat" json:"supply_price_vat"`
	Sum                 float64    `gorm:"sum" json:"sum"`
	SumVat              float64    `gorm:"sum_vat" json:"sum_vat"`
}

type InventoryData1C struct {
	Dok    Document             `json:"Dok"`
	Apteka Apteka               `json:"Apteka"`
	Товары []InventoryProduct1C `json:"Товары"`
}

// inventory product price options
type InventoryPriceOption struct {
	ProductId   string     `gorm:"product_id" json:"product_id"`
	RetailPrice float64    `gorm:"retail_price" json:"retail_price"`
	ExpireDate  *time.Time `gorm:"expire_date" json:"expire_date"`
}

// inventory helper
type InventoryHelper struct {
	Method   string `gorm:"method" json:"method"`
	Payload  any    `gorm:"payload" json:"payload"`
	Response any    `gorm:"response" json:"response"`
	Action   string `gorm:"action" json:"action"`
	DocDate  string `gorm:"doc_date" json:"doc_date"`
	DocNum   string `gorm:"doc_num" json:"doc_num"`
	Status   string `gorm:"status" json:"status"`
}

// InventoryMovementParam inventory'dagi productlarning inventory'gacha bo'lgan harakatlari uchun query params.
// StoreId va CompanyId so'rovdan olinmaydi, handler foydalanuvchi huquqidan to'ldiradi.
type InventoryMovementParam struct {
	InventoryId string `form:"inventory_id"`
	ProductId   string `form:"product_id"`
	Search      string `form:"search"`
	OnlyDiff    bool   `form:"only_diff"`
	Type        string `form:"type"`
	Order       string `form:"order"`
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	StoreId     string `form:"-"`
	CompanyId   string `form:"-"`
}

// InventoryMovementHeader hisob-kitob qilinayotgan inventory.
// CutoffAt = imports.created_at: received_count snapshot'i shu tranzaksiyada olinadi,
// shu vaqtdan keyingi harakatlar hisobga kirmaydi.
type InventoryMovementHeader struct {
	Id            string     `gorm:"id" json:"id"`
	PublicId      int        `gorm:"public_id" json:"public_id"`
	Name          string     `gorm:"name" json:"name"`
	InventoryType string     `gorm:"inventory_type" json:"type"`
	Status        string     `gorm:"status" json:"status"`
	StoreId       string     `gorm:"store_id" json:"store_id"`
	StoreName     string     `gorm:"store_name" json:"store_name"`
	CompanyId     string     `gorm:"company_id" json:"-"`
	CutoffAt      *time.Time `gorm:"cutoff_at" json:"cutoff_at"`
}

// InventoryMovementItem bitta product bo'yicha inventory natijasi va inventory'gacha hisoblangan qoldiq.
// Barcha miqdorlar dona (unit) hisobida.
type InventoryMovementItem struct {
	ProductId           string                  `json:"product_id"`
	MaterialCode        int                     `json:"material_code"`
	Name                string                  `json:"name"`
	Barcode             string                  `json:"barcode"`
	UnitPerPack         int                     `json:"unit_per_pack"`
	Inventory           InventoryMovementCounts `json:"inventory"`
	Movements           InventoryMovementTotals `json:"movements"`
	CalculatedQuantity  float64                 `json:"calculated_quantity"`
	Difference          *float64                `json:"difference"`
	UntrackedQuantity   float64                 `json:"untracked_quantity"`
	StockAfterInventory *float64                `json:"stock_after_inventory"`
	FirstMovementAt     *time.Time              `json:"first_movement_at"`
	LastMovementAt      *time.Time              `json:"last_movement_at"`
}

// InventoryMovementCounts inventory'ning o'zidagi sonlar (import_details yig'indisi).
type InventoryMovementCounts struct {
	ReceivedCount float64  `json:"received_count"`
	ScannedCount  float64  `json:"scanned_count"`
	Difference    *float64 `json:"difference"`
	IsCounted     bool     `json:"is_counted"`
}

// InventoryMovementTotals inventory'gacha bo'lgan harakatlar yig'indisi (musbat sonlar).
type InventoryMovementTotals struct {
	ImportQuantity      float64 `gorm:"import_quantity" json:"import_quantity"`
	SoldQuantity        float64 `gorm:"sold_quantity" json:"sold_quantity"`
	ReturnedQuantity    float64 `gorm:"returned_quantity" json:"returned_quantity"`
	TransferInQuantity  float64 `gorm:"transfer_in_quantity" json:"transfer_in_quantity"`
	TransferOutQuantity float64 `gorm:"transfer_out_quantity" json:"transfer_out_quantity"`
	VozvratQuantity     float64 `gorm:"vozvrat_quantity" json:"vozvrat_quantity"`
	InventoryPlusCount  float64 `gorm:"inventory_plus_count" json:"inventory_plus_count"`
	InventoryMinusCount float64 `gorm:"inventory_minus_count" json:"inventory_minus_count"`
}

// InventoryMovementHistoryItem bitta hujjat bo'yicha product harakati.
// Quantity ishorali: kirim +, chiqim -. Balance - shu hujjatdan keyingi hisoblangan qoldiq.
type InventoryMovementHistoryItem struct {
	Type         string    `gorm:"type" json:"type"`
	DocumentId   string    `gorm:"document_id" json:"document_id"`
	PublicId     string    `gorm:"public_id" json:"public_id"`
	Status       string    `gorm:"status" json:"status"`
	Counterparty string    `gorm:"counterparty" json:"counterparty"`
	MovementAt   time.Time `gorm:"movement_at" json:"movement_at"`
	Quantity     float64   `gorm:"quantity" json:"quantity"`
	Balance      float64   `gorm:"balance" json:"balance"`
	TotalCount   int64     `gorm:"total_count" json:"-"`
}
