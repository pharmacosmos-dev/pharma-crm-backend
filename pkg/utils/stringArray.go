package utils

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray represents a one-dimensional array of the PostgreSQL character types.
type StringArray []string

// Scan implements the sql.Scanner interface.
func (a *StringArray) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return a.scanBytes(src)
	case string:
		return a.scanBytes([]byte(src))
	case nil:
		*a = nil
		return nil
	}

	return fmt.Errorf("pq: cannot convert %T to StringArray", src)
}

func (a *StringArray) scanBytes(src []byte) error {
	elems, err := scanLinearArray(src, []byte{','}, "StringArray")
	if err != nil {
		return err
	}
	if *a != nil && len(elems) == 0 {
		*a = (*a)[:0]
	} else {
		b := make(StringArray, len(elems))
		for i, v := range elems {
			if b[i] = string(v); v == nil {
				return fmt.Errorf("pq: parsing array element index %d: cannot convert nil to string", i)
			}
		}
		*a = b
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	if n := len(a); n > 0 {
		// There will be at least two curly brackets, 2*N bytes of quotes,
		// and N-1 bytes of delimiters.
		b := make([]byte, 1, 1+3*n)
		b[0] = '{'

		b = appendArrayQuotedBytes(b, []byte(a[0]))
		for i := 1; i < n; i++ {
			b = append(b, ',')
			b = appendArrayQuotedBytes(b, []byte(a[i]))
		}

		return string(append(b, '}')), nil
	}

	return "{}", nil
}

func scanLinearArray(src, del []byte, typ string) (elems [][]byte, err error) {
	dims, elems, err := parseArray(src, del)
	if err != nil {
		return nil, err
	}
	if len(dims) > 1 {
		return nil, fmt.Errorf("pq: cannot convert ARRAY%s to %s", strings.Replace(fmt.Sprint(dims), " ", "][", -1), typ)
	}
	return elems, err
}

// parseArray extracts the dimensions and elements of an array represented in
// text format. Only representations emitted by the backend are supported.
// Notably, whitespace around brackets and delimiters is significant, and NULL
// is case-sensitive.
//
// See http://www.postgresql.org/docs/current/static/arrays.html#ARRAYS-IO
func parseArray(src, del []byte) (dims []int, elems [][]byte, err error) {
	var depth, i int

	if len(src) < 1 || src[0] != '{' {
		return nil, nil, fmt.Errorf("pq: unable to parse array; expected %q at offset %d", '{', 0)
	}

Open:
	for i < len(src) {
		switch src[i] {
		case '{':
			depth++
			i++
		case '}':
			elems = make([][]byte, 0)
			goto Close
		default:
			break Open
		}
	}
	dims = make([]int, i)

Element:
	for i < len(src) {
		switch src[i] {
		case '{':
			if depth == len(dims) {
				break Element
			}
			depth++
			dims[depth-1] = 0
			i++
		case '"':
			var elem = []byte{}
			var escape bool
			for i++; i < len(src); i++ {
				if escape {
					elem = append(elem, src[i])
					escape = false
				} else {
					switch src[i] {
					default:
						elem = append(elem, src[i])
					case '\\':
						escape = true
					case '"':
						elems = append(elems, elem)
						i++
						break Element
					}
				}
			}
		default:
			for start := i; i < len(src); i++ {
				if bytes.HasPrefix(src[i:], del) || src[i] == '}' {
					elem := src[start:i]
					if len(elem) == 0 {
						return nil, nil, fmt.Errorf("pq: unable to parse array; unexpected %q at offset %d", src[i], i)
					}
					if bytes.Equal(elem, []byte("NULL")) {
						elem = nil
					}
					elems = append(elems, elem)
					break Element
				}
			}
		}
	}

	for i < len(src) {
		if bytes.HasPrefix(src[i:], del) && depth > 0 {
			dims[depth-1]++
			i += len(del)
			goto Element
		} else if src[i] == '}' && depth > 0 {
			dims[depth-1]++
			depth--
			i++
		} else {
			return nil, nil, fmt.Errorf("pq: unable to parse array; unexpected %q at offset %d", src[i], i)
		}
	}

Close:
	for i < len(src) {
		if src[i] == '}' && depth > 0 {
			depth--
			i++
		} else {
			return nil, nil, fmt.Errorf("pq: unable to parse array; unexpected %q at offset %d", src[i], i)
		}
	}
	if depth > 0 {
		err = fmt.Errorf("pq: unable to parse array; expected %q at offset %d", '}', i)
	}
	if err == nil {
		for _, d := range dims {
			if (len(elems) % d) != 0 {
				err = fmt.Errorf("pq: multidimensional arrays must have elements with matching dimensions")
			}
		}
	}
	return
}

func appendArrayQuotedBytes(b, v []byte) []byte {
	b = append(b, '"')
	for {
		i := bytes.IndexAny(v, `"\`)
		if i < 0 {
			b = append(b, v...)
			break
		}
		if i > 0 {
			b = append(b, v[:i]...)
		}
		b = append(b, '\\', v[i])
		v = v[i+1:]
	}
	return append(b, '"')
}

func BuildProductReport(orderField string) string {
	allowedFields := map[string]string{
		"cart_item_id":     "cart_item_id",
		"material_code":    "p.material_code",
		"store_name":       "store_name",
		"product_name":     "product_name",
		"producer_name":    "pr.name",
		"serial_number":    "sp.serial_number",
		"expire_date":      "sp.expire_date",
		"quantity":         "quantity",
		"supply_price":     "sp.supply_price",
		"retail_price":     "sp.retail_price",
		"supply_price_sum": "supply_price_sum",
		"retail_price_sum": "retail_price_sum",
		"markup_sum":       "markup_sum",
		"vat_sum":          "vat_sum",
		"completed_at":     "sl.completed_at",
		"full_name":        "e.full_name",
		"sale_number":      "sl.sale_number",
		"sale_type":        "sl.sale_type",
		"marking_count":    "sl.marking_count",
	}

	if orderField == "" {
		return " ORDER BY sl.completed_at DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	return " ORDER BY sl.completed_at DESC "
}

func BuildStoreReportOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"store_code":    "s.store_code",
		"store_name":    "store_name",
		"cash":          "cash",
		"uzcard":        "uzcard",
		"humo":          "humo",
		"click":         "click",
		"return_amount": "return_amount",
		"total_amount":  "total_amount",
	}

	if orderField == "" {
		return " ORDER BY store_name "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(field, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(field, "-")
	} else if strings.HasPrefix(field, "+") {
		field = strings.TrimPrefix(field, "+")
	}

	if col, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", col, direction)
	}

	// fallback default
	return " ORDER BY store_name, start_date "
}

func BuildTopProductOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"name":                  "curr.name",
		"count":                 "curr.count",
		"unit_quantity":         "curr.unit_quantity",
		"total_amount":          "curr.total_amount",
		"previous_total_amount": "prev.total_amount",
		"percent":               "percent",
	}

	if orderField == "" {
		return " ORDER BY curr.total_amount DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	return " ORDER BY curr.total_amount DESC "
}

func BuildBonusProductOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"name":                  "curr.name",
		"count":                 "curr.count",
		"unit_quantity":         "curr.unit_quantity",
		"bonus_amount":          "curr.bonus_amount",
		"previous_bonus_amount": "prev.bonus_amount",
		"percent":               "percent",
	}

	if orderField == "" {
		return " ORDER BY curr.bonus_amount DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	return " ORDER BY curr.bonus_amount DESC "
}

func BuildTopStoreOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"name":                  "stores.name",
		"count":                 "curr.count",
		"total_amount":          "curr.total_amount",
		"previous_total_amount": "prev.total_amount",
		"percent":               "percent",
	}

	if orderField == "" {
		return " ORDER BY curr.total_amount DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	return " ORDER BY curr.total_amount DESC "
}

func BuildBonusReportOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"full_name": "e.full_name",
		"amount":    "amount",
		"count":     "count",
	}

	if orderField == "" {
		return " ORDER BY e.full_name "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s", dbColumn, direction)
	}

	return " ORDER BY e.full_name"
}

func BuildTopSellerOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"full_name":             "curr.full_name",
		"store_name":            "curr.store_name",
		"count":                 "curr.count",
		"total_amount":          "curr.total_amount",
		"previous_total_amount": "prev.total_amount",
		"percent":               "percent",
	}

	if orderField == "" {
		return " ORDER BY curr.total_amount DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s", dbColumn, direction)
	}

	return " ORDER BY curr.total_amount DESC "
}

func BuildStoreSummaryOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"name":          "st.name",
		"sale_amount":   "sale_amount",
		"import_amount": "import_amount",
		"stock_amount":  "stock_amount",
		"total":         "total",
	}

	if orderField == "" {
		return " ORDER BY total DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	// Default fallback
	return " ORDER BY total DESC "
}

func BuildProductOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"product_id":            "b.product_id",
		"name":                  "b.name",
		"final_pack_quantity":   "final_pack_quantity",
		"final_unit_quantity":   "final_unit_quantity",
		"pack_qty":              "b.pack_qty",
		"unit_qty":              "b.unit_qty",
		"import_pack_change":    "import_pack_change",
		"import_unit_change":    "import_unit_change",
		"sales_pack_change":     "sales_pack_change",
		"sales_unit_change":     "sales_unit_change",
		"return_pack_change":    "return_pack_change",
		"return_unit_change":    "return_unit_change",
		"transfer_pack_change":  "transfer_pack_change",
		"transfer_unit_change":  "transfer_unit_change",
		"inventory_pack_change": "inventory_pack_change",
		"inventory_unit_change": "inventory_unit_change",
	}

	if orderField == "" {
		return " ORDER BY b.product_id"
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s ", dbColumn, direction)
	}

	// Default fallback
	return " ORDER BY b.product_id"
}

func BuildDiscountCardOrderClause(orderField string) string {
	allowedFields := map[string]string{
		"check_count":            "check_count",
		"total_without_discount": "total_without_discount",
		"total_discount":         "total_discount",
		"total_with_discount":    "total_with_discount",
		"percent":                "percent",
		"customer_name":          "customer_name",
		"store_name":             "store_name",
	}

	if orderField == "" {
		return " ORDER BY check_count DESC "
	}

	direction := "ASC"
	field := orderField

	if strings.HasPrefix(orderField, "-") {
		direction = "DESC"
		field = strings.TrimPrefix(orderField, "-")
	} else if strings.HasPrefix(orderField, "+") {
		field = strings.TrimPrefix(orderField, "+")
	}

	if dbColumn, ok := allowedFields[field]; ok {
		return fmt.Sprintf(" ORDER BY %s %s", dbColumn, direction)
	}

	return " ORDER BY check_count DESC "
}
