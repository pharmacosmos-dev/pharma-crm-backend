package v1

import (
	"time"

	"github.com/pharma-crm-backend/domain"
)

// ropAptekaReportDays — РОП hisobot va sotuvlarda ko'ra oladigan eng uzoq oraliq (kunlarda)
const ropAptekaReportDays = 30

// ropAptekaScope — РОП uchun hisoblangan filtr qiymatlari
type ropAptekaScope struct {
	StoreIds   []string
	StoreId    string
	CompanyId  string
	CompanyIds []string
	StartDate  *domain.CustomTime
}

// newRopAptekaScope — РОП (employees.role_type: ROP_APTEKA yoki regional_sales_manager)
// uchun filtr chegaralarini hisoblaydi: faqat o'ziga biriktirilgan do'konlar va oxirgi
// ropAptekaReportDays kunlik oraliq. So'rovda store_id/store_ids kelsa, ular xodimga
// biriktirilgan do'konlar bilan kesishtiriladi — ya'ni РОП o'z do'konlaridan bittasini
// yoki bir nechtasini tanlashi mumkin, lekin begonasini ko'ra olmaydi. Kesishma bo'sh
// bo'lsa (barchasi begona), so'rovdagi filtr e'tiborga olinmaydi va o'z do'konlari
// qaytadi. Xodimga umuman do'kon biriktirilmagan bo'lsa filtr o'z kompaniyasiga
// tushiriladi, aks holda hech qanday chegara qolmasdi.
// Ikkinchi qiymat — xodim РОП emasligini bildiradi.
func newRopAptekaScope(user *domain.EmployeeClaims, requestedStoreIds []string, requestedStoreId string, startDate *domain.CustomTime) (ropAptekaScope, bool) {
	if !domain.IsRopAptekaRoleType(user.RoleType) {
		return ropAptekaScope{}, false
	}

	scope := ropAptekaScope{StartDate: startDate}

	// handler yuqorida params.StoreId ga xodimning o'z do'konini yozib qo'yadi — bu mijoz
	// so'ragan filtr emas, shuning uchun uni tanlangan do'kon deb hisoblamaymiz
	if requestedStoreId != "" && requestedStoreId == user.StoreId {
		requestedStoreId = ""
	}

	if len(user.StoreIds) > 0 {
		scope.StoreIds = allowedStoreIds(user.StoreIds, requestedStoreIds, requestedStoreId)
	} else {
		scope.CompanyId = user.CompanyId
		scope.CompanyIds = []string{user.CompanyId}
	}

	limitDate := time.Now().
		AddDate(0, 0, -ropAptekaReportDays).
		Truncate(24 * time.Hour)
	customLimitDate := domain.CustomTime(limitDate)

	// start_date yuborilmagan yoki oraliqdan eski bo'lsa — chegaraga kesiladi
	if scope.StartDate == nil || scope.StartDate.GetTime().IsZero() ||
		scope.StartDate.GetTime().Before(limitDate) {
		scope.StartDate = &customLimitDate
	}

	return scope, true
}

// allowedStoreIds — so'ralgan do'konlardan faqat ruxsat etilganlarini qaytaradi.
// So'rovda do'kon ko'rsatilmagan yoki hech biri ruxsat etilganlar orasida bo'lmasa,
// ruxsat etilganlarning hammasi qaytadi.
func allowedStoreIds(allowed, requestedStoreIds []string, requestedStoreId string) []string {
	requested := requestedStoreIds
	if len(requested) == 0 && requestedStoreId != "" {
		requested = []string{requestedStoreId}
	}
	if len(requested) == 0 {
		return allowed
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}

	matched := make([]string, 0, len(requested))
	for _, id := range requested {
		if _, ok := allowedSet[id]; ok {
			matched = append(matched, id)
		}
	}
	if len(matched) == 0 {
		return allowed
	}

	return matched
}

// applyRopAptekaScope — hisobot so'rovlariga РОП chegaralarini qo'llaydi
func applyRopAptekaScope(user *domain.EmployeeClaims, params *domain.ReportQueryParam) {
	scope, ok := newRopAptekaScope(user, params.StoreIds, params.StoreId, params.StartDate)
	if !ok {
		return
	}

	params.StoreIds = scope.StoreIds
	params.StoreId = scope.StoreId
	params.CompanyId = scope.CompanyId
	params.CompanyIds = scope.CompanyIds
	params.StartDate = scope.StartDate
}

// applyRopAptekaSaleScope — sotuvlar ro'yxati va statistikasiga РОП chegaralarini qo'llaydi
func applyRopAptekaSaleScope(user *domain.EmployeeClaims, params *domain.SaleQueryParams) {
	scope, ok := newRopAptekaScope(user, params.StoreIds, params.StoreId, params.StartDate)
	if !ok {
		return
	}

	params.StoreIds = scope.StoreIds
	params.StoreId = scope.StoreId
	params.CompanyId = scope.CompanyId
	params.CompanyIds = scope.CompanyIds
	params.StartDate = scope.StartDate
}
