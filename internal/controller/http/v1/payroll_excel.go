package v1

import (
	"fmt"

	"github.com/pharma-crm-backend/domain"
	"github.com/xuri/excelize/v2"
)

// Oylik hisobotining Excel ko'rinishi.
//
// Varaq tuzilishi:
//
//	1-qator — birlashtirilgan guruh sarlavhalari (Филиал, Оклад, KPI, ...)
//	2-qator — ustun sarlavhalari
//	keyin   — har bir do'kon uchun blok: do'kon yig'indisi qatori + xodimlar
//	oxirida — barcha do'konlar bo'yicha umumiy jami
//
// Ustunlar ro'yxat sifatida e'lon qilinadi: sarlavhalar, kengliklar, ranglar va
// raqam formati bir joyda turadi, shuning uchun ustun qo'shish yoki tartibini
// o'zgartirish uchun faqat shu ro'yxatni tahrirlash yetadi.

// payrollExcelGroup — birlashtirilgan yuqori sarlavha va uning rangi.
type payrollExcelGroup struct {
	title      string
	headerFill string // guruh sarlavhasi (to'q)
	cellFill   string // ustun sarlavhasi va yig'indi qatorlari (och)
}

var (
	groupBranch    = payrollExcelGroup{"Филиал", "B8CCE4", "DCE6F1"}
	groupSalary    = payrollExcelGroup{"Оклад", "D8E4BC", "EBF1DE"}
	groupKpi       = payrollExcelGroup{"KPI (План)", "CCC0DA", "E4DFEC"}
	groupBonus     = payrollExcelGroup{"Бонус", "E6B8B7", "F2DCDB"}
	groupGross     = payrollExcelGroup{"Чистый Оклад", "B8CCE4", "DCE6F1"}
	groupAdvance   = payrollExcelGroup{"Аванс", "FCD5B4", "FDE9D9"}
	groupDeduction = payrollExcelGroup{"Удержания", "D99694", "F2DCDB"}
	groupNetPay    = payrollExcelGroup{"Оклад на руки", "C4D79B", "EBF1DE"}
)

const (
	fmtMoney = "#,##0.00"
	fmtInt   = "#,##0"
	fmtHours = "#,##0.##"
)

// payrollExcelColumn — bitta ustun: sarlavhasi, qaysi guruhga tegishli va
// xodim/yig'indi qatorlarida qanday qiymat chiqishi.
//
// value nil qaytarsa katak bo'sh qoladi — masalan "План" faqat do'kon
// qatorida to'ladi, chunki u do'kon darajasidagi qiymat va uni har bir xodim
// yonida takrorlash chalg'ituvchi bo'lardi.
type payrollExcelColumn struct {
	group  payrollExcelGroup
	title  string
	width  float64
	numFmt string
	// employee — xodim qatoridagi qiymat (idx — do'kon ichidagi tartib raqami)
	employee func(idx int, r domain.EmployeePayrollRow) any
	// total — yig'indi qatoridagi qiymat (do'kon va umumiy jami uchun)
	total func(t *payrollExcelTotals) any
}

// payrollExcelTotals — do'kon yoki butun hisobot bo'yicha yig'indi.
//
// StorePlan/StoreSales SUM emas: ular do'kon darajasidagi qiymatlar bo'lib har
// bir xodim qatorida takrorlanadi, shuning uchun do'kon ichida bittasidan
// olinadi va umumiy jamiga do'kon bo'yicha bir marta qo'shiladi.
type payrollExcelTotals struct {
	StoreName string

	AvgMonthlyHours  float64
	WorkedHours      float64
	SalaryRate       float64
	ActualSalary     float64
	IndividualSales  float64
	StorePlan        float64
	StoreSales       float64
	KpiAmount        float64
	Bonus            float64
	Gross            float64
	AdvanceCard      float64
	AdvanceCash      float64
	DeductionTerm    float64
	DeductionRecount float64
	DeductionFine    float64
	NetPay           float64
}

// addEmployee — xodim qatorini yig'indiga qo'shadi.
func (t *payrollExcelTotals) addEmployee(r domain.EmployeePayrollRow) {
	t.AvgMonthlyHours += r.AvgMonthlyHours
	t.WorkedHours += r.WorkedHours
	t.SalaryRate += r.SalaryRateAmount
	t.ActualSalary += r.ActualSalaryAmount
	t.IndividualSales += r.IndividualSalesAmount
	t.KpiAmount += r.KpiAmount
	t.Bonus += r.BonusAmount
	t.Gross += r.GrossSalaryAmount
	t.AdvanceCard += r.AdvanceCardAmount
	t.AdvanceCash += r.AdvanceCashAmount
	t.DeductionTerm += r.DeductionTermAmount
	t.DeductionRecount += r.DeductionRecountAmount
	t.DeductionFine += r.DeductionFineAmount
	t.NetPay += r.NetPayAmount
}

// addStore — do'kon yig'indisini umumiy jamiga qo'shadi.
func (t *payrollExcelTotals) addStore(s *payrollExcelTotals) {
	t.AvgMonthlyHours += s.AvgMonthlyHours
	t.WorkedHours += s.WorkedHours
	t.SalaryRate += s.SalaryRate
	t.ActualSalary += s.ActualSalary
	t.IndividualSales += s.IndividualSales
	// Do'kon darajasidagi qiymatlar: har do'kondan bir marta
	t.StorePlan += s.StorePlan
	t.StoreSales += s.StoreSales
	t.KpiAmount += s.KpiAmount
	t.Bonus += s.Bonus
	t.Gross += s.Gross
	t.AdvanceCard += s.AdvanceCard
	t.AdvanceCash += s.AdvanceCash
	t.DeductionTerm += s.DeductionTerm
	t.DeductionRecount += s.DeductionRecount
	t.DeductionFine += s.DeductionFine
	t.NetPay += s.NetPay
}

// payrollExcelColumns — varaqning to'liq tuzilishi.
var payrollExcelColumns = []payrollExcelColumn{
	{groupBranch, "№", 5, fmtInt,
		func(i int, _ domain.EmployeePayrollRow) any { return i },
		func(t *payrollExcelTotals) any { return nil }},
	{groupBranch, "Сотрудник", 38, "",
		func(_ int, r domain.EmployeePayrollRow) any { return r.FullName },
		func(t *payrollExcelTotals) any { return t.StoreName }},
	{groupBranch, "Должность", 18, "",
		func(_ int, r domain.EmployeePayrollRow) any { return payrollPosition(r) },
		func(t *payrollExcelTotals) any { return nil }},
	{groupBranch, "Стаж", 8, fmtHours,
		func(_ int, r domain.EmployeePayrollRow) any { return r.ExperienceYears },
		func(t *payrollExcelTotals) any { return nil }},

	{groupSalary, "Норма Час", 12, fmtHours,
		func(_ int, r domain.EmployeePayrollRow) any { return r.AvgMonthlyHours },
		func(t *payrollExcelTotals) any { return t.AvgMonthlyHours }},
	{groupSalary, "Час", 10, fmtHours,
		func(_ int, r domain.EmployeePayrollRow) any { return r.WorkedHours },
		func(t *payrollExcelTotals) any { return t.WorkedHours }},
	{groupSalary, "Оклад", 14, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.SalaryRateAmount },
		func(t *payrollExcelTotals) any { return t.SalaryRate }},
	{groupSalary, "Факт Оклад", 14, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.ActualSalaryAmount },
		func(t *payrollExcelTotals) any { return t.ActualSalary }},

	{groupKpi, "Окошка", 16, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.IndividualSalesAmount },
		func(t *payrollExcelTotals) any { return t.IndividualSales }},
	// Reja do'kon darajasida: xodim qatorida bo'sh qoladi
	{groupKpi, "План", 16, fmtInt,
		func(_ int, _ domain.EmployeePayrollRow) any { return nil },
		func(t *payrollExcelTotals) any { return t.StorePlan }},
	{groupKpi, "Процент KPI", 12, fmtHours,
		func(_ int, r domain.EmployeePayrollRow) any { return r.KpiPercent },
		func(t *payrollExcelTotals) any { return nil }},
	{groupKpi, "KPI", 16, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.KpiAmount },
		func(t *payrollExcelTotals) any { return t.KpiAmount }},

	{groupBonus, "Бонус", 12, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.BonusAmount },
		func(t *payrollExcelTotals) any { return t.Bonus }},

	{groupGross, "Чистый Оклад", 16, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.GrossSalaryAmount },
		func(t *payrollExcelTotals) any { return t.Gross }},

	{groupAdvance, "Пластик", 14, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.AdvanceCardAmount },
		func(t *payrollExcelTotals) any { return t.AdvanceCard }},
	{groupAdvance, "Аванс", 14, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.AdvanceCashAmount },
		func(t *payrollExcelTotals) any { return t.AdvanceCash }},

	{groupDeduction, "Срок", 12, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.DeductionTermAmount },
		func(t *payrollExcelTotals) any { return t.DeductionTerm }},
	{groupDeduction, "Переучут", 12, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.DeductionRecountAmount },
		func(t *payrollExcelTotals) any { return t.DeductionRecount }},
	{groupDeduction, "Штраф", 12, fmtInt,
		func(_ int, r domain.EmployeePayrollRow) any { return r.DeductionFineAmount },
		func(t *payrollExcelTotals) any { return t.DeductionFine }},

	{groupNetPay, "Оклад на руки", 18, fmtMoney,
		func(_ int, r domain.EmployeePayrollRow) any { return r.NetPayAmount },
		func(t *payrollExcelTotals) any { return t.NetPay }},
}

// payrollPosition — "Должность" ustuni. roles jadvalidagi nom ustun turadi,
// u bo'sh bo'lsa tizim roli (role_type) ko'rsatiladi.
func payrollPosition(r domain.EmployeePayrollRow) string {
	if r.Role != "" {
		return r.Role
	}
	return r.RoleType
}

// buildPayrollExcel — hisobot qatorlaridan Excel faylini yasaydi.
//
// Qatorlar do'kon nomi bo'yicha tartiblangan holda keladi (so'rovdagi ORDER BY),
// shuning uchun do'konlar ketma-ket kelishiga tayanib guruhlanadi — qo'shimcha
// saralash yoki map kerak emas va do'konlar tartibi saqlanadi.
func buildPayrollExcel(sheet string, rows []domain.EmployeePayrollRow) (*excelize.File, error) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheet)

	styles, err := newPayrollExcelStyles(f)
	if err != nil {
		return nil, err
	}

	if err := writePayrollExcelHeader(f, sheet, styles); err != nil {
		return nil, err
	}

	rowNum := 3 // 1-2 qatorlar sarlavha
	grand := &payrollExcelTotals{StoreName: "АПТЕКА ИТОГ"}

	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && storeKeyOf(rows[end]) == storeKeyOf(rows[start]) {
			end++
		}
		storeRows := rows[start:end]

		store := &payrollExcelTotals{StoreName: storeRows[0].StoreName}
		for _, r := range storeRows {
			store.addEmployee(r)
		}
		// Do'kon darajasidagi qiymatlar takrorlanadi — bittasidan olinadi
		store.StorePlan = storeRows[0].StorePlanAmount
		store.StoreSales = storeRows[0].StoreSalesAmount

		// Do'kon sarlavhasi + yig'indisi xodimlardan OLDIN turadi
		writePayrollExcelTotalRow(f, sheet, rowNum, store, styles, styles.storeTotal)
		rowNum++

		for i, r := range storeRows {
			writePayrollExcelEmployeeRow(f, sheet, rowNum, i+1, r, styles)
			rowNum++
		}

		grand.addStore(store)
		start = end
	}

	writePayrollExcelTotalRow(f, sheet, rowNum, grand, styles, styles.grandTotal)

	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, Split: false, XSplit: 4, YSplit: 2,
		TopLeftCell: "E3", ActivePane: "bottomRight",
	}); err != nil {
		return nil, err
	}

	return f, nil
}

// storeKeyOf — guruhlash kaliti. store_id bo'lmasa nomga tushiladi, aks holda
// do'konga biriktirilmagan xodimlar bitta guruhga qo'shilib ketardi.
func storeKeyOf(r domain.EmployeePayrollRow) string {
	if r.StoreId != nil {
		return *r.StoreId
	}
	return "name:" + r.StoreName
}

// payrollExcelStyles — varaqda ishlatiladigan tayyor stillar.
type payrollExcelStyles struct {
	groupHeader map[string]int // guruh sarlavhasi rangi bo'yicha
	colHeader   map[string]int
	employee    map[string]int // ustun raqam formati bo'yicha
	storeTotal  map[string]int
	grandTotal  map[string]int
}

func newPayrollExcelStyles(f *excelize.File) (*payrollExcelStyles, error) {
	s := &payrollExcelStyles{
		groupHeader: map[string]int{},
		colHeader:   map[string]int{},
		employee:    map[string]int{},
		storeTotal:  map[string]int{},
		grandTotal:  map[string]int{},
	}

	border := []excelize.Border{
		{Type: "left", Color: "9E9E9E", Style: 1},
		{Type: "right", Color: "9E9E9E", Style: 1},
		{Type: "top", Color: "9E9E9E", Style: 1},
		{Type: "bottom", Color: "9E9E9E", Style: 1},
	}
	fill := func(color string) excelize.Fill {
		return excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1}
	}

	for _, col := range payrollExcelColumns {
		g := col.group

		if _, ok := s.groupHeader[g.headerFill]; !ok {
			id, err := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Bold: true, Size: 11},
				Fill:      fill(g.headerFill),
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
				Border:    border,
			})
			if err != nil {
				return nil, err
			}
			s.groupHeader[g.headerFill] = id
		}

		if _, ok := s.colHeader[g.cellFill]; !ok {
			id, err := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Bold: true, Size: 10},
				Fill:      fill(g.cellFill),
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
				Border:    border,
			})
			if err != nil {
				return nil, err
			}
			s.colHeader[g.cellFill] = id
		}

		// Xodim qatorlari oq fonda, faqat raqam formati bilan farqlanadi
		if _, ok := s.employee[col.numFmt]; !ok {
			style := &excelize.Style{Border: border}
			if col.numFmt != "" {
				numFmt := col.numFmt
				style.CustomNumFmt = &numFmt
			}
			id, err := f.NewStyle(style)
			if err != nil {
				return nil, err
			}
			s.employee[col.numFmt] = id
		}

		// Yig'indi qatorlari guruh rangida va qalin
		key := g.cellFill + "|" + col.numFmt
		if _, ok := s.storeTotal[key]; !ok {
			style := &excelize.Style{
				Font:   &excelize.Font{Bold: true, Size: 10},
				Fill:   fill(g.cellFill),
				Border: border,
			}
			if col.numFmt != "" {
				numFmt := col.numFmt
				style.CustomNumFmt = &numFmt
			}
			id, err := f.NewStyle(style)
			if err != nil {
				return nil, err
			}
			s.storeTotal[key] = id
		}

		grandKey := g.headerFill + "|" + col.numFmt
		if _, ok := s.grandTotal[grandKey]; !ok {
			style := &excelize.Style{
				Font:   &excelize.Font{Bold: true, Size: 12},
				Fill:   fill(g.headerFill),
				Border: border,
			}
			if col.numFmt != "" {
				numFmt := col.numFmt
				style.CustomNumFmt = &numFmt
			}
			id, err := f.NewStyle(style)
			if err != nil {
				return nil, err
			}
			s.grandTotal[grandKey] = id
		}
	}

	return s, nil
}

func writePayrollExcelHeader(f *excelize.File, sheet string, s *payrollExcelStyles) error {
	f.SetRowHeight(sheet, 1, 28)
	f.SetRowHeight(sheet, 2, 30)

	// Guruh sarlavhalari: bir xil guruhga tegishli ketma-ket ustunlar birlashtiriladi
	for start := 0; start < len(payrollExcelColumns); {
		end := start
		for end < len(payrollExcelColumns) &&
			payrollExcelColumns[end].group.title == payrollExcelColumns[start].group.title {
			end++
		}
		g := payrollExcelColumns[start].group

		from, err := excelize.CoordinatesToCellName(start+1, 1)
		if err != nil {
			return err
		}
		to, err := excelize.CoordinatesToCellName(end, 1)
		if err != nil {
			return err
		}
		if err := f.MergeCell(sheet, from, to); err != nil {
			return err
		}
		f.SetCellValue(sheet, from, g.title)
		if err := f.SetCellStyle(sheet, from, to, s.groupHeader[g.headerFill]); err != nil {
			return err
		}
		start = end
	}

	// Ustun sarlavhalari va kengliklari
	for i, col := range payrollExcelColumns {
		cell, err := excelize.CoordinatesToCellName(i+1, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, cell, col.title)
		if err := f.SetCellStyle(sheet, cell, cell, s.colHeader[col.group.cellFill]); err != nil {
			return err
		}

		name, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, name, name, col.width); err != nil {
			return err
		}
	}

	return nil
}

func writePayrollExcelEmployeeRow(
	f *excelize.File, sheet string, rowNum, idx int,
	r domain.EmployeePayrollRow, s *payrollExcelStyles,
) {
	for i, col := range payrollExcelColumns {
		cell, err := excelize.CoordinatesToCellName(i+1, rowNum)
		if err != nil {
			continue
		}
		if v := col.employee(idx, r); v != nil {
			f.SetCellValue(sheet, cell, v)
		}
		f.SetCellStyle(sheet, cell, cell, s.employee[col.numFmt])
	}
}

func writePayrollExcelTotalRow(
	f *excelize.File, sheet string, rowNum int,
	t *payrollExcelTotals, s *payrollExcelStyles, styleSet map[string]int,
) {
	for i, col := range payrollExcelColumns {
		cell, err := excelize.CoordinatesToCellName(i+1, rowNum)
		if err != nil {
			continue
		}
		if v := col.total(t); v != nil {
			f.SetCellValue(sheet, cell, v)
		}
		key := col.group.cellFill + "|" + col.numFmt
		if _, ok := styleSet[key]; !ok {
			key = col.group.headerFill + "|" + col.numFmt
		}
		f.SetCellStyle(sheet, cell, cell, styleSet[key])
	}
}

// payrollExcelFileName — fayl nomi prefiksi.
func payrollExcelFileName(year, month int) string {
	return fmt.Sprintf("Oylik_hisobot_%d-%02d", year, month)
}
