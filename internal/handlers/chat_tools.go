package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mtzanidakis/budgeting/internal/ai"
	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/database"
	"github.com/mtzanidakis/budgeting/internal/models"
)

const (
	chatSearchDefaultLimit = 20
	chatSearchMaxLimit     = 50
	// chatMaxCreates bounds how many entries one chat request may add.
	chatMaxCreates           = 10
	chatMaxDescriptionLength = 200
)

const chatFilterProperties = `
	"date_from": {"type": "string", "description": "Inclusive start date, YYYY-MM-DD"},
	"date_to": {"type": "string", "description": "Inclusive end date, YYYY-MM-DD"},
	"type": {"type": "string", "enum": ["income", "expense"]},
	"username": {"type": "string", "description": "Only entries of this family member"},
	"category_id": {"type": "integer", "description": "Only entries in this category"},
	"search": {"type": "string", "description": "Case-insensitive substring of the entry description"}`

var chatToolDefinitions = []ai.Tool{
	chatTool("summarize_actions",
		"Sum and count income/expense entries matching the filters. Results are always split by type, and optionally grouped.",
		`{"type": "object", "properties": {`+chatFilterProperties+`,
			"group_by": {"type": "string", "enum": ["category", "month", "user"], "description": "Optional grouping of the totals"}
		}}`),
	chatTool("search_actions",
		"List individual income/expense entries matching the filters, newest first.",
		`{"type": "object", "properties": {`+chatFilterProperties+`,
			"limit": {"type": "integer", "description": "Maximum entries to return (default 20, max 50)"}
		}}`),
	chatTool("create_action",
		"Record a new income or expense entry for the current user.",
		`{"type": "object", "properties": {
			"type": {"type": "string", "enum": ["income", "expense"]},
			"date": {"type": "string", "description": "YYYY-MM-DD"},
			"description": {"type": "string"},
			"amount": {"type": "number", "description": "Positive amount"},
			"category_id": {"type": "integer", "description": "A category of the same type"}
		}, "required": ["type", "date", "description", "amount", "category_id"]}`),
}

func chatTool(name, description, parameters string) ai.Tool {
	return ai.Tool{
		Type: "function",
		Function: ai.FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  json.RawMessage(parameters),
		},
	}
}

// chatTools executes the model's tool calls on behalf of one user.
type chatTools struct {
	db       *database.DB
	session  *auth.Session
	onCreate func(*ActionResponse)
	created  int
}

type chatFilterArgs struct {
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	Type       string `json:"type"`
	Username   string `json:"username"`
	CategoryID *int64 `json:"category_id"`
	Search     string `json:"search"`
	GroupBy    string `json:"group_by"`
	Limit      int    `json:"limit"`
}

type chatCreateArgs struct {
	Type        string  `json:"type"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	CategoryID  *int64  `json:"category_id"`
}

type chatActionResult struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Category    string  `json:"category,omitempty"`
	Username    string  `json:"username,omitempty"`
}

// execute runs a tool call and returns its JSON result. Errors are returned
// to the model as {"error": ...} so it can fix its arguments or tell the user.
func (t *chatTools) execute(_ context.Context, name, arguments string) string {
	var result any
	var err error

	switch name {
	case "summarize_actions":
		result, err = t.summarize(arguments)
	case "search_actions":
		result, err = t.search(arguments)
	case "create_action":
		result, err = t.create(arguments)
	default:
		err = fmt.Errorf("unknown tool %q", name)
	}

	if err != nil {
		result = map[string]string{"error": err.Error()}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return `{"error": "failed to encode result"}`
	}
	return string(encoded)
}

func (t *chatTools) parseFilters(arguments string) (database.ActionFilters, chatFilterArgs, error) {
	var args chatFilterArgs
	if err := decodeToolArgs(arguments, &args); err != nil {
		return database.ActionFilters{}, args, err
	}
	for _, date := range []string{args.DateFrom, args.DateTo} {
		if date == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return database.ActionFilters{}, args, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", date)
		}
	}
	if args.Type != "" && args.Type != "income" && args.Type != "expense" {
		return database.ActionFilters{}, args, fmt.Errorf("type must be 'income' or 'expense'")
	}
	return database.ActionFilters{
		Username:   args.Username,
		Type:       args.Type,
		DateFrom:   args.DateFrom,
		DateTo:     args.DateTo,
		Search:     args.Search,
		CategoryID: args.CategoryID,
	}, args, nil
}

func (t *chatTools) summarize(arguments string) (any, error) {
	filters, args, err := t.parseFilters(arguments)
	if err != nil {
		return nil, err
	}
	switch args.GroupBy {
	case database.GroupByNone, database.GroupByCategory, database.GroupByMonth, database.GroupByUser:
	default:
		return nil, fmt.Errorf("group_by must be 'category', 'month' or 'user'")
	}

	summaries, err := t.db.SummarizeActions(filters, args.GroupBy)
	if err != nil {
		slog.Error("chat tool summarize_actions failed", "error", err)
		return nil, fmt.Errorf("query failed")
	}
	for i := range summaries {
		summaries[i].Total = roundAmount(summaries[i].Total)
	}
	return map[string]any{"totals": summaries}, nil
}

func (t *chatTools) search(arguments string) (any, error) {
	filters, args, err := t.parseFilters(arguments)
	if err != nil {
		return nil, err
	}

	total, err := t.db.CountActions(filters)
	if err != nil {
		slog.Error("chat tool search_actions failed", "error", err)
		return nil, fmt.Errorf("query failed")
	}

	filters.Limit = args.Limit
	if filters.Limit <= 0 {
		filters.Limit = chatSearchDefaultLimit
	}
	filters.Limit = min(filters.Limit, chatSearchMaxLimit)

	actions, err := t.db.ListActions(filters)
	if err != nil {
		slog.Error("chat tool search_actions failed", "error", err)
		return nil, fmt.Errorf("query failed")
	}

	usernames, categoryNames, err := t.lookups()
	if err != nil {
		slog.Error("chat tool search_actions failed", "error", err)
		return nil, fmt.Errorf("query failed")
	}

	results := make([]chatActionResult, 0, len(actions))
	for _, a := range actions {
		item := chatActionResult{
			ID:          a.ID,
			Date:        dateOnly(a.Date),
			Type:        string(a.Type),
			Amount:      a.Amount,
			Description: a.Description,
			Username:    usernames[a.UserID],
		}
		if a.CategoryID != nil {
			item.Category = categoryNames[*a.CategoryID]
		}
		results = append(results, item)
	}
	return map[string]any{"total_count": total, "actions": results}, nil
}

func (t *chatTools) lookups() (map[int64]string, map[int64]string, error) {
	users, err := t.db.ListUsers()
	if err != nil {
		return nil, nil, err
	}
	categories, err := t.db.ListCategories("")
	if err != nil {
		return nil, nil, err
	}
	usernames := make(map[int64]string, len(users))
	for _, u := range users {
		usernames[u.ID] = u.Username
	}
	categoryNames := make(map[int64]string, len(categories))
	for _, c := range categories {
		categoryNames[c.ID] = c.Description
	}
	return usernames, categoryNames, nil
}

func (t *chatTools) create(arguments string) (any, error) {
	var args chatCreateArgs
	if err := decodeToolArgs(arguments, &args); err != nil {
		return nil, err
	}

	if t.created >= chatMaxCreates {
		return nil, fmt.Errorf("too many entries created in one request")
	}
	if args.Type != "income" && args.Type != "expense" {
		return nil, fmt.Errorf("type must be 'income' or 'expense'")
	}
	if _, err := time.Parse("2006-01-02", args.Date); err != nil {
		return nil, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", args.Date)
	}
	// Descriptions can come straight from text in a photo, so keep markup
	// out of them.
	args.Description = strings.TrimSpace(strings.NewReplacer("<", "", ">", "").Replace(args.Description))
	if args.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if utf8.RuneCountInString(args.Description) > chatMaxDescriptionLength {
		return nil, fmt.Errorf("description is too long (max %d characters)", chatMaxDescriptionLength)
	}
	amount := roundAmount(args.Amount)
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if args.CategoryID == nil {
		return nil, fmt.Errorf("category_id is required")
	}
	category, err := t.db.GetCategoryByID(*args.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("category %d does not exist", *args.CategoryID)
	}
	if string(category.ActionType) != args.Type {
		return nil, fmt.Errorf("category %q is for %s entries, not %s", category.Description, category.ActionType, args.Type)
	}

	action, err := t.db.CreateAction(t.session.UserID, models.ActionType(args.Type), args.Date, args.Description, amount, args.CategoryID)
	if err != nil {
		slog.Error("chat tool create_action failed", "error", err)
		return nil, fmt.Errorf("failed to create entry")
	}
	t.created++

	response := &ActionResponse{
		ID:          action.ID,
		UserID:      action.UserID,
		Username:    t.session.Username,
		Type:        string(action.Type),
		Date:        action.Date,
		Description: action.Description,
		Amount:      action.Amount,
		CategoryID:  action.CategoryID,
		CreatedAt:   action.CreatedAt.Format(time.RFC3339),
	}
	if user, err := t.db.GetUserByID(action.UserID); err == nil {
		response.Name = user.Name
	}
	t.onCreate(response)

	return map[string]any{
		"created": chatActionResult{
			ID:          action.ID,
			Date:        action.Date,
			Type:        string(action.Type),
			Amount:      action.Amount,
			Description: action.Description,
			Category:    category.Description,
		},
	}, nil
}

func decodeToolArgs(arguments string, into any) error {
	if strings.TrimSpace(arguments) == "" {
		arguments = "{}"
	}
	if err := json.Unmarshal([]byte(arguments), into); err != nil {
		return fmt.Errorf("arguments are not valid JSON for this tool")
	}
	return nil
}

// dateOnly trims the time part SQLite adds when reading DATE columns back
// (2026-11-05T00:00:00Z), leaving YYYY-MM-DD.
func dateOnly(date string) string {
	if len(date) > 10 {
		return date[:10]
	}
	return date
}

func roundAmount(amount float64) float64 {
	return math.Round(amount*100) / 100
}
