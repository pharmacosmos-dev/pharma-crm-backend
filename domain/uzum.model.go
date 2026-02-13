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
