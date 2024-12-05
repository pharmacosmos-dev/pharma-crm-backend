package domain

// Category structure
type Category struct {
	Id            string     `gorm:"primaryKey;column:id" json:"id"`
	Name          string     `gorm:"column:name" json:"name"`
	CategoryId    *string    `gorm:"column:category_id" json:"parent_id"`
	CreatedBy     *string    `gorm:"column:created_by" json:"created_by"`
	UpdatedBy     *string    `gorm:"column:updated_by" json:"updated_by"`
	DeletedBy     *string    `gorm:"column:deleted_by" json:"deleted_by"`
	IsActive      bool       `gorm:"column:is_active" json:"is_active"`
	SubCategories []Category `gorm:"foreignKey:CategoryId" json:"sub_category"`
}

// Category create request
type CategoryRequest struct {
	Id         string  `gorm:"id" json:"-"`
	Name       string  `gorm:"name" json:"name"`
	CreatedBy  *string `gorm:"column:created_by" json:"created_by"`
	CategoryId string  `gorm:"category_id" json:"category_id"`
}
