package domain

import (
	"strings"
	"time"

	"github.com/pharma-crm-backend/pkg/utils"
)

// Role type slug'lari — roles.role_type va employees.role_type uchun umumiy lug'at.
const (
	RoleTypeHeadPharmacist       = "head_pharmacist"        // Заведующий
	RoleTypePharmacist           = "pharmacist"             // Фармацевт
	RoleTypeRegionalSalesManager = "regional_sales_manager" // РОП
	RoleTypeIntern               = "intern"                 // Стажер
	RoleTypeOperator             = "operator"               // Оператор
	RoleTypeFounder              = "founder"                // Учредитель
	RoleTypeAccountant           = "accountant"             // Бухгалтер
	RoleTypeOperationsDirector   = "operations_director"    // Операционный директор
	RoleTypeAutoOrderManager     = "auto_order_manager"     // Менеджер автозаказов
	RoleTypeReturnsManager       = "returns_manager"        // Менеджер по возвратам
	RoleTypeTechnicalSupport     = "technical_support"      // Техподдержка
)

// restrictedRoleListViewers — bu role_type'dagi xodimlarga rollar ro'yxati
// to'liq ko'rinmaydi (employees.role_type bo'yicha).
var restrictedRoleListViewers = map[string]struct{}{
	RoleTypeOperator:         {},
	RoleTypeTechnicalSupport: {},
}

// RoleListVisibleRoleTypes — cheklangan xodim GET /role/list'da ko'ra oladigan
// roles.role_type qiymatlari.
var RoleListVisibleRoleTypes = []string{
	RoleTypeHeadPharmacist,
	RoleTypePharmacist,
	RoleTypeRegionalSalesManager,
	RoleTypeIntern,
}

// IsRestrictedRoleListViewer — berilgan employees.role_type uchun rollar ro'yxati
// RoleListVisibleRoleTypes bilan cheklanishi kerakligini bildiradi.
func IsRestrictedRoleListViewer(employeeRoleType string) bool {
	_, ok := restrictedRoleListViewers[strings.ToLower(strings.TrimSpace(employeeRoleType))]
	return ok
}

type Role struct {
	Id              string     `gorm:"id" json:"id"`
	PublicID        int        `gorm:"public_id" json:"public_id"`
	Name            string     `gorm:"name" json:"name"`
	PermissionCount int        `gorm:"permission_count" json:"permission_count"`
	Description     string     `gorm:"description" json:"description"`
	RoleType        *string    `gorm:"column:role_type" json:"role_type"`
	CreatedAt       *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"updated_at" json:"updated_at"`
}

// RoleRequest structure for create, update
type RoleRequest struct {
	Id          string              `gorm:"id" json:"-"`
	Name        string              `gorm:"name" json:"name"`
	Description string              `gorm:"description" json:"description"`
	RoleType    *string             `gorm:"column:role_type" json:"role_type" binding:"omitempty,max=55" example:"CASHIER"`
	Permissions []RolePermissionReq `json:"permissions"`
}

// RoleUpdateRequest structure for update.
// RoleType berilmasa (yuborilmasa yoki null bo'lsa) mavjud qiymat o'zgarmaydi.
type RoleUpdateRequest struct {
	Name        string              `gorm:"name" json:"name"`
	Description string              `gorm:"description" json:"description"`
	RoleType    *string             `gorm:"column:role_type" json:"role_type" binding:"omitempty,max=55" example:"CASHIER"`
	Permissions []RolePermissionReq `gorm:"-" json:"permissions"`
}

// RolePermissionRequest structure for create, update
type RolePermissionReq struct {
	RoleID       string   `gorm:"role_id" json:"-"`
	PermissionId string   `gorm:"permission_id" json:"parent_id"`
	IsActive     bool     `gorm:"is_active" json:"is_active"`
	ChildIds     []string `json:"children_ids"`
}

type RoleRef struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PermissionWithRoles struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Key         string               `json:"key"`
	Route       string               `json:"route"`
	Type        string               `json:"type"`
	ParentId    string               `json:"parent_id"`
	Method      utils.StringArray    `json:"method"`
	Roles       []RoleRef            `json:"roles"`
	Children    []PermissionWithRoles `json:"children"`
}

type MainPermWithRoles struct {
	ID          string               `json:"id"`
	Key         string               `json:"key"`
	Name		string               `json:"name"`
	Description string               `json:"description"`
	Permissions []PermissionWithRoles `json:"permissions"`
}
