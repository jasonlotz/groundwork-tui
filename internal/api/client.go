// Package api provides a typed HTTP client for the Groundwork tRPC API.
//
// tRPC 11 protocol:
//   - Queries:   GET /api/trpc/<procedure>?input=<url-encoded-json>
//     Response: {"result":{"data":{"json":<output>}}}
//   - Mutations: POST /api/trpc/<procedure>  body: {"json":<input>}
//     Response: {"result":{"data":{"json":<output>}}}
//
// Auth: Authorization: Bearer <api_key>
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jasonlotz/groundwork-tui/internal/model"
)

// debugLog writes a message to ~/.cache/groundwork-tui/debug.log.
// Errors opening/writing the file are silently ignored so they never
// surface to the user.
func debugLog(format string, args ...any) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(dir, "groundwork-tui")
	_ = os.MkdirAll(logDir, 0o700)
	f, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	log.New(f, "", log.LstdFlags).Printf(format, args...)
}

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

// trpcRequestWithMeta is used when the input contains Date fields that must be
// described in SuperJSON's meta.values map so tRPC can reconstruct them as JS
// Date objects on the server side.
//
// SuperJSON wire format for a Date:
//
//	{"json": {"startDate": "2025-01-01T00:00:00.000Z"},
//	 "meta": {"values": {"startDate": ["Date"]}}}
type trpcRequestWithMeta struct {
	JSON any                `json:"json"`
	Meta *trpcSuperJSONMeta `json:"meta,omitempty"`
}

type trpcSuperJSONMeta struct {
	Values map[string][]string `json:"values"`
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

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized — check your API key")
	}
	if resp.StatusCode >= 400 {
		debugLog("HTTP %d %s: %s", resp.StatusCode, req.URL.Path, string(raw))
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func parseResponse[T any](raw []byte) (T, error) {
	var zero T
	var result trpcResponse[T]
	if err := json.Unmarshal(raw, &result); err != nil {
		return zero, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return zero, fmt.Errorf("tRPC error: %s", result.Error.Message)
	}
	return result.Result.Data.JSON, nil
}

// query sends a GET request for a tRPC query procedure.
// Input is serialized as ?input=<url-encoded-json-wrapped-in-{"json":...}>
func query[T any](c *Client, procedure string, input any) (T, error) {
	var zero T

	inputJSON, err := json.Marshal(trpcRequest{JSON: input})
	if err != nil {
		return zero, fmt.Errorf("marshal input: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s?input=%s",
		c.baseURL, procedure, url.QueryEscape(string(inputJSON)))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// mutation sends a POST request for a tRPC mutation procedure.
func mutation[T any](c *Client, procedure string, input any) (T, error) {
	var zero T

	body, err := json.Marshal(trpcRequest{JSON: input})
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s", c.baseURL, procedure)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// mutationWithMeta is like mutation but accepts a trpcRequestWithMeta so that
// SuperJSON meta (e.g. date type hints) is included in the request body.
func mutationWithMeta[T any](c *Client, procedure string, body trpcRequestWithMeta) (T, error) {
	var zero T

	b, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s", c.baseURL, procedure)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// superJSONDates builds a trpcRequestWithMeta wrapping input, adding a
// meta.values entry for each non-empty date field path so tRPC/SuperJSON
// reconstructs them as JS Date objects on the server.
// datePaths are the dot-notation key paths of date fields (e.g. "startDate").
func superJSONDates(input any, datePaths ...string) trpcRequestWithMeta {
	meta := &trpcSuperJSONMeta{Values: make(map[string][]string)}
	for _, p := range datePaths {
		meta.Values[p] = []string{"Date"}
	}
	return trpcRequestWithMeta{JSON: input, Meta: meta}
}

// --- typed procedures ---

// GetOverview calls dashboard.getOverview.
func (c *Client) GetOverview() (*model.Overview, error) {
	out, err := query[model.Overview](c, "dashboard.getOverview", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetChartData calls dashboard.getChartData.
func (c *Client) GetChartData() (*model.ChartData, error) {
	out, err := query[model.ChartData](c, "dashboard.getChartData", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetActiveMaterials calls dashboard.getActiveMaterials.
func (c *Client) GetActiveMaterials() ([]model.ActiveMaterial, error) {
	out, err := query[[]model.ActiveMaterial](c, "dashboard.getActiveMaterials", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllSkills calls skill.getAll (no filters — returns all non-archived skills).
func (c *Client) GetAllSkills() ([]model.Skill, error) {
	out, err := query[[]model.Skill](c, "skill.getAll", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllCategories calls category.getAll.
func (c *Client) GetAllCategories() ([]model.Category, error) {
	out, err := query[[]model.Category](c, "category.getAll", struct{}{})
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
	out, err := query[[]model.Material](c, "material.getAll", input)
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
	out, err := query[[]model.ProgressLog](c, "progress.getAll", getProgressInput{MaterialID: materialID})
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

// LogUnits calls progress.logUnits (mutation).
func (c *Client) LogUnits(materialID, date string, units float64, notes *string) error {
	input := logUnitsInput{
		MaterialID: materialID,
		Date:       date + "T00:00:00.000Z",
		Units:      units,
		Notes:      notes,
	}
	_, err := mutationWithMeta[struct{}](c, "progress.logUnits", superJSONDates(input, "date"))
	return err
}

type getCategoryDataInput struct {
	CategoryID string `json:"categoryId"`
}

// GetCategoryData calls dashboard.getCategoryData.
func (c *Client) GetCategoryData(categoryID string) (*model.CategoryDetail, error) {
	out, err := query[model.CategoryDetail](c, "dashboard.getCategoryData", getCategoryDataInput{CategoryID: categoryID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getSkillDataInput struct {
	SkillID string `json:"skillId"`
}

// GetSkillData calls dashboard.getSkillData.
func (c *Client) GetSkillData(skillID string) (*model.SkillDetail, error) {
	out, err := query[model.SkillDetail](c, "dashboard.getSkillData", getSkillDataInput{SkillID: skillID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getMaterialDetailInput struct {
	MaterialID string `json:"materialId"`
}

// GetMaterialDetail calls dashboard.getMaterialDetail.
func (c *Client) GetMaterialDetail(materialID string) (*model.MaterialDetail, error) {
	out, err := query[model.MaterialDetail](c, "dashboard.getMaterialDetail", getMaterialDetailInput{MaterialID: materialID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// --- category mutations ---

type categoryCreateInput struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// CreateCategory calls category.create.
func (c *Client) CreateCategory(name string, color *string) error {
	_, err := mutation[struct{}](c, "category.create", categoryCreateInput{Name: name, Color: color})
	return err
}

type categoryUpdateInput struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// UpdateCategory calls category.update.
func (c *Client) UpdateCategory(id, name string, color *string) error {
	_, err := mutation[struct{}](c, "category.update", categoryUpdateInput{ID: id, Name: name, Color: color})
	return err
}

type categoryIDInput struct {
	ID string `json:"id"`
}

// ArchiveCategory calls category.archive.
func (c *Client) ArchiveCategory(id string) error {
	_, err := mutation[struct{}](c, "category.archive", categoryIDInput{ID: id})
	return err
}

// UnarchiveCategory calls category.unarchive.
func (c *Client) UnarchiveCategory(id string) error {
	_, err := mutation[struct{}](c, "category.unarchive", categoryIDInput{ID: id})
	return err
}

// DeleteCategory calls category.delete.
func (c *Client) DeleteCategory(id string) error {
	_, err := mutation[struct{}](c, "category.delete", categoryIDInput{ID: id})
	return err
}

// --- skill mutations ---

type skillCreateInput struct {
	Name       string  `json:"name"`
	CategoryID string  `json:"categoryId"`
	Color      *string `json:"color,omitempty"`
}

// CreateSkill calls skill.create.
func (c *Client) CreateSkill(name, categoryID string, color *string) error {
	_, err := mutation[struct{}](c, "skill.create", skillCreateInput{Name: name, CategoryID: categoryID, Color: color})
	return err
}

type skillUpdateInput struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CategoryID string  `json:"categoryId"`
	Color      *string `json:"color,omitempty"`
}

// UpdateSkill calls skill.update.
func (c *Client) UpdateSkill(id, name, categoryID string, color *string) error {
	_, err := mutation[struct{}](c, "skill.update", skillUpdateInput{ID: id, Name: name, CategoryID: categoryID, Color: color})
	return err
}

type skillIDInput struct {
	ID string `json:"id"`
}

// ArchiveSkill calls skill.archive.
func (c *Client) ArchiveSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.archive", skillIDInput{ID: id})
	return err
}

// UnarchiveSkill calls skill.unarchive.
func (c *Client) UnarchiveSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.unarchive", skillIDInput{ID: id})
	return err
}

// DeleteSkill calls skill.delete.
func (c *Client) DeleteSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.delete", skillIDInput{ID: id})
	return err
}

// --- materialType queries ---

// GetAllMaterialTypes calls materialType.getAll.
func (c *Client) GetAllMaterialTypes() ([]model.MaterialType, error) {
	out, err := query[[]model.MaterialType](c, "materialType.getAll", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --- material mutations ---

type materialCreateInput struct {
	Name           string  `json:"name"`
	SkillID        string  `json:"skillId"`
	TypeID         string  `json:"typeId"`
	UnitType       string  `json:"unitType"`
	TotalUnits     float64 `json:"totalUnits"`
	URL            *string `json:"url,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	StartDate      *string `json:"startDate,omitempty"`
	CompletedDate  *string `json:"completedDate,omitempty"`
	WeeklyUnitGoal *int    `json:"weeklyUnitGoal,omitempty"`
}

// MaterialCreateResult carries the fields needed to create a material.
// Populated by common.MaterialFormResult and passed to CreateMaterial.
type MaterialCreateResult struct {
	Name          string
	SkillID       string
	TypeID        string
	UnitType      string
	TotalUnits    float64
	URL           *string
	Notes         *string
	StartDate     *string
	CompletedDate *string
	WeeklyGoal    *int
}

// CreateMaterial calls material.create.
func (c *Client) CreateMaterial(r MaterialCreateResult) error {
	input := materialCreateInput{
		Name:           r.Name,
		SkillID:        r.SkillID,
		TypeID:         r.TypeID,
		UnitType:       r.UnitType,
		TotalUnits:     r.TotalUnits,
		URL:            r.URL,
		Notes:          r.Notes,
		WeeklyUnitGoal: r.WeeklyGoal,
	}
	// Date fields need SuperJSON meta so tRPC deserializes them as JS Dates.
	var datePaths []string
	if r.StartDate != nil {
		s := *r.StartDate + "T00:00:00.000Z"
		input.StartDate = &s
		datePaths = append(datePaths, "startDate")
	}
	if r.CompletedDate != nil {
		s := *r.CompletedDate + "T00:00:00.000Z"
		input.CompletedDate = &s
		datePaths = append(datePaths, "completedDate")
	}
	_, err := mutationWithMeta[struct{}](c, "material.create", superJSONDates(input, datePaths...))
	return err
}

type materialUpdateInput struct {
	ID             string   `json:"id"`
	Name           *string  `json:"name,omitempty"`
	SkillID        *string  `json:"skillId,omitempty"`
	TypeID         *string  `json:"typeId,omitempty"`
	UnitType       *string  `json:"unitType,omitempty"`
	TotalUnits     *float64 `json:"totalUnits,omitempty"`
	URL            *string  `json:"url,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	StartDate      *string  `json:"startDate,omitempty"`
	CompletedDate  *string  `json:"completedDate,omitempty"`
	WeeklyUnitGoal *int     `json:"weeklyUnitGoal,omitempty"`
}

// MaterialUpdateResult carries the fields needed to update a material.
type MaterialUpdateResult struct {
	ID            string
	Name          string
	SkillID       string
	TypeID        string
	UnitType      string
	TotalUnits    float64
	URL           *string
	Notes         *string
	StartDate     *string
	CompletedDate *string
	WeeklyGoal    *int
}

// UpdateMaterial calls material.update.
func (c *Client) UpdateMaterial(r MaterialUpdateResult) error {
	input := materialUpdateInput{
		ID:             r.ID,
		Name:           &r.Name,
		SkillID:        &r.SkillID,
		TypeID:         &r.TypeID,
		UnitType:       &r.UnitType,
		TotalUnits:     &r.TotalUnits,
		URL:            r.URL,
		Notes:          r.Notes,
		WeeklyUnitGoal: r.WeeklyGoal,
	}
	var datePaths []string
	if r.StartDate != nil {
		s := *r.StartDate + "T00:00:00.000Z"
		input.StartDate = &s
		datePaths = append(datePaths, "startDate")
	}
	if r.CompletedDate != nil {
		s := *r.CompletedDate + "T00:00:00.000Z"
		input.CompletedDate = &s
		datePaths = append(datePaths, "completedDate")
	}
	_, err := mutationWithMeta[struct{}](c, "material.update", superJSONDates(input, datePaths...))
	return err
}

type materialDeleteInput struct {
	ID string `json:"id"`
}

// DeleteMaterial calls material.delete.
func (c *Client) DeleteMaterial(id string) error {
	_, err := mutation[struct{}](c, "material.delete", materialDeleteInput{ID: id})
	return err
}
