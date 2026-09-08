package services

import (
	"context"
	"errors"
	"fmt"
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
		Id:        uuid.New().String(),
		StoreId:   req.StoreId,
		CompanyId: store.CompanyId,
		Year:      req.Year,
		Month:     req.Month,
		Status:    domain.DeductionStatusOpen,
		Comment:   req.Comment,
		CreatedBy: nullIfEmpty(userId),
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

func (s *Services) GetDeductions(
	ctx context.Context, params *domain.DeductionQueryParams,
) ([]domain.Deduction, int64, error) {
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&domain.Deduction{})
		if params.CompanyId != "" {
			q = q.Where("company_id = ?", params.CompanyId)
		}
		if params.StoreId != "" {
			q = q.Where("store_id = ?", params.StoreId)
		}
		if params.Status != "" {
			q = q.Where("status = ?", params.Status)
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
		s.log.Errorf("deduction: could not count: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.Deduction
	err := newQuery().
		Order("year DESC, month DESC, created_at DESC").
		Limit(payrollNoLimit(params.Limit)).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("deduction: could not get list: %v", err)
		return nil, 0, domain.InternalServerError
	}
	return res, totalCount, nil
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

	d := domain.DeductionDetail{
		Id:              uuid.New().String(),
		DeductionId:     parent.Id,
		DeductionTypeId: req.DeductionTypeId,
		EmployeeId:      req.EmployeeId,
		StoreId:         parent.StoreId,
		Year:            parent.Year,
		Month:           parent.Month,
		Amount:          req.Amount,
		Comment:         req.Comment,
		CreatedBy:       nullIfEmpty(userId),
	}
	if err := s.db.WithContext(ctx).Create(&d).Error; err != nil {
		s.log.Errorf("deduction: could not create detail: %v", err)
		return nil, domain.InternalServerError
	}

	if err := s.recalcDeduction(ctx, parent.Id); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Services) GetDeductionDetails(
	ctx context.Context, params *domain.DeductionDetailQueryParams,
) ([]domain.DeductionDetail, int64, error) {
	newQuery := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&domain.DeductionDetail{})
		if params.DeductionId != "" {
			q = q.Where("deduction_id = ?", params.DeductionId)
		}
		if params.EmployeeId != "" {
			q = q.Where("employee_id = ?", params.EmployeeId)
		}
		if params.StoreId != "" {
			q = q.Where("store_id = ?", params.StoreId)
		}
		if params.DeductionTypeId != "" {
			q = q.Where("deduction_type_id = ?", params.DeductionTypeId)
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
		s.log.Errorf("deduction: could not count details: %v", err)
		return nil, 0, domain.InternalServerError
	}

	var res []domain.DeductionDetail
	err := newQuery().
		Order("created_at DESC").
		Limit(payrollNoLimit(params.Limit)).
		Offset(params.Offset).
		Find(&res).Error
	if err != nil {
		s.log.Errorf("deduction: could not get details: %v", err)
		return nil, 0, domain.InternalServerError
	}
	return res, totalCount, nil
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

	updates := map[string]any{
		"updated_by": nullIfEmpty(userId),
		"updated_at": time.Now(),
	}
	if req.DeductionTypeId != nil {
		updates["deduction_type_id"] = *req.DeductionTypeId
	}
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.IsPaid != nil {
		updates["is_paid"] = *req.IsPaid
		// paid_at to'lov belgilangan paytga qo'yiladi, bekor qilinsa tozalanadi
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
		Model(&domain.DeductionDetail{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		s.log.Errorf("deduction: could not update detail: %v", err)
		return nil, domain.InternalServerError
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

// recalcDeduction — sarlavha yig'indilarini qatorlardan qayta hisoblaydi.
//
// Har bir detal o'zgarishidan keyin chaqiriladi, shuning uchun sarlavhadagi
// summa hamisha uning ostidagi qatorlarga teng bo'ladi.
//
// status: barcha qatorlar to'langan bo'lsa 'paid', aks holda 'open'.
// Qator umuman bo'lmasa 'open' — hali hech narsa yozilmagan oy yopilgan
// deb hisoblanmasligi kerak.
func (s *Services) recalcDeduction(ctx context.Context, deductionId string) error {
	const query = `
		UPDATE deductions d
		SET total_amount  = t.total,
		    paid_amount   = t.paid,
		    status        = CASE WHEN t.cnt > 0 AND t.unpaid = 0 THEN @paid ELSE @open END,
		    completed_at  = CASE WHEN t.cnt > 0 AND t.unpaid = 0 THEN COALESCE(d.completed_at, NOW()) END,
		    updated_at    = NOW()
		FROM (
		    SELECT
		        COUNT(*)                                             AS cnt,
		        COUNT(*) FILTER (WHERE NOT is_paid)                  AS unpaid,
		        COALESCE(SUM(amount), 0)                             AS total,
		        COALESCE(SUM(amount) FILTER (WHERE is_paid), 0)      AS paid
		    FROM deduction_details
		    WHERE deduction_id = CAST(@id AS uuid)
		) t
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
