package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pharma-crm-backend/domain"
)

// payrollImportReplacer — ismlardagi apostrof va harf variantlarini bir xil ko'rinishga keltiradi
// (O‘G‘LI / O'G'LI / O`G`LI), aks holda bir xil xodim topilmay qolardi.
var payrollImportReplacer = strings.NewReplacer(
	"‘", "'", "’", "'", "ʻ", "'", "ʼ", "'", "`", "'", "´", "'", "\"", "'", "Ё", "Е", "ё", "е",
)

// payrollImportKey — filial yoki ism solishtirish kaliti: katta harf, bitta bo'sh joy.
func payrollImportKey(s string) string {
	s = strings.ToUpper(payrollImportReplacer.Replace(s))
	return strings.Join(strings.Fields(s), " ")
}

type payrollImportCandidate struct {
	Id                string  `gorm:"column:id"`
	StoreName         string  `gorm:"column:store_name"`
	SnapStoreName     string  `gorm:"column:snap_store_name"`
	FullName          string  `gorm:"column:full_name"`
	FirstName         string  `gorm:"column:first_name"`
	LastName          string  `gorm:"column:last_name"`
	AdvanceCardAmount float64 `gorm:"column:advance_card_amount"`
	AdvanceCashAmount float64 `gorm:"column:advance_cash_amount"`
}

// ImportPayrollAdvances — Excel'dan kelgan avanslarni shu oyning payroll qatorlariga yozadi.
//
// Excel'da payroll id yo'q, shuning uchun xodim oy + filial nomi + ism bo'yicha
// topiladi (payroll qatoridagi full_name, yoki first_name + last_name). Aynan bitta
// qator mos kelsa, avans UpdateEmployeePayrollAdvance orqali yoziladi — net_pay
// va boshqa summalar qo'lda tahrirlashdagi bilan bir xil qayta hisoblanadi.
//
// overwrite=false bo'lsa avansi allaqachon bor qatorlar o'tkazib yuboriladi.
// dryRun=true bo'lsa hech narsa yozilmaydi, faqat natija qaytadi.
func (s *Services) ImportPayrollAdvances(
	ctx context.Context, year, month int, companyId, storeId, updatedBy string,
	rows []domain.PayrollAdvanceImportRow, dryRun, overwrite bool,
) (*domain.PayrollAdvanceImportResult, error) {
	period, err := resolvePayrollPeriod(year, month)
	if err != nil {
		s.log.Errorf("payroll import: invalid period: %v", err)
		return nil, domain.BadRequestError
	}

	var candidates []payrollImportCandidate
	err = s.db.WithContext(ctx).Raw(`
		SELECT p.id, COALESCE(st.name, '') AS store_name, p.store_name AS snap_store_name,
		       p.full_name, p.first_name, p.last_name,
		       p.advance_card_amount, p.advance_cash_amount
		FROM employee_payrolls p
		LEFT JOIN stores st ON st.id = p.store_id
		WHERE p.year = ?
		  AND p.month = ?
		  AND (CAST(? AS uuid) IS NULL OR p.company_id = CAST(? AS uuid))
		  AND (CAST(? AS uuid) IS NULL OR p.store_id   = CAST(? AS uuid))`,
		period.Year, period.Month,
		nullIfEmpty(companyId), nullIfEmpty(companyId),
		nullIfEmpty(storeId), nullIfEmpty(storeId),
	).Scan(&candidates).Error
	if err != nil {
		s.log.Errorf("payroll import: could not load payrolls: %v", err)
		return nil, domain.InternalServerError
	}

	// filial|ism -> mos payroll qatorlari. Filial payroll'dagi nusxa (export
	// Excel'idagi nom) va stores.name bo'yicha, ism full_name va first_name +
	// last_name bo'yicha — bitta qator bir nechta kalitga tushadi, takror qo'shilmaydi.
	index := map[string][]*payrollImportCandidate{}
	for i := range candidates {
		c := &candidates[i]
		seen := map[string]bool{}
		for _, store := range []string{c.SnapStoreName, c.StoreName} {
			for _, name := range []string{c.FullName, c.FirstName + " " + c.LastName} {
				key := payrollImportKey(store) + "|" + payrollImportKey(name)
				if payrollImportKey(store) == "" || payrollImportKey(name) == "" || seen[key] {
					continue
				}
				seen[key] = true
				index[key] = append(index[key], c)
			}
		}
	}

	res := &domain.PayrollAdvanceImportResult{
		Period: period,
		DryRun: dryRun,
		Total:  len(rows),
		Counts: map[string]int{},
		Rows:   make([]domain.PayrollAdvanceImportRowResult, 0, len(rows)),
	}
	used := map[string]int{} // payroll id -> fayldagi qator

	for _, row := range rows {
		r := domain.PayrollAdvanceImportRowResult{
			Row:               row.Row,
			StoreName:         row.StoreName,
			FullName:          row.FullName,
			AdvanceCardAmount: row.Card,
			AdvanceCashAmount: row.Cash,
		}

		var match []*payrollImportCandidate
		switch {
		case row.Invalid != "":
			r.Status, r.Reason = domain.PayrollImportInvalid, row.Invalid
		default:
			match = index[payrollImportKey(row.StoreName)+"|"+payrollImportKey(row.FullName)]
			switch len(match) {
			case 0:
				r.Status, r.Reason = domain.PayrollImportNotFound, "shu oy, filial va ism bo'yicha payroll topilmadi"
			case 1:
			default:
				r.Status, r.Reason = domain.PayrollImportAmbiguous,
					fmt.Sprintf("shu filialda bu ism bilan %d ta payroll bor", len(match))
			}
		}

		if r.Status == "" {
			p := match[0]
			r.PayrollId = p.Id
			switch {
			case used[p.Id] != 0:
				r.Status, r.Reason = domain.PayrollImportDuplicate,
					"bu xodim faylning "+strconv.Itoa(used[p.Id])+"-qatorida ham bor"
			case !overwrite && p.AdvanceCardAmount+p.AdvanceCashAmount > 0:
				r.Status, r.Reason = domain.PayrollImportSkipped,
					fmt.Sprintf("avans allaqachon bor (karta %.2f, naqd %.2f)", p.AdvanceCardAmount, p.AdvanceCashAmount)
			case dryRun:
				r.Status = domain.PayrollImportWouldUpdate
			default:
				_, err := s.UpdateEmployeePayrollAdvance(ctx, p.Id, updatedBy, &domain.EmployeePayrollAdvanceRequest{
					AdvanceCardAmount: row.Card,
					AdvanceCashAmount: row.Cash,
				})
				if err != nil {
					r.Status, r.Reason = domain.PayrollImportError, payrollImportErrorText(err)
				} else {
					r.Status = domain.PayrollImportUpdated
				}
			}
			used[p.Id] = row.Row
		}

		res.Counts[r.Status]++
		res.Rows = append(res.Rows, r)
	}

	return res, nil
}

func payrollImportErrorText(err error) string {
	var e *domain.Error
	if errors.As(err, &e) {
		return e.Error()
	}
	return "could not update payroll"
}
