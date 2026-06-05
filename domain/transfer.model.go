package domain

import "time"

type Transfer struct {
	Id                string                       `gorm:"id" json:"id"`
	PublicId          string                       `gorm:"public_id" json:"public_id"`
	FromStoreId       string                       `gorm:"from_store_id" json:"from_store_id"`
	ToStoreId         string                       `gorm:"to_store_id" json:"to_store_id"`
	Name              string                       `gorm:"name" json:"name"`
	Status            string                       `gorm:"status" json:"status"`
	ReceivedCount     float64                      `gorm:"received_count" json:"received_count"`
	ExpectedCount     float64                      `gorm:"expected_count" json:"expected_count"`
	ScannedCount      float64                      `gorm:"scanned_count" json:"scanned_count"`
	AcceptedCount     float64                      `gorm:"accepted_count" json:"accepted_count"`
	Comment           string                       `gorm:"comment" json:"comment"`
	CreatedAt         *time.Time                   `gorm:"created_at" json:"created_at"`
	UpdatedAt         *time.Time                   `gorm:"updated_at" json:"updated_at"`
	AcceptedAt        *time.Time                   `gorm:"accepted_at" json:"accepted_at"`
	ReceivedSupplySum float64                      `gorm:"received_supply_sum" json:"received_supply_sum"`
	ReceivedRetailSum float64                      `gorm:"received_retail_sum" json:"received_retail_sum"`
	ExpectedSupplySum float64                      `gorm:"expected_supply_sum" json:"expected_supply_sum"`
	ExpectedRetailSum float64                      `gorm:"expected_retail_sum" json:"expected_retail_sum"`
	AcceptedSupplySum float64                      `gorm:"accepted_supply_sum" json:"accepted_supply_sum"`
	AcceptedRetailSum float64                      `gorm:"accepted_retail_sum" json:"accepted_retail_sum"`
	ScannedSupplySum  float64                      `gorm:"scanned_supply_sum" json:"scanned_supply_sum"`
	ScannedRetailSum  float64                      `gorm:"scanned_retail_sum" json:"scanned_retail_sum"`
	IsAuto            bool                         `gorm:"is_auto" json:"is_auto"`
	DriverOffice      string                       `gorm:"driver_office" json:"driver_office"`
	DriverStoreA      string                       `gorm:"driver_store_a" json:"driver_store_a"`
	DriverStoreB      string                       `gorm:"driver_store_b" json:"driver_store_b"`
	CreatedById       string                       `gorm:"column:created_by" json:"created_by_id"`
	UpdatedById       string                       `gorm:"column:updated_by" json:"updated_by_id"`
	AcceptedById      string                       `gorm:"column:accepted_by" json:"accepted_by_id"`
	FromStore         NullStruct[TransferStore]    `gorm:"-" json:"store"`
	ToStore           NullStruct[TransferStore]    `gorm:"-" json:"to_store"`
	CreatedBy         NullStruct[TransferEmployee] `gorm:"-" json:"created_by"`
	UpdatedBy         NullStruct[TransferEmployee] `gorm:"-" json:"updated_by"`
	AcceptedBy        NullStruct[TransferEmployee] `gorm:"-" json:"accepted_by"`
}

type TransferEmployee struct {
	Id       string `gorm:"id" json:"id"`
	FullName string `gorm:"full_name" json:"full_name"`
}

type TransferStore struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type TransferStatusSummary struct {
	ReceivedCount     float64 `json:"received_count"`
	AcceptedCount     float64 `json:"accepted_count"`
	ReceivedRetailSum float64 `json:"received_retail_sum"`
	AcceptedRetailSum float64 `json:"accepted_retail_sum"`
}

// return off create request
type TransferRequest struct {
	PublicId    string `gorm:"public_id" json:"public_id"`
	Name        string `gorm:"name" json:"name"`
	FromStoreId string `gorm:"from_store_id" json:"from_store_id"`
	ToStoreId   string `gorm:"to_store_id" json:"to_store_id"`
	CreatedBy   string `gorm:"created_by" json:"created_by"`
	Status      string `gorm:"status" json:"status"`
	Comment     string `gorm:"comment" json:"comment"`
}

type TransferDetail struct {
	Id             string     `gorm:"id" json:"id"`
	TransferId     string     `gorm:"transfer_id" json:"transfer_id"`
	StoreProductId string     `gorm:"store_product_id" json:"store_product_id"`
	ProductId      string     `gorm:"product_id" json:"product_id"`
	ProductName    string     `gorm:"product_name" json:"product_name"`
	ProducerCode   string     `gorm:"producer_code" json:"producer_code"`
	ReceivedCount  float64    `gorm:"received_count" json:"received_count"`
	ExpectedCount  float64    `gorm:"expected_count" json:"expected_count"`
	AcceptedCount  float64    `gorm:"accepted_count" json:"accepted_count"`
	ScannedCount   float64    `gorm:"scanned_count" json:"scanned_count"`
	ExpireDate     *time.Time `gorm:"expire_date" json:"expire_date"`
	SerialNumber   string     `gorm:"serial_number" json:"serial_number"`
	SupplyPrice    float64    `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat float64    `gorm:"supply_price_vat" json:"supply_price_vat"`
	RetailPrice    float64    `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat float64    `gorm:"retail_price_vat" json:"retail_price_vat"`
	CreatedAt      *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"updated_at" json:"updated_at"`
	Name           string     `gorm:"name" json:"name"`
	MaterialCode   int        `gorm:"material_code" json:"material_code"`
	UnitPerPack    int        `gorm:"unit_per_pack" json:"unit_per_pack"`
	Barcode        string     `gorm:"barcode" json:"barcode"`
	ShortName      string     `gorm:"short_name" json:"short_name"`
	ReceivedSum    float64    `gorm:"received_sum" json:"received_sum"`
	ScannedSum     float64    `gorm:"scanned_sum" json:"scanned_sum"`
	IsMarking      bool       `gorm:"is_marking" json:"is_marking"`
	ScannedPack    int        `gorm:"scanned_pack" json:"scanned_pack"`
	ScannedUnit    int        `gorm:"scanned_unit" json:"scanned_unit"`
	// Pack+Unit input (NULL = old transfer, NOT NULL = new transfer with explicit pack/unit entry)
	ExpectedPack *int `gorm:"expected_pack" json:"expected_pack"`
	ExpectedUnit *int `gorm:"expected_unit" json:"expected_unit"`
	// Computed from received_count in SQL (not a stored column)
	ReceivedPack int `gorm:"received_pack" json:"received_pack"`
	ReceivedUnit int `gorm:"received_unit" json:"received_unit"`
}

type TransferDetailStatus struct {
	Scanned           int     `gorm:"scanned" json:"scanned"`
	Shortage          int     `gorm:"shortage" json:"shortage"`
	Surplus           int     `gorm:"surplus" json:"surplus"`
	All               int     `gorm:"all" json:"all"`
	New               int     `gorm:"new" json:"new"`
	Accepted          int     `gorm:"accepted" json:"accepted"`
	ShortageSupplySum float64 `gorm:"shortage_supply_sum" json:"shortage_supply_sum"`
	ShortageRetailSum float64 `gorm:"shortage_retail_sum" json:"shortage_retail_sum"`
	SurplusSupplySum  float64 `gorm:"surplus_supply_sum" json:"surplus_supply_sum"`
	SurplusRetailSum  float64 `gorm:"surplus_retail_sum" json:"surplus_retail_sum"`
}

type TransferData1C struct {
	Dok         Document            `json:"Dok"`
	Apteka      Apteka              `json:"Apteka"`
	AptekaOtkud Apteka              `json:"Apteka_otkuda"`
	Товары      []TransferProduct1C `json:"Товары"`
}
type TransferProduct1C struct {
	MaterialCode        int        `gorm:"material_code" json:"material_code"`
	Name                string     `gorm:"name" json:"name"`
	Barcode             string     `gorm:"barcode" json:"barcode"`
	Manufacturer        string     `gorm:"manufacturer" json:"manufacturer"`
	ProductSeriesNumber string     `gorm:"product_series_number" json:"product_series_number"`
	ExpireDate          *time.Time `gorm:"expire_date" json:"expire_date"`
	Quantity            float64    `gorm:"quantity" json:"quantity"`
	RetailPrice         float64    `gorm:"retail_price" json:"retail_price"`
	RetailPriceVat      float64    `gorm:"retail_price_vat" json:"retail_price_vat"`
	SupplyPrice         float64    `gorm:"supply_price" json:"supply_price"`
	SupplyPriceVat      float64    `gorm:"supply_price_vat" json:"supply_price_vat"`
	Sum                 float64    `gorm:"sum" json:"sum"`
	SumVat              float64    `gorm:"sum_vat" json:"sum_vat"`
}

// transfer details for barcode response
type TransferBarcodeResponse struct {
	Id        string `gorm:"id" json:"id"`
	Name      string `gorm:"name" json:"name"`
	ProductId string `gorm:"product_id" json:"product_id"`
}

type TransferLog struct {
	Id               int64                           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TransferId       string                          `gorm:"transfer_id" json:"transfer_id"`
	TransferDetailId string                          `gorm:"transfer_detail_id" json:"transfer_detail_id"`
	ProductId        string                          `gorm:"product_id" json:"product_id"`
	UserId           string                          `gorm:"user_id" json:"user_id"`
	TransferType     int                             `gorm:"transfer_type" json:"transfer_type"`
	Stage            int                             `gorm:"stage" json:"stage"`
	Quantity         int                             `gorm:"quantity" json:"quantity"`
	CreatedAt        *time.Time                      `gorm:"created_at" json:"created_at"`
	UpdatedAt        *time.Time                      `gorm:"updated_at" json:"updated_at"`
	Employee         NullStruct[EmployeeTransferLog] `gorm:"-" json:"employee,omitempty"`
}

type EmployeeTransferLog struct {
	Id       string `gorm:"id" json:"id"`
	FullName string `gorm:"full_name" json:"full_name"`
}


// OnecTransferProduct — return request uchun (count bilan)
type OnecTransferProduct struct {
	MaterialCode int     `json:"material_code"`
	Count        float64 `json:"count"`
}

// OnecTransferItem — transfer request uchun (count bilan, pack/unit server da hisoblanadi)
type OnecTransferItem struct {
	MaterialCode int     `json:"material_code"`
	ProductName  string  `json:"product_name"`
	Count        float64 `json:"count"`
}

type OnecTransferRequest struct {
	Name          string             `json:"name"`
	FromStoreCode int                `json:"from_store_code"`
	ToStoreCode   int                `json:"to_store_code"`
	Products      []OnecTransferItem `json:"products"`
}

type OnecTransferUnfulfilled struct {
	MaterialCode int     `json:"material_code"`
	Name         string  `json:"name"`
	Requested    float64 `json:"requested"`
	Accepted     float64 `json:"accepted"`
	Remaining    float64 `json:"remaining"`
}

type OnecTransferResponse struct {
	TransferId string `json:"transfer_id"`
}

type OnecReturnRequest struct {
	Name     string                `json:"name"`
	StoreCode  string                `json:"store_code"`
	Products []OnecTransferProduct `json:"products"`
}

type OnecReturnResponse struct {
  ReturnId    string                    `json:"return_id"`
  Unfulfilled []OnecTransferUnfulfilled `json:"unfulfilled"`
}

type SendTransferRequest struct {
	DriverName string `json:"driver_name"`
}

type ConfirmTransferRequest struct {
	DriverName string `json:"driver_name"`
}

type EditStatusToCheckingRequest struct {
	DriverName *string `json:"driver_name"`
	Type         string  `json:"type"`
}

