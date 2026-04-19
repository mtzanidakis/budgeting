package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/mtzanidakis/budgeting/internal/apiclient"
)

// commonFlags registers the common flags (--url, --token, --pretty) on fs and
// returns accessors that resolve env var fallbacks after fs.Parse is called.
type commonOpts struct {
	url    *string
	token  *string
	pretty *bool
}

func registerCommonFlags(fs *flag.FlagSet) *commonOpts {
	return &commonOpts{
		url:    fs.String("url", "", "Base URL (defaults to BUDGETING_URL)"),
		token:  fs.String("token", "", "API token (defaults to BUDGETING_TOKEN)"),
		pretty: fs.Bool("pretty", false, "Indent JSON output"),
	}
}

func (o *commonOpts) client() (*apiclient.Client, error) {
	url := *o.url
	if url == "" {
		url = os.Getenv("BUDGETING_URL")
	}
	if url == "" {
		return nil, errors.New("BUDGETING_URL is not set (or use --url)")
	}
	token := *o.token
	if token == "" {
		token = os.Getenv("BUDGETING_TOKEN")
	}
	if token == "" {
		return nil, errors.New("BUDGETING_TOKEN is not set (or use --token)")
	}
	return apiclient.New(url, token), nil
}

func emitJSON(v any, pretty bool) {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		fail("failed to encode JSON: %v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// runMe: budgeting-cli me
func runMe(args []string) {
	fs := flag.NewFlagSet("me", flag.ExitOnError)
	common := registerCommonFlags(fs)
	_ = fs.Parse(args)

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	resp, err := c.Me()
	if err != nil {
		fail("%v", err)
	}
	emitJSON(resp, *common.pretty)
}

// runActions dispatches subcommands under `actions`.
func runActions(args []string) {
	if len(args) == 0 {
		fail("actions: missing subcommand (list|create|update|delete)")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		actionsList(rest)
	case "create":
		actionsCreate(rest)
	case "update":
		actionsUpdate(rest)
	case "delete":
		actionsDelete(rest)
	default:
		fail("actions: unknown subcommand %q", sub)
	}
}

func actionsList(args []string) {
	fs := flag.NewFlagSet("actions list", flag.ExitOnError)
	common := registerCommonFlags(fs)
	from := fs.String("from", "", "Date from (YYYY-MM-DD)")
	to := fs.String("to", "", "Date to (YYYY-MM-DD)")
	typ := fs.String("type", "", "Action type (income|expense)")
	category := fs.String("category", "", "Category ID")
	user := fs.String("user", "", "Filter by username")
	search := fs.String("search", "", "Search in description")
	limit := fs.Int("limit", 0, "Limit results")
	offset := fs.Int("offset", 0, "Offset for pagination")
	_ = fs.Parse(args)

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.ListActions(apiclient.ActionFilters{
		Username:   *user,
		Type:       *typ,
		DateFrom:   *from,
		DateTo:     *to,
		Search:     *search,
		CategoryID: *category,
		Limit:      *limit,
		Offset:     *offset,
	})
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func actionsCreate(args []string) {
	fs := flag.NewFlagSet("actions create", flag.ExitOnError)
	common := registerCommonFlags(fs)
	typ := fs.String("type", "", "Action type (income|expense) (required)")
	date := fs.String("date", "", "Date YYYY-MM-DD (required)")
	desc := fs.String("description", "", "Description (required)")
	amount := fs.Float64("amount", 0, "Amount (required)")
	category := fs.Int64("category", 0, "Category ID (required)")
	_ = fs.Parse(args)

	if *typ == "" || *date == "" || *desc == "" || *amount == 0 || *category == 0 {
		fail("actions create: --type, --date, --description, --amount, --category are required")
	}
	cat := *category
	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.CreateAction(apiclient.ActionRequest{
		Type: *typ, Date: *date, Description: *desc, Amount: *amount, CategoryID: &cat,
	})
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func actionsUpdate(args []string) {
	if len(args) == 0 {
		fail("actions update: missing ID")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail("actions update: invalid ID %q", args[0])
	}
	fs := flag.NewFlagSet("actions update", flag.ExitOnError)
	common := registerCommonFlags(fs)
	typ := fs.String("type", "", "Action type (income|expense) (required)")
	date := fs.String("date", "", "Date YYYY-MM-DD (required)")
	desc := fs.String("description", "", "Description (required)")
	amount := fs.Float64("amount", 0, "Amount (required)")
	category := fs.Int64("category", 0, "Category ID (required)")
	_ = fs.Parse(args[1:])

	if *typ == "" || *date == "" || *desc == "" || *amount == 0 || *category == 0 {
		fail("actions update: --type, --date, --description, --amount, --category are required")
	}
	cat := *category
	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.UpdateAction(id, apiclient.ActionRequest{
		Type: *typ, Date: *date, Description: *desc, Amount: *amount, CategoryID: &cat,
	})
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func actionsDelete(args []string) {
	if len(args) == 0 {
		fail("actions delete: missing ID")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail("actions delete: invalid ID %q", args[0])
	}
	fs := flag.NewFlagSet("actions delete", flag.ExitOnError)
	common := registerCommonFlags(fs)
	_ = fs.Parse(args[1:])

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	if err := c.DeleteAction(id); err != nil {
		fail("%v", err)
	}
	emitJSON(map[string]bool{"success": true}, *common.pretty)
}

// runCategories dispatches subcommands under `categories`.
func runCategories(args []string) {
	if len(args) == 0 {
		fail("categories: missing subcommand (list|create|update|delete)")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		categoriesList(rest)
	case "create":
		categoriesCreate(rest)
	case "update":
		categoriesUpdate(rest)
	case "delete":
		categoriesDelete(rest)
	default:
		fail("categories: unknown subcommand %q", sub)
	}
}

func categoriesList(args []string) {
	fs := flag.NewFlagSet("categories list", flag.ExitOnError)
	common := registerCommonFlags(fs)
	typ := fs.String("type", "", "Filter by action_type (income|expense)")
	_ = fs.Parse(args)

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	cats, err := c.ListCategories(*typ)
	if err != nil {
		fail("%v", err)
	}
	emitJSON(cats, *common.pretty)
}

func categoriesCreate(args []string) {
	fs := flag.NewFlagSet("categories create", flag.ExitOnError)
	common := registerCommonFlags(fs)
	desc := fs.String("description", "", "Description (required)")
	typ := fs.String("type", "", "Action type (income|expense) (required)")
	_ = fs.Parse(args)

	if *desc == "" || *typ == "" {
		fail("categories create: --description and --type are required")
	}
	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.CreateCategory(apiclient.CategoryRequest{Description: *desc, ActionType: *typ})
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func categoriesUpdate(args []string) {
	if len(args) == 0 {
		fail("categories update: missing ID")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail("categories update: invalid ID %q", args[0])
	}
	fs := flag.NewFlagSet("categories update", flag.ExitOnError)
	common := registerCommonFlags(fs)
	desc := fs.String("description", "", "Description (required)")
	typ := fs.String("type", "", "Action type (income|expense) (required)")
	_ = fs.Parse(args[1:])

	if *desc == "" || *typ == "" {
		fail("categories update: --description and --type are required")
	}
	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.UpdateCategory(id, apiclient.CategoryRequest{Description: *desc, ActionType: *typ})
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func categoriesDelete(args []string) {
	if len(args) == 0 {
		fail("categories delete: missing ID")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fail("categories delete: invalid ID %q", args[0])
	}
	fs := flag.NewFlagSet("categories delete", flag.ExitOnError)
	common := registerCommonFlags(fs)
	_ = fs.Parse(args[1:])

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	if err := c.DeleteCategory(id); err != nil {
		fail("%v", err)
	}
	emitJSON(map[string]bool{"success": true}, *common.pretty)
}

// runCharts dispatches subcommands under `charts`.
func runCharts(args []string) {
	if len(args) == 0 {
		fail("charts: missing subcommand (monthly|categories)")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "monthly":
		chartsMonthly(rest)
	case "categories":
		chartsCategories(rest)
	default:
		fail("charts: unknown subcommand %q", sub)
	}
}

func chartsMonthly(args []string) {
	fs := flag.NewFlagSet("charts monthly", flag.ExitOnError)
	common := registerCommonFlags(fs)
	year := fs.Int("year", 0, "Year (defaults to current year)")
	_ = fs.Parse(args)

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.MonthlyChart(*year)
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

func chartsCategories(args []string) {
	fs := flag.NewFlagSet("charts categories", flag.ExitOnError)
	common := registerCommonFlags(fs)
	year := fs.Int("year", 0, "Year (defaults to current year)")
	month := fs.Int("month", 0, "Month 1-12 (defaults to current month)")
	_ = fs.Parse(args)

	c, err := common.client()
	if err != nil {
		fail("%v", err)
	}
	raw, err := c.CategoryChart(*year, *month)
	if err != nil {
		fail("%v", err)
	}
	emitRaw(raw, *common.pretty)
}

// emitRaw prints a json.RawMessage either compact or indented.
func emitRaw(raw json.RawMessage, pretty bool) {
	if !pretty {
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			fmt.Println(string(raw))
		} else {
			fmt.Print(string(raw))
		}
		return
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fail("failed to decode response: %v", err)
	}
	emitJSON(v, true)
}
