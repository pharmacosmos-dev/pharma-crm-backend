package services

import (
	"context"
	"time"

	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

// inactiveDismissRoleTypes — face-id orqali check-in qilishi shart bo'lgan
// employees.role_type'lar. Boshqa rollarga (admin, ROP, buxgalter...) tegilmaydi.
var inactiveDismissRoleTypes = []string{
	domain.RoleTypeHeadPharmacist,
	domain.RoleTypePharmacist,
	domain.RoleTypeHeadPharmacistIntern,
	domain.RoleTypePharmacyAssistant,
}

// inactiveEmployeeCandidatesSQL — inactiveDismissRoleTypes'dagi, cutoff'dan oldin
// yaratilgan va cutoff'dan beri birorta ham attendance yozuvi yo'q aktiv xodimlar.
const inactiveEmployeeCandidatesSQL = `
	SELECT
		e.id,
		e.full_name,
		e.phone,
		e.role_type,
		NULLIF(e.store_id::text, '') AS store_id,
		(SELECT MAX(al.event_at) FROM attendance_logs al WHERE al.employee_id = e.id) AS last_event_at
	FROM employees e
	WHERE e.status = @active
	  AND e.role_type IN @role_types
	  AND (e.created_at IS NULL OR e.created_at < @cutoff)
	  AND NOT EXISTS (
		SELECT 1 FROM attendance_logs al
		WHERE al.employee_id = e.id AND al.event_at >= @cutoff
	  )
`

// DismissInactiveEmployees — oxirgi days kun (hozirgi vaqtdan days*24 soat orqaga)
// ichida face-id orqali check-in/check-out qilmagan xodimlarni "dismissed" (Уволен)
// qiladi. dryRun=true bo'lsa faqat nomzodlar ro'yxati qaytadi.
func (s *Services) DismissInactiveEmployees(ctx context.Context, days int, dryRun bool, updatedBy string) (*domain.InactiveEmployeeDismissResult, error) {
	params := map[string]any{
		"active":     constants.GeneralStatusActive,
		"dismissed":  constants.GeneralStatusDismissed,
		"role_types": inactiveDismissRoleTypes,
		"cutoff":     time.Now().Add(-time.Duration(days) * 24 * time.Hour),
		"updated_by": updatedBy,
	}

	query := inactiveEmployeeCandidatesSQL
	if !dryRun {
		// Bitta statement: tanlash va yangilash orasida xodim check-in qilib qolsa,
		// UPDATE'dagi status sharti qayta tekshiradi.
		query = `
			WITH targets AS (` + inactiveEmployeeCandidatesSQL + `)
			UPDATE employees e
			SET status = @dismissed, updated_by = @updated_by, updated_at = NOW()
			FROM targets t
			WHERE e.id = t.id AND e.status = @active
			RETURNING t.id, t.full_name, t.phone, t.role_type, t.store_id, t.last_event_at
		`
	}

	employees := []domain.InactiveDismissedEmployee{}
	if err := s.db.WithContext(ctx).Raw(query, params).Scan(&employees).Error; err != nil {
		s.log.Errorf("could not dismiss inactive employees: %v", err)
		return nil, domain.InternalServerError
	}

	return &domain.InactiveEmployeeDismissResult{
		Days:           days,
		DryRun:         dryRun,
		DismissedCount: len(employees),
		Employees:      employees,
	}, nil
}
