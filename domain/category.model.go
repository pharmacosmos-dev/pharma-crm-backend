package domain

// Category structure
type Category struct {
	Id            string     `gorm:"id" json:"id"`
	Name          string     `gorm:"name" json:"name"`
	Photo         string     `gorm:"photo" json:"photo"`
	CategoryId    *string    `gorm:"category_id" json:"parent_id"`
	CreatedBy     *string    `gorm:"created_by" json:"created_by"`
	IsOpen        bool       `gorm:"is_open" json:"is_open"`
	ProductCount  int64      `gorm:"product_count" json:"product_count"`
	SubCategories []Category `gorm:"foreignKey:CategoryId" json:"sub_category"`
}

// Category create request
type CategoryRequest struct {
	Id          string             `gorm:"id" json:"-"`
	Name        string             `gorm:"name" json:"name"`
	Photo       string             `gorm:"photo" json:"photo"`
	CategoryId  *string            `gorm:"category_id" json:"-"`
	SubCategory []*CategoryRequest `gorm:"-" json:"sub_category"`
}

// Category update request
type CategoryUpdateRequest struct {
	Id            string                  `gorm:"id" json:"id"`
	Name          string                  `gorm:"name" json:"name"`
	Photo         string                  `gorm:"photo" json:"photo"`
	CategoryId    *string                 `gorm:"category_id" json:"parent_id"`
	SubCategories []CategoryUpdateRequest `gorm:"-" json:"sub_category"`
}

type CategoryProduct struct {
	CategoryId string `gorm:"category_id" json:"category_id"`
	ProductId  string `gorm:"product_id" json:"product_id"`
	IsOpen     bool   `gorm:"is_open" json:"is_open"`
}

type CategoryParams struct {
	ParentId  string `form:"parent_id"`
	Search    string `form:"search"`
	ProductId string `form:"product_id"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}
