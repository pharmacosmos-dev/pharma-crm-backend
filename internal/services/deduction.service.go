package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

// Oylikdan ushlab qolishlar: turlar lug'ati, do'kon+oy sarlavhalari va xodim
// qatorlari.
//
// Sarlavhaning total_amount / paid_amount / status maydonlari QO'LDA
// kiritilmaydi — har bir detal o'zgarganda recalcDeduction ularni qatorlardan
// qayta hisoblaydi. Aks holda sarlavhadagi summa va uning ostidagi qatorlar
// bir-biriga mos kelmay qolardi.

// region Types

func (s *Services) CreateDeductionType(
	ctx context.Context, req *domain.DeductionTypeRequest,
) (*domain.DeductionType, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	t := domain.DeductionType{
		Id:       uuid.New().String(),
		Code:     req.Code,
		Name:     req.Name,
		IsActive: isActive,
	}
	if err := s.db.WithContext(ctx).Create(&t).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.AlreadyExistsError
		}
		s.log.Errorf("deduction: could not create type: %v", err)
		return nil, domain.InternalServerError
	}
	return &t, nil
}

func (s *Services) GetDeductionTypes(ctx context.Context) ([]domain.DeductionType, error) {
	var types []domain.DeductionType
	if err := s.db.WithContext(ctx).Order("name").Find(&types).Error; err != nil {
		s.log.Errorf("deduction: could not get types: %v", err)
		return nil, domain.InternalServerError
	}
	return types, nil
}

func (s *Services) GetDeductionTypeById(ctx context.Context, id string) (*domain.DeductionType, error) {
	var t domain.DeductionType
	if err := s.db.WithContext(ctx).Take(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("deduction: could not get type: %v", err)
		return nil, domain.InternalServerError
	}
	return &t, nil
}

func (s *Services) UpdateDeductionType(
	ctx context.Context, id string, req *domain.DeductionTypeRequest,
) (*domain.DeductionType, error) {
	updates := map[string]any{
		"code":       req.Code,
		"name":       req.Name,
		"updated_at": time.Now(),
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	result := s.db.WithContext(ctx).
		Model(&domain.DeductionType{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return nil, domain.AlreadyExistsError
		}
		s.log.Errorf("deduction: could not update type: %v", result.Error)
		return nil, domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return nil, domain.ResourceNotFoundError
	}
	return s.GetDeductionTypeById(ctx, id)
}

// DeleteDeductionType — turni o'chiradi. Unga bog'langan qatorlar bo'lsa
// o'chirilmaydi (FK NO ACTION): tarixni yo'qotmaslik uchun bunday turni
// is_active = false qilib qo'yish to'g'riroq.
func (s *Services) DeleteDeductionType(ctx context.Context, id string) error {
	var used int64
	if err := s.db.WithContext(ctx).Model(&domain.DeductionDetail{}).
		Where("deduction_type_id = ?", id).Count(&used).Error; err != nil {
		s.log.Errorf("deduction: could not check type usage: %v", err)
		return domain.InternalServerError
	}
	if used > 0 {
		return domain.InUseError
	}

	result := s.db.WithContext(ctx).Delete(&domain.DeductionType{}, "id = ?", id)
	if result.Error != nil {
		s.log.Errorf("deduction: could not delete type: %v", result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return domain.ResourceNotFoundError
	}
	return nil
}

// region Deductions

func (s *Services) CreateDeduction(
	ctx context.Context, userId string, req *domain.DeductionRequest,
) (*domain.Deduction, error) {
	// company_id do'kondan olinadi: chaqiruvchi uni yubormaydi va ikkalasi
	// bir-biriga zid bo'lib qolishi mumkin emas.
	var store struct {
		CompanyId *string `gorm:"column:company_id"`
	}
	if err := s.db.WithContext(ctx).Table("stores").
		Select("company_id").Where("id = ?", req.StoreId).Take(&store).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("deduction: could not get store: %v", err)
		return nil, domain.InternalServerError
	}

	d := domain.Deduction{
		Id:              uuid.New().String(),
		DeductionTypeId: req.DeductionTypeId,
		StoreId:         req.StoreId,
		CompanyId:       store.CompanyId,
		Year:            req.Year,
		Month:           req.Month,
		ShortageAmount:  req.ShortageAmount,
		Status:          domain.DeductionStatusOpen,
		Comment:         req.Comment,
		CreatedBy:       nullIfEmpty(userId),
	}
	if err := s.db.WithContext(ctx).Create(&d).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.AlreadyExistsError
		}
		s.log.Errorf("deduction: could not create: %v", err)
		return nil, domain.InternalServerError
	}
	return &d, nil
}

// deductionListQuery — sarlavhalar ro'yxati, bog'langan nomlar bilan.
// Tuzilishi deductionDetailListQuery bilan bir xil: nomlar JOIN orqali jonli
// olinadi, umumiy son esa COUNT(*) OVER () bilan o'sha so'rovdan keladi.
const deductionListQuery = `
SELECT
    d.*,
    COALESCE(s.name, '') AS store_name,
    COALESCE(t.name, '') AS deduction_type_name,
    COALESCE(t.code, '') AS deduction_type_code,

    COALESCE(cb.first_name, '') AS created_by_first_name,
    COALESCE(cb.last_name, '')  AS created_by_last_name,
    COALESCE(ub.first_name, '') AS updated_by_first_name,
    COALESCE(ub.last_name, '')  AS updated_by_last_name,
    COALESCE(ab.first_name, '') AS approved_by_first_name,
    COALESCE(ab.last_name, '')  AS approved_by_last_name,

    COUNT(*) OVER () AS total_count
FROM deductions d
LEFT JOIN stores s          ON s.id = d.store_id
LEFT JOIN deduction_types t ON t.id = d.deduction_type_id
LEFT JOIN employees cb      ON cb.id = d.created_by
LEFT JOIN employees ub      ON ub.id = d.updated_by
LEFT JOIN employees ab      ON ab.id = d.approved_by
WHERE (CAST(@company_id AS uuid)        IS NULL OR d.company_id        = CAST(@company_id AS uuid))
  AND (CAST(@store_id AS uuid)          IS NULL OR d.store_id          = CAST(@store_id AS uuid))
  AND (CAST(@deduction_type_id AS uuid) IS NULL OR d.deduction_type_id = CAST(@deduction_type_id AS uuid))
  AND (CAST(@status AS text)            IS NULL OR d.status            = CAST(@status AS text))
  AND (CAST(@year AS int)               IS NULL OR d.year              = CAST(@year AS int))
  AND (CAST(@month AS int)              IS NULL OR d.month             = CAST(@month AS int))
ORDER BY d.year DESC, d.month DESC, d.created_at DESC
LIMIT NULLIF(@limit, 0) OFFSET @offset`

func (s *Services) GetDeductions(
	ctx context.Context, params *domain.DeductionQueryParams,
) ([]domain.DeductionRow, int64, error) {
	var page []struct {
		domain.DeductionRow `gorm:"embedded"`

		TotalCount int64 `gorm:"column:total_count"`
	}

	err := s.db.WithContext(ctx).Raw(deductionListQuery, map[string]any{
		"company_id":        nullIfEmpty(params.CompanyId),
		"store_id":          nullIfEmpty(params.StoreId),
		"deduction_type_id": nullIfEmpty(params.DeductionTypeId),
		"status":            nullIfEmpty(params.Status),
		"year":              nullIfZero(params.Year),
		"month":             nullIfZero(params.Month),
		"limit":             params.Limit,
		"offset":            params.Offset,
	}).Scan(&page).Error
	if err != nil {
		s.log.Errorf("deduction: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var totalCount int64
	if len(page) > 0 {
		totalCount = page[0].TotalCount
	}

	rows := make([]domain.DeductionRow, len(page))
	for i := range page {
		rows[i] = page[i].DeductionRow
	}
	return rows, totalCount, nil
}

func (s *Services) GetDeductionById(ctx context.Context, id string) (*domain.Deduction, error) {
	var d domain.Deduction
	if err := s.db.WithContext(ctx).Take(&d, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("deduction: could not get: %v", err)
		return nil, domain.InternalServerError
	}
	return &d, nil
}

func (s *Services) UpdateDeduction(
	ctx context.Context, id, userId string, req *domain.DeductionUpdateRequest,
) (*domain.Deduction, error) {
	updates := map[string]any{
		"updated_by": nullIfEmpty(userId),
		"updated_at": time.Now(),
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.Approve != nil {
		if *req.Approve {
			updates["approved_by"] = nullIfEmpty(userId)
			updates["approved_at"] = time.Now()
		} else {
			updates["approved_by"] = nil
			updates["approved_at"] = nil
		}
	}

	result := s.db.WithContext(ctx).
		Model(&domain.Deduction{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		s.log.Errorf("deduction: could not update: %v", result.Error)
		return nil, domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return nil, domain.ResourceNotFoundError
	}
	return s.GetDeductionById(ctx, id)
}

// DeleteDeduction — sarlavhani va (CASCADE orqali) uning barcha qatorlarini
// o'chiradi.
func (s *Services) DeleteDeduction(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&domain.Deduction{}, "id = ?", id)
	if result.Error != nil {
		s.log.Errorf("deduction: could not delete: %v", result.Error)
		return domain.InternalServerError
	}
	if result.RowsAffected == 0 {
		return domain.ResourceNotFoundError
	}
	return nil
}

// region Details

func (s *Services) CreateDeductionDetail(
	ctx context.Context, userId string, req *domain.DeductionDetailRequest,
) (*domain.DeductionDetail, error) {
	// store_id/year/month sarlavhadan olinadi: qator sarlavhasidan boshqa oyga
	// tegishli bo'lib qolishi mumkin emas.
	parent, err := s.GetDeductionById(ctx, req.DeductionId)
	if err != nil {
		return nil, err
	}

	months := req.MonthsCount
	if months < 1 {
		months = 1
	}

	d := domain.DeductionDetail{
		Id:          uuid.New().String(),
		DeductionId: parent.Id,
		// Tur sarlavhadan meros olinadi — qator sarlavhasidan boshqa turda
		// bo'lib qolishi mumkin emas
		DeductionTypeId: parent.DeductionTypeId,
		EmployeeId:      req.EmployeeId,
		StoreId:         parent.StoreId,
		Year:            parent.Year,
		Month:           parent.Month,
		Amount:          req.Amount,
		MonthsCount:     months,
		Comment:         req.Comment,
		CreatedBy:       nullIfEmpty(userId),
	}

	// Qarz va uning to'lov jadvali birga yaratiladi: jadvalsiz qarz "qachon
	// to'lanadi" degan savolga javob bermaydi va yig'indilar noto'g'ri chiqadi.
	// Jadval oldindan tuziladi: notekis jadval berilgan bo'lsa yig'indisi qarzga
	// mos kelmasa, hech narsa yozilmasdan 400 qaytadi.
	installments, err := buildInstallments(&d, req.Installments)
	if err != nil {
		s.log.Errorf("deduction: invalid installment plan: %v", err)
		return nil, domain.BadRequestError
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&d).Error; err != nil {
			return err
		}
		if err := tx.Create(installments).Error; err != nil {
			return err
		}
		// Bitta qator qo'shishda taqsimot kamomaddan OSHIB ketmasligi
		// tekshiriladi. Tenglik talab qilinmaydi — taqsimot bosqichma-bosqich
		// to'ldirilishi mumkin; to'liq tenglik bulk chaqiruvida tekshiriladi.
		return checkDeductionOverflow(tx, parent)
	})
	if err != nil {
		if overflow, ok := err.(*domain.Error); ok {
			return nil, overflow
		}
		s.log.Errorf("deduction: could not create detail: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.recalcDeduction(ctx, parent.Id); err != nil {
		return nil, err
	}
	return &d, nil
}

// CreateDeductionDetailsBulk — bir necha xodimga bir vaqtda taqsimlash.
//
// Hammasi BITTA tranzaksiyada: sarlavhada shortage_amount berilgan bo'lsa,
// yozilgandan keyin barcha detallar yig'indisi unga teng ekani tekshiriladi.
// Teng kelmasa tranzaksiya bekor qilinadi — bitta ham qator qolmaydi.
//
// Tekshiruv "tur RECOUNT bo'lsa" emas, "shortage_amount qo'yilgan bo'lsa"
// shartiga bog'langan: yangi tur qo'shilsa ham kod o'zgartirmasdan ishlaydi,
// shtrafda esa shortage_amount 0 bo'lgani uchun tekshiruv o'zi o'chadi.
func (s *Services) CreateDeductionDetailsBulk(
	ctx context.Context, userId string, req *domain.DeductionDetailBulkRequest,
) ([]domain.DeductionDetail, error) {
	parent, err := s.GetDeductionById(ctx, req.DeductionId)
	if err != nil {
		return nil, err
	}

	// Qatorlar va jadvallar oldindan tuziladi: notekis jadval yig'indisi
	// qarzga mos kelmasa, bazaga tegmasdan 400 qaytadi.
	details := make([]domain.DeductionDetail, 0, len(req.Items))
	installments := make([]domain.DeductionInstallment, 0, len(req.Items))

	for _, item := range req.Items {
		months := item.MonthsCount
		if months < 1 {
			months = 1
		}

		d := domain.DeductionDetail{
			Id:              uuid.New().String(),
			DeductionId:     parent.Id,
			DeductionTypeId: parent.DeductionTypeId,
			EmployeeId:      item.EmployeeId,
			StoreId:         parent.StoreId,
			Year:            parent.Year,
			Month:           parent.Month,
			Amount:          item.Amount,
			MonthsCount:     months,
			Comment:         item.Comment,
			CreatedBy:       nullIfEmpty(userId),
		}

		plan, err := buildInstallments(&d, item.Installments)
		if err != nil {
			s.log.Errorf("deduction: invalid installment plan: %v", err)
			return nil, domain.BadRequestError
		}

		details = append(details, d)
		installments = append(installments, plan...)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&details).Error; err != nil {
			return err
		}
		if err := tx.Create(&installments).Error; err != nil {
			return err
		}
		// Tekshiruv YOZILGANDAN KEYIN: shunda sarlavhada oldindan turgan
		// qatorlar ham hisobga kiradi va "yana qo'shib yuborish" ham tutiladi.
		return checkDeductionDistribution(tx, parent)
	})
	if err != nil {
		if mismatch, ok := err.(*domain.Error); ok {
			return nil, mismatch
		}
		s.log.Errorf("deduction: could not create details in bulk: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.recalcDeduction(ctx, parent.Id); err != nil {
		return nil, err
	}
	return details, nil
}

// checkDeductionDistribution — detallar yig'indisi kutilgan kamomadga tengmi.
//
// shortage_amount 0 bo'lsa tekshirilmaydi. Taqqoslash tiyin aniqligida —
// float qo'shishdagi mayda farq to'g'ri taqsimotni rad etmasligi uchun.
func checkDeductionDistribution(tx *gorm.DB, parent *domain.Deduction) error {
	if parent.ShortageAmount <= 0 {
		return nil
	}

	distributed, err := sumDeductionDetails(tx, parent.Id)
	if err != nil {
		return err
	}

	if math.Round(distributed*100) != math.Round(parent.ShortageAmount*100) {
		return domain.NewError(http.StatusBadRequest, fmt.Sprintf(
			"deduction.distribution.mismatch: noto'g'ri taqsimlandi — taqsimot %.2f, kamomad %.2f",
			distributed, parent.ShortageAmount))
	}
	return nil
}

// checkDeductionOverflow — taqsimot kamomaddan oshib ketmadimi.
//
// Bitta qator qo'shilganda ishlatiladi: tenglik talab qilinmaydi (taqsimot
// bosqichma-bosqich to'ldirilishi mumkin), faqat oshib ketish to'siladi.
func checkDeductionOverflow(tx *gorm.DB, parent *domain.Deduction) error {
	if parent.ShortageAmount <= 0 {
		return nil
	}

	distributed, err := sumDeductionDetails(tx, parent.Id)
	if err != nil {
		return err
	}

	if math.Round(distributed*100) > math.Round(parent.ShortageAmount*100) {
		return domain.NewError(http.StatusBadRequest, fmt.Sprintf(
			"deduction.distribution.overflow: taqsimot kamomaddan oshib ketdi — taqsimot %.2f, kamomad %.2f",
			distributed, parent.ShortageAmount))
	}
	return nil
}

func sumDeductionDetails(tx *gorm.DB, deductionId string) (float64, error) {
	var sum float64
	err := tx.Model(&domain.DeductionDetail{}).
		Where("deduction_id = ?", deductionId).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// buildInstallments — qarzning to'lov jadvalini tuzadi.
//
// custom berilgan bo'lsa jadval aynan shundan olinadi (notekis to'lash:
// masalan birinchi oyda 2.5 mln, keyingi ikki oyda 750 mingdan). Aks holda
// summa d.MonthsCount oyga teng bo'linadi.
//
// Ikkala holatda ham to'lovlar yig'indisi qarz summasiga aniq teng bo'lishi
// ta'minlanadi: teng bo'lishda qoldiq tiyinlar oxirgi to'lovga qo'shiladi,
// notekis jadvalda esa yig'indi tekshiriladi va mos kelmasa xato qaytadi.
func buildInstallments(
	d *domain.DeductionDetail, custom []domain.DeductionInstallmentInput,
) ([]domain.DeductionInstallment, error) {
	plans, err := planInstallments(d, custom)
	if err != nil {
		return nil, err
	}

	res := make([]domain.DeductionInstallment, 0, len(plans))
	for i, p := range plans {
		res = append(res, domain.DeductionInstallment{
			Id:                uuid.New().String(),
			DeductionDetailId: d.Id,
			EmployeeId:        d.EmployeeId,
			StoreId:           d.StoreId,
			Year:              p.Year,
			Month:             p.Month,
			Seq:               i + 1,
			Amount:            p.Amount,
		})
	}
	return res, nil
}

// installmentPlan — bitta to'lovning summasi va oyi.
type installmentPlan struct {
	Amount float64
	Year   int
	Month  int
}

func planInstallments(
	d *domain.DeductionDetail, custom []domain.DeductionInstallmentInput,
) ([]installmentPlan, error) {
	if len(custom) > 0 {
		return planCustom(d, custom)
	}
	return planEqual(d), nil
}

// planEqual — summani teng bo'ladi, qoldiq tiyinlarni oxirgi to'lovga qo'shadi.
func planEqual(d *domain.DeductionDetail) []installmentPlan {
	months := d.MonthsCount
	if months < 1 {
		months = 1
	}

	per := math.Floor(d.Amount/float64(months)*100) / 100

	plans := make([]installmentPlan, 0, months)
	for i := 0; i < months; i++ {
		amount := per
		if i == months-1 {
			amount = math.Round((d.Amount-per*float64(months-1))*100) / 100
		}
		y, m := shiftMonth(d.Year, d.Month, i)
		plans = append(plans, installmentPlan{Amount: amount, Year: y, Month: m})
	}
	return plans
}

// planCustom — foydalanuvchi bergan jadvalni tekshiradi.
//
// Yig'indi qarz summasiga teng bo'lishi SHART: aks holda xodim qarzidan
// ko'proq yoki kamroq to'lab, sarlavha yig'indilari qarzga mos kelmay qolardi.
// Taqqoslash tiyin aniqligida qilinadi — float qo'shishda paydo bo'ladigan
// mayda farq xato deb hisoblanmasligi uchun.
func planCustom(
	d *domain.DeductionDetail, custom []domain.DeductionInstallmentInput,
) ([]installmentPlan, error) {
	plans := make([]installmentPlan, 0, len(custom))
	var sum float64

	for i, in := range custom {
		amount := math.Round(in.Amount*100) / 100
		sum += amount

		y, m := in.Year, in.Month
		if y == 0 || m == 0 {
			// Berilmagan bo'lsa ketma-ket: qarz oyidan boshlab
			y, m = shiftMonth(d.Year, d.Month, i)
		}
		plans = append(plans, installmentPlan{Amount: amount, Year: y, Month: m})
	}

	if math.Round(sum*100) != math.Round(d.Amount*100) {
		return nil, fmt.Errorf("installments sum %.2f does not match debt %.2f", sum, d.Amount)
	}
	return plans, nil
}

// shiftMonth — sanadan n oy keyingi yil/oyni qaytaradi.
// time.Date oy 12 dan oshsa yilni o'zi to'g'rilaydi.
func shiftMonth(year, month, n int) (int, int) {
	t := time.Date(year, time.Month(month)+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	return t.Year(), int(t.Month())
}

// deductionDetailListQuery — qarzlar ro'yxati, bog'langan nomlar bilan.
//
// Nomlar JOIN orqali jonli olinadi (snapshot emas): xodim ismi yoki turning
// nomi tahrirlansa ro'yxatda darhol ko'rinadi.
//
// Umumiy son COUNT(*) OVER () bilan o'sha so'rovdan keladi — alohida COUNT
// so'rovi yo'q va u filtrdan ajralib qolishi mumkin emas.
const deductionDetailListQuery = `
SELECT
    d.*,
    COALESCE(e.first_name, '') AS employee_first_name,
    COALESCE(e.last_name, '')  AS employee_last_name,
    COALESCE(e.full_name, '')  AS employee_full_name,
    COALESCE(s.name, '')       AS store_name,
    COALESCE(t.name, '')       AS deduction_type_name,
    COALESCE(t.code, '')       AS deduction_type_code,

    COALESCE(cb.first_name, '') AS created_by_first_name,
    COALESCE(cb.last_name, '')  AS created_by_last_name,
    COALESCE(ub.first_name, '') AS updated_by_first_name,
    COALESCE(ub.last_name, '')  AS updated_by_last_name,
    COALESCE(ab.first_name, '') AS approved_by_first_name,
    COALESCE(ab.last_name, '')  AS approved_by_last_name,

    COUNT(*) OVER ()           AS total_count
FROM deduction_details d
LEFT JOIN employees e       ON e.id = d.employee_id
LEFT JOIN stores s          ON s.id = d.store_id
LEFT JOIN deduction_types t ON t.id = d.deduction_type_id
LEFT JOIN employees cb      ON cb.id = d.created_by
LEFT JOIN employees ub      ON ub.id = d.updated_by
LEFT JOIN employees ab      ON ab.id = d.approved_by
WHERE (CAST(@deduction_id AS uuid)      IS NULL OR d.deduction_id      = CAST(@deduction_id AS uuid))
  AND (CAST(@employee_id AS uuid)       IS NULL OR d.employee_id       = CAST(@employee_id AS uuid))
  AND (CAST(@store_id AS uuid)          IS NULL OR d.store_id          = CAST(@store_id AS uuid))
  AND (CAST(@deduction_type_id AS uuid) IS NULL OR d.deduction_type_id = CAST(@deduction_type_id AS uuid))
  AND (CAST(@is_paid AS boolean)        IS NULL OR d.is_paid           = CAST(@is_paid AS boolean))
  AND (CAST(@year AS int)               IS NULL OR d.year              = CAST(@year AS int))
  AND (CAST(@month AS int)              IS NULL OR d.month             = CAST(@month AS int))
ORDER BY d.created_at DESC
LIMIT NULLIF(@limit, 0) OFFSET @offset`

func (s *Services) GetDeductionDetails(
	ctx context.Context, params *domain.DeductionDetailQueryParams,
) ([]domain.DeductionDetailRow, int64, error) {
	var page []struct {
		domain.DeductionDetailRow `gorm:"embedded"`

		TotalCount int64 `gorm:"column:total_count"`
	}

	err := s.db.WithContext(ctx).Raw(deductionDetailListQuery, map[string]any{
		"deduction_id":      nullIfEmpty(params.DeductionId),
		"employee_id":       nullIfEmpty(params.EmployeeId),
		"store_id":          nullIfEmpty(params.StoreId),
		"deduction_type_id": nullIfEmpty(params.DeductionTypeId),
		"is_paid":           params.IsPaid,
		"year":              nullIfZero(params.Year),
		"month":             nullIfZero(params.Month),
		"limit":             params.Limit,
		"offset":            params.Offset,
	}).Scan(&page).Error
	if err != nil {
		s.log.Errorf("deduction: could not get details: %v", err)
		return nil, 0, domain.InternalServerError
	}

	// Umumiy son har bir qatorda takrorlanadi, birinchisidan olinadi.
	var totalCount int64
	if len(page) > 0 {
		totalCount = page[0].TotalCount
	}

	rows := make([]domain.DeductionDetailRow, len(page))
	for i := range page {
		rows[i] = page[i].DeductionDetailRow
	}
	return rows, totalCount, nil
}

// nullIfZero — 0 ni SQL NULL'ga aylantiradi: "filtr berilmagan" degani.
func nullIfZero(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func (s *Services) GetDeductionDetailById(ctx context.Context, id string) (*domain.DeductionDetail, error) {
	var d domain.DeductionDetail
	if err := s.db.WithContext(ctx).Take(&d, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("deduction: could not get detail: %v", err)
		return nil, domain.InternalServerError
	}
	return &d, nil
}

func (s *Services) UpdateDeductionDetail(
	ctx context.Context, id, userId string, req *domain.DeductionDetailUpdateRequest,
) (*domain.DeductionDetail, error) {
	existing, err := s.GetDeductionDetailById(ctx, id)
	if err != nil {
		return nil, err
	}

	// Summa yoki oylar soni o'zgarsa jadval qaytadan tuziladi. Allaqachon
	// to'langan to'lovi bor qarzda buni qilib bo'lmaydi — to'lov tarixi
	// yo'qolardi. Bunday holatda qarzni o'chirib, yangisini yaratish kerak.
	if req.RebuildsSchedule() {
		var paidCount int64
		err := s.db.WithContext(ctx).Model(&domain.DeductionInstallment{}).
			Where("deduction_detail_id = ? AND is_paid", id).Count(&paidCount).Error
		if err != nil {
			s.log.Errorf("deduction: could not check paid installments: %v", err)
			return nil, domain.InternalServerError
		}
		if paidCount > 0 {
			return nil, domain.InUseError
		}
	}

	updates := map[string]any{
		"updated_by": nullIfEmpty(userId),
		"updated_at": time.Now(),
	}
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.MonthsCount != nil {
		updates["months_count"] = *req.MonthsCount
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.Approve != nil {
		if *req.Approve {
			updates["approved_by"] = nullIfEmpty(userId)
			updates["approved_at"] = time.Now()
		} else {
			updates["approved_by"] = nil
			updates["approved_at"] = nil
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.DeductionDetail{}).
			Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if !req.RebuildsSchedule() {
			return nil
		}

		if err := tx.Where("deduction_detail_id = ?", id).
			Delete(&domain.DeductionInstallment{}).Error; err != nil {
			return err
		}

		var updated domain.DeductionDetail
		if err := tx.Take(&updated, "id = ?", id).Error; err != nil {
			return err
		}
		rebuilt, err := buildInstallments(&updated, req.Installments)
		if err != nil {
			return err
		}
		return tx.Create(rebuilt).Error
	})
	if err != nil {
		s.log.Errorf("deduction: could not update detail: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.recalcDeductionDetail(ctx, id); err != nil {
		return nil, err
	}
	if err := s.recalcDeduction(ctx, existing.DeductionId); err != nil {
		return nil, err
	}
	return s.GetDeductionDetailById(ctx, id)
}

func (s *Services) DeleteDeductionDetail(ctx context.Context, id string) error {
	existing, err := s.GetDeductionDetailById(ctx, id)
	if err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).
		Delete(&domain.DeductionDetail{}, "id = ?", id).Error; err != nil {
		s.log.Errorf("deduction: could not delete detail: %v", err)
		return domain.InternalServerError
	}

	return s.recalcDeduction(ctx, existing.DeductionId)
}

// region Helpers

// recalcDeductionDetail — qarzning to'lov holatini jadvalidan qayta hisoblaydi.
//
// is_paid qo'lda o'rnatilmaydi: barcha to'lovlar to'langandagina true bo'ladi.
// Shuning uchun "3 oy to'ladi, 1 oy qoldi" holatida qarz ochiq ko'rinadi.
func (s *Services) recalcDeductionDetail(ctx context.Context, detailId string) error {
	const query = `
		UPDATE deduction_details d
		SET is_paid    = (t.cnt > 0 AND t.unpaid = 0),
		    paid_at    = CASE WHEN t.cnt > 0 AND t.unpaid = 0
		                      THEN COALESCE(d.paid_at, NOW()) END,
		    updated_at = NOW()
		FROM (
		    SELECT COUNT(*) AS cnt,
		           COUNT(*) FILTER (WHERE NOT is_paid) AS unpaid
		    FROM deduction_installments
		    WHERE deduction_detail_id = CAST(@id AS uuid)
		) t
		WHERE d.id = CAST(@id AS uuid)`

	if err := s.db.WithContext(ctx).Exec(query, map[string]any{"id": detailId}).Error; err != nil {
		s.log.Errorf("deduction: could not recalculate detail: %v", err)
		return fmt.Errorf("recalculate deduction detail: %w", err)
	}
	return nil
}

// recalcDeduction — sarlavha yig'indilarini qayta hisoblaydi.
//
// total_amount qarzlardan (deduction_details.amount), paid_amount esa TO'LOV
// JADVALIDAN olinadi: qarz qisman to'langan bo'lsa sarlavhada aynan to'langan
// qismi ko'rinishi kerak, butun qarz emas.
//
// status: barcha to'lovlar bajarilgan bo'lsa 'paid', aks holda 'open'.
// Qator umuman bo'lmasa 'open' — hali hech narsa yozilmagan oy yopilgan
// deb hisoblanmasligi kerak.
func (s *Services) recalcDeduction(ctx context.Context, deductionId string) error {
	const query = `
		UPDATE deductions d
		SET total_amount  = t.total,
		    paid_amount   = i.paid,
		    status        = CASE WHEN i.cnt > 0 AND i.unpaid = 0 THEN @paid ELSE @open END,
		    completed_at  = CASE WHEN i.cnt > 0 AND i.unpaid = 0 THEN COALESCE(d.completed_at, NOW()) END,
		    updated_at    = NOW()
		FROM (
		    SELECT COALESCE(SUM(amount), 0) AS total
		    FROM deduction_details
		    WHERE deduction_id = CAST(@id AS uuid)
		) t,
		(
		    SELECT COUNT(*)                                        AS cnt,
		           COUNT(*) FILTER (WHERE NOT ins.is_paid)         AS unpaid,
		           COALESCE(SUM(ins.amount) FILTER (WHERE ins.is_paid), 0) AS paid
		    FROM deduction_installments ins
		    JOIN deduction_details dd ON dd.id = ins.deduction_detail_id
		    WHERE dd.deduction_id = CAST(@id AS uuid)
		) i
		WHERE d.id = CAST(@id AS uuid)`

	err := s.db.WithContext(ctx).Exec(query, map[string]any{
		"id":   deductionId,
		"paid": domain.DeductionStatusPaid,
		"open": domain.DeductionStatusOpen,
	}).Error
	if err != nil {
		s.log.Errorf("deduction: could not recalculate totals: %v", err)
		return fmt.Errorf("recalculate deduction: %w", err)
	}
	return nil
}

// region Installments

func (s *Services) GetDeductionInstallments(
	ctx context.Context, params *domain.DeductionInstallmentQueryParams,
) ([]domain.DeductionInstallment, int64, error) {
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&domain.DeductionInstallment{})
		if params.DeductionDetailId != "" {
			q = q.Where("deduction_detail_id = ?", params.DeductionDetailId)
		}
		if params.EmployeeId != "" {
			q = q.Where("employee_id = ?", params.EmployeeId)
		}
		if params.StoreId != "" {
			q = q.Where("store_id = ?", params.StoreId)
		}
		if params.IsPaid != nil {
			q = q.Where("is_paid = ?", *params.IsPaid)
		}
		if params.Year != 0 {
			q = q.Where("year = ?", params.Year)
		}
		if params.Month != 0 {
			q = q.Where("month = ?", params.Month)
		}
		return q
	}

	var totalCount int64
	if err := newQuery().Count(&totalCount).Error; err != nil {
		s.log.Errorf("deduction: could not count installments: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.DeductionInstallment
	err := newQuery().
		Order("year, month, seq").
		Limit(payrollNoLimit(params.Limit)).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("deduction: could not get installments: %v", err)
		return nil, 0, domain.InternalServerError
	}
	return res, totalCount, nil
}

func (s *Services) GetDeductionInstallmentById(
	ctx context.Context, id string,
) (*domain.DeductionInstallment, error) {
	var i domain.DeductionInstallment
	if err := s.db.WithContext(ctx).Take(&i, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ResourceNotFoundError
		}
		s.log.Errorf("deduction: could not get installment: %v", err)
		return nil, domain.InternalServerError
	}
	return &i, nil
}

// UpdateDeductionInstallment — oylik to'lovni to'langan deb belgilaydi yoki
// tasdiqlaydi. Har o'zgarishdan keyin qarz va sarlavha qayta hisoblanadi.
func (s *Services) UpdateDeductionInstallment(
	ctx context.Context, id, userId string, req *domain.DeductionInstallmentUpdateRequest,
) (*domain.DeductionInstallment, error) {
	existing, err := s.GetDeductionInstallmentById(ctx, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"updated_by": nullIfEmpty(userId),
		"updated_at": time.Now(),
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.IsPaid != nil {
		updates["is_paid"] = *req.IsPaid
		if *req.IsPaid {
			updates["paid_at"] = time.Now()
		} else {
			updates["paid_at"] = nil
		}
	}
	if req.Approve != nil {
		if *req.Approve {
			updates["approved_by"] = nullIfEmpty(userId)
			updates["approved_at"] = time.Now()
		} else {
			updates["approved_by"] = nil
			updates["approved_at"] = nil
		}
	}

	if err := s.db.WithContext(ctx).
		Model(&domain.DeductionInstallment{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		s.log.Errorf("deduction: could not update installment: %v", err)
		return nil, domain.InternalServerError
	}

	// Zanjir: to'lov -> qarz -> sarlavha
	detail, err := s.GetDeductionDetailById(ctx, existing.DeductionDetailId)
	if err != nil {
		return nil, err
	}
	if err := s.recalcDeductionDetail(ctx, detail.Id); err != nil {
		return nil, err
	}
	if err := s.recalcDeduction(ctx, detail.DeductionId); err != nil {
		return nil, err
	}

	return s.GetDeductionInstallmentById(ctx, id)
}

// GetEmployeeDebt — xodimning qarz holati va so'ralgan oydagi to'lovi.
//
// Oylik ekranida "shu xodimda qarz bormi, bu oyda qancha ushlab qolinadi"
// degan savolga shu javob beradi.
func (s *Services) GetEmployeeDebt(
	ctx context.Context, employeeId string, year, month int,
) (*domain.EmployeeDebt, error) {
	const query = `
		SELECT
		    COALESCE(SUM(amount), 0)                                AS total_amount,
		    COALESCE(SUM(amount) FILTER (WHERE is_paid), 0)         AS paid_amount,
		    COALESCE(SUM(amount) FILTER (WHERE NOT is_paid), 0)     AS remaining_amount,
		    COUNT(*) FILTER (WHERE NOT is_paid)                     AS unpaid_count,
		    COALESCE(SUM(amount) FILTER (
		        WHERE year = @year AND month = @month), 0)          AS current_month_amount,
		    COALESCE(bool_and(is_paid) FILTER (
		        WHERE year = @year AND month = @month), false)      AS current_month_paid,
		    COALESCE(bool_and(approved_at IS NOT NULL) FILTER (
		        WHERE year = @year AND month = @month), false)      AS current_month_approved
		FROM deduction_installments
		WHERE employee_id = CAST(@employee_id AS uuid)`

	var debt domain.EmployeeDebt
	err := s.db.WithContext(ctx).Raw(query, map[string]any{
		"employee_id": employeeId,
		"year":        year,
		"month":       month,
	}).Scan(&debt).Error
	if err != nil {
		s.log.Errorf("deduction: could not get employee debt: %v", err)
		return nil, domain.InternalServerError
	}

	debt.EmployeeId = employeeId
	return &debt, nil
}
