package postgres

import (
	"fmt"
	"strings"
)

// QueryBuilder helps construct parameterized SQL queries with dynamic WHERE
// conditions. It tracks the positional argument index ($1, $2, ...) automatically.
type QueryBuilder struct {
	conditions []string
	args       []any
	argIdx     int
}

// NewQueryBuilder returns a ready-to-use QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// AddCondition appends a WHERE condition with a parameterized value.
// Example: qb.AddCondition("ticker", "=", "AAPL") produces "ticker = $1".
func (qb *QueryBuilder) AddCondition(column, op string, value any) {
	qb.argIdx++
	qb.args = append(qb.args, value)
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s %s $%d", column, op, qb.argIdx))
}

// AddRawCondition appends a condition that does not use a parameter
// (e.g., "closed_at IS NULL").
func (qb *QueryBuilder) AddRawCondition(condition string) {
	qb.conditions = append(qb.conditions, condition)
}

// NextParam adds a value to args and returns its positional placeholder (e.g., "$3").
func (qb *QueryBuilder) NextParam(value any) string {
	qb.argIdx++
	qb.args = append(qb.args, value)
	return fmt.Sprintf("$%d", qb.argIdx)
}

// WhereClause returns the WHERE clause string. Returns empty string when no
// conditions were added.
func (qb *QueryBuilder) WhereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(qb.conditions, " AND ")
}

// Pagination appends " LIMIT $N OFFSET $M" to the query.
func (qb *QueryBuilder) Pagination(query string, limit, offset int) string {
	return query + fmt.Sprintf(" LIMIT %s OFFSET %s", qb.NextParam(limit), qb.NextParam(offset))
}

// Args returns the accumulated query arguments.
func (qb *QueryBuilder) Args() []any {
	return qb.args
}
