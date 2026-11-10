package database

import (
	"fmt"
	"strings"
)

// ActionSummary is one row of an aggregated view over actions. Income and
// expense are always reported separately so totals never mix the two.
type ActionSummary struct {
	Group string  `json:"group,omitempty"`
	Type  string  `json:"type"`
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

// Supported groupings for SummarizeActions.
const (
	GroupByNone     = ""
	GroupByCategory = "category"
	GroupByMonth    = "month"
	GroupByUser     = "user"
)

// SummarizeActions sums the actions matching filters, split by type and
// optionally grouped by category, month (YYYY-MM) or user. Limit and Offset
// in filters are ignored.
func (db *DB) SummarizeActions(filters ActionFilters, groupBy string) ([]ActionSummary, error) {
	var groupExpr string
	switch groupBy {
	case GroupByNone:
		groupExpr = "''"
	case GroupByCategory:
		groupExpr = "COALESCE(c.description, 'Uncategorized')"
	case GroupByMonth:
		groupExpr = "strftime('%Y-%m', a.date)"
	case GroupByUser:
		groupExpr = "COALESCE(u.username, '')"
	default:
		return nil, fmt.Errorf("failed to summarize actions: unsupported grouping %q", groupBy)
	}

	query := `
		SELECT ` + groupExpr + ` AS grp, a.type, SUM(a.amount), COUNT(*)
		FROM actions a
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN users u ON a.user_id = u.id`

	var conditions []string
	var args []interface{}

	if filters.Username != "" {
		conditions = append(conditions, "u.username = ?")
		args = append(args, filters.Username)
	}
	if filters.Type != "" {
		conditions = append(conditions, "a.type = ?")
		args = append(args, filters.Type)
	}
	if filters.DateFrom != "" {
		conditions = append(conditions, "a.date >= ?")
		args = append(args, filters.DateFrom)
	}
	if filters.DateTo != "" {
		conditions = append(conditions, "a.date <= ?")
		args = append(args, filters.DateTo)
	}
	if filters.Search != "" {
		// Same Unicode-aware matching as ListActions, for Greek text.
		conditions = append(conditions, "lower_unicode(a.description) LIKE lower_unicode(?)")
		args = append(args, "%"+filters.Search+"%")
	}
	if filters.CategoryID != nil {
		conditions = append(conditions, "a.category_id = ?")
		args = append(args, *filters.CategoryID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY grp, a.type"
	if groupBy == GroupByMonth {
		query += " ORDER BY grp ASC, a.type ASC"
	} else {
		query += " ORDER BY a.type ASC, SUM(a.amount) DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize actions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	summaries := []ActionSummary{}
	for rows.Next() {
		var s ActionSummary
		if err := rows.Scan(&s.Group, &s.Type, &s.Total, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan action summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read action summaries: %w", err)
	}

	return summaries, nil
}
