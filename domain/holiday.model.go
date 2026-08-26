package domain

import "time"

// Holiday — dam olish kuni (bayram). Payroll'da ish kunlari shu jadval orqali
// hisoblanadi: kalendar kun − yakshanba − shu yerdagi sanalar.
//
// Yakshanba kodda aniqlanadi (SQL'da EXTRACT(DOW) = 0), bu jadvalga faqat
// bayramlar kiritiladi. 31-dekabr va 21-mart 2025-2035 uchun migratsiyada
// oldindan to'ldirilgan; Hayit sanalari har yili o'zgargani sababli ularni
// qo'lda kiritish kerak.
type Holiday struct {
	Id        string     `json:"id" gorm:"column:id;primaryKey"`
	Date      string     `json:"date" gorm:"column:date"`
	Name      string     `json:"name" gorm:"column:name"`
	CreatedAt *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (Holiday) TableName() string {
	return "holidays"
}

// HolidayRequest — yaratish va yangilash uchun.
type HolidayRequest struct {
	Date string `json:"date" binding:"required"` // YYYY-MM-DD
	Name string `json:"name" binding:"required"`
}

type HolidayQueryParams struct {
	Year   int    `form:"year"`  // faqat shu yildagilar
	Month  int    `form:"month"` // faqat shu oydagilar (year bilan birga)
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
	Search string `form:"search"`
}
