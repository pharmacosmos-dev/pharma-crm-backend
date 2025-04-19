package builder

import (
	"errors"
	"fmt"
	"strings"
)

// QueryBuilder holds the components of an SQL query
type QueryBuilder struct {
	columns    []string
	table      string
	conditions []string
	joins      []string
	args       []any
	orderBy    string
	limit      int
	offset     int
	isCount    bool
}

// Join represents a single JOIN clause
type Join struct {
	Type      string // e.g., "INNER", "LEFT", "RIGHT"
	Table     string
	Condition string
}

// NewQueryBuilder initializes a new QueryBuilder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// Select specifies the columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = append(qb.columns, columns...)
	return qb
}

// From specifies the table to query
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.table = table
	return qb
}

// Join adds a JOIN clause with the specified type, table, and condition
func (qb *QueryBuilder) Join(query string) *QueryBuilder {
	qb.joins = append(qb.joins, query)
	return qb
}

// Where adds a condition with parameterized arguments
func (qb *QueryBuilder) Where(condition string, args ...any) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy specifies the ORDER BY clause
func (qb *QueryBuilder) OrderBy(order string) *QueryBuilder {
	qb.orderBy = order
	return qb
}

// Limit specifies the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset specifies the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Count modifies the query to return the total count of rows
func (qb *QueryBuilder) Count() *QueryBuilder {
	qb.isCount = true
	return qb
}

// Build generates the SQL query and arguments
func (qb *QueryBuilder) Build() (string, []any, error) {
	if qb.table == "" {
		return "", nil, errors.New("no table specified")
	}

	// Determine columns (use COUNT(*) if isCount is true)
	columns := qb.columns
	if qb.isCount {
		columns = []string{"COUNT(*)"}
	} else if len(qb.columns) == 0 {
		return "", nil, errors.New("no columns specified")
	}

	// Build SELECT clause
	query := []string{"SELECT", strings.Join(columns, ", "), "FROM", qb.table}

	// Build JOIN clauses
	query = append(query, qb.joins...)

	// Build WHERE clause
	if len(qb.conditions) > 0 {
		whereClause := strings.Join(qb.conditions, " AND ")
		query = append(query, "WHERE", whereClause)
	}

	// Build ORDER BY clause
	if qb.orderBy != "" {
		query = append(query, "ORDER BY", qb.orderBy)
	}

	// Build LIMIT clause
	if qb.limit > 0 {
		query = append(query, "LIMIT", fmt.Sprintf("%d", qb.limit))
	}

	// Build OFFSET clause
	if qb.offset > 0 {
		query = append(query, "OFFSET", fmt.Sprintf("%d", qb.offset))
	}

	// Replace ? placeholders with $1, $2, etc. for PostgreSQL
	sqlQuery := strings.Join(query, " ")
	for i := 0; i < len(qb.args); i++ {
		sqlQuery = strings.Replace(sqlQuery, "?", fmt.Sprintf("$%d", i+1), 1)
	}

	return sqlQuery, qb.args, nil
}
