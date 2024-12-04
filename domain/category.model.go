package domain

// Category structure
type Category struct {
	Id            string     `gorm:"primaryKey;column:id" json:"id"`
	Name          string     `gorm:"column:name" json:"name"`
	CategoryId    *string    `gorm:"column:category_id" json:"parent_id"`
	SubCategories []Category `gorm:"foreignKey:CategoryId" json:"sub_category"`
}

// Category create request
type CategoryRequest struct {
	Id         string `gorm:"id" json:"-"`
	Name       string `gorm:"name" json:"name"`
	CategoryId string `gorm:"category_id" json:"category_id"`
}
