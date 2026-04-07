// Package api provides a typed HTTP client for the Groundwork tRPC API.
//
// Each tRPC call is a POST to /api/trpc/<procedure> with a JSON body of the
// form {"json": <input>} and a response of {"result":{"data":{"json":<output>}}}.
// Superjson is used for serialization on the server, but for the types the TUI
// needs, all values are plain JSON-compatible (no Date objects in inputs).
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jasonlotz/groundwork-tui/internal/model"
)

// Client is a typed tRPC HTTP client.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// --- low-level tRPC plumbing ---

type trpcRequest struct {
	JSON any `json:"json"`
}

type trpcResponse[T any] struct {
	Result struct {
		Data struct {
			JSON T `json:"json"`
		} `json:"data"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func call[T any](c *Client, procedure string, input any) (T, error) {
	var zero T

	body, err := json.Marshal(trpcRequest{JSON: input})
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/trpc/%s", c.baseURL, procedure)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return zero, fmt.Errorf("unauthorized — check your API key")
	}
	if resp.StatusCode >= 400 {
		return zero, fmt.Errorf("server error %d: %s", resp.StatusCode, string(raw))
	}

	var result trpcResponse[T]
	if err := json.Unmarshal(raw, &result); err != nil {
		return zero, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return zero, fmt.Errorf("tRPC error: %s", result.Error.Message)
	}

	return result.Result.Data.JSON, nil
}

// --- typed procedures ---

// GetOverview calls dashboard.getOverview.
func (c *Client) GetOverview() (*model.Overview, error) {
	out, err := call[model.Overview](c, "dashboard.getOverview", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetChartData calls dashboard.getChartData.
func (c *Client) GetChartData() (*model.ChartData, error) {
	out, err := call[model.ChartData](c, "dashboard.getChartData", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type activeMaterialsInput struct {
	// no input needed — server returns active materials for the session user
}

// GetActiveMaterials calls dashboard.getActiveMaterials.
func (c *Client) GetActiveMaterials() ([]model.ActiveMaterial, error) {
	out, err := call[[]model.ActiveMaterial](c, "dashboard.getActiveMaterials", activeMaterialsInput{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllSkills calls skill.getAll (no filters — returns all non-archived skills).
func (c *Client) GetAllSkills() ([]model.Skill, error) {
	out, err := call[[]model.Skill](c, "skill.getAll", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllCategories calls category.getAll.
func (c *Client) GetAllCategories() ([]model.Category, error) {
	out, err := call[[]model.Category](c, "category.getAll", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getAllMaterialsInput struct {
	IsActive *bool `json:"isActive,omitempty"`
}

// GetAllMaterials calls material.getAll with an optional isActive filter.
func (c *Client) GetAllMaterials(activeOnly bool) ([]model.Material, error) {
	var input getAllMaterialsInput
	if activeOnly {
		t := true
		input.IsActive = &t
	}
	out, err := call[[]model.Material](c, "material.getAll", input)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getProgressInput struct {
	MaterialID *string `json:"materialId,omitempty"`
}

// GetAllProgress calls progress.getAll.
func (c *Client) GetAllProgress(materialID *string) ([]model.ProgressLog, error) {
	out, err := call[[]model.ProgressLog](c, "progress.getAll", getProgressInput{MaterialID: materialID})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type logUnitsInput struct {
	MaterialID string  `json:"materialId"`
	Date       string  `json:"date"`
	Units      float64 `json:"units"`
	Notes      *string `json:"notes,omitempty"`
}

// LogUnits calls progress.logUnits.
func (c *Client) LogUnits(materialID, date string, units float64, notes *string) error {
	_, err := call[struct{}](c, "progress.logUnits", logUnitsInput{
		MaterialID: materialID,
		Date:       date,
		Units:      units,
		Notes:      notes,
	})
	return err
}
