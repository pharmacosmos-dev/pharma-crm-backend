package domain

import (
	"encoding/json"
	"time"
)

// EmployeeFaceDescriptor — xodimning yuzini aniqlash uchun saqlanadigan descriptor(lar).
// Har bir xodim uchun bitta yozuv, "descriptor" ustunida bir nechta namuna (array of arrays) saqlanadi.
type EmployeeFaceDescriptor struct {
	Id         string          `gorm:"column:id" json:"id"`
	EmployeeId string          `gorm:"column:employee_id" json:"employee_id"`
	Descriptor json.RawMessage `gorm:"column:descriptor" json:"descriptor"`
	CreatedAt  *time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  *time.Time      `gorm:"column:updated_at" json:"updated_at"`
}

func (EmployeeFaceDescriptor) TableName() string {
	return "employee_face_descriptors"
}

// FaceDescriptorRequest — PATCH /employee/{id}/face-descriptor so'rov tanasi.
// Frontend (face-api.js) tomonidan olingan bir nechta descriptor namunasi "face_id" nomi bilan yuboriladi.
type FaceDescriptorRequest struct {
	Descriptor [][]float64 `json:"face_id" binding:"required,min=1,dive,min=1"`
}

// EmployeeFaceDescriptorListItem — GET /employee/list/face-descriptor javobi uchun.
type EmployeeFaceDescriptorListItem struct {
	Id           string          `json:"id"`
	EmployeeId   string          `json:"employee_id"`
	EmployeeName string          `json:"employee_name"`
	StoreId      *string         `json:"store_id"`
	Descriptor   json.RawMessage `json:"descriptor"`
	UpdatedAt    *time.Time      `json:"updated_at"`
}

// EmployeeFaceDescriptorQueryParams — face descriptor ro'yxati uchun filter parametrlari.
type EmployeeFaceDescriptorQueryParams struct {
	StoreId    string `form:"store_id"`
	CompanyId  string `form:"company_id"`
	EmployeeId string `form:"employee_id"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}
