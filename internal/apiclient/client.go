// Package apiclient provides an HTTP client for the budgeting API.
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/manolis/budgeting/internal/models"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error: HTTP %d: %s", e.Status, e.Body)
}

func (c *Client) do(method, path string, query url.Values, body any, out any) error {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

// Me returns the authenticated user.
type MeResponse struct {
	Success bool `json:"success"`
	User    struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user"`
}

func (c *Client) Me() (*MeResponse, error) {
	var out MeResponse
	if err := c.do(http.MethodGet, "/api/me", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ActionFilters holds list filters.
type ActionFilters struct {
	Username   string
	Type       string
	DateFrom   string
	DateTo     string
	Search     string
	CategoryID string
	Limit      int
	Offset     int
}

func (f ActionFilters) values() url.Values {
	v := url.Values{}
	if f.Username != "" {
		v.Set("username", f.Username)
	}
	if f.Type != "" {
		v.Set("type", f.Type)
	}
	if f.DateFrom != "" {
		v.Set("date_from", f.DateFrom)
	}
	if f.DateTo != "" {
		v.Set("date_to", f.DateTo)
	}
	if f.Search != "" {
		v.Set("search", f.Search)
	}
	if f.CategoryID != "" {
		v.Set("category_id", f.CategoryID)
	}
	if f.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", f.Limit))
	}
	if f.Offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", f.Offset))
	}
	return v
}

// ListActions returns the raw JSON (actions list or paginated response).
func (c *Client) ListActions(f ActionFilters) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodGet, "/api/actions", f.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ActionRequest struct {
	Type        string  `json:"type"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	CategoryID  *int64  `json:"category_id,omitempty"`
}

func (c *Client) CreateAction(req ActionRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/api/actions", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateAction(id int64, req ActionRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPut, fmt.Sprintf("/api/actions/%d", id), nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteAction(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/actions/%d", id), nil, nil, nil)
}

// ListCategories returns all categories, optionally filtered by action_type.
func (c *Client) ListCategories(actionType string) ([]models.Category, error) {
	v := url.Values{}
	if actionType != "" {
		v.Set("action_type", actionType)
	}
	var out []models.Category
	if err := c.do(http.MethodGet, "/api/categories", v, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type CategoryRequest struct {
	Description string `json:"description"`
	ActionType  string `json:"action_type"`
}

func (c *Client) CreateCategory(req CategoryRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/api/categories", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateCategory(id int64, req CategoryRequest) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPut, fmt.Sprintf("/api/categories/%d", id), nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteCategory(id int64) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/categories/%d", id), nil, nil, nil)
}

// MonthlyChart returns the monthly income/expense summary for a year.
func (c *Client) MonthlyChart(year int) (json.RawMessage, error) {
	v := url.Values{}
	if year > 0 {
		v.Set("year", fmt.Sprintf("%d", year))
	}
	var out json.RawMessage
	if err := c.do(http.MethodGet, "/api/charts/monthly", v, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CategoryChart returns the category breakdown for a period.
func (c *Client) CategoryChart(year, month int) (json.RawMessage, error) {
	v := url.Values{}
	if year > 0 {
		v.Set("year", fmt.Sprintf("%d", year))
	}
	if month > 0 {
		v.Set("month", fmt.Sprintf("%d", month))
	}
	var out json.RawMessage
	if err := c.do(http.MethodGet, "/api/charts/categories", v, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
